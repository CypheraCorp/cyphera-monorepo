package handlers

import (
	"cyphera-api/internal/db"
	"cyphera-api/internal/pkg/actalink"
)

type HandlerClient struct {
	actalink *actalink.ActaLinkClient
	db       *db.Queries
}

func NewHandlerClient(apiKey string, db *db.Queries) *HandlerClient {
	return &HandlerClient{
		actalink: actalink.NewActaLinkClient(apiKey),
		db:       db,
	}
}
