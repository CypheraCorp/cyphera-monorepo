package requests

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
