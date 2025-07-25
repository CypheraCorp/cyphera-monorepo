package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
)

type CurrencyHandler struct {
	common *CommonServices
}

// NewCurrencyHandler creates a new currency handler
func NewCurrencyHandler(common *CommonServices) *CurrencyHandler {
	return &CurrencyHandler{
		common: common,
	}
}

// CurrencyResponse represents a currency in API responses
type CurrencyResponse struct {
	Code              string   `json:"code"`
	Name              string   `json:"name"`
	Symbol            string   `json:"symbol"`
	DecimalPlaces     int32    `json:"decimal_places"`
	IsActive          bool     `json:"is_active"`
	SymbolPosition    string   `json:"symbol_position"`
	ThousandSeparator string   `json:"thousand_separator"`
	DecimalSeparator  string   `json:"decimal_separator"`
	Countries         []string `json:"countries"`
}

// ListCurrenciesResponse represents the response for listing currencies
type ListCurrenciesResponse struct {
	Currencies []CurrencyResponse `json:"currencies"`
}

// WorkspaceCurrencySettingsResponse represents workspace currency settings
type WorkspaceCurrencySettingsResponse struct {
	DefaultCurrency     string   `json:"default_currency"`
	SupportedCurrencies []string `json:"supported_currencies"`
}

// UpdateWorkspaceCurrencyRequest represents a request to update workspace currency settings
type UpdateWorkspaceCurrencyRequest struct {
	DefaultCurrency     *string  `json:"default_currency,omitempty"`
	SupportedCurrencies []string `json:"supported_currencies,omitempty"`
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
	currencies, err := h.common.db.ListActiveFiatCurrencies(c.Request.Context())
	if err != nil {
		logger.Log.Error("Failed to list active currencies", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch currencies"})
		return
	}

	response := ListCurrenciesResponse{
		Currencies: make([]CurrencyResponse, len(currencies)),
	}

	for i, currency := range currencies {
		response.Currencies[i] = h.currencyToResponse(&currency)
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
// @Success 200 {object} CurrencyResponse
// @Router /currencies/{code} [get]
func (h *CurrencyHandler) GetCurrency(c *gin.Context) {
	code := c.Param("code")

	currency, err := h.common.db.GetFiatCurrency(c.Request.Context(), code)
	if err != nil {
		logger.Log.Error("Failed to get currency", zap.String("code", code), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Currency not found"})
		return
	}

	c.JSON(http.StatusOK, h.currencyToResponse(&currency))
}

// GetWorkspaceCurrencySettings returns currency settings for the workspace
// @Summary Get workspace currency settings
// @Description Get the default and supported currencies for the current workspace
// @Tags currencies
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} WorkspaceCurrencySettingsResponse
// @Router /workspaces/current/currency-settings [get]
func (h *CurrencyHandler) GetWorkspaceCurrencySettings(c *gin.Context) {
	workspaceIDStr := c.MustGet("workspaceID").(string)
	workspaceID, _ := uuid.Parse(workspaceIDStr)

	// Get workspace
	workspace, err := h.common.db.GetWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		logger.Log.Error("Failed to get workspace", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch workspace"})
		return
	}

	// Parse supported currencies
	var supportedCurrencies []string
	if workspace.SupportedCurrencies != nil {
		if err := json.Unmarshal(workspace.SupportedCurrencies, &supportedCurrencies); err != nil {
			logger.Log.Error("Failed to parse supported currencies", zap.Error(err))
			supportedCurrencies = []string{"USD"} // Default fallback
		}
	}

	response := WorkspaceCurrencySettingsResponse{
		DefaultCurrency:     workspace.DefaultCurrency.String,
		SupportedCurrencies: supportedCurrencies,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateWorkspaceCurrencySettings updates currency settings for the workspace
// @Summary Update workspace currency settings
// @Description Update the default and supported currencies for the current workspace
// @Tags currencies
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body UpdateWorkspaceCurrencyRequest true "Currency settings"
// @Success 200 {object} WorkspaceCurrencySettingsResponse
// @Router /workspaces/current/currency-settings [put]
func (h *CurrencyHandler) UpdateWorkspaceCurrencySettings(c *gin.Context) {
	workspaceIDStr := c.MustGet("workspaceID").(string)
	workspaceID, _ := uuid.Parse(workspaceIDStr)

	var req UpdateWorkspaceCurrencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update default currency if provided
	if req.DefaultCurrency != nil {
		// Validate currency exists and is active
		if _, err := h.common.db.GetFiatCurrency(c.Request.Context(), *req.DefaultCurrency); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid default currency"})
			return
		}

		if err := h.common.db.UpdateWorkspaceDefaultCurrency(c.Request.Context(), db.UpdateWorkspaceDefaultCurrencyParams{
			ID:              workspaceID,
			DefaultCurrency: pgtype.Text{String: *req.DefaultCurrency, Valid: true},
		}); err != nil {
			logger.Log.Error("Failed to update default currency", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update default currency"})
			return
		}
	}

	// Update supported currencies if provided
	if req.SupportedCurrencies != nil {
		// Validate all currencies exist and are active
		for _, code := range req.SupportedCurrencies {
			if _, err := h.common.db.GetFiatCurrency(c.Request.Context(), code); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid currency: %s", code)})
				return
			}
		}

		// Convert to JSON
		supportedJSON, err := json.Marshal(req.SupportedCurrencies)
		if err != nil {
			logger.Log.Error("Failed to marshal supported currencies", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update supported currencies"})
			return
		}

		if err := h.common.db.UpdateWorkspaceSupportedCurrencies(c.Request.Context(), db.UpdateWorkspaceSupportedCurrenciesParams{
			ID:                  workspaceID,
			SupportedCurrencies: supportedJSON,
		}); err != nil {
			logger.Log.Error("Failed to update supported currencies", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update supported currencies"})
			return
		}
	}

	// Return updated settings
	h.GetWorkspaceCurrencySettings(c)
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

	// Get currency from database
	currency, err := h.common.db.GetFiatCurrency(c.Request.Context(), req.CurrencyCode)
	if err != nil {
		logger.Log.Error("Failed to get currency", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid currency code"})
		return
	}

	var formatted string
	if req.UseSymbol {
		formatted = FormatCurrencyAmount(req.AmountCents, &currency)
	} else {
		formatted = FormatCurrencyAmountWithCode(req.AmountCents, &currency)
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

	currencies, err := h.common.db.ListWorkspaceSupportedCurrencies(c.Request.Context(), workspaceID)
	if err != nil {
		logger.Log.Error("Failed to list workspace currencies", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch currencies"})
		return
	}

	response := ListCurrenciesResponse{
		Currencies: make([]CurrencyResponse, len(currencies)),
	}

	for i, currency := range currencies {
		response.Currencies[i] = h.currencyToResponse(&currency)
	}

	c.JSON(http.StatusOK, response)
}

// Helper function to convert db.FiatCurrency to CurrencyResponse
func (h *CurrencyHandler) currencyToResponse(currency *db.FiatCurrency) CurrencyResponse {
	response := CurrencyResponse{
		Code:          currency.Code,
		Name:          currency.Name,
		Symbol:        currency.Symbol,
		DecimalPlaces: currency.DecimalPlaces,
		IsActive:      currency.IsActive.Bool,
	}

	if currency.SymbolPosition.Valid {
		response.SymbolPosition = currency.SymbolPosition.String
	} else {
		response.SymbolPosition = "before"
	}

	if currency.ThousandSeparator.Valid {
		response.ThousandSeparator = currency.ThousandSeparator.String
	} else {
		response.ThousandSeparator = ","
	}

	if currency.DecimalSeparator.Valid {
		response.DecimalSeparator = currency.DecimalSeparator.String
	} else {
		response.DecimalSeparator = "."
	}

	// Parse countries JSON
	if currency.Countries != nil {
		if err := json.Unmarshal(currency.Countries, &response.Countries); err != nil {
			logger.Log.Error("Failed to parse countries", zap.Error(err))
			response.Countries = []string{}
		}
	}

	return response
}
