package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func init() {
	logger.InitLogger("test")
}

func TestNetworkService_GetNetwork(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewNetworkService(mockQuerier)
	ctx := context.Background()

	networkID := uuid.New()
	expectedNetwork := db.Network{
		ID:          networkID,
		Name:        "Ethereum",
		Type:        "evm",
		NetworkType: "ethereum",
		ChainID:     1,
		Active:      true,
	}

	tests := []struct {
		name        string
		networkID   uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:      "successfully gets network",
			networkID: networkID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetNetwork(ctx, networkID).Return(expectedNetwork, nil)
			},
			wantErr: false,
		},
		{
			name:      "network not found",
			networkID: networkID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetNetwork(ctx, networkID).Return(db.Network{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "network not found",
		},
		{
			name:      "database error",
			networkID: networkID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetNetwork(ctx, networkID).Return(db.Network{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			network, err := service.GetNetwork(ctx, tt.networkID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, network)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, network)
				assert.Equal(t, tt.networkID, network.ID)
				assert.Equal(t, expectedNetwork.Name, network.Name)
				assert.Equal(t, expectedNetwork.ChainID, network.ChainID)
			}
		})
	}
}

func TestNetworkService_GetNetworkByChainID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewNetworkService(mockQuerier)
	ctx := context.Background()

	chainID := int32(1)
	expectedNetwork := db.Network{
		ID:          uuid.New(),
		Name:        "Ethereum",
		Type:        "evm",
		NetworkType: "ethereum",
		ChainID:     chainID,
		Active:      true,
	}

	tests := []struct {
		name        string
		chainID     int32
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:    "successfully gets network by chain ID",
			chainID: chainID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetNetworkByChainID(ctx, chainID).Return(expectedNetwork, nil)
			},
			wantErr: false,
		},
		{
			name:    "network not found by chain ID",
			chainID: chainID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetNetworkByChainID(ctx, chainID).Return(db.Network{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "network not found",
		},
		{
			name:    "database error",
			chainID: chainID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetNetworkByChainID(ctx, chainID).Return(db.Network{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			network, err := service.GetNetworkByChainID(ctx, tt.chainID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, network)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, network)
				assert.Equal(t, tt.chainID, network.ChainID)
				assert.Equal(t, expectedNetwork.Name, network.Name)
			}
		})
	}
}

func TestNetworkService_CreateNetwork(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewNetworkService(mockQuerier)
	ctx := context.Background()

	tests := []struct {
		name        string
		params      params.CreateNetworkParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully creates basic network",
			params: params.CreateNetworkParams{
				Name:              "Ethereum",
				Type:              "evm",
				NetworkType:       "ethereum",
				CircleNetworkType: "ETH",
				BlockExplorerURL:  "https://etherscan.io",
				ChainID:           1,
				IsTestnet:         false,
				Active:            true,
				LogoURL:           "https://example.com/eth.png",
				DisplayName:       "Ethereum Mainnet",
				ChainNamespace:    "eip155",
			},
			setupMocks: func() {
				expectedNetwork := db.Network{
					ID:          uuid.New(),
					Name:        "Ethereum",
					Type:        "evm",
					NetworkType: "ethereum",
					ChainID:     1,
					Active:      true,
				}
				mockQuerier.EXPECT().CreateNetwork(ctx, gomock.Any()).Return(expectedNetwork, nil)
			},
			wantErr: false,
		},
		{
			name: "successfully creates network with gas config",
			params: params.CreateNetworkParams{
				Name:              "Polygon",
				Type:              "evm",
				NetworkType:       "polygon",
				CircleNetworkType: "MATIC",
				ChainID:           137,
				IsTestnet:         false,
				Active:            true,
				GasConfig: &params.CreateGasConfigParams{
					BaseFeeMultiplier:     1.5,
					PriorityFeeMultiplier: 2.0,
					DeploymentGasLimit:    "2000000",
					TokenTransferGasLimit: "21000",
					SupportsEIP1559:       true,
					GasOracleURL:          "https://gasstation-mainnet.matic.network",
					GasRefreshIntervalMs:  30000,
					AverageBlockTimeMs:    2000,
					PeakHoursMultiplier:   1.2,
					GasPriorityLevels: map[string]interface{}{
						"slow":     map[string]interface{}{"maxFeePerGas": "30", "maxPriorityFeePerGas": "1"},
						"standard": map[string]interface{}{"maxFeePerGas": "50", "maxPriorityFeePerGas": "2"},
						"fast":     map[string]interface{}{"maxFeePerGas": "100", "maxPriorityFeePerGas": "5"},
					},
				},
			},
			setupMocks: func() {
				expectedNetwork := db.Network{
					ID:          uuid.New(),
					Name:        "Polygon",
					Type:        "evm",
					NetworkType: "polygon",
					ChainID:     137,
					Active:      true,
				}
				mockQuerier.EXPECT().CreateNetwork(ctx, gomock.Any()).Return(expectedNetwork, nil)
			},
			wantErr: false,
		},
		{
			name: "fails with invalid gas priority levels JSON",
			params: params.CreateNetworkParams{
				Name:              "Test Network",
				Type:              "evm",
				NetworkType:       "ethereum",
				CircleNetworkType: "ETH",
				ChainID:           1337,
				IsTestnet:         true,
				Active:            true,
				GasConfig: &params.CreateGasConfigParams{
					GasPriorityLevels: map[string]interface{}{
						"invalid": make(chan int), // This will cause JSON marshal to fail
					},
				},
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "failed to process gas priority levels",
		},
		{
			name: "handles database error",
			params: params.CreateNetworkParams{
				Name:              "Ethereum",
				Type:              "evm",
				NetworkType:       "ethereum",
				CircleNetworkType: "ETH",
				ChainID:           1,
				IsTestnet:         false,
				Active:            true,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().CreateNetwork(ctx, gomock.Any()).Return(db.Network{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to create network",
		},
		{
			name: "creates testnet network successfully",
			params: params.CreateNetworkParams{
				Name:              "Goerli",
				Type:              "evm",
				NetworkType:       "ethereum",
				CircleNetworkType: "ETH-GOERLI",
				BlockExplorerURL:  "https://goerli.etherscan.io",
				ChainID:           5,
				IsTestnet:         true,
				Active:            true,
				DisplayName:       "Ethereum Goerli Testnet",
			},
			setupMocks: func() {
				expectedNetwork := db.Network{
					ID:        uuid.New(),
					Name:      "Goerli",
					Type:      "evm",
					ChainID:   5,
					IsTestnet: true,
					Active:    true,
				}
				mockQuerier.EXPECT().CreateNetwork(ctx, gomock.Any()).Return(expectedNetwork, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			network, err := service.CreateNetwork(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, network)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, network)
				assert.Equal(t, tt.params.Name, network.Name)
				assert.Equal(t, tt.params.ChainID, network.ChainID)
				assert.Equal(t, tt.params.IsTestnet, network.IsTestnet)
				assert.Equal(t, tt.params.Active, network.Active)
			}
		})
	}
}

func TestNetworkService_UpdateNetwork(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewNetworkService(mockQuerier)
	ctx := context.Background()

	networkID := uuid.New()
	isTestnet := false
	isActive := true

	tests := []struct {
		name        string
		params      params.UpdateNetworkParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully updates network",
			params: params.UpdateNetworkParams{
				ID:                networkID,
				Name:              "Updated Ethereum",
				Type:              "evm",
				NetworkType:       "ethereum",
				CircleNetworkType: "ETH",
				BlockExplorerURL:  "https://etherscan.io",
				ChainID:           1,
				IsTestnet:         &isTestnet,
				Active:            &isActive,
				LogoURL:           "https://example.com/eth-new.png",
				DisplayName:       "Ethereum Mainnet Updated",
				ChainNamespace:    "eip155",
			},
			setupMocks: func() {
				updatedNetwork := db.Network{
					ID:        networkID,
					Name:      "Updated Ethereum",
					Type:      "evm",
					ChainID:   1,
					IsTestnet: false,
					Active:    true,
				}
				mockQuerier.EXPECT().UpdateNetwork(ctx, gomock.Any()).Return(updatedNetwork, nil)
			},
			wantErr: false,
		},
		{
			name: "successfully updates network with gas config",
			params: params.UpdateNetworkParams{
				ID:                networkID,
				Name:              "Polygon",
				Type:              "evm",
				NetworkType:       "polygon",
				CircleNetworkType: "MATIC",
				ChainID:           137,
				IsTestnet:         &isTestnet,
				Active:            &isActive,
				GasConfig: &params.UpdateGasConfigParams{
					BaseFeeMultiplier:     networkFloatPtr(1.8),
					PriorityFeeMultiplier: networkFloatPtr(2.2),
					DeploymentGasLimit:    networkStringPtr("2500000"),
					TokenTransferGasLimit: networkStringPtr("25000"),
					SupportsEIP1559:       networkBoolPtr(true),
					GasOracleURL:          networkStringPtr("https://gasstation-mainnet.matic.network/v2"),
					GasRefreshIntervalMs:  networkInt32Ptr(25000),
					AverageBlockTimeMs:    networkInt32Ptr(1800),
					PeakHoursMultiplier:   networkFloatPtr(1.3),
					GasPriorityLevels: map[string]interface{}{
						"eco":      map[string]interface{}{"maxFeePerGas": "20", "maxPriorityFeePerGas": "1"},
						"standard": map[string]interface{}{"maxFeePerGas": "60", "maxPriorityFeePerGas": "3"},
						"fast":     map[string]interface{}{"maxFeePerGas": "120", "maxPriorityFeePerGas": "6"},
					},
				},
			},
			setupMocks: func() {
				updatedNetwork := db.Network{
					ID:      networkID,
					Name:    "Polygon",
					Type:    "evm",
					ChainID: 137,
					Active:  true,
				}
				mockQuerier.EXPECT().UpdateNetwork(ctx, gomock.Any()).Return(updatedNetwork, nil)
			},
			wantErr: false,
		},
		{
			name: "network not found",
			params: params.UpdateNetworkParams{
				ID:                networkID,
				Name:              "Not Found Network",
				Type:              "evm",
				NetworkType:       "ethereum",
				CircleNetworkType: "ETH",
				ChainID:           1,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().UpdateNetwork(ctx, gomock.Any()).Return(db.Network{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "network not found",
		},
		{
			name: "fails with invalid gas priority levels JSON",
			params: params.UpdateNetworkParams{
				ID:                networkID,
				Name:              "Test Network",
				Type:              "evm",
				NetworkType:       "ethereum",
				CircleNetworkType: "ETH",
				ChainID:           1337,
				GasConfig: &params.UpdateGasConfigParams{
					GasPriorityLevels: map[string]interface{}{
						"invalid": make(chan int), // This will cause JSON marshal to fail
					},
				},
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "failed to process gas priority levels",
		},
		{
			name: "database update error",
			params: params.UpdateNetworkParams{
				ID:                networkID,
				Name:              "Ethereum",
				Type:              "evm",
				NetworkType:       "ethereum",
				CircleNetworkType: "ETH",
				ChainID:           1,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().UpdateNetwork(ctx, gomock.Any()).Return(db.Network{}, errors.New("update error"))
			},
			wantErr:     true,
			errorString: "failed to update network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			network, err := service.UpdateNetwork(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, network)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, network)
				assert.Equal(t, tt.params.ID, network.ID)
				assert.Equal(t, tt.params.Name, network.Name)
				assert.Equal(t, tt.params.ChainID, network.ChainID)
			}
		})
	}
}

func TestNetworkService_DeleteNetwork(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewNetworkService(mockQuerier)
	ctx := context.Background()

	networkID := uuid.New()

	tests := []struct {
		name        string
		networkID   uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:      "successfully deletes network",
			networkID: networkID,
			setupMocks: func() {
				mockQuerier.EXPECT().DeleteNetwork(ctx, networkID).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "network not found",
			networkID: networkID,
			setupMocks: func() {
				mockQuerier.EXPECT().DeleteNetwork(ctx, networkID).Return(pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "network not found",
		},
		{
			name:      "database delete error",
			networkID: networkID,
			setupMocks: func() {
				mockQuerier.EXPECT().DeleteNetwork(ctx, networkID).Return(errors.New("delete error"))
			},
			wantErr:     true,
			errorString: "failed to delete network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.DeleteNetwork(ctx, tt.networkID)

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

func TestNetworkService_ListNetworks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewNetworkService(mockQuerier)
	ctx := context.Background()

	expectedNetworks := []db.Network{
		{ID: uuid.New(), Name: "Ethereum", ChainID: 1, IsTestnet: false, Active: true},
		{ID: uuid.New(), Name: "Polygon", ChainID: 137, IsTestnet: false, Active: true},
		{ID: uuid.New(), Name: "Goerli", ChainID: 5, IsTestnet: true, Active: true},
	}

	isTestnet := false
	isActive := true

	tests := []struct {
		name        string
		params      params.ListNetworksParams
		setupMocks  func()
		wantErr     bool
		errorString string
		wantCount   int
	}{
		{
			name:   "successfully lists all networks",
			params: params.ListNetworksParams{},
			setupMocks: func() {
				mockQuerier.EXPECT().ListNetworks(ctx, db.ListNetworksParams{}).Return(expectedNetworks, nil)
			},
			wantErr:   false,
			wantCount: 3,
		},
		{
			name: "successfully lists mainnet networks only",
			params: params.ListNetworksParams{
				IsTestnet: &isTestnet,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().ListNetworks(ctx, db.ListNetworksParams{
					IsTestnet: pgtype.Bool{Bool: false, Valid: true},
				}).Return(expectedNetworks[:2], nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "successfully lists active networks only",
			params: params.ListNetworksParams{
				IsActive: &isActive,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().ListNetworks(ctx, db.ListNetworksParams{
					IsActive: pgtype.Bool{Bool: true, Valid: true},
				}).Return(expectedNetworks, nil)
			},
			wantErr:   false,
			wantCount: 3,
		},
		{
			name: "successfully lists with both filters",
			params: params.ListNetworksParams{
				IsTestnet: &isTestnet,
				IsActive:  &isActive,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().ListNetworks(ctx, db.ListNetworksParams{
					IsTestnet: pgtype.Bool{Bool: false, Valid: true},
					IsActive:  pgtype.Bool{Bool: true, Valid: true},
				}).Return(expectedNetworks[:2], nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:   "returns empty list",
			params: params.ListNetworksParams{},
			setupMocks: func() {
				mockQuerier.EXPECT().ListNetworks(ctx, db.ListNetworksParams{}).Return([]db.Network{}, nil)
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:   "database error",
			params: params.ListNetworksParams{},
			setupMocks: func() {
				mockQuerier.EXPECT().ListNetworks(ctx, db.ListNetworksParams{}).Return(nil, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve networks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			networks, err := service.ListNetworks(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, networks)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, networks, tt.wantCount)
			}
		})
	}
}

func TestNetworkService_ListActiveTokensByNetwork(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewNetworkService(mockQuerier)
	ctx := context.Background()

	networkID := uuid.New()
	expectedTokens := []db.Token{
		{ID: uuid.New(), NetworkID: networkID, Symbol: "ETH", Name: "Ethereum", Active: true},
		{ID: uuid.New(), NetworkID: networkID, Symbol: "USDC", Name: "USD Coin", Active: true},
		{ID: uuid.New(), NetworkID: networkID, Symbol: "USDT", Name: "Tether", Active: true},
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
			networkID: networkID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListActiveTokensByNetwork(ctx, networkID).Return(expectedTokens, nil)
			},
			wantErr:   false,
			wantCount: 3,
		},
		{
			name:      "returns empty list for network with no tokens",
			networkID: networkID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListActiveTokensByNetwork(ctx, networkID).Return([]db.Token{}, nil)
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:      "database error",
			networkID: networkID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListActiveTokensByNetwork(ctx, networkID).Return(nil, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			tokens, err := service.ListActiveTokensByNetwork(ctx, tt.networkID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tokens)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, tokens, tt.wantCount)
				if tt.wantCount > 0 {
					for _, token := range tokens {
						assert.Equal(t, networkID, token.NetworkID)
						assert.True(t, token.Active)
					}
				}
			}
		})
	}
}

// Test edge cases and validation scenarios
func TestNetworkService_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewNetworkService(mockQuerier)
	ctx := context.Background()

	t.Run("create network with nil gas config is handled correctly", func(t *testing.T) {
		params := params.CreateNetworkParams{
			Name:              "Simple Network",
			Type:              "evm",
			NetworkType:       "ethereum",
			CircleNetworkType: "ETH",
			ChainID:           1,
			IsTestnet:         false,
			Active:            true,
			GasConfig:         nil,
		}

		expectedNetwork := db.Network{
			ID:      uuid.New(),
			Name:    "Simple Network",
			Type:    "evm",
			ChainID: 1,
			Active:  true,
		}

		mockQuerier.EXPECT().CreateNetwork(ctx, gomock.Any()).Return(expectedNetwork, nil)

		network, err := service.CreateNetwork(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, network)
		assert.Equal(t, params.Name, network.Name)
	})

	t.Run("update network with nil gas config is handled correctly", func(t *testing.T) {
		networkID := uuid.New()
		params := params.UpdateNetworkParams{
			ID:                networkID,
			Name:              "Updated Network",
			Type:              "evm",
			NetworkType:       "ethereum",
			CircleNetworkType: "ETH",
			ChainID:           1,
			GasConfig:         nil,
		}

		expectedNetwork := db.Network{
			ID:      networkID,
			Name:    "Updated Network",
			Type:    "evm",
			ChainID: 1,
		}

		mockQuerier.EXPECT().UpdateNetwork(ctx, gomock.Any()).Return(expectedNetwork, nil)

		network, err := service.UpdateNetwork(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, network)
		assert.Equal(t, params.Name, network.Name)
	})

	t.Run("empty string fields are handled correctly", func(t *testing.T) {
		params := params.CreateNetworkParams{
			Name:              "Network With Empty Fields",
			Type:              "evm",
			NetworkType:       "ethereum",
			CircleNetworkType: "ETH",
			ChainID:           1,
			IsTestnet:         false,
			Active:            true,
			BlockExplorerURL:  "", // Empty string
			LogoURL:           "", // Empty string
			DisplayName:       "", // Empty string
			ChainNamespace:    "", // Empty string
		}

		expectedNetwork := db.Network{
			ID:      uuid.New(),
			Name:    "Network With Empty Fields",
			Type:    "evm",
			ChainID: 1,
			Active:  true,
		}

		mockQuerier.EXPECT().CreateNetwork(ctx, gomock.Any()).Return(expectedNetwork, nil)

		network, err := service.CreateNetwork(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, network)
	})

	t.Run("gas config with nil priority levels is handled correctly", func(t *testing.T) {
		params := params.CreateNetworkParams{
			Name:              "Network With Partial Gas Config",
			Type:              "evm",
			NetworkType:       "ethereum",
			CircleNetworkType: "ETH",
			ChainID:           1,
			IsTestnet:         false,
			Active:            true,
			GasConfig: &params.CreateGasConfigParams{
				BaseFeeMultiplier:     1.5,
				PriorityFeeMultiplier: 2.0,
				SupportsEIP1559:       true,
				GasPriorityLevels:     nil, // Nil priority levels
			},
		}

		expectedNetwork := db.Network{
			ID:      uuid.New(),
			Name:    "Network With Partial Gas Config",
			Type:    "evm",
			ChainID: 1,
			Active:  true,
		}

		mockQuerier.EXPECT().CreateNetwork(ctx, gomock.Any()).Return(expectedNetwork, nil)

		network, err := service.CreateNetwork(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, network)
	})
}

// Test helper functions
func TestNetworkService_HelperFunctions(t *testing.T) {
	t.Run("nullableString handles empty string correctly", func(t *testing.T) {
		// This tests the private helper function indirectly through service usage
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mocks.NewMockQuerier(ctrl)
		service := services.NewNetworkService(mockQuerier)
		ctx := context.Background()

		params := params.CreateNetworkParams{
			Name:              "Test Network",
			Type:              "evm",
			NetworkType:       "ethereum",
			CircleNetworkType: "ETH",
			ChainID:           1,
			IsTestnet:         false,
			Active:            true,
			BlockExplorerURL:  "", // This should result in a pgtype.Text{Valid: false}
		}

		expectedNetwork := db.Network{
			ID:      uuid.New(),
			Name:    "Test Network",
			Type:    "evm",
			ChainID: 1,
			Active:  true,
		}

		// The mock should be called with a CreateNetworkParams that has BlockExplorerUrl.Valid = false
		mockQuerier.EXPECT().CreateNetwork(ctx, gomock.Any()).DoAndReturn(
			func(ctx context.Context, params db.CreateNetworkParams) (db.Network, error) {
				assert.False(t, params.BlockExplorerUrl.Valid)
				return expectedNetwork, nil
			})

		network, err := service.CreateNetwork(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, network)
	})

	t.Run("nullableString handles non-empty string correctly", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mocks.NewMockQuerier(ctrl)
		service := services.NewNetworkService(mockQuerier)
		ctx := context.Background()

		params := params.CreateNetworkParams{
			Name:              "Test Network",
			Type:              "evm",
			NetworkType:       "ethereum",
			CircleNetworkType: "ETH",
			ChainID:           1,
			IsTestnet:         false,
			Active:            true,
			BlockExplorerURL:  "https://etherscan.io",
		}

		expectedNetwork := db.Network{
			ID:      uuid.New(),
			Name:    "Test Network",
			Type:    "evm",
			ChainID: 1,
			Active:  true,
		}

		// The mock should be called with a CreateNetworkParams that has BlockExplorerUrl.Valid = true
		mockQuerier.EXPECT().CreateNetwork(ctx, gomock.Any()).DoAndReturn(
			func(ctx context.Context, params db.CreateNetworkParams) (db.Network, error) {
				assert.True(t, params.BlockExplorerUrl.Valid)
				assert.Equal(t, "https://etherscan.io", params.BlockExplorerUrl.String)
				return expectedNetwork, nil
			})

		network, err := service.CreateNetwork(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, network)
	})
}

// Helper functions to create pointers for update parameters
func networkStringPtr(s string) *string {
	return &s
}

func networkBoolPtr(b bool) *bool {
	return &b
}

func networkInt32Ptr(i int32) *int32 {
	return &i
}

func networkFloatPtr(f float64) *float64 {
	return &f
}
