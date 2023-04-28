package main

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"log"
	"os"
	"sync/atomic"
	"time"

	"bazil.org/fuse/fs"
	"github.com/pkg/errors"
)

const cooldownTime = 5 * time.Second

const certificatesDirName = "certificates"
const keysDirName = "keys"
const secretsDirName = "secrets"

type entryType int

const (
	certificateEntryType entryType = iota + 1
	keyEntryType
	secretEntryType
	keyResponseEntryType
	certificateResponseEntryType
	secretResponseEntryType
)

type listingEntry struct {
	name     string
	azKvName string
	modTime  time.Time
	inode    uint64

	vaultClients *AzKVClients
	parent       *listingEntry
	children     []*listingEntry
	root         *listingEntry
	isRoot       bool
	entryType    entryType

	fetchTime *time.Time

	fs.Node

	nextInode *atomic.Uint64

	filter     func(typ string, data []byte) []byte
	filterType string

	isCertChain bool
}

var (
	_ os.FileInfo = (*listingEntry)(nil)
)

func (entry *listingEntry) IsDir() bool {
	return entry.isRoot || entry.parent.isRoot
}

func (entry *listingEntry) isCertificatesDir() bool {
	return entry.parent.isRoot && entry.name == certificatesDirName
}

func (entry *listingEntry) isSecretsDir() bool {
	return entry.parent.isRoot && entry.name == secretsDirName
}

func (entry *listingEntry) isKeysDir() bool {
	return entry.parent.isRoot && entry.name == keysDirName
}

func (entry *listingEntry) retrieveDirectoryListing(ctx context.Context) error {
	if entry.fetchTime != nil && time.Now().Before(entry.fetchTime.Add(cooldownTime)) {
		return nil
	}
	log.Println("Retrieving directory listing for", entry.name, "inode", entry.inode)
	if entry.isRoot {
		if len(entry.children) > 0 {
			return nil
		} else {
			now := time.Now()
			entry.children = []*listingEntry{
				{
					name:         certificatesDirName,
					modTime:      now,
					inode:        entry.advanceInode(),
					vaultClients: entry.vaultClients,
					parent:       entry,
					root:         entry.root,
					fetchTime:    nil,
				},
				{
					name:         keysDirName,
					modTime:      now,
					inode:        entry.advanceInode(),
					vaultClients: entry.vaultClients,
					parent:       entry,
					root:         entry.root,
					fetchTime:    nil,
				},
				{
					name:         secretsDirName,
					modTime:      now,
					inode:        entry.advanceInode(),
					vaultClients: entry.vaultClients,
					parent:       entry,
					root:         entry.root,
					fetchTime:    nil,
				},
			}
			return nil
		}
	}
	switch {
	case entry.isSecretsDir():
		return entry.retrieveSecretsDirectoryListing(ctx)
	case entry.isCertificatesDir():
		return entry.retrieveCertificatesDirectoryListing(ctx)
	case entry.isKeysDir():
		return entry.retrieveKeysDirectoryListing(ctx)
	default:
		return errors.New("Directory is untracked")
	}
}

func (entry *listingEntry) retrieveKeysDirectoryListing(ctx context.Context) error {
	pager := entry.vaultClients.keys.NewListKeysPager(nil)
	entry.children = nil
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return errors.Wrap(err, "could not get next page for secrets")
		}
		for _, key := range page.Value {
			modTime := time.UnixMilli(0)
			if key.Attributes.Updated != nil {
				modTime = *key.Attributes.Updated
			} else if key.Attributes.Created != nil {
				modTime = *key.Attributes.Created
			}
			entry.children = append(entry.children,
				&listingEntry{
					name:         key.KID.Name(),
					azKvName:     key.KID.Name(),
					modTime:      modTime,
					inode:        entry.advanceInode(),
					vaultClients: entry.vaultClients,
					parent:       entry,
					children:     nil,
					fetchTime:    nil,
					root:         entry.root,
					entryType:    keyEntryType,
				},
				&listingEntry{
					name:         key.KID.Name() + ".pem",
					azKvName:     key.KID.Name(),
					modTime:      modTime,
					inode:        entry.advanceInode(),
					vaultClients: entry.vaultClients,
					parent:       entry,
					children:     nil,
					fetchTime:    nil,
					root:         entry.root,
					entryType:    keyEntryType,
					filter:       ConvertEntry,
					filterType:   pemPrivKeyType,
				},
				&listingEntry{
					name:         key.KID.Name() + ".response",
					azKvName:     key.KID.Name(),
					modTime:      modTime,
					inode:        entry.advanceInode(),
					vaultClients: entry.vaultClients,
					parent:       entry,
					children:     nil,
					fetchTime:    nil,
					root:         entry.root,
					entryType:    keyResponseEntryType,
				},
			)
		}
	}
	now := time.Now()
	entry.fetchTime = &now
	return nil
}

func (entry *listingEntry) retrieveCertificatesDirectoryListing(ctx context.Context) error {
	pager := entry.vaultClients.certificates.NewListCertificatesPager(nil)
	entry.children = nil
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return errors.Wrap(err, "could not get next page for secrets")
		}
		for _, certificate := range page.Value {
			modTime := time.UnixMilli(0)
			if certificate.Attributes.Updated != nil {
				modTime = *certificate.Attributes.Updated
			} else if certificate.Attributes.Created != nil {
				modTime = *certificate.Attributes.Created
			}
			entry.children = append(entry.children,
				&listingEntry{
					name:         certificate.ID.Name(),
					azKvName:     certificate.ID.Name(),
					modTime:      modTime,
					inode:        entry.advanceInode(),
					vaultClients: entry.vaultClients,
					parent:       entry,
					children:     nil,
					fetchTime:    nil,
					root:         entry.root,
					entryType:    certificateEntryType,
				},
				&listingEntry{
					name:         certificate.ID.Name() + ".pem",
					azKvName:     certificate.ID.Name(),
					modTime:      modTime,
					inode:        entry.advanceInode(),
					vaultClients: entry.vaultClients,
					parent:       entry,
					children:     nil,
					fetchTime:    nil,
					root:         entry.root,
					entryType:    certificateEntryType,
					filter:       ConvertEntry,
					filterType:   pemCertType,
				},
				&listingEntry{
					name:         certificate.ID.Name() + ".chain.pem",
					azKvName:     certificate.ID.Name(),
					modTime:      modTime,
					inode:        entry.advanceInode(),
					vaultClients: entry.vaultClients,
					parent:       entry,
					children:     nil,
					fetchTime:    nil,
					root:         entry.root,
					entryType:    certificateEntryType,
					isCertChain:  true,
				},
				&listingEntry{
					name:         certificate.ID.Name() + ".response",
					azKvName:     certificate.ID.Name(),
					modTime:      modTime,
					inode:        entry.advanceInode(),
					vaultClients: entry.vaultClients,
					parent:       entry,
					children:     nil,
					fetchTime:    nil,
					root:         entry.root,
					entryType:    certificateResponseEntryType,
				},
			)
		}
	}
	now := time.Now()
	entry.fetchTime = &now
	return nil
}

func (entry *listingEntry) retrieveSecretsDirectoryListing(ctx context.Context) error {
	pager := entry.vaultClients.secrets.NewListSecretsPager(nil)
	entry.children = nil
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return errors.Wrap(err, "could not get next page for secrets")
		}
		for _, secret := range page.Value {
			modTime := time.UnixMilli(0)
			if secret.Attributes.Updated != nil {
				modTime = *secret.Attributes.Updated
			} else if secret.Attributes.Created != nil {
				modTime = *secret.Attributes.Created
			}
			entry.children = append(entry.children,
				&listingEntry{
					name:         secret.ID.Name(),
					azKvName:     secret.ID.Name(),
					modTime:      modTime,
					inode:        entry.advanceInode(),
					vaultClients: entry.vaultClients,
					parent:       entry,
					children:     nil,
					fetchTime:    nil,
					root:         entry.root,
					entryType:    secretEntryType,
				},
				&listingEntry{
					name:         secret.ID.Name() + ".response",
					azKvName:     secret.ID.Name(),
					modTime:      modTime,
					inode:        entry.advanceInode(),
					vaultClients: entry.vaultClients,
					parent:       entry,
					children:     nil,
					fetchTime:    nil,
					root:         entry.root,
					entryType:    secretResponseEntryType,
				},
			)

			if secret.ContentType != nil && *secret.ContentType == "application/x-pkcs12" {
				// Provide the base64-decoded value as .pfx file
				entry.children = append(entry.children,
					&listingEntry{
						name:         secret.ID.Name() + ".pfx",
						azKvName:     secret.ID.Name(),
						modTime:      modTime,
						inode:        entry.advanceInode(),
						vaultClients: entry.vaultClients,
						parent:       entry,
						children:     nil,
						fetchTime:    nil,
						root:         entry.root,
						entryType:    secretEntryType,
						filter:       ConvertEntry,
						filterType:   base64PfxType,
					},
				)
			}
		}
	}
	now := time.Now()
	entry.fetchTime = &now
	return nil
}

func (entry *listingEntry) advanceInode() uint64 {
	return entry.root.nextInode.Add(1)
}

func (entry *listingEntry) Find(name string, ctx context.Context) *listingEntry {
	log.Println("Find", name, "in", entry.name, "inode", entry.inode)
	if entry.fetchTime == nil {
		err := entry.retrieveDirectoryListing(ctx)
		if err != nil {
			return nil
		}
	}
	for _, child := range entry.children {
		if child.name == name {
			return child
		}
	}
	return nil
}

func encodeCertificate(der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	})
}

func buildCertificateChain(der []byte) ([][]byte, error) {
	var chain [][]byte
	cert, _ := x509.ParseCertificate(der)
	if cert != nil {
		chain = append(chain, encodeCertificate(der))
		for _, url := range cert.IssuingCertificateURL {
			log.Println("Found intermediate, downloading:", url)
			resp, err := httpClient.Get(url)
			if err != nil {
				return nil, err
			}
			intermediateCertBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			_ = resp.Body.Close()

			cert, _ := x509.ParseCertificate(der)
			if cert != nil {
				if len(cert.IssuingCertificateURL) != 0 {
					parentChain, err := buildCertificateChain(intermediateCertBytes)
					if err != nil {
						return nil, err
					}
					chain = append(chain, parentChain...)
				}
			}
		}
	}
	return chain, nil
}

func (entry *listingEntry) certChain(data []byte) ([]byte, error) {
	chain, err := buildCertificateChain(data)
	if err != nil {
		return nil, err
	}
	var chainBytes bytes.Buffer
	// All except root certificate
	for _, pemCert := range chain[:len(chain)-1] {
		chainBytes.Write(pemCert)
	}
	return chainBytes.Bytes(), nil
}

func (entry *listingEntry) Download(ctx context.Context) ([]byte, error) {
	log.Println("Download file", entry.name, "inode", entry.inode)
	var result []byte = nil
	switch entry.entryType {
	case certificateEntryType:
		certificateResponse, err :=
			entry.vaultClients.certificates.GetCertificate(ctx, entry.azKvName, "", nil)
		if err != nil {
			return nil, errors.Wrap(err, "could not get certificate")
		}
		result = certificateResponse.CER
		if entry.isCertChain {
			result, err = entry.certChain(result)
			if err != nil {
				return nil, err
			}
		}
	case certificateResponseEntryType:
		certificateResponse, err :=
			entry.vaultClients.certificates.GetCertificate(ctx, entry.azKvName, "", nil)
		if err != nil {
			return nil, errors.Wrap(err, "could not get certificate")
		}
		// Marshal certificateResponse
		result, err = json.Marshal(certificateResponse)
		if err != nil {
			return nil, errors.Wrap(err, "could not marshal certificate response")
		}
	case keyEntryType:
		keyResponse, err := entry.vaultClients.keys.GetKey(ctx, entry.azKvName, "", nil)
		if err != nil {
			return nil, errors.Wrap(err, "could not get key")
		}
		result = keyResponse.Key.K
	case keyResponseEntryType:
		keyResponse, err := entry.vaultClients.keys.GetKey(ctx, entry.azKvName, "", nil)
		if err != nil {
			return nil, errors.Wrap(err, "could not get key")
		}
		// Marshal keyResponse
		result, err = json.Marshal(keyResponse)
		if err != nil {
			return nil, errors.Wrap(err, "could not marshal key response")
		}
	case secretEntryType:
		secretResponse, err := entry.vaultClients.secrets.GetSecret(ctx, entry.azKvName, "", nil)
		if err != nil {
			return nil, errors.Wrap(err, "could not get secret")
		}
		result = []byte(*secretResponse.Value)
	case secretResponseEntryType:
		secretResponse, err := entry.vaultClients.secrets.GetSecret(ctx, entry.azKvName, "", nil)
		if err != nil {
			return nil, errors.Wrap(err, "could not get secret")
		}
		// Marshal secretResponse
		result, err = json.Marshal(secretResponse)
		if err != nil {
			return nil, errors.Wrap(err, "could not marshal secret response")
		}
	}

	now := time.Now()
	entry.fetchTime = &now

	if entry.filter != nil {
		return entry.filter(entry.filterType, result), nil
	}

	return result, nil
}

func (entry *listingEntry) Size() int64 {
	log.Println("Determining size of", entry.name, "inode", entry.inode)
	if entry.IsDir() {
		return int64(len(entry.children))
	}
	ctx := context.Background()

	switch entry.entryType {
	case certificateEntryType:
		certificateResponse, err :=
			entry.vaultClients.certificates.GetCertificate(ctx, entry.azKvName, "", nil)
		if err != nil {
			return -1
		}

		result := certificateResponse.CER

		if entry.isCertChain {
			result, err = entry.certChain(result)
			if err != nil {
				return -1
			}
		}

		if entry.filter != nil {
			return int64(len(entry.filter(entry.filterType, result)))
		} else {
			return int64(len(result))
		}
	case certificateResponseEntryType:
		certificateResponse, err :=
			entry.vaultClients.certificates.GetCertificate(ctx, entry.azKvName, "", nil)
		if err != nil {
			return -1
		}

		// Marshal certificateResponse
		result, err := json.Marshal(certificateResponse)
		if err != nil {
			return -1
		}

		if entry.filter != nil {
			return int64(len(entry.filter(entry.filterType, result)))
		} else {
			return int64(len(result))
		}
	case keyEntryType:
		keyResponse, err := entry.vaultClients.keys.GetKey(ctx, entry.azKvName, "", nil)
		if err != nil {
			return -1
		}
		if entry.filter != nil {
			return int64(len(entry.filter(entry.filterType, keyResponse.Key.K)))
		} else {
			return int64(len(keyResponse.Key.K))
		}
	case keyResponseEntryType:
		keyResponse, err := entry.vaultClients.keys.GetKey(ctx, entry.azKvName, "", nil)
		if err != nil {
			return -1
		}
		// Marshal keyResponse
		result, err := json.Marshal(keyResponse)
		if err != nil {
			return -1
		}
		if entry.filter != nil {
			return int64(len(entry.filter(entry.filterType, result)))
		} else {
			return int64(len(result))
		}
	case secretEntryType:
		secretResponse, err := entry.vaultClients.secrets.GetSecret(ctx, entry.azKvName, "", nil)
		if err != nil {
			return -1
		}
		if entry.filter != nil {
			return int64(len(entry.filter(entry.filterType, []byte(*secretResponse.Value))))
		} else {
			return int64(len([]byte(*secretResponse.Value)))
		}
	case secretResponseEntryType:
		secretResponse, err := entry.vaultClients.secrets.GetSecret(ctx, entry.azKvName, "", nil)
		if err != nil {
			return -1
		}
		// Marshal secretResponse
		result, err := json.Marshal(secretResponse)
		if err != nil {
			return -1
		}
		if entry.filter != nil {
			return int64(len(entry.filter(entry.filterType, result)))
		} else {
			return int64(len(result))
		}
	default:
		return -1
	}
}
