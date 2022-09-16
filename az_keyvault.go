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

	return &AzKVClients{
		secrets:      azsecrets.NewClient(url, cred, nil),
		keys:         azkeys.NewClient(url, cred, nil),
		certificates: azcertificates.NewClient(url, cred, nil),
	}
}

func ConvertEntry(typ string, data []byte) []byte {
	switch typ {
	case pemType:
		return pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: data,
		})
	default:
		return data
	}
}
