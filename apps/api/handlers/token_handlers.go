package handlers

import (
	"net/http"
	"strings"

	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TokenHandler handles token-related operations
type TokenHandler struct {
	common       *CommonServices
	tokenService interfaces.TokenService
}

// NewTokenHandler creates a handler with interface dependencies
func NewTokenHandler(
	common *CommonServices,
	tokenService interfaces.TokenService,
) *TokenHandler {
	return &TokenHandler{
		common:       common,
		tokenService: tokenService,
	}
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
	Object string                  `json:"object"`
	Data   []helpers.TokenResponse `json:"data"`
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

	token, err := h.tokenService.GetToken(c.Request.Context(), parsedUUID)
	if err != nil {
		if err.Error() == "token not found" {
			sendError(c, http.StatusNotFound, "Token not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	sendSuccess(c, http.StatusOK, helpers.ToTokenResponse(*token))
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
	token, err := h.tokenService.GetTokenByAddress(c.Request.Context(), parsedNetworkID, address)
	if err != nil {
		if err.Error() == "token not found" {
			sendError(c, http.StatusNotFound, "Token not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	sendSuccess(c, http.StatusOK, helpers.ToTokenResponse(*token))
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
	tokens, err := h.tokenService.ListTokens(c.Request.Context())
	if err != nil {
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	response := make([]helpers.TokenResponse, len(tokens))
	for i, token := range tokens {
		response[i] = helpers.ToTokenResponse(token)
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

	tokens, err := h.tokenService.ListTokensByNetwork(c.Request.Context(), parsedNetworkID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	response := make([]helpers.TokenResponse, len(tokens))
	for i, token := range tokens {
		response[i] = helpers.ToTokenResponse(token)
	}

	sendList(c, response)
}

// GetTokenQuote godoc
// @Summary Get token quote
// @Description Retrieves the price of a given token symbol in the specified fiat currency using CoinMarketCap API.
// @Tags tokens
// @Accept json
// @Produce json
// @Param quote body GetTokenQuoteRequest true "Token and Fiat symbols"
// @Success 200 {object} GetTokenQuoteResponse
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

	// Get token quote using the service
	result, err := h.tokenService.GetTokenQuote(c.Request.Context(), services.TokenQuoteParams{
		TokenSymbol: req.TokenSymbol,
		FiatSymbol:  req.FiatSymbol,
	})
	if err != nil {
		if err.Error() == "price service is unavailable" {
			sendError(c, http.StatusServiceUnavailable, err.Error(), nil)
			return
		}
		// Check if it's a not found error
		if strings.Contains(err.Error(), "price data not found") {
			sendError(c, http.StatusNotFound, err.Error(), nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	// Prepare and send success response
	response := GetTokenQuoteResponse{
		FiatSymbol:        result.FiatSymbol,
		TokenSymbol:       result.TokenSymbol,
		TokenAmountInFiat: result.TokenAmountInFiat,
	}

	sendSuccess(c, http.StatusOK, response)
}
