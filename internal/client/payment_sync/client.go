package payment_sync

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"cyphera-api/internal/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// PaymentSyncClient manages workspace-specific payment provider configurations
// and provides the top-level interface for payment synchronization operations
type PaymentSyncClient struct {
	db            *db.Queries
	logger        *zap.Logger
	encryptionKey []byte
	providers     map[string]PaymentSyncService // Registry of available payment providers
}

// PaymentProviderConfig represents the configuration for a payment provider
type PaymentProviderConfig struct {
	APIKey         string `json:"api_key"`
	WebhookSecret  string `json:"webhook_secret"`
	PublishableKey string `json:"publishable_key,omitempty"`
	Environment    string `json:"environment"` // "test" or "live"
	BaseURL        string `json:"base_url,omitempty"`
}

// WorkspacePaymentConfig represents a workspace payment configuration
type WorkspacePaymentConfig struct {
	ID                 string                 `json:"id"`
	WorkspaceID        string                 `json:"workspace_id"`
	ProviderName       string                 `json:"provider_name"`
	IsActive           bool                   `json:"is_active"`
	IsTestMode         bool                   `json:"is_test_mode"`
	Configuration      PaymentProviderConfig  `json:"configuration"`
	WebhookEndpointURL string                 `json:"webhook_endpoint_url,omitempty"`
	ConnectedAccountID string                 `json:"connected_account_id,omitempty"`
	LastSyncAt         *int64                 `json:"last_sync_at,omitempty"`
	LastWebhookAt      *int64                 `json:"last_webhook_at,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt          int64                  `json:"created_at"`
	UpdatedAt          int64                  `json:"updated_at"`
}

// NewPaymentSyncClient creates a new workspace payment client
func NewPaymentSyncClient(dbQueries *db.Queries, logger *zap.Logger, encryptionKey string) *PaymentSyncClient {
	// Convert hex encryption key to bytes
	// The encryption key is expected to be hex-encoded (64 hex characters for 32 bytes)
	key, err := hex.DecodeString(encryptionKey)
	if err != nil {
		logger.Fatal("Invalid encryption key format - expected hex", zap.Error(err))
	}
	if len(key) != 32 { // AES-256 requires 32-byte key
		logger.Fatal("Encryption key must be 32 bytes for AES-256", zap.Int("length", len(key)))
	}

	return &PaymentSyncClient{
		db:            dbQueries,
		logger:        logger,
		encryptionKey: key,
		providers:     make(map[string]PaymentSyncService),
	}
}

// RegisterProvider registers a payment provider service
func (c *PaymentSyncClient) RegisterProvider(providerName string, service PaymentSyncService) {
	c.providers[providerName] = service
	c.logger.Info("Registered payment provider", zap.String("provider", providerName))
}

// GetProviderService returns a configured payment provider service for a workspace
func (c *PaymentSyncClient) GetProviderService(ctx context.Context, workspaceID, providerName string) (PaymentSyncService, error) {
	// Get the provider service
	service, exists := c.providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider %s not registered", providerName)
	}

	// Get the workspace configuration for this provider
	config, err := c.GetConfiguration(ctx, workspaceID, providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace configuration: %w", err)
	}

	// Configure the service with workspace-specific settings
	configMap := map[string]string{
		"api_key":        config.Configuration.APIKey,
		"webhook_secret": config.Configuration.WebhookSecret,
		"environment":    config.Configuration.Environment,
	}

	if config.Configuration.PublishableKey != "" {
		configMap["publishable_key"] = config.Configuration.PublishableKey
	}
	if config.Configuration.BaseURL != "" {
		configMap["base_url"] = config.Configuration.BaseURL
	}

	err = service.Configure(ctx, configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to configure provider service: %w", err)
	}

	return service, nil
}

// CreateConfiguration creates a new payment provider configuration for a workspace
func (c *PaymentSyncClient) CreateConfiguration(ctx context.Context, config WorkspacePaymentConfig) (*WorkspacePaymentConfig, error) {
	wsID, err := uuid.Parse(config.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace ID: %w", err)
	}

	// Validate that the provider is registered
	if _, exists := c.providers[config.ProviderName]; !exists {
		return nil, fmt.Errorf("provider %s is not registered", config.ProviderName)
	}

	// Encrypt the configuration
	encryptedConfig, err := c.encryptConfiguration(config.Configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt configuration: %w", err)
	}

	// Prepare metadata
	metadata := make([]byte, 0)
	if config.Metadata != nil {
		metadata, err = json.Marshal(config.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	// Create in database
	dbConfig, err := c.db.CreateWorkspacePaymentConfiguration(ctx, db.CreateWorkspacePaymentConfigurationParams{
		WorkspaceID:        wsID,
		ProviderName:       config.ProviderName,
		IsActive:           config.IsActive,
		IsTestMode:         config.IsTestMode,
		Configuration:      encryptedConfig,
		WebhookEndpointUrl: pgtype.Text{String: config.WebhookEndpointURL, Valid: config.WebhookEndpointURL != ""},
		WebhookSecretKey:   pgtype.Text{String: c.encryptWebhookSecret(config.Configuration.WebhookSecret), Valid: config.Configuration.WebhookSecret != ""},
		ConnectedAccountID: pgtype.Text{String: config.ConnectedAccountID, Valid: config.ConnectedAccountID != ""},
		Metadata:           metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace payment configuration: %w", err)
	}

	result := c.mapDBConfigToService(dbConfig)
	return &result, nil
}

// GetConfiguration retrieves a workspace payment configuration by provider
func (c *PaymentSyncClient) GetConfiguration(ctx context.Context, workspaceID, providerName string) (*WorkspacePaymentConfig, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace ID: %w", err)
	}

	dbConfig, err := c.db.GetWorkspacePaymentConfiguration(ctx, db.GetWorkspacePaymentConfigurationParams{
		WorkspaceID:  wsID,
		ProviderName: providerName,
	})
	if err != nil {
		return nil, fmt.Errorf("configuration not found: %w", err)
	}

	result := c.mapDBConfigToService(dbConfig)
	return &result, nil
}

// GetConfigurationByID retrieves a workspace payment configuration by ID
func (c *PaymentSyncClient) GetConfigurationByID(ctx context.Context, workspaceID, configID string) (*WorkspacePaymentConfig, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace ID: %w", err)
	}

	cfgID, err := uuid.Parse(configID)
	if err != nil {
		return nil, fmt.Errorf("invalid configuration ID: %w", err)
	}

	dbConfig, err := c.db.GetWorkspacePaymentConfigurationByID(ctx, db.GetWorkspacePaymentConfigurationByIDParams{
		ID:          cfgID,
		WorkspaceID: wsID,
	})
	if err != nil {
		return nil, fmt.Errorf("configuration not found: %w", err)
	}

	result := c.mapDBConfigToService(dbConfig)
	return &result, nil
}

// ListConfigurations lists all payment configurations for a workspace
func (c *PaymentSyncClient) ListConfigurations(ctx context.Context, workspaceID string, limit, offset int) ([]WorkspacePaymentConfig, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace ID: %w", err)
	}

	dbConfigs, err := c.db.ListWorkspacePaymentConfigurations(ctx, db.ListWorkspacePaymentConfigurationsParams{
		WorkspaceID: wsID,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list configurations: %w", err)
	}

	results := make([]WorkspacePaymentConfig, len(dbConfigs))
	for i, dbConfig := range dbConfigs {
		results[i] = c.mapDBConfigToService(dbConfig)
	}

	return results, nil
}

// UpdateConfiguration updates an existing payment configuration
func (c *PaymentSyncClient) UpdateConfiguration(ctx context.Context, workspaceID, configID string, updates WorkspacePaymentConfig) (*WorkspacePaymentConfig, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace ID: %w", err)
	}

	cfgID, err := uuid.Parse(configID)
	if err != nil {
		return nil, fmt.Errorf("invalid configuration ID: %w", err)
	}

	// Encrypt the configuration
	encryptedConfig, err := c.encryptConfiguration(updates.Configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt configuration: %w", err)
	}

	// Prepare metadata
	metadata := make([]byte, 0)
	if updates.Metadata != nil {
		metadata, err = json.Marshal(updates.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	// Update in database
	dbConfig, err := c.db.UpdateWorkspacePaymentConfiguration(ctx, db.UpdateWorkspacePaymentConfigurationParams{
		ID:                 cfgID,
		WorkspaceID:        wsID,
		IsActive:           updates.IsActive,
		IsTestMode:         updates.IsTestMode,
		Configuration:      encryptedConfig,
		WebhookEndpointUrl: pgtype.Text{String: updates.WebhookEndpointURL, Valid: updates.WebhookEndpointURL != ""},
		WebhookSecretKey:   pgtype.Text{String: c.encryptWebhookSecret(updates.Configuration.WebhookSecret), Valid: updates.Configuration.WebhookSecret != ""},
		ConnectedAccountID: pgtype.Text{String: updates.ConnectedAccountID, Valid: updates.ConnectedAccountID != ""},
		Metadata:           metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update configuration: %w", err)
	}

	result := c.mapDBConfigToService(dbConfig)
	return &result, nil
}

// DeleteConfiguration soft deletes a payment configuration
func (c *PaymentSyncClient) DeleteConfiguration(ctx context.Context, workspaceID, configID string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return fmt.Errorf("invalid workspace ID: %w", err)
	}

	cfgID, err := uuid.Parse(configID)
	if err != nil {
		return fmt.Errorf("invalid configuration ID: %w", err)
	}

	_, err = c.db.DeleteWorkspacePaymentConfiguration(ctx, db.DeleteWorkspacePaymentConfigurationParams{
		ID:          cfgID,
		WorkspaceID: wsID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete configuration: %w", err)
	}

	return nil
}

// TestConnection tests the connection to a payment provider using the configuration
func (c *PaymentSyncClient) TestConnection(ctx context.Context, workspaceID, configID string) error {
	config, err := c.GetConfigurationByID(ctx, workspaceID, configID)
	if err != nil {
		return fmt.Errorf("failed to get configuration: %w", err)
	}

	// Get the provider service and test the connection
	service, err := c.GetProviderService(ctx, workspaceID, config.ProviderName)
	if err != nil {
		return fmt.Errorf("failed to get provider service: %w", err)
	}

	err = service.CheckConnection(ctx)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	c.logger.Info("Connection test successful",
		zap.String("workspace_id", workspaceID),
		zap.String("provider", config.ProviderName))

	return nil
}

// GetAvailableProviders returns a list of registered payment providers
func (c *PaymentSyncClient) GetAvailableProviders() []string {
	providers := make([]string, 0, len(c.providers))
	for name := range c.providers {
		providers = append(providers, name)
	}
	return providers
}

// StartInitialSync initiates an initial sync for a workspace and provider
func (c *PaymentSyncClient) StartInitialSync(ctx context.Context, workspaceID, providerName string, config InitialSyncConfig) (SyncSession, error) {
	service, err := c.GetProviderService(ctx, workspaceID, providerName)
	if err != nil {
		return SyncSession{}, fmt.Errorf("failed to get provider service: %w", err)
	}

	return service.StartInitialSync(ctx, workspaceID, config)
}

// Helper methods

func (c *PaymentSyncClient) mapDBConfigToService(dbConfig db.WorkspacePaymentConfiguration) WorkspacePaymentConfig {
	// Decrypt configuration
	config, err := c.decryptConfiguration(dbConfig.Configuration)
	if err != nil {
		c.logger.Error("Failed to decrypt configuration", zap.Error(err))
		config = PaymentProviderConfig{} // Use empty config if decryption fails
	}

	// Decrypt webhook secret from the database field
	if dbConfig.WebhookSecretKey.Valid && dbConfig.WebhookSecretKey.String != "" {
		decryptedWebhookSecret := c.decryptWebhookSecret(dbConfig.WebhookSecretKey.String)
		if decryptedWebhookSecret != "" {
			config.WebhookSecret = decryptedWebhookSecret
		}
	}

	result := WorkspacePaymentConfig{
		ID:                 dbConfig.ID.String(),
		WorkspaceID:        dbConfig.WorkspaceID.String(),
		ProviderName:       dbConfig.ProviderName,
		IsActive:           dbConfig.IsActive,
		IsTestMode:         dbConfig.IsTestMode,
		Configuration:      config,
		WebhookEndpointURL: dbConfig.WebhookEndpointUrl.String,
		ConnectedAccountID: dbConfig.ConnectedAccountID.String,
		CreatedAt:          dbConfig.CreatedAt.Time.Unix(),
		UpdatedAt:          dbConfig.UpdatedAt.Time.Unix(),
	}

	// Handle metadata
	if len(dbConfig.Metadata) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(dbConfig.Metadata, &metadata); err == nil {
			result.Metadata = metadata
		}
	}

	// Handle optional fields
	if dbConfig.LastSyncAt.Valid {
		lastSync := dbConfig.LastSyncAt.Time.Unix()
		result.LastSyncAt = &lastSync
	}
	if dbConfig.LastWebhookAt.Valid {
		lastWebhook := dbConfig.LastWebhookAt.Time.Unix()
		result.LastWebhookAt = &lastWebhook
	}

	return result
}

func (c *PaymentSyncClient) encryptConfiguration(config PaymentProviderConfig) ([]byte, error) {
	// Convert config to JSON
	plaintext, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(c.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to create nonce: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func (c *PaymentSyncClient) decryptConfiguration(ciphertext []byte) (PaymentProviderConfig, error) {
	var config PaymentProviderConfig

	// Create AES cipher
	block, err := aes.NewCipher(c.encryptionKey)
	if err != nil {
		return config, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return config, fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(ciphertext) < gcm.NonceSize() {
		return config, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce := ciphertext[:gcm.NonceSize()]
	ciphertext = ciphertext[gcm.NonceSize():]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return config, fmt.Errorf("failed to decrypt: %w", err)
	}

	// Unmarshal JSON
	if err := json.Unmarshal(plaintext, &config); err != nil {
		return config, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return config, nil
}

// encryptWebhookSecret encrypts a webhook secret using the same encryption as configuration
func (c *PaymentSyncClient) encryptWebhookSecret(webhookSecret string) string {
	if webhookSecret == "" {
		return ""
	}

	// Create AES cipher
	block, err := aes.NewCipher(c.encryptionKey)
	if err != nil {
		c.logger.Error("Failed to create cipher for webhook secret encryption", zap.Error(err))
		return ""
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		c.logger.Error("Failed to create GCM for webhook secret encryption", zap.Error(err))
		return ""
	}

	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		c.logger.Error("Failed to create nonce for webhook secret encryption", zap.Error(err))
		return ""
	}

	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, []byte(webhookSecret), nil)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

// decryptWebhookSecret decrypts a webhook secret using the same decryption as configuration
func (c *PaymentSyncClient) decryptWebhookSecret(encryptedSecret string) string {
	if encryptedSecret == "" {
		return ""
	}

	// Decode base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedSecret)
	if err != nil {
		c.logger.Error("Failed to decode webhook secret", zap.Error(err))
		return ""
	}

	// Create AES cipher
	block, err := aes.NewCipher(c.encryptionKey)
	if err != nil {
		c.logger.Error("Failed to create cipher for webhook secret decryption", zap.Error(err))
		return ""
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		c.logger.Error("Failed to create GCM for webhook secret decryption", zap.Error(err))
		return ""
	}

	if len(ciphertext) < gcm.NonceSize() {
		c.logger.Error("Webhook secret ciphertext too short")
		return ""
	}

	// Extract nonce and ciphertext
	nonce := ciphertext[:gcm.NonceSize()]
	ciphertext = ciphertext[gcm.NonceSize():]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		c.logger.Error("Failed to decrypt webhook secret", zap.Error(err))
		return ""
	}

	return string(plaintext)
}
