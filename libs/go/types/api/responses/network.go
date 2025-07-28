package responses

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
	Network NetworkResponse `json:"network"`
	Tokens  []TokenResponse `json:"tokens"`
}
