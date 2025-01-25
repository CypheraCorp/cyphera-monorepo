package handlers

import "cyphera-api/pkg/actalink"

type HandlerClient struct {
	actalink *actalink.ActaLinkClient
}

func NewHandlerClient(apiKey string) *HandlerClient {
	return &HandlerClient{
		actalink: actalink.NewActaLinkClient(apiKey),
	}
}
