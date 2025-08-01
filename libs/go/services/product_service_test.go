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

// Helper functions for creating pointers
func ptrString(s string) *string {
	return &s
}

func ptrBool(b bool) *bool {
	return &b
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
	networkID := uuid.New()
	tokenID := uuid.New()

	metadata := json.RawMessage(`{"key": "value"}`)

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
				// Embedded price fields
				PriceType:           "one_time",
				Currency:            "USD",
				UnitAmountInPennies: 1000,
				PriceNickname:       "Standard Price",
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

				// Create product with embedded price fields
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
					// Embedded price fields
					PriceType:           "one_time",
					Currency:            "USD",
					UnitAmountInPennies: 1000,
					PriceNickname:       pgtype.Text{String: "Standard Price", Valid: true},
				}
				mockQuerier.EXPECT().CreateProduct(ctx, gomock.Any()).Return(expectedProduct, nil)

				// Create product token
				mockQuerier.EXPECT().CreateProductToken(ctx, gomock.Any()).Return(db.ProductsToken{
					ID:        uuid.New(),
					ProductID: productID,
					NetworkID: networkID,
					TokenID:   tokenID,
					Active:    true,
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "successfully creates product with minimal fields",
			params: params.CreateProductParams{
				WorkspaceID: workspaceID,
				WalletID:    walletID,
				Name:        "Minimal Product",
				Active:      true,
				// Required embedded price fields
				PriceType:           "one_time",
				Currency:            "USD",
				UnitAmountInPennies: 500,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(validWallet, nil)

				expectedProduct := db.Product{
					ID:                  productID,
					WorkspaceID:         workspaceID,
					WalletID:            walletID,
					Name:                "Minimal Product",
					Active:              true,
					PriceType:           "one_time",
					Currency:            "USD",
					UnitAmountInPennies: 500,
				}
				mockQuerier.EXPECT().CreateProduct(ctx, gomock.Any()).Return(expectedProduct, nil)
			},
			wantErr: false,
		},
		{
			name: "successfully creates recurring product",
			params: params.CreateProductParams{
				WorkspaceID: workspaceID,
				WalletID:    walletID,
				Name:        "Subscription Product",
				Active:      true,
				// Recurring price fields
				PriceType:           "recurring",
				Currency:            "USD",
				UnitAmountInPennies: 1999,
				IntervalType:        "month",
				TermLength:          12,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetWalletByID(ctx, db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(validWallet, nil)

				expectedProduct := db.Product{
					ID:                  productID,
					WorkspaceID:         workspaceID,
					WalletID:            walletID,
					Name:                "Subscription Product",
					Active:              true,
					PriceType:           "recurring",
					Currency:            "USD",
					UnitAmountInPennies: 1999,
					IntervalType:        db.NullIntervalType{IntervalType: db.IntervalTypeMonth, Valid: true},
					TermLength:          pgtype.Int4{Int32: 12, Valid: true},
				}
				mockQuerier.EXPECT().CreateProduct(ctx, gomock.Any()).Return(expectedProduct, nil)
			},
			wantErr: false,
		},
		{
			name: "fails with missing product name",
			params: params.CreateProductParams{
				WorkspaceID:         workspaceID,
				WalletID:            walletID,
				Name:                "",
				PriceType:           "one_time",
				Currency:            "USD",
				UnitAmountInPennies: 1000,
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "product name is required",
		},
		{
			name: "fails with missing workspace ID",
			params: params.CreateProductParams{
				WalletID:            walletID,
				Name:                "Test Product",
				PriceType:           "one_time",
				Currency:            "USD",
				UnitAmountInPennies: 1000,
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "workspace ID is required",
		},
		{
			name: "fails with missing wallet ID",
			params: params.CreateProductParams{
				WorkspaceID:         workspaceID,
				Name:                "Test Product",
				PriceType:           "one_time",
				Currency:            "USD",
				UnitAmountInPennies: 1000,
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "wallet ID is required",
		},
		{
			name: "fails with missing price type",
			params: params.CreateProductParams{
				WorkspaceID:         workspaceID,
				WalletID:            walletID,
				Name:                "Test Product",
				Currency:            "USD",
				UnitAmountInPennies: 1000,
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "price type is required",
		},
		{
			name: "fails with missing currency",
			params: params.CreateProductParams{
				WorkspaceID:         workspaceID,
				WalletID:            walletID,
				Name:                "Test Product",
				PriceType:           "one_time",
				UnitAmountInPennies: 1000,
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "currency is required",
		},
		{
			name: "fails with invalid wallet ownership",
			params: params.CreateProductParams{
				WorkspaceID:         workspaceID,
				WalletID:            walletID,
				Name:                "Test Product",
				PriceType:           "one_time",
				Currency:            "USD",
				UnitAmountInPennies: 1000,
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
			name: "handles database error during product creation",
			params: params.CreateProductParams{
				WorkspaceID:         workspaceID,
				WalletID:            walletID,
				Name:                "Test Product",
				PriceType:           "one_time",
				Currency:            "USD",
				UnitAmountInPennies: 1000,
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
			name: "handles database error during product token creation",
			params: params.CreateProductParams{
				WorkspaceID:         workspaceID,
				WalletID:            walletID,
				Name:                "Test Product",
				PriceType:           "one_time",
				Currency:            "USD",
				UnitAmountInPennies: 1000,
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

			product, err := service.CreateProduct(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, product)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, product)
				assert.Equal(t, tt.params.Name, product.Name)
				assert.Equal(t, tt.params.WorkspaceID, product.WorkspaceID)
				assert.Equal(t, tt.params.WalletID, product.WalletID)
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
		ID:                  productID,
		WorkspaceID:         workspaceID,
		Name:                "Test Product",
		PriceType:           "one_time",
		Currency:            "USD",
		UnitAmountInPennies: 1000,
	}

	tests := []struct {
		name        string
		productID   uuid.UUID
		workspaceID uuid.UUID
		setupMocks  func()
		wantProduct *db.Product
		wantErr     bool
		errorString string
	}{
		{
			name:        "successfully retrieves product",
			productID:   productID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(expectedProduct, nil)
			},
			wantProduct: &expectedProduct,
			wantErr:     false,
		},
		{
			name:        "fails when product not found",
			productID:   uuid.New(),
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, gomock.Any()).Return(db.Product{}, pgx.ErrNoRows)
			},
			wantProduct: nil,
			wantErr:     true,
			errorString: "product not found",
		},
		{
			name:        "fails when product belongs to different workspace",
			productID:   productID,
			workspaceID: otherWorkspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: otherWorkspaceID,
				}).Return(db.Product{}, pgx.ErrNoRows)
			},
			wantProduct: nil,
			wantErr:     true,
			errorString: "product not found",
		},
		{
			name:        "handles database error",
			productID:   productID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, gomock.Any()).Return(db.Product{}, errors.New("database error"))
			},
			wantProduct: nil,
			wantErr:     true,
			errorString: "product not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			product, err := service.GetProduct(ctx, params.GetProductParams{
				ProductID:   tt.productID,
				WorkspaceID: tt.workspaceID,
			})

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, product)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, product)
				assert.Equal(t, tt.wantProduct.ID, product.ID)
				assert.Equal(t, tt.wantProduct.Name, product.Name)
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
	metadata := json.RawMessage(`{"updated": true}`)

	existingProduct := db.Product{
		ID:                  productID,
		WorkspaceID:         workspaceID,
		Name:                "Original Product",
		Active:              true,
		PriceType:           "one_time",
		Currency:            "USD",
		UnitAmountInPennies: 1000,
	}

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
				Name:        ptrString("Updated Product"),
				Description: ptrString("Updated Description"),
				ImageURL:    ptrString("https://example.com/new-image.jpg"),
				URL:         ptrString("https://example.com/updated"),
				Active:      ptrBool(false),
				Metadata:    metadata,
			},
			setupMocks: func() {
				// First get the existing product
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(existingProduct, nil)

				// Then update it
				updatedProduct := existingProduct
				updatedProduct.Name = "Updated Product"
				updatedProduct.Active = false
				mockQuerier.EXPECT().UpdateProduct(ctx, gomock.Any()).Return(updatedProduct, nil)
			},
			wantErr: false,
		},
		{
			name: "successfully updates product with partial fields",
			params: params.UpdateProductParams{
				ProductID:   productID,
				WorkspaceID: workspaceID,
				Name:        ptrString("Partially Updated"),
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(existingProduct, nil)

				updatedProduct := existingProduct
				updatedProduct.Name = "Partially Updated"
				mockQuerier.EXPECT().UpdateProduct(ctx, gomock.Any()).Return(updatedProduct, nil)
			},
			wantErr: false,
		},
		{
			name: "fails when product not found",
			params: params.UpdateProductParams{
				ProductID:   uuid.New(),
				WorkspaceID: workspaceID,
				Name:        ptrString("Updated"),
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, gomock.Any()).Return(db.Product{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "product not found",
		},
		{
			name: "handles database error during update",
			params: params.UpdateProductParams{
				ProductID:   productID,
				WorkspaceID: workspaceID,
				Name:        ptrString("Updated"),
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(existingProduct, nil)

				mockQuerier.EXPECT().UpdateProduct(ctx, gomock.Any()).Return(db.Product{}, errors.New("database error"))
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
				// First get the product to validate ownership
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(db.Product{
					ID:          productID,
					WorkspaceID: workspaceID,
				}, nil)
				// Then delete it
				mockQuerier.EXPECT().DeleteProduct(ctx, db.DeleteProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "fails when product not found",
			productID:   uuid.New(),
			workspaceID: workspaceID,
			setupMocks: func() {
				// Get product fails - not found
				mockQuerier.EXPECT().GetProduct(ctx, gomock.Any()).Return(db.Product{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "product not found",
		},
		{
			name:        "handles database error",
			productID:   productID,
			workspaceID: workspaceID,
			setupMocks: func() {
				// Get product succeeds
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(db.Product{
					ID:          productID,
					WorkspaceID: workspaceID,
				}, nil)
				// Delete fails
				mockQuerier.EXPECT().DeleteProduct(ctx, gomock.Any()).Return(errors.New("database error"))
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

func TestProductService_ListProducts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewProductService(mockQuerier)
	ctx := context.Background()

	workspaceID := uuid.New()

	expectedProducts := []db.Product{
		{
			ID:                  uuid.New(),
			WorkspaceID:         workspaceID,
			Name:                "Product 1",
			PriceType:           "one_time",
			Currency:            "USD",
			UnitAmountInPennies: 1000,
		},
		{
			ID:                  uuid.New(),
			WorkspaceID:         workspaceID,
			Name:                "Product 2",
			PriceType:           "recurring",
			Currency:            "USD",
			UnitAmountInPennies: 2000,
			IntervalType:        db.NullIntervalType{IntervalType: db.IntervalTypeMonth, Valid: true},
		},
	}

	tests := []struct {
		name        string
		workspaceID uuid.UUID
		setupMocks  func()
		wantCount   int
		wantErr     bool
		errorString string
	}{
		{
			name:        "successfully lists products",
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListProductsWithPagination(ctx, db.ListProductsWithPaginationParams{
					WorkspaceID: workspaceID,
					Limit:       100,
					Offset:      0,
				}).Return(expectedProducts, nil)
				mockQuerier.EXPECT().CountProducts(ctx, workspaceID).Return(int64(2), nil)
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:        "returns empty list when no products",
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListProductsWithPagination(ctx, db.ListProductsWithPaginationParams{
					WorkspaceID: workspaceID,
					Limit:       100,
					Offset:      0,
				}).Return([]db.Product{}, nil)
				mockQuerier.EXPECT().CountProducts(ctx, workspaceID).Return(int64(0), nil)
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:        "handles database error",
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListProductsWithPagination(ctx, db.ListProductsWithPaginationParams{
					WorkspaceID: workspaceID,
					Limit:       100,
					Offset:      0,
				}).Return(nil, errors.New("database error"))
			},
			wantCount:   0,
			wantErr:     true,
			errorString: "failed to retrieve products",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.ListProducts(ctx, params.ListProductsParams{
				WorkspaceID: tt.workspaceID,
				Limit:       100,
				Offset:      0,
			})

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
				assert.Equal(t, int64(tt.wantCount), result.Total)
			}
		})
	}
}
