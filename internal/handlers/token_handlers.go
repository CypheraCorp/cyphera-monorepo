package handlers

import (
	"cyphera-api/internal/db"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	Active          bool   `json:"active"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

// CreateTokenRequest represents the request body for creating a token
type CreateTokenRequest struct {
	NetworkID       string `json:"network_id" binding:"required"`
	GasToken        bool   `json:"gas_token"`
	Name            string `json:"name" binding:"required"`
	Symbol          string `json:"symbol" binding:"required"`
	ContractAddress string `json:"contract_address" binding:"required"`
	Active          bool   `json:"active"`
}

// UpdateTokenRequest represents the request body for updating a token
type UpdateTokenRequest struct {
	Name            string `json:"name,omitempty"`
	Symbol          string `json:"symbol,omitempty"`
	ContractAddress string `json:"contract_address,omitempty"`
	GasToken        *bool  `json:"gas_token,omitempty"`
	Active          *bool  `json:"active,omitempty"`
}

// ListTokensResponse represents the paginated response for token list operations
type ListTokensResponse struct {
	Object string          `json:"object"`
	Data   []TokenResponse `json:"data"`
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

	token, err := h.common.db.UpdateToken(c.Request.Context(), db.UpdateTokenParams{
		ID:              parsedUUID,
		Name:            req.Name,
		Symbol:          req.Symbol,
		ContractAddress: req.ContractAddress,
		GasToken:        *req.GasToken,
		Active:          *req.Active,
	})
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

// Helper function to convert database model to API response
func toTokenResponse(t db.Token) TokenResponse {
	return TokenResponse{
		ID:              t.ID.String(),
		Object:          "token",
		NetworkID:       t.NetworkID.String(),
		GasToken:        t.GasToken,
		Name:            t.Name,
		Symbol:          t.Symbol,
		ContractAddress: t.ContractAddress,
		Active:          t.Active,
		CreatedAt:       t.CreatedAt.Time.Unix(),
		UpdatedAt:       t.UpdatedAt.Time.Unix(),
	}
}
