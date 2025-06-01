package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"cyphera-api/internal/client/payment_sync/stripe"
	"cyphera-api/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// PaymentSyncHandlers handles payment sync related API endpoints
type PaymentSyncHandlers struct {
	db                 *db.Queries
	logger             *zap.Logger
	stripeService      *stripe.StripeService
	initialSyncService *stripe.InitialSyncService
}

// NewPaymentSyncHandlers creates a new payment sync handlers instance
func NewPaymentSyncHandlers(dbQueries *db.Queries, logger *zap.Logger, stripeService *stripe.StripeService) *PaymentSyncHandlers {
	initialSyncService := stripe.NewInitialSyncService(stripeService, dbQueries, logger)

	return &PaymentSyncHandlers{
		db:                 dbQueries,
		logger:             logger,
		stripeService:      stripeService,
		initialSyncService: initialSyncService,
	}
}

// InitialSyncRequest represents the request payload for triggering initial sync
type InitialSyncRequest struct {
	EntityTypes   []string `json:"entity_types,omitempty"`
	BatchSize     int      `json:"batch_size,omitempty"`
	FullSync      bool     `json:"full_sync"`
	StartingAfter string   `json:"starting_after,omitempty"`
	EndingBefore  string   `json:"ending_before,omitempty"`
}

// InitialSyncResponse represents the response for initial sync requests
type InitialSyncResponse struct {
	SessionID   string                 `json:"session_id"`
	Status      string                 `json:"status"`
	Provider    string                 `json:"provider"`
	EntityTypes []string               `json:"entity_types"`
	Config      map[string]interface{} `json:"config"`
	CreatedAt   string                 `json:"created_at"`
}

// SyncSessionResponse represents a sync session in API responses
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

// StartStripeInitialSync handles POST /api/sync/stripe/initial
// @Summary Start Stripe initial sync
// @Description Initiates an initial data synchronization from Stripe to Cyphera
// @Tags payment-sync
// @Accept json
// @Produce json
// @Param request body InitialSyncRequest true "Initial sync configuration"
// @Success 201 {object} InitialSyncResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sync/stripe/initial [post]
func (h *PaymentSyncHandlers) StartStripeInitialSync(c *gin.Context) {
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

	var req InitialSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind request", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request payload"})
		return
	}

	// Use defaults if not provided
	config := stripe.DefaultInitialSyncConfig()
	if len(req.EntityTypes) > 0 {
		config.EntityTypes = req.EntityTypes
	}
	if req.BatchSize > 0 {
		config.BatchSize = req.BatchSize
	}
	config.FullSync = req.FullSync
	if req.StartingAfter != "" {
		config.StartingAfter = req.StartingAfter
	}
	if req.EndingBefore != "" {
		config.EndingBefore = req.EndingBefore
	}

	h.logger.Info("Starting Stripe initial sync",
		zap.String("workspace_id", wsID.String()),
		zap.Any("config", config))

	session, err := h.initialSyncService.StartInitialSync(c.Request.Context(), wsID, config)
	if err != nil {
		h.logger.Error("Failed to start initial sync", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to start initial sync"})
		return
	}

	// Parse config for response
	var configMap map[string]interface{}
	if err := json.Unmarshal(session.Config, &configMap); err != nil {
		h.logger.Warn("Failed to unmarshal config for response", zap.Error(err))
		configMap = make(map[string]interface{})
	}

	response := InitialSyncResponse{
		SessionID:   session.ID.String(),
		Status:      session.Status,
		Provider:    session.ProviderName,
		EntityTypes: session.EntityTypes,
		Config:      configMap,
		CreatedAt:   session.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}

	h.logger.Info("Successfully started initial sync",
		zap.String("session_id", session.ID.String()),
		zap.String("workspace_id", wsID.String()))

	c.JSON(http.StatusCreated, response)
}

// GetSyncSession handles GET /api/sync/sessions/:id
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

// ListSyncSessions handles GET /api/sync/sessions
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

// GetSyncSessionStatus handles GET /api/sync/sessions/:id/status
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
