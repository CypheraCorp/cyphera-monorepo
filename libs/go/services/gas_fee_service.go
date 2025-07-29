package services

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// GasFeeService handles gas fee calculation and management
type GasFeeService struct {
	queries             db.Querier
	exchangeRateService *ExchangeRateService
	logger              *zap.Logger
}

// NewGasFeeService creates a new gas fee service
func NewGasFeeService(queries db.Querier, exchangeRateService *ExchangeRateService) *GasFeeService {
	return &GasFeeService{
		queries:             queries,
		exchangeRateService: exchangeRateService,
		logger:              logger.Log,
	}
}

// CalculateActualGasFee calculates gas fees from completed blockchain transactions
func (s *GasFeeService) CalculateActualGasFee(ctx context.Context, gasFeeCalculationParams params.GasFeeCalculationParams) (*responses.GasFeeResult, error) {
	s.logger.Info("Calculating actual gas fee",
		zap.String("network_id", gasFeeCalculationParams.NetworkID.String()))

	// // Get network information
	// _, err := s.queries.GetNetwork(ctx, gasFeeCalculationParams.NetworkID)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get network: %w", err)
	// }

	// // Get token information
	// _, err = s.queries.GetToken(ctx, gasFeeCalculationParams.TokenID)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get token: %w", err)
	// }

	// Calculate total gas cost in Wei
	var totalGasWei *big.Int
	switch gasFeeCalculationParams.TransactionType {
	case "eip1559":
		// For EIP-1559: gasUsed * (baseFee + priorityFee)
		if gasFeeCalculationParams.BaseFeeWei != nil && gasFeeCalculationParams.MaxPriorityFeeWei != nil {
			effectiveGasPrice := new(big.Int).Add(gasFeeCalculationParams.BaseFeeWei, gasFeeCalculationParams.MaxPriorityFeeWei)
			totalGasWei = new(big.Int).Mul(gasFeeCalculationParams.GasUsed, effectiveGasPrice)
		} else {
			return nil, fmt.Errorf("EIP-1559 transaction missing base fee or priority fee")
		}
	case "legacy":
		// For legacy: gasUsed * gasPrice
		if gasFeeCalculationParams.GasPriceWei != nil {
			totalGasWei = new(big.Int).Mul(gasFeeCalculationParams.GasUsed, gasFeeCalculationParams.GasPriceWei)
		} else {
			return nil, fmt.Errorf("legacy transaction missing gas price")
		}
	default:
		return nil, fmt.Errorf("unsupported transaction type: %s", gasFeeCalculationParams.TransactionType)
	}

	// Convert Wei to ETH (1 ETH = 10^18 Wei)
	totalGasETH := weiToEth(totalGasWei)

	// Get exchange rate from ETH to target currency
	exchangeRateResult, err := s.exchangeRateService.GetExchangeRate(ctx, params.ExchangeRateParams{
		FromSymbol: "ETH",
		ToSymbol:   gasFeeCalculationParams.Currency,
		NetworkID:  &gasFeeCalculationParams.NetworkID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange rate: %w", err)
	}

	// Convert ETH to target currency
	totalGasUSD := totalGasETH * exchangeRateResult.Rate
	totalGasCents := int64(totalGasUSD * 100) // Convert to cents

	return &responses.GasFeeResult{
		EstimatedGasUnits:    uint64(gasFeeCalculationParams.GasUsed.Int64()),
		GasPriceWei:          gasFeeCalculationParams.GasPriceWei.String(),
		TotalGasCostWei:      totalGasWei.String(),
		TotalGasCostEth:      totalGasETH,
		TotalGasCostUSD:      totalGasUSD,
		TotalGasCostUSDCents: totalGasCents,
		Confidence:           0.95, // High confidence for actual transactions
	}, nil
}

// EstimateGasFee provides gas fee estimates for upcoming transactions
func (s *GasFeeService) EstimateGasFee(ctx context.Context, estimateGasFeeParams params.EstimateGasFeeParams) (*responses.EstimateGasFeeResult, error) {
	s.logger.Info("Estimating gas fee",
		zap.String("network_id", estimateGasFeeParams.NetworkID.String()),
		zap.String("tx_type", estimateGasFeeParams.TransactionType))

	// Get network information
	network, err := s.queries.GetNetwork(ctx, estimateGasFeeParams.NetworkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get network: %w", err)
	}

	// Get current gas price estimate
	gasPriceWei, confidence := s.estimateGasPrice(ctx, network)

	// Adjust gas limit based on transaction type
	adjustedGasLimit := s.adjustGasLimitByType(estimateGasFeeParams.EstimatedGasLimit, estimateGasFeeParams.TransactionType)

	// Calculate estimated cost
	estimatedCostWei := new(big.Int).Mul(big.NewInt(int64(adjustedGasLimit)), gasPriceWei)
	estimatedCostETH := weiToEth(estimatedCostWei)

	// Get exchange rate
	exchangeRateResult, err := s.exchangeRateService.GetExchangeRate(ctx, params.ExchangeRateParams{
		FromSymbol: "ETH",
		ToSymbol:   estimateGasFeeParams.Currency,
		NetworkID:  &estimateGasFeeParams.NetworkID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange rate: %w", err)
	}

	estimatedCostUSD := estimatedCostETH * exchangeRateResult.Rate
	estimatedCostCents := int64(estimatedCostUSD * 100)

	return &responses.EstimateGasFeeResult{
		NetworkName:           network.Name,
		TransactionType:       estimateGasFeeParams.TransactionType,
		EstimatedGasUnits:     adjustedGasLimit,
		CurrentGasPriceWei:    gasPriceWei.String(),
		EstimatedCostWei:      estimatedCostWei.String(),
		EstimatedCostEth:      estimatedCostETH,
		EstimatedCostUSD:      estimatedCostUSD,
		EstimatedCostUSDCents: estimatedCostCents,
		Confidence:            confidence,
	}, nil
}

// CreateGasFeePaymentRecord creates a gas fee payment record
func (s *GasFeeService) CreateGasFeePaymentRecord(ctx context.Context, paymentID uuid.UUID, gasFeeResult *responses.GasFeeResult, isSponsored bool, sponsorID *uuid.UUID) error {
	// Convert Wei to Gwei for storage
	gasPriceWei, _ := new(big.Int).SetString(gasFeeResult.GasPriceWei, 10)
	gasPriceGwei := new(big.Int).Div(gasPriceWei, big.NewInt(1e9))

	// For now, use a placeholder network ID until we have proper network lookup
	// TODO: Implement network lookup by name
	networkID := uuid.New() // Placeholder network ID

	params := db.CreateGasFeePaymentParams{
		PaymentID:      paymentID,
		GasFeeWei:      gasFeeResult.TotalGasCostWei,
		GasPriceGwei:   gasPriceGwei.String(),
		GasUnitsUsed:   int64(gasFeeResult.EstimatedGasUnits),
		MaxGasUnits:    int64(gasFeeResult.EstimatedGasUnits), // Use same as used for now
		PaymentMethod:  "native",                              // ETH was used for gas
		SponsorType:    "none",
		NetworkID:      networkID,
		GasFeeUsdCents: pgtype.Int8{Int64: gasFeeResult.TotalGasCostUSDCents, Valid: true},
		EthUsdPrice:    pgtype.Numeric{}, // Would need proper decimal conversion
		BlockTimestamp: pgtype.Timestamptz{Time: time.Now(), Valid: true},
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
		zap.Int64("cost_cents", gasFeeResult.TotalGasCostUSDCents),
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
func (s *GasFeeService) GetGasFeeMetrics(ctx context.Context, workspaceID uuid.UUID, startDate, endDate time.Time) (*business.GasFeeMetrics, error) {
	metrics, err := s.queries.GetGasFeeMetrics(ctx, db.GetGasFeeMetricsParams{
		WorkspaceID: workspaceID,
		CreatedAt:   pgtype.Timestamptz{Time: startDate, Valid: true},
		CreatedAt_2: pgtype.Timestamptz{Time: endDate, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get gas fee metrics: %w", err)
	}

	return &business.GasFeeMetrics{
		TotalTransactions:   metrics.TotalTransactions,
		TotalGasCostCents:   metrics.TotalGasFeesCents,
		SponsoredCostCents:  metrics.MerchantSponsoredCents + metrics.PlatformSponsoredCents,
		AverageGasCostCents: int64(metrics.AvgGasFeeCents),
		NetworkBreakdown:    make(map[string]business.NetworkGasStats), // Empty for now
	}, nil
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

// GetCurrentGasPrice returns the current gas price for a network
func (s *GasFeeService) GetCurrentGasPrice(ctx context.Context, networkID uuid.UUID) (int, error) {
	// Get network information
	network, err := s.queries.GetNetwork(ctx, networkID)
	if err != nil {
		return 0, fmt.Errorf("failed to get network: %w", err)
	}

	// Get current gas price estimate
	gasPriceWei, _ := s.estimateGasPrice(ctx, network)

	// Convert Wei to Gwei for return (divide by 1e9)
	gasPriceGwei := new(big.Int).Div(gasPriceWei, big.NewInt(1e9))

	// Convert to int (assuming it fits in int range)
	return int(gasPriceGwei.Int64()), nil
}

// parseNetworkBreakdown parses JSON network breakdown from database
func parseNetworkBreakdown(data []byte) map[string]business.NetworkGasStats {
	// This would parse JSON data from the database
	// For now, return empty map as placeholder
	return make(map[string]business.NetworkGasStats)
}
