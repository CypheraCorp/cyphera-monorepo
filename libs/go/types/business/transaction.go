package business

import (
	"math/big"

	"github.com/google/uuid"
)

// TransactionData contains all relevant data from a blockchain transaction
type TransactionData struct {
	// Basic transaction info
	Hash           string
	BlockNumber    uint64
	BlockTimestamp uint64
	Status         uint64 // 1 = success, 0 = failed
	From           string
	To             string
	Value          *big.Int
	Input          []byte

	// Gas information
	GasPrice          *big.Int // Wei per gas unit
	GasUsed           uint64   // Actual gas units used
	GasLimit          uint64   // Max gas units allowed
	EffectiveGasPrice *big.Int // For EIP-1559 transactions

	// EIP-1559 fields (if applicable)
	MaxFeePerGas         *big.Int
	MaxPriorityFeePerGas *big.Int
	BaseFeePerGas        *big.Int // From block

	// Calculated values
	TotalGasCostWei *big.Int // gasUsed * effectiveGasPrice
	NetworkID       uuid.UUID
}
