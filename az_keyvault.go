package main

import (
	"encoding/base64"
	"encoding/pem"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azcertificates"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azkeys"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
)

const pemPrivKeyType = "pemPrivKey"
const pemCertType = "pemCert"
const base64PfxType = "base64Pfx"

type AzKVClients struct {
	secrets      *azsecrets.Client
	keys         *azkeys.Client
	certificates *azcertificates.Client
}

func ConnectToKeyVault(url string) *AzKVClients {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("failed to obtain a credential: %v", err)
	}

	secretClient, err := azsecrets.NewClient(url, cred, nil)
	if err != nil {
		log.Fatalf("failed to create a secret client: %v", err)
	}

	keyClient, err := azkeys.NewClient(url, cred, nil)
	if err != nil {
		log.Fatalf("failed to create a key client: %v", err)
	}

	certClient, err := azcertificates.NewClient(url, cred, nil)
	if err != nil {
		log.Fatalf("failed to create a certificate client: %v", err)
	}

	return &AzKVClients{
		secrets:      secretClient,
		keys:         keyClient,
		certificates: certClient,
	}
}

func ConvertEntry(typ string, data []byte) []byte {
	switch typ {
	case pemCertType:
		return pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: data,
		})
	case pemPrivKeyType:
		return pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: data,
		})
	case base64PfxType:
		// Decode the base64 data
		decoded, err := base64.StdEncoding.DecodeString(string(data))
		if err != nil {
			return []byte("Error decoding base64 data")
		}
		return decoded
	default:
		return data
	}
}
