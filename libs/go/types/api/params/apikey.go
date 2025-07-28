package params

import (
	"time"

	"github.com/google/uuid"
)

// CreateAPIKeyParams contains parameters for creating an API key
type CreateAPIKeyParams struct {
	WorkspaceID  uuid.UUID
	Name         string
	Description  string
	Permissions  []string
	ExpiresAt    *time.Time
	AccessLevel  string
	RateLimitRPM *int32
	RateLimitRPD *int32
	Metadata     map[string]interface{}
}

// UpdateAPIKeyParams contains parameters for updating an API key
type UpdateAPIKeyParams struct {
	ID           uuid.UUID
	WorkspaceID  uuid.UUID
	Name         *string
	Description  *string
	Permissions  []string
	AccessLevel  string
	IsActive     *bool
	ExpiresAt    *time.Time
	RateLimitRPM *int32
	RateLimitRPD *int32
	Metadata     map[string]interface{}
}
