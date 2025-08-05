package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func init() {
	logger.InitLogger("test")
}

// stringPtr is a helper function to get a pointer to a string
func stringPtr(s string) *string {
	return &s
}

func TestPaymentService_CreatePaymentFromSubscriptionEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewPaymentService(mockQuerier, "test-api-key")
	ctx := context.Background()

	workspaceID := uuid.New()
	customerID := uuid.New()
	subscriptionID := uuid.New()
	eventID := uuid.New()
	productID := uuid.New()
	networkID := uuid.New()
	tokenID := uuid.New()

	tests := []struct {
		name       string
		params     params.CreatePaymentFromSubscriptionEventParams
		setupMocks func()
		wantErr    bool
		errString  string
	}{
		{
			name: "successful payment creation from redeemed event",
			params: params.CreatePaymentFromSubscriptionEventParams{
				SubscriptionEvent: &db.SubscriptionEvent{
					ID:            eventID,
					EventType:     db.SubscriptionEventTypeRedeem,
					AmountInCents: int32(1000),
				},
				Subscription: &db.Subscription{
					ID:          subscriptionID,
					WorkspaceID: workspaceID,
				},
				Product: &db.Product{
					ID:       productID,
					Currency: "USD",
				},
				Customer: &db.Customer{
					ID: customerID,
				},
				TransactionHash: "0x123abc",
				NetworkID:       networkID,
				TokenID:         tokenID,
				CryptoAmount:    "1.5",
				ExchangeRate:    "666.67",
				GasFeeUSDCents:  50,
				GasSponsored:    false,
			},
			setupMocks: func() {
				// Check for existing payment
				mockQuerier.EXPECT().
					GetPaymentBySubscriptionEvent(ctx, pgtype.UUID{Bytes: eventID, Valid: true}).
					Return(db.Payment{}, assert.AnError).
					Times(1)

				// Create payment
				mockQuerier.EXPECT().
					CreatePayment(ctx, gomock.Any()).
					Return(db.Payment{
						ID:            uuid.New(),
						WorkspaceID:   workspaceID,
						CustomerID:    customerID,
						AmountInCents: 1050, // 1000 + 50 gas fee
						Currency:      "usd",
						Status:        "completed",
					}, nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name: "payment already exists for subscription event",
			params: params.CreatePaymentFromSubscriptionEventParams{
				SubscriptionEvent: &db.SubscriptionEvent{
					ID:            eventID,
					EventType:     db.SubscriptionEventTypeRedeem,
					AmountInCents: int32(1000),
				},
				Subscription: &db.Subscription{
					ID:          subscriptionID,
					WorkspaceID: workspaceID,
				},
				Customer: &db.Customer{
					ID: customerID,
				},
			},
			setupMocks: func() {
				// Payment already exists
				mockQuerier.EXPECT().
					GetPaymentBySubscriptionEvent(ctx, pgtype.UUID{Bytes: eventID, Valid: true}).
					Return(db.Payment{
						ID:            uuid.New(),
						WorkspaceID:   workspaceID,
						CustomerID:    customerID,
						AmountInCents: 1000,
					}, nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name: "invalid event type",
			params: params.CreatePaymentFromSubscriptionEventParams{
				SubscriptionEvent: &db.SubscriptionEvent{
					ID:            eventID,
					EventType:     db.SubscriptionEventTypeCreate,
					AmountInCents: int32(1000),
				},
			},
			setupMocks: func() {},
			wantErr:    true,
			errString:  "can only create payments for redeemed subscription events",
		},
		{
			name: "database error during payment creation",
			params: params.CreatePaymentFromSubscriptionEventParams{
				SubscriptionEvent: &db.SubscriptionEvent{
					ID:            eventID,
					EventType:     db.SubscriptionEventTypeRedeem,
					AmountInCents: int32(1000),
				},
				Subscription: &db.Subscription{
					ID:          subscriptionID,
					WorkspaceID: workspaceID,
				},
				Product: &db.Product{
					ID:       productID,
					Currency: "USD",
				},
				Customer: &db.Customer{
					ID: customerID,
				},
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetPaymentBySubscriptionEvent(ctx, pgtype.UUID{Bytes: eventID, Valid: true}).
					Return(db.Payment{}, assert.AnError).
					Times(1)

				mockQuerier.EXPECT().
					CreatePayment(ctx, gomock.Any()).
					Return(db.Payment{}, assert.AnError).
					Times(1)
			},
			wantErr:   true,
			errString: "failed to create payment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.CreatePaymentFromSubscriptionEvent(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestPaymentService_GetPayment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewPaymentService(mockQuerier, "test-api-key")
	ctx := context.Background()

	paymentID := uuid.New()
	workspaceID := uuid.New()

	tests := []struct {
		name       string
		params     params.GetPaymentParams
		setupMocks func()
		wantErr    bool
		errString  string
	}{
		{
			name: "successful payment retrieval",
			params: params.GetPaymentParams{
				PaymentID:   paymentID,
				WorkspaceID: workspaceID,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetPayment(ctx, db.GetPaymentParams{
						ID:          paymentID,
						WorkspaceID: workspaceID,
					}).
					Return(db.Payment{
						ID:            paymentID,
						WorkspaceID:   workspaceID,
						AmountInCents: 1000,
						Status:        "completed",
					}, nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name: "payment not found",
			params: params.GetPaymentParams{
				PaymentID:   paymentID,
				WorkspaceID: workspaceID,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetPayment(ctx, db.GetPaymentParams{
						ID:          paymentID,
						WorkspaceID: workspaceID,
					}).
					Return(db.Payment{}, assert.AnError).
					Times(1)
			},
			wantErr:   true,
			errString: "payment not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.GetPayment(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, paymentID, result.ID)
			}
		})
	}
}

func TestPaymentService_GetPaymentByTransactionHash(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewPaymentService(mockQuerier, "test-api-key")
	ctx := context.Background()

	txHash := "0x123abc"
	paymentID := uuid.New()

	tests := []struct {
		name       string
		txHash     string
		setupMocks func()
		wantErr    bool
		errString  string
	}{
		{
			name:   "successful payment retrieval by tx hash",
			txHash: txHash,
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetPaymentByTransactionHash(ctx, pgtype.Text{String: txHash, Valid: true}).
					Return(db.Payment{
						ID:              paymentID,
						AmountInCents:   1000,
						TransactionHash: pgtype.Text{String: txHash, Valid: true},
					}, nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name:       "empty transaction hash",
			txHash:     "",
			setupMocks: func() {},
			wantErr:    true,
			errString:  "transaction hash is required",
		},
		{
			name:   "payment not found by tx hash",
			txHash: txHash,
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetPaymentByTransactionHash(ctx, pgtype.Text{String: txHash, Valid: true}).
					Return(db.Payment{}, assert.AnError).
					Times(1)
			},
			wantErr:   true,
			errString: "payment not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.GetPaymentByTransactionHash(ctx, tt.txHash)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, txHash, result.TransactionHash.String)
			}
		})
	}
}

func TestPaymentService_ListPayments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewPaymentService(mockQuerier, "test-api-key")
	ctx := context.Background()

	workspaceID := uuid.New()
	customerID := uuid.New()

	tests := []struct {
		name       string
		params     params.ListPaymentsParams
		setupMocks func()
		wantErr    bool
		errString  string
	}{
		{
			name: "successful payment list by workspace",
			params: params.ListPaymentsParams{
				WorkspaceID: workspaceID,
				Limit:       10,
				Offset:      0,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetPaymentsByWorkspace(ctx, db.GetPaymentsByWorkspaceParams{
						WorkspaceID: workspaceID,
						Limit:       10,
						Offset:      0,
					}).
					Return([]db.Payment{
						{ID: uuid.New(), WorkspaceID: workspaceID},
						{ID: uuid.New(), WorkspaceID: workspaceID},
					}, nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name: "successful payment list by customer",
			params: params.ListPaymentsParams{
				WorkspaceID: workspaceID,
				CustomerID:  &customerID,
				Limit:       20,
				Offset:      10,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetPaymentsByCustomer(ctx, db.GetPaymentsByCustomerParams{
						CustomerID:  customerID,
						WorkspaceID: workspaceID,
						Limit:       20,
						Offset:      10,
					}).
					Return([]db.Payment{
						{ID: uuid.New(), CustomerID: customerID},
					}, nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name: "successful payment list by status",
			params: params.ListPaymentsParams{
				WorkspaceID: workspaceID,
				Status:      stringPtr("completed"),
				Limit:       5,
				Offset:      0,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetPaymentsByStatus(ctx, db.GetPaymentsByStatusParams{
						WorkspaceID: workspaceID,
						Status:      "completed",
						Limit:       5,
						Offset:      0,
					}).
					Return([]db.Payment{
						{ID: uuid.New(), Status: "completed"},
					}, nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name: "default limit applied when zero",
			params: params.ListPaymentsParams{
				WorkspaceID: workspaceID,
				Limit:       0,
				Offset:      0,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetPaymentsByWorkspace(ctx, db.GetPaymentsByWorkspaceParams{
						WorkspaceID: workspaceID,
						Limit:       20,
						Offset:      0,
					}).
					Return([]db.Payment{}, nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name: "max limit applied when exceeded",
			params: params.ListPaymentsParams{
				WorkspaceID: workspaceID,
				Limit:       150,
				Offset:      0,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetPaymentsByWorkspace(ctx, db.GetPaymentsByWorkspaceParams{
						WorkspaceID: workspaceID,
						Limit:       100,
						Offset:      0,
					}).
					Return([]db.Payment{}, nil).
					Times(1)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.ListPayments(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestPaymentService_UpdatePaymentStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewPaymentService(mockQuerier, "test-api-key")
	ctx := context.Background()

	paymentID := uuid.New()
	workspaceID := uuid.New()

	tests := []struct {
		name       string
		params     params.UpdatePaymentStatusParams
		setupMocks func()
		wantErr    bool
		errString  string
	}{
		{
			name: "successful status update",
			params: params.UpdatePaymentStatusParams{
				PaymentID:   paymentID,
				WorkspaceID: workspaceID,
				Status:      "completed",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					UpdatePaymentStatus(ctx, db.UpdatePaymentStatusParams{
						ID:           paymentID,
						WorkspaceID:  workspaceID,
						Status:       "completed",
						ErrorMessage: pgtype.Text{String: "", Valid: false},
					}).
					Return(db.Payment{
						ID:          paymentID,
						WorkspaceID: workspaceID,
						Status:      "completed",
					}, nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name: "successful status update with error message",
			params: params.UpdatePaymentStatusParams{
				PaymentID:     paymentID,
				WorkspaceID:   workspaceID,
				Status:        "failed",
				FailureReason: stringPtr("Transaction failed"),
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					UpdatePaymentStatus(ctx, db.UpdatePaymentStatusParams{
						ID:           paymentID,
						WorkspaceID:  workspaceID,
						Status:       "failed",
						ErrorMessage: pgtype.Text{String: "Transaction failed", Valid: true},
					}).
					Return(db.Payment{
						ID:          paymentID,
						WorkspaceID: workspaceID,
						Status:      "failed",
					}, nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name: "database error during update",
			params: params.UpdatePaymentStatusParams{
				PaymentID:   paymentID,
				WorkspaceID: workspaceID,
				Status:      "completed",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					UpdatePaymentStatus(ctx, gomock.Any()).
					Return(db.Payment{}, assert.AnError).
					Times(1)
			},
			wantErr:   true,
			errString: "failed to update payment status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.UpdatePaymentStatus(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.params.Status, result.Status)
			}
		})
	}
}

func TestPaymentService_GetPaymentMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewPaymentService(mockQuerier, "test-api-key")
	ctx := context.Background()

	workspaceID := uuid.New()
	startTime := time.Now().AddDate(0, 0, -30)
	endTime := time.Now()

	tests := []struct {
		name       string
		setupMocks func()
		wantErr    bool
		errString  string
	}{
		{
			name: "successful metrics retrieval",
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetPaymentMetrics(ctx, db.GetPaymentMetricsParams{
						WorkspaceID: workspaceID,
						CreatedAt:   pgtype.Timestamptz{Time: startTime, Valid: true},
						CreatedAt_2: pgtype.Timestamptz{Time: endTime, Valid: true},
						Currency:    "USD",
					}).
					Return(db.GetPaymentMetricsRow{
						CompletedCount:        10,
						FailedCount:           2,
						PendingCount:          1,
						TotalCompletedCents:   50000,
						TotalGasFeesCents:     500,
						SponsoredGasFeesCents: 200,
					}, nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name: "database error during metrics retrieval",
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetPaymentMetrics(ctx, gomock.Any()).
					Return(db.GetPaymentMetricsRow{}, assert.AnError).
					Times(1)
			},
			wantErr:   true,
			errString: "failed to get payment metrics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.GetPaymentMetrics(ctx, workspaceID, startTime, endTime, "USD")

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestPaymentService_CreateManualPayment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewPaymentService(mockQuerier, "test-api-key")
	ctx := context.Background()

	workspaceID := uuid.New()
	customerID := uuid.New()
	subscriptionID := uuid.New()

	tests := []struct {
		name       string
		params     params.CreateManualPaymentParams
		setupMocks func()
		wantErr    bool
		errString  string
	}{
		{
			name: "successful manual payment creation",
			params: params.CreateManualPaymentParams{
				WorkspaceID:       workspaceID,
				CustomerID:        customerID,
				SubscriptionID:    &subscriptionID,
				AmountInCents:     2000,
				Currency:          "USD",
				PaymentMethod:     "crypto",
				TransactionHash:   "0x456def",
				ExternalPaymentID: "ext_123",
				PaymentProvider:   "circle",
				Metadata:          map[string]interface{}{"source": "manual"},
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					CreatePayment(ctx, gomock.Any()).
					Return(db.Payment{
						ID:            uuid.New(),
						WorkspaceID:   workspaceID,
						CustomerID:    customerID,
						AmountInCents: 2000,
						Currency:      "USD",
						Status:        "completed",
					}, nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name: "database error during manual payment creation",
			params: params.CreateManualPaymentParams{
				WorkspaceID:   workspaceID,
				CustomerID:    customerID,
				AmountInCents: 2000,
				Currency:      "USD",
				PaymentMethod: "crypto",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					CreatePayment(ctx, gomock.Any()).
					Return(db.Payment{}, assert.AnError).
					Times(1)
			},
			wantErr:   true,
			errString: "failed to create payment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.CreateManualPayment(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.params.AmountInCents, result.AmountInCents)
			}
		})
	}
}

func TestPaymentService_CreateComprehensivePayment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewPaymentService(mockQuerier, "test-api-key")
	ctx := context.Background()

	workspaceID := uuid.New()
	customerID := uuid.New()
	productID := uuid.New()
	subscriptionID := uuid.New()
	invoiceID := uuid.New()
	networkID := uuid.New()
	tokenID := uuid.New()
	paymentID := uuid.New()

	tests := []struct {
		name           string
		params         params.CreateComprehensivePaymentParams
		setupMocks     func()
		wantErr        bool
		errorString    string
		validateResult func(*db.Payment)
	}{
		{
			name: "successfully creates comprehensive payment with minimal parameters",
			params: params.CreateComprehensivePaymentParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				AmountCents:     1000,
				Currency:        "USD",
				PaymentMethod:   "crypto",
				ProductType:     "digital",
				TransactionType: "transfer",
			},
			setupMocks: func() {
				// Mock successful payment creation
				mockQuerier.EXPECT().CreatePayment(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error) {
						// Validate the basic parameters
						assert.Equal(t, workspaceID, params.WorkspaceID)
						assert.Equal(t, customerID, params.CustomerID)
						assert.Equal(t, int64(1000), params.AmountInCents)
						assert.Equal(t, "USD", params.Currency)
						assert.Equal(t, "pending", params.Status)
						assert.Equal(t, "crypto", params.PaymentMethod)
						assert.Equal(t, int64(1000), params.ProductAmountCents)

						return db.Payment{
							ID:                 paymentID,
							WorkspaceID:        workspaceID,
							CustomerID:         customerID,
							AmountInCents:      1000,
							Currency:           "USD",
							Status:             "pending",
							PaymentMethod:      "crypto",
							ProductAmountCents: 1000,
							CreatedAt:          pgtype.Timestamptz{Time: time.Now(), Valid: true},
							UpdatedAt:          pgtype.Timestamptz{Time: time.Now(), Valid: true},
						}, nil
					})
			},
			wantErr: false,
			validateResult: func(payment *db.Payment) {
				assert.NotNil(t, payment)
				assert.Equal(t, paymentID, payment.ID)
				assert.Equal(t, workspaceID, payment.WorkspaceID)
				assert.Equal(t, customerID, payment.CustomerID)
				assert.Equal(t, int64(1000), payment.AmountInCents)
				assert.Equal(t, "USD", payment.Currency)
				assert.Equal(t, "pending", payment.Status)
				assert.Equal(t, "crypto", payment.PaymentMethod)
			},
		},
		{
			name: "successfully creates comprehensive payment with discount code",
			params: params.CreateComprehensivePaymentParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				ProductID:       &productID,
				AmountCents:     1000,
				Currency:        "USD",
				PaymentMethod:   "crypto",
				ProductType:     "digital",
				TransactionType: "transfer",
				DiscountCode:    &[]string{"SAVE10"}[0],
			},
			setupMocks: func() {
				// Mock customer lookup for discount validation
				mockQuerier.EXPECT().GetCustomer(ctx, customerID).Return(db.Customer{
					ID:        customerID,
					Email:     pgtype.Text{String: "test@example.com", Valid: true},
					CreatedAt: pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -30), Valid: true}, // 30 days old
				}, nil)

				// Mock successful payment creation with discount applied
				mockQuerier.EXPECT().CreatePayment(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error) {
						// Amount should remain 1000 since discount service is mocked to fail
						assert.Equal(t, int64(1000), params.AmountInCents)

						return db.Payment{
							ID:                 paymentID,
							WorkspaceID:        workspaceID,
							CustomerID:         customerID,
							AmountInCents:      1000,
							Currency:           "USD",
							Status:             "pending",
							PaymentMethod:      "crypto",
							ProductAmountCents: 1000,
							CreatedAt:          pgtype.Timestamptz{Time: time.Now(), Valid: true},
							UpdatedAt:          pgtype.Timestamptz{Time: time.Now(), Valid: true},
						}, nil
					})
			},
			wantErr: false,
			validateResult: func(payment *db.Payment) {
				assert.NotNil(t, payment)
				assert.Equal(t, paymentID, payment.ID)
			},
		},
		{
			name: "successfully creates comprehensive crypto payment with gas estimation",
			params: params.CreateComprehensivePaymentParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				ProductID:       &productID,
				AmountCents:     1000,
				Currency:        "USD",
				PaymentMethod:   "crypto",
				ProductType:     "digital",
				TransactionType: "transfer",
				NetworkID:       &networkID,
				TokenID:         &tokenID,
				TransactionHash: &[]string{"0x123abc"}[0],
				GasFeeUSDCents:  &[]int64{100}[0],
				CryptoAmount:    &[]string{"0.5"}[0],
			},
			setupMocks: func() {
				// Mock the network retrieval for gas estimation
				mockQuerier.EXPECT().GetNetwork(ctx, networkID).Return(db.Network{
					ID:      networkID,
					Name:    "Ethereum",
					ChainID: 1,
				}, nil).AnyTimes()

				// Mock gas sponsorship config call
				mockQuerier.EXPECT().GetGasSponsorshipConfig(ctx, workspaceID).Return(db.GasSponsorshipConfig{
					ID:                       uuid.New(),
					WorkspaceID:              workspaceID,
					SponsorshipEnabled:       pgtype.Bool{Bool: true, Valid: true},
					SponsorCustomerGas:       pgtype.Bool{Bool: true, Valid: true},
					SponsorThresholdUsdCents: pgtype.Int8{Int64: 1000, Valid: true},
					MonthlyBudgetUsdCents:    pgtype.Int8{Int64: 10000, Valid: true},
					SponsorForProducts:       []byte("[]"),
					SponsorForCustomers:      []byte("[]"),
					SponsorForTiers:          []byte("[]"),
					CurrentMonthSpentCents:   pgtype.Int8{Int64: 0, Valid: true},
				}, nil).AnyTimes()

				// Mock gas sponsorship spending update
				mockQuerier.EXPECT().UpdateGasSponsorshipSpending(ctx, gomock.Any()).Return(nil).AnyTimes()

				// Mock successful payment creation with crypto parameters
				mockQuerier.EXPECT().CreatePayment(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error) {
						assert.Equal(t, workspaceID, params.WorkspaceID)
						assert.Equal(t, customerID, params.CustomerID)
						assert.Equal(t, int64(1000), params.AmountInCents)
						assert.Equal(t, "USD", params.Currency)
						assert.Equal(t, "crypto", params.PaymentMethod)

						// Check crypto-specific fields
						assert.True(t, params.TransactionHash.Valid)
						assert.Equal(t, "0x123abc", params.TransactionHash.String)
						assert.True(t, params.NetworkID.Valid)
						assert.Equal(t, networkID, uuid.UUID(params.NetworkID.Bytes))
						assert.True(t, params.TokenID.Valid)
						assert.Equal(t, tokenID, uuid.UUID(params.TokenID.Bytes))

						return db.Payment{
							ID:              paymentID,
							WorkspaceID:     workspaceID,
							CustomerID:      customerID,
							AmountInCents:   1000,
							Currency:        "USD",
							Status:          "pending",
							PaymentMethod:   "crypto",
							TransactionHash: pgtype.Text{String: "0x123abc", Valid: true},
							NetworkID:       pgtype.UUID{Bytes: networkID, Valid: true},
							TokenID:         pgtype.UUID{Bytes: tokenID, Valid: true},
							CreatedAt:       pgtype.Timestamptz{Time: time.Now(), Valid: true},
							UpdatedAt:       pgtype.Timestamptz{Time: time.Now(), Valid: true},
						}, nil
					})
			},
			wantErr: false,
			validateResult: func(payment *db.Payment) {
				assert.NotNil(t, payment)
				assert.Equal(t, paymentID, payment.ID)
				assert.True(t, payment.TransactionHash.Valid)
				assert.Equal(t, "0x123abc", payment.TransactionHash.String)
				assert.True(t, payment.NetworkID.Valid)
				assert.Equal(t, networkID, uuid.UUID(payment.NetworkID.Bytes))
			},
		},
		{
			name: "successfully creates comprehensive payment with all optional fields",
			params: params.CreateComprehensivePaymentParams{
				WorkspaceID:       workspaceID,
				CustomerID:        customerID,
				InvoiceID:         &invoiceID,
				SubscriptionID:    &subscriptionID,
				ProductID:         &productID,
				AmountCents:       1000,
				Currency:          "USD",
				PaymentMethod:     "crypto",
				ProductType:       "digital",
				TransactionType:   "transfer",
				NetworkID:         &networkID,
				TokenID:           &tokenID,
				TransactionHash:   &[]string{"0x456def"}[0],
				GasFeeUSDCents:    &[]int64{150}[0],
				CryptoAmount:      &[]string{"0.75"}[0],
				DiscountCode:      &[]string{"WELCOME20"}[0],
				CustomerVATNumber: &[]string{"GB123456789"}[0],
				IsB2B:             true,
				CustomerAddress: &params.PaymentAddress{
					Street1:    "789 Customer Rd",
					City:       "London",
					State:      "ENG",
					PostalCode: "SW1A 1AA",
					Country:    "GB",
				},
				BusinessAddress: &params.PaymentAddress{
					Street1:    "101 Business St",
					City:       "Manchester",
					State:      "ENG",
					PostalCode: "M1 1AA",
					Country:    "GB",
				},
			},
			setupMocks: func() {
				// Mock customer lookup for discount validation
				mockQuerier.EXPECT().GetCustomer(ctx, customerID).Return(db.Customer{
					ID:        customerID,
					Email:     pgtype.Text{String: "business@example.com", Valid: true},
					CreatedAt: pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -1), Valid: true}, // 1 day old (new customer)
				}, nil)

				// Mock successful payment creation with all fields
				mockQuerier.EXPECT().CreatePayment(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error) {
						// Validate all fields are set
						assert.Equal(t, workspaceID, params.WorkspaceID)
						assert.Equal(t, customerID, params.CustomerID)

						// Check optional UUID fields
						assert.True(t, params.InvoiceID.Valid)
						assert.Equal(t, invoiceID, uuid.UUID(params.InvoiceID.Bytes))
						assert.True(t, params.SubscriptionID.Valid)
						assert.Equal(t, subscriptionID, uuid.UUID(params.SubscriptionID.Bytes))
						assert.True(t, params.NetworkID.Valid)
						assert.Equal(t, networkID, uuid.UUID(params.NetworkID.Bytes))
						assert.True(t, params.TokenID.Valid)
						assert.Equal(t, tokenID, uuid.UUID(params.TokenID.Bytes))

						// Check transaction hash
						assert.True(t, params.TransactionHash.Valid)
						assert.Equal(t, "0x456def", params.TransactionHash.String)

						return db.Payment{
							ID:              paymentID,
							WorkspaceID:     workspaceID,
							CustomerID:      customerID,
							InvoiceID:       pgtype.UUID{Bytes: invoiceID, Valid: true},
							SubscriptionID:  pgtype.UUID{Bytes: subscriptionID, Valid: true},
							AmountInCents:   1000,
							Currency:        "USD",
							Status:          "pending",
							PaymentMethod:   "crypto",
							TransactionHash: pgtype.Text{String: "0x456def", Valid: true},
							NetworkID:       pgtype.UUID{Bytes: networkID, Valid: true},
							TokenID:         pgtype.UUID{Bytes: tokenID, Valid: true},
							CreatedAt:       pgtype.Timestamptz{Time: time.Now(), Valid: true},
							UpdatedAt:       pgtype.Timestamptz{Time: time.Now(), Valid: true},
						}, nil
					})
			},
			wantErr: false,
			validateResult: func(payment *db.Payment) {
				assert.NotNil(t, payment)
				assert.Equal(t, paymentID, payment.ID)
				assert.True(t, payment.InvoiceID.Valid)
				assert.Equal(t, invoiceID, uuid.UUID(payment.InvoiceID.Bytes))
				assert.True(t, payment.SubscriptionID.Valid)
				assert.Equal(t, subscriptionID, uuid.UUID(payment.SubscriptionID.Bytes))
			},
		},
		{
			name: "fails when customer lookup for discount validation fails",
			params: params.CreateComprehensivePaymentParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				AmountCents:     1000,
				Currency:        "USD",
				PaymentMethod:   "crypto",
				ProductType:     "digital",
				TransactionType: "transfer",
				DiscountCode:    &[]string{"INVALID"}[0],
			},
			setupMocks: func() {
				// Mock customer lookup failure
				mockQuerier.EXPECT().GetCustomer(ctx, customerID).Return(db.Customer{}, assert.AnError)
			},
			wantErr:     true,
			errorString: "failed to get customer for discount validation",
		},
		{
			name: "fails when payment creation fails",
			params: params.CreateComprehensivePaymentParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				AmountCents:     1000,
				Currency:        "USD",
				PaymentMethod:   "crypto",
				ProductType:     "digital",
				TransactionType: "transfer",
			},
			setupMocks: func() {
				// Mock payment creation failure
				mockQuerier.EXPECT().CreatePayment(ctx, gomock.Any()).Return(db.Payment{}, assert.AnError)
			},
			wantErr:     true,
			errorString: "failed to create payment",
		},
		{
			name: "handles zero amount gracefully",
			params: params.CreateComprehensivePaymentParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				AmountCents:     0,
				Currency:        "USD",
				PaymentMethod:   "crypto",
				ProductType:     "digital",
				TransactionType: "transfer",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().CreatePayment(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error) {
						assert.Equal(t, int64(0), params.AmountInCents)
						assert.Equal(t, int64(0), params.ProductAmountCents)

						return db.Payment{
							ID:                 paymentID,
							WorkspaceID:        workspaceID,
							CustomerID:         customerID,
							AmountInCents:      0,
							Currency:           "USD",
							Status:             "pending",
							PaymentMethod:      "crypto",
							ProductAmountCents: 0,
							CreatedAt:          pgtype.Timestamptz{Time: time.Now(), Valid: true},
							UpdatedAt:          pgtype.Timestamptz{Time: time.Now(), Valid: true},
						}, nil
					})
			},
			wantErr: false,
			validateResult: func(payment *db.Payment) {
				assert.NotNil(t, payment)
				assert.Equal(t, int64(0), payment.AmountInCents)
			},
		},
		{
			name: "handles edge case with very large amount",
			params: params.CreateComprehensivePaymentParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				AmountCents:     999999999999, // Very large amount
				Currency:        "USD",
				PaymentMethod:   "crypto",
				ProductType:     "digital",
				TransactionType: "transfer",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().CreatePayment(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error) {
						assert.Equal(t, int64(999999999999), params.AmountInCents)

						return db.Payment{
							ID:                 paymentID,
							WorkspaceID:        workspaceID,
							CustomerID:         customerID,
							AmountInCents:      999999999999,
							Currency:           "USD",
							Status:             "pending",
							PaymentMethod:      "crypto",
							ProductAmountCents: 999999999999,
							CreatedAt:          pgtype.Timestamptz{Time: time.Now(), Valid: true},
							UpdatedAt:          pgtype.Timestamptz{Time: time.Now(), Valid: true},
						}, nil
					})
			},
			wantErr: false,
			validateResult: func(payment *db.Payment) {
				assert.NotNil(t, payment)
				assert.Equal(t, int64(999999999999), payment.AmountInCents)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.CreateComprehensivePayment(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Nil(t, result)
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
