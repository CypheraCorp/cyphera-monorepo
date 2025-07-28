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

func TestProductService_CreateProduct(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewProductService(mockQuerier)
	ctx := context.Background()

	workspaceID := uuid.New()
	walletID := uuid.New()
	productID := uuid.New()
	priceID := uuid.New()
	networkID := uuid.New()
	tokenID := uuid.New()

	metadata := json.RawMessage(`{"key": "value"}`)
	priceMetadata := json.RawMessage(`{"tier": "premium"}`)

	validWallet := db.Wallet{
		ID:          walletID,
		WorkspaceID: workspaceID,
	}

	tests := []struct {
		name        string
		params      params.CreateProductParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully creates product with all fields",
			params: params.CreateProductParams{
				WorkspaceID: workspaceID,
				WalletID:    walletID,
				Name:        "Test Product",
				Description: "Test Description",
				ImageURL:    "https://example.com/image.jpg",
				URL:         "https://example.com/product",
				Active:      true,
				Metadata:    metadata,
				Prices: []params.CreatePriceParams{
					{
						Active:              true,
						Type:                "one_off",
						Nickname:            "Standard Price",
						Currency:            "USD",
						UnitAmountInPennies: 1000,
						Metadata:            priceMetadata,
					},
				},
				ProductTokens: []params.CreateProductTokenParams{
					{
						NetworkID: networkID,
						TokenID:   tokenID,
						Active:    true,
					},
				},
			},
			setupMocks: func() {
				// Validate wallet ownership
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(validWallet, nil)

				// Create product
				expectedProduct := db.Product{
					ID:          productID,
					WorkspaceID: workspaceID,
					WalletID:    walletID,
					Name:        "Test Product",
					Description: pgtype.Text{String: "Test Description", Valid: true},
					ImageUrl:    pgtype.Text{String: "https://example.com/image.jpg", Valid: true},
					Url:         pgtype.Text{String: "https://example.com/product", Valid: true},
					Active:      true,
					Metadata:    metadata,
				}
				mockQuerier.EXPECT().CreateProduct(ctx, gomock.Any()).Return(expectedProduct, nil)

				// Create price
				expectedPrice := db.Price{
					ID:                  priceID,
					ProductID:           productID,
					Active:              true,
					Type:                "one_time",
					Nickname:            pgtype.Text{String: "Standard Price", Valid: true},
					Currency:            "USD",
					UnitAmountInPennies: 1000,
					IntervalType:        "month",
					TermLength:          12,
					Metadata:            priceMetadata,
				}
				mockQuerier.EXPECT().CreatePrice(ctx, gomock.Any()).Return(expectedPrice, nil)

				// Create product token
				expectedProductToken := db.ProductsToken{
					ID:        uuid.New(),
					ProductID: productID,
					NetworkID: networkID,
					TokenID:   tokenID,
					Active:    true,
				}
				mockQuerier.EXPECT().CreateProductToken(ctx, gomock.Any()).Return(expectedProductToken, nil)
			},
			wantErr: false,
		},
		{
			name: "successfully creates product with minimal fields",
			params: params.CreateProductParams{
				WorkspaceID: workspaceID,
				WalletID:    walletID,
				Name:        "Minimal Product",
				Active:      false,
			},
			setupMocks: func() {
				// Validate wallet ownership
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(validWallet, nil)

				// Create product
				expectedProduct := db.Product{
					ID:          productID,
					WorkspaceID: workspaceID,
					WalletID:    walletID,
					Name:        "Minimal Product",
					Active:      false,
				}
				mockQuerier.EXPECT().CreateProduct(ctx, gomock.Any()).Return(expectedProduct, nil)
			},
			wantErr: false,
		},
		{
			name: "fails with empty product name",
			params: params.CreateProductParams{
				WorkspaceID: workspaceID,
				WalletID:    walletID,
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "product name is required",
		},
		{
			name: "fails with empty workspace ID",
			params: params.CreateProductParams{
				WalletID: walletID,
				Name:     "Test Product",
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "workspace ID is required",
		},
		{
			name: "fails with empty wallet ID",
			params: params.CreateProductParams{
				WorkspaceID: workspaceID,
				Name:        "Test Product",
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "wallet ID is required",
		},
		{
			name: "fails with invalid product name (too long)",
			params: params.CreateProductParams{
				WorkspaceID: workspaceID,
				WalletID:    walletID,
				Name:        string(make([]byte, 300)), // Too long name
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "name must be less than 255 characters",
		},
		{
			name: "fails with wallet not found",
			params: params.CreateProductParams{
				WorkspaceID: workspaceID,
				WalletID:    walletID,
				Name:        "Test Product",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(db.Wallet{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "wallet not found or not accessible",
		},
		{
			name: "fails with wallet not belonging to workspace",
			params: params.CreateProductParams{
				WorkspaceID: workspaceID,
				WalletID:    walletID,
				Name:        "Test Product",
			},
			setupMocks: func() {
				invalidWallet := db.Wallet{
					ID:          walletID,
					WorkspaceID: uuid.New(), // Different workspace
				}
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(invalidWallet, nil)
			},
			wantErr:     true,
			errorString: "wallet does not belong to workspace",
		},
		{
			name: "fails with invalid price validation",
			params: params.CreateProductParams{
				WorkspaceID: workspaceID,
				WalletID:    walletID,
				Name:        "Test Product",
				Prices: []params.CreatePriceParams{
					{
						Type:                "invalid_type",
						UnitAmountInPennies: -100, // Negative price
					},
				},
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(validWallet, nil)
			},
			wantErr:     true,
			errorString: "price 1 validation failed",
		},
		{
			name: "handles database error during product creation",
			params: params.CreateProductParams{
				WorkspaceID: workspaceID,
				WalletID:    walletID,
				Name:        "Test Product",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(validWallet, nil)
				mockQuerier.EXPECT().CreateProduct(ctx, gomock.Any()).Return(db.Product{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to create product",
		},
		{
			name: "handles database error during price creation",
			params: params.CreateProductParams{
				WorkspaceID: workspaceID,
				WalletID:    walletID,
				Name:        "Test Product",
				Prices: []params.CreatePriceParams{
					{
						Active:              true,
						Type:                "one_off",
						Currency:            "USD",
						UnitAmountInPennies: 1000,
					},
				},
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(validWallet, nil)

				expectedProduct := db.Product{
					ID:          productID,
					WorkspaceID: workspaceID,
					WalletID:    walletID,
					Name:        "Test Product",
				}
				mockQuerier.EXPECT().CreateProduct(ctx, gomock.Any()).Return(expectedProduct, nil)
				mockQuerier.EXPECT().CreatePrice(ctx, gomock.Any()).Return(db.Price{}, errors.New("price creation error"))
			},
			wantErr:     true,
			errorString: "failed to create price 1",
		},
		{
			name: "handles database error during product token creation",
			params: params.CreateProductParams{
				WorkspaceID: workspaceID,
				WalletID:    walletID,
				Name:        "Test Product",
				ProductTokens: []params.CreateProductTokenParams{
					{
						NetworkID: networkID,
						TokenID:   tokenID,
						Active:    true,
					},
				},
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(validWallet, nil)

				expectedProduct := db.Product{
					ID:          productID,
					WorkspaceID: workspaceID,
					WalletID:    walletID,
					Name:        "Test Product",
				}
				mockQuerier.EXPECT().CreateProduct(ctx, gomock.Any()).Return(expectedProduct, nil)
				mockQuerier.EXPECT().CreateProductToken(ctx, gomock.Any()).Return(db.ProductsToken{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to create product tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			product, prices, err := service.CreateProduct(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, product)
				assert.Nil(t, prices)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, product)
				assert.NotNil(t, prices)
				assert.Equal(t, tt.params.Name, product.Name)
				assert.Equal(t, tt.params.WorkspaceID, product.WorkspaceID)
				assert.Equal(t, tt.params.WalletID, product.WalletID)
				assert.Equal(t, len(tt.params.Prices), len(prices))
			}
		})
	}
}

func TestProductService_GetProduct(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewProductService(mockQuerier)
	ctx := context.Background()

	productID := uuid.New()
	workspaceID := uuid.New()
	otherWorkspaceID := uuid.New()

	expectedProduct := db.Product{
		ID:          productID,
		WorkspaceID: workspaceID,
		Name:        "Test Product",
	}

	expectedPrices := []db.Price{
		{
			ID:        uuid.New(),
			ProductID: productID,
			Active:    true,
		},
	}

	tests := []struct {
		name        string
		params      params.GetProductParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully gets product with prices",
			params: params.GetProductParams{
				ProductID:   productID,
				WorkspaceID: workspaceID,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(expectedProduct, nil)
				mockQuerier.EXPECT().ListPricesByProduct(ctx, productID).Return(expectedPrices, nil)
			},
			wantErr: false,
		},
		{
			name: "successfully gets product without workspace validation",
			params: params.GetProductParams{
				ProductID: productID,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: uuid.Nil,
				}).Return(expectedProduct, nil)
				mockQuerier.EXPECT().ListPricesByProduct(ctx, productID).Return(expectedPrices, nil)
			},
			wantErr: false,
		},
		{
			name: "fails with empty product ID",
			params: params.GetProductParams{
				WorkspaceID: workspaceID,
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "product ID is required",
		},
		{
			name: "product not found",
			params: params.GetProductParams{
				ProductID:   productID,
				WorkspaceID: workspaceID,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(db.Product{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "product not found",
		},
		{
			name: "product not found in workspace",
			params: params.GetProductParams{
				ProductID:   productID,
				WorkspaceID: otherWorkspaceID,
			},
			setupMocks: func() {
				productInDifferentWorkspace := db.Product{
					ID:          productID,
					WorkspaceID: workspaceID, // Different workspace than requested
					Name:        "Test Product",
				}
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: otherWorkspaceID,
				}).Return(productInDifferentWorkspace, nil)
			},
			wantErr:     true,
			errorString: "product not found in workspace",
		},
		{
			name: "database error getting prices",
			params: params.GetProductParams{
				ProductID:   productID,
				WorkspaceID: workspaceID,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(expectedProduct, nil)
				mockQuerier.EXPECT().ListPricesByProduct(ctx, productID).Return(nil, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve product prices",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			product, prices, err := service.GetProduct(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, product)
				assert.Nil(t, prices)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, product)
				assert.NotNil(t, prices)
				assert.Equal(t, productID, product.ID)
			}
		})
	}
}

func TestProductService_ListProducts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewProductService(mockQuerier)
	ctx := context.Background()

	workspaceID := uuid.New()
	products := []db.Product{
		{ID: uuid.New(), Name: "Product 1", WorkspaceID: workspaceID},
		{ID: uuid.New(), Name: "Product 2", WorkspaceID: workspaceID},
	}

	tests := []struct {
		name        string
		params      params.ListProductsParams
		setupMocks  func()
		wantErr     bool
		errorString string
		wantCount   int
		wantHasMore bool
	}{
		{
			name: "successfully lists products with default pagination",
			params: params.ListProductsParams{
				WorkspaceID: workspaceID,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().ListProductsWithPagination(ctx, db.ListProductsWithPaginationParams{
					WorkspaceID: workspaceID,
					Limit:       10, // Default limit
					Offset:      0,
				}).Return(products, nil)
				mockQuerier.EXPECT().CountProducts(ctx, workspaceID).Return(int64(25), nil)
			},
			wantErr:     false,
			wantCount:   2,
			wantHasMore: true, // 10 offset + 10 limit < 25 total
		},
		{
			name: "successfully lists products with custom pagination",
			params: params.ListProductsParams{
				WorkspaceID: workspaceID,
				Limit:       5,
				Offset:      10,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().ListProductsWithPagination(ctx, db.ListProductsWithPaginationParams{
					WorkspaceID: workspaceID,
					Limit:       5,
					Offset:      10,
				}).Return(products, nil)
				mockQuerier.EXPECT().CountProducts(ctx, workspaceID).Return(int64(12), nil)
			},
			wantErr:     false,
			wantCount:   2,
			wantHasMore: false, // 10 offset + 5 limit >= 12 total
		},
		{
			name: "handles limit too high (caps at 100)",
			params: params.ListProductsParams{
				WorkspaceID: workspaceID,
				Limit:       200, // Should be capped at 100
			},
			setupMocks: func() {
				mockQuerier.EXPECT().ListProductsWithPagination(ctx, db.ListProductsWithPaginationParams{
					WorkspaceID: workspaceID,
					Limit:       100, // Capped limit
					Offset:      0,
				}).Return(products, nil)
				mockQuerier.EXPECT().CountProducts(ctx, workspaceID).Return(int64(2), nil)
			},
			wantErr:     false,
			wantCount:   2,
			wantHasMore: false,
		},
		{
			name: "fails with empty workspace ID",
			params: params.ListProductsParams{
				Limit:  10,
				Offset: 0,
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "workspace ID is required",
		},
		{
			name: "database error during listing",
			params: params.ListProductsParams{
				WorkspaceID: workspaceID,
				Limit:       10,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().ListProductsWithPagination(ctx, gomock.Any()).Return(nil, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve products",
		},
		{
			name: "database error during counting",
			params: params.ListProductsParams{
				WorkspaceID: workspaceID,
				Limit:       10,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().ListProductsWithPagination(ctx, gomock.Any()).Return(products, nil)
				mockQuerier.EXPECT().CountProducts(ctx, workspaceID).Return(int64(0), errors.New("count error"))
			},
			wantErr:     true,
			errorString: "failed to count products",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.ListProducts(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result.Products, tt.wantCount)
				assert.Equal(t, tt.wantHasMore, result.HasMore)
			}
		})
	}
}

func TestProductService_UpdateProduct(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewProductService(mockQuerier)
	ctx := context.Background()

	productID := uuid.New()
	workspaceID := uuid.New()
	walletID := uuid.New()
	newWalletID := uuid.New()

	existingProduct := db.Product{
		ID:          productID,
		WorkspaceID: workspaceID,
		WalletID:    walletID,
		Name:        "Original Product",
		Description: pgtype.Text{String: "Original Description", Valid: true},
		Active:      true,
	}

	validWallet := db.Wallet{
		ID:          newWalletID,
		WorkspaceID: workspaceID,
	}

	updatedName := "Updated Product"
	updatedDescription := "Updated Description"
	updatedActive := false
	metadata := json.RawMessage(`{"updated": true}`)

	tests := []struct {
		name        string
		params      params.UpdateProductParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully updates product with all fields",
			params: params.UpdateProductParams{
				ProductID:   productID,
				WorkspaceID: workspaceID,
				Name:        &updatedName,
				Description: &updatedDescription,
				Active:      &updatedActive,
				Metadata:    metadata,
				WalletID:    &newWalletID,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(existingProduct, nil)

				// Validate new wallet ownership
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          newWalletID,
					WorkspaceID: workspaceID,
				}).Return(validWallet, nil)

				updatedProduct := existingProduct
				updatedProduct.Name = updatedName
				updatedProduct.Description = pgtype.Text{String: updatedDescription, Valid: true}
				updatedProduct.Active = updatedActive
				updatedProduct.Metadata = metadata
				updatedProduct.WalletID = newWalletID

				mockQuerier.EXPECT().UpdateProduct(ctx, gomock.Any()).Return(updatedProduct, nil)
			},
			wantErr: false,
		},
		{
			name: "successfully updates product with partial fields",
			params: params.UpdateProductParams{
				ProductID:   productID,
				WorkspaceID: workspaceID,
				Name:        &updatedName,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(existingProduct, nil)

				updatedProduct := existingProduct
				updatedProduct.Name = updatedName
				mockQuerier.EXPECT().UpdateProduct(ctx, gomock.Any()).Return(updatedProduct, nil)
			},
			wantErr: false,
		},
		{
			name: "fails with empty product ID",
			params: params.UpdateProductParams{
				WorkspaceID: workspaceID,
				Name:        &updatedName,
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "product ID is required",
		},
		{
			name: "product not found",
			params: params.UpdateProductParams{
				ProductID:   productID,
				WorkspaceID: workspaceID,
				Name:        &updatedName,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(db.Product{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "product not found",
		},
		{
			name: "product not found in workspace",
			params: params.UpdateProductParams{
				ProductID:   productID,
				WorkspaceID: uuid.New(), // Different workspace
				Name:        &updatedName,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, gomock.Any()).Return(existingProduct, nil)
			},
			wantErr:     true,
			errorString: "product not found in workspace",
		},
		{
			name: "fails with invalid product name (too long)",
			params: params.UpdateProductParams{
				ProductID:   productID,
				WorkspaceID: workspaceID,
				Name:        func() *string { s := string(make([]byte, 300)); return &s }(), // Too long name
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(existingProduct, nil)
			},
			wantErr:     true,
			errorString: "name must be less than 255 characters",
		},
		{
			name: "fails with wallet not belonging to workspace",
			params: params.UpdateProductParams{
				ProductID:   productID,
				WorkspaceID: workspaceID,
				WalletID:    &newWalletID,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(existingProduct, nil)

				invalidWallet := db.Wallet{
					ID:          newWalletID,
					WorkspaceID: uuid.New(), // Different workspace
				}
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          newWalletID,
					WorkspaceID: workspaceID,
				}).Return(invalidWallet, nil)
			},
			wantErr:     true,
			errorString: "wallet does not belong to workspace",
		},
		{
			name: "database error during update",
			params: params.UpdateProductParams{
				ProductID:   productID,
				WorkspaceID: workspaceID,
				Name:        &updatedName,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(existingProduct, nil)
				mockQuerier.EXPECT().UpdateProduct(ctx, gomock.Any()).Return(db.Product{}, errors.New("update error"))
			},
			wantErr:     true,
			errorString: "failed to update product",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			product, err := service.UpdateProduct(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, product)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, product)
				if tt.params.Name != nil {
					assert.Equal(t, *tt.params.Name, product.Name)
				}
			}
		})
	}
}

func TestProductService_DeleteProduct(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewProductService(mockQuerier)
	ctx := context.Background()

	productID := uuid.New()
	workspaceID := uuid.New()

	existingProduct := db.Product{
		ID:          productID,
		WorkspaceID: workspaceID,
		Name:        "Test Product",
	}

	tests := []struct {
		name        string
		productID   uuid.UUID
		workspaceID uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "successfully deletes product",
			productID:   productID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(existingProduct, nil)
				mockQuerier.EXPECT().DeleteProduct(ctx, db.DeleteProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "fails with empty product ID",
			productID:   uuid.Nil,
			workspaceID: workspaceID,
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "product ID is required",
		},
		{
			name:        "product not found",
			productID:   productID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(db.Product{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "product not found",
		},
		{
			name:        "product not found in workspace",
			productID:   productID,
			workspaceID: uuid.New(), // Different workspace
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, gomock.Any()).Return(existingProduct, nil)
			},
			wantErr:     true,
			errorString: "product not found in workspace",
		},
		{
			name:        "database error during deletion",
			productID:   productID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(existingProduct, nil)
				mockQuerier.EXPECT().DeleteProduct(ctx, db.DeleteProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(errors.New("delete error"))
			},
			wantErr:     true,
			errorString: "failed to delete product",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.DeleteProduct(ctx, tt.productID, tt.workspaceID)

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

func TestProductService_GetPublicProductByPriceID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewProductService(mockQuerier)
	ctx := context.Background()

	priceID := uuid.New()
	productID := uuid.New()
	workspaceID := uuid.New()
	walletID := uuid.New()
	networkID := uuid.New()
	tokenID := uuid.New()

	validPrice := db.Price{
		ID:        priceID,
		ProductID: productID,
		Active:    true,
	}

	validProduct := db.Product{
		ID:          productID,
		WorkspaceID: workspaceID,
		WalletID:    walletID,
		Name:        "Test Product",
		Active:      true,
	}

	validWallet := db.Wallet{
		ID:            walletID,
		WorkspaceID:   workspaceID,
		WalletAddress: "0x123456789abcdef",
	}

	validWorkspace := db.Workspace{
		ID:        workspaceID,
		Name:      "Test Workspace",
		AccountID: uuid.New(),
	}

	validProductTokens := []db.GetActiveProductTokensByProductRow{
		{
			ID:        uuid.New(),
			ProductID: productID,
			NetworkID: networkID,
			TokenID:   tokenID,
		},
	}

	validToken := db.Token{
		ID:              tokenID,
		ContractAddress: "0xtoken123",
	}

	tests := []struct {
		name        string
		priceID     uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:    "successfully gets public product by price ID",
			priceID: priceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetPrice(ctx, priceID).Return(validPrice, nil)
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(ctx, productID).Return(validProduct, nil)
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(validWallet, nil)
				mockQuerier.EXPECT().GetWorkspace(ctx, workspaceID).Return(validWorkspace, nil)
				mockQuerier.EXPECT().GetActiveProductTokensByProduct(ctx, productID).Return(validProductTokens, nil)
				mockQuerier.EXPECT().GetToken(ctx, tokenID).Return(validToken, nil)
			},
			wantErr: false,
		},
		{
			name:        "fails with empty price ID",
			priceID:     uuid.Nil,
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "price ID is required",
		},
		{
			name:    "price not found",
			priceID: priceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetPrice(ctx, priceID).Return(db.Price{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "price not found",
		},
		{
			name:    "price not active",
			priceID: priceID,
			setupMocks: func() {
				inactivePrice := db.Price{
					ID:        priceID,
					ProductID: productID,
					Active:    false,
				}
				mockQuerier.EXPECT().GetPrice(ctx, priceID).Return(inactivePrice, nil)
			},
			wantErr:     true,
			errorString: "price is not active",
		},
		{
			name:    "product not found",
			priceID: priceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetPrice(ctx, priceID).Return(validPrice, nil)
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(ctx, productID).Return(db.Product{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "product not found for the given price",
		},
		{
			name:    "product not active",
			priceID: priceID,
			setupMocks: func() {
				inactiveProduct := db.Product{
					ID:          productID,
					WorkspaceID: workspaceID,
					WalletID:    walletID,
					Name:        "Test Product",
					Active:      false,
				}
				mockQuerier.EXPECT().GetPrice(ctx, priceID).Return(validPrice, nil)
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(ctx, productID).Return(inactiveProduct, nil)
			},
			wantErr:     true,
			errorString: "product associated with this price is not active",
		},
		{
			name:    "wallet not found",
			priceID: priceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetPrice(ctx, priceID).Return(validPrice, nil)
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(ctx, productID).Return(validProduct, nil)
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(db.Wallet{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "wallet not found for the product",
		},
		{
			name:    "workspace not found",
			priceID: priceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetPrice(ctx, priceID).Return(validPrice, nil)
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(ctx, productID).Return(validProduct, nil)
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(validWallet, nil)
				mockQuerier.EXPECT().GetWorkspace(ctx, workspaceID).Return(db.Workspace{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "workspace not found for the product",
		},
		{
			name:    "failed to retrieve product tokens",
			priceID: priceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetPrice(ctx, priceID).Return(validPrice, nil)
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(ctx, productID).Return(validProduct, nil)
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(validWallet, nil)
				mockQuerier.EXPECT().GetWorkspace(ctx, workspaceID).Return(validWorkspace, nil)
				mockQuerier.EXPECT().GetActiveProductTokensByProduct(ctx, productID).Return(nil, errors.New("token error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve product tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			response, err := service.GetPublicProductByPriceID(ctx, tt.priceID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, response)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, productID.String(), response.ID)
				assert.Equal(t, validProduct.Name, response.Name)
			}
		})
	}
}

// TestProductService_EdgeCases tests various edge cases and boundary conditions
func TestProductService_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewProductService(mockQuerier)
	ctx := context.Background()

	t.Run("nil metadata is handled correctly", func(t *testing.T) {
		workspaceID := uuid.New()
		walletID := uuid.New()
		params := params.CreateProductParams{
			WorkspaceID: workspaceID,
			WalletID:    walletID,
			Name:        "Test Product",
			Metadata:    nil,
		}

		validWallet := db.Wallet{
			ID:          walletID,
			WorkspaceID: workspaceID,
		}

		mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
			ID:          walletID,
			WorkspaceID: workspaceID,
		}).Return(validWallet, nil)

		expectedProduct := db.Product{
			ID:          uuid.New(),
			WorkspaceID: workspaceID,
			WalletID:    walletID,
			Name:        "Test Product",
		}
		mockQuerier.EXPECT().CreateProduct(ctx, gomock.Any()).Return(expectedProduct, nil)

		product, prices, err := service.CreateProduct(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, product)
		assert.NotNil(t, prices)
	})

	t.Run("empty metadata is handled correctly", func(t *testing.T) {
		workspaceID := uuid.New()
		walletID := uuid.New()
		params := params.CreateProductParams{
			WorkspaceID: workspaceID,
			WalletID:    walletID,
			Name:        "Test Product",
			Metadata:    json.RawMessage(`{}`),
		}

		validWallet := db.Wallet{
			ID:          walletID,
			WorkspaceID: workspaceID,
		}

		mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
			ID:          walletID,
			WorkspaceID: workspaceID,
		}).Return(validWallet, nil)

		expectedProduct := db.Product{
			ID:          uuid.New(),
			WorkspaceID: workspaceID,
			WalletID:    walletID,
			Name:        "Test Product",
		}
		mockQuerier.EXPECT().CreateProduct(ctx, gomock.Any()).Return(expectedProduct, nil)

		product, prices, err := service.CreateProduct(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, product)
		assert.NotNil(t, prices)
	})

	t.Run("context cancellation", func(t *testing.T) {
		// Create a cancelled context
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		productID := uuid.New()
		params := params.GetProductParams{
			ProductID:   productID,
			WorkspaceID: uuid.New(),
		}

		mockQuerier.EXPECT().GetProduct(cancelledCtx, gomock.Any()).Return(db.Product{}, context.Canceled)

		_, _, err := service.GetProduct(cancelledCtx, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "product not found")
	})

	t.Run("zero limit gets set to default", func(t *testing.T) {
		workspaceID := uuid.New()
		params := params.ListProductsParams{
			WorkspaceID: workspaceID,
			Limit:       0, // Should default to 10
		}

		mockQuerier.EXPECT().ListProductsWithPagination(ctx, db.ListProductsWithPaginationParams{
			WorkspaceID: workspaceID,
			Limit:       10, // Default applied
			Offset:      0,
		}).Return([]db.Product{}, nil)
		mockQuerier.EXPECT().CountProducts(ctx, workspaceID).Return(int64(0), nil)

		result, err := service.ListProducts(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("negative limit gets set to default", func(t *testing.T) {
		workspaceID := uuid.New()
		params := params.ListProductsParams{
			WorkspaceID: workspaceID,
			Limit:       -5, // Should default to 10
		}

		mockQuerier.EXPECT().ListProductsWithPagination(ctx, db.ListProductsWithPaginationParams{
			WorkspaceID: workspaceID,
			Limit:       10, // Default applied
			Offset:      0,
		}).Return([]db.Product{}, nil)
		mockQuerier.EXPECT().CountProducts(ctx, workspaceID).Return(int64(0), nil)

		result, err := service.ListProducts(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}
