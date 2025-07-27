package handlers

import (
	"net/http"
	"strconv"

	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// NetworkHandler handles network related operations
type NetworkHandler struct {
	common         *CommonServices
	networkService interfaces.NetworkService
}

// NewNetworkHandler creates a handler with interface dependencies
func NewNetworkHandler(
	common *CommonServices,
	networkService interfaces.NetworkService,
) *NetworkHandler {
	return &NetworkHandler{
		common:         common,
		networkService: networkService,
	}
}

// CreateNetworkRequest represents the request body for creating a network
type CreateNetworkRequest struct {
	Name              string                  `json:"name" binding:"required"`
	Type              string                  `json:"type" binding:"required"`
	NetworkType       string                  `json:"network_type" binding:"required"`
	CircleNetworkType string                  `json:"circle_network_type" binding:"required"`
	BlockExplorerURL  string                  `json:"block_explorer_url,omitempty"`
	ChainID           int32                   `json:"chain_id" binding:"required"`
	IsTestnet         bool                    `json:"is_testnet"`
	Active            bool                    `json:"active"`
	LogoURL           string                  `json:"logo_url,omitempty"`
	DisplayName       string                  `json:"display_name,omitempty"`
	ChainNamespace    string                  `json:"chain_namespace,omitempty"`
	GasConfig         *CreateGasConfigRequest `json:"gas_config,omitempty"`
}

// CreateGasConfigRequest represents gas configuration for creating a network
type CreateGasConfigRequest struct {
	BaseFeeMultiplier     float64                `json:"base_fee_multiplier,omitempty"`
	PriorityFeeMultiplier float64                `json:"priority_fee_multiplier,omitempty"`
	DeploymentGasLimit    string                 `json:"deployment_gas_limit,omitempty"`
	TokenTransferGasLimit string                 `json:"token_transfer_gas_limit,omitempty"`
	SupportsEIP1559       bool                   `json:"supports_eip1559"`
	GasOracleURL          string                 `json:"gas_oracle_url,omitempty"`
	GasRefreshIntervalMs  int32                  `json:"gas_refresh_interval_ms,omitempty"`
	GasPriorityLevels     map[string]interface{} `json:"gas_priority_levels,omitempty"`
	AverageBlockTimeMs    int32                  `json:"average_block_time_ms,omitempty"`
	PeakHoursMultiplier   float64                `json:"peak_hours_multiplier,omitempty"`
}

// UpdateNetworkRequest represents the request body for updating a network
type UpdateNetworkRequest struct {
	Name              string                  `json:"name,omitempty"`
	Type              string                  `json:"type,omitempty"`
	NetworkType       string                  `json:"network_type,omitempty"`
	CircleNetworkType string                  `json:"circle_network_type,omitempty"`
	BlockExplorerURL  string                  `json:"block_explorer_url,omitempty"`
	ChainID           int32                   `json:"chain_id,omitempty"`
	IsTestnet         *bool                   `json:"is_testnet,omitempty"`
	Active            *bool                   `json:"active,omitempty"`
	LogoURL           string                  `json:"logo_url,omitempty"`
	DisplayName       string                  `json:"display_name,omitempty"`
	ChainNamespace    string                  `json:"chain_namespace,omitempty"`
	GasConfig         *UpdateGasConfigRequest `json:"gas_config,omitempty"`
}

// UpdateGasConfigRequest represents gas configuration for updating a network
type UpdateGasConfigRequest struct {
	BaseFeeMultiplier     *float64               `json:"base_fee_multiplier,omitempty"`
	PriorityFeeMultiplier *float64               `json:"priority_fee_multiplier,omitempty"`
	DeploymentGasLimit    *string                `json:"deployment_gas_limit,omitempty"`
	TokenTransferGasLimit *string                `json:"token_transfer_gas_limit,omitempty"`
	SupportsEIP1559       *bool                  `json:"supports_eip1559,omitempty"`
	GasOracleURL          *string                `json:"gas_oracle_url,omitempty"`
	GasRefreshIntervalMs  *int32                 `json:"gas_refresh_interval_ms,omitempty"`
	GasPriorityLevels     map[string]interface{} `json:"gas_priority_levels,omitempty"`
	AverageBlockTimeMs    *int32                 `json:"average_block_time_ms,omitempty"`
	PeakHoursMultiplier   *float64               `json:"peak_hours_multiplier,omitempty"`
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

	network, err := h.networkService.GetNetwork(c.Request.Context(), parsedUUID)
	if err != nil {
		if err.Error() == "network not found" {
			sendError(c, http.StatusNotFound, "Network not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	sendSuccess(c, http.StatusOK, helpers.ToNetworkResponse(*network))
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
	chainID64, err := strconv.ParseInt(chainIDStr, 10, 32)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid chain ID format", err)
		return
	}
	chainID := int32(chainID64)

	network, err := h.networkService.GetNetworkByChainID(c.Request.Context(), chainID)
	if err != nil {
		if err.Error() == "network not found" {
			sendError(c, http.StatusNotFound, "Network not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	sendSuccess(c, http.StatusOK, helpers.ToNetworkResponse(*network))
}

// CreateNetwork godoc
// @Summary Create a new network
// @Description Creates a new network with the specified details
// @Tags networks
// @Accept json
// @Produce json
// @Tags exclude
func (h *NetworkHandler) CreateNetwork(c *gin.Context) {
	var req CreateNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	params := services.CreateNetworkParams{
		Name:              req.Name,
		Type:              req.Type,
		NetworkType:       req.NetworkType,
		CircleNetworkType: req.CircleNetworkType,
		BlockExplorerURL:  req.BlockExplorerURL,
		ChainID:           req.ChainID,
		IsTestnet:         req.IsTestnet,
		Active:            req.Active,
		LogoURL:           req.LogoURL,
		DisplayName:       req.DisplayName,
		ChainNamespace:    req.ChainNamespace,
	}

	if req.GasConfig != nil {
		params.GasConfig = &services.CreateGasConfigParams{
			BaseFeeMultiplier:     req.GasConfig.BaseFeeMultiplier,
			PriorityFeeMultiplier: req.GasConfig.PriorityFeeMultiplier,
			DeploymentGasLimit:    req.GasConfig.DeploymentGasLimit,
			TokenTransferGasLimit: req.GasConfig.TokenTransferGasLimit,
			SupportsEIP1559:       req.GasConfig.SupportsEIP1559,
			GasOracleURL:          req.GasConfig.GasOracleURL,
			GasRefreshIntervalMs:  req.GasConfig.GasRefreshIntervalMs,
			GasPriorityLevels:     req.GasConfig.GasPriorityLevels,
			AverageBlockTimeMs:    req.GasConfig.AverageBlockTimeMs,
			PeakHoursMultiplier:   req.GasConfig.PeakHoursMultiplier,
		}
	}

	network, err := h.networkService.CreateNetwork(c.Request.Context(), params)
	if err != nil {
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	sendSuccess(c, http.StatusCreated, helpers.ToNetworkResponse(*network))
}

// UpdateNetwork godoc
// @Summary Update a network
// @Description Updates an existing network with the specified details
// @Tags networks
// @Accept json
// @Produce json
// @Tags exclude
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

	params := services.UpdateNetworkParams{
		ID:                parsedUUID,
		Name:              req.Name,
		Type:              req.Type,
		NetworkType:       req.NetworkType,
		CircleNetworkType: req.CircleNetworkType,
		BlockExplorerURL:  req.BlockExplorerURL,
		ChainID:           req.ChainID,
		IsTestnet:         req.IsTestnet,
		Active:            req.Active,
		LogoURL:           req.LogoURL,
		DisplayName:       req.DisplayName,
		ChainNamespace:    req.ChainNamespace,
	}

	if req.GasConfig != nil {
		params.GasConfig = &services.UpdateGasConfigParams{
			BaseFeeMultiplier:     req.GasConfig.BaseFeeMultiplier,
			PriorityFeeMultiplier: req.GasConfig.PriorityFeeMultiplier,
			DeploymentGasLimit:    req.GasConfig.DeploymentGasLimit,
			TokenTransferGasLimit: req.GasConfig.TokenTransferGasLimit,
			SupportsEIP1559:       req.GasConfig.SupportsEIP1559,
			GasOracleURL:          req.GasConfig.GasOracleURL,
			GasRefreshIntervalMs:  req.GasConfig.GasRefreshIntervalMs,
			GasPriorityLevels:     req.GasConfig.GasPriorityLevels,
			AverageBlockTimeMs:    req.GasConfig.AverageBlockTimeMs,
			PeakHoursMultiplier:   req.GasConfig.PeakHoursMultiplier,
		}
	}

	network, err := h.networkService.UpdateNetwork(c.Request.Context(), params)
	if err != nil {
		if err.Error() == "network not found" {
			sendError(c, http.StatusNotFound, "Network not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	sendSuccess(c, http.StatusOK, helpers.ToNetworkResponse(*network))
}

// DeleteNetwork godoc
// @Summary Delete a network
// @Description Deletes a network with the specified ID
// @Tags networks
// @Accept json
// @Produce json
// @Tags exclude
func (h *NetworkHandler) DeleteNetwork(c *gin.Context) {
	networkId := c.Param("network_id")
	parsedUUID, err := uuid.Parse(networkId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid network ID format", err)
		return
	}

	err = h.networkService.DeleteNetwork(c.Request.Context(), parsedUUID)
	if err != nil {
		if err.Error() == "network not found" {
			sendError(c, http.StatusNotFound, "Network not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
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

	params := services.ListNetworksParams{}

	if testnetStr != "" {
		testnet := testnetStr == "true"
		params.IsTestnet = &testnet
	}
	if activeStr != "" {
		active := activeStr == "true"
		params.IsActive = &active
	}

	networks, err := h.networkService.ListNetworks(c.Request.Context(), params)
	if err != nil {
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	response := make([]helpers.NetworkWithTokensResponse, len(networks))
	for i, network := range networks {
		tokens, err := h.networkService.ListActiveTokensByNetwork(c.Request.Context(), network.ID)
		if err != nil {
			sendError(c, http.StatusInternalServerError, err.Error(), err)
			return
		}

		response[i] = helpers.ToNetworkWithTokensResponse(network, tokens)
	}

	sendList(c, response)
}
