package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cyphera/cyphera-api/libs/go/client/coinmarketcap"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func init() {
	logger.InitLogger("test")
}

func TestTokenService_GetToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewTokenService(mockQuerier, nil)
	ctx := context.Background()

	testID := uuid.New()
	expectedToken := db.Token{
		ID:              testID,
		Name:            "Test Token",
		Symbol:          "TEST",
		ContractAddress: "0x123...",
		Decimals:        18,
		Active:          true,
	}

	tests := []struct {
		name        string
		tokenID     uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:    "successfully gets token",
			tokenID: testID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetToken(ctx, testID).Return(expectedToken, nil)
			},
			wantErr: false,
		},
		{
			name:    "token not found",
			tokenID: testID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetToken(ctx, testID).Return(db.Token{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "token not found",
		},
		{
			name:    "database error",
			tokenID: testID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetToken(ctx, testID).Return(db.Token{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			token, err := service.GetToken(ctx, tt.tokenID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, token)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, token)
				assert.Equal(t, testID, token.ID)
			}
		})
	}
}

func TestTokenService_GetTokenByAddress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewTokenService(mockQuerier, nil)
	ctx := context.Background()

	testNetworkID := uuid.New()
	contractAddress := "0x123..."
	expectedToken := db.Token{
		ID:              uuid.New(),
		NetworkID:       testNetworkID,
		Name:            "Test Token",
		Symbol:          "TEST",
		ContractAddress: contractAddress,
		Decimals:        18,
		Active:          true,
	}

	tests := []struct {
		name            string
		networkID       uuid.UUID
		contractAddress string
		setupMocks      func()
		wantErr         bool
		errorString     string
	}{
		{
			name:            "successfully gets token by address",
			networkID:       testNetworkID,
			contractAddress: contractAddress,
			setupMocks: func() {
				mockQuerier.EXPECT().GetTokenByAddress(ctx, db.GetTokenByAddressParams{
					NetworkID:       testNetworkID,
					ContractAddress: contractAddress,
				}).Return(expectedToken, nil)
			},
			wantErr: false,
		},
		{
			name:            "token not found",
			networkID:       testNetworkID,
			contractAddress: contractAddress,
			setupMocks: func() {
				mockQuerier.EXPECT().GetTokenByAddress(ctx, db.GetTokenByAddressParams{
					NetworkID:       testNetworkID,
					ContractAddress: contractAddress,
				}).Return(db.Token{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "token not found",
		},
		{
			name:            "database error",
			networkID:       testNetworkID,
			contractAddress: contractAddress,
			setupMocks: func() {
				mockQuerier.EXPECT().GetTokenByAddress(ctx, db.GetTokenByAddressParams{
					NetworkID:       testNetworkID,
					ContractAddress: contractAddress,
				}).Return(db.Token{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			token, err := service.GetTokenByAddress(ctx, tt.networkID, tt.contractAddress)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, token)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, token)
				assert.Equal(t, testNetworkID, token.NetworkID)
				assert.Equal(t, contractAddress, token.ContractAddress)
			}
		})
	}
}

func TestTokenService_ListTokens(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewTokenService(mockQuerier, nil)
	ctx := context.Background()

	expectedTokens := []db.Token{
		{ID: uuid.New(), Name: "Token 1", Symbol: "TK1"},
		{ID: uuid.New(), Name: "Token 2", Symbol: "TK2"},
	}

	tests := []struct {
		name        string
		setupMocks  func()
		wantErr     bool
		errorString string
		wantCount   int
	}{
		{
			name: "successfully lists tokens",
			setupMocks: func() {
				mockQuerier.EXPECT().ListTokens(ctx).Return(expectedTokens, nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "returns empty list",
			setupMocks: func() {
				mockQuerier.EXPECT().ListTokens(ctx).Return([]db.Token{}, nil)
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "database error",
			setupMocks: func() {
				mockQuerier.EXPECT().ListTokens(ctx).Return(nil, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			tokens, err := service.ListTokens(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tokens)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, tokens, tt.wantCount)
			}
		})
	}
}

func TestTokenService_ListTokensByNetwork(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewTokenService(mockQuerier, nil)
	ctx := context.Background()

	testNetworkID := uuid.New()
	expectedTokens := []db.Token{
		{ID: uuid.New(), NetworkID: testNetworkID, Name: "Token 1", Symbol: "TK1"},
		{ID: uuid.New(), NetworkID: testNetworkID, Name: "Token 2", Symbol: "TK2"},
	}

	tests := []struct {
		name        string
		networkID   uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
		wantCount   int
	}{
		{
			name:      "successfully lists tokens by network",
			networkID: testNetworkID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListTokensByNetwork(ctx, testNetworkID).Return(expectedTokens, nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:      "returns empty list for network",
			networkID: testNetworkID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListTokensByNetwork(ctx, testNetworkID).Return([]db.Token{}, nil)
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:      "database error",
			networkID: testNetworkID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListTokensByNetwork(ctx, testNetworkID).Return(nil, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			tokens, err := service.ListTokensByNetwork(ctx, tt.networkID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tokens)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, tokens, tt.wantCount)
			}
		})
	}
}

func TestTokenService_GetTokenQuote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)

	tests := []struct {
		name           string
		params         params.TokenQuoteParams
		cmcClient      *coinmarketcap.Client
		setupMocks     func(*coinmarketcap.Client)
		wantErr        bool
		errorString    string
		validateResult func(*responses.TokenQuoteResult)
	}{
		{
			name: "nil CMC client returns error",
			params: params.TokenQuoteParams{
				TokenID:    uuid.New(),
				NetworkID:  uuid.New(),
				AmountWei:  "1000000000000000000",
				ToCurrency: "USD",
			},
			cmcClient:   nil,
			setupMocks:  func(*coinmarketcap.Client) {},
			wantErr:     true,
			errorString: "price service is unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := services.NewTokenService(mockQuerier, tt.cmcClient)
			tt.setupMocks(tt.cmcClient)

			result, err := service.GetTokenQuote(context.Background(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validateResult != nil {
					tt.validateResult(result)
				}
			}
		})
	}
}

func TestTokenService_CreateToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewTokenService(mockQuerier, nil)
	ctx := context.Background()

	testNetworkID := uuid.New()
	expectedToken := db.Token{
		ID:              uuid.New(),
		NetworkID:       testNetworkID,
		Name:            "Test Token",
		Symbol:          "TEST",
		ContractAddress: "0x123...",
		Decimals:        18,
		Active:          true,
	}

	tests := []struct {
		name        string
		params      services.CreateTokenParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully creates token",
			params: services.CreateTokenParams{
				NetworkID:       testNetworkID,
				GasToken:        false,
				Name:            "Test Token",
				Symbol:          "TEST",
				ContractAddress: "0x123...",
				Decimals:        18,
				Active:          true,
			},
			setupMocks: func() {
				// Check if token already exists (should return not found)
				mockQuerier.EXPECT().GetTokenByAddress(ctx, db.GetTokenByAddressParams{
					NetworkID:       testNetworkID,
					ContractAddress: "0x123...",
				}).Return(db.Token{}, pgx.ErrNoRows)

				// Create the token
				mockQuerier.EXPECT().CreateToken(ctx, db.CreateTokenParams{
					NetworkID:       testNetworkID,
					GasToken:        false,
					Name:            "Test Token",
					Symbol:          "TEST",
					ContractAddress: "0x123...",
					Decimals:        18,
					Active:          true,
				}).Return(expectedToken, nil)
			},
			wantErr: false,
		},
		{
			name: "fails with empty name",
			params: services.CreateTokenParams{
				NetworkID:       testNetworkID,
				Name:            "",
				Symbol:          "TEST",
				ContractAddress: "0x123...",
				Decimals:        18,
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "token name is required",
		},
		{
			name: "fails with empty symbol",
			params: services.CreateTokenParams{
				NetworkID:       testNetworkID,
				Name:            "Test Token",
				Symbol:          "",
				ContractAddress: "0x123...",
				Decimals:        18,
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "token symbol is required",
		},
		{
			name: "fails with empty contract address",
			params: services.CreateTokenParams{
				NetworkID:       testNetworkID,
				Name:            "Test Token",
				Symbol:          "TEST",
				ContractAddress: "",
				Decimals:        18,
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "contract address is required",
		},
		{
			name: "fails with negative decimals",
			params: services.CreateTokenParams{
				NetworkID:       testNetworkID,
				Name:            "Test Token",
				Symbol:          "TEST",
				ContractAddress: "0x123...",
				Decimals:        -1,
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "decimals must be non-negative",
		},
		{
			name: "fails when token already exists",
			params: services.CreateTokenParams{
				NetworkID:       testNetworkID,
				Name:            "Test Token",
				Symbol:          "TEST",
				ContractAddress: "0x123...",
				Decimals:        18,
			},
			setupMocks: func() {
				// Token already exists
				mockQuerier.EXPECT().GetTokenByAddress(ctx, db.GetTokenByAddressParams{
					NetworkID:       testNetworkID,
					ContractAddress: "0x123...",
				}).Return(expectedToken, nil)
			},
			wantErr:     true,
			errorString: "token already exists for this network and contract address",
		},
		{
			name: "handles database error during creation",
			params: services.CreateTokenParams{
				NetworkID:       testNetworkID,
				Name:            "Test Token",
				Symbol:          "TEST",
				ContractAddress: "0x123...",
				Decimals:        18,
			},
			setupMocks: func() {
				// Check if token exists (returns not found)
				mockQuerier.EXPECT().GetTokenByAddress(ctx, db.GetTokenByAddressParams{
					NetworkID:       testNetworkID,
					ContractAddress: "0x123...",
				}).Return(db.Token{}, pgx.ErrNoRows)

				// Creation fails
				mockQuerier.EXPECT().CreateToken(ctx, gomock.Any()).Return(db.Token{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to create token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			token, err := service.CreateToken(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, token)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, token)
				assert.Equal(t, tt.params.Name, token.Name)
				assert.Equal(t, tt.params.Symbol, token.Symbol)
			}
		})
	}
}

func TestTokenService_UpdateToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewTokenService(mockQuerier, nil)
	ctx := context.Background()

	testID := uuid.New()
	existingToken := db.Token{
		ID:     testID,
		Name:   "Original Token",
		Symbol: "ORIG",
	}

	tests := []struct {
		name        string
		params      services.UpdateTokenParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "update not implemented",
			params: services.UpdateTokenParams{
				ID: testID,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetToken(ctx, testID).Return(existingToken, nil)
			},
			wantErr:     true,
			errorString: "token update not implemented",
		},
		{
			name: "token not found",
			params: services.UpdateTokenParams{
				ID: testID,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetToken(ctx, testID).Return(db.Token{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "token not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			token, err := service.UpdateToken(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, token)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, token)
			}
		})
	}
}

func TestTokenService_DeleteToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewTokenService(mockQuerier, nil)
	ctx := context.Background()

	testID := uuid.New()
	existingToken := db.Token{
		ID:   testID,
		Name: "Test Token",
	}

	tests := []struct {
		name        string
		tokenID     uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:    "delete not implemented",
			tokenID: testID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetToken(ctx, testID).Return(existingToken, nil)
			},
			wantErr:     true,
			errorString: "token deletion not implemented",
		},
		{
			name:    "token not found",
			tokenID: testID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetToken(ctx, testID).Return(db.Token{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "token not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.DeleteToken(ctx, tt.tokenID)

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

func TestTokenService_ValidateTokenSymbol(t *testing.T) {
	service := services.NewTokenService(nil, nil)

	tests := []struct {
		name        string
		symbol      string
		wantErr     bool
		errorString string
	}{
		{
			name:    "valid symbol",
			symbol:  "BTC",
			wantErr: false,
		},
		{
			name:    "valid symbol with max length",
			symbol:  "ABCDEFGHIJ", // 10 characters
			wantErr: false,
		},
		{
			name:        "empty symbol",
			symbol:      "",
			wantErr:     true,
			errorString: "token symbol cannot be empty",
		},
		{
			name:        "symbol too long",
			symbol:      "ABCDEFGHIJK", // 11 characters
			wantErr:     true,
			errorString: "token symbol too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateTokenSymbol(tt.symbol)

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

func TestTokenService_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewTokenService(mockQuerier, nil)

	tests := []struct {
		name        string
		operation   func() error
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil context handling",
			operation: func() error {
				// Mock the database call that will happen even with nil context
				mockQuerier.EXPECT().GetToken(gomock.Any(), gomock.Any()).Return(db.Token{}, assert.AnError)
				_, err := service.GetToken(context.TODO(), uuid.New())
				return err
			},
			expectError: true,
		},
		{
			name: "zero UUID handling",
			operation: func() error {
				mockQuerier.EXPECT().GetToken(gomock.Any(), uuid.Nil).Return(db.Token{}, pgx.ErrNoRows)
				_, err := service.GetToken(context.Background(), uuid.Nil)
				return err
			},
			expectError: true,
			errorMsg:    "token not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()

			if tt.expectError {
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

func TestTokenService_BoundaryConditions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewTokenService(mockQuerier, nil)
	ctx := context.Background()

	tests := []struct {
		name        string
		params      services.CreateTokenParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "maximum decimals value",
			params: services.CreateTokenParams{
				NetworkID:       uuid.New(),
				Name:            "Test Token",
				Symbol:          "TEST",
				ContractAddress: "0x123...",
				Decimals:        2147483647, // Max int32
				Active:          true,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetTokenByAddress(ctx, gomock.Any()).Return(db.Token{}, pgx.ErrNoRows)
				mockQuerier.EXPECT().CreateToken(ctx, gomock.Any()).Return(db.Token{}, nil)
			},
			wantErr: false,
		},
		{
			name: "zero decimals",
			params: services.CreateTokenParams{
				NetworkID:       uuid.New(),
				Name:            "Test Token",
				Symbol:          "TEST",
				ContractAddress: "0x123...",
				Decimals:        0,
				Active:          true,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetTokenByAddress(ctx, gomock.Any()).Return(db.Token{}, pgx.ErrNoRows)
				mockQuerier.EXPECT().CreateToken(ctx, gomock.Any()).Return(db.Token{}, nil)
			},
			wantErr: false,
		},
		{
			name: "very long token name",
			params: services.CreateTokenParams{
				NetworkID:       uuid.New(),
				Name:            "Very Long Token Name That Exceeds Normal Limits And Keeps Going And Going Until It Becomes Ridiculously Long But We Should Support It Anyway",
				Symbol:          "LONG",
				ContractAddress: "0x123...",
				Decimals:        18,
				Active:          true,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetTokenByAddress(ctx, gomock.Any()).Return(db.Token{}, pgx.ErrNoRows)
				mockQuerier.EXPECT().CreateToken(ctx, gomock.Any()).Return(db.Token{}, nil)
			},
			wantErr: false,
		},
		{
			name: "very long contract address",
			params: services.CreateTokenParams{
				NetworkID:       uuid.New(),
				Name:            "Test Token",
				Symbol:          "TEST",
				ContractAddress: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				Decimals:        18,
				Active:          true,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetTokenByAddress(ctx, gomock.Any()).Return(db.Token{}, pgx.ErrNoRows)
				mockQuerier.EXPECT().CreateToken(ctx, gomock.Any()).Return(db.Token{}, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			token, err := service.CreateToken(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, token)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTokenService_ConcurrentAccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewTokenService(mockQuerier, nil)
	ctx := context.Background()

	// Test concurrent token retrieval
	t.Run("concurrent token retrieval", func(t *testing.T) {
		const numGoroutines = 10
		resultsChan := make(chan *db.Token, numGoroutines)
		errorsChan := make(chan error, numGoroutines)

		// Setup mock for concurrent calls
		testToken := db.Token{
			ID:     uuid.New(),
			Name:   "Concurrent Token",
			Symbol: "CONC",
		}

		mockQuerier.EXPECT().GetToken(ctx, testToken.ID).Return(testToken, nil).Times(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				token, err := service.GetToken(ctx, testToken.ID)
				if err != nil {
					errorsChan <- err
				} else {
					resultsChan <- token
				}
			}()
		}

		// Collect results
		var results []*db.Token
		var errors []error

		for i := 0; i < numGoroutines; i++ {
			select {
			case result := <-resultsChan:
				results = append(results, result)
			case err := <-errorsChan:
				errors = append(errors, err)
			}
		}

		// Verify all operations completed without errors
		assert.Empty(t, errors)
		assert.Len(t, results, numGoroutines)

		// Verify all results are consistent
		for _, result := range results {
			assert.NotNil(t, result)
			assert.Equal(t, testToken.ID, result.ID)
			assert.Equal(t, testToken.Name, result.Name)
		}
	})
}
