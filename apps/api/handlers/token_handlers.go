package handlers

import (
	"net/http"
	"strings"

	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TokenHandler handles token-related operations
type TokenHandler struct {
	common       *CommonServices
	tokenService interfaces.TokenService
}

// Use types from the centralized packages
type CreateTokenRequest = requests.CreateTokenRequest
type UpdateTokenRequest = requests.UpdateTokenRequest
type GetTokenQuoteRequest = requests.GetTokenQuoteRequest

type TokenResponse = responses.TokenResponse
type ListTokensResponse = responses.ListTokensResponse
type GetTokenQuoteResponse = responses.GetTokenQuoteResponse

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

	response := make([]responses.TokenResponse, len(tokens))
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

	response := make([]responses.TokenResponse, len(tokens))
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

	// Parse UUIDs
	tokenID, err := uuid.Parse(req.TokenID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid token ID format", err)
		return
	}

	networkID, err := uuid.Parse(req.NetworkID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid network ID format", err)
		return
	}

	// Get token quote using the service
	result, err := h.tokenService.GetTokenQuote(c.Request.Context(), params.TokenQuoteParams{
		TokenID:    tokenID,
		NetworkID:  networkID,
		AmountWei:  req.AmountWei,
		ToCurrency: req.ToCurrency,
	})
	if err != nil {
		if err.Error() == "price service is unavailable" {
			sendError(c, http.StatusServiceUnavailable, err.Error(), nil)
			return
		}
		// Check if it's a not found error
		if strings.Contains(err.Error(), "price data not found") || strings.Contains(err.Error(), "token not found") {
			sendError(c, http.StatusNotFound, err.Error(), nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	// Convert TokenQuoteResult to GetTokenQuoteResponse
	response := GetTokenQuoteResponse{
		FiatSymbol:        req.ToCurrency,
		TokenSymbol:       "", // We'll need to add this
		TokenAmountInFiat: result.ExchangeRate, // Price per token in fiat
	}
	
	// Get the token to get its symbol
	token, err := h.tokenService.GetToken(c.Request.Context(), tokenID)
	if err == nil {
		response.TokenSymbol = token.Symbol
	}
	
	sendSuccess(c, http.StatusOK, response)
}
