package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	ps "cyphera-api/internal/client/payment_sync"
	"cyphera-api/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// PaymentSyncHandlers handles all payment sync related API endpoints
// including both configuration management and sync operations
type PaymentSyncHandlers struct {
	db         *db.Queries
	logger     *zap.Logger
	syncClient *ps.PaymentSyncClient // Unified client for all operations
}

// NewPaymentSyncHandlers creates a new payment sync handlers instance
func NewPaymentSyncHandlers(dbQueries *db.Queries, logger *zap.Logger, syncClient *ps.PaymentSyncClient) *PaymentSyncHandlers {
	return &PaymentSyncHandlers{
		db:         dbQueries,
		logger:     logger,
		syncClient: syncClient,
	}
}

// Configuration Management Request/Response Types

type CreateConfigurationRequest struct {
	ProviderName       string                   `json:"provider_name" binding:"required"`
	IsActive           bool                     `json:"is_active"`
	IsTestMode         bool                     `json:"is_test_mode"`
	Configuration      ps.PaymentProviderConfig `json:"configuration" binding:"required"`
	WebhookEndpointURL string                   `json:"webhook_endpoint_url,omitempty"`
	ConnectedAccountID string                   `json:"connected_account_id,omitempty"`
	Metadata           map[string]interface{}   `json:"metadata,omitempty"`
}

type UpdateConfigurationRequest struct {
	IsActive           *bool                     `json:"is_active,omitempty"`
	IsTestMode         *bool                     `json:"is_test_mode,omitempty"`
	Configuration      *ps.PaymentProviderConfig `json:"configuration,omitempty"`
	WebhookEndpointURL *string                   `json:"webhook_endpoint_url,omitempty"`
	ConnectedAccountID *string                   `json:"connected_account_id,omitempty"`
	Metadata           map[string]interface{}    `json:"metadata,omitempty"`
}

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

// Sync Operation Request/Response Types

type InitialSyncRequest struct {
	EntityTypes   []string `json:"entity_types,omitempty"`
	BatchSize     int      `json:"batch_size,omitempty"`
	FullSync      bool     `json:"full_sync"`
	StartingAfter string   `json:"starting_after,omitempty"`
	EndingBefore  string   `json:"ending_before,omitempty"`
}

type InitialSyncResponse struct {
	SessionID   string                 `json:"session_id"`
	Status      string                 `json:"status"`
	Provider    string                 `json:"provider"`
	EntityTypes []string               `json:"entity_types"`
	Config      map[string]interface{} `json:"config"`
	CreatedAt   string                 `json:"created_at"`
}

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

// Configuration Management Handlers

// CreateConfiguration godoc
// @Summary Create payment provider configuration
// @Description Creates a new payment provider configuration for a workspace
// @Tags payment-sync
// @Accept json
// @Produce json
// @Param request body CreateConfigurationRequest true "Configuration details"
// @Success 201 {object} ConfigurationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /sync/config [post]
func (h *PaymentSyncHandlers) CreateConfiguration(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID == "" {
		h.logger.Error("No workspace ID found in header")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "workspace_id is required"})
		return
	}

	var req CreateConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind request", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request payload"})
		return
	}

	config := ps.WorkspacePaymentConfig{
		WorkspaceID:        workspaceID,
		ProviderName:       req.ProviderName,
		IsActive:           req.IsActive,
		IsTestMode:         req.IsTestMode,
		Configuration:      req.Configuration,
		WebhookEndpointURL: req.WebhookEndpointURL,
		ConnectedAccountID: req.ConnectedAccountID,
		Metadata:           req.Metadata,
	}

	createdConfig, err := h.syncClient.CreateConfiguration(c.Request.Context(), config)
	if err != nil {
		h.logger.Error("Failed to create configuration", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create configuration"})
		return
	}

	response := h.toConfigurationResponse(*createdConfig)
	c.JSON(http.StatusCreated, response)
}

// GetConfiguration godoc
// @Summary Get payment provider configuration
// @Description Retrieves a payment provider configuration by provider name
// @Tags payment-sync
// @Produce json
// @Param provider path string true "Provider name"
// @Success 200 {object} ConfigurationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /sync/config/{provider} [get]
func (h *PaymentSyncHandlers) GetConfiguration(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID == "" {
		h.logger.Error("No workspace ID found in header")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "workspace_id is required"})
		return
	}

	provider := c.Param("provider")
	if provider == "" {
		h.logger.Error("No provider specified")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "provider is required"})
		return
	}

	config, err := h.syncClient.GetConfiguration(c.Request.Context(), workspaceID, provider)
	if err != nil {
		h.logger.Error("Failed to get configuration", zap.Error(err))
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "configuration not found"})
		return
	}

	response := h.toConfigurationResponse(*config)
	c.JSON(http.StatusOK, response)
}

// GetConfigurationByID godoc
// @Summary Get payment provider configuration by ID
// @Description Retrieves a payment provider configuration by ID
// @Tags payment-sync
// @Produce json
// @Param config_id path string true "Configuration ID"
// @Success 200 {object} ConfigurationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /sync/config/id/{config_id} [get]
func (h *PaymentSyncHandlers) GetConfigurationByID(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID == "" {
		h.logger.Error("No workspace ID found in header")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "workspace_id is required"})
		return
	}

	configID := c.Param("config_id")
	if _, err := uuid.Parse(configID); err != nil {
		h.logger.Error("Invalid configuration ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid configuration ID"})
		return
	}

	config, err := h.syncClient.GetConfigurationByID(c.Request.Context(), workspaceID, configID)
	if err != nil {
		h.logger.Error("Failed to get configuration", zap.Error(err))
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "configuration not found"})
		return
	}

	response := h.toConfigurationResponse(*config)
	c.JSON(http.StatusOK, response)
}

// ListConfigurations godoc
// @Summary List payment provider configurations
// @Description Lists all payment provider configurations for a workspace
// @Tags payment-sync
// @Produce json
// @Param limit query int false "Number of results to return" default(50)
// @Param offset query int false "Number of results to skip" default(0)
// @Success 200 {object} object{configurations=[]ConfigurationResponse,total=int}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /sync/config [get]
func (h *PaymentSyncHandlers) ListConfigurations(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID == "" {
		h.logger.Error("No workspace ID found in header")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "workspace_id is required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	configs, err := h.syncClient.ListConfigurations(c.Request.Context(), workspaceID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list configurations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to list configurations"})
		return
	}

	responses := make([]ConfigurationResponse, len(configs))
	for i, config := range configs {
		responses[i] = h.toConfigurationResponse(config)
	}

	c.JSON(http.StatusOK, gin.H{
		"configurations": responses,
		"total":          len(responses),
		"limit":          limit,
		"offset":         offset,
	})
}

// UpdateConfiguration godoc
// @Summary Update payment provider configuration
// @Description Updates an existing payment provider configuration
// @Tags payment-sync
// @Accept json
// @Produce json
// @Param config_id path string true "Configuration ID"
// @Param request body UpdateConfigurationRequest true "Configuration updates"
// @Success 200 {object} ConfigurationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /sync/config/{config_id} [put]
func (h *PaymentSyncHandlers) UpdateConfiguration(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID == "" {
		h.logger.Error("No workspace ID found in header")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "workspace_id is required"})
		return
	}

	configID := c.Param("config_id")
	if _, err := uuid.Parse(configID); err != nil {
		h.logger.Error("Invalid configuration ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid configuration ID"})
		return
	}

	var req UpdateConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind request", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request payload"})
		return
	}

	// Get existing configuration to merge with updates
	existing, err := h.syncClient.GetConfigurationByID(c.Request.Context(), workspaceID, configID)
	if err != nil {
		h.logger.Error("Failed to get existing configuration", zap.Error(err))
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "configuration not found"})
		return
	}

	// Apply updates to existing configuration
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	if req.IsTestMode != nil {
		existing.IsTestMode = *req.IsTestMode
	}
	if req.Configuration != nil {
		existing.Configuration = *req.Configuration
	}
	if req.WebhookEndpointURL != nil {
		existing.WebhookEndpointURL = *req.WebhookEndpointURL
	}
	if req.ConnectedAccountID != nil {
		existing.ConnectedAccountID = *req.ConnectedAccountID
	}
	if req.Metadata != nil {
		existing.Metadata = req.Metadata
	}

	updatedConfig, err := h.syncClient.UpdateConfiguration(c.Request.Context(), workspaceID, configID, *existing)
	if err != nil {
		h.logger.Error("Failed to update configuration", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update configuration"})
		return
	}

	response := h.toConfigurationResponse(*updatedConfig)
	c.JSON(http.StatusOK, response)
}

// DeleteConfiguration godoc
// @Summary Delete payment provider configuration
// @Description Deletes a payment provider configuration
// @Tags payment-sync
// @Produce json
// @Param config_id path string true "Configuration ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /sync/config/{config_id} [delete]
func (h *PaymentSyncHandlers) DeleteConfiguration(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID == "" {
		h.logger.Error("No workspace ID found in header")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "workspace_id is required"})
		return
	}

	configID := c.Param("config_id")
	if _, err := uuid.Parse(configID); err != nil {
		h.logger.Error("Invalid configuration ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid configuration ID"})
		return
	}

	err := h.syncClient.DeleteConfiguration(c.Request.Context(), workspaceID, configID)
	if err != nil {
		h.logger.Error("Failed to delete configuration", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to delete configuration"})
		return
	}

	c.Status(http.StatusNoContent)
}

// TestConnection godoc
// @Summary Test payment provider connection
// @Description Tests the connection to a payment provider using the configuration
// @Tags payment-sync
// @Produce json
// @Param config_id path string true "Configuration ID"
// @Success 200 {object} object{status=string,message=string}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /sync/config/{config_id}/test [post]
func (h *PaymentSyncHandlers) TestConnection(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID == "" {
		h.logger.Error("No workspace ID found in header")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "workspace_id is required"})
		return
	}

	configID := c.Param("config_id")
	if _, err := uuid.Parse(configID); err != nil {
		h.logger.Error("Invalid configuration ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid configuration ID"})
		return
	}

	err := h.syncClient.TestConnection(c.Request.Context(), workspaceID, configID)
	if err != nil {
		h.logger.Error("Connection test failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "connection test failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "connection test successful",
	})
}

// GetAvailableProviders godoc
// @Summary Get available payment providers
// @Description Lists all available payment providers that can be configured
// @Tags payment-sync
// @Produce json
// @Success 200 {object} object{providers=[]string}
// @Security ApiKeyAuth
// @Router /sync/providers [get]
func (h *PaymentSyncHandlers) GetAvailableProviders(c *gin.Context) {
	providers := h.syncClient.GetAvailableProviders()
	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
	})
}

// Sync Operation Handlers

// StartInitialSync handles POST /sync/{provider}/initial
// @Summary Start initial sync for any payment provider
// @Description Initiates an initial data synchronization from payment provider to Cyphera
// @Tags payment-sync
// @Accept json
// @Produce json
// @Param provider path string true "Payment provider name (e.g., stripe, chargebee)"
// @Param request body InitialSyncRequest true "Initial sync configuration"
// @Success 201 {object} InitialSyncResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sync/{provider}/initial [post]
func (h *PaymentSyncHandlers) StartInitialSync(c *gin.Context) {
	// Get workspace ID from header (following existing pattern)
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID == "" {
		h.logger.Error("No workspace ID found in header")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "workspace_id is required"})
		return
	}

	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		h.logger.Error("Invalid workspace ID format", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid workspace_id"})
		return
	}

	// Get provider from URL path
	provider := c.Param("provider")
	if provider == "" {
		h.logger.Error("No provider specified in URL")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "provider is required"})
		return
	}

	var req InitialSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind request", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request payload"})
		return
	}

	// Use defaults if not provided - convert to generic config
	config := ps.InitialSyncConfig{
		BatchSize:   100,
		EntityTypes: []string{"customers", "products", "prices", "subscriptions"},
		FullSync:    req.FullSync,
		MaxRetries:  3,
		RetryDelay:  2, // seconds
	}

	if len(req.EntityTypes) > 0 {
		config.EntityTypes = req.EntityTypes
	}
	if req.BatchSize > 0 {
		config.BatchSize = req.BatchSize
	}
	if req.StartingAfter != "" {
		config.StartingAfter = req.StartingAfter
	}
	if req.EndingBefore != "" {
		config.EndingBefore = req.EndingBefore
	}

	h.logger.Info("Starting initial sync",
		zap.String("provider", provider),
		zap.String("workspace_id", wsID.String()),
		zap.Any("config", config))

	session, err := h.syncClient.StartInitialSync(c.Request.Context(), wsID.String(), provider, config)
	if err != nil {
		h.logger.Error("Failed to start initial sync", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to start initial sync"})
		return
	}

	response := InitialSyncResponse{
		SessionID:   session.ID,
		Status:      session.Status,
		Provider:    session.Provider,
		EntityTypes: session.EntityTypes,
		Config:      session.Config,
		CreatedAt:   time.Unix(session.CreatedAt, 0).Format("2006-01-02T15:04:05Z"),
	}

	h.logger.Info("Successfully started initial sync",
		zap.String("session_id", session.ID),
		zap.String("workspace_id", wsID.String()))

	c.JSON(http.StatusCreated, response)
}

// GetSyncSession handles GET /sync/sessions/:id
// @Summary Get sync session details
// @Description Retrieves details of a specific sync session
// @Tags payment-sync
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} SyncSessionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sync/sessions/{id} [get]
func (h *PaymentSyncHandlers) GetSyncSession(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID == "" {
		h.logger.Error("No workspace ID found in header")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "workspace_id is required"})
		return
	}

	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		h.logger.Error("Invalid workspace ID format", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid workspace_id"})
		return
	}

	sessionIDStr := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.logger.Error("Invalid session ID", zap.String("session_id", sessionIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid session_id"})
		return
	}

	session, err := h.db.GetSyncSession(c.Request.Context(), db.GetSyncSessionParams{
		ID:          sessionID,
		WorkspaceID: wsID,
	})
	if err != nil {
		h.logger.Error("Failed to get sync session", zap.Error(err))
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "sync session not found"})
		return
	}

	response := h.mapSyncSessionToResponse(session)
	c.JSON(http.StatusOK, response)
}

// ListSyncSessions handles GET /sync/sessions
// @Summary List sync sessions
// @Description Retrieves a list of sync sessions for the workspace
// @Tags payment-sync
// @Produce json
// @Param provider query string false "Filter by provider name"
// @Param status query string false "Filter by status"
// @Param limit query int false "Number of results to return" default(50)
// @Param offset query int false "Number of results to skip" default(0)
// @Success 200 {object} object{sessions=[]SyncSessionResponse,total=int}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sync/sessions [get]
func (h *PaymentSyncHandlers) ListSyncSessions(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID == "" {
		h.logger.Error("No workspace ID found in header")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "workspace_id is required"})
		return
	}

	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		h.logger.Error("Invalid workspace ID format", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid workspace_id"})
		return
	}

	// Parse query parameters
	provider := c.Query("provider")
	status := c.Query("status")
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	var sessions []db.PaymentSyncSession
	var total int64

	ctx := c.Request.Context()

	// Get sessions based on filters
	if provider != "" {
		sessions, err = h.db.ListSyncSessionsByProvider(ctx, db.ListSyncSessionsByProviderParams{
			WorkspaceID:  wsID,
			ProviderName: provider,
			Limit:        int32(limit),
			Offset:       int32(offset),
		})
		if err == nil {
			total, _ = h.db.CountSyncSessionsByProvider(ctx, db.CountSyncSessionsByProviderParams{
				WorkspaceID:  wsID,
				ProviderName: provider,
			})
		}
	} else if status != "" {
		sessions, err = h.db.ListSyncSessionsByStatus(ctx, db.ListSyncSessionsByStatusParams{
			WorkspaceID: wsID,
			Status:      status,
			Limit:       int32(limit),
			Offset:      int32(offset),
		})
	} else {
		sessions, err = h.db.ListSyncSessions(ctx, db.ListSyncSessionsParams{
			WorkspaceID: wsID,
			Limit:       int32(limit),
			Offset:      int32(offset),
		})
		if err == nil {
			total, _ = h.db.CountSyncSessions(ctx, wsID)
		}
	}

	if err != nil {
		h.logger.Error("Failed to list sync sessions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to list sync sessions"})
		return
	}

	// Convert to response format
	responses := make([]SyncSessionResponse, len(sessions))
	for i, session := range sessions {
		responses[i] = h.mapSyncSessionToResponse(session)
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": responses,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetSyncSessionStatus handles GET /sync/sessions/:id/status
// @Summary Get sync session status
// @Description Retrieves the current status and progress of a sync session
// @Tags payment-sync
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} object{status=string,progress=object,error_summary=object}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sync/sessions/{id}/status [get]
func (h *PaymentSyncHandlers) GetSyncSessionStatus(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID == "" {
		h.logger.Error("No workspace ID found in header")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "workspace_id is required"})
		return
	}

	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		h.logger.Error("Invalid workspace ID format", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid workspace_id"})
		return
	}

	sessionIDStr := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.logger.Error("Invalid session ID", zap.String("session_id", sessionIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid session_id"})
		return
	}

	session, err := h.db.GetSyncSession(c.Request.Context(), db.GetSyncSessionParams{
		ID:          sessionID,
		WorkspaceID: wsID,
	})
	if err != nil {
		h.logger.Error("Failed to get sync session", zap.Error(err))
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "sync session not found"})
		return
	}

	// Parse progress and error summary
	var progress, errorSummary map[string]interface{}

	if len(session.Progress) > 0 {
		if err := json.Unmarshal(session.Progress, &progress); err != nil {
			h.logger.Warn("Failed to unmarshal progress", zap.Error(err))
			progress = make(map[string]interface{})
		}
	}

	if len(session.ErrorSummary) > 0 {
		if err := json.Unmarshal(session.ErrorSummary, &errorSummary); err != nil {
			h.logger.Warn("Failed to unmarshal error summary", zap.Error(err))
			errorSummary = make(map[string]interface{})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        session.Status,
		"progress":      progress,
		"error_summary": errorSummary,
		"started_at":    h.formatNullTimestamp(session.StartedAt),
		"completed_at":  h.formatNullTimestamp(session.CompletedAt),
	})
}

// Helper methods

// toConfigurationResponse converts a workspace payment config to API response format (excludes sensitive data)
func (h *PaymentSyncHandlers) toConfigurationResponse(config ps.WorkspacePaymentConfig) ConfigurationResponse {
	return ConfigurationResponse{
		ID:                 config.ID,
		WorkspaceID:        config.WorkspaceID,
		ProviderName:       config.ProviderName,
		IsActive:           config.IsActive,
		IsTestMode:         config.IsTestMode,
		WebhookEndpointURL: config.WebhookEndpointURL,
		ConnectedAccountID: config.ConnectedAccountID,
		LastSyncAt:         config.LastSyncAt,
		LastWebhookAt:      config.LastWebhookAt,
		Metadata:           config.Metadata,
		CreatedAt:          config.CreatedAt,
		UpdatedAt:          config.UpdatedAt,
		// Note: Configuration field is intentionally omitted for security
	}
}

// mapSyncSessionToResponse converts a database PaymentSyncSession to API response format
func (h *PaymentSyncHandlers) mapSyncSessionToResponse(session db.PaymentSyncSession) SyncSessionResponse {
	response := SyncSessionResponse{
		ID:          session.ID.String(),
		WorkspaceID: session.WorkspaceID.String(),
		Provider:    session.ProviderName,
		SessionType: session.SessionType,
		Status:      session.Status,
		EntityTypes: session.EntityTypes,
		CreatedAt:   session.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   session.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}

	// Parse JSON fields
	if len(session.Config) > 0 {
		if err := json.Unmarshal(session.Config, &response.Config); err != nil {
			h.logger.Warn("Failed to unmarshal config", zap.Error(err))
		}
	}

	if len(session.Progress) > 0 {
		if err := json.Unmarshal(session.Progress, &response.Progress); err != nil {
			h.logger.Warn("Failed to unmarshal progress", zap.Error(err))
		}
	}

	if len(session.ErrorSummary) > 0 {
		if err := json.Unmarshal(session.ErrorSummary, &response.ErrorSummary); err != nil {
			h.logger.Warn("Failed to unmarshal error summary", zap.Error(err))
		}
	}

	// Format nullable timestamps
	response.StartedAt = h.formatNullTimestamp(session.StartedAt)
	response.CompletedAt = h.formatNullTimestamp(session.CompletedAt)

	return response
}

// formatNullTimestamp formats a nullable timestamp for API response
func (h *PaymentSyncHandlers) formatNullTimestamp(ts pgtype.Timestamptz) *string {
	if ts.Valid {
		formatted := ts.Time.Format("2006-01-02T15:04:05Z")
		return &formatted
	}
	return nil
}

// CreateProviderAccount creates a provider account mapping for webhook routing
// @godoc CreateProviderAccount creates a provider account mapping for webhook routing
func (h *PaymentSyncHandlers) CreateProviderAccount(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "X-Workspace-ID header is required"})
		return
	}

	var req CreateProviderAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	// Set defaults
	if req.Environment == "" {
		req.Environment = "live"
	}
	if req.AccountType == "" {
		req.AccountType = "standard"
	}

	// Validate workspace ID format
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid workspace ID format"})
		return
	}

	// Serialize metadata
	var metadataBytes []byte
	if req.Metadata != nil {
		metadataBytes, err = json.Marshal(req.Metadata)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to serialize metadata"})
			return
		}
	} else {
		metadataBytes = []byte("{}")
	}

	// Create provider account
	providerAccount, err := h.db.CreateWorkspaceProviderAccount(c.Request.Context(), db.CreateWorkspaceProviderAccountParams{
		WorkspaceID:       wsID,
		ProviderName:      req.ProviderName,
		ProviderAccountID: req.ProviderAccountID,
		AccountType:       req.AccountType,
		IsActive:          req.IsActive,
		Environment:       req.Environment,
		DisplayName:       pgtype.Text{String: req.DisplayName, Valid: req.DisplayName != ""},
		Metadata:          metadataBytes,
	})
	if err != nil {
		h.logger.Error("Failed to create provider account", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create provider account"})
		return
	}

	response := h.mapProviderAccountToResponse(providerAccount)
	c.JSON(http.StatusCreated, response)
}

// GetProviderAccounts lists provider accounts for a workspace
// @godoc GetProviderAccounts lists provider accounts for a workspace
func (h *PaymentSyncHandlers) GetProviderAccounts(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "X-Workspace-ID header is required"})
		return
	}

	// Validate workspace ID format
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid workspace ID format"})
		return
	}

	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Get provider accounts
	accounts, err := h.db.ListProviderAccountsByWorkspace(c.Request.Context(), db.ListProviderAccountsByWorkspaceParams{
		WorkspaceID: wsID,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		h.logger.Error("Failed to list provider accounts", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to list provider accounts"})
		return
	}

	// Convert to response format
	responses := make([]ProviderAccountResponse, len(accounts))
	for i, account := range accounts {
		responses[i] = h.mapProviderAccountToResponse(account)
	}

	c.JSON(http.StatusOK, gin.H{
		"provider_accounts": responses,
		"total":             len(responses),
		"limit":             limit,
		"offset":            offset,
	})
}

// mapProviderAccountToResponse converts DB model to API response
func (h *PaymentSyncHandlers) mapProviderAccountToResponse(account db.WorkspaceProviderAccount) ProviderAccountResponse {
	var metadata map[string]interface{}
	if len(account.Metadata) > 0 {
		json.Unmarshal(account.Metadata, &metadata)
	}

	response := ProviderAccountResponse{
		ID:                account.ID.String(),
		WorkspaceID:       account.WorkspaceID.String(),
		ProviderName:      account.ProviderName,
		ProviderAccountID: account.ProviderAccountID,
		AccountType:       account.AccountType,
		IsActive:          account.IsActive,
		Environment:       account.Environment,
		Metadata:          metadata,
		CreatedAt:         account.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:         account.UpdatedAt.Time.Format(time.RFC3339),
	}

	if account.DisplayName.Valid {
		response.DisplayName = account.DisplayName.String
	}

	return response
}
