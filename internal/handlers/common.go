package handlers

import (
	"cyphera-api/internal/db"
	"cyphera-api/internal/pkg/actalink"
)

// CommonServices holds common dependencies used across handlers
type CommonServices struct {
	db       *db.Queries
	actalink *actalink.ActaLinkClient
	// other shared dependencies
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse represents a standard success response
type SuccessResponse struct {
	Message string `json:"message"`
}

// NewCommonServices creates a new instance of CommonServices
func NewCommonServices(db *db.Queries, actalink *actalink.ActaLinkClient) *CommonServices {
	return &CommonServices{
		db:       db,
		actalink: actalink,
	}
}
