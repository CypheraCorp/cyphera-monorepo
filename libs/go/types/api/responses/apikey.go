package responses

// APIKeyResponse represents the standardized API response for API key operations
type APIKeyResponse struct {
	ID          string                 `json:"id"`
	Object      string                 `json:"object"`
	Name        string                 `json:"name"`
	AccessLevel string                 `json:"access_level"`
	ExpiresAt   *int64                 `json:"expires_at,omitempty"`
	LastUsedAt  *int64                 `json:"last_used_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   int64                  `json:"created_at"`
	UpdatedAt   int64                  `json:"updated_at"`
	KeyPrefix   string                 `json:"key_prefix,omitempty"` // Shows first part of key for identification
	Key         string                 `json:"key,omitempty"`        // Only included on creation
}

// ListAPIKeysResponse represents the paginated response for API key list operations
type ListAPIKeysResponse struct {
	Object  string           `json:"object"`
	Data    []APIKeyResponse `json:"data"`
	HasMore bool             `json:"has_more"`
	Total   int64            `json:"total"`
}
