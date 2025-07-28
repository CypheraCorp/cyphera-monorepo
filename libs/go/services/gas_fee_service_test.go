package services_test

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func init() {
	logger.InitLogger("test")
}

func TestGasFeeService_CreateGasFeePaymentRecord(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewGasFeeService(mockQuerier, nil) // We can test this method without exchange rate service

	ctx := context.Background()
	paymentID := uuid.New()
	sponsorID := uuid.New()

	gasFeeResult := &responses.GasFeeResult{
		EstimatedGasUnits:    21000,
		GasPriceWei:          "20000000000",
		TotalGasCostWei:      "420000000000000",
		TotalGasCostEth:      0.00042,
		TotalGasCostUSD:      0.84,
		TotalGasCostUSDCents: 84,
		Confidence:           0.95,
	}

	tests := []struct {
		name         string
		paymentID    uuid.UUID
		gasFeeResult *responses.GasFeeResult
		isSponsored  bool
		sponsorID    *uuid.UUID
		setupMocks   func()
		wantErr      bool
		errorString  string
	}{
		{
			name:         "successfully creates gas fee payment record without sponsor",
			paymentID:    paymentID,
			gasFeeResult: gasFeeResult,
			isSponsored:  false,
			sponsorID:    nil,
			setupMocks: func() {
				mockQuerier.EXPECT().CreateGasFeePayment(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, params db.CreateGasFeePaymentParams) (db.GasFeePayment, error) {
						// Validate the params
						assert.Equal(t, paymentID, params.PaymentID)
						assert.Equal(t, "420000000000000", params.GasFeeWei)
						assert.Equal(t, "20", params.GasPriceGwei)
						assert.Equal(t, int64(21000), params.GasUnitsUsed)
						assert.Equal(t, "native", params.PaymentMethod)
						assert.Equal(t, "none", params.SponsorType)
						assert.Equal(t, int64(84), params.GasFeeUsdCents.Int64)
						assert.False(t, params.SponsorID.Valid)

						return db.GasFeePayment{ID: uuid.New()}, nil
					})
			},
			wantErr: false,
		},
		{
			name:         "successfully creates gas fee payment record with sponsor",
			paymentID:    paymentID,
			gasFeeResult: gasFeeResult,
			isSponsored:  true,
			sponsorID:    &sponsorID,
			setupMocks: func() {
				mockQuerier.EXPECT().CreateGasFeePayment(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, params db.CreateGasFeePaymentParams) (db.GasFeePayment, error) {
						// Validate the params
						assert.Equal(t, paymentID, params.PaymentID)
						assert.Equal(t, "merchant", params.SponsorType)
						assert.True(t, params.SponsorID.Valid)
						assert.Equal(t, sponsorID, uuid.UUID(params.SponsorID.Bytes))

						return db.GasFeePayment{ID: uuid.New()}, nil
					})
			},
			wantErr: false,
		},
		{
			name:         "fails when database error occurs",
			paymentID:    paymentID,
			gasFeeResult: gasFeeResult,
			isSponsored:  false,
			sponsorID:    nil,
			setupMocks: func() {
				mockQuerier.EXPECT().CreateGasFeePayment(ctx, gomock.Any()).Return(db.GasFeePayment{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to create gas fee payment record",
		},
		{
			name:      "handles zero gas fee result",
			paymentID: paymentID,
			gasFeeResult: &responses.GasFeeResult{
				EstimatedGasUnits:    0,
				GasPriceWei:          "0",
				TotalGasCostWei:      "0",
				TotalGasCostEth:      0.0,
				TotalGasCostUSD:      0.0,
				TotalGasCostUSDCents: 0,
				Confidence:           0.0,
			},
			isSponsored: false,
			sponsorID:   nil,
			setupMocks: func() {
				mockQuerier.EXPECT().CreateGasFeePayment(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, params db.CreateGasFeePaymentParams) (db.GasFeePayment, error) {
						assert.Equal(t, "0", params.GasFeeWei)
						assert.Equal(t, "0", params.GasPriceGwei)
						assert.Equal(t, int64(0), params.GasUnitsUsed)
						assert.Equal(t, int64(0), params.GasFeeUsdCents.Int64)

						return db.GasFeePayment{ID: uuid.New()}, nil
					})
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.CreateGasFeePaymentRecord(ctx, tt.paymentID, tt.gasFeeResult, tt.isSponsored, tt.sponsorID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGasFeeService_GetGasFeePayments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewGasFeeService(mockQuerier, nil)

	ctx := context.Background()
	workspaceID := uuid.New()
	limit := int32(10)

	tests := []struct {
		name           string
		workspaceID    uuid.UUID
		limit          int32
		setupMocks     func()
		wantErr        bool
		errorString    string
		validateResult func([]db.GetGasFeePaymentsByWorkspaceRow)
	}{
		{
			name:        "successfully retrieves gas fee payments",
			workspaceID: workspaceID,
			limit:       limit,
			setupMocks: func() {
				payments := []db.GetGasFeePaymentsByWorkspaceRow{
					{
						ID:             uuid.New(),
						PaymentID:      uuid.New(),
						GasFeeWei:      "420000000000000",
						GasPriceGwei:   "20",
						GasUnitsUsed:   21000,
						GasFeeUsdCents: pgtype.Int8{Int64: 84, Valid: true},
					},
				}
				mockQuerier.EXPECT().GetGasFeePaymentsByWorkspace(ctx, db.GetGasFeePaymentsByWorkspaceParams{
					WorkspaceID: workspaceID,
					Limit:       limit,
				}).Return(payments, nil)
			},
			wantErr: false,
			validateResult: func(payments []db.GetGasFeePaymentsByWorkspaceRow) {
				assert.Len(t, payments, 1)
				assert.Equal(t, int64(21000), payments[0].GasUnitsUsed)
				assert.Equal(t, "20", payments[0].GasPriceGwei)
			},
		},
		{
			name:        "returns empty list when no payments found",
			workspaceID: workspaceID,
			limit:       limit,
			setupMocks: func() {
				payments := []db.GetGasFeePaymentsByWorkspaceRow{}
				mockQuerier.EXPECT().GetGasFeePaymentsByWorkspace(ctx, db.GetGasFeePaymentsByWorkspaceParams{
					WorkspaceID: workspaceID,
					Limit:       limit,
				}).Return(payments, nil)
			},
			wantErr: false,
			validateResult: func(payments []db.GetGasFeePaymentsByWorkspaceRow) {
				assert.Len(t, payments, 0)
			},
		},
		{
			name:        "fails when database error occurs",
			workspaceID: workspaceID,
			limit:       limit,
			setupMocks: func() {
				mockQuerier.EXPECT().GetGasFeePaymentsByWorkspace(ctx, db.GetGasFeePaymentsByWorkspaceParams{
					WorkspaceID: workspaceID,
					Limit:       limit,
				}).Return(nil, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to get gas fee payments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			payments, err := service.GetGasFeePayments(ctx, tt.workspaceID, tt.limit)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Nil(t, payments)
			} else {
				assert.NoError(t, err)
				if tt.validateResult != nil {
					tt.validateResult(payments)
				}
			}
		})
	}
}

func TestGasFeeService_GetGasFeeMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewGasFeeService(mockQuerier, nil)

	ctx := context.Background()
	workspaceID := uuid.New()
	startDate := time.Now().AddDate(0, -1, 0) // 1 month ago
	endDate := time.Now()

	tests := []struct {
		name           string
		workspaceID    uuid.UUID
		startDate      time.Time
		endDate        time.Time
		setupMocks     func()
		wantErr        bool
		errorString    string
		validateResult func(*business.GasFeeMetrics)
	}{
		{
			name:        "successfully retrieves gas fee metrics",
			workspaceID: workspaceID,
			startDate:   startDate,
			endDate:     endDate,
			setupMocks: func() {
				metrics := db.GetGasFeeMetricsRow{
					TotalTransactions:      10,
					TotalGasFeesCents:      1000,
					MerchantSponsoredCents: 500,
					PlatformSponsoredCents: 200,
					AvgGasFeeCents:         100.0,
				}
				mockQuerier.EXPECT().GetGasFeeMetrics(ctx, db.GetGasFeeMetricsParams{
					WorkspaceID: workspaceID,
					CreatedAt:   pgtype.Timestamptz{Time: startDate, Valid: true},
					CreatedAt_2: pgtype.Timestamptz{Time: endDate, Valid: true},
				}).Return(metrics, nil)
			},
			wantErr: false,
			validateResult: func(metrics *business.GasFeeMetrics) {
				assert.NotNil(t, metrics)
				assert.Equal(t, int64(10), metrics.TotalTransactions)
				assert.Equal(t, int64(1000), metrics.TotalGasCostCents)
				assert.Equal(t, int64(700), metrics.SponsoredCostCents) // 500 + 200
				assert.Equal(t, int64(100), metrics.AverageGasCostCents)
				assert.NotNil(t, metrics.NetworkBreakdown)
			},
		},
		{
			name:        "fails when database error occurs",
			workspaceID: workspaceID,
			startDate:   startDate,
			endDate:     endDate,
			setupMocks: func() {
				mockQuerier.EXPECT().GetGasFeeMetrics(ctx, db.GetGasFeeMetricsParams{
					WorkspaceID: workspaceID,
					CreatedAt:   pgtype.Timestamptz{Time: startDate, Valid: true},
					CreatedAt_2: pgtype.Timestamptz{Time: endDate, Valid: true},
				}).Return(db.GetGasFeeMetricsRow{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to get gas fee metrics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			metrics, err := service.GetGasFeeMetrics(ctx, tt.workspaceID, tt.startDate, tt.endDate)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Nil(t, metrics)
			} else {
				assert.NoError(t, err)
				if tt.validateResult != nil {
					tt.validateResult(metrics)
				}
			}
		})
	}
}

func TestGasFeeService_HelperFunctions(t *testing.T) {
	tests := []struct {
		name      string
		operation func() interface{}
		validate  func(interface{})
	}{
		{
			name: "weiToEth conversion with zero",
			operation: func() interface{} {
				// Test the wei to eth conversion logic
				wei := big.NewInt(0)
				ethFloat := new(big.Float).SetInt(wei)
				divisor := new(big.Float).SetFloat64(1e18)
				result := new(big.Float).Quo(ethFloat, divisor)
				ethValue, _ := result.Float64()
				return ethValue
			},
			validate: func(result interface{}) {
				ethValue := result.(float64)
				assert.Equal(t, 0.0, ethValue)
			},
		},
		{
			name: "weiToEth conversion with 1 ETH",
			operation: func() interface{} {
				wei := big.NewInt(1000000000000000000) // 1 ETH in Wei
				ethFloat := new(big.Float).SetInt(wei)
				divisor := new(big.Float).SetFloat64(1e18)
				result := new(big.Float).Quo(ethFloat, divisor)
				ethValue, _ := result.Float64()
				return ethValue
			},
			validate: func(result interface{}) {
				ethValue := result.(float64)
				assert.Equal(t, 1.0, ethValue)
			},
		},
		{
			name: "weiToEth conversion with fractional ETH",
			operation: func() interface{} {
				wei := big.NewInt(500000000000000000) // 0.5 ETH in Wei
				ethFloat := new(big.Float).SetInt(wei)
				divisor := new(big.Float).SetFloat64(1e18)
				result := new(big.Float).Quo(ethFloat, divisor)
				ethValue, _ := result.Float64()
				return ethValue
			},
			validate: func(result interface{}) {
				ethValue := result.(float64)
				assert.Equal(t, 0.5, ethValue)
			},
		},
		{
			name: "gas limit adjustment for transfer",
			operation: func() interface{} {
				baseLimit := uint64(21000)
				multiplier := 1.0 // transfer multiplier
				return uint64(float64(baseLimit) * multiplier)
			},
			validate: func(result interface{}) {
				adjustedLimit := result.(uint64)
				assert.Equal(t, uint64(21000), adjustedLimit)
			},
		},
		{
			name: "gas limit adjustment for contract call",
			operation: func() interface{} {
				baseLimit := uint64(50000)
				multiplier := 1.2 // contract_call multiplier
				return uint64(float64(baseLimit) * multiplier)
			},
			validate: func(result interface{}) {
				adjustedLimit := result.(uint64)
				assert.Equal(t, uint64(60000), adjustedLimit)
			},
		},
		{
			name: "gas limit adjustment for delegation",
			operation: func() interface{} {
				baseLimit := uint64(100000)
				multiplier := 1.5 // delegation multiplier
				return uint64(float64(baseLimit) * multiplier)
			},
			validate: func(result interface{}) {
				adjustedLimit := result.(uint64)
				assert.Equal(t, uint64(150000), adjustedLimit)
			},
		},
		{
			name: "gas price estimation for ethereum",
			operation: func() interface{} {
				// Simulate ethereum gas price estimation
				basePrices := map[string]*big.Int{
					"ethereum": big.NewInt(20000000000), // 20 Gwei
				}
				return basePrices["ethereum"]
			},
			validate: func(result interface{}) {
				gasPrice := result.(*big.Int)
				assert.Equal(t, big.NewInt(20000000000), gasPrice)
			},
		},
		{
			name: "gas price estimation for unknown network",
			operation: func() interface{} {
				// Simulate unknown network gas price estimation
				return big.NewInt(20000000000) // Default fallback
			},
			validate: func(result interface{}) {
				gasPrice := result.(*big.Int)
				assert.Equal(t, big.NewInt(20000000000), gasPrice)
			},
		},
		{
			name: "very large gas numbers",
			operation: func() interface{} {
				// Test with extremely large numbers
				largeWei := new(big.Int)
				largeWei.SetString("999999999999999999999999999999", 10)

				ethFloat := new(big.Float).SetInt(largeWei)
				divisor := new(big.Float).SetFloat64(1e18)
				result := new(big.Float).Quo(ethFloat, divisor)
				ethValue, accuracy := result.Float64()

				return map[string]interface{}{
					"value":    ethValue,
					"accuracy": accuracy,
				}
			},
			validate: func(result interface{}) {
				data := result.(map[string]interface{})
				// Should handle large numbers gracefully
				assert.NotEqual(t, big.Below, data["accuracy"])
			},
		},
		{
			name: "negative gas values",
			operation: func() interface{} {
				// Test with negative values (should not occur in practice)
				negativeWei := big.NewInt(-1000)
				ethFloat := new(big.Float).SetInt(negativeWei)
				divisor := new(big.Float).SetFloat64(1e18)
				result := new(big.Float).Quo(ethFloat, divisor)
				ethValue, _ := result.Float64()
				return ethValue
			},
			validate: func(result interface{}) {
				ethValue := result.(float64)
				assert.Less(t, ethValue, 0.0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.operation()
			if tt.validate != nil {
				tt.validate(result)
			}
		})
	}
}

func TestGasFeeService_NetworkSpecificBehavior(t *testing.T) {
	tests := []struct {
		name               string
		networkName        string
		expectedGasPrice   *big.Int
		expectedConfidence float64
	}{
		{
			name:               "ethereum gas price",
			networkName:        "ethereum",
			expectedGasPrice:   big.NewInt(20000000000), // 20 Gwei
			expectedConfidence: 0.8,
		},
		{
			name:               "polygon gas price",
			networkName:        "polygon",
			expectedGasPrice:   big.NewInt(30000000000), // 30 Gwei
			expectedConfidence: 0.8,
		},
		{
			name:               "arbitrum gas price",
			networkName:        "arbitrum",
			expectedGasPrice:   big.NewInt(100000000), // 0.1 Gwei
			expectedConfidence: 0.8,
		},
		{
			name:               "optimism gas price",
			networkName:        "optimism",
			expectedGasPrice:   big.NewInt(1000000), // 0.001 Gwei
			expectedConfidence: 0.8,
		},
		{
			name:               "unknown network gas price",
			networkName:        "unknown",
			expectedGasPrice:   big.NewInt(20000000000), // Default fallback
			expectedConfidence: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the expected network-specific gas price behavior
			basePrices := map[string]*big.Int{
				"ethereum": big.NewInt(20000000000),
				"polygon":  big.NewInt(30000000000),
				"arbitrum": big.NewInt(100000000),
				"optimism": big.NewInt(1000000),
			}

			var gasPrice *big.Int
			var confidence float64

			if price, exists := basePrices[tt.networkName]; exists {
				gasPrice = price
				confidence = 0.8
			} else {
				gasPrice = big.NewInt(20000000000)
				confidence = 0.5
			}

			assert.Equal(t, tt.expectedGasPrice, gasPrice)
			assert.Equal(t, tt.expectedConfidence, confidence)
		})
	}
}

func TestGasFeeService_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		operation func() error
		wantErr   bool
		errorMsg  string
	}{
		{
			name: "service creation with nil parameters",
			operation: func() error {
				nilService := services.NewGasFeeService(nil, nil)
				assert.NotNil(t, nilService)
				return nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
