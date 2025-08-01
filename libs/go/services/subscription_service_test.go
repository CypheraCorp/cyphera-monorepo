package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	dsClient "github.com/cyphera/cyphera-api/libs/go/client/delegation_server"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func init() {
	logger.InitLogger("test")
}

// Helper function to create subscription service with all required dependencies
func createSubscriptionService(ctrl *gomock.Controller, mockQuerier *mocks.MockQuerier, mockDelegationClient *dsClient.DelegationClient) *services.SubscriptionService {
	paymentService := services.NewPaymentService(mockQuerier, "test-api-key")
	customerService := services.NewCustomerService(mockQuerier)
	mockInvoiceService := mocks.NewMockInvoiceService(ctrl)
	return services.NewSubscriptionService(mockQuerier, mockDelegationClient, paymentService, customerService, mockInvoiceService)
}

// TestProcessSingleSubscription tests the processSingleSubscription method via ProcessDueSubscriptions
// Since processSingleSubscription is not exported, we test it through ProcessDueSubscriptions
// This test focuses on the edge cases and error handling within processSingleSubscription
func TestSubscriptionService_ProcessSingleSubscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	// Use nil delegation client since we're testing error paths that don't reach delegation
	service := createSubscriptionService(ctrl, mockQuerier, nil)

	subscriptionID := uuid.New()
	customerID := uuid.New()
	productID := uuid.New()
	productTokenID := uuid.New()
	delegationID := uuid.New()
	customerWalletID := uuid.New()
	walletID := uuid.New()
	workspaceID := uuid.New()

	tests := []struct {
		name      string
		setupMock func()
		wantErr   bool
		errString string
	}{
		{
			name: "handles already completed subscription",
			setupMock: func() {
				// Mock getting due subscriptions
				dueSubscriptions := []db.ListSubscriptionsDueForRedemptionRow{
					{
						ID:               subscriptionID,
						CustomerID:       customerID,
						ProductID:        productID,
						ProductTokenID:   productTokenID,
						TokenAmount:      1000000,
						DelegationID:     delegationID,
						CustomerWalletID: pgtype.UUID{Bytes: customerWalletID, Valid: true},
						Status:           db.SubscriptionStatusActive,
					},
				}
				mockQuerier.EXPECT().ListSubscriptionsDueForRedemption(gomock.Any(), gomock.Any()).Return(dueSubscriptions, nil)

				// Mock re-fetching subscription - already completed
				currentSub := db.Subscription{
					ID:               subscriptionID,
					CustomerID:       customerID,
					ProductID:        productID,
					ProductTokenID:   productTokenID,
					TokenAmount:      1000000,
					DelegationID:     delegationID,
					CustomerWalletID: pgtype.UUID{Bytes: customerWalletID, Valid: true},
					Status:           db.SubscriptionStatusCompleted, // Already completed
					TotalRedemptions: 1,
				}
				mockQuerier.EXPECT().GetSubscription(gomock.Any(), subscriptionID).Return(currentSub, nil)

				// No further mocks needed - should return early
			},
			wantErr: false,
		},
		{
			name: "handles subscription fetch error",
			setupMock: func() {
				// Mock getting due subscriptions
				dueSubscriptions := []db.ListSubscriptionsDueForRedemptionRow{
					{
						ID:               subscriptionID,
						CustomerID:       customerID,
						ProductID:        productID,
						ProductTokenID:   productTokenID,
						TokenAmount:      1000000,
						DelegationID:     delegationID,
						CustomerWalletID: pgtype.UUID{Bytes: customerWalletID, Valid: true},
						Status:           db.SubscriptionStatusActive,
					},
				}
				mockQuerier.EXPECT().ListSubscriptionsDueForRedemption(gomock.Any(), gomock.Any()).Return(dueSubscriptions, nil)

				// Mock subscription fetch error
				mockQuerier.EXPECT().GetSubscription(gomock.Any(), subscriptionID).Return(db.Subscription{}, errors.New("subscription fetch error"))
			},
			wantErr: false, // ProcessDueSubscriptions handles individual failures gracefully
		},
		{
			name: "handles product fetch error",
			setupMock: func() {
				// Mock getting due subscriptions
				dueSubscriptions := []db.ListSubscriptionsDueForRedemptionRow{
					{
						ID:               subscriptionID,
						CustomerID:       customerID,
						ProductID:        productID,
						ProductTokenID:   productTokenID,
						TokenAmount:      1000000,
						DelegationID:     delegationID,
						CustomerWalletID: pgtype.UUID{Bytes: customerWalletID, Valid: true},
						Status:           db.SubscriptionStatusActive,
					},
				}
				mockQuerier.EXPECT().ListSubscriptionsDueForRedemption(gomock.Any(), gomock.Any()).Return(dueSubscriptions, nil)

				// Mock re-fetching subscription
				currentSub := db.Subscription{
					ID:               subscriptionID,
					CustomerID:       customerID,
					ProductID:        productID,
					ProductTokenID:   productTokenID,
					TokenAmount:      1000000,
					DelegationID:     delegationID,
					CustomerWalletID: pgtype.UUID{Bytes: customerWalletID, Valid: true},
					Status:           db.SubscriptionStatusActive,
					TotalRedemptions: 0,
				}
				mockQuerier.EXPECT().GetSubscription(gomock.Any(), subscriptionID).Return(currentSub, nil)

				// Mock product fetch error
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(gomock.Any(), productID).Return(db.Product{}, errors.New("product fetch error"))
			},
			wantErr: false, // ProcessDueSubscriptions handles individual failures gracefully
		},
		// Test case removed: "handles price fetch error" - prices are now embedded in products
		{
			name: "handles delegation data fetch error",
			setupMock: func() {
				// Mock getting due subscriptions
				dueSubscriptions := []db.ListSubscriptionsDueForRedemptionRow{
					{
						ID:               subscriptionID,
						CustomerID:       customerID,
						ProductID:        productID,
						ProductTokenID:   productTokenID,
						TokenAmount:      1000000,
						DelegationID:     delegationID,
						CustomerWalletID: pgtype.UUID{Bytes: customerWalletID, Valid: true},
						Status:           db.SubscriptionStatusActive,
					},
				}
				mockQuerier.EXPECT().ListSubscriptionsDueForRedemption(gomock.Any(), gomock.Any()).Return(dueSubscriptions, nil)

				// Mock re-fetching subscription
				currentSub := db.Subscription{
					ID:               subscriptionID,
					CustomerID:       customerID,
					ProductID:        productID,
					ProductTokenID:   productTokenID,
					TokenAmount:      1000000,
					DelegationID:     delegationID,
					CustomerWalletID: pgtype.UUID{Bytes: customerWalletID, Valid: true},
					Status:           db.SubscriptionStatusActive,
					TotalRedemptions: 0,
				}
				mockQuerier.EXPECT().GetSubscription(gomock.Any(), subscriptionID).Return(currentSub, nil)

				// Mock getting product
				product := db.Product{
					ID:          productID,
					WorkspaceID: workspaceID,
					WalletID:    walletID,
					Name:        "Test Product",
					Active:      true,
				}
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(gomock.Any(), productID).Return(product, nil)

				// Mock getting price

				// Mock getting customer
				customer := db.Customer{
					ID:    customerID,
					Email: pgtype.Text{String: "test@example.com", Valid: true},
				}
				mockQuerier.EXPECT().GetCustomer(gomock.Any(), customerID).Return(customer, nil)

				// Mock delegation data fetch error
				mockQuerier.EXPECT().GetDelegationData(gomock.Any(), delegationID).Return(db.DelegationDatum{}, errors.New("delegation data fetch error"))
			},
			wantErr: false, // ProcessDueSubscriptions handles individual failures gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := service.ProcessDueSubscriptions(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
			} else {
				// ProcessDueSubscriptions should not return an error even if individual subscriptions fail
				assert.NoError(t, err)
				assert.NotNil(t, result)

				// For the successful case with already completed subscription
				if tt.name == "handles already completed subscription" {
					assert.Equal(t, 1, result.ProcessedCount)
					assert.Equal(t, 1, result.SuccessfulCount)
					assert.Equal(t, 0, result.FailedCount)
				} else {
					// For error cases, we expect the subscription to be marked as failed
					assert.Equal(t, 1, result.ProcessedCount)
					assert.Equal(t, 0, result.SuccessfulCount)
					assert.Equal(t, 1, result.FailedCount)
				}
			}
		})
	}
}

func TestSubscriptionService_GetSubscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockDelegationClient := &dsClient.DelegationClient{} // Mock client
	// Create subscription service with all dependencies
	service := createSubscriptionService(ctrl, mockQuerier, mockDelegationClient)
	ctx := context.Background()

	workspaceID := uuid.New()
	subscriptionID := uuid.New()
	otherWorkspaceID := uuid.New()

	expectedSubscription := db.Subscription{
		ID:          subscriptionID,
		WorkspaceID: workspaceID,
		Status:      db.SubscriptionStatusActive,
		CustomerID:  uuid.New(),
		ProductID:   uuid.New(),
		CurrentPeriodStart: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
		CurrentPeriodEnd: pgtype.Timestamptz{
			Time:  time.Now().Add(30 * 24 * time.Hour),
			Valid: true,
		},
	}

	tests := []struct {
		name           string
		workspaceID    uuid.UUID
		subscriptionID uuid.UUID
		setupMocks     func()
		wantErr        bool
		errorString    string
	}{
		{
			name:           "successfully gets subscription",
			workspaceID:    workspaceID,
			subscriptionID: subscriptionID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionWithWorkspace(ctx, db.GetSubscriptionWithWorkspaceParams{
					ID:          subscriptionID,
					WorkspaceID: workspaceID,
				}).Return(expectedSubscription, nil)
			},
			wantErr: false,
		},
		{
			name:           "subscription not found",
			workspaceID:    workspaceID,
			subscriptionID: subscriptionID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionWithWorkspace(ctx, db.GetSubscriptionWithWorkspaceParams{
					ID:          subscriptionID,
					WorkspaceID: workspaceID,
				}).Return(db.Subscription{}, errors.New("no rows in result set"))
			},
			wantErr:     true,
			errorString: "no rows in result set",
		},
		{
			name:           "database error",
			workspaceID:    workspaceID,
			subscriptionID: subscriptionID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionWithWorkspace(ctx, db.GetSubscriptionWithWorkspaceParams{
					ID:          subscriptionID,
					WorkspaceID: workspaceID,
				}).Return(db.Subscription{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "database error",
		},
		{
			name:           "wrong workspace access",
			workspaceID:    otherWorkspaceID,
			subscriptionID: subscriptionID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionWithWorkspace(ctx, db.GetSubscriptionWithWorkspaceParams{
					ID:          subscriptionID,
					WorkspaceID: otherWorkspaceID,
				}).Return(db.Subscription{}, errors.New("no rows in result set"))
			},
			wantErr:     true,
			errorString: "no rows in result set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			subscription, err := service.GetSubscription(ctx, tt.workspaceID, tt.subscriptionID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, subscription)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, subscription)
				assert.Equal(t, subscriptionID, subscription.ID)
				assert.Equal(t, workspaceID, subscription.WorkspaceID)
			}
		})
	}
}

func TestSubscriptionService_ListSubscriptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockDelegationClient := &dsClient.DelegationClient{}
	// Create subscription service with all dependencies
	service := createSubscriptionService(ctrl, mockQuerier, mockDelegationClient)
	ctx := context.Background()

	workspaceID := uuid.New()
	customerID := uuid.New()
	productID := uuid.New()

	expectedRows := []db.ListSubscriptionDetailsWithPaginationRow{
		{
			SubscriptionID:                 uuid.New(),
			CustomerID:                     customerID,
			ProductID:                      productID,
			ProductWorkspaceID:             workspaceID,
			SubscriptionStatus:             db.SubscriptionStatusActive,
			CustomerName:                   pgtype.Text{String: "Test Customer", Valid: true},
			CustomerEmail:                  pgtype.Text{String: "test@example.com", Valid: true},
			ProductName:                    "Test Product",
			SubscriptionCurrentPeriodStart: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			SubscriptionCurrentPeriodEnd:   pgtype.Timestamptz{Time: time.Now().Add(30 * 24 * time.Hour), Valid: true},
			ProductActive:                  true,
			PriceType:                      "recurring",
			PriceCurrency:                  "USD",
			PriceUnitAmountInPennies:       1000,
			PriceIntervalType:              db.NullIntervalType{IntervalType: db.IntervalTypeMonth, Valid: true},
			PriceTermLength:                pgtype.Int4{Int32: 1, Valid: true},
			ProductTokenID:                 uuid.New(),
			ProductTokenTokenID:            uuid.New(),
			ProductTokenNetworkID:          uuid.New(),
			TokenSymbol:                    "USDC",
		},
		{
			SubscriptionID:                 uuid.New(),
			CustomerID:                     customerID,
			ProductID:                      productID,
			ProductWorkspaceID:             workspaceID,
			SubscriptionStatus:             db.SubscriptionStatusCanceled,
			CustomerName:                   pgtype.Text{String: "Test Customer 2", Valid: true},
			CustomerEmail:                  pgtype.Text{String: "test2@example.com", Valid: true},
			ProductName:                    "Test Product 2",
			SubscriptionCurrentPeriodStart: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			SubscriptionCurrentPeriodEnd:   pgtype.Timestamptz{Time: time.Now().Add(30 * 24 * time.Hour), Valid: true},
			ProductActive:                  true,
			PriceType:                      "recurring",
			PriceCurrency:                  "USD",
			PriceUnitAmountInPennies:       2000,
			PriceIntervalType:              db.NullIntervalType{IntervalType: db.IntervalTypeMonth, Valid: true},
			PriceTermLength:                pgtype.Int4{Int32: 1, Valid: true},
			ProductTokenID:                 uuid.New(),
			ProductTokenTokenID:            uuid.New(),
			ProductTokenNetworkID:          uuid.New(),
			TokenSymbol:                    "USDC",
		},
	}

	expectedCount := int64(10)

	tests := []struct {
		name        string
		workspaceID uuid.UUID
		limit       int32
		offset      int32
		setupMocks  func()
		wantErr     bool
		errorString string
		wantCount   int
		wantTotal   int64
	}{
		{
			name:        "successfully lists subscriptions with pagination",
			workspaceID: workspaceID,
			limit:       20,
			offset:      0,
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionDetailsWithPagination(ctx, db.ListSubscriptionDetailsWithPaginationParams{
					WorkspaceID: workspaceID,
					Limit:       20,
					Offset:      0,
				}).Return(expectedRows, nil)
				mockQuerier.EXPECT().CountSubscriptions(ctx).Return(expectedCount, nil)
			},
			wantErr:   false,
			wantCount: 2,
			wantTotal: expectedCount,
		},
		{
			name:        "database error getting subscriptions",
			workspaceID: workspaceID,
			limit:       20,
			offset:      0,
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionDetailsWithPagination(ctx, db.ListSubscriptionDetailsWithPaginationParams{
					WorkspaceID: workspaceID,
					Limit:       20,
					Offset:      0,
				}).Return(nil, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve subscriptions",
		},
		{
			name:        "database error counting subscriptions",
			workspaceID: workspaceID,
			limit:       20,
			offset:      0,
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionDetailsWithPagination(ctx, db.ListSubscriptionDetailsWithPaginationParams{
					WorkspaceID: workspaceID,
					Limit:       20,
					Offset:      0,
				}).Return(expectedRows, nil)
				mockQuerier.EXPECT().CountSubscriptions(ctx).Return(int64(0), errors.New("count error"))
			},
			wantErr:     true,
			errorString: "failed to count subscriptions",
		},
		{
			name:        "empty subscription list",
			workspaceID: workspaceID,
			limit:       20,
			offset:      0,
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionDetailsWithPagination(ctx, db.ListSubscriptionDetailsWithPaginationParams{
					WorkspaceID: workspaceID,
					Limit:       20,
					Offset:      0,
				}).Return([]db.ListSubscriptionDetailsWithPaginationRow{}, nil)
				mockQuerier.EXPECT().CountSubscriptions(ctx).Return(int64(0), nil)
			},
			wantErr:   false,
			wantCount: 0,
			wantTotal: 0,
		},
		{
			name:        "pagination with offset",
			workspaceID: workspaceID,
			limit:       10,
			offset:      20,
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionDetailsWithPagination(ctx, db.ListSubscriptionDetailsWithPaginationParams{
					WorkspaceID: workspaceID,
					Limit:       10,
					Offset:      20,
				}).Return(expectedRows[:1], nil)
				mockQuerier.EXPECT().CountSubscriptions(ctx).Return(expectedCount, nil)
			},
			wantErr:   false,
			wantCount: 1,
			wantTotal: expectedCount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			subscriptions, totalCount, err := service.ListSubscriptions(ctx, tt.workspaceID, tt.limit, tt.offset)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, subscriptions)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, subscriptions, tt.wantCount)
				assert.Equal(t, tt.wantTotal, totalCount)
				// Verify conversion worked properly
				for _, sub := range subscriptions {
					assert.IsType(t, responses.SubscriptionResponse{}, sub)
					if tt.wantCount > 0 {
						assert.NotEmpty(t, sub.CustomerName)
						assert.NotEmpty(t, sub.Product.Name)
					}
				}
			}
		})
	}
}

func TestSubscriptionService_ListSubscriptionsByCustomer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockDelegationClient := &dsClient.DelegationClient{}
	// Create subscription service with all dependencies
	service := createSubscriptionService(ctrl, mockQuerier, mockDelegationClient)
	ctx := context.Background()

	workspaceID := uuid.New()
	customerID := uuid.New()
	otherWorkspaceID := uuid.New()

	expectedSubscriptions := []db.Subscription{
		{
			ID:          uuid.New(),
			CustomerID:  customerID,
			WorkspaceID: workspaceID,
			Status:      db.SubscriptionStatusActive,
		},
		{
			ID:          uuid.New(),
			CustomerID:  customerID,
			WorkspaceID: workspaceID,
			Status:      db.SubscriptionStatusCanceled,
		},
	}

	tests := []struct {
		name        string
		workspaceID uuid.UUID
		customerID  uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
		wantCount   int
	}{
		{
			name:        "successfully lists customer subscriptions",
			workspaceID: workspaceID,
			customerID:  customerID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionsByCustomer(ctx, db.ListSubscriptionsByCustomerParams{
					CustomerID:  customerID,
					WorkspaceID: workspaceID,
				}).Return(expectedSubscriptions, nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:        "database error",
			workspaceID: workspaceID,
			customerID:  customerID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionsByCustomer(ctx, db.ListSubscriptionsByCustomerParams{
					CustomerID:  customerID,
					WorkspaceID: workspaceID,
				}).Return(nil, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve customer subscriptions",
		},
		{
			name:        "empty result",
			workspaceID: workspaceID,
			customerID:  customerID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionsByCustomer(ctx, db.ListSubscriptionsByCustomerParams{
					CustomerID:  customerID,
					WorkspaceID: workspaceID,
				}).Return([]db.Subscription{}, nil)
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:        "different workspace",
			workspaceID: otherWorkspaceID,
			customerID:  customerID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionsByCustomer(ctx, db.ListSubscriptionsByCustomerParams{
					CustomerID:  customerID,
					WorkspaceID: otherWorkspaceID,
				}).Return([]db.Subscription{}, nil)
			},
			wantErr:   false,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			subscriptions, err := service.ListSubscriptionsByCustomer(ctx, tt.workspaceID, tt.customerID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, subscriptions)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, subscriptions, tt.wantCount)
			}
		})
	}
}

func TestSubscriptionService_ListSubscriptionsByProduct(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockDelegationClient := &dsClient.DelegationClient{}
	// Create subscription service with all dependencies
	service := createSubscriptionService(ctrl, mockQuerier, mockDelegationClient)
	ctx := context.Background()

	workspaceID := uuid.New()
	productID := uuid.New()

	expectedSubscriptions := []db.Subscription{
		{
			ID:          uuid.New(),
			ProductID:   productID,
			WorkspaceID: workspaceID,
			Status:      db.SubscriptionStatusActive,
		},
		{
			ID:          uuid.New(),
			ProductID:   productID,
			WorkspaceID: workspaceID,
			Status:      db.SubscriptionStatusActive,
		},
	}

	tests := []struct {
		name        string
		workspaceID uuid.UUID
		productID   uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
		wantCount   int
	}{
		{
			name:        "successfully lists product subscriptions",
			workspaceID: workspaceID,
			productID:   productID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionsByProduct(ctx, db.ListSubscriptionsByProductParams{
					ProductID:   productID,
					WorkspaceID: workspaceID,
				}).Return(expectedSubscriptions, nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:        "database error",
			workspaceID: workspaceID,
			productID:   productID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionsByProduct(ctx, db.ListSubscriptionsByProductParams{
					ProductID:   productID,
					WorkspaceID: workspaceID,
				}).Return(nil, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve product subscriptions",
		},
		{
			name:        "no subscriptions for product",
			workspaceID: workspaceID,
			productID:   productID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionsByProduct(ctx, db.ListSubscriptionsByProductParams{
					ProductID:   productID,
					WorkspaceID: workspaceID,
				}).Return([]db.Subscription{}, nil)
			},
			wantErr:   false,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			subscriptions, err := service.ListSubscriptionsByProduct(ctx, tt.workspaceID, tt.productID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, subscriptions)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, subscriptions, tt.wantCount)
			}
		})
	}
}

func TestSubscriptionService_UpdateSubscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockDelegationClient := &dsClient.DelegationClient{}
	// Create subscription service with all dependencies
	service := createSubscriptionService(ctrl, mockQuerier, mockDelegationClient)
	ctx := context.Background()

	subscriptionID := uuid.New()
	customerID := uuid.New()
	productID := uuid.New()
	productTokenID := uuid.New()
	delegationID := uuid.New()
	customerWalletID := uuid.New()

	existingSubscription := db.Subscription{
		ID:               subscriptionID,
		CustomerID:       customerID,
		ProductID:        productID,
		ProductTokenID:   productTokenID,
		DelegationID:     delegationID,
		CustomerWalletID: pgtype.UUID{Bytes: customerWalletID, Valid: true},
		Status:           db.SubscriptionStatusActive,
		CurrentPeriodStart: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
		CurrentPeriodEnd: pgtype.Timestamptz{
			Time:  time.Now().Add(30 * 24 * time.Hour),
			Valid: true,
		},
		NextRedemptionDate: pgtype.Timestamptz{
			Time:  time.Now().Add(24 * time.Hour),
			Valid: true,
		},
		TotalRedemptions:   5,
		TotalAmountInCents: 10000,
		Metadata:           json.RawMessage(`{"key": "value"}`),
	}

	updatedSubscription := existingSubscription
	updatedSubscription.Status = db.SubscriptionStatusCanceled

	tests := []struct {
		name           string
		subscriptionID uuid.UUID
		req            requests.UpdateSubscriptionRequest
		setupMocks     func()
		wantErr        bool
		errorString    string
		validateResult func(*db.Subscription)
	}{
		{
			name:           "successfully updates subscription status",
			subscriptionID: subscriptionID,
			req: requests.UpdateSubscriptionRequest{
				Status: "canceled",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(existingSubscription, nil)
				mockQuerier.EXPECT().UpdateSubscription(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, params db.UpdateSubscriptionParams) (db.Subscription, error) {
						assert.Equal(t, subscriptionID, params.ID)
						assert.Equal(t, db.SubscriptionStatusCanceled, params.Status)
						return updatedSubscription, nil
					})
			},
			wantErr: false,
			validateResult: func(sub *db.Subscription) {
				assert.Equal(t, db.SubscriptionStatusCanceled, sub.Status)
			},
		},
		{
			name:           "successfully updates multiple fields",
			subscriptionID: subscriptionID,
			req: requests.UpdateSubscriptionRequest{
				CustomerID:     uuid.New().String(),
				ProductID:      uuid.New().String(),
				Status:         "suspended",
				StartDate:      time.Now().Unix(),
				EndDate:        time.Now().Add(60 * 24 * time.Hour).Unix(),
				NextRedemption: time.Now().Add(7 * 24 * time.Hour).Unix(),
				Metadata:       json.RawMessage(`{"updated": true}`),
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(existingSubscription, nil)
				mockQuerier.EXPECT().UpdateSubscription(ctx, gomock.Any()).Return(updatedSubscription, nil)
			},
			wantErr: false,
		},
		{
			name:           "subscription not found",
			subscriptionID: subscriptionID,
			req: requests.UpdateSubscriptionRequest{
				Status: "canceled",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(db.Subscription{}, errors.New("no rows in result set"))
			},
			wantErr:     true,
			errorString: "subscription not found",
		},
		{
			name:           "invalid customer ID format",
			subscriptionID: subscriptionID,
			req: requests.UpdateSubscriptionRequest{
				CustomerID: "invalid-uuid",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(existingSubscription, nil)
			},
			wantErr:     true,
			errorString: "invalid customer ID format",
		},
		{
			name:           "invalid product ID format",
			subscriptionID: subscriptionID,
			req: requests.UpdateSubscriptionRequest{
				ProductID: "invalid-uuid",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(existingSubscription, nil)
			},
			wantErr:     true,
			errorString: "invalid product ID format",
		},
		{
			name:           "invalid product token ID format",
			subscriptionID: subscriptionID,
			req: requests.UpdateSubscriptionRequest{
				ProductTokenID: "invalid-uuid",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(existingSubscription, nil)
			},
			wantErr:     true,
			errorString: "invalid product token ID format",
		},
		{
			name:           "invalid delegation ID format",
			subscriptionID: subscriptionID,
			req: requests.UpdateSubscriptionRequest{
				DelegationID: "invalid-uuid",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(existingSubscription, nil)
			},
			wantErr:     true,
			errorString: "invalid delegation ID format",
		},
		{
			name:           "invalid customer wallet ID format",
			subscriptionID: subscriptionID,
			req: requests.UpdateSubscriptionRequest{
				CustomerWalletID: "invalid-uuid",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(existingSubscription, nil)
			},
			wantErr:     true,
			errorString: "invalid customer wallet ID format",
		},
		{
			name:           "invalid status value",
			subscriptionID: subscriptionID,
			req: requests.UpdateSubscriptionRequest{
				Status: "invalid_status",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(existingSubscription, nil)
			},
			wantErr:     true,
			errorString: "invalid status value",
		},
		{
			name:           "database error on update",
			subscriptionID: subscriptionID,
			req: requests.UpdateSubscriptionRequest{
				Status: "canceled",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(existingSubscription, nil)
				mockQuerier.EXPECT().UpdateSubscription(ctx, gomock.Any()).Return(db.Subscription{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to update subscription",
		},
		{
			name:           "empty update request uses existing values",
			subscriptionID: subscriptionID,
			req:            requests.UpdateSubscriptionRequest{},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(existingSubscription, nil)
				mockQuerier.EXPECT().UpdateSubscription(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, params db.UpdateSubscriptionParams) (db.Subscription, error) {
						// Verify all fields remain unchanged
						assert.Equal(t, existingSubscription.CustomerID, params.CustomerID)
						assert.Equal(t, existingSubscription.ProductID, params.ProductID)
						assert.Equal(t, existingSubscription.Status, params.Status)
						return existingSubscription, nil
					})
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			subscription, err := service.UpdateSubscription(ctx, tt.subscriptionID, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, subscription)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, subscription)
				if tt.validateResult != nil {
					tt.validateResult(subscription)
				}
			}
		})
	}
}

func TestSubscriptionService_DeleteSubscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockDelegationClient := &dsClient.DelegationClient{}
	// Create subscription service with all dependencies
	service := createSubscriptionService(ctrl, mockQuerier, mockDelegationClient)
	ctx := context.Background()

	workspaceID := uuid.New()
	subscriptionID := uuid.New()
	otherWorkspaceID := uuid.New()

	canceledSubscription := db.Subscription{
		ID:          subscriptionID,
		WorkspaceID: workspaceID,
		Status:      db.SubscriptionStatusCanceled,
	}

	expiredSubscription := db.Subscription{
		ID:          subscriptionID,
		WorkspaceID: workspaceID,
		Status:      db.SubscriptionStatusExpired,
	}

	activeSubscription := db.Subscription{
		ID:          subscriptionID,
		WorkspaceID: workspaceID,
		Status:      db.SubscriptionStatusActive,
	}

	wrongWorkspaceSubscription := db.Subscription{
		ID:          subscriptionID,
		WorkspaceID: otherWorkspaceID,
		Status:      db.SubscriptionStatusCanceled,
	}

	tests := []struct {
		name           string
		workspaceID    uuid.UUID
		subscriptionID uuid.UUID
		setupMocks     func()
		wantErr        bool
		errorString    string
	}{
		{
			name:           "successfully deletes canceled subscription",
			workspaceID:    workspaceID,
			subscriptionID: subscriptionID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionWithWorkspace(ctx, db.GetSubscriptionWithWorkspaceParams{
					ID:          subscriptionID,
					WorkspaceID: workspaceID,
				}).Return(canceledSubscription, nil)
				mockQuerier.EXPECT().DeleteSubscription(ctx, subscriptionID).Return(nil)
			},
			wantErr: false,
		},
		{
			name:           "successfully deletes expired subscription",
			workspaceID:    workspaceID,
			subscriptionID: subscriptionID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionWithWorkspace(ctx, db.GetSubscriptionWithWorkspaceParams{
					ID:          subscriptionID,
					WorkspaceID: workspaceID,
				}).Return(expiredSubscription, nil)
				mockQuerier.EXPECT().DeleteSubscription(ctx, subscriptionID).Return(nil)
			},
			wantErr: false,
		},
		{
			name:           "subscription not found",
			workspaceID:    workspaceID,
			subscriptionID: subscriptionID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionWithWorkspace(ctx, db.GetSubscriptionWithWorkspaceParams{
					ID:          subscriptionID,
					WorkspaceID: workspaceID,
				}).Return(db.Subscription{}, errors.New("no rows in result set"))
			},
			wantErr:     true,
			errorString: "subscription not found",
		},
		{
			name:           "cannot delete active subscription",
			workspaceID:    workspaceID,
			subscriptionID: subscriptionID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionWithWorkspace(ctx, db.GetSubscriptionWithWorkspaceParams{
					ID:          subscriptionID,
					WorkspaceID: workspaceID,
				}).Return(activeSubscription, nil)
			},
			wantErr:     true,
			errorString: "subscription is not canceled or expired",
		},
		{
			name:           "subscription belongs to different workspace",
			workspaceID:    workspaceID,
			subscriptionID: subscriptionID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionWithWorkspace(ctx, db.GetSubscriptionWithWorkspaceParams{
					ID:          subscriptionID,
					WorkspaceID: workspaceID,
				}).Return(wrongWorkspaceSubscription, nil)
			},
			wantErr:     true,
			errorString: "subscription does not belong to this workspace",
		},
		{
			name:           "database error on delete",
			workspaceID:    workspaceID,
			subscriptionID: subscriptionID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionWithWorkspace(ctx, db.GetSubscriptionWithWorkspaceParams{
					ID:          subscriptionID,
					WorkspaceID: workspaceID,
				}).Return(canceledSubscription, nil)
				mockQuerier.EXPECT().DeleteSubscription(ctx, subscriptionID).Return(errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to delete subscription",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.DeleteSubscription(ctx, tt.workspaceID, tt.subscriptionID)

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

func TestSubscriptionService_SubscriptionExistsError(t *testing.T) {
	subscription := &db.Subscription{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
		ProductID:  uuid.New(),
	}

	err := &services.SubscriptionExistsError{
		Subscription: subscription,
	}

	assert.Equal(t, "subscription already exists for this customer and product", err.Error())
	assert.Equal(t, subscription, err.Subscription)
}

func TestSubscriptionService_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockDelegationClient := &dsClient.DelegationClient{}
	// Create subscription service with all dependencies
	service := createSubscriptionService(ctrl, mockQuerier, mockDelegationClient)

	tests := []struct {
		name        string
		operation   func() error
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil context handling",
			operation: func() error {
				// The service will still try to make the database call even with nil context
				mockQuerier.EXPECT().GetSubscriptionWithWorkspace(gomock.Any(), gomock.Any()).Return(db.Subscription{}, errors.New("context error"))
				_, err := service.GetSubscription(context.TODO(), uuid.New(), uuid.New())
				return err
			},
			expectError: true,
		},
		{
			name: "zero UUID handling in GetSubscription",
			operation: func() error {
				mockQuerier.EXPECT().GetSubscriptionWithWorkspace(gomock.Any(), gomock.Any()).Return(db.Subscription{}, errors.New("no rows in result set"))
				_, err := service.GetSubscription(context.Background(), uuid.Nil, uuid.Nil)
				return err
			},
			expectError: true,
		},
		{
			name: "negative pagination parameters",
			operation: func() error {
				mockQuerier.EXPECT().ListSubscriptionDetailsWithPagination(gomock.Any(), gomock.Any()).Return([]db.ListSubscriptionDetailsWithPaginationRow{}, nil)
				mockQuerier.EXPECT().CountSubscriptions(gomock.Any()).Return(int64(0), nil)
				_, _, err := service.ListSubscriptions(context.Background(), uuid.New(), -10, -5)
				return err
			},
			expectError: false,
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

func TestSubscriptionService_BoundaryConditions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockDelegationClient := &dsClient.DelegationClient{}
	// Create subscription service with all dependencies
	service := createSubscriptionService(ctrl, mockQuerier, mockDelegationClient)
	ctx := context.Background()

	tests := []struct {
		name       string
		setupMocks func()
		operation  func() error
		wantErr    bool
	}{
		{
			name: "update with very long metadata",
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, gomock.Any()).Return(db.Subscription{
					ID: uuid.New(),
				}, nil)
				mockQuerier.EXPECT().UpdateSubscription(ctx, gomock.Any()).Return(db.Subscription{}, nil)
			},
			operation: func() error {
				longMetadata := make([]byte, 10000)
				for i := range longMetadata {
					longMetadata[i] = 'a'
				}
				_, err := service.UpdateSubscription(ctx, uuid.New(), requests.UpdateSubscriptionRequest{
					Metadata: json.RawMessage(longMetadata),
				})
				return err
			},
			wantErr: false,
		},
		{
			name: "update with all valid statuses",
			setupMocks: func() {
				validStatuses := []string{"active", "canceled", "expired", "suspended", "failed"}
				for i := range validStatuses {
					mockQuerier.EXPECT().GetSubscription(ctx, gomock.Any()).Return(db.Subscription{
						ID: uuid.New(),
					}, nil)
					if i < len(validStatuses) {
						mockQuerier.EXPECT().UpdateSubscription(ctx, gomock.Any()).Return(db.Subscription{}, nil)
					}
				}
			},
			operation: func() error {
				validStatuses := []string{"active", "canceled", "expired", "suspended", "failed"}
				for _, status := range validStatuses {
					_, err := service.UpdateSubscription(ctx, uuid.New(), requests.UpdateSubscriptionRequest{
						Status: status,
					})
					if err != nil {
						return err
					}
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "list with maximum limit",
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionDetailsWithPagination(ctx, gomock.Any()).Return([]db.ListSubscriptionDetailsWithPaginationRow{}, nil)
				mockQuerier.EXPECT().CountSubscriptions(ctx).Return(int64(1000000), nil)
			},
			operation: func() error {
				_, _, err := service.ListSubscriptions(ctx, uuid.New(), 2147483647, 0)
				return err
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := tt.operation()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSubscriptionService_ConcurrentAccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockDelegationClient := &dsClient.DelegationClient{}
	// Create subscription service with all dependencies
	service := createSubscriptionService(ctrl, mockQuerier, mockDelegationClient)
	ctx := context.Background()

	// Test concurrent subscription retrieval
	t.Run("concurrent subscription retrieval", func(t *testing.T) {
		const numGoroutines = 10
		resultsChan := make(chan *db.Subscription, numGoroutines)
		errorsChan := make(chan error, numGoroutines)

		workspaceID := uuid.New()
		subscriptionID := uuid.New()
		testSubscription := db.Subscription{
			ID:          subscriptionID,
			WorkspaceID: workspaceID,
			Status:      db.SubscriptionStatusActive,
		}

		// Setup mocks for concurrent calls
		mockQuerier.EXPECT().GetSubscriptionWithWorkspace(ctx, db.GetSubscriptionWithWorkspaceParams{
			ID:          subscriptionID,
			WorkspaceID: workspaceID,
		}).Return(testSubscription, nil).Times(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				subscription, err := service.GetSubscription(ctx, workspaceID, subscriptionID)
				if err != nil {
					errorsChan <- err
				} else {
					resultsChan <- subscription
				}
			}()
		}

		// Collect results
		var results []*db.Subscription
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
			assert.Equal(t, subscriptionID, result.ID)
			assert.Equal(t, workspaceID, result.WorkspaceID)
		}
	})
}

// TestSubscriptionService_SubscribeToProductByPriceID tests the critical subscription workflow
func TestSubscriptionService_SubscribeToProductByPriceID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Test data setup
	productID := uuid.New()
	productTokenID := uuid.New()
	walletID := uuid.New()
	workspaceID := uuid.New()
	tokenID := uuid.New()
	networkID := uuid.New()
	customerID := uuid.New()
	customerWalletID := uuid.New()
	delegationDataID := uuid.New()
	subscriptionID := uuid.New()

	// Valid test objects

	validProduct := db.Product{
		ID:          productID,
		WorkspaceID: workspaceID,
		WalletID:    walletID,
		Name:        "Test Product",
		Active:      true,
	}

	validProductToken := db.GetProductTokenRow{
		ID:          productTokenID,
		ProductID:   productID,
		TokenID:     tokenID,
		NetworkID:   networkID,
		NetworkType: string(db.NetworkTypeEvm),
		Active:      true,
	}

	validToken := db.Token{
		ID:              tokenID,
		NetworkID:       networkID,
		ContractAddress: "0xtoken123",
		Symbol:          "USDC",
		Active:          true,
		Decimals:        6, // USDC has 6 decimals
	}

	validNetwork := db.Network{
		ID:          networkID,
		Name:        "ethereum",
		Type:        "evm",
		NetworkType: db.NetworkTypeEvm,
		ChainID:     1, // Ethereum mainnet
	}

	validWallet := db.Wallet{
		ID:            walletID,
		WorkspaceID:   workspaceID,
		WalletAddress: "0xmerchant123",
	}

	validCustomer := db.Customer{
		ID:    customerID,
		Email: pgtype.Text{String: "test@example.com", Valid: true},
	}

	validCustomerWallet := db.CustomerWallet{
		ID:            customerWalletID,
		CustomerID:    customerID,
		WalletAddress: "0xcustomer123",
		NetworkType:   "evm",
	}

	validDelegationData := db.DelegationDatum{
		ID:        delegationDataID,
		Delegate:  "0xcyphera123",
		Delegator: "0xcustomer123",
		Authority: "0xauthority123",
		Salt:      "0xsalt123",
		Signature: "0xsignature123",
		Caveats:   json.RawMessage(`[{"type":"test"}]`),
	}

	validSubscription := db.Subscription{
		ID:               subscriptionID,
		CustomerID:       customerID,
		WorkspaceID:      workspaceID,
		ProductID:        productID,
		ProductTokenID:   productTokenID,
		CustomerWalletID: pgtype.UUID{Bytes: customerWalletID, Valid: true},
		DelegationID:     delegationDataID,
		TokenAmount:      1000000,
		Status:           db.SubscriptionStatusActive,
		CurrentPeriodStart: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
		CurrentPeriodEnd: pgtype.Timestamptz{
			Time:  time.Now().Add(30 * 24 * time.Hour),
			Valid: true,
		},
		NextRedemptionDate: pgtype.Timestamptz{
			Time:  time.Now().Add(24 * time.Hour),
			Valid: true,
		},
	}

	// Create valid subscription params
	validParams := params.SubscribeToProductByPriceIDParams{
		ProductID:         productID, // Now using product ID directly instead of price ID
		SubscriberAddress: "0xcustomer123",
		ProductTokenID:    productTokenID.String(),
		TokenAmount:       "1000000",
		DelegationData: params.DelegationParams{
			Delegate:  "0xcyphera123",
			Delegator: "0xcustomer123",
			Authority: "0xauthority123",
			Salt:      "0xsalt123",
			Signature: "0xsignature123",
			Caveats:   json.RawMessage(`["test"]`),
		},
	}

	tests := []struct {
		name        string
		params      params.SubscribeToProductByPriceIDParams
		setupMocks  func(*mocks.MockQuerier)
		wantErr     bool
		wantSuccess bool
		errorString string
	}{
		{
			name:   "creates subscription but fails initial redemption due to nil delegation client",
			params: validParams,
			setupMocks: func(mockQuerier *mocks.MockQuerier) {
				// Mock getting product
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(gomock.Any(), productID).Return(validProduct, nil)

				// Mock getting merchant wallet
				mockQuerier.EXPECT().GetWalletByID(gomock.Any(), db.GetWalletByIDParams{
					ID:          walletID,
					WorkspaceID: workspaceID,
				}).Return(validWallet, nil)

				// Mock getting product token
				mockQuerier.EXPECT().GetProductToken(gomock.Any(), productTokenID).Return(validProductToken, nil)

				// Mock getting token
				mockQuerier.EXPECT().GetToken(gomock.Any(), tokenID).Return(validToken, nil)

				// Mock getting network
				mockQuerier.EXPECT().GetNetwork(gomock.Any(), networkID).Return(validNetwork, nil)

				// Mock customer processing queries
				mockQuerier.EXPECT().GetCustomersByWalletAddress(gomock.Any(), "0xcustomer123").Return([]db.Customer{validCustomer}, nil)
				mockQuerier.EXPECT().IsCustomerInWorkspace(gomock.Any(), gomock.Any()).Return(true, nil)
				mockQuerier.EXPECT().ListCustomerWallets(gomock.Any(), customerID).Return([]db.CustomerWallet{validCustomerWallet}, nil)
				mockQuerier.EXPECT().UpdateCustomerWalletUsageTime(gomock.Any(), customerWalletID).Return(validCustomerWallet, nil)

				// Mock delegation data storage
				mockQuerier.EXPECT().CreateDelegationData(gomock.Any(), gomock.Any()).Return(validDelegationData, nil)

				// Mock checking for existing subscriptions
				mockQuerier.EXPECT().ListSubscriptionsByCustomer(gomock.Any(), gomock.Any()).Return([]db.Subscription{}, nil)

				// Mock subscription creation
				mockQuerier.EXPECT().CreateSubscription(gomock.Any(), gomock.Any()).Return(validSubscription, nil)

				// Mock subscription event creation
				mockQuerier.EXPECT().CreateSubscriptionEvent(gomock.Any(), gomock.Any()).Return(db.SubscriptionEvent{}, nil)

				// Since delegation client is nil in tests, the initial redemption will fail immediately
				// We need to mock the cleanup operations
				mockQuerier.EXPECT().UpdateSubscriptionStatus(gomock.Any(), gomock.Any()).Return(db.Subscription{}, nil)
				mockQuerier.EXPECT().DeleteSubscription(gomock.Any(), gomock.Any()).Return(nil)
				mockQuerier.EXPECT().CreateSubscriptionEvent(gomock.Any(), gomock.Any()).Return(db.SubscriptionEvent{}, nil)
			},
			wantErr:     false,
			wantSuccess: false,
			errorString: "Initial redemption failed, subscription marked as failed and soft-deleted",
		},
		// Test case removed: "fails with inactive price" - prices are now embedded in products
		{
			name: "fails with inactive product",
			params: params.SubscribeToProductByPriceIDParams{
				ProductID:         productID,
				SubscriberAddress: "0xcustomer123",
				ProductTokenID:    productTokenID.String(),
				TokenAmount:       "1000000",
			},
			setupMocks: func(mockQuerier *mocks.MockQuerier) {

				inactiveProduct := validProduct
				inactiveProduct.Active = false
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(gomock.Any(), productID).Return(inactiveProduct, nil)
			},
			wantErr:     false,
			wantSuccess: false,
			errorString: "Cannot subscribe to inactive product",
		},
		{
			name: "fails with invalid product token ID format",
			params: params.SubscribeToProductByPriceIDParams{
				ProductID:         productID,
				SubscriberAddress: "0xcustomer123",
				ProductTokenID:    "invalid-uuid",
				TokenAmount:       "1000000",
			},
			setupMocks: func(mockQuerier *mocks.MockQuerier) {
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(gomock.Any(), productID).Return(validProduct, nil)
			},
			wantErr:     false,
			wantSuccess: false,
			errorString: "Invalid product token ID format",
		},
		{
			name: "fails when product token doesn't belong to product",
			params: params.SubscribeToProductByPriceIDParams{
				ProductID:         productID,
				SubscriberAddress: "0xcustomer123",
				ProductTokenID:    productTokenID.String(),
				TokenAmount:       "1000000",
			},
			setupMocks: func(mockQuerier *mocks.MockQuerier) {
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(gomock.Any(), productID).Return(validProduct, nil)
				mockQuerier.EXPECT().GetWalletByID(gomock.Any(), gomock.Any()).Return(validWallet, nil)

				wrongProductToken := validProductToken
				wrongProductToken.ProductID = uuid.New() // Different product
				mockQuerier.EXPECT().GetProductToken(gomock.Any(), productTokenID).Return(wrongProductToken, nil)
			},
			wantErr:     false,
			wantSuccess: false,
			errorString: "Product token does not belong to the specified product",
		},
		{
			name: "fails with invalid token amount format",
			params: params.SubscribeToProductByPriceIDParams{
				ProductID:         productID,
				SubscriberAddress: "0xcustomer123",
				ProductTokenID:    productTokenID.String(),
				TokenAmount:       "invalid-amount",
			},
			setupMocks: func(mockQuerier *mocks.MockQuerier) {
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(gomock.Any(), productID).Return(validProduct, nil)
				mockQuerier.EXPECT().GetWalletByID(gomock.Any(), gomock.Any()).Return(validWallet, nil)
				mockQuerier.EXPECT().GetProductToken(gomock.Any(), productTokenID).Return(validProductToken, nil)
				mockQuerier.EXPECT().GetToken(gomock.Any(), tokenID).Return(validToken, nil)
				mockQuerier.EXPECT().GetNetwork(gomock.Any(), networkID).Return(validNetwork, nil)
			},
			wantErr:     false,
			wantSuccess: false,
			errorString: "Invalid token amount format",
		},
		{
			name:   "handles product not found error with empty params",
			params: params.SubscribeToProductByPriceIDParams{},
			setupMocks: func(mockQuerier *mocks.MockQuerier) {
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(gomock.Any(), uuid.UUID{}).Return(db.Product{}, errors.New("no rows"))
			},
			wantErr:     true,
			errorString: "failed to get product",
		},
		{
			name: "handles product not found error",
			params: params.SubscribeToProductByPriceIDParams{
				ProductID: productID,
			},
			setupMocks: func(mockQuerier *mocks.MockQuerier) {
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(gomock.Any(), productID).Return(db.Product{}, errors.New("no rows"))
			},
			wantErr:     true,
			errorString: "failed to get product",
		},
		{
			name: "handles merchant wallet not found error",
			params: params.SubscribeToProductByPriceIDParams{
				ProductID:         productID,
				SubscriberAddress: "0xcustomer123",
				ProductTokenID:    productTokenID.String(),
				TokenAmount:       "1000000",
			},
			setupMocks: func(mockQuerier *mocks.MockQuerier) {
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(gomock.Any(), productID).Return(validProduct, nil)
				mockQuerier.EXPECT().GetWalletByID(gomock.Any(), gomock.Any()).Return(db.Wallet{}, errors.New("no rows"))
			},
			wantErr:     true,
			errorString: "failed to get merchant wallet",
		},
		{
			name:   "handles subscription already exists error",
			params: validParams,
			setupMocks: func(mockQuerier *mocks.MockQuerier) {
				// Setup all the initial mocks
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(gomock.Any(), productID).Return(validProduct, nil)
				mockQuerier.EXPECT().GetWalletByID(gomock.Any(), gomock.Any()).Return(validWallet, nil)
				mockQuerier.EXPECT().GetProductToken(gomock.Any(), productTokenID).Return(validProductToken, nil)
				mockQuerier.EXPECT().GetToken(gomock.Any(), tokenID).Return(validToken, nil)
				mockQuerier.EXPECT().GetNetwork(gomock.Any(), networkID).Return(validNetwork, nil)
				mockQuerier.EXPECT().GetCustomersByWalletAddress(gomock.Any(), "0xcustomer123").Return([]db.Customer{validCustomer}, nil)
				mockQuerier.EXPECT().IsCustomerInWorkspace(gomock.Any(), gomock.Any()).Return(true, nil)
				mockQuerier.EXPECT().ListCustomerWallets(gomock.Any(), customerID).Return([]db.CustomerWallet{validCustomerWallet}, nil)
				mockQuerier.EXPECT().UpdateCustomerWalletUsageTime(gomock.Any(), customerWalletID).Return(validCustomerWallet, nil)
				mockQuerier.EXPECT().CreateDelegationData(gomock.Any(), gomock.Any()).Return(validDelegationData, nil)

				// Mock existing subscription check - returns existing subscription
				mockQuerier.EXPECT().ListSubscriptionsByCustomer(gomock.Any(), gomock.Any()).Return([]db.Subscription{validSubscription}, nil)
			},
			wantErr:     false,
			wantSuccess: false,
			errorString: "Subscription already exists for this customer and product",
		},
		{
			name: "handles caveats marshaling error",
			params: params.SubscribeToProductByPriceIDParams{
				ProductID:         productID,
				SubscriberAddress: "0xcustomer123",
				ProductTokenID:    productTokenID.String(),
				TokenAmount:       "1000000",
				DelegationData: params.DelegationParams{
					Delegate:  "0xcyphera123",
					Delegator: "0xcustomer123",
					Authority: "0xauthority123",
					Salt:      "0xsalt123",
					Signature: "0xsignature123",
					Caveats:   json.RawMessage(`{"invalid": make(chan int)}`), // Invalid JSON to trigger marshaling error
				},
			},
			setupMocks: func(mockQuerier *mocks.MockQuerier) {
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(gomock.Any(), productID).Return(validProduct, nil)
				mockQuerier.EXPECT().GetWalletByID(gomock.Any(), gomock.Any()).Return(validWallet, nil)
				mockQuerier.EXPECT().GetProductToken(gomock.Any(), productTokenID).Return(validProductToken, nil)
				mockQuerier.EXPECT().GetToken(gomock.Any(), tokenID).Return(validToken, nil)
				mockQuerier.EXPECT().GetNetwork(gomock.Any(), networkID).Return(validNetwork, nil)
			},
			wantErr:     true,
			errorString: "failed to marshal caveats",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQuerier := mocks.NewMockQuerier(ctrl)
			service := createSubscriptionService(ctrl, mockQuerier, nil)

			tt.setupMocks(mockQuerier)

			result, err := service.SubscribeToProductByPriceID(context.Background(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.wantSuccess, result.Success)
				if !tt.wantSuccess && tt.errorString != "" {
					assert.Equal(t, tt.errorString, result.ErrorMessage)
				}
				if tt.wantSuccess {
					assert.NotNil(t, result.Subscription)
				}
			}
		})
	}
}
