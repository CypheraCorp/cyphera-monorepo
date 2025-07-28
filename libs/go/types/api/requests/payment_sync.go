package requests

// PaymentProviderConfig represents the configuration for a payment provider
type PaymentProviderConfig struct {
	APIKey         string `json:"api_key" binding:"required"`
	WebhookSecret  string `json:"webhook_secret" binding:"required"`
	PublishableKey string `json:"publishable_key,omitempty"`
	Environment    string `json:"environment" binding:"required"` // "test" or "live"
	BaseURL        string `json:"base_url,omitempty"`
}

// CreateConfigurationRequest represents the request to create payment provider configuration
type CreateConfigurationRequest struct {
	ProviderName       string                 `json:"provider_name" binding:"required"`
	IsActive           bool                   `json:"is_active"`
	IsTestMode         bool                   `json:"is_test_mode"`
	Configuration      PaymentProviderConfig  `json:"configuration" binding:"required"`
	WebhookEndpointURL string                 `json:"webhook_endpoint_url,omitempty"`
	ConnectedAccountID string                 `json:"connected_account_id,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateConfigurationRequest represents the request to update payment provider configuration
type UpdateConfigurationRequest struct {
	IsActive           *bool                  `json:"is_active,omitempty"`
	IsTestMode         *bool                  `json:"is_test_mode,omitempty"`
	Configuration      *PaymentProviderConfig `json:"configuration,omitempty"`
	WebhookEndpointURL *string                `json:"webhook_endpoint_url,omitempty"`
	ConnectedAccountID *string                `json:"connected_account_id,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// InitialSyncRequest represents the request to start initial sync
type InitialSyncRequest struct {
	EntityTypes   []string `json:"entity_types,omitempty"`
	BatchSize     int      `json:"batch_size,omitempty"`
	FullSync      bool     `json:"full_sync"`
	StartingAfter string   `json:"starting_after,omitempty"`
	EndingBefore  string   `json:"ending_before,omitempty"`
}

// CreateProviderAccountRequest represents the request to create a provider account mapping
type CreateProviderAccountRequest struct {
	ProviderName      string                 `json:"provider_name" binding:"required"`
	ProviderAccountID string                 `json:"provider_account_id" binding:"required"`
	AccountType       string                 `json:"account_type" binding:"required"`
	IsActive          bool                   `json:"is_active"`
	Environment       string                 `json:"environment" binding:"required"`
	DisplayName       string                 `json:"display_name,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}
