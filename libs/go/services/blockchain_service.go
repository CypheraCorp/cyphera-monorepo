package services

import (
	"context"
	"fmt"
	"math/big"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// BlockchainService handles blockchain interactions
type BlockchainService struct {
	queries   db.Querier
	logger    *zap.Logger
	clients   map[uuid.UUID]*ethclient.Client // networkID -> client
	rpcURLs   map[uuid.UUID]string            // networkID -> RPC URL
	rpcAPIKey string
}

// TransactionData contains all relevant transaction information
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

// NewBlockchainService creates a new blockchain service
func NewBlockchainService(queries db.Querier, rpcAPIKey string) *BlockchainService {
	return &BlockchainService{
		queries:   queries,
		logger:    logger.Log,
		clients:   make(map[uuid.UUID]*ethclient.Client),
		rpcURLs:   make(map[uuid.UUID]string),
		rpcAPIKey: rpcAPIKey,
	}
}

// Initialize sets up RPC connections for all networks
func (s *BlockchainService) Initialize(ctx context.Context) error {
	// Validate API key
	if s.rpcAPIKey == "" {
		return fmt.Errorf("RPC API key not provided")
	}

	// Get all active networks from database
	networks, err := s.queries.ListActiveNetworks(ctx)
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	for _, network := range networks {
		// Check if network has RPC ID for Infura
		if network.RpcID == "" {
			s.logger.Warn("Network missing RPC ID, skipping",
				zap.String("network", network.Name),
			)
			continue
		}

		// Construct Infura RPC URL
		// Pattern: https://<rpc_id>.infura.io/v3/<api_key>
		rpcURL := fmt.Sprintf("https://%s.infura.io/v3/%s", network.RpcID, s.rpcAPIKey)

		client, err := ethclient.Dial(rpcURL)
		if err != nil {
			s.logger.Error("Failed to connect to network RPC",
				zap.String("network", network.Name),
				zap.String("rpc_id", network.RpcID),
				zap.Error(err),
			)
			continue
		}

		s.clients[network.ID] = client
		s.rpcURLs[network.ID] = rpcURL

		s.logger.Info("Connected to network RPC",
			zap.String("network", network.Name),
			zap.String("network_id", network.ID.String()),
			zap.String("rpc_id", network.RpcID),
		)
	}

	if len(s.clients) == 0 {
		return fmt.Errorf("no RPC connections established")
	}

	return nil
}

// GetTransactionData fetches complete transaction data from the blockchain
func (s *BlockchainService) GetTransactionData(ctx context.Context, txHash string, networkID uuid.UUID) (*TransactionData, error) {
	client, ok := s.clients[networkID]
	if !ok {
		return nil, fmt.Errorf("no RPC client for network %s", networkID)
	}

	// Parse transaction hash
	hash := common.HexToHash(txHash)

	// Get transaction
	tx, isPending, err := client.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	if isPending {
		return nil, fmt.Errorf("transaction is still pending")
	}

	// Get transaction receipt for gas usage and status
	receipt, err := client.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	// Get block for timestamp and base fee
	block, err := client.BlockByNumber(ctx, big.NewInt(int64(receipt.BlockNumber.Uint64())))
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	// Get sender address
	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	signer := types.LatestSignerForChainID(chainID)
	from, err := types.Sender(signer, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender: %w", err)
	}

	// Build transaction data
	txData := &TransactionData{
		Hash:           txHash,
		BlockNumber:    receipt.BlockNumber.Uint64(),
		BlockTimestamp: block.Time(),
		Status:         receipt.Status,
		From:           from.Hex(),
		Value:          tx.Value(),
		Input:          tx.Data(),
		GasUsed:        receipt.GasUsed,
		GasLimit:       tx.Gas(),
		NetworkID:      networkID,
	}

	// Handle different transaction types
	if tx.Type() == types.LegacyTxType {
		// Legacy transaction
		txData.GasPrice = tx.GasPrice()
		txData.EffectiveGasPrice = tx.GasPrice()
	} else if tx.Type() == types.DynamicFeeTxType {
		// EIP-1559 transaction
		txData.MaxFeePerGas = tx.GasFeeCap()
		txData.MaxPriorityFeePerGas = tx.GasTipCap()
		txData.BaseFeePerGas = block.BaseFee()
		txData.EffectiveGasPrice = receipt.EffectiveGasPrice
		txData.GasPrice = receipt.EffectiveGasPrice // For compatibility
	}

	// Set To address (might be nil for contract creation)
	if tx.To() != nil {
		txData.To = tx.To().Hex()
	}

	// Calculate total gas cost
	txData.TotalGasCostWei = new(big.Int).Mul(
		new(big.Int).SetUint64(txData.GasUsed),
		txData.EffectiveGasPrice,
	)

	return txData, nil
}

// GetTransactionDataFromEvent fetches transaction data for a subscription event
func (s *BlockchainService) GetTransactionDataFromEvent(ctx context.Context, event *db.SubscriptionEvent) (*TransactionData, error) {
	if !event.TransactionHash.Valid || event.TransactionHash.String == "" {
		return nil, fmt.Errorf("subscription event has no transaction hash")
	}

	// Get the network ID from the subscription's product token
	subscription, err := s.queries.GetSubscription(ctx, event.SubscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	productToken, err := s.queries.GetProductToken(ctx, subscription.ProductTokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product token: %w", err)
	}

	return s.GetTransactionData(ctx, event.TransactionHash.String, productToken.NetworkID)
}

// Close closes all RPC connections
func (s *BlockchainService) Close() {
	for networkID, client := range s.clients {
		client.Close()
		s.logger.Info("Closed RPC connection",
			zap.String("network_id", networkID.String()),
		)
	}
}

// TODO: Future blockchain service capabilities
// - GetBlockData(blockNumber) - fetch block information
// - GetTokenBalance(address, tokenAddress) - check ERC20 balances
// - GetCurrentGasPrice() - get current gas price estimates
// - SubscribeToEvents(contractAddress, eventSignature) - listen for events
// - GetContractState(contractAddress, slot) - read contract storage
// - EstimateGas(transaction) - estimate gas for a transaction
// - GetENSName(address) / ResolveENS(name) - ENS resolution
// - GetTransactionsByAddress(address) - transaction history
// - VerifyDelegation(delegationData) - verify delegation signatures
// - GetNonce(address) - get current nonce for an address
