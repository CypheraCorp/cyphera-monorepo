package params

import "github.com/google/uuid"

// CreateNetworkParams contains parameters for creating a network
type CreateNetworkParams struct {
	Name              string
	Type              string
	NetworkType       string
	CircleNetworkType string
	BlockExplorerURL  string
	ChainID           int32
	IsTestnet         bool
	Active            bool
	LogoURL           string
	DisplayName       string
	ChainNamespace    string
	GasConfig         *CreateGasConfigParams
}

// CreateGasConfigParams contains gas configuration parameters
type CreateGasConfigParams struct {
	BaseFeeMultiplier     float64
	PriorityFeeMultiplier float64
	DeploymentGasLimit    string
	TokenTransferGasLimit string
	SupportsEIP1559       bool
	GasOracleURL          string
	GasRefreshIntervalMs  int32
	GasPriorityLevels     map[string]interface{}
	AverageBlockTimeMs    int32
	PeakHoursMultiplier   float64
}

// ListNetworksParams contains parameters for listing networks
type ListNetworksParams struct {
	IsActive  *bool
	IsMainnet *bool
	IsTestnet *bool
	Limit     int32
	Offset    int32
	Search    *string
}

// UpdateNetworkParams contains parameters for updating a network
type UpdateNetworkParams struct {
	ID                uuid.UUID
	Name              string
	Type              string
	NetworkType       string
	CircleNetworkType string
	BlockExplorerURL  string
	ChainID           int32
	IsTestnet         *bool
	Active            *bool
	LogoURL           string
	DisplayName       string
	ChainNamespace    string
	GasConfig         *UpdateGasConfigParams
}

// UpdateGasConfigParams contains gas configuration parameters for updates
type UpdateGasConfigParams struct {
	BaseFeeMultiplier     *float64
	PriorityFeeMultiplier *float64
	DeploymentGasLimit    *string
	TokenTransferGasLimit *string
	SupportsEIP1559       *bool
	GasOracleURL          *string
	GasRefreshIntervalMs  *int32
	GasPriorityLevels     map[string]interface{}
	AverageBlockTimeMs    *int32
	PeakHoursMultiplier   *float64
}
