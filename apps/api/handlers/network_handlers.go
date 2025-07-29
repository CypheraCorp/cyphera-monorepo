package handlers

import (
	"net/http"
	"strconv"

	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// NetworkHandler handles network-related operations
type NetworkHandler struct {
	common         *CommonServices
	networkService interfaces.NetworkService
}

// Use types from the centralized packages
type CreateNetworkRequest = requests.CreateNetworkRequest
type CreateGasConfigRequest = requests.CreateGasConfigRequest
type UpdateNetworkRequest = requests.UpdateNetworkRequest
type UpdateGasConfigRequest = requests.UpdateGasConfigRequest

type NetworkResponse = responses.NetworkResponse
type GasConfigResponse = responses.GasConfigResponse
type NetworkWithTokensResponse = responses.NetworkWithTokensResponse

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

// GetNetwork godoc
// @Summary Get a network by ID
// @Description Retrieves a network by its ID
// @Tags networks
// @Accept json
// @Produce json
// @Param network_id path string true "Network ID"
// @Success 200 {object} NetworkResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
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

	networkCreateParams := params.CreateNetworkParams{
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
		networkCreateParams.GasConfig = &params.CreateGasConfigParams{
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

	network, err := h.networkService.CreateNetwork(c.Request.Context(), networkCreateParams)
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

	networkUpdateParams := params.UpdateNetworkParams{
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
		networkUpdateParams.GasConfig = &params.UpdateGasConfigParams{
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

	network, err := h.networkService.UpdateNetwork(c.Request.Context(), networkUpdateParams)
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

	listNetworkParams := params.ListNetworksParams{}

	if testnetStr != "" {
		testnet := testnetStr == "true"
		listNetworkParams.IsTestnet = &testnet
	}
	if activeStr != "" {
		active := activeStr == "true"
		listNetworkParams.IsActive = &active
	}

	networks, err := h.networkService.ListNetworks(c.Request.Context(), listNetworkParams)
	if err != nil {
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	response := make([]responses.NetworkWithTokensResponse, len(networks))
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

// GetNetworkByChainID godoc
// @Summary Get a network by chain ID
// @Description Retrieves a network by its chain ID (supports both decimal and hex formats)
// @Tags networks
// @Accept json
// @Produce json
// @Param chain_id path string true "Chain ID (decimal or hex with 0x prefix)"
// @Success 200 {object} NetworkResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /networks/chain/{chain_id} [get]
func (h *NetworkHandler) GetNetworkByChainID(c *gin.Context) {
	chainIdStr := c.Param("chain_id")

	var chainId int64
	var err error

	// Check if it's a hex string (starts with 0x or 0X)
	if len(chainIdStr) > 2 && (chainIdStr[:2] == "0x" || chainIdStr[:2] == "0X") {
		// Parse as hex
		chainId, err = strconv.ParseInt(chainIdStr[2:], 16, 32)
	} else {
		// Parse as decimal
		chainId, err = strconv.ParseInt(chainIdStr, 10, 32)
	}

	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid chain ID format", err)
		return
	}

	network, err := h.networkService.GetNetworkByChainID(c.Request.Context(), int32(chainId))
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
