package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/services"
)

type DunningHandler struct {
	common         *CommonServices
	dunningService interfaces.DunningService
	retryEngine    interfaces.DunningRetryEngine
}

// NewDunningHandler creates a handler with interface dependencies
func NewDunningHandler(
	common *CommonServices,
	dunningService interfaces.DunningService,
	retryEngine interfaces.DunningRetryEngine,
) *DunningHandler {
	return &DunningHandler{
		common:         common,
		dunningService: dunningService,
		retryEngine:    retryEngine,
	}
}

// Configuration endpoints

// CreateConfiguration creates a new dunning configuration
// @Summary Create dunning configuration
// @Description Create a new dunning configuration for the workspace
// @Tags dunning
// @Accept json
// @Produce json
// @Param configuration body CreateConfigurationRequest true "Configuration details"
// @Success 201 {object} db.DunningConfiguration
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/dunning/configurations [post]
func (h *DunningHandler) CreateConfiguration(c *gin.Context) {
	var req CreateDunningConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.dunningService.CreateConfiguration(c.Request.Context(), services.DunningConfigParams{
		WorkspaceID:            workspaceID,
		Name:                   req.Name,
		Description:            req.Description,
		IsActive:               req.IsActive,
		IsDefault:              req.IsDefault,
		MaxRetryAttempts:       req.MaxRetryAttempts,
		RetryIntervalDays:      req.RetryIntervalDays,
		AttemptActions:         req.AttemptActions,
		FinalAction:            req.FinalAction,
		FinalActionConfig:      req.FinalActionConfig,
		SendPreDunningReminder: req.SendPreDunningReminder,
		PreDunningDays:         req.PreDunningDays,
		AllowCustomerRetry:     req.AllowCustomerRetry,
		GracePeriodHours:       req.GracePeriodHours,
	})
	if err != nil {
		h.common.logger.Error("failed to create dunning configuration", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create configuration"})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// GetConfiguration gets a dunning configuration by ID
// @Summary Get dunning configuration
// @Description Get a dunning configuration by ID
// @Tags dunning
// @Produce json
// @Param id path string true "Configuration ID"
// @Success 200 {object} db.DunningConfiguration
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/dunning/configurations/{id} [get]
func (h *DunningHandler) GetConfiguration(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid configuration ID"})
		return
	}

	config, err := h.dunningService.GetConfiguration(c.Request.Context(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Configuration not found"})
			return
		}
		h.common.logger.Error("failed to get dunning configuration", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get configuration"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// ListConfigurations lists all dunning configurations for a workspace
// @Summary List dunning configurations
// @Description List all dunning configurations for the workspace
// @Tags dunning
// @Produce json
// @Success 200 {array} db.DunningConfiguration
// @Failure 500 {object} map[string]string
// @Router /api/v1/dunning/configurations [get]
func (h *DunningHandler) ListConfigurations(c *gin.Context) {
	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	configs, err := h.common.db.ListDunningConfigurations(c.Request.Context(), workspaceID)
	if err != nil {
		h.common.logger.Error("failed to list dunning configurations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list configurations"})
		return
	}

	c.JSON(http.StatusOK, configs)
}

// Campaign endpoints

// ListCampaigns lists dunning campaigns with filtering
// @Summary List dunning campaigns
// @Description List dunning campaigns with optional filtering
// @Tags dunning
// @Produce json
// @Param status query string false "Filter by status"
// @Param customer_id query string false "Filter by customer ID"
// @Param limit query int false "Limit results" default(20)
// @Param offset query int false "Offset results" default(0)
// @Success 200 {array} DunningCampaignResponse
// @Failure 500 {object} map[string]string
// @Router /api/v1/dunning/campaigns [get]
func (h *DunningHandler) ListCampaigns(c *gin.Context) {
	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit, offset := GetPaginationParams(c)

	// Parse filters
	var status *string
	if s := c.Query("status"); s != "" {
		status = &s
	}

	var customerID *uuid.UUID
	if cid := c.Query("customer_id"); cid != "" {
		parsed, err := uuid.Parse(cid)
		if err == nil {
			customerID = &parsed
		}
	}

	params := db.ListDunningCampaignsParams{
		WorkspaceID: workspaceID,
		Limit:       int32(limit),
		Offset:      int32(offset),
	}

	// Set optional parameters
	if status != nil {
		params.Status = *status
	}
	if customerID != nil {
		params.CustomerID = *customerID
	}

	campaigns, err := h.common.db.ListDunningCampaigns(c.Request.Context(), params)
	if err != nil {
		h.common.logger.Error("failed to list dunning campaigns", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list campaigns"})
		return
	}

	// Convert to response format
	responses := make([]DunningCampaignResponse, len(campaigns))
	for i, campaign := range campaigns {
		responses[i] = DunningCampaignResponse{
			ID:                    campaign.ID,
			WorkspaceID:           campaign.WorkspaceID,
			ConfigurationID:       campaign.ConfigurationID,
			SubscriptionID:        campaign.SubscriptionID,
			PaymentID:             campaign.PaymentID,
			CustomerID:            campaign.CustomerID,
			Status:                campaign.Status,
			StartedAt:             campaign.StartedAt,
			CompletedAt:           campaign.CompletedAt,
			CurrentAttempt:        campaign.CurrentAttempt,
			NextRetryAt:           campaign.NextRetryAt,
			LastRetryAt:           campaign.LastRetryAt,
			Recovered:             campaign.Recovered,
			RecoveredAt:           campaign.RecoveredAt,
			RecoveredAmountCents:  campaign.RecoveredAmountCents,
			FinalActionTaken:      campaign.FinalActionTaken,
			FinalActionAt:         campaign.FinalActionAt,
			OriginalFailureReason: campaign.OriginalFailureReason,
			OriginalAmountCents:   campaign.OriginalAmountCents,
			Currency:              campaign.Currency,
			Metadata:              campaign.Metadata,
			CreatedAt:             campaign.CreatedAt,
			UpdatedAt:             campaign.UpdatedAt,
			CustomerEmail:         campaign.CustomerEmail.String,
			CustomerName:          campaign.CustomerName.String,
			SubscriptionProductID: uuid.UUID(campaign.SubscriptionProductID.Bytes),
		}
	}

	c.JSON(http.StatusOK, responses)
}

// GetCampaign gets a single dunning campaign
// @Summary Get dunning campaign
// @Description Get a dunning campaign by ID
// @Tags dunning
// @Produce json
// @Param id path string true "Campaign ID"
// @Success 200 {object} DunningCampaignDetailResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/dunning/campaigns/{id} [get]
func (h *DunningHandler) GetCampaign(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid campaign ID"})
		return
	}

	campaign, err := h.common.db.GetDunningCampaign(c.Request.Context(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Campaign not found"})
			return
		}
		h.common.logger.Error("failed to get dunning campaign", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get campaign"})
		return
	}

	// Get attempts
	attempts, err := h.common.db.ListDunningAttempts(c.Request.Context(), id)
	if err != nil {
		h.common.logger.Error("failed to get dunning attempts", zap.Error(err))
		attempts = []db.DunningAttempt{}
	}

	response := DunningCampaignDetailResponse{
		GetDunningCampaignRow: campaign,
		ConfigurationName:     campaign.ConfigurationName,
		MaxRetryAttempts:      campaign.MaxRetryAttempts,
		RetryIntervalDays:     campaign.RetryIntervalDays,
		CustomerEmail:         campaign.CustomerEmail.String,
		CustomerName:          campaign.CustomerName.String,
		Attempts:              attempts,
	}

	c.JSON(http.StatusOK, response)
}

// PauseCampaign pauses an active dunning campaign
// @Summary Pause dunning campaign
// @Description Pause an active dunning campaign
// @Tags dunning
// @Produce json
// @Param id path string true "Campaign ID"
// @Success 200 {object} db.DunningCampaign
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/dunning/campaigns/{id}/pause [post]
func (h *DunningHandler) PauseCampaign(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid campaign ID"})
		return
	}

	campaign, err := h.common.db.PauseDunningCampaign(c.Request.Context(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Campaign not found"})
			return
		}
		h.common.logger.Error("failed to pause dunning campaign", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to pause campaign"})
		return
	}

	c.JSON(http.StatusOK, campaign)
}

// ResumeCampaign resumes a paused dunning campaign
// @Summary Resume dunning campaign
// @Description Resume a paused dunning campaign
// @Tags dunning
// @Produce json
// @Param id path string true "Campaign ID"
// @Success 200 {object} db.DunningCampaign
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/dunning/campaigns/{id}/resume [post]
func (h *DunningHandler) ResumeCampaign(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid campaign ID"})
		return
	}

	// Resume with next retry in 24 hours
	nextRetry := time.Now().Add(24 * time.Hour)
	campaign, err := h.common.db.ResumeDunningCampaign(c.Request.Context(), db.ResumeDunningCampaignParams{
		ID:          id,
		NextRetryAt: pgtype.Timestamptz{Time: nextRetry, Valid: true},
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Campaign not found"})
			return
		}
		h.common.logger.Error("failed to resume dunning campaign", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resume campaign"})
		return
	}

	c.JSON(http.StatusOK, campaign)
}

// Email template endpoints

// CreateEmailTemplate creates a new email template
// @Summary Create email template
// @Description Create a new dunning email template
// @Tags dunning
// @Accept json
// @Produce json
// @Param template body CreateEmailTemplateRequest true "Template details"
// @Success 201 {object} db.DunningEmailTemplate
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/dunning/email-templates [post]
func (h *DunningHandler) CreateEmailTemplate(c *gin.Context) {
	var req CreateEmailTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template, err := h.dunningService.CreateEmailTemplate(c.Request.Context(), services.EmailTemplateParams{
		WorkspaceID:        workspaceID,
		Name:               req.Name,
		TemplateType:       req.TemplateType,
		Subject:            req.Subject,
		BodyHTML:           req.BodyHTML,
		BodyText:           req.BodyText,
		AvailableVariables: req.AvailableVariables,
		IsActive:           req.IsActive,
	})
	if err != nil {
		h.common.logger.Error("failed to create email template", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create template"})
		return
	}

	c.JSON(http.StatusCreated, template)
}

// ListEmailTemplates lists all email templates
// @Summary List email templates
// @Description List all dunning email templates
// @Tags dunning
// @Produce json
// @Success 200 {array} db.DunningEmailTemplate
// @Failure 500 {object} map[string]string
// @Router /api/v1/dunning/email-templates [get]
func (h *DunningHandler) ListEmailTemplates(c *gin.Context) {
	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	templates, err := h.common.db.ListDunningEmailTemplates(c.Request.Context(), workspaceID)
	if err != nil {
		h.common.logger.Error("failed to list email templates", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list templates"})
		return
	}

	c.JSON(http.StatusOK, templates)
}

// Analytics endpoints

// GetCampaignStats gets dunning campaign statistics
// @Summary Get campaign statistics
// @Description Get dunning campaign statistics for a date range
// @Tags dunning
// @Produce json
// @Param start_date query string true "Start date (YYYY-MM-DD)"
// @Param end_date query string true "End date (YYYY-MM-DD)"
// @Success 200 {object} CampaignStatsResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/dunning/stats [get]
func (h *DunningHandler) GetCampaignStats(c *gin.Context) {
	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startStr := c.Query("start_date")
	endStr := c.Query("end_date")

	if startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_date and end_date are required"})
		return
	}

	startDate, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format"})
		return
	}

	endDate, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format"})
		return
	}

	stats, err := h.dunningService.GetCampaignStats(c.Request.Context(), workspaceID, startDate, endDate)
	if err != nil {
		h.common.logger.Error("failed to get campaign stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get statistics"})
		return
	}

	response := CampaignStatsResponse{
		ActiveCampaigns:      stats.ActiveCampaigns,
		RecoveredCampaigns:   stats.RecoveredCampaigns,
		LostCampaigns:        stats.LostCampaigns,
		AtRiskAmountCents:    stats.AtRiskAmountCents,
		RecoveredAmountCents: stats.RecoveredAmountCents,
		LostAmountCents:      stats.LostAmountCents,
		RecoveryRate:         calculateRecoveryRate(stats.RecoveredCampaigns, stats.LostCampaigns),
	}

	c.JSON(http.StatusOK, response)
}

// Request/Response types

type CreateDunningConfigurationRequest struct {
	Name                   string          `json:"name" binding:"required"`
	Description            *string         `json:"description"`
	IsActive               bool            `json:"is_active"`
	IsDefault              bool            `json:"is_default"`
	MaxRetryAttempts       int32           `json:"max_retry_attempts" binding:"required,min=1,max=10"`
	RetryIntervalDays      []int32         `json:"retry_interval_days" binding:"required"`
	AttemptActions         json.RawMessage `json:"attempt_actions" binding:"required"`
	FinalAction            string          `json:"final_action" binding:"required,oneof=cancel pause downgrade"`
	FinalActionConfig      json.RawMessage `json:"final_action_config"`
	SendPreDunningReminder bool            `json:"send_pre_dunning_reminder"`
	PreDunningDays         int32           `json:"pre_dunning_days"`
	AllowCustomerRetry     bool            `json:"allow_customer_retry"`
	GracePeriodHours       int32           `json:"grace_period_hours"`
}

type CreateEmailTemplateRequest struct {
	Name               string          `json:"name" binding:"required"`
	TemplateType       string          `json:"template_type" binding:"required"`
	Subject            string          `json:"subject" binding:"required"`
	BodyHTML           string          `json:"body_html" binding:"required"`
	BodyText           *string         `json:"body_text"`
	AvailableVariables json.RawMessage `json:"available_variables"`
	IsActive           bool            `json:"is_active"`
}

type DunningCampaignResponse struct {
	ID                    uuid.UUID          `json:"id"`
	WorkspaceID           uuid.UUID          `json:"workspace_id"`
	ConfigurationID       uuid.UUID          `json:"configuration_id"`
	SubscriptionID        pgtype.UUID        `json:"subscription_id"`
	PaymentID             pgtype.UUID        `json:"payment_id"`
	CustomerID            uuid.UUID          `json:"customer_id"`
	Status                string             `json:"status"`
	StartedAt             pgtype.Timestamptz `json:"started_at"`
	CompletedAt           pgtype.Timestamptz `json:"completed_at"`
	CurrentAttempt        int32              `json:"current_attempt"`
	NextRetryAt           pgtype.Timestamptz `json:"next_retry_at"`
	LastRetryAt           pgtype.Timestamptz `json:"last_retry_at"`
	Recovered             pgtype.Bool        `json:"recovered"`
	RecoveredAt           pgtype.Timestamptz `json:"recovered_at"`
	RecoveredAmountCents  pgtype.Int8        `json:"recovered_amount_cents"`
	FinalActionTaken      pgtype.Text        `json:"final_action_taken"`
	FinalActionAt         pgtype.Timestamptz `json:"final_action_at"`
	OriginalFailureReason pgtype.Text        `json:"original_failure_reason"`
	OriginalAmountCents   int64              `json:"original_amount_cents"`
	Currency              string             `json:"currency"`
	Metadata              []byte             `json:"metadata"`
	CreatedAt             pgtype.Timestamptz `json:"created_at"`
	UpdatedAt             pgtype.Timestamptz `json:"updated_at"`
	CustomerEmail         string             `json:"customer_email"`
	CustomerName          string             `json:"customer_name"`
	SubscriptionProductID uuid.UUID          `json:"subscription_product_id,omitempty"`
}

type DunningCampaignDetailResponse struct {
	db.GetDunningCampaignRow
	ConfigurationName string              `json:"configuration_name"`
	MaxRetryAttempts  int32               `json:"max_retry_attempts"`
	RetryIntervalDays []int32             `json:"retry_interval_days"`
	CustomerEmail     string              `json:"customer_email"`
	CustomerName      string              `json:"customer_name"`
	Attempts          []db.DunningAttempt `json:"attempts"`
}

type CampaignStatsResponse struct {
	ActiveCampaigns      int64   `json:"active_campaigns"`
	RecoveredCampaigns   int64   `json:"recovered_campaigns"`
	LostCampaigns        int64   `json:"lost_campaigns"`
	AtRiskAmountCents    int64   `json:"at_risk_amount_cents"`
	RecoveredAmountCents int64   `json:"recovered_amount_cents"`
	LostAmountCents      int64   `json:"lost_amount_cents"`
	RecoveryRate         float64 `json:"recovery_rate"`
}

// ProcessDueCampaigns manually triggers processing of due campaigns
// @Summary Process due campaigns
// @Description Manually trigger processing of campaigns that are due for retry
// @Tags dunning
// @Produce json
// @Param limit query int false "Maximum number of campaigns to process" default(10)
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/v1/dunning/process [post]
func (h *DunningHandler) ProcessDueCampaigns(c *gin.Context) {
	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// NOTE: Automatic dunning campaign processing is handled by the subscription processor Lambda.
	// This endpoint is for manual testing and debugging purposes only.

	h.common.logger.Info("manually processing due campaigns", zap.Int("limit", limit))

	// Process campaigns if retry engine is available
	if h.retryEngine != nil {
		go func() {
			ctx := context.Background()
			if err := h.retryEngine.ProcessDueCampaigns(ctx, int32(limit)); err != nil {
				h.common.logger.Error("failed to process due campaigns", zap.Error(err))
			}
		}()

		c.JSON(http.StatusOK, gin.H{
			"message": "Campaign processing initiated",
			"limit":   limit,
			"status":  "processing",
			"note":    "This endpoint is for testing only. In production, use scheduled jobs.",
		})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Retry engine not initialized",
			"note":  "Check that email service is properly configured",
		})
	}
}

func calculateRecoveryRate(recovered, lost int64) float64 {
	total := recovered + lost
	if total == 0 {
		return 0
	}
	return float64(recovered) / float64(total)
}
