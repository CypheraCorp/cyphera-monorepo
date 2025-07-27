package services

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// GasFeeService handles gas fee calculation and management
type GasFeeService struct {
	queries           db.Querier
	exchangeRateService *ExchangeRateService
	logger            *zap.Logger
}

// NewGasFeeService creates a new gas fee service
func NewGasFeeService(queries db.Querier, exchangeRateService *ExchangeRateService) *GasFeeService {
	return &GasFeeService{
		queries:           queries,
		exchangeRateService: exchangeRateService,
		logger:            logger.Log,
	}
}

// GasFeeCalculationParams contains parameters for gas fee calculation
type GasFeeCalculationParams struct {
	TransactionHash   string
	NetworkID         uuid.UUID
	TokenID           uuid.UUID
	GasUsed          *big.Int
	GasPriceWei      *big.Int
	BaseFeeWei       *big.Int    // For EIP-1559 transactions
	MaxPriorityFeeWei *big.Int   // For EIP-1559 transactions
	TransactionType   string     // "legacy", "eip1559"
	Currency          string     // Target currency for conversion (e.g., "USD")
}

// GasFeeResult contains the calculated gas fee information
type GasFeeResult struct {
	GasUsed           *big.Int
	GasPriceWei       *big.Int
	TotalGasWei       *big.Int
	TotalGasETH       float64
	TotalGasUSD       float64
	TotalGasCents     int64
	NetworkName       string
	TokenSymbol       string
	TransactionType   string
	ExchangeRate      float64
	CalculatedAt      time.Time
}

// EstimateGasFeeParams contains parameters for gas fee estimation
type EstimateGasFeeParams struct {
	NetworkID         uuid.UUID
	TransactionType   string // "transfer", "contract_call", "delegation"
	ContractAddress   *string
	EstimatedGasLimit uint64
	Currency          string
}

// EstimateGasFeeResult contains estimated gas fee information
type EstimateGasFeeResult struct {
	EstimatedGasLimit uint64
	GasPriceWei       *big.Int
	EstimatedCostWei  *big.Int
	EstimatedCostETH  float64
	EstimatedCostUSD  float64
	EstimatedCostCents int64
	NetworkName       string
	Confidence        float64 // 0.0 to 1.0, reliability of the estimate
}

// CalculateActualGasFee calculates gas fees from completed blockchain transactions
func (s *GasFeeService) CalculateActualGasFee(ctx context.Context, params GasFeeCalculationParams) (*GasFeeResult, error) {
	s.logger.Info("Calculating actual gas fee",
		zap.String("tx_hash", params.TransactionHash),
		zap.String("network_id", params.NetworkID.String()))

	// Get network information
	network, err := s.queries.GetNetwork(ctx, params.NetworkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get network: %w", err)
	}

	// Get token information
	token, err := s.queries.GetToken(ctx, params.TokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Calculate total gas cost in Wei
	var totalGasWei *big.Int
	switch params.TransactionType {
	case "eip1559":
		// For EIP-1559: gasUsed * (baseFee + priorityFee)
		if params.BaseFeeWei != nil && params.MaxPriorityFeeWei != nil {
			effectiveGasPrice := new(big.Int).Add(params.BaseFeeWei, params.MaxPriorityFeeWei)
			totalGasWei = new(big.Int).Mul(params.GasUsed, effectiveGasPrice)
		} else {
			return nil, fmt.Errorf("EIP-1559 transaction missing base fee or priority fee")
		}
	case "legacy":
		// For legacy: gasUsed * gasPrice
		if params.GasPriceWei != nil {
			totalGasWei = new(big.Int).Mul(params.GasUsed, params.GasPriceWei)
		} else {
			return nil, fmt.Errorf("legacy transaction missing gas price")
		}
	default:
		return nil, fmt.Errorf("unsupported transaction type: %s", params.TransactionType)
	}

	// Convert Wei to ETH (1 ETH = 10^18 Wei)
	totalGasETH := weiToEth(totalGasWei)

	// Get exchange rate from ETH to target currency
	exchangeRateResult, err := s.exchangeRateService.GetExchangeRate(ctx, ExchangeRateParams{
		FromSymbol: "ETH",
		ToSymbol:   params.Currency,
		NetworkID:  &params.NetworkID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange rate: %w", err)
	}

	// Convert ETH to target currency
	totalGasUSD := totalGasETH * exchangeRateResult.Rate
	totalGasCents := int64(totalGasUSD * 100) // Convert to cents

	return &GasFeeResult{
		GasUsed:         params.GasUsed,
		GasPriceWei:     params.GasPriceWei,
		TotalGasWei:     totalGasWei,
		TotalGasETH:     totalGasETH,
		TotalGasUSD:     totalGasUSD,
		TotalGasCents:   totalGasCents,
		NetworkName:     network.Name,
		TokenSymbol:     token.Symbol,
		TransactionType: params.TransactionType,
		ExchangeRate:    exchangeRateResult.Rate,
		CalculatedAt:    time.Now(),
	}, nil
}

// EstimateGasFee provides gas fee estimates for upcoming transactions
func (s *GasFeeService) EstimateGasFee(ctx context.Context, params EstimateGasFeeParams) (*EstimateGasFeeResult, error) {
	s.logger.Info("Estimating gas fee",
		zap.String("network_id", params.NetworkID.String()),
		zap.String("tx_type", params.TransactionType))

	// Get network information
	network, err := s.queries.GetNetwork(ctx, params.NetworkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get network: %w", err)
	}

	// Get current gas price estimate
	gasPriceWei, confidence := s.estimateGasPrice(ctx, network)
	
	// Adjust gas limit based on transaction type
	adjustedGasLimit := s.adjustGasLimitByType(params.EstimatedGasLimit, params.TransactionType)
	
	// Calculate estimated cost
	estimatedCostWei := new(big.Int).Mul(big.NewInt(int64(adjustedGasLimit)), gasPriceWei)
	estimatedCostETH := weiToEth(estimatedCostWei)

	// Get exchange rate
	exchangeRateResult, err := s.exchangeRateService.GetExchangeRate(ctx, ExchangeRateParams{
		FromSymbol: "ETH",
		ToSymbol:   params.Currency,
		NetworkID:  &params.NetworkID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange rate: %w", err)
	}

	estimatedCostUSD := estimatedCostETH * exchangeRateResult.Rate
	estimatedCostCents := int64(estimatedCostUSD * 100)

	return &EstimateGasFeeResult{
		EstimatedGasLimit:  adjustedGasLimit,
		GasPriceWei:        gasPriceWei,
		EstimatedCostWei:   estimatedCostWei,
		EstimatedCostETH:   estimatedCostETH,
		EstimatedCostUSD:   estimatedCostUSD,
		EstimatedCostCents: estimatedCostCents,
		NetworkName:        network.Name,
		Confidence:         confidence,
	}, nil
}

// CreateGasFeePaymentRecord creates a gas fee payment record
func (s *GasFeeService) CreateGasFeePaymentRecord(ctx context.Context, paymentID uuid.UUID, gasFeeResult *GasFeeResult, isSponsored bool, sponsorID *uuid.UUID) error {
	// Convert Wei to Gwei for storage
	gasPriceGwei := new(big.Int).Div(gasFeeResult.GasPriceWei, big.NewInt(1e9))
	
	// For now, use a placeholder network ID until we have proper network lookup
	// TODO: Implement network lookup by name
	networkID := uuid.New() // Placeholder network ID
	
	params := db.CreateGasFeePaymentParams{
		PaymentID:      paymentID,
		GasFeeWei:      gasFeeResult.TotalGasWei.String(),
		GasPriceGwei:   gasPriceGwei.String(),
		GasUnitsUsed:   gasFeeResult.GasUsed.Int64(),
		MaxGasUnits:    gasFeeResult.GasUsed.Int64(), // Use same as used for now
		PaymentMethod:  "native", // ETH was used for gas
		SponsorType:    "none",
		NetworkID:      networkID,
		GasFeeUsdCents: pgtype.Int8{Int64: gasFeeResult.TotalGasCents, Valid: true},
		EthUsdPrice:    pgtype.Numeric{}, // Would need proper decimal conversion
		BlockTimestamp: pgtype.Timestamptz{Time: gasFeeResult.CalculatedAt, Valid: true},
	}

	if isSponsored && sponsorID != nil {
		params.SponsorType = "merchant"
		params.SponsorID = pgtype.UUID{Bytes: *sponsorID, Valid: true}
	}

	_, err := s.queries.CreateGasFeePayment(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to create gas fee payment record: %w", err)
	}

	s.logger.Info("Created gas fee payment record",
		zap.String("payment_id", paymentID.String()),
		zap.Int64("cost_cents", gasFeeResult.TotalGasCents),
		zap.Bool("sponsored", isSponsored))

	return nil
}

// GetGasFeePayments retrieves gas fee payments for a workspace
func (s *GasFeeService) GetGasFeePayments(ctx context.Context, workspaceID uuid.UUID, limit int32) ([]db.GetGasFeePaymentsByWorkspaceRow, error) {
	payments, err := s.queries.GetGasFeePaymentsByWorkspace(ctx, db.GetGasFeePaymentsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get gas fee payments: %w", err)
	}

	return payments, nil
}

// GetGasFeeMetrics calculates gas fee metrics for a workspace
func (s *GasFeeService) GetGasFeeMetrics(ctx context.Context, workspaceID uuid.UUID, startDate, endDate time.Time) (*GasFeeMetrics, error) {
	metrics, err := s.queries.GetGasFeeMetrics(ctx, db.GetGasFeeMetricsParams{
		WorkspaceID: workspaceID,
		CreatedAt:   pgtype.Timestamptz{Time: startDate, Valid: true},
		CreatedAt_2: pgtype.Timestamptz{Time: endDate, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get gas fee metrics: %w", err)
	}

	return &GasFeeMetrics{
		TotalTransactions:   metrics.TotalTransactions,
		TotalGasCostCents:   metrics.TotalGasFeesCents,
		SponsoredCostCents:  metrics.MerchantSponsoredCents + metrics.PlatformSponsoredCents,
		AverageGasCostCents: int64(metrics.AvgGasFeeCents),
		NetworkBreakdown:    make(map[string]NetworkGasStats), // Empty for now
	}, nil
}

// GasFeeMetrics represents aggregated gas fee metrics
type GasFeeMetrics struct {
	TotalTransactions   int64                      `json:"total_transactions"`
	TotalGasCostCents   int64                      `json:"total_gas_cost_cents"`
	SponsoredCostCents  int64                      `json:"sponsored_cost_cents"`
	AverageGasCostCents int64                      `json:"average_gas_cost_cents"`
	NetworkBreakdown    map[string]NetworkGasStats `json:"network_breakdown"`
}

// NetworkGasStats represents gas statistics for a specific network
type NetworkGasStats struct {
	Transactions int64 `json:"transactions"`
	CostCents    int64 `json:"cost_cents"`
	AvgCostCents int64 `json:"avg_cost_cents"`
}

// Helper functions

// weiToEth converts Wei to ETH
func weiToEth(wei *big.Int) float64 {
	// 1 ETH = 10^18 Wei
	ethFloat := new(big.Float).SetInt(wei)
	divisor := new(big.Float).SetFloat64(1e18)
	result := new(big.Float).Quo(ethFloat, divisor)
	ethValue, _ := result.Float64()
	return ethValue
}

// estimateGasPrice estimates current gas price for a network
func (s *GasFeeService) estimateGasPrice(ctx context.Context, network db.Network) (*big.Int, float64) {
	// This is a simplified implementation
	// In a real scenario, you'd call the blockchain RPC to get current gas prices
	
	basePrices := map[string]*big.Int{
		"ethereum": big.NewInt(20000000000), // 20 Gwei
		"polygon":  big.NewInt(30000000000), // 30 Gwei
		"arbitrum": big.NewInt(100000000),   // 0.1 Gwei
		"optimism": big.NewInt(1000000),     // 0.001 Gwei
	}
	
	networkName := network.Name
	if price, exists := basePrices[networkName]; exists {
		return price, 0.8 // 80% confidence for static estimates
	}
	
	// Default fallback
	return big.NewInt(20000000000), 0.5 // 50% confidence for unknown networks
}

// adjustGasLimitByType adjusts gas limit based on transaction type
func (s *GasFeeService) adjustGasLimitByType(baseLimit uint64, txType string) uint64 {
	multipliers := map[string]float64{
		"transfer":      1.0,
		"contract_call": 1.2,
		"delegation":    1.5,
		"complex":       2.0,
	}
	
	if multiplier, exists := multipliers[txType]; exists {
		return uint64(float64(baseLimit) * multiplier)
	}
	
	return baseLimit
}

// parseNetworkBreakdown parses JSON network breakdown from database
func parseNetworkBreakdown(data []byte) map[string]NetworkGasStats {
	// This would parse JSON data from the database
	// For now, return empty map as placeholder
	return make(map[string]NetworkGasStats)
}