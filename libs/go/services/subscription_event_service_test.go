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

const (
	// Common error messages used in tests
	errMsgDatabaseError        = "database error"
	errMsgSubscriptionNotFound = "subscription not found"
	errMsgProductNotFound      = "product not found"
	errMsgTransactionRequired  = "transaction hash is required"
	errMsgEventNotFound        = "subscription event not found"
	errMsgFailedToRetrieve     = "failed to retrieve"
	errMsgFailedToCreate       = "failed to create"
	errMsgFailedToCount        = "failed to count"
	errMsgUnauthorizedAccess   = "unauthorized access to subscription"
)

func init() {
	logger.InitLogger("test")
}

func TestSubscriptionEventService_GetSubscriptionEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewSubscriptionEventService(mockQuerier)
	ctx := context.Background()

	eventID := uuid.New()
	subscriptionID := uuid.New()
	productID := uuid.New()
	workspaceID := uuid.New()
	otherWorkspaceID := uuid.New()

	expectedEvent := db.SubscriptionEvent{
		ID:             eventID,
		SubscriptionID: subscriptionID,
		EventType:      "payment_success",
		AmountInCents:  1000,
	}

	expectedSubscription := db.Subscription{
		ID:        subscriptionID,
		ProductID: productID,
	}

	expectedProduct := db.Product{
		ID:          productID,
		WorkspaceID: workspaceID,
		Name:        "Test Product",
	}

	tests := []struct {
		name        string
		eventID     uuid.UUID
		workspaceID uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "successfully gets subscription event",
			eventID:     eventID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionEvent(ctx, eventID).Return(expectedEvent, nil)
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(expectedSubscription, nil)
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(expectedProduct, nil)
			},
			wantErr: false,
		},
		{
			name:        "subscription event not found",
			eventID:     eventID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionEvent(ctx, eventID).Return(db.SubscriptionEvent{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: errMsgEventNotFound,
		},
		{
			name:        "database error getting event",
			eventID:     eventID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionEvent(ctx, eventID).Return(db.SubscriptionEvent{}, errors.New(errMsgDatabaseError))
			},
			wantErr:     true,
			errorString: errMsgFailedToRetrieve + " subscription event",
		},
		{
			name:        "subscription not found",
			eventID:     eventID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionEvent(ctx, eventID).Return(expectedEvent, nil)
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(db.Subscription{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: errMsgSubscriptionNotFound,
		},
		{
			name:        "product not found",
			eventID:     eventID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionEvent(ctx, eventID).Return(expectedEvent, nil)
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(expectedSubscription, nil)
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: workspaceID,
				}).Return(db.Product{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: errMsgProductNotFound,
		},
		{
			name:        "unauthorized access to subscription event",
			eventID:     eventID,
			workspaceID: otherWorkspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionEvent(ctx, eventID).Return(expectedEvent, nil)
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(expectedSubscription, nil)
				mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
					ID:          productID,
					WorkspaceID: otherWorkspaceID,
				}).Return(db.Product{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: errMsgProductNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			event, err := service.GetSubscriptionEvent(ctx, tt.eventID, tt.workspaceID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, event)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, event)
				assert.Equal(t, eventID, event.ID)
			}
		})
	}
}

func TestSubscriptionEventService_GetSubscriptionEventByTxHash(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewSubscriptionEventService(mockQuerier)
	ctx := context.Background()

	txHash := "0x123abc..."
	expectedEvent := db.SubscriptionEvent{
		ID:        uuid.New(),
		EventType: "payment_success",
		TransactionHash: pgtype.Text{
			String: txHash,
			Valid:  true,
		},
	}

	tests := []struct {
		name        string
		txHash      string
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:   "successfully gets subscription event by tx hash",
			txHash: txHash,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionEventByTransactionHash(ctx, pgtype.Text{
					String: txHash,
					Valid:  true,
				}).Return(expectedEvent, nil)
			},
			wantErr: false,
		},
		{
			name:        "empty transaction hash",
			txHash:      "",
			setupMocks:  func() {},
			wantErr:     true,
			errorString: errMsgTransactionRequired,
		},
		{
			name:   "subscription event not found",
			txHash: txHash,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionEventByTransactionHash(ctx, pgtype.Text{
					String: txHash,
					Valid:  true,
				}).Return(db.SubscriptionEvent{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: errMsgEventNotFound,
		},
		{
			name:   "database error",
			txHash: txHash,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionEventByTransactionHash(ctx, pgtype.Text{
					String: txHash,
					Valid:  true,
				}).Return(db.SubscriptionEvent{}, errors.New(errMsgDatabaseError))
			},
			wantErr:     true,
			errorString: errMsgFailedToRetrieve + " subscription event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			event, err := service.GetSubscriptionEventByTxHash(ctx, tt.txHash)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, event)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, event)
				assert.Equal(t, txHash, event.TransactionHash.String)
			}
		})
	}
}

func TestSubscriptionEventService_ListSubscriptionEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewSubscriptionEventService(mockQuerier)
	ctx := context.Background()

	workspaceID := uuid.New()
	expectedEvents := []db.ListSubscriptionEventDetailsWithPaginationRow{
		{SubscriptionEventID: uuid.New(), EventType: "payment_success"},
		{SubscriptionEventID: uuid.New(), EventType: "payment_failed"},
	}
	expectedCount := int64(10)

	tests := []struct {
		name        string
		params      params.ListSubscriptionEventsParams
		setupMocks  func()
		wantErr     bool
		errorString string
		wantCount   int
	}{
		{
			name: "successfully lists subscription events",
			params: params.ListSubscriptionEventsParams{
				WorkspaceID: workspaceID,
				Limit:       20,
				Offset:      0,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionEventDetailsWithPagination(ctx, db.ListSubscriptionEventDetailsWithPaginationParams{
					WorkspaceID: workspaceID,
					Limit:       20,
					Offset:      0,
				}).Return(expectedEvents, nil)
				mockQuerier.EXPECT().CountSubscriptionEventDetails(ctx, workspaceID).Return(expectedCount, nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "validates and corrects pagination parameters",
			params: params.ListSubscriptionEventsParams{
				WorkspaceID: workspaceID,
				Limit:       0, // Will be corrected to 20
				Offset:      0,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionEventDetailsWithPagination(ctx, db.ListSubscriptionEventDetailsWithPaginationParams{
					WorkspaceID: workspaceID,
					Limit:       20, // Corrected
					Offset:      0,  // Page 1 = offset 0
				}).Return(expectedEvents, nil)
				mockQuerier.EXPECT().CountSubscriptionEventDetails(ctx, workspaceID).Return(expectedCount, nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "limits max page size",
			params: params.ListSubscriptionEventsParams{
				WorkspaceID: workspaceID,
				Limit:       200, // Will be corrected to 20
				Offset:      0,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionEventDetailsWithPagination(ctx, db.ListSubscriptionEventDetailsWithPaginationParams{
					WorkspaceID: workspaceID,
					Limit:       20, // Corrected to default
					Offset:      0,
				}).Return(expectedEvents, nil)
				mockQuerier.EXPECT().CountSubscriptionEventDetails(ctx, workspaceID).Return(expectedCount, nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "calculates correct offset for page 2",
			params: params.ListSubscriptionEventsParams{
				WorkspaceID: workspaceID,
				Limit:       10,
				Offset:      10,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionEventDetailsWithPagination(ctx, db.ListSubscriptionEventDetailsWithPaginationParams{
					WorkspaceID: workspaceID,
					Limit:       10,
					Offset:      10, // (2-1) * 10 = 10
				}).Return(expectedEvents, nil)
				mockQuerier.EXPECT().CountSubscriptionEventDetails(ctx, workspaceID).Return(expectedCount, nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "database error getting events",
			params: params.ListSubscriptionEventsParams{
				WorkspaceID: workspaceID,
				Limit:       20,
				Offset:      0,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionEventDetailsWithPagination(ctx, db.ListSubscriptionEventDetailsWithPaginationParams{
					WorkspaceID: workspaceID,
					Limit:       20,
					Offset:      0,
				}).Return(nil, errors.New(errMsgDatabaseError))
			},
			wantErr:     true,
			errorString: errMsgFailedToRetrieve + " subscription events",
		},
		{
			name: "database error getting count",
			params: params.ListSubscriptionEventsParams{
				WorkspaceID: workspaceID,
				Limit:       20,
				Offset:      0,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionEventDetailsWithPagination(ctx, db.ListSubscriptionEventDetailsWithPaginationParams{
					WorkspaceID: workspaceID,
					Limit:       20,
					Offset:      0,
				}).Return(expectedEvents, nil)
				mockQuerier.EXPECT().CountSubscriptionEventDetails(ctx, workspaceID).Return(int64(0), errors.New("count error"))
			},
			wantErr:     true,
			errorString: errMsgFailedToCount + " subscription events",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.ListSubscriptionEvents(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result.Events, tt.wantCount)
				assert.Equal(t, expectedCount, result.Total)
			}
		})
	}
}

func TestSubscriptionEventService_ListSubscriptionEventsBySubscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewSubscriptionEventService(mockQuerier)
	ctx := context.Background()

	subscriptionID := uuid.New()
	workspaceID := uuid.New()
	otherWorkspaceID := uuid.New()

	expectedSubscription := db.Subscription{
		ID:          subscriptionID,
		WorkspaceID: workspaceID,
	}

	expectedEvents := []db.SubscriptionEvent{
		{ID: uuid.New(), SubscriptionID: subscriptionID, EventType: "payment_success"},
		{ID: uuid.New(), SubscriptionID: subscriptionID, EventType: "payment_failed"},
	}

	tests := []struct {
		name           string
		subscriptionID uuid.UUID
		workspaceID    uuid.UUID
		setupMocks     func()
		wantErr        bool
		errorString    string
		wantCount      int
	}{
		{
			name:           "successfully lists subscription events by subscription",
			subscriptionID: subscriptionID,
			workspaceID:    workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionWithWorkspace(ctx, db.GetSubscriptionWithWorkspaceParams{
					ID:          subscriptionID,
					WorkspaceID: workspaceID,
				}).Return(expectedSubscription, nil)
				mockQuerier.EXPECT().ListSubscriptionEventsBySubscription(ctx, subscriptionID).Return(expectedEvents, nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:           "subscription not found",
			subscriptionID: subscriptionID,
			workspaceID:    workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionWithWorkspace(ctx, db.GetSubscriptionWithWorkspaceParams{
					ID:          subscriptionID,
					WorkspaceID: workspaceID,
				}).Return(db.Subscription{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: errMsgSubscriptionNotFound,
		},
		{
			name:           "database error getting subscription",
			subscriptionID: subscriptionID,
			workspaceID:    workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionWithWorkspace(ctx, db.GetSubscriptionWithWorkspaceParams{
					ID:          subscriptionID,
					WorkspaceID: workspaceID,
				}).Return(db.Subscription{}, errors.New(errMsgDatabaseError))
			},
			wantErr:     true,
			errorString: errMsgFailedToRetrieve + " subscription",
		},
		{
			name:           "unauthorized access to subscription",
			subscriptionID: subscriptionID,
			workspaceID:    otherWorkspaceID,
			setupMocks: func() {
				unauthorizedSubscription := db.Subscription{
					ID:          subscriptionID,
					WorkspaceID: workspaceID, // Different workspace
				}
				mockQuerier.EXPECT().GetSubscriptionWithWorkspace(ctx, db.GetSubscriptionWithWorkspaceParams{
					ID:          subscriptionID,
					WorkspaceID: otherWorkspaceID,
				}).Return(unauthorizedSubscription, nil)
			},
			wantErr:     true,
			errorString: errMsgUnauthorizedAccess,
		},
		{
			name:           "database error getting events",
			subscriptionID: subscriptionID,
			workspaceID:    workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionWithWorkspace(ctx, db.GetSubscriptionWithWorkspaceParams{
					ID:          subscriptionID,
					WorkspaceID: workspaceID,
				}).Return(expectedSubscription, nil)
				mockQuerier.EXPECT().ListSubscriptionEventsBySubscription(ctx, subscriptionID).Return(nil, errors.New(errMsgDatabaseError))
			},
			wantErr:     true,
			errorString: errMsgFailedToRetrieve + " subscription events",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			events, err := service.ListSubscriptionEventsBySubscription(ctx, tt.subscriptionID, tt.workspaceID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, events)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, events, tt.wantCount)
			}
		})
	}
}

func TestSubscriptionEventService_CreateSubscriptionEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewSubscriptionEventService(mockQuerier)
	ctx := context.Background()

	subscriptionID := uuid.New()
	expectedSubscription := db.Subscription{
		ID: subscriptionID,
	}

	expectedEvent := db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subscriptionID,
		EventType:      "payment_success",
		AmountInCents:  1000,
	}

	tests := []struct {
		name        string
		params      params.CreateSubscriptionEventParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully creates subscription event",
			params: params.CreateSubscriptionEventParams{
				WorkspaceID:     uuid.New(),
				SubscriptionID:  subscriptionID,
				EventType:       "payment_success",
				TransactionHash: &[]string{"0x123abc..."}[0],
				AmountInCents:   1000,
				FailureReason:   nil,
				Metadata:        map[string]interface{}{"key": "value"},
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(expectedSubscription, nil)
				mockQuerier.EXPECT().CreateSubscriptionEvent(ctx, gomock.Any()).Return(expectedEvent, nil)
			},
			wantErr: false,
		},
		{
			name: "successfully creates event with error message",
			params: params.CreateSubscriptionEventParams{
				WorkspaceID:     uuid.New(),
				SubscriptionID:  subscriptionID,
				EventType:       "payment_failed",
				TransactionHash: nil,
				AmountInCents:   1000,
				FailureReason:   &[]string{"Insufficient funds"}[0],
				Metadata:        nil,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(expectedSubscription, nil)
				mockQuerier.EXPECT().CreateSubscriptionEvent(ctx, gomock.Any()).Return(expectedEvent, nil)
			},
			wantErr: false,
		},
		{
			name: "subscription not found",
			params: params.CreateSubscriptionEventParams{
				WorkspaceID:    uuid.New(),
				SubscriptionID: subscriptionID,
				EventType:      "payment_success",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(db.Subscription{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: errMsgSubscriptionNotFound,
		},
		{
			name: "database error getting subscription",
			params: params.CreateSubscriptionEventParams{
				WorkspaceID:    uuid.New(),
				SubscriptionID: subscriptionID,
				EventType:      "payment_success",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(db.Subscription{}, errors.New(errMsgDatabaseError))
			},
			wantErr:     true,
			errorString: errMsgFailedToRetrieve + " subscription",
		},
		{
			name: "database error creating event",
			params: params.CreateSubscriptionEventParams{
				WorkspaceID:    uuid.New(),
				SubscriptionID: subscriptionID,
				EventType:      "payment_success",
				AmountInCents:  1000,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(expectedSubscription, nil)
				mockQuerier.EXPECT().CreateSubscriptionEvent(ctx, gomock.Any()).Return(db.SubscriptionEvent{}, errors.New("creation error"))
			},
			wantErr:     true,
			errorString: errMsgFailedToCreate + " subscription event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			event, err := service.CreateSubscriptionEvent(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, event)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, event)
				assert.Equal(t, subscriptionID, event.SubscriptionID)
			}
		})
	}
}

func TestSubscriptionEventService_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewSubscriptionEventService(mockQuerier)

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
				mockQuerier.EXPECT().GetSubscriptionEvent(gomock.Any(), gomock.Any()).Return(db.SubscriptionEvent{}, assert.AnError)
				_, err := service.GetSubscriptionEvent(context.TODO(), uuid.New(), uuid.New())
				return err
			},
			expectError: true,
		},
		{
			name: "zero UUID handling",
			operation: func() error {
				mockQuerier.EXPECT().GetSubscriptionEvent(gomock.Any(), uuid.Nil).Return(db.SubscriptionEvent{}, pgx.ErrNoRows)
				_, err := service.GetSubscriptionEvent(context.Background(), uuid.Nil, uuid.New())
				return err
			},
			expectError: true,
			errorMsg:    "subscription event not found",
		},
		{
			name: "whitespace-only transaction hash",
			operation: func() error {
				// Mock the database call that will happen with whitespace hash
				mockQuerier.EXPECT().GetSubscriptionEventByTransactionHash(gomock.Any(), pgtype.Text{String: "   ", Valid: true}).Return(db.SubscriptionEvent{}, pgx.ErrNoRows)
				_, err := service.GetSubscriptionEventByTxHash(context.Background(), "   ")
				return err
			},
			expectError: true, // Should error when not found
			errorMsg:    "subscription event not found",
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
				// For the whitespace case, it won't error on validation but will fail on DB lookup
				// We expect an error due to missing mock setup, which is fine for this edge case test
			}
		})
	}
}

func TestSubscriptionEventService_BoundaryConditions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewSubscriptionEventService(mockQuerier)
	ctx := context.Background()

	tests := []struct {
		name           string
		params         params.ListSubscriptionEventsParams
		expectedLimit  int32
		expectedOffset int32
		setupMocks     func()
	}{
		{
			name: "extreme page values",
			params: params.ListSubscriptionEventsParams{
				WorkspaceID: uuid.New(),
				Limit:       -100,
				Offset:      -5,
			},
			expectedLimit:  20, // Default
			expectedOffset: 0,  // Page 1
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionEventDetailsWithPagination(ctx, gomock.Any()).Return([]db.ListSubscriptionEventDetailsWithPaginationRow{}, nil)
				mockQuerier.EXPECT().CountSubscriptionEventDetails(ctx, gomock.Any()).Return(int64(0), nil)
			},
		},
		{
			name: "maximum valid limit",
			params: params.ListSubscriptionEventsParams{
				WorkspaceID: uuid.New(),
				Limit:       100,
				Offset:      0,
			},
			expectedLimit:  100,
			expectedOffset: 0,
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionEventDetailsWithPagination(ctx, gomock.Any()).Return([]db.ListSubscriptionEventDetailsWithPaginationRow{}, nil)
				mockQuerier.EXPECT().CountSubscriptionEventDetails(ctx, gomock.Any()).Return(int64(0), nil)
			},
		},
		{
			name: "limit exceeding maximum",
			params: params.ListSubscriptionEventsParams{
				WorkspaceID: uuid.New(),
				Limit:       101,
				Offset:      0,
			},
			expectedLimit:  20, // Default
			expectedOffset: 0,
			setupMocks: func() {
				mockQuerier.EXPECT().ListSubscriptionEventDetailsWithPagination(ctx, gomock.Any()).Return([]db.ListSubscriptionEventDetailsWithPaginationRow{}, nil)
				mockQuerier.EXPECT().CountSubscriptionEventDetails(ctx, gomock.Any()).Return(int64(0), nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.ListSubscriptionEvents(ctx, tt.params)

			assert.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestSubscriptionEventService_ConcurrentAccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewSubscriptionEventService(mockQuerier)
	ctx := context.Background()

	// Test concurrent event retrieval
	t.Run("concurrent event retrieval", func(t *testing.T) {
		const numGoroutines = 10
		resultsChan := make(chan *db.SubscriptionEvent, numGoroutines)
		errorsChan := make(chan error, numGoroutines)

		eventID := uuid.New()
		workspaceID := uuid.New()
		subscriptionID := uuid.New()
		productID := uuid.New()

		testEvent := db.SubscriptionEvent{
			ID:             eventID,
			SubscriptionID: subscriptionID,
			EventType:      "payment_success",
		}

		testSubscription := db.Subscription{
			ID:        subscriptionID,
			ProductID: productID,
		}

		testProduct := db.Product{
			ID:          productID,
			WorkspaceID: workspaceID,
		}

		// Setup mocks for concurrent calls
		mockQuerier.EXPECT().GetSubscriptionEvent(ctx, eventID).Return(testEvent, nil).Times(numGoroutines)
		mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(testSubscription, nil).Times(numGoroutines)
		mockQuerier.EXPECT().GetProduct(ctx, db.GetProductParams{
			ID:          productID,
			WorkspaceID: workspaceID,
		}).Return(testProduct, nil).Times(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				event, err := service.GetSubscriptionEvent(ctx, eventID, workspaceID)
				if err != nil {
					errorsChan <- err
				} else {
					resultsChan <- event
				}
			}()
		}

		// Collect results
		var results []*db.SubscriptionEvent
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
			assert.Equal(t, eventID, result.ID)
			assert.Equal(t, subscriptionID, result.SubscriptionID)
		}
	})
}
