package main

import (
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azcertificates"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azkeys"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
)

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
