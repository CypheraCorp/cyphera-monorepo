package services_test

import (
	"context"
	"encoding/json"
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

func TestWalletService_CreateWallet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWalletService(mockQuerier)
	ctx := context.Background()

	workspaceID := uuid.New()
	networkID := uuid.New()

	tests := []struct {
		name        string
		params      params.CreateWalletParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully creates wallet",
			params: params.CreateWalletParams{
				WorkspaceID:   workspaceID,
				WalletType:    "wallet",
				WalletAddress: "0x123456789abcdef",
				NetworkType:   "ethereum",
				NetworkID:     &networkID,
				Nickname:      "My Wallet",
				ENS:           "test.eth",
				IsPrimary:     true,
				Verified:      true,
				Metadata:      map[string]interface{}{"key": "value"},
			},
			setupMocks: func() {
				expectedWallet := db.Wallet{
					ID:            uuid.New(),
					WorkspaceID:   workspaceID,
					WalletType:    "wallet",
					WalletAddress: "0x123456789abcdef",
					NetworkType:   "ethereum",
				}
				mockQuerier.EXPECT().CreateWallet(ctx, gomock.Any()).Return(expectedWallet, nil)
			},
			wantErr: false,
		},
		{
			name: "fails with empty wallet type",
			params: params.CreateWalletParams{
				WorkspaceID:   workspaceID,
				WalletAddress: "0x123456789abcdef",
				NetworkType:   "ethereum",
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "wallet type is required",
		},
		{
			name: "fails with empty wallet address",
			params: params.CreateWalletParams{
				WorkspaceID: workspaceID,
				WalletType:  "wallet",
				NetworkType: "ethereum",
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "wallet address is required",
		},
		{
			name: "fails with empty network type",
			params: params.CreateWalletParams{
				WorkspaceID:   workspaceID,
				WalletType:    "wallet",
				WalletAddress: "0x123456789abcdef",
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "network type is required",
		},
		{
			name: "fails with invalid wallet type",
			params: params.CreateWalletParams{
				WorkspaceID:   workspaceID,
				WalletType:    "invalid_type",
				WalletAddress: "0x123456789abcdef",
				NetworkType:   "ethereum",
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "invalid wallet type: invalid_type",
		},
		{
			name: "fails with invalid metadata",
			params: params.CreateWalletParams{
				WorkspaceID:   workspaceID,
				WalletType:    "wallet",
				WalletAddress: "0x123456789abcdef",
				NetworkType:   "ethereum",
				Metadata:      map[string]interface{}{"key": make(chan int)}, // Invalid JSON
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "failed to marshal metadata",
		},
		{
			name: "handles database error",
			params: params.CreateWalletParams{
				WorkspaceID:   workspaceID,
				WalletType:    "wallet",
				WalletAddress: "0x123456789abcdef",
				NetworkType:   "ethereum",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().CreateWallet(ctx, gomock.Any()).Return(db.Wallet{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to create wallet",
		},
		{
			name: "creates circle wallet successfully",
			params: params.CreateWalletParams{
				WorkspaceID:   workspaceID,
				WalletType:    "circle_wallet",
				WalletAddress: "0x987654321fedcba",
				NetworkType:   "polygon",
				IsPrimary:     false,
				Verified:      false,
			},
			setupMocks: func() {
				expectedWallet := db.Wallet{
					ID:            uuid.New(),
					WorkspaceID:   workspaceID,
					WalletType:    "circle_wallet",
					WalletAddress: "0x987654321fedcba",
					NetworkType:   "polygon",
				}
				mockQuerier.EXPECT().CreateWallet(ctx, gomock.Any()).Return(expectedWallet, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			wallet, err := service.CreateWallet(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, wallet)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, wallet)
				assert.Equal(t, tt.params.WorkspaceID, wallet.WorkspaceID)
				assert.Equal(t, tt.params.WalletType, wallet.WalletType)
				assert.Equal(t, tt.params.WalletAddress, wallet.WalletAddress)
			}
		})
	}
}

func TestWalletService_GetWallet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWalletService(mockQuerier)
	ctx := context.Background()

	walletID := uuid.New()
	workspaceID := uuid.New()
	expectedWallet := db.Wallet{
		ID:            walletID,
		WorkspaceID:   workspaceID,
		WalletType:    "wallet",
		WalletAddress: "0x123456789abcdef",
		NetworkType:   "ethereum",
	}

	tests := []struct {
		name        string
		walletID    uuid.UUID
		workspaceID uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "successfully gets wallet",
			walletID:    walletID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(expectedWallet, nil)
			},
			wantErr: false,
		},
		{
			name:        "wallet not found",
			walletID:    walletID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(db.Wallet{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "wallet not found",
		},
		{
			name:        "database error",
			walletID:    walletID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(db.Wallet{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve wallet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			wallet, err := service.GetWallet(ctx, tt.walletID, tt.workspaceID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, wallet)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, wallet)
				assert.Equal(t, tt.walletID, wallet.ID)
				assert.Equal(t, tt.workspaceID, wallet.WorkspaceID)
			}
		})
	}
}

func TestWalletService_GetWalletWithCircleData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWalletService(mockQuerier)
	ctx := context.Background()

	walletID := uuid.New()
	workspaceID := uuid.New()

	tests := []struct {
		name        string
		walletID    uuid.UUID
		workspaceID uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
		hasCircle   bool
	}{
		{
			name:        "successfully gets wallet with Circle data",
			walletID:    walletID,
			workspaceID: workspaceID,
			setupMocks: func() {
				expectedResult := db.GetWalletWithCircleDataByIDRow{
					ID:             walletID,
					WorkspaceID:    workspaceID,
					WalletType:     "circle_wallet",
					WalletAddress:  "0x123456789abcdef",
					NetworkType:    "ethereum",
					CircleWalletID: pgtype.UUID{Bytes: uuid.New(), Valid: true},
					CircleUserID:   pgtype.UUID{Bytes: uuid.New(), Valid: true},
					CircleID:       pgtype.Text{String: "circle-123", Valid: true},
					ChainID:        pgtype.Int4{Int32: 1, Valid: true},
					CircleState:    pgtype.Text{String: "LIVE", Valid: true},
				}
				mockQuerier.EXPECT().GetWalletWithCircleDataByID(ctx, db.GetWalletWithCircleDataByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(expectedResult, nil)
			},
			wantErr:   false,
			hasCircle: true,
		},
		{
			name:        "successfully gets wallet without Circle data",
			walletID:    walletID,
			workspaceID: workspaceID,
			setupMocks: func() {
				expectedResult := db.GetWalletWithCircleDataByIDRow{
					ID:             walletID,
					WorkspaceID:    workspaceID,
					WalletType:     "wallet",
					WalletAddress:  "0x123456789abcdef",
					NetworkType:    "ethereum",
					CircleWalletID: pgtype.UUID{Valid: false},
					CircleUserID:   pgtype.UUID{Valid: false},
				}
				mockQuerier.EXPECT().GetWalletWithCircleDataByID(ctx, db.GetWalletWithCircleDataByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(expectedResult, nil)
			},
			wantErr:   false,
			hasCircle: false,
		},
		{
			name:        "wallet not found",
			walletID:    walletID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletWithCircleDataByID(ctx, db.GetWalletWithCircleDataByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(db.GetWalletWithCircleDataByIDRow{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "wallet not found",
		},
		{
			name:        "database error",
			walletID:    walletID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletWithCircleDataByID(ctx, db.GetWalletWithCircleDataByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(db.GetWalletWithCircleDataByIDRow{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve wallet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			walletData, err := service.GetWalletWithCircleData(ctx, tt.walletID, tt.workspaceID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, walletData)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, walletData)
				assert.Equal(t, tt.walletID, walletData.Wallet.ID)
				assert.Equal(t, tt.workspaceID, walletData.Wallet.WorkspaceID)

				if tt.hasCircle {
					assert.NotNil(t, walletData.CircleData)
					assert.NotEmpty(t, walletData.CircleData.CircleWalletID)
					assert.NotEmpty(t, walletData.CircleData.CircleUserID)
					assert.Equal(t, int32(1), walletData.CircleData.ChainID)
					assert.Equal(t, "LIVE", walletData.CircleData.State)
				} else {
					assert.Nil(t, walletData.CircleData)
				}
			}
		})
	}
}

func TestWalletService_ListWalletsByWorkspace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWalletService(mockQuerier)
	ctx := context.Background()

	workspaceID := uuid.New()
	expectedWallets := []db.Wallet{
		{ID: uuid.New(), WorkspaceID: workspaceID, WalletType: "wallet"},
		{ID: uuid.New(), WorkspaceID: workspaceID, WalletType: "circle_wallet"},
	}

	tests := []struct {
		name        string
		workspaceID uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
		wantCount   int
	}{
		{
			name:        "successfully lists wallets",
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListWalletsByWorkspaceID(ctx, workspaceID).Return(expectedWallets, nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:        "returns empty list",
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListWalletsByWorkspaceID(ctx, workspaceID).Return([]db.Wallet{}, nil)
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:        "database error",
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListWalletsByWorkspaceID(ctx, workspaceID).Return(nil, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve wallets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			wallets, err := service.ListWalletsByWorkspace(ctx, tt.workspaceID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, wallets)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, wallets, tt.wantCount)
			}
		})
	}
}

func TestWalletService_ListWalletsByType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWalletService(mockQuerier)
	ctx := context.Background()

	workspaceID := uuid.New()
	walletType := "circle_wallet"
	expectedWallets := []db.Wallet{
		{ID: uuid.New(), WorkspaceID: workspaceID, WalletType: walletType},
	}

	tests := []struct {
		name        string
		workspaceID uuid.UUID
		walletType  string
		setupMocks  func()
		wantErr     bool
		errorString string
		wantCount   int
	}{
		{
			name:        "successfully lists wallets by type",
			workspaceID: workspaceID,
			walletType:  walletType,
			setupMocks: func() {
				mockQuerier.EXPECT().ListWalletsByWalletType(ctx, db.ListWalletsByWalletTypeParams{
					WorkspaceID: workspaceID,
					WalletType:  walletType,
				}).Return(expectedWallets, nil)
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:        "returns empty list for type",
			workspaceID: workspaceID,
			walletType:  "nonexistent_type",
			setupMocks: func() {
				mockQuerier.EXPECT().ListWalletsByWalletType(ctx, db.ListWalletsByWalletTypeParams{
					WorkspaceID: workspaceID,
					WalletType:  "nonexistent_type",
				}).Return([]db.Wallet{}, nil)
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:        "database error",
			workspaceID: workspaceID,
			walletType:  walletType,
			setupMocks: func() {
				mockQuerier.EXPECT().ListWalletsByWalletType(ctx, db.ListWalletsByWalletTypeParams{
					WorkspaceID: workspaceID,
					WalletType:  walletType,
				}).Return(nil, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve wallets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			wallets, err := service.ListWalletsByType(ctx, tt.workspaceID, tt.walletType)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, wallets)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, wallets, tt.wantCount)
			}
		})
	}
}

func TestWalletService_UpdateWallet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWalletService(mockQuerier)
	ctx := context.Background()

	walletID := uuid.New()
	workspaceID := uuid.New()
	existingWallet := db.Wallet{
		ID:            walletID,
		WorkspaceID:   workspaceID,
		WalletType:    "wallet",
		WalletAddress: "0x123456789abcdef",
		NetworkType:   "ethereum",
	}

	tests := []struct {
		name        string
		workspaceID uuid.UUID
		params      params.UpdateWalletParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "successfully updates wallet",
			workspaceID: workspaceID,
			params: params.UpdateWalletParams{
				ID:       walletID,
				Nickname: walletStringPtr("Updated Wallet"),
				ENS:      walletStringPtr("updated.eth"),
			},
			setupMocks: func() {
				// GetWallet call for validation
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(existingWallet, nil)

				// UpdateWallet call
				updatedWallet := existingWallet
				updatedWallet.Nickname = pgtype.Text{String: "Updated Wallet", Valid: true}
				mockQuerier.EXPECT().UpdateWallet(ctx, gomock.Any()).Return(updatedWallet, nil)
			},
			wantErr: false,
		},
		{
			name:        "wallet not found during validation",
			workspaceID: workspaceID,
			params: params.UpdateWalletParams{
				ID:       walletID,
				Nickname: walletStringPtr("Updated Wallet"),
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(db.Wallet{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "wallet not found",
		},
		{
			name:        "invalid metadata format",
			workspaceID: workspaceID,
			params: params.UpdateWalletParams{
				ID:       walletID,
				Metadata: map[string]interface{}{"key": make(chan int)},
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(existingWallet, nil)
			},
			wantErr:     true,
			errorString: "failed to marshal metadata",
		},
		{
			name:        "database update error",
			workspaceID: workspaceID,
			params: params.UpdateWalletParams{
				ID:       walletID,
				Nickname: walletStringPtr("Updated Wallet"),
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(existingWallet, nil)
				mockQuerier.EXPECT().UpdateWallet(ctx, gomock.Any()).Return(db.Wallet{}, errors.New("update error"))
			},
			wantErr:     true,
			errorString: "failed to update wallet",
		},
		{
			name:        "updates with new metadata",
			workspaceID: workspaceID,
			params: params.UpdateWalletParams{
				ID:       walletID,
				Metadata: map[string]interface{}{"new": "metadata"},
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(existingWallet, nil)
				mockQuerier.EXPECT().UpdateWallet(ctx, gomock.Any()).Return(existingWallet, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			wallet, err := service.UpdateWallet(ctx, tt.workspaceID, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, wallet)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, wallet)
				assert.Equal(t, tt.params.ID, wallet.ID)
			}
		})
	}
}

func TestWalletService_DeleteWallet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWalletService(mockQuerier)
	ctx := context.Background()

	walletID := uuid.New()
	workspaceID := uuid.New()
	existingWallet := db.Wallet{
		ID:          walletID,
		WorkspaceID: workspaceID,
		WalletType:  "wallet",
	}

	tests := []struct {
		name        string
		walletID    uuid.UUID
		workspaceID uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "successfully deletes wallet",
			walletID:    walletID,
			workspaceID: workspaceID,
			setupMocks: func() {
				// GetWallet call for validation
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(existingWallet, nil)
				// Check for active products
				mockQuerier.EXPECT().GetActiveProductsByWalletID(ctx, walletID).Return([]db.Product{}, nil)
				// Soft delete
				mockQuerier.EXPECT().SoftDeleteWallet(ctx, walletID).Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "wallet not found",
			walletID:    walletID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(db.Wallet{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "wallet not found",
		},
		{
			name:        "wallet used by active products",
			walletID:    walletID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(existingWallet, nil)
				// Return active products
				activeProducts := []db.Product{
					{ID: uuid.New(), Name: "Product 1"},
					{ID: uuid.New(), Name: "Product 2"},
				}
				mockQuerier.EXPECT().GetActiveProductsByWalletID(ctx, walletID).Return(activeProducts, nil)
			},
			wantErr:     true,
			errorString: "cannot delete wallet: used by 2 active product(s)",
		},
		{
			name:        "error checking product usage",
			walletID:    walletID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(existingWallet, nil)
				mockQuerier.EXPECT().GetActiveProductsByWalletID(ctx, walletID).Return(nil, errors.New("product check error"))
			},
			wantErr:     true,
			errorString: "failed to check product usage",
		},
		{
			name:        "database delete error",
			walletID:    walletID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(existingWallet, nil)
				mockQuerier.EXPECT().GetActiveProductsByWalletID(ctx, walletID).Return([]db.Product{}, nil)
				mockQuerier.EXPECT().SoftDeleteWallet(ctx, walletID).Return(errors.New("delete error"))
			},
			wantErr:     true,
			errorString: "failed to delete wallet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.DeleteWallet(ctx, tt.walletID, tt.workspaceID)

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

func TestWalletService_CreateWalletsForAllNetworks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWalletService(mockQuerier)
	ctx := context.Background()

	workspaceID := uuid.New()
	network1ID := uuid.New()
	network2ID := uuid.New()

	tests := []struct {
		name        string
		params      params.CreateWalletParams
		setupMocks  func()
		wantErr     bool
		errorString string
		wantCount   int
	}{
		{
			name: "successfully creates wallets for all networks",
			params: params.CreateWalletParams{
				WorkspaceID:   workspaceID,
				WalletType:    "wallet",
				WalletAddress: "0x123456789abcdef",
			},
			setupMocks: func() {
				// List active networks
				networks := []db.Network{
					{ID: network1ID, Name: "Ethereum", NetworkType: "ethereum", Active: true},
					{ID: network2ID, Name: "Polygon", NetworkType: "polygon", Active: true},
				}
				mockQuerier.EXPECT().ListNetworks(ctx, db.ListNetworksParams{
					IsActive: pgtype.Bool{Bool: true, Valid: true},
				}).Return(networks, nil)

				// Create wallet for each network
				wallet1 := db.Wallet{ID: uuid.New(), WorkspaceID: workspaceID, NetworkType: "ethereum"}
				wallet2 := db.Wallet{ID: uuid.New(), WorkspaceID: workspaceID, NetworkType: "polygon"}

				mockQuerier.EXPECT().CreateWallet(ctx, gomock.Any()).Return(wallet1, nil)
				mockQuerier.EXPECT().CreateWallet(ctx, gomock.Any()).Return(wallet2, nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "no active networks found",
			params: params.CreateWalletParams{
				WorkspaceID:   workspaceID,
				WalletType:    "wallet",
				WalletAddress: "0x123456789abcdef",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().ListNetworks(ctx, db.ListNetworksParams{
					IsActive: pgtype.Bool{Bool: true, Valid: true},
				}).Return([]db.Network{}, nil)
			},
			wantErr:     true,
			errorString: "no active networks found",
		},
		{
			name: "error retrieving networks",
			params: params.CreateWalletParams{
				WorkspaceID:   workspaceID,
				WalletType:    "wallet",
				WalletAddress: "0x123456789abcdef",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().ListNetworks(ctx, db.ListNetworksParams{
					IsActive: pgtype.Bool{Bool: true, Valid: true},
				}).Return(nil, errors.New("network error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve active networks",
		},
		{
			name: "error creating wallet for one network",
			params: params.CreateWalletParams{
				WorkspaceID:   workspaceID,
				WalletType:    "wallet",
				WalletAddress: "0x123456789abcdef",
			},
			setupMocks: func() {
				networks := []db.Network{
					{ID: network1ID, Name: "Ethereum", NetworkType: "ethereum", Active: true},
				}
				mockQuerier.EXPECT().ListNetworks(ctx, db.ListNetworksParams{
					IsActive: pgtype.Bool{Bool: true, Valid: true},
				}).Return(networks, nil)

				mockQuerier.EXPECT().CreateWallet(ctx, gomock.Any()).Return(db.Wallet{}, errors.New("wallet creation error"))
			},
			wantErr:     true,
			errorString: "failed to create wallet for network Ethereum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			wallets, err := service.CreateWalletsForAllNetworks(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, wallets)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, wallets, tt.wantCount)
			}
		})
	}
}

func TestWalletService_ValidateWalletAccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWalletService(mockQuerier)
	ctx := context.Background()

	walletID := uuid.New()
	workspaceID := uuid.New()
	otherWorkspaceID := uuid.New()

	validWallet := db.Wallet{
		ID:          walletID,
		WorkspaceID: workspaceID,
	}

	invalidWallet := db.Wallet{
		ID:          walletID,
		WorkspaceID: otherWorkspaceID,
	}

	tests := []struct {
		name        string
		walletID    uuid.UUID
		workspaceID uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "valid access",
			walletID:    walletID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(validWallet, nil)
			},
			wantErr: false,
		},
		{
			name:        "wallet not found",
			walletID:    walletID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(db.Wallet{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "wallet not found",
		},
		{
			name:        "wallet belongs to different workspace",
			walletID:    walletID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(invalidWallet, nil)
			},
			wantErr:     true,
			errorString: "wallet does not belong to this workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.ValidateWalletAccess(ctx, tt.walletID, tt.workspaceID)

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

func TestWalletService_UpdateWalletLastUsed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWalletService(mockQuerier)
	ctx := context.Background()

	walletID := uuid.New()

	tests := []struct {
		name        string
		walletID    uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:     "successfully updates last used",
			walletID: walletID,
			setupMocks: func() {
				mockQuerier.EXPECT().UpdateWalletLastUsed(ctx, walletID).Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "database error",
			walletID: walletID,
			setupMocks: func() {
				mockQuerier.EXPECT().UpdateWalletLastUsed(ctx, walletID).Return(errors.New("update error"))
			},
			wantErr:     true,
			errorString: "failed to update wallet last used",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.UpdateWalletLastUsed(ctx, tt.walletID)

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

// Test edge cases and validation scenarios
func TestWalletService_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWalletService(mockQuerier)
	ctx := context.Background()

	t.Run("nil metadata is handled correctly", func(t *testing.T) {
		params := params.CreateWalletParams{
			WorkspaceID:   uuid.New(),
			WalletType:    "wallet",
			WalletAddress: "0x123456789abcdef",
			NetworkType:   "ethereum",
			Metadata:      nil,
		}

		expectedWallet := db.Wallet{
			ID:            uuid.New(),
			WorkspaceID:   params.WorkspaceID,
			WalletType:    params.WalletType,
			WalletAddress: params.WalletAddress,
		}

		mockQuerier.EXPECT().CreateWallet(ctx, gomock.Any()).Return(expectedWallet, nil)

		wallet, err := service.CreateWallet(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, wallet)
	})

	t.Run("empty metadata map is handled correctly", func(t *testing.T) {
		params := params.CreateWalletParams{
			WorkspaceID:   uuid.New(),
			WalletType:    "wallet",
			WalletAddress: "0x123456789abcdef",
			NetworkType:   "ethereum",
			Metadata:      map[string]interface{}{},
		}

		expectedWallet := db.Wallet{
			ID:            uuid.New(),
			WorkspaceID:   params.WorkspaceID,
			WalletType:    params.WalletType,
			WalletAddress: params.WalletAddress,
		}

		mockQuerier.EXPECT().CreateWallet(ctx, gomock.Any()).Return(expectedWallet, nil)

		wallet, err := service.CreateWallet(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, wallet)
	})

	t.Run("update with nil metadata preserves existing", func(t *testing.T) {
		walletID := uuid.New()
		workspaceID := uuid.New()
		existingMetadata, _ := json.Marshal(map[string]interface{}{"existing": "data"})
		existingWallet := db.Wallet{
			ID:          walletID,
			WorkspaceID: workspaceID,
			Metadata:    existingMetadata,
		}

		params := params.UpdateWalletParams{
			ID:       walletID,
			Nickname: walletStringPtr("Updated Nickname"),
			Metadata: nil, // Should preserve existing metadata
		}

		mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
			ID:          walletID,
			WorkspaceID: workspaceID,
		}).Return(existingWallet, nil)
		mockQuerier.EXPECT().UpdateWallet(ctx, gomock.Any()).Return(existingWallet, nil)

		wallet, err := service.UpdateWallet(ctx, workspaceID, params)
		assert.NoError(t, err)
		assert.NotNil(t, wallet)
	})

	t.Run("empty string fields are handled correctly", func(t *testing.T) {
		params := params.CreateWalletParams{
			WorkspaceID:   uuid.New(),
			WalletType:    "wallet",
			WalletAddress: "0x123456789abcdef",
			NetworkType:   "ethereum",
			Nickname:      "",
			ENS:           "",
		}

		expectedWallet := db.Wallet{
			ID:            uuid.New(),
			WorkspaceID:   params.WorkspaceID,
			WalletType:    params.WalletType,
			WalletAddress: params.WalletAddress,
		}

		mockQuerier.EXPECT().CreateWallet(ctx, gomock.Any()).Return(expectedWallet, nil)

		wallet, err := service.CreateWallet(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, wallet)
	})
}

// Helper function to create string pointer
func walletStringPtr(s string) *string {
	return &s
}
