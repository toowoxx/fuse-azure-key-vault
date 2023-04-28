package main

import (
	"encoding/pem"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azcertificates"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azkeys"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
)

const pemType = "pem"
const pemPrivKeyType = "pemPrivKey"
const pemCertType = "pemCert"

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
	default:
		return data
	}
}
