package main

import (
	"crypto/tls"
	"net/http"
	"time"
)

var transport = &http.Transport{
	ReadBufferSize:  32 * 1024,
	IdleConnTimeout: time.Minute * 1,
	TLSClientConfig: &tls.Config{
		InsecureSkipVerify: false,
	},
}

var httpClient = &http.Client{
	Transport: transport,
}

func client() *http.Client {
	return httpClient
}
