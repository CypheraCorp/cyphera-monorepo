package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// GasSponsorshipHandler manages gas sponsorship configuration endpoints
type GasSponsorshipHandler struct {
	common  *CommonServices
	service interfaces.GasSponsorshipService
	logger  *zap.Logger
}

// NewGasSponsorshipHandler creates a handler with interface dependency
func NewGasSponsorshipHandler(
	common *CommonServices,
	service interfaces.GasSponsorshipService,
	logger *zap.Logger,
) *GasSponsorshipHandler {
	if logger == nil {
		logger = zap.L()
	}
	return &GasSponsorshipHandler{
		common:  common,
		service: service,
		logger:  logger,
	}
}

// GasSponsorshipConfigRequest represents the request to create/update gas sponsorship config
type GasSponsorshipConfigRequest struct {
	SponsorshipEnabled       bool        `json:"sponsorship_enabled"`
	SponsorCustomerGas       bool        `json:"sponsor_customer_gas"`
	SponsorThresholdUsdCents *int64      `json:"sponsor_threshold_usd_cents,omitempty"`
	MonthlyBudgetUsdCents    *int64      `json:"monthly_budget_usd_cents,omitempty"`
	SponsorForProducts       []uuid.UUID `json:"sponsor_for_products,omitempty"`
	SponsorForCustomers      []uuid.UUID `json:"sponsor_for_customers,omitempty"`
	SponsorForTiers          []string    `json:"sponsor_for_tiers,omitempty"`
}

// GasSponsorshipConfigResponse represents a gas sponsorship configuration
type GasSponsorshipConfigResponse struct {
	WorkspaceID              uuid.UUID   `json:"workspace_id"`
	SponsorshipEnabled       bool        `json:"sponsorship_enabled"`
	SponsorCustomerGas       bool        `json:"sponsor_customer_gas"`
	SponsorThresholdUsdCents *int64      `json:"sponsor_threshold_usd_cents,omitempty"`
	MonthlyBudgetUsdCents    *int64      `json:"monthly_budget_usd_cents,omitempty"`
	SponsorForProducts       []uuid.UUID `json:"sponsor_for_products"`
	SponsorForCustomers      []uuid.UUID `json:"sponsor_for_customers"`
	SponsorForTiers          []string    `json:"sponsor_for_tiers"`
	CurrentMonthSpentCents   int64       `json:"current_month_spent_cents"`
	RemainingBudgetCents     *int64      `json:"remaining_budget_cents,omitempty"`
}

// GetGasSponsorshipConfig retrieves gas sponsorship configuration
// @Summary Get gas sponsorship configuration
// @Description Get the gas sponsorship configuration for a workspace
// @Tags Gas Sponsorship
// @Accept json
// @Produce json
// @Param workspace_id query string true "Workspace ID"
// @Success 200 {object} GasSponsorshipConfigResponse
// @Router /gas-sponsorship/config [get]
func (h *GasSponsorshipHandler) GetGasSponsorshipConfig(c *gin.Context) {
	workspaceIDStr := c.Query("workspace_id")
	if workspaceIDStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "workspace_id is required"})
		return
	}

	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid workspace_id"})
		return
	}

	// Get config from database
	config, err := h.common.db.GetGasSponsorshipConfig(c.Request.Context(), workspaceID)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Return default config if none exists
			c.JSON(http.StatusOK, GasSponsorshipConfigResponse{
				WorkspaceID:         workspaceID,
				SponsorshipEnabled:  false,
				SponsorCustomerGas:  false,
				SponsorForProducts:  []uuid.UUID{},
				SponsorForCustomers: []uuid.UUID{},
				SponsorForTiers:     []string{},
			})
			return
		}
		h.logger.Error("Failed to get gas sponsorship config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get configuration"})
		return
	}

	// Parse JSON arrays
	products := []uuid.UUID{}
	if len(config.SponsorForProducts) > 0 {
		_ = json.Unmarshal(config.SponsorForProducts, &products)
	}

	customers := []uuid.UUID{}
	if len(config.SponsorForCustomers) > 0 {
		_ = json.Unmarshal(config.SponsorForCustomers, &customers)
	}

	tiers := []string{}
	if len(config.SponsorForTiers) > 0 {
		_ = json.Unmarshal(config.SponsorForTiers, &tiers)
	}

	response := GasSponsorshipConfigResponse{
		WorkspaceID:            config.WorkspaceID,
		SponsorshipEnabled:     config.SponsorshipEnabled.Bool,
		SponsorCustomerGas:     config.SponsorCustomerGas.Bool,
		SponsorForProducts:     products,
		SponsorForCustomers:    customers,
		SponsorForTiers:        tiers,
		CurrentMonthSpentCents: config.CurrentMonthSpentCents.Int64,
	}

	if config.SponsorThresholdUsdCents.Valid {
		threshold := config.SponsorThresholdUsdCents.Int64
		response.SponsorThresholdUsdCents = &threshold
	}

	if config.MonthlyBudgetUsdCents.Valid {
		budget := config.MonthlyBudgetUsdCents.Int64
		response.MonthlyBudgetUsdCents = &budget
		remaining := budget - config.CurrentMonthSpentCents.Int64
		response.RemainingBudgetCents = &remaining
	}

	c.JSON(http.StatusOK, response)
}

// UpdateGasSponsorshipConfig updates gas sponsorship configuration
// @Summary Update gas sponsorship configuration
// @Description Update the gas sponsorship configuration for a workspace
// @Tags Gas Sponsorship
// @Accept json
// @Produce json
// @Param workspace_id query string true "Workspace ID"
// @Param config body GasSponsorshipConfigRequest true "Configuration"
// @Success 200 {object} GasSponsorshipConfigResponse
// @Router /gas-sponsorship/config [put]
func (h *GasSponsorshipHandler) UpdateGasSponsorshipConfig(c *gin.Context) {
	workspaceIDStr := c.Query("workspace_id")
	if workspaceIDStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "workspace_id is required"})
		return
	}

	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid workspace_id"})
		return
	}

	var req GasSponsorshipConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	// Prepare update parameters
	params := db.UpdateGasSponsorshipConfigParams{
		WorkspaceID: workspaceID,
	}

	// Set boolean fields
	params.SponsorshipEnabled = pgtype.Bool{Bool: req.SponsorshipEnabled, Valid: true}
	params.SponsorCustomerGas = pgtype.Bool{Bool: req.SponsorCustomerGas, Valid: true}

	// Set optional fields
	if req.SponsorThresholdUsdCents != nil {
		params.SponsorThresholdUsdCents = pgtype.Int8{Int64: *req.SponsorThresholdUsdCents, Valid: true}
	}

	if req.MonthlyBudgetUsdCents != nil {
		params.MonthlyBudgetUsdCents = pgtype.Int8{Int64: *req.MonthlyBudgetUsdCents, Valid: true}
	}

	// Convert arrays to JSON
	if req.SponsorForProducts != nil {
		productsJSON, _ := json.Marshal(req.SponsorForProducts)
		params.SponsorForProducts = productsJSON
	}

	if req.SponsorForCustomers != nil {
		customersJSON, _ := json.Marshal(req.SponsorForCustomers)
		params.SponsorForCustomers = customersJSON
	}

	if req.SponsorForTiers != nil {
		tiersJSON, _ := json.Marshal(req.SponsorForTiers)
		params.SponsorForTiers = tiersJSON
	}

	// Convert to service update type
	updates := services.SponsorshipConfigUpdates{
		SponsorshipEnabled:       &req.SponsorshipEnabled,
		SponsorCustomerGas:       &req.SponsorCustomerGas,
		MonthlyBudgetUSDCents:    req.MonthlyBudgetUsdCents,
		SponsorThresholdUSDCents: req.SponsorThresholdUsdCents,
		SponsorForProducts:       &req.SponsorForProducts,
		SponsorForCustomers:      &req.SponsorForCustomers,
		SponsorForTiers:          &req.SponsorForTiers,
	}

	// Update configuration
	err = h.service.UpdateSponsorshipConfig(c.Request.Context(), workspaceID, updates)
	if err != nil {
		h.logger.Error("Failed to update gas sponsorship config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update configuration"})
		return
	}

	// Return updated config
	h.GetGasSponsorshipConfig(c)
}

// GetGasSponsorshipBudgetStatus gets current budget status
// @Summary Get gas sponsorship budget status
// @Description Get the current budget status for gas sponsorship
// @Tags Gas Sponsorship
// @Accept json
// @Produce json
// @Param workspace_id query string true "Workspace ID"
// @Success 200 {object} services.BudgetStatus
// @Router /gas-sponsorship/budget-status [get]
func (h *GasSponsorshipHandler) GetGasSponsorshipBudgetStatus(c *gin.Context) {
	workspaceIDStr := c.Query("workspace_id")
	if workspaceIDStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "workspace_id is required"})
		return
	}

	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid workspace_id"})
		return
	}

	status, err := h.service.GetSponsorshipBudgetStatus(c.Request.Context(), workspaceID)
	if err != nil {
		h.logger.Error("Failed to get budget status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get budget status"})
		return
	}

	c.JSON(http.StatusOK, status)
}
