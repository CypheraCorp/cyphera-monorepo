package handlers

import (
	"cyphera-api/internal/client/coinmarketcap"
	"cyphera-api/internal/db"
	"cyphera-api/internal/logger"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TokenHandler handles token-related operations
type TokenHandler struct {
	common *CommonServices
}

// NewTokenHandler creates a new TokenHandler instance
func NewTokenHandler(common *CommonServices) *TokenHandler {
	return &TokenHandler{common: common}
}

// TokenResponse represents the standardized API response for token operations
type TokenResponse struct {
	ID              string `json:"id"`
	Object          string `json:"object"`
	NetworkID       string `json:"network_id"`
	GasToken        bool   `json:"gas_token"`
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	ContractAddress string `json:"contract_address"`
	Decimals        int32  `json:"decimals"`
	Active          bool   `json:"active"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
	DeletedAt       *int64 `json:"deleted_at,omitempty"`
}

// CreateTokenRequest represents the request body for creating a token
type CreateTokenRequest struct {
	NetworkID       string `json:"network_id" binding:"required"`
	GasToken        bool   `json:"gas_token"`
	Name            string `json:"name" binding:"required"`
	Symbol          string `json:"symbol" binding:"required"`
	ContractAddress string `json:"contract_address" binding:"required"`
	Decimals        int32  `json:"decimals" binding:"required,gte=0"`
	Active          bool   `json:"active"`
}

// UpdateTokenRequest represents the request body for updating a token
type UpdateTokenRequest struct {
	Name            string `json:"name,omitempty"`
	Symbol          string `json:"symbol,omitempty"`
	ContractAddress string `json:"contract_address,omitempty"`
	Decimals        *int32 `json:"decimals,omitempty,gte=0"`
	GasToken        *bool  `json:"gas_token,omitempty"`
	Active          *bool  `json:"active,omitempty"`
}

// ListTokensResponse represents the paginated response for token list operations
type ListTokensResponse struct {
	Object string          `json:"object"`
	Data   []TokenResponse `json:"data"`
}

// GetTokenQuoteRequest mirrors TokenAmountPayload
type GetTokenQuoteRequest struct {
	FiatSymbol  string `json:"fiat_symbol" binding:"required"`
	TokenSymbol string `json:"token_symbol" binding:"required"`
}

// GetTokenQuoteResponse mirrors TokenAmountResponse
type GetTokenQuoteResponse struct {
	FiatSymbol        string  `json:"fiat_symbol"`
	TokenSymbol       string  `json:"token_symbol"`
	TokenAmountInFiat float64 `json:"token_amount_in_fiat"`
}

// GetToken godoc
// @Summary Get token by ID
// @Description Get token details by token ID
// @Tags tokens
// @Accept json
// @Produce json
// @Param token_id path string true "Token ID"
// @Success 200 {object} TokenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /tokens/{token_id} [get]
func (h *TokenHandler) GetToken(c *gin.Context) {
	tokenId := c.Param("token_id")
	parsedUUID, err := uuid.Parse(tokenId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid token ID format", err)
		return
	}

	token, err := h.common.db.GetToken(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Token not found")
		return
	}

	sendSuccess(c, http.StatusOK, toTokenResponse(token))
}

// GetTokenByAddress godoc
// @Summary Get token by address
// @Description Get token details by network ID and contract address
// @Tags tokens
// @Accept json
// @Produce json
// @Param network_id path string true "Network ID"
// @Param address path string true "Contract Address"
// @Success 200 {object} TokenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /tokens/network/{network_id}/address/{address} [get]
func (h *TokenHandler) GetTokenByAddress(c *gin.Context) {
	networkID := c.Param("network_id")
	parsedNetworkID, err := uuid.Parse(networkID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid network ID format", err)
		return
	}

	address := c.Param("address")
	token, err := h.common.db.GetTokenByAddress(c.Request.Context(), db.GetTokenByAddressParams{
		NetworkID:       parsedNetworkID,
		ContractAddress: address,
	})
	if err != nil {
		handleDBError(c, err, "Token not found")
		return
	}

	sendSuccess(c, http.StatusOK, toTokenResponse(token))
}

// ListTokens godoc
// @Summary List tokens
// @Description Retrieves all tokens
// @Tags tokens
// @Accept json
// @Produce json
// @Success 200 {object} ListTokensResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /tokens [get]
func (h *TokenHandler) ListTokens(c *gin.Context) {
	tokens, err := h.common.db.ListTokens(c.Request.Context())
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve tokens", err)
		return
	}

	response := make([]TokenResponse, len(tokens))
	for i, token := range tokens {
		response[i] = toTokenResponse(token)
	}

	sendList(c, response)
}

// ListTokensByNetwork godoc
// @Summary List tokens by network
// @Description Retrieves all tokens for a specific network
// @Tags tokens
// @Accept json
// @Produce json
// @Param network_id path string true "Network ID"
// @Success 200 {object} ListTokensResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /tokens/network/{network_id} [get]
func (h *TokenHandler) ListTokensByNetwork(c *gin.Context) {
	networkID := c.Param("network_id")
	parsedNetworkID, err := uuid.Parse(networkID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid network ID format", err)
		return
	}

	tokens, err := h.common.db.ListTokensByNetwork(c.Request.Context(), parsedNetworkID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve tokens", err)
		return
	}

	response := make([]TokenResponse, len(tokens))
	for i, token := range tokens {
		response[i] = toTokenResponse(token)
	}

	sendList(c, response)
}

// GetTokenQuote godoc
// @Summary Get token quote
// @Description Retrieves the price of a given token symbol in the specified fiat currency using CoinMarketCap API.
// @Tags tokens
// @Accept json
// @Produce json
// @Param quote body GetConversionRateRequest true "Token and Fiat symbols"
// @Success 200 {object} GetConversionRateResponse
// @Failure 400 {object} ErrorResponse "Invalid request body or missing parameters"
// @Failure 500 {object} ErrorResponse "Internal server error or failed to fetch price from CoinMarketCap"
// @Security ApiKeyAuth
// @Router /tokens/quote [post]
func (h *TokenHandler) GetTokenQuote(c *gin.Context) {
	var req GetTokenQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Use the injected CMC client from CommonServices
	if h.common.CMCClient == nil {
		logger.Log.Error("CoinMarketCap client is not initialized in CommonServices")
		sendError(c, http.StatusInternalServerError, "Price service is unavailable", nil)
		return
	}

	tokenSymbols := []string{req.TokenSymbol}
	fiatSymbols := []string{req.FiatSymbol}

	cmcResponse, err := h.common.CMCClient.GetLatestQuotes(tokenSymbols, fiatSymbols)
	if err != nil {
		logger.Log.Error("Failed to get quotes from CoinMarketCap", zap.Error(err))

		// Handle specific CMC client errors if defined (like *coinmarketcap.Error)
		var cmcErr *coinmarketcap.Error
		if errors.As(err, &cmcErr) {
			// Use CMC status code if available, otherwise default
			statusCode := http.StatusInternalServerError
			if cmcErr.StatusCode >= 400 && cmcErr.StatusCode < 500 {
				statusCode = cmcErr.StatusCode // e.g., 400 Bad Request, 401 Unauthorized, 404 Not Found
			}
			sendError(c, statusCode, fmt.Sprintf("Failed to get price: %s", cmcErr.Message), err)
		} else {
			// Generic error
			sendError(c, http.StatusInternalServerError, "Failed to fetch price data", err)
		}
		return
	}

	// Extract the price from the structured response
	upperTokenSymbol := strings.ToUpper(req.TokenSymbol)
	upperFiatSymbol := strings.ToUpper(req.FiatSymbol)

	var amount float64
	found := false

	if tokenDataList, ok := cmcResponse.Data[upperTokenSymbol]; ok && len(tokenDataList) > 0 {
		tokenData := tokenDataList[0]
		if quoteData, ok := tokenData.Quote[upperFiatSymbol]; ok {
			amount = quoteData.Price
			found = true
		}
	}

	if !found {
		logger.Log.Warn("Price not found in CoinMarketCap response",
			zap.String("token", upperTokenSymbol),
			zap.String("fiat", upperFiatSymbol),
			zap.Any("cmc_response_data", cmcResponse.Data), // Log the data part for debugging
		)
		sendError(c, http.StatusNotFound, fmt.Sprintf("Price data not found for %s in %s", upperTokenSymbol, upperFiatSymbol), nil)
		return
	}

	// Prepare and send success response
	response := GetTokenQuoteResponse{
		FiatSymbol:        upperFiatSymbol,
		TokenSymbol:       upperTokenSymbol,
		TokenAmountInFiat: amount,
	}

	sendSuccess(c, http.StatusOK, response)
}

// Helper function to convert database model to API response
func toTokenResponse(t db.Token) TokenResponse {
	var deletedAt *int64
	if t.DeletedAt.Valid {
		unixTime := t.DeletedAt.Time.Unix()
		deletedAt = &unixTime
	}

	return TokenResponse{
		ID:              t.ID.String(),
		Object:          "token",
		NetworkID:       t.NetworkID.String(),
		GasToken:        t.GasToken,
		Name:            t.Name,
		Symbol:          t.Symbol,
		ContractAddress: t.ContractAddress,
		Decimals:        t.Decimals,
		Active:          t.Active,
		CreatedAt:       t.CreatedAt.Time.Unix(),
		UpdatedAt:       t.UpdatedAt.Time.Unix(),
		DeletedAt:       deletedAt,
	}
}
