package params

import (
	"math/big"

	"github.com/google/uuid"
)

// GasFeeCalculationParams contains parameters for gas fee calculation from actual transaction
type GasFeeCalculationParams struct {
	NetworkID         uuid.UUID
	TokenID           uuid.UUID
	TransactionType   string   // "eip1559" or "legacy"
	GasUsed           *big.Int // Actual gas used
	BaseFeeWei        *big.Int // Base fee for EIP-1559
	MaxPriorityFeeWei *big.Int // Priority fee for EIP-1559
	GasPriceWei       *big.Int // Gas price for legacy transactions
	Currency          string   // Target currency for conversion
}

// EstimateGasFeeParams contains parameters for estimating gas fees
type EstimateGasFeeParams struct {
	NetworkID         uuid.UUID
	TransactionType   string
	ContractAddress   *string
	MethodSignature   *string
	TokenTransfer     bool
	EstimatedGasLimit uint64
	Currency          string
}

// SponsorshipCheckParams contains parameters for checking gas sponsorship eligibility
type SponsorshipCheckParams struct {
	WorkspaceID     uuid.UUID
	CustomerID      uuid.UUID
	ProductID       uuid.UUID
	GasCostUSDCents int64
	TransactionType string // e.g., "subscription", "one_time", "refund"
	CustomerTier    string // e.g., "free", "pro", "enterprise"
}
