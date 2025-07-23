package handlers

import (
	"github.com/cyphera/cyphera-api/libs/go/db"
	"encoding/json"
	"fmt"
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
	LogoURL           string `json:"logo_url,omitempty"`
	DisplayName       string `json:"display_name,omitempty"`
	ChainNamespace    string `json:"chain_namespace,omitempty"`
	CreatedAt         int64  `json:"created_at"`
	UpdatedAt         int64  `json:"updated_at"`
	// Gas configuration
	GasConfig         *GasConfigResponse `json:"gas_config,omitempty"`
}

// GasConfigResponse represents gas configuration for a network
type GasConfigResponse struct {
	BaseFeeMultiplier       float64                 `json:"base_fee_multiplier"`
	PriorityFeeMultiplier   float64                 `json:"priority_fee_multiplier"`
	DeploymentGasLimit      string                  `json:"deployment_gas_limit"`
	TokenTransferGasLimit   string                  `json:"token_transfer_gas_limit"`
	SupportsEIP1559         bool                    `json:"supports_eip1559"`
	GasOracleURL            string                  `json:"gas_oracle_url,omitempty"`
	GasRefreshIntervalMs    int32                   `json:"gas_refresh_interval_ms"`
	GasPriorityLevels       map[string]interface{}  `json:"gas_priority_levels"`
	AverageBlockTimeMs      int32                   `json:"average_block_time_ms"`
	PeakHoursMultiplier     float64                 `json:"peak_hours_multiplier"`
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
	LogoURL           string `json:"logo_url,omitempty"`
	DisplayName       string `json:"display_name,omitempty"`
	ChainNamespace    string `json:"chain_namespace,omitempty"`
	// Gas configuration
	GasConfig         *CreateGasConfigRequest `json:"gas_config,omitempty"`
}

// CreateGasConfigRequest represents gas configuration for creating a network
type CreateGasConfigRequest struct {
	BaseFeeMultiplier       float64                 `json:"base_fee_multiplier,omitempty"`
	PriorityFeeMultiplier   float64                 `json:"priority_fee_multiplier,omitempty"`
	DeploymentGasLimit      string                  `json:"deployment_gas_limit,omitempty"`
	TokenTransferGasLimit   string                  `json:"token_transfer_gas_limit,omitempty"`
	SupportsEIP1559         bool                    `json:"supports_eip1559"`
	GasOracleURL            string                  `json:"gas_oracle_url,omitempty"`
	GasRefreshIntervalMs    int32                   `json:"gas_refresh_interval_ms,omitempty"`
	GasPriorityLevels       map[string]interface{}  `json:"gas_priority_levels,omitempty"`
	AverageBlockTimeMs      int32                   `json:"average_block_time_ms,omitempty"`
	PeakHoursMultiplier     float64                 `json:"peak_hours_multiplier,omitempty"`
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
	LogoURL           string `json:"logo_url,omitempty"`
	DisplayName       string `json:"display_name,omitempty"`
	ChainNamespace    string `json:"chain_namespace,omitempty"`
	// Gas configuration
	GasConfig         *UpdateGasConfigRequest `json:"gas_config,omitempty"`
}

// UpdateGasConfigRequest represents gas configuration for updating a network
type UpdateGasConfigRequest struct {
	BaseFeeMultiplier       *float64                `json:"base_fee_multiplier,omitempty"`
	PriorityFeeMultiplier   *float64                `json:"priority_fee_multiplier,omitempty"`
	DeploymentGasLimit      *string                 `json:"deployment_gas_limit,omitempty"`
	TokenTransferGasLimit   *string                 `json:"token_transfer_gas_limit,omitempty"`
	SupportsEIP1559         *bool                   `json:"supports_eip1559,omitempty"`
	GasOracleURL            *string                 `json:"gas_oracle_url,omitempty"`
	GasRefreshIntervalMs    *int32                  `json:"gas_refresh_interval_ms,omitempty"`
	GasPriorityLevels       map[string]interface{}  `json:"gas_priority_levels,omitempty"`
	AverageBlockTimeMs      *int32                  `json:"average_block_time_ms,omitempty"`
	PeakHoursMultiplier     *float64                `json:"peak_hours_multiplier,omitempty"`
}

// ListNetworksResponse represents the paginated response for network list operations
type ListNetworksResponse struct {
	Object string            `json:"object"`
	Data   []NetworkResponse `json:"data"`
}

// NetworkWithTokensResponse represents a network with its associated tokens
type NetworkWithTokensResponse struct {
	NetworkResponse NetworkResponse `json:"network"`
	Tokens          []TokenResponse `json:"tokens"`
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

	params := db.CreateNetworkParams{
		Name:              req.Name,
		Type:              req.Type,
		NetworkType:       db.NetworkType(req.NetworkType),
		CircleNetworkType: db.CircleNetworkType(req.CircleNetworkType),
		BlockExplorerUrl:  nullableString(req.BlockExplorerURL),
		ChainID:           req.ChainID,
		IsTestnet:         req.IsTestnet,
		Active:            req.Active,
		LogoUrl:           nullableString(req.LogoURL),
		DisplayName:       nullableString(req.DisplayName),
		ChainNamespace:    nullableString(req.ChainNamespace),
	}
	
	// Set gas config if provided
	if req.GasConfig != nil {
		params.BaseFeeMultiplier = nullableNumeric(req.GasConfig.BaseFeeMultiplier)
		params.PriorityFeeMultiplier = nullableNumeric(req.GasConfig.PriorityFeeMultiplier)
		params.DeploymentGasLimit = nullableString(req.GasConfig.DeploymentGasLimit)
		params.TokenTransferGasLimit = nullableString(req.GasConfig.TokenTransferGasLimit)
		params.SupportsEip1559 = pgtype.Bool{Bool: req.GasConfig.SupportsEIP1559, Valid: true}
		params.GasOracleUrl = nullableString(req.GasConfig.GasOracleURL)
		params.GasRefreshIntervalMs = pgtype.Int4{Int32: req.GasConfig.GasRefreshIntervalMs, Valid: true}
		params.AverageBlockTimeMs = pgtype.Int4{Int32: req.GasConfig.AverageBlockTimeMs, Valid: true}
		params.PeakHoursMultiplier = nullableNumeric(req.GasConfig.PeakHoursMultiplier)
		
		if req.GasConfig.GasPriorityLevels != nil {
			levelsJSON, err := json.Marshal(req.GasConfig.GasPriorityLevels)
			if err == nil {
				params.GasPriorityLevels = levelsJSON
			}
		}
	}
	
	network, err := h.common.db.CreateNetwork(c.Request.Context(), params)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create network", err)
		return
	}

	sendSuccess(c, http.StatusCreated, toNetworkResponse(network))
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

	params := db.UpdateNetworkParams{
		ID:                parsedUUID,
		Name:              req.Name,
		Type:              req.Type,
		NetworkType:       db.NetworkType(req.NetworkType),
		CircleNetworkType: db.CircleNetworkType(req.CircleNetworkType),
		BlockExplorerUrl:  nullableString(req.BlockExplorerURL),
		ChainID:           req.ChainID,
		LogoUrl:           nullableString(req.LogoURL),
		DisplayName:       nullableString(req.DisplayName),
		ChainNamespace:    nullableString(req.ChainNamespace),
	}
	
	if req.IsTestnet != nil {
		params.IsTestnet = *req.IsTestnet
	}
	if req.Active != nil {
		params.Active = *req.Active
	}
	
	// Set gas config if provided
	if req.GasConfig != nil {
		if req.GasConfig.BaseFeeMultiplier != nil {
			params.BaseFeeMultiplier = nullableNumeric(*req.GasConfig.BaseFeeMultiplier)
		}
		if req.GasConfig.PriorityFeeMultiplier != nil {
			params.PriorityFeeMultiplier = nullableNumeric(*req.GasConfig.PriorityFeeMultiplier)
		}
		if req.GasConfig.DeploymentGasLimit != nil {
			params.DeploymentGasLimit = nullableString(*req.GasConfig.DeploymentGasLimit)
		}
		if req.GasConfig.TokenTransferGasLimit != nil {
			params.TokenTransferGasLimit = nullableString(*req.GasConfig.TokenTransferGasLimit)
		}
		if req.GasConfig.SupportsEIP1559 != nil {
			params.SupportsEip1559 = pgtype.Bool{Bool: *req.GasConfig.SupportsEIP1559, Valid: true}
		}
		if req.GasConfig.GasOracleURL != nil {
			params.GasOracleUrl = nullableString(*req.GasConfig.GasOracleURL)
		}
		if req.GasConfig.GasRefreshIntervalMs != nil {
			params.GasRefreshIntervalMs = pgtype.Int4{Int32: *req.GasConfig.GasRefreshIntervalMs, Valid: true}
		}
		if req.GasConfig.AverageBlockTimeMs != nil {
			params.AverageBlockTimeMs = pgtype.Int4{Int32: *req.GasConfig.AverageBlockTimeMs, Valid: true}
		}
		if req.GasConfig.PeakHoursMultiplier != nil {
			params.PeakHoursMultiplier = nullableNumeric(*req.GasConfig.PeakHoursMultiplier)
		}
		
		if req.GasConfig.GasPriorityLevels != nil {
			levelsJSON, err := json.Marshal(req.GasConfig.GasPriorityLevels)
			if err == nil {
				params.GasPriorityLevels = levelsJSON
			}
		}
	}
	
	network, err := h.common.db.UpdateNetwork(c.Request.Context(), params)
	if err != nil {
		handleDBError(c, err, "Failed to update network")
		return
	}

	sendSuccess(c, http.StatusOK, toNetworkResponse(network))
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
	
	var logoURL string
	if n.LogoUrl.Valid {
		logoURL = n.LogoUrl.String
	}
	
	var displayName string
	if n.DisplayName.Valid {
		displayName = n.DisplayName.String
	}
	
	var chainNamespace string
	if n.ChainNamespace.Valid {
		chainNamespace = n.ChainNamespace.String
	}

	// Build gas config
	gasConfig := &GasConfigResponse{
		DeploymentGasLimit:      n.DeploymentGasLimit.String,
		TokenTransferGasLimit:   n.TokenTransferGasLimit.String,
		SupportsEIP1559:         n.SupportsEip1559.Bool,
		GasRefreshIntervalMs:    n.GasRefreshIntervalMs.Int32,
		AverageBlockTimeMs:      n.AverageBlockTimeMs.Int32,
	}
	
	// Convert numeric values
	if n.BaseFeeMultiplier.Valid {
		if f8, err := n.BaseFeeMultiplier.Float64Value(); err == nil {
			gasConfig.BaseFeeMultiplier = f8.Float64
		}
	}
	if n.PriorityFeeMultiplier.Valid {
		if f8, err := n.PriorityFeeMultiplier.Float64Value(); err == nil {
			gasConfig.PriorityFeeMultiplier = f8.Float64
		}
	}
	if n.PeakHoursMultiplier.Valid {
		if f8, err := n.PeakHoursMultiplier.Float64Value(); err == nil {
			gasConfig.PeakHoursMultiplier = f8.Float64
		}
	}
	
	if n.GasOracleUrl.Valid {
		gasConfig.GasOracleURL = n.GasOracleUrl.String
	}
	
	// Parse gas priority levels JSON
	if n.GasPriorityLevels != nil {
		var levels map[string]interface{}
		if err := json.Unmarshal(n.GasPriorityLevels, &levels); err == nil {
			gasConfig.GasPriorityLevels = levels
		}
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
		LogoURL:           logoURL,
		DisplayName:       displayName,
		ChainNamespace:    chainNamespace,
		CreatedAt:         n.CreatedAt.Time.Unix(),
		UpdatedAt:         n.UpdatedAt.Time.Unix(),
		GasConfig:         gasConfig,
	}
}

// Helper functions for nullable types (consider moving to a common place)
func nullableString(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

// Helper function to convert float64 to pgtype.Numeric
func nullableNumeric(f float64) pgtype.Numeric {
	n := pgtype.Numeric{}
	// Convert float to string and scan it
	strVal := fmt.Sprintf("%f", f)
	n.Scan(strVal)
	return n
}
