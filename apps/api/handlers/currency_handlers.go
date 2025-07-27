package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/services"
)

type CurrencyHandler struct {
	common          *CommonServices
	currencyService interfaces.CurrencyService
}

// NewCurrencyHandler creates a handler with interface dependencies
func NewCurrencyHandler(
	common *CommonServices,
	currencyService interfaces.CurrencyService,
) *CurrencyHandler {
	return &CurrencyHandler{
		common:          common,
		currencyService: currencyService,
	}
}

// ListCurrenciesResponse represents the response for listing currencies
type ListCurrenciesResponse struct {
	Currencies []helpers.CurrencyResponse `json:"currencies"`
}

// FormatAmountRequest represents a request to format an amount
type FormatAmountRequest struct {
	AmountCents  int64  `json:"amount_cents" binding:"required"`
	CurrencyCode string `json:"currency_code" binding:"required,len=3"`
	UseSymbol    bool   `json:"use_symbol"`
}

// FormatAmountResponse represents the response for amount formatting
type FormatAmountResponse struct {
	Formatted string `json:"formatted"`
	Currency  string `json:"currency"`
}

// ListActiveCurrencies returns all active currencies
// @Summary List active currencies
// @Description Get a list of all active fiat currencies
// @Tags currencies
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} ListCurrenciesResponse
// @Router /currencies [get]
func (h *CurrencyHandler) ListActiveCurrencies(c *gin.Context) {
	currencies, err := h.currencyService.ListActiveCurrencies(c.Request.Context())
	if err != nil {
		logger.Log.Error("Failed to list active currencies", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch currencies"})
		return
	}

	response := ListCurrenciesResponse{
		Currencies: currencies,
	}

	c.JSON(http.StatusOK, response)
}

// GetCurrency returns a specific currency by code
// @Summary Get currency details
// @Description Get details of a specific currency by its code
// @Tags currencies
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param code path string true "Currency code (e.g., USD, EUR)"
// @Success 200 {object} helpers.CurrencyResponse
// @Router /currencies/{code} [get]
func (h *CurrencyHandler) GetCurrency(c *gin.Context) {
	code := c.Param("code")

	currency, err := h.currencyService.GetCurrency(c.Request.Context(), code)
	if err != nil {
		logger.Log.Error("Failed to get currency", zap.String("code", code), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Currency not found"})
		return
	}

	c.JSON(http.StatusOK, currency)
}

// GetWorkspaceCurrencySettings returns currency settings for the workspace
// @Summary Get workspace currency settings
// @Description Get the default and supported currencies for the current workspace
// @Tags currencies
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} services.WorkspaceCurrencySettings
// @Router /workspaces/current/currency-settings [get]
func (h *CurrencyHandler) GetWorkspaceCurrencySettings(c *gin.Context) {
	workspaceIDStr := c.MustGet("workspaceID").(string)
	workspaceID, _ := uuid.Parse(workspaceIDStr)

	settings, err := h.currencyService.GetWorkspaceCurrencySettings(c.Request.Context(), workspaceID)
	if err != nil {
		logger.Log.Error("Failed to get workspace currency settings", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch workspace settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateWorkspaceCurrencySettings updates currency settings for the workspace
// @Summary Update workspace currency settings
// @Description Update the default and supported currencies for the current workspace
// @Tags currencies
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body services.UpdateWorkspaceCurrencyRequest true "Currency settings"
// @Success 200 {object} services.WorkspaceCurrencySettings
// @Router /workspaces/current/currency-settings [put]
func (h *CurrencyHandler) UpdateWorkspaceCurrencySettings(c *gin.Context) {
	workspaceIDStr := c.MustGet("workspaceID").(string)
	workspaceID, _ := uuid.Parse(workspaceIDStr)

	var req services.UpdateWorkspaceCurrencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settings, err := h.currencyService.UpdateWorkspaceCurrencySettings(c.Request.Context(), workspaceID, &req)
	if err != nil {
		logger.Log.Error("Failed to update workspace currency settings", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// FormatAmount formats an amount in the specified currency
// @Summary Format currency amount
// @Description Format an amount in cents to a human-readable string with currency symbol
// @Tags currencies
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body FormatAmountRequest true "Amount to format"
// @Success 200 {object} FormatAmountResponse
// @Router /currencies/format [post]
func (h *CurrencyHandler) FormatAmount(c *gin.Context) {
	var req FormatAmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var formatted string
	var err error

	if req.UseSymbol {
		formatted, err = h.currencyService.FormatAmount(c.Request.Context(), req.AmountCents, req.CurrencyCode)
	} else {
		formatted, err = h.currencyService.FormatAmountWithCode(c.Request.Context(), req.AmountCents, req.CurrencyCode)
	}

	if err != nil {
		logger.Log.Error("Failed to format amount", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid currency code"})
		return
	}

	response := FormatAmountResponse{
		Formatted: formatted,
		Currency:  req.CurrencyCode,
	}

	c.JSON(http.StatusOK, response)
}

// ListWorkspaceSupportedCurrencies returns currencies supported by the workspace
// @Summary List workspace supported currencies
// @Description Get a list of currencies supported by the current workspace
// @Tags currencies
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} ListCurrenciesResponse
// @Router /workspaces/current/currencies [get]
func (h *CurrencyHandler) ListWorkspaceSupportedCurrencies(c *gin.Context) {
	workspaceIDStr := c.MustGet("workspaceID").(string)
	workspaceID, _ := uuid.Parse(workspaceIDStr)

	currencies, err := h.currencyService.ListWorkspaceSupportedCurrencies(c.Request.Context(), workspaceID)
	if err != nil {
		logger.Log.Error("Failed to list workspace currencies", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch currencies"})
		return
	}

	response := ListCurrenciesResponse{
		Currencies: currencies,
	}

	c.JSON(http.StatusOK, response)
}
