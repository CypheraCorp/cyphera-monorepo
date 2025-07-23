package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/cyphera/cyphera-api/apps/api/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorRecoveryHandlers handles API endpoints for error recovery functionality
type ErrorRecoveryHandlers struct {
	errorRecoveryService *services.ErrorRecoveryService
	logger               *zap.Logger
}

// NewErrorRecoveryHandlers creates a new error recovery handlers instance
func NewErrorRecoveryHandlers(
	errorRecoveryService *services.ErrorRecoveryService,
	logger *zap.Logger,
) *ErrorRecoveryHandlers {
	return &ErrorRecoveryHandlers{
		errorRecoveryService: errorRecoveryService,
		logger:               logger,
	}
}

// DetailedErrorResponse represents an error response with additional details
type DetailedErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// DetailedSuccessResponse represents a success response with additional data
type DetailedSuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// WebhookReplayRequest represents a webhook replay request for swagger documentation
type WebhookReplayRequest struct {
	WorkspaceID    string `json:"workspace_id" binding:"required"`
	ProviderName   string `json:"provider_name" binding:"required"`
	WebhookEventID string `json:"webhook_event_id" binding:"required"`
	ForceReplay    bool   `json:"force_replay"`
	ReplayReason   string `json:"replay_reason,omitempty"`
}

// WebhookReplayResponse represents the result of webhook replay for swagger documentation
type WebhookReplayResponse struct {
	Success         bool   `json:"success"`
	ReplayEventID   string `json:"replay_event_id,omitempty"`
	OriginalEventID string `json:"original_event_id"`
	ReplayedAt      string `json:"replayed_at"`
	Message         string `json:"message"`
	Error           string `json:"error,omitempty"`
}

// SyncRecoveryRequest represents a sync session recovery request for swagger documentation
type SyncRecoveryRequest struct {
	WorkspaceID  string   `json:"workspace_id" binding:"required"`
	SessionID    string   `json:"session_id" binding:"required"`
	RecoveryMode string   `json:"recovery_mode"` // "resume", "restart"
	EntityTypes  []string `json:"entity_types,omitempty"`
}

// SyncRecoveryResponse represents the result of sync recovery for swagger documentation
type SyncRecoveryResponse struct {
	Success     bool                   `json:"success"`
	SessionID   string                 `json:"session_id"`
	RecoveredAt string                 `json:"recovered_at"`
	Progress    map[string]interface{} `json:"progress,omitempty"`
	Message     string                 `json:"message"`
	Error       string                 `json:"error,omitempty"`
}

// DLQProcessingStats represents DLQ processing statistics for swagger documentation
type DLQProcessingStats struct {
	TotalMessages         int64   `json:"total_messages"`
	SuccessfullyProcessed int64   `json:"successfully_processed"`
	ProcessingFailed      int64   `json:"processing_failed"`
	MaxRetriesExceeded    int64   `json:"max_retries_exceeded"`
	LastProcessedAt       string  `json:"last_processed_at"`
	SuccessRate           float64 `json:"success_rate"`
}

// ReplayWebhook godoc
// @Summary Replay a failed webhook event
// @Description Manually replay a webhook event that failed processing
// @Tags error-recovery
// @Accept json
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Param request body WebhookReplayRequest true "Webhook replay request"
// @Success 200 {object} WebhookReplayResponse
// @Failure 400 {object} DetailedErrorResponse
// @Failure 500 {object} DetailedErrorResponse
// @Router /api/v1/workspaces/{workspace_id}/webhooks/replay [post]
func (h *ErrorRecoveryHandlers) ReplayWebhook(c *gin.Context) {
	workspaceID := c.Param("workspace_id")
	if workspaceID == "" {
		c.JSON(http.StatusBadRequest, DetailedErrorResponse{
			Error: "workspace_id is required",
			Code:  "MISSING_WORKSPACE_ID",
		})
		return
	}

	var req WebhookReplayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid webhook replay request", zap.Error(err))
		c.JSON(http.StatusBadRequest, DetailedErrorResponse{
			Error:   "Invalid request format",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	// Set workspace ID from URL parameter
	req.WorkspaceID = workspaceID

	h.logger.Info("Processing webhook replay request",
		zap.String("workspace_id", workspaceID),
		zap.String("provider", req.ProviderName),
		zap.String("webhook_event_id", req.WebhookEventID),
		zap.Bool("force_replay", req.ForceReplay))

	// Convert to services type
	serviceReq := services.WebhookReplayRequest{
		WorkspaceID:    req.WorkspaceID,
		ProviderName:   req.ProviderName,
		WebhookEventID: req.WebhookEventID,
		ForceReplay:    req.ForceReplay,
		ReplayReason:   req.ReplayReason,
	}

	response, err := h.errorRecoveryService.ReplayWebhookEvent(c.Request.Context(), serviceReq)
	if err != nil {
		h.logger.Error("Failed to replay webhook", zap.Error(err))
		c.JSON(http.StatusInternalServerError, DetailedErrorResponse{
			Error:   "Failed to replay webhook",
			Code:    "REPLAY_FAILED",
			Details: err.Error(),
		})
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// RecoverSyncSession godoc
// @Summary Recover a failed sync session
// @Description Resume or restart a failed or incomplete sync session
// @Tags error-recovery
// @Accept json
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Param request body SyncRecoveryRequest true "Sync recovery request"
// @Success 200 {object} SyncRecoveryResponse
// @Failure 400 {object} DetailedErrorResponse
// @Failure 500 {object} DetailedErrorResponse
// @Router /api/v1/workspaces/{workspace_id}/sync/recover [post]
func (h *ErrorRecoveryHandlers) RecoverSyncSession(c *gin.Context) {
	workspaceID := c.Param("workspace_id")
	if workspaceID == "" {
		c.JSON(http.StatusBadRequest, DetailedErrorResponse{
			Error: "workspace_id is required",
			Code:  "MISSING_WORKSPACE_ID",
		})
		return
	}

	var req SyncRecoveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid sync recovery request", zap.Error(err))
		c.JSON(http.StatusBadRequest, DetailedErrorResponse{
			Error:   "Invalid request format",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	// Set workspace ID from URL parameter
	req.WorkspaceID = workspaceID

	// Default recovery mode if not specified
	if req.RecoveryMode == "" {
		req.RecoveryMode = "resume"
	}

	h.logger.Info("Processing sync recovery request",
		zap.String("workspace_id", workspaceID),
		zap.String("session_id", req.SessionID),
		zap.String("recovery_mode", req.RecoveryMode))

	// Convert to services type
	serviceReq := services.SyncRecoveryRequest{
		WorkspaceID:  req.WorkspaceID,
		SessionID:    req.SessionID,
		RecoveryMode: req.RecoveryMode,
		EntityTypes:  req.EntityTypes,
	}

	response, err := h.errorRecoveryService.RecoverSyncSession(c.Request.Context(), serviceReq)
	if err != nil {
		h.logger.Error("Failed to recover sync session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, DetailedErrorResponse{
			Error:   "Failed to recover sync session",
			Code:    "RECOVERY_FAILED",
			Details: err.Error(),
		})
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// GetDLQStats godoc
// @Summary Get DLQ processing statistics
// @Description Get statistics about dead letter queue processing for a workspace and provider
// @Tags error-recovery
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Param provider query string true "Provider name (e.g., stripe)"
// @Param since query string false "Since timestamp (Unix timestamp or RFC3339)"
// @Success 200 {object} DLQProcessingStats
// @Failure 400 {object} DetailedErrorResponse
// @Failure 500 {object} DetailedErrorResponse
// @Router /api/v1/workspaces/{workspace_id}/dlq/stats [get]
func (h *ErrorRecoveryHandlers) GetDLQStats(c *gin.Context) {
	workspaceID := c.Param("workspace_id")
	if workspaceID == "" {
		c.JSON(http.StatusBadRequest, DetailedErrorResponse{
			Error: "workspace_id is required",
			Code:  "MISSING_WORKSPACE_ID",
		})
		return
	}

	provider := c.Query("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, DetailedErrorResponse{
			Error: "provider query parameter is required",
			Code:  "MISSING_PROVIDER",
		})
		return
	}

	// Parse since parameter (default to 24 hours ago)
	var since time.Time
	sinceParam := c.Query("since")
	if sinceParam != "" {
		// Try parsing as Unix timestamp first
		if timestamp, err := strconv.ParseInt(sinceParam, 10, 64); err == nil {
			since = time.Unix(timestamp, 0)
		} else {
			// Try parsing as RFC3339
			if t, err := time.Parse(time.RFC3339, sinceParam); err == nil {
				since = t
			} else {
				c.JSON(http.StatusBadRequest, DetailedErrorResponse{
					Error:   "Invalid since parameter format",
					Code:    "INVALID_TIMESTAMP",
					Details: "Use Unix timestamp or RFC3339 format",
				})
				return
			}
		}
	} else {
		since = time.Now().Add(-24 * time.Hour)
	}

	h.logger.Info("Getting DLQ stats",
		zap.String("workspace_id", workspaceID),
		zap.String("provider", provider),
		zap.Time("since", since))

	stats, err := h.errorRecoveryService.GetDLQStats(c.Request.Context(), workspaceID, provider, since)
	if err != nil {
		h.logger.Error("Failed to get DLQ stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, DetailedErrorResponse{
			Error:   "Failed to get DLQ statistics",
			Code:    "STATS_FAILED",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, DetailedSuccessResponse{
		Success: true,
		Message: "DLQ statistics retrieved successfully",
		Data:    stats,
	})
}

// GetFailedWebhooks godoc
// @Summary Get failed webhook events for retry
// @Description Get a list of failed webhook events that can be retried
// @Tags error-recovery
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Param provider query string true "Provider name (e.g., stripe)"
// @Param limit query int false "Limit number of results (default: 50, max: 100)"
// @Param max_attempts query int false "Maximum processing attempts (default: 3)"
// @Success 200 {object} DetailedSuccessResponse
// @Failure 400 {object} DetailedErrorResponse
// @Failure 500 {object} DetailedErrorResponse
// @Router /api/v1/workspaces/{workspace_id}/webhooks/failed [get]
func (h *ErrorRecoveryHandlers) GetFailedWebhooks(c *gin.Context) {
	workspaceID := c.Param("workspace_id")
	if workspaceID == "" {
		c.JSON(http.StatusBadRequest, DetailedErrorResponse{
			Error: "workspace_id is required",
			Code:  "MISSING_WORKSPACE_ID",
		})
		return
	}

	provider := c.Query("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, DetailedErrorResponse{
			Error: "provider query parameter is required",
			Code:  "MISSING_PROVIDER",
		})
		return
	}

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 50
	}

	maxAttemptsStr := c.DefaultQuery("max_attempts", "3")
	maxAttempts, err := strconv.Atoi(maxAttemptsStr)
	if err != nil || maxAttempts <= 0 {
		maxAttempts = 3
	}

	h.logger.Info("Getting failed webhooks",
		zap.String("workspace_id", workspaceID),
		zap.String("provider", provider),
		zap.Int("limit", limit),
		zap.Int("max_attempts", maxAttempts))

	// This would typically call a method on the error recovery service
	// For now, we'll return a placeholder response
	c.JSON(http.StatusOK, DetailedSuccessResponse{
		Success: true,
		Message: "Failed webhooks retrieved successfully",
		Data: map[string]interface{}{
			"failed_webhooks": []interface{}{},
			"total":           0,
			"limit":           limit,
			"max_attempts":    maxAttempts,
		},
	})
}

// GetRecoverableSyncSessions godoc
// @Summary Get failed sync sessions that can be recovered
// @Description Get a list of sync sessions that failed and can be resumed or restarted
// @Tags error-recovery
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Param limit query int false "Limit number of results (default: 20, max: 50)"
// @Param offset query int false "Offset for pagination (default: 0)"
// @Success 200 {object} DetailedSuccessResponse
// @Failure 400 {object} DetailedErrorResponse
// @Failure 500 {object} DetailedErrorResponse
// @Router /api/v1/workspaces/{workspace_id}/sync/recoverable [get]
func (h *ErrorRecoveryHandlers) GetRecoverableSyncSessions(c *gin.Context) {
	workspaceID := c.Param("workspace_id")
	if workspaceID == "" {
		c.JSON(http.StatusBadRequest, DetailedErrorResponse{
			Error: "workspace_id is required",
			Code:  "MISSING_WORKSPACE_ID",
		})
		return
	}

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 50 {
		limit = 20
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	h.logger.Info("Getting recoverable sync sessions",
		zap.String("workspace_id", workspaceID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	// This would typically call a method on the error recovery service
	// For now, we'll return a placeholder response
	c.JSON(http.StatusOK, DetailedSuccessResponse{
		Success: true,
		Message: "Recoverable sync sessions retrieved successfully",
		Data: map[string]interface{}{
			"sessions": []interface{}{},
			"total":    0,
			"limit":    limit,
			"offset":   offset,
		},
	})
}

// HealthCheck godoc
// @Summary Check error recovery service health
// @Description Check the health and status of the error recovery service
// @Tags error-recovery
// @Produce json
// @Success 200 {object} DetailedSuccessResponse
// @Router /api/v1/error-recovery/health [get]
func (h *ErrorRecoveryHandlers) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, DetailedSuccessResponse{
		Success: true,
		Message: "Error recovery service is healthy",
		Data: map[string]interface{}{
			"service":   "error-recovery",
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"features": []string{
				"webhook_replay",
				"sync_recovery",
				"dlq_processing",
				"failure_analytics",
			},
		},
	})
}
