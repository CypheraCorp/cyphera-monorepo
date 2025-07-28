package helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
)

// ToNetworkResponse converts database model to API response
func ToNetworkResponse(n db.Network) responses.NetworkResponse {
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
	gasConfig := &responses.GasConfigResponse{
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

	return responses.NetworkResponse{
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
func ToNetworkWithTokensResponse(network db.Network, tokens []db.Token) responses.NetworkWithTokensResponse {
	tokenResponses := make([]responses.TokenResponse, len(tokens))
	for i, token := range tokens {
		tokenResponses[i] = ToTokenResponse(token)
	}

	return responses.NetworkWithTokensResponse{
		Network: ToNetworkResponse(network),
		Tokens:  tokenResponses,
	}
}

// Circle blockchain identifiers constants
const (
	CircleEthSepolia      = "ETH-SEPOLIA"
	CircleEth             = "ETH"
	CircleArb             = "ARB"
	CircleArbSepolia      = "ARB-SEPOLIA"
	CircleMatic           = "MATIC"
	CircleMaticAmoy       = "MATIC-AMOY"
	CircleBase            = "BASE"
	CircleBaseSepolia     = "BASE-SEPOLIA"
	CircleUnichain        = "UNICHAIN"
	CircleUnichainSepolia = "UNICHAIN-SEPOLIA"
	CircleOp              = "OP"
	CircleOPSepolia       = "OP-SEPOLIA"
)

// GetNetworkType converts a Circle blockchain identifier to the corresponding NetworkType enum.
// It returns NetworkTypeEvm for Ethereum-based chains and NetworkTypeSolana for Solana chains.
func GetNetworkType(blockchain string) db.NetworkType {
	switch blockchain {
	case CircleEth, CircleEthSepolia, CircleArb, CircleArbSepolia, CircleMatic, CircleMaticAmoy, CircleBase, CircleBaseSepolia, CircleUnichain, CircleUnichainSepolia:
		return db.NetworkTypeEvm
	default:
		return db.NetworkTypeEvm // Default to EVM
	}
}

// GetCircleNetworkType converts a Circle blockchain identifier to the corresponding CircleNetworkType enum.
// It returns an error if the blockchain is not supported.
func GetCircleNetworkType(blockchain string) (db.CircleNetworkType, error) {
	switch blockchain {
	case CircleEth:
		return db.CircleNetworkTypeETH, nil
	case CircleEthSepolia:
		return db.CircleNetworkTypeETHSEPOLIA, nil
	case CircleArb:
		return db.CircleNetworkTypeARB, nil
	case CircleArbSepolia:
		return db.CircleNetworkTypeARBSEPOLIA, nil
	case CircleMatic:
		return db.CircleNetworkTypeMATIC, nil
	case CircleMaticAmoy:
		return db.CircleNetworkTypeMATICAMOY, nil
	case CircleBase:
		return db.CircleNetworkTypeBASE, nil
	case CircleBaseSepolia:
		return db.CircleNetworkTypeBASESEPOLIA, nil
	case CircleUnichain:
		return db.CircleNetworkTypeUNI, nil
	case CircleUnichainSepolia:
		return db.CircleNetworkTypeUNI, nil
	case CircleOp:
		return db.CircleNetworkTypeOP, nil
	case CircleOPSepolia:
		return db.CircleNetworkTypeOPSEPOLIA, nil
	default:
		return "", fmt.Errorf("unsupported blockchain: %s", blockchain)
	}
}
