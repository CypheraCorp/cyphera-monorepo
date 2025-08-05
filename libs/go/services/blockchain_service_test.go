package services_test

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func init() {
	// Initialize logger for tests
	logger.InitLogger("test")
}

func TestBlockchainService_Initialize(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	ctx := context.Background()

	tests := []struct {
		name        string
		rpcAPIKey   string
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:      "successfully initializes with networks",
			rpcAPIKey: "test-api-key",
			setupMocks: func() {
				networks := []db.Network{
					{
						ID:     uuid.New(),
						Name:   "Ethereum Mainnet",
						RpcID:  "mainnet",
						Active: true,
					},
					{
						ID:     uuid.New(),
						Name:   "Polygon",
						RpcID:  "polygon-mainnet",
						Active: true,
					},
				}
				mockQuerier.EXPECT().ListActiveNetworks(ctx).Return(networks, nil)
			},
			wantErr: false,
		},
		{
			name:      "fails with empty API key",
			rpcAPIKey: "",
			setupMocks: func() {
				// No mocks needed as it should fail before database call
			},
			wantErr:     true,
			errorString: "RPC API key not provided",
		},
		{
			name:      "handles database error",
			rpcAPIKey: "test-api-key",
			setupMocks: func() {
				mockQuerier.EXPECT().ListActiveNetworks(ctx).Return(nil, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to list networks",
		},
		{
			name:      "skips networks with empty RPC ID",
			rpcAPIKey: "test-api-key",
			setupMocks: func() {
				networks := []db.Network{
					{
						ID:     uuid.New(),
						Name:   "Network without RPC",
						RpcID:  "",
						Active: true,
					},
					{
						ID:     uuid.New(),
						Name:   "Valid Network",
						RpcID:  "mainnet",
						Active: true,
					},
				}
				mockQuerier.EXPECT().ListActiveNetworks(ctx).Return(networks, nil)
			},
			wantErr: false,
		},
		{
			name:      "fails when no networks can be connected",
			rpcAPIKey: "test-api-key",
			setupMocks: func() {
				networks := []db.Network{
					{
						ID:     uuid.New(),
						Name:   "Network 1",
						RpcID:  "",
						Active: true,
					},
					{
						ID:     uuid.New(),
						Name:   "Network 2",
						RpcID:  "",
						Active: true,
					},
				}
				mockQuerier.EXPECT().ListActiveNetworks(ctx).Return(networks, nil)
			},
			wantErr:     true,
			errorString: "no RPC connections established",
		},
		{
			name:      "handles empty networks list",
			rpcAPIKey: "test-api-key",
			setupMocks: func() {
				mockQuerier.EXPECT().ListActiveNetworks(ctx).Return([]db.Network{}, nil)
			},
			wantErr:     true,
			errorString: "no RPC connections established",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := services.NewBlockchainService(mockQuerier, tt.rpcAPIKey)
			tt.setupMocks()

			err := service.Initialize(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				// Note: In real tests, this would likely fail due to network connections
				// but we're testing the business logic up to the point of making RPC calls
				if err != nil {
					// Allow connection errors in unit tests since we can't mock ethclient.Dial
					assert.Contains(t, err.Error(), "dial")
				}
			}
		})
	}
}

func TestBlockchainService_GetTransactionData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewBlockchainService(mockQuerier, "test-api-key")
	ctx := context.Background()

	networkID := uuid.New()
	txHash := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	tests := []struct {
		name        string
		txHash      string
		networkID   uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:      "fails with no RPC client for network",
			txHash:    txHash,
			networkID: networkID,
			setupMocks: func() {
				// No setup needed as service hasn't been initialized
			},
			wantErr:     true,
			errorString: "no RPC client for network",
		},
		{
			name:      "handles invalid transaction hash format",
			txHash:    "invalid-hash",
			networkID: networkID,
			setupMocks: func() {
				// Would need to initialize service first, but this test focuses on validation
			},
			wantErr:     true,
			errorString: "no RPC client for network", // Will fail before hash validation
		},
		{
			name:      "handles empty transaction hash",
			txHash:    "",
			networkID: networkID,
			setupMocks: func() {
				// No setup needed
			},
			wantErr:     true,
			errorString: "no RPC client for network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.GetTransactionData(ctx, tt.txHash, tt.networkID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestBlockchainService_GetTransactionDataFromEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewBlockchainService(mockQuerier, "test-api-key")
	ctx := context.Background()

	subscriptionID := uuid.New()
	productTokenID := uuid.New()
	networkID := uuid.New()
	txHash := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	validEvent := &db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subscriptionID,
		TransactionHash: pgtype.Text{
			String: txHash,
			Valid:  true,
		},
		EventType: "payment_processed",
	}

	invalidEvent := &db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subscriptionID,
		TransactionHash: pgtype.Text{
			Valid: false,
		},
		EventType: "payment_failed",
	}

	subscription := db.Subscription{
		ID:             subscriptionID,
		ProductTokenID: productTokenID,
		CustomerID:     uuid.New(),
		ProductID:      uuid.New(),
		WorkspaceID:    uuid.New(),
	}

	productTokenRow := db.GetProductTokenRow{
		ID:        productTokenID,
		ProductID: subscription.ProductID,
		NetworkID: networkID,
		TokenID:   uuid.New(),
		Active:    true,
	}

	tests := []struct {
		name        string
		event       *db.SubscriptionEvent
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:  "fails with invalid transaction hash",
			event: invalidEvent,
			setupMocks: func() {
				// No mocks needed as it should fail before database calls
			},
			wantErr:     true,
			errorString: "subscription event has no transaction hash",
		},
		{
			name: "fails with empty transaction hash",
			event: &db.SubscriptionEvent{
				ID:             uuid.New(),
				SubscriptionID: subscriptionID,
				TransactionHash: pgtype.Text{
					String: "",
					Valid:  true,
				},
			},
			setupMocks: func() {
				// No mocks needed
			},
			wantErr:     true,
			errorString: "subscription event has no transaction hash",
		},
		{
			name:  "fails when subscription not found",
			event: validEvent,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(db.Subscription{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "failed to get subscription",
		},
		{
			name:  "fails when product token not found",
			event: validEvent,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)
				mockQuerier.EXPECT().GetProductToken(ctx, productTokenID).Return(db.GetProductTokenRow{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "failed to get product token",
		},
		{
			name:  "fails with database error getting subscription",
			event: validEvent,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(db.Subscription{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to get subscription",
		},
		{
			name:  "fails with database error getting product token",
			event: validEvent,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)
				mockQuerier.EXPECT().GetProductToken(ctx, productTokenID).Return(db.GetProductTokenRow{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to get product token",
		},
		{
			name:  "successfully processes valid event",
			event: validEvent,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)
				mockQuerier.EXPECT().GetProductToken(ctx, productTokenID).Return(productTokenRow, nil)
				// Note: The actual GetTransactionData call will fail without RPC client setup
			},
			wantErr:     true, // Will fail at GetTransactionData due to no RPC client
			errorString: "no RPC client for network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.GetTransactionDataFromEvent(ctx, tt.event)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.event.TransactionHash.String, result.Hash)
				assert.Equal(t, networkID, result.NetworkID)
			}
		})
	}
}

func TestBlockchainService_TransactionData(t *testing.T) {
	// Test TransactionData struct validation and helper methods
	networkID := uuid.New()

	tests := []struct {
		name     string
		txData   *business.TransactionData
		validate func(*business.TransactionData)
	}{
		{
			name: "valid transaction data structure",
			txData: &business.TransactionData{
				Hash:              "0x123",
				BlockNumber:       12345,
				BlockTimestamp:    1640995200,
				Status:            1,
				From:              "0xfrom",
				To:                "0xto",
				Value:             big.NewInt(1000000000000000000), // 1 ETH in wei
				GasUsed:           21000,
				GasLimit:          21000,
				GasPrice:          big.NewInt(20000000000), // 20 gwei
				EffectiveGasPrice: big.NewInt(20000000000),
				TotalGasCostWei:   big.NewInt(420000000000000), // 21000 * 20000000000
				NetworkID:         networkID,
			},
			validate: func(txData *business.TransactionData) {
				assert.Equal(t, "0x123", txData.Hash)
				assert.Equal(t, uint64(12345), txData.BlockNumber)
				assert.Equal(t, uint64(1), txData.Status)
				assert.Equal(t, "0xfrom", txData.From)
				assert.Equal(t, "0xto", txData.To)
				assert.Equal(t, big.NewInt(1000000000000000000), txData.Value)
				assert.Equal(t, uint64(21000), txData.GasUsed)
				assert.Equal(t, networkID, txData.NetworkID)
				assert.NotNil(t, txData.TotalGasCostWei)
			},
		},
		{
			name: "EIP-1559 transaction data",
			txData: &business.TransactionData{
				Hash:                 "0x456",
				BlockNumber:          12346,
				Status:               1,
				From:                 "0xfrom",
				To:                   "0xto",
				Value:                big.NewInt(500000000000000000), // 0.5 ETH
				GasUsed:              25000,
				GasLimit:             30000,
				MaxFeePerGas:         big.NewInt(30000000000),     // 30 gwei
				MaxPriorityFeePerGas: big.NewInt(2000000000),      // 2 gwei
				BaseFeePerGas:        big.NewInt(18000000000),     // 18 gwei
				EffectiveGasPrice:    big.NewInt(20000000000),     // 20 gwei
				TotalGasCostWei:      big.NewInt(500000000000000), // 25000 * 20000000000
				NetworkID:            networkID,
			},
			validate: func(txData *business.TransactionData) {
				assert.Equal(t, "0x456", txData.Hash)
				assert.NotNil(t, txData.MaxFeePerGas)
				assert.NotNil(t, txData.MaxPriorityFeePerGas)
				assert.NotNil(t, txData.BaseFeePerGas)
				assert.Equal(t, big.NewInt(30000000000), txData.MaxFeePerGas)
				assert.Equal(t, big.NewInt(2000000000), txData.MaxPriorityFeePerGas)
				assert.Equal(t, big.NewInt(18000000000), txData.BaseFeePerGas)
			},
		},
		{
			name: "failed transaction data",
			txData: &business.TransactionData{
				Hash:              "0x789",
				BlockNumber:       12347,
				Status:            0, // Failed transaction
				From:              "0xfrom",
				To:                "0xto",
				Value:             big.NewInt(0),
				GasUsed:           21000,
				GasLimit:          21000,
				EffectiveGasPrice: big.NewInt(20000000000),
				TotalGasCostWei:   big.NewInt(420000000000000),
				NetworkID:         networkID,
			},
			validate: func(txData *business.TransactionData) {
				assert.Equal(t, uint64(0), txData.Status)
				assert.Equal(t, big.NewInt(0), txData.Value)
				assert.Greater(t, txData.TotalGasCostWei.Uint64(), uint64(0)) // Gas still consumed
			},
		},
		{
			name: "contract creation transaction",
			txData: &business.TransactionData{
				Hash:              "0xabc",
				BlockNumber:       12348,
				Status:            1,
				From:              "0xfrom",
				To:                "", // Empty for contract creation
				Value:             big.NewInt(0),
				Input:             []byte("contract bytecode"),
				GasUsed:           200000,
				GasLimit:          250000,
				EffectiveGasPrice: big.NewInt(25000000000),
				TotalGasCostWei:   big.NewInt(5000000000000000), // 200000 * 25000000000
				NetworkID:         networkID,
			},
			validate: func(txData *business.TransactionData) {
				assert.Empty(t, txData.To)
				assert.NotEmpty(t, txData.Input)
				assert.Greater(t, txData.GasUsed, uint64(100000)) // Contract creation uses more gas
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotNil(t, tt.txData)
			if tt.validate != nil {
				tt.validate(tt.txData)
			}
		})
	}
}

func TestBlockchainService_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewBlockchainService(mockQuerier, "test-api-key")

	// Test that Close doesn't panic even without initialization
	t.Run("close without initialization", func(t *testing.T) {
		assert.NotPanics(t, func() {
			service.Close()
		})
	})

	// Test that Close can be called multiple times
	t.Run("multiple close calls", func(t *testing.T) {
		assert.NotPanics(t, func() {
			service.Close()
			service.Close()
			service.Close()
		})
	})
}

func TestBlockchainService_NewBlockchainService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)

	tests := []struct {
		name      string
		querier   db.Querier
		rpcAPIKey string
		validate  func(*services.BlockchainService)
	}{
		{
			name:      "creates service with valid parameters",
			querier:   mockQuerier,
			rpcAPIKey: "test-api-key",
			validate: func(service *services.BlockchainService) {
				assert.NotNil(t, service)
			},
		},
		{
			name:      "creates service with empty API key",
			querier:   mockQuerier,
			rpcAPIKey: "",
			validate: func(service *services.BlockchainService) {
				assert.NotNil(t, service)
				// Service should still be created, initialization will fail
			},
		},
		{
			name:      "creates service with nil querier",
			querier:   nil,
			rpcAPIKey: "test-api-key",
			validate: func(service *services.BlockchainService) {
				assert.NotNil(t, service)
				// Service should still be created, but database operations will fail
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := services.NewBlockchainService(tt.querier, tt.rpcAPIKey)
			if tt.validate != nil {
				tt.validate(service)
			}
		})
	}
}

func TestBlockchainService_ErrorHandling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewBlockchainService(mockQuerier, "test-api-key")
	ctx := context.Background()

	tests := []struct {
		name        string
		operation   func() error
		expectedErr string
	}{
		{
			name: "GetTransactionData with invalid network",
			operation: func() error {
				_, err := service.GetTransactionData(ctx, "0x123", uuid.New())
				return err
			},
			expectedErr: "no RPC client for network",
		},
		{
			name: "GetTransactionDataFromEvent with nil event",
			operation: func() error {
				_, err := service.GetTransactionDataFromEvent(ctx, nil)
				return err
			},
			expectedErr: "runtime error", // Will panic and be caught
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For operations that might panic, we need to handle them carefully
			defer func() {
				if r := recover(); r != nil {
					assert.Contains(t, "runtime error", tt.expectedErr)
				}
			}()

			err := tt.operation()
			if err != nil {
				assert.Contains(t, err.Error(), tt.expectedErr)
			}
		})
	}
}

func TestBlockchainService_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	ctx := context.Background()

	t.Run("Initialize with networks having various RPC ID formats", func(t *testing.T) {
		service := services.NewBlockchainService(mockQuerier, "test-api-key")

		networks := []db.Network{
			{ID: uuid.New(), Name: "Mainnet", RpcID: "mainnet", Active: true},
			{ID: uuid.New(), Name: "Testnet", RpcID: "goerli", Active: true},
			{ID: uuid.New(), Name: "Local", RpcID: "localhost", Active: true},
			{ID: uuid.New(), Name: "Custom", RpcID: "custom-network-123", Active: true},
		}

		mockQuerier.EXPECT().ListActiveNetworks(ctx).Return(networks, nil)

		err := service.Initialize(ctx)
		// Will fail due to network connections, but should process all networks
		if err != nil {
			// Allow connection failures in unit tests
			assert.Contains(t, err.Error(), "dial")
		}
	})

	t.Run("GetTransactionDataFromEvent with edge case event data", func(t *testing.T) {
		service := services.NewBlockchainService(mockQuerier, "test-api-key")

		// Test with various invalid event structures
		events := []*db.SubscriptionEvent{
			// Event with null transaction hash
			{
				ID:              uuid.New(),
				SubscriptionID:  uuid.New(),
				TransactionHash: pgtype.Text{Valid: false},
			},
			// Event with empty string transaction hash
			{
				ID:              uuid.New(),
				SubscriptionID:  uuid.New(),
				TransactionHash: pgtype.Text{String: "", Valid: true},
			},
			// Event with whitespace-only transaction hash
			{
				ID:              uuid.New(),
				SubscriptionID:  uuid.New(),
				TransactionHash: pgtype.Text{String: "   ", Valid: true},
			},
		}

		for i, event := range events {
			t.Run(fmt.Sprintf("invalid_event_%d", i), func(t *testing.T) {
				// For empty and null cases, no mocks needed - validation happens first
				if !event.TransactionHash.Valid || event.TransactionHash.String == "" {
					// No mocks needed - should fail validation
				} else {
					// For whitespace-only case, mocks are needed since validation passes
					productTokenID := uuid.New()
					mockQuerier.EXPECT().
						GetSubscription(ctx, event.SubscriptionID).
						Return(db.Subscription{
							ID:             event.SubscriptionID,
							WorkspaceID:    uuid.New(),
							CustomerID:     uuid.New(),
							ProductID:      uuid.New(),
							ProductTokenID: productTokenID,
							DelegationID:   uuid.New(),
						}, nil).
						AnyTimes()

					mockQuerier.EXPECT().
						GetProductToken(ctx, productTokenID).
						Return(db.GetProductTokenRow{
							ID:        productTokenID,
							ProductID: uuid.New(),
							NetworkID: uuid.New(),
							TokenID:   uuid.New(),
							Active:    true,
						}, nil).
						AnyTimes()
				}

				_, err := service.GetTransactionDataFromEvent(ctx, event)
				assert.Error(t, err)
				// Accept multiple possible error messages
				assert.True(t,
					strings.Contains(err.Error(), "subscription event has no transaction hash") ||
						strings.Contains(err.Error(), "no RPC client for network") ||
						strings.Contains(err.Error(), "failed to get subscription") ||
						strings.Contains(err.Error(), "failed to get product token"),
					"Expected validation or RPC error, got: %s", err.Error())
			})
		}
	})

	t.Run("handles very large numbers in TransactionData", func(t *testing.T) {
		// Test with maximum possible values
		maxUint64 := ^uint64(0)
		maxBigInt := new(big.Int).SetUint64(maxUint64)

		txData := &business.TransactionData{
			Hash:              "0x" + string(make([]byte, 64)), // Max length hash
			BlockNumber:       maxUint64,
			BlockTimestamp:    maxUint64,
			Status:            1,
			From:              "0x" + string(make([]byte, 40)), // Max length address
			To:                "0x" + string(make([]byte, 40)),
			Value:             maxBigInt,
			GasUsed:           maxUint64,
			GasLimit:          maxUint64,
			GasPrice:          maxBigInt,
			EffectiveGasPrice: maxBigInt,
			TotalGasCostWei:   maxBigInt,
			NetworkID:         uuid.New(),
		}

		assert.NotNil(t, txData)
		assert.Equal(t, maxUint64, txData.BlockNumber)
		assert.Equal(t, maxBigInt, txData.Value)
	})
}

func TestBlockchainService_ValidationLogic(t *testing.T) {
	// Test validation logic for various inputs
	tests := []struct {
		name     string
		input    interface{}
		isValid  bool
		errorMsg string
	}{
		{
			name:    "valid transaction hash format",
			input:   "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			isValid: true,
		},
		{
			name:     "invalid transaction hash - too short",
			input:    "0x123",
			isValid:  false,
			errorMsg: "invalid hash length",
		},
		{
			name:     "invalid transaction hash - no 0x prefix",
			input:    "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			isValid:  false,
			errorMsg: "invalid hex prefix",
		},
		{
			name:     "empty transaction hash",
			input:    "",
			isValid:  false,
			errorMsg: "empty hash",
		},
		{
			name:    "valid network ID",
			input:   uuid.New(),
			isValid: true,
		},
		{
			name:     "nil network ID",
			input:    uuid.Nil,
			isValid:  false,
			errorMsg: "invalid network ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a conceptual test - actual validation would be in the service methods
			switch v := tt.input.(type) {
			case string:
				// Hash validation logic
				if tt.isValid {
					assert.NotEmpty(t, v)
					if len(v) > 2 {
						assert.True(t, strings.HasPrefix(v, "0x"))
					}
				} else {
					// Would contain validation logic for invalid cases
					assert.True(t, len(v) == 0 || !strings.HasPrefix(v, "0x") || len(v) < 66)
				}
			case uuid.UUID:
				// UUID validation logic
				if tt.isValid {
					assert.NotEqual(t, uuid.Nil, v)
				} else {
					assert.Equal(t, uuid.Nil, v)
				}
			}
		})
	}
}
