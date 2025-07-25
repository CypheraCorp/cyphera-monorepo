package helpers

import (
	"encoding/json"

	"github.com/cyphera/cyphera-api/libs/go/db"
)

// NetworkResponse represents the standardized API response for network operations
type NetworkResponse struct {
	ID                string             `json:"id"`
	Object            string             `json:"object"`
	Name              string             `json:"name"`
	Type              string             `json:"type"`
	ChainID           int32              `json:"chain_id"`
	NetworkType       string             `json:"network_type"`
	CircleNetworkType string             `json:"circle_network_type"`
	BlockExplorerURL  string             `json:"block_explorer_url,omitempty"`
	IsTestnet         bool               `json:"is_testnet"`
	Active            bool               `json:"active"`
	LogoURL           string             `json:"logo_url,omitempty"`
	DisplayName       string             `json:"display_name,omitempty"`
	ChainNamespace    string             `json:"chain_namespace,omitempty"`
	CreatedAt         int64              `json:"created_at"`
	UpdatedAt         int64              `json:"updated_at"`
	GasConfig         *GasConfigResponse `json:"gas_config,omitempty"`
}

// GasConfigResponse represents gas configuration for a network
type GasConfigResponse struct {
	BaseFeeMultiplier     float64                `json:"base_fee_multiplier"`
	PriorityFeeMultiplier float64                `json:"priority_fee_multiplier"`
	DeploymentGasLimit    string                 `json:"deployment_gas_limit"`
	TokenTransferGasLimit string                 `json:"token_transfer_gas_limit"`
	SupportsEIP1559       bool                   `json:"supports_eip1559"`
	GasOracleURL          string                 `json:"gas_oracle_url,omitempty"`
	GasRefreshIntervalMs  int32                  `json:"gas_refresh_interval_ms"`
	GasPriorityLevels     map[string]interface{} `json:"gas_priority_levels"`
	AverageBlockTimeMs    int32                  `json:"average_block_time_ms"`
	PeakHoursMultiplier   float64                `json:"peak_hours_multiplier"`
}

// NetworkWithTokensResponse represents a network with its associated tokens
type NetworkWithTokensResponse struct {
	NetworkResponse NetworkResponse `json:"network"`
	Tokens          []TokenResponse `json:"tokens"`
}

// ToNetworkResponse converts database model to API response
func ToNetworkResponse(n db.Network) NetworkResponse {
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
		DeploymentGasLimit:    n.DeploymentGasLimit.String,
		TokenTransferGasLimit: n.TokenTransferGasLimit.String,
		SupportsEIP1559:       n.SupportsEip1559.Bool,
		GasRefreshIntervalMs:  n.GasRefreshIntervalMs.Int32,
		AverageBlockTimeMs:    n.AverageBlockTimeMs.Int32,
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

// ToNetworkWithTokensResponse converts network and tokens to response
func ToNetworkWithTokensResponse(network db.Network, tokens []db.Token) NetworkWithTokensResponse {
	tokenResponses := make([]TokenResponse, len(tokens))
	for i, token := range tokens {
		tokenResponses[i] = ToTokenResponse(token)
	}

	return NetworkWithTokensResponse{
		NetworkResponse: ToNetworkResponse(network),
		Tokens:          tokenResponses,
	}
}
