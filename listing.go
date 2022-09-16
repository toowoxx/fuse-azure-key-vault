package main

import (
	"context"
	"os"
	"sync/atomic"
	"time"

	"bazil.org/fuse/fs"
	"github.com/pkg/errors"
)

const certificatesDirName = "certificates"
const keysDirName = "keys"
const secretsDirName = "secrets"

type entryType int

const (
	certificateEntryType entryType = iota + 1
	keyEntryType
	secretEntryType
)

type listingEntry struct {
	name    string
	size    int64
	modTime time.Time
	inode   uint64

	vaultClients *AzKVClients
	parent       *listingEntry
	children     []*listingEntry
	fileCount    int
	root         *listingEntry
	isRoot       bool
	entryType    entryType

	fetchTime *time.Time

	fs.Node

	nextInode *atomic.Uint64
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
					fetchTime:    &now,
				},
				{
					name:         keysDirName,
					modTime:      now,
					inode:        entry.advanceInode(),
					vaultClients: entry.vaultClients,
					parent:       entry,
					root:         entry.root,
					fetchTime:    &now,
				},
				{
					name:         secretsDirName,
					modTime:      now,
					inode:        entry.advanceInode(),
					vaultClients: entry.vaultClients,
					parent:       entry,
					root:         entry.root,
					fetchTime:    &now,
				},
			}
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
	entry.fileCount = 0
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return errors.Wrap(err, "could not get next page for secrets")
		}
		now := time.Now()
		for _, key := range page.Value {
			modTime := time.UnixMilli(0)
			if key.Attributes.Updated != nil {
				modTime = *key.Attributes.Updated
			} else if key.Attributes.Created != nil {
				modTime = *key.Attributes.Created
			}
			entry.children = append(entry.children, &listingEntry{
				name:         key.KID.Name(),
				size:         0,
				modTime:      modTime,
				inode:        entry.advanceInode(),
				vaultClients: entry.vaultClients,
				parent:       entry,
				children:     nil,
				fileCount:    0,
				fetchTime:    &now,
				root:         entry.root,
				entryType:    keyEntryType,
			})
			entry.fileCount++
		}
	}
	return nil
}

func (entry *listingEntry) retrieveCertificatesDirectoryListing(ctx context.Context) error {
	pager := entry.vaultClients.certificates.NewListCertificatesPager(nil)
	entry.children = nil
	entry.fileCount = 0
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return errors.Wrap(err, "could not get next page for secrets")
		}
		now := time.Now()
		for _, certificate := range page.Value {
			modTime := time.UnixMilli(0)
			if certificate.Attributes.Updated != nil {
				modTime = *certificate.Attributes.Updated
			} else if certificate.Attributes.Created != nil {
				modTime = *certificate.Attributes.Created
			}
			entry.children = append(entry.children, &listingEntry{
				name:         certificate.ID.Name(),
				size:         0,
				modTime:      modTime,
				inode:        entry.advanceInode(),
				vaultClients: entry.vaultClients,
				parent:       entry,
				children:     nil,
				fileCount:    0,
				fetchTime:    &now,
				root:         entry.root,
				entryType:    certificateEntryType,
			})
			entry.fileCount++
		}
	}
	return nil
}

func (entry *listingEntry) retrieveSecretsDirectoryListing(ctx context.Context) error {
	pager := entry.vaultClients.secrets.NewListSecretsPager(nil)
	entry.children = nil
	entry.fileCount = 0
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return errors.Wrap(err, "could not get next page for secrets")
		}
		now := time.Now()
		for _, secret := range page.Value {
			modTime := time.UnixMilli(0)
			if secret.Attributes.Updated != nil {
				modTime = *secret.Attributes.Updated
			} else if secret.Attributes.Created != nil {
				modTime = *secret.Attributes.Created
			}
			entry.children = append(entry.children, &listingEntry{
				name:         secret.ID.Name(),
				size:         0,
				modTime:      modTime,
				inode:        entry.advanceInode(),
				vaultClients: entry.vaultClients,
				parent:       entry,
				children:     nil,
				fileCount:    0,
				fetchTime:    &now,
				root:         entry.root,
				entryType:    secretEntryType,
			})
			entry.fileCount++
		}
	}
	return nil
}

func (entry *listingEntry) advanceInode() uint64 {
	return entry.root.nextInode.Add(1)
}

func (entry *listingEntry) Find(name string) *listingEntry {
	for _, child := range entry.children {
		if child.name == name {
			return child
		}
	}
	return nil
}

func (entry *listingEntry) Download(ctx context.Context) ([]byte, error) {
	var result []byte = nil
	switch entry.entryType {
	case certificateEntryType:
		certificateResponse, err := entry.vaultClients.certificates.GetCertificate(ctx, entry.name, "", nil)
		if err != nil {
			return nil, errors.Wrap(err, "could not get certificate")
		}
		result = certificateResponse.CER
	case keyEntryType:
		keyResponse, err := entry.vaultClients.keys.GetKey(ctx, entry.name, "", nil)
		if err != nil {
			return nil, errors.Wrap(err, "could not get key")
		}
		result = keyResponse.Key.K
	case secretEntryType:
		secretResponse, err := entry.vaultClients.secrets.GetSecret(ctx, entry.name, "", nil)
		if err != nil {
			return nil, errors.Wrap(err, "could not get secret")
		}
		result = []byte(*secretResponse.Value)
	}

	entry.size = int64(len(result))
	return result, nil
}
