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
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func init() {
	logger.InitLogger("test")
}

func TestSubscriptionService_GetSubscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockDelegationClient := &dsClient.DelegationClient{} // Mock client
	// Create a real PaymentService since constructor expects concrete type
	paymentService := services.NewPaymentService(mockQuerier, "test-api-key")
	service := services.NewSubscriptionService(mockQuerier, mockDelegationClient, paymentService)
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
				}).Return(db.Subscription{}, pgx.ErrNoRows)
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
				}).Return(db.Subscription{}, pgx.ErrNoRows)
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
	// Create a real PaymentService since constructor expects concrete type
	paymentService := services.NewPaymentService(mockQuerier, "test-api-key")
	service := services.NewSubscriptionService(mockQuerier, mockDelegationClient, paymentService)
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
			PriceID:                        uuid.New(),
			PriceProductID:                 productID,
			PriceActive:                    true,
			PriceType:                      "recurring",
			PriceCurrency:                  "USD",
			PriceUnitAmountInPennies:       1000,
			PriceIntervalType:              "month",
			PriceTermLength:                1,
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
			PriceID:                        uuid.New(),
			PriceProductID:                 productID,
			PriceActive:                    true,
			PriceType:                      "recurring",
			PriceCurrency:                  "USD",
			PriceUnitAmountInPennies:       2000,
			PriceIntervalType:              "month",
			PriceTermLength:                1,
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
	// Create a real PaymentService since constructor expects concrete type
	paymentService := services.NewPaymentService(mockQuerier, "test-api-key")
	service := services.NewSubscriptionService(mockQuerier, mockDelegationClient, paymentService)
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
	// Create a real PaymentService since constructor expects concrete type
	paymentService := services.NewPaymentService(mockQuerier, "test-api-key")
	service := services.NewSubscriptionService(mockQuerier, mockDelegationClient, paymentService)
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
	// Create a real PaymentService since constructor expects concrete type
	paymentService := services.NewPaymentService(mockQuerier, "test-api-key")
	service := services.NewSubscriptionService(mockQuerier, mockDelegationClient, paymentService)
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
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(db.Subscription{}, pgx.ErrNoRows)
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
	// Create a real PaymentService since constructor expects concrete type
	paymentService := services.NewPaymentService(mockQuerier, "test-api-key")
	service := services.NewSubscriptionService(mockQuerier, mockDelegationClient, paymentService)
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
				}).Return(db.Subscription{}, pgx.ErrNoRows)
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
	// Create a real PaymentService since constructor expects concrete type
	paymentService := services.NewPaymentService(mockQuerier, "test-api-key")
	service := services.NewSubscriptionService(mockQuerier, mockDelegationClient, paymentService)

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
				_, err := service.GetSubscription(nil, uuid.New(), uuid.New())
				return err
			},
			expectError: true,
		},
		{
			name: "zero UUID handling in GetSubscription",
			operation: func() error {
				mockQuerier.EXPECT().GetSubscriptionWithWorkspace(gomock.Any(), gomock.Any()).Return(db.Subscription{}, pgx.ErrNoRows)
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
	// Create a real PaymentService since constructor expects concrete type
	paymentService := services.NewPaymentService(mockQuerier, "test-api-key")
	service := services.NewSubscriptionService(mockQuerier, mockDelegationClient, paymentService)
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
	// Create a real PaymentService since constructor expects concrete type
	paymentService := services.NewPaymentService(mockQuerier, "test-api-key")
	service := services.NewSubscriptionService(mockQuerier, mockDelegationClient, paymentService)
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
