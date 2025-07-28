package requests

import "time"

// CreateAPIKeyRequest represents the request body for creating an API key
type CreateAPIKeyRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description,omitempty"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	AccessLevel string                 `json:"access_level" binding:"required,oneof=read write admin"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateAPIKeyRequest represents the request body for updating an API key
type UpdateAPIKeyRequest struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	AccessLevel string                 `json:"access_level,omitempty" binding:"omitempty,oneof=read write admin"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
