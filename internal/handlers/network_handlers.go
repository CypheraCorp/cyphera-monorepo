package handlers

import (
	"cyphera-api/internal/db"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// NetworkHandler handles network-related operations
type NetworkHandler struct {
	common *CommonServices
}

// NewNetworkHandler creates a new NetworkHandler instance
func NewNetworkHandler(common *CommonServices) *NetworkHandler {
	return &NetworkHandler{common: common}
}

// NetworkResponse represents the standardized API response for network operations
type NetworkResponse struct {
	ID        string          `json:"id"`
	Object    string          `json:"object"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	ChainID   int32           `json:"chain_id"`
	Active    bool            `json:"active"`
	CreatedAt int64           `json:"created_at"`
	UpdatedAt int64           `json:"updated_at"`
	Tokens    []TokenResponse `json:"tokens"`
}

// CreateNetworkRequest represents the request body for creating a network
type CreateNetworkRequest struct {
	Name    string `json:"name" binding:"required"`
	Type    string `json:"type" binding:"required"`
	ChainID int32  `json:"chain_id" binding:"required"`
	Active  bool   `json:"active"`
}

// UpdateNetworkRequest represents the request body for updating a network
type UpdateNetworkRequest struct {
	Name    string `json:"name,omitempty"`
	Type    string `json:"type,omitempty"`
	ChainID int32  `json:"chain_id,omitempty"`
	Active  *bool  `json:"active,omitempty"`
}

// GetNetwork godoc
// @Summary Get a network
// @Description Retrieves the details of an existing network
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid network ID format"})
		return
	}

	network, err := h.common.db.GetNetwork(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Network not found"})
		return
	}

	c.JSON(http.StatusOK, toNetworkResponse(network))
}

// GetNetworkByChainID godoc
// @Summary Get a network by chain ID
// @Description Retrieves the details of an existing network by its chain ID
// @Tags networks
// @Accept json
// @Produce json
// @Param chain_id path int true "Chain ID"
// @Success 200 {object} NetworkResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /networks/chain/{chain_id} [get]
func (h *NetworkHandler) GetNetworkByChainID(c *gin.Context) {
	chainIDStr := c.Param("chain_id")
	chainID, err := strconv.ParseInt(chainIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid chain ID format"})
		return
	}

	network, err := h.common.db.GetNetworkByChainID(c.Request.Context(), int32(chainID))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Network not found"})
		return
	}

	c.JSON(http.StatusOK, toNetworkResponse(network))
}

// ListNetworks godoc
// @Summary List all networks
// @Description Returns a list of all networks
// @Tags networks
// @Accept json
// @Produce json
// @Success 200 {array} NetworkResponse
// @Security ApiKeyAuth
// @Router /networks [get]
func (h *NetworkHandler) ListNetworks(c *gin.Context) {
	networks, err := h.common.db.ListNetworks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve networks"})
		return
	}

	response := make([]NetworkResponse, len(networks))
	for i, network := range networks {
		response[i] = toNetworkResponse(network)
	}

	c.JSON(http.StatusOK, response)
}

// ListActiveNetworks godoc
// @Summary List active networks
// @Description Returns a list of all active networks
// @Tags networks
// @Accept json
// @Produce json
// @Success 200 {array} NetworkResponse
// @Security ApiKeyAuth
// @Router /networks/active [get]
func (h *NetworkHandler) ListActiveNetworks(c *gin.Context) {
	networks, err := h.common.db.ListActiveNetworks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve active networks"})
		return
	}

	response := make([]NetworkResponse, len(networks))
	for i, network := range networks {
		response[i] = toNetworkResponse(network)
	}

	c.JSON(http.StatusOK, response)
}

// CreateNetwork godoc
// @Summary Create a network
// @Description Creates a new network
// @Tags networks
// @Accept json
// @Produce json
// @Param network body CreateNetworkRequest true "Network creation data"
// @Success 201 {object} NetworkResponse
// @Failure 400 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /networks [post]
func (h *NetworkHandler) CreateNetwork(c *gin.Context) {
	var req CreateNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	network, err := h.common.db.CreateNetwork(c.Request.Context(), db.CreateNetworkParams{
		Name:    req.Name,
		Type:    req.Type,
		ChainID: req.ChainID,
		Active:  req.Active,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create network"})
		return
	}

	c.JSON(http.StatusCreated, toNetworkResponse(network))
}

// UpdateNetwork godoc
// @Summary Update a network
// @Description Updates an existing network
// @Tags networks
// @Accept json
// @Produce json
// @Param network_id path string true "Network ID"
// @Param network body UpdateNetworkRequest true "Network update data"
// @Success 200 {object} NetworkResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /networks/{network_id} [put]
func (h *NetworkHandler) UpdateNetwork(c *gin.Context) {
	networkId := c.Param("network_id")
	parsedUUID, err := uuid.Parse(networkId)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid network ID format"})
		return
	}

	var req UpdateNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	network, err := h.common.db.UpdateNetwork(c.Request.Context(), db.UpdateNetworkParams{
		ID:      parsedUUID,
		Name:    req.Name,
		Type:    req.Type,
		ChainID: req.ChainID,
		Active:  *req.Active,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update network"})
		return
	}

	c.JSON(http.StatusOK, toNetworkResponse(network))
}

// DeleteNetwork godoc
// @Summary Delete a network
// @Description Soft deletes a network
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid network ID format"})
		return
	}

	err = h.common.db.DeleteNetwork(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Network not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

//	ListNetworksWithTokens godoc
//
// @Summary List all tokens on each network
// @Description Returns a list of all tokens for each network
// @Tags networks
// @Accept json
// @Produce json
// @Success 200 {array} NetworkResponse
// @Security ApiKeyAuth
// @Router /networks/tokens [get]
func (h *NetworkHandler) ListNetworksWithTokens(c *gin.Context) {
	networks, err := h.common.db.ListNetworks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve networks"})
		return
	}

	networkResponses := make([]NetworkResponse, len(networks))

	for i, network := range networks {
		networkResponses[i] = toNetworkResponse(network)
		tokens, err := h.common.db.ListTokensByNetwork(c.Request.Context(), network.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve tokens"})
			return
		}
		tokenResponses := make([]TokenResponse, len(tokens))
		for j, token := range tokens {
			tokenResponses[j] = toTokenResponse(token)
		}
		networkResponses[i].Tokens = tokenResponses
	}

	c.JSON(http.StatusOK, networkResponses)
}

// Helper function to convert database model to API response
func toNetworkResponse(n db.Network) NetworkResponse {
	return NetworkResponse{
		ID:        n.ID.String(),
		Object:    "network",
		Name:      n.Name,
		Type:      n.Type,
		ChainID:   n.ChainID,
		Active:    n.Active,
		CreatedAt: n.CreatedAt.Time.Unix(),
		UpdatedAt: n.UpdatedAt.Time.Unix(),
	}
}
