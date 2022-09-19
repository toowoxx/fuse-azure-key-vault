package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_buildCertificateChain(t *testing.T) {
	log.Println("Downloading Let's Encrypt certificate")
	conn, err := tls.Dial("tcp", "letsencrypt.org:443", &tls.Config{})
	assert.NoError(t, err)
	defer conn.Close()

	cert := conn.ConnectionState().PeerCertificates[0].Raw

	log.Println("Getting certificate chain")
	chain, err := buildCertificateChain(cert)

	log.Println(chain)
	log.Println("Printing chain")

	for _, pem := range chain {
		fmt.Print(string(pem))
	}

	assert.NoError(t, err)
}
