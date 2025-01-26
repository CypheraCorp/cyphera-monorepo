package handlers

import (
	"cyphera-api/internal/db"
	"cyphera-api/internal/pkg/actalink"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

// Shared services
type CommonServices struct {
	db       *db.Queries
	actalink *actalink.ActaLinkClient
	// other shared dependencies
}

func NewCommonServices(db *db.Queries, actalink *actalink.ActaLinkClient) *CommonServices {
	return &CommonServices{
		db:       db,
		actalink: actalink,
	}
}
