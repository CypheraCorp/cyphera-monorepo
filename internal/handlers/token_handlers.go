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

// ListActiveTokensByNetwork godoc
// @Summary List active tokens by network
// @Description Retrieves all active tokens for a specific network
// @Tags tokens
// @Accept json
// @Produce json
// @Param network_id path string true "Network ID"
// @Success 200 {object} ListTokensResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /tokens/network/{network_id}/active [get]
func (h *TokenHandler) ListActiveTokensByNetwork(c *gin.Context) {
	networkID := c.Param("network_id")
	parsedNetworkID, err := uuid.Parse(networkID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid network ID format", err)
		return
	}

	tokens, err := h.common.db.ListActiveTokensByNetwork(c.Request.Context(), parsedNetworkID)
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

// GetGasToken godoc
// @Summary Get gas token for network
// @Description Retrieves the gas token for a specific network
// @Tags tokens
// @Accept json
// @Produce json
// @Param network_id path string true "Network ID"
// @Success 200 {object} TokenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /tokens/network/{network_id}/gas [get]
func (h *TokenHandler) GetGasToken(c *gin.Context) {
	networkID := c.Param("network_id")
	parsedNetworkID, err := uuid.Parse(networkID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid network ID format", err)
		return
	}

	token, err := h.common.db.GetGasToken(c.Request.Context(), parsedNetworkID)
	if err != nil {
		handleDBError(c, err, "Gas token not found")
		return
	}

	sendSuccess(c, http.StatusOK, toTokenResponse(token))
}

// CreateToken godoc
// @Summary Create token
// @Description Creates a new token
// @Tags tokens
// @Accept json
// @Produce json
// @Param token body CreateTokenRequest true "Token creation data"
// @Success 201 {object} TokenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /tokens [post]
func (h *TokenHandler) CreateToken(c *gin.Context) {
	var req CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	parsedNetworkID, err := uuid.Parse(req.NetworkID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid network ID format", err)
		return
	}

	token, err := h.common.db.CreateToken(c.Request.Context(), db.CreateTokenParams{
		NetworkID:       parsedNetworkID,
		GasToken:        req.GasToken,
		Name:            req.Name,
		Symbol:          req.Symbol,
		ContractAddress: req.ContractAddress,
		Decimals:        req.Decimals,
		Active:          req.Active,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create token", err)
		return
	}

	sendSuccess(c, http.StatusCreated, toTokenResponse(token))
}

// UpdateToken godoc
// @Summary Update token
// @Description Updates an existing token
// @Tags tokens
// @Accept json
// @Produce json
// @Param token_id path string true "Token ID"
// @Param token body UpdateTokenRequest true "Token update data"
// @Success 200 {object} TokenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /tokens/{token_id} [put]
func (h *TokenHandler) UpdateToken(c *gin.Context) {
	tokenId := c.Param("token_id")
	parsedUUID, err := uuid.Parse(tokenId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid token ID format", err)
		return
	}

	var req UpdateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Fetch existing token to get current values for comparison
	existingToken, err := h.common.db.GetToken(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Token not found")
		return
	}

	// Prepare params for sqlc query, manually handling nullable fields
	params := db.UpdateTokenParams{
		ID:              parsedUUID,
		Name:            req.Name,            // Rely on COALESCE in SQL
		Symbol:          req.Symbol,          // Rely on COALESCE in SQL
		ContractAddress: req.ContractAddress, // Rely on COALESCE in SQL
		// Assign value from request if provided, otherwise keep existing value
		// SQL COALESCE will still apply if req field is empty string, but this handles Go types
		GasToken: existingToken.GasToken, // Start with existing value
		Decimals: existingToken.Decimals, // Start with existing value
		Active:   existingToken.Active,   // Start with existing value
	}

	// Overwrite with request values ONLY if they are not nil
	if req.GasToken != nil {
		params.GasToken = *req.GasToken
	}
	if req.Decimals != nil {
		params.Decimals = *req.Decimals
	}
	if req.Active != nil {
		params.Active = *req.Active
	}

	// Now call the update query
	token, err := h.common.db.UpdateToken(c.Request.Context(), params)
	if err != nil {
		handleDBError(c, err, "Failed to update token")
		return
	}

	sendSuccess(c, http.StatusOK, toTokenResponse(token))
}

// DeleteToken godoc
// @Summary Delete token
// @Description Deletes a token
// @Tags tokens
// @Accept json
// @Produce json
// @Param token_id path string true "Token ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /tokens/{token_id} [delete]
func (h *TokenHandler) DeleteToken(c *gin.Context) {
	tokenId := c.Param("token_id")
	parsedUUID, err := uuid.Parse(tokenId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid token ID format", err)
		return
	}

	err = h.common.db.DeleteToken(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Failed to delete token")
		return
	}

	sendSuccess(c, http.StatusNoContent, nil)
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
