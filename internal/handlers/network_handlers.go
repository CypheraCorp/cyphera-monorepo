package handlers

import (
	"cyphera-api/internal/db"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// NetworkHandler handles network related operations
type NetworkHandler struct {
	common *CommonServices
}

// NewNetworkHandler creates a new instance of NetworkHandler
func NewNetworkHandler(common *CommonServices) *NetworkHandler {
	return &NetworkHandler{common: common}
}

// NetworkResponse represents the standardized API response for network operations
type NetworkResponse struct {
	ID                string `json:"id"`
	Object            string `json:"object"`
	Name              string `json:"name"`
	Type              string `json:"type"`
	ChainID           int32  `json:"chain_id"`
	NetworkType       string `json:"network_type"`
	CircleNetworkType string `json:"circle_network_type"`
	BlockExplorerURL  string `json:"block_explorer_url,omitempty"`
	IsTestnet         bool   `json:"is_testnet"`
	Active            bool   `json:"active"`
	CreatedAt         int64  `json:"created_at"`
	UpdatedAt         int64  `json:"updated_at"`
}

// CreateNetworkRequest represents the request body for creating a network
type CreateNetworkRequest struct {
	Name              string `json:"name" binding:"required"`
	Type              string `json:"type" binding:"required"`
	NetworkType       string `json:"network_type" binding:"required"`
	CircleNetworkType string `json:"circle_network_type" binding:"required"`
	BlockExplorerURL  string `json:"block_explorer_url,omitempty"`
	ChainID           int32  `json:"chain_id" binding:"required"`
	IsTestnet         bool   `json:"is_testnet"`
	Active            bool   `json:"active"`
}

// UpdateNetworkRequest represents the request body for updating a network
type UpdateNetworkRequest struct {
	Name              string `json:"name,omitempty"`
	Type              string `json:"type,omitempty"`
	NetworkType       string `json:"network_type,omitempty"`
	CircleNetworkType string `json:"circle_network_type,omitempty"`
	BlockExplorerURL  string `json:"block_explorer_url,omitempty"`
	ChainID           int32  `json:"chain_id,omitempty"`
	IsTestnet         *bool  `json:"is_testnet,omitempty"`
	Active            *bool  `json:"active,omitempty"`
}

// ListNetworksResponse represents the paginated response for network list operations
type ListNetworksResponse struct {
	Object string            `json:"object"`
	Data   []NetworkResponse `json:"data"`
}

// NetworkWithTokensResponse represents a network with its associated tokens
type NetworkWithTokensResponse struct {
	NetworkResponse
	Tokens []TokenResponse `json:"tokens"`
}

// ListNetworksWithTokensResponse represents the response for listing networks with tokens
type ListNetworksWithTokensResponse struct {
	Object string                      `json:"object"`
	Data   []NetworkWithTokensResponse `json:"data"`
}

// GetNetwork godoc
// @Summary Get network by ID
// @Description Get network details by network ID
// @Tags networks
// @Accept json
// @Produce json
// @Param network_id path string true "Network ID"
// @Success 200 {object} NetworkResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /networks/{network_id} [get]
func (h *NetworkHandler) GetNetwork(c *gin.Context) {
	networkId := c.Param("network_id")
	parsedUUID, err := uuid.Parse(networkId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid network ID format", err)
		return
	}

	network, err := h.common.db.GetNetwork(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Network not found")
		return
	}

	sendSuccess(c, http.StatusOK, toNetworkResponse(network))
}

// GetNetworkByChainID godoc
// @Summary Get network by chain ID
// @Description Get network details by chain ID
// @Tags networks
// @Accept json
// @Produce json
// @Param chain_id path string true "Chain ID"
// @Success 200 {object} NetworkResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /networks/chain/{chain_id} [get]
func (h *NetworkHandler) GetNetworkByChainID(c *gin.Context) {
	chainIDStr := c.Param("chain_id")
	chainID, err := safeParseInt32(chainIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid chain ID format", err)
		return
	}

	network, err := h.common.db.GetNetworkByChainID(c.Request.Context(), chainID)
	if err != nil {
		handleDBError(c, err, "Network not found")
		return
	}

	sendSuccess(c, http.StatusOK, toNetworkResponse(network))
}

// CreateNetwork godoc
// @Summary Create network
// @Description Creates a new network
// @Tags networks
// @Accept json
// @Produce json
// @Param network body CreateNetworkRequest true "Network creation data"
// @Success 201 {object} NetworkResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /networks [post]
func (h *NetworkHandler) CreateNetwork(c *gin.Context) {
	var req CreateNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	network, err := h.common.db.CreateNetwork(c.Request.Context(), db.CreateNetworkParams{
		Name:              req.Name,
		Type:              req.Type,
		NetworkType:       db.NetworkType(req.NetworkType),
		CircleNetworkType: db.CircleNetworkType(req.CircleNetworkType),
		BlockExplorerUrl:  nullableString(req.BlockExplorerURL),
		ChainID:           req.ChainID,
		IsTestnet:         req.IsTestnet,
		Active:            req.Active,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create network", err)
		return
	}

	sendSuccess(c, http.StatusCreated, toNetworkResponse(network))
}

// UpdateNetwork godoc
// @Summary Update network
// @Description Updates an existing network
// @Tags networks
// @Accept json
// @Produce json
// @Param network_id path string true "Network ID"
// @Param network body UpdateNetworkRequest true "Network update data"
// @Success 200 {object} NetworkResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /networks/{network_id} [put]
func (h *NetworkHandler) UpdateNetwork(c *gin.Context) {
	networkId := c.Param("network_id")
	parsedUUID, err := uuid.Parse(networkId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid network ID format", err)
		return
	}

	var req UpdateNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	network, err := h.common.db.UpdateNetwork(c.Request.Context(), db.UpdateNetworkParams{
		ID:                parsedUUID,
		Name:              req.Name,
		Type:              req.Type,
		NetworkType:       db.NetworkType(req.NetworkType),
		CircleNetworkType: db.CircleNetworkType(req.CircleNetworkType),
		BlockExplorerUrl:  nullableString(req.BlockExplorerURL),
		ChainID:           req.ChainID,
		IsTestnet:         *req.IsTestnet,
		Active:            *req.Active,
	})
	if err != nil {
		handleDBError(c, err, "Failed to update network")
		return
	}

	sendSuccess(c, http.StatusOK, toNetworkResponse(network))
}

// DeleteNetwork godoc
// @Summary Delete network
// @Description Deletes a network
// @Tags networks
// @Accept json
// @Produce json
// @Param network_id path string true "Network ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /networks/{network_id} [delete]
func (h *NetworkHandler) DeleteNetwork(c *gin.Context) {
	networkId := c.Param("network_id")
	parsedUUID, err := uuid.Parse(networkId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid network ID format", err)
		return
	}

	err = h.common.db.DeleteNetwork(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Failed to delete network")
		return
	}

	c.Status(http.StatusNoContent)
}

// GetNetworks godoc
// @Summary List networks with tokens
// @Description Retrieves all networks with their associated tokens
// @Tags networks
// @Accept json
// @Produce json
// @Param testnet query boolean false "Filter networks by testnet status"
// @Param active query boolean false "Filter networks by active status"
// @Success 200 {object} ListNetworksWithTokensResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /networks [get]
func (h *NetworkHandler) ListNetworks(c *gin.Context) {
	// Parse query parameters
	testnetStr := c.Query("testnet")
	activeStr := c.Query("active")

	params := db.ListNetworksParams{}

	if testnetStr != "" {
		params.IsTestnet.Valid = true
		params.IsTestnet.Bool = testnetStr == "true"
	}
	if activeStr != "" {
		params.IsActive.Valid = true
		params.IsActive.Bool = activeStr == "true"
	}

	networks, err := h.common.db.ListNetworks(c.Request.Context(), params)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve active networks", err)
		return
	}

	response := make([]NetworkWithTokensResponse, len(networks))
	for i, network := range networks {
		tokens, err := h.common.db.ListActiveTokensByNetwork(c.Request.Context(), network.ID)
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to retrieve tokens for network", err)
			return
		}

		tokenResponses := make([]TokenResponse, len(tokens))
		for j, token := range tokens {
			tokenResponses[j] = toTokenResponse(token)
		}

		response[i] = NetworkWithTokensResponse{
			NetworkResponse: toNetworkResponse(network),
			Tokens:          tokenResponses,
		}
	}

	sendList(c, response)
}

// Helper function to convert database model to API response
func toNetworkResponse(n db.Network) NetworkResponse {
	var blockExplorerURL string
	if n.BlockExplorerUrl.Valid {
		blockExplorerURL = n.BlockExplorerUrl.String
	}

	return NetworkResponse{
		ID:                n.ID.String(),
		Object:            "network",
		Name:              n.Name,
		Type:              n.Type,
		NetworkType:       string(n.NetworkType),
		CircleNetworkType: string(n.CircleNetworkType),
		BlockExplorerURL:  blockExplorerURL,
		ChainID:           n.ChainID,
		IsTestnet:         n.IsTestnet,
		Active:            n.Active,
		CreatedAt:         n.CreatedAt.Time.Unix(),
		UpdatedAt:         n.UpdatedAt.Time.Unix(),
	}
}

// Helper functions for nullable types (consider moving to a common place)
func nullableString(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}
