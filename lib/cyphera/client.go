package cyphera

import "cyphera-api/pkg/actalink"

type CypheraClient struct {
	actalink *actalink.ActaLinkClient
}

func NewCypheraClient(apiKey string) *CypheraClient {
	return &CypheraClient{
		actalink: actalink.NewActaLinkClient(apiKey),
	}
}
