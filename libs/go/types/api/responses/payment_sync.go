package responses

// ConfigurationResponse represents payment provider configuration response
type ConfigurationResponse struct {
	ID                 string                 `json:"id"`
	WorkspaceID        string                 `json:"workspace_id"`
	ProviderName       string                 `json:"provider_name"`
	IsActive           bool                   `json:"is_active"`
	IsTestMode         bool                   `json:"is_test_mode"`
	WebhookEndpointURL string                 `json:"webhook_endpoint_url,omitempty"`
	ConnectedAccountID string                 `json:"connected_account_id,omitempty"`
	LastSyncAt         *int64                 `json:"last_sync_at,omitempty"`
	LastWebhookAt      *int64                 `json:"last_webhook_at,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt          int64                  `json:"created_at"`
	UpdatedAt          int64                  `json:"updated_at"`
	// Note: Configuration is not returned for security reasons
}

// InitialSyncResponse represents the response for initial sync operation
type InitialSyncResponse struct {
	SessionID   string                 `json:"session_id"`
	Status      string                 `json:"status"`
	Provider    string                 `json:"provider"`
	EntityTypes []string               `json:"entity_types"`
	Config      map[string]interface{} `json:"config"`
	CreatedAt   string                 `json:"created_at"`
}

// SyncSessionResponse represents a sync session response
type SyncSessionResponse struct {
	ID           string                 `json:"id"`
	WorkspaceID  string                 `json:"workspace_id"`
	Provider     string                 `json:"provider"`
	SessionType  string                 `json:"session_type"`
	Status       string                 `json:"status"`
	EntityTypes  []string               `json:"entity_types"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Progress     map[string]interface{} `json:"progress,omitempty"`
	ErrorSummary map[string]interface{} `json:"error_summary,omitempty"`
	StartedAt    *string                `json:"started_at,omitempty"`
	CompletedAt  *string                `json:"completed_at,omitempty"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
}

// ProviderAccountResponse represents the response for provider account operations
type ProviderAccountResponse struct {
	ID                string                 `json:"id"`
	WorkspaceID       string                 `json:"workspace_id"`
	ProviderName      string                 `json:"provider_name"`
	ProviderAccountID string                 `json:"provider_account_id"`
	AccountType       string                 `json:"account_type"`
	IsActive          bool                   `json:"is_active"`
	Environment       string                 `json:"environment"`
	DisplayName       string                 `json:"display_name,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt         string                 `json:"created_at"`
	UpdatedAt         string                 `json:"updated_at"`
}
