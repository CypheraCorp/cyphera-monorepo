package services_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func init() {
	logger.InitLogger("test")
}

// Mock implementations for the service interfaces
type mockProrationCalculator struct {
	ctrl *gomock.Controller
}

func newMockProrationCalculator(ctrl *gomock.Controller) *mockProrationCalculator {
	return &mockProrationCalculator{ctrl: ctrl}
}

func (m *mockProrationCalculator) CalculateUpgradeProration(currentPeriodStart, currentPeriodEnd time.Time, oldAmountCents, newAmountCents int64, changeDate time.Time) *business.ProrationResult {
	return &business.ProrationResult{
		DaysTotal:     30,
		DaysUsed:      10,
		DaysRemaining: 20,
		CreditAmount:  1000,
		ChargeAmount:  2000,
		NetAmount:     1000,
		Calculation:   map[string]interface{}{"test": "data"},
	}
}

func (m *mockProrationCalculator) ScheduleDowngrade(currentPeriodEnd time.Time, changeType string) *business.ScheduleChangeResult {
	return &business.ScheduleChangeResult{
		ScheduledFor: currentPeriodEnd,
		Message:      "Downgrade scheduled for end of period",
	}
}

func (m *mockProrationCalculator) CalculatePauseCredit(currentPeriodStart, currentPeriodEnd time.Time, amountCents int64, pauseDate time.Time) *business.ProrationResult {
	return &business.ProrationResult{
		DaysTotal:     30,
		DaysUsed:      10,
		DaysRemaining: 20,
		CreditAmount:  500,
		ChargeAmount:  0,
		NetAmount:     -500,
		Calculation:   map[string]interface{}{"pause": "credit"},
	}
}

func (m *mockProrationCalculator) AddBillingPeriod(start time.Time, intervalType string, intervalCount int) time.Time {
	return start.AddDate(0, intervalCount, 0)
}

func (m *mockProrationCalculator) FormatProrationExplanation(result *business.ProrationResult) string {
	return "Proration explanation"
}

func (m *mockProrationCalculator) DaysBetween(start, end time.Time) int {
	return int(end.Sub(start).Hours() / 24)
}

func (m *mockProrationCalculator) CalculateTrialEndDate(start time.Time, trialDays int) time.Time {
	return start.AddDate(0, 0, trialDays)
}

func (m *mockProrationCalculator) IsInTrial(trialEnd *time.Time) bool {
	if trialEnd == nil {
		return false
	}
	return time.Now().Before(*trialEnd)
}

func (m *mockProrationCalculator) GetDailyRate(amountCents int64, periodStart, periodEnd time.Time) float64 {
	days := m.DaysBetween(periodStart, periodEnd)
	if days == 0 {
		return 0
	}
	return float64(amountCents) / float64(days)
}

type mockPaymentService struct {
	ctrl *gomock.Controller
}

func newMockPaymentService(ctrl *gomock.Controller) *mockPaymentService {
	return &mockPaymentService{ctrl: ctrl}
}

func (m *mockPaymentService) CreatePaymentFromSubscriptionEvent(ctx context.Context, params params.CreatePaymentFromSubscriptionEventParams) (*db.Payment, error) {
	return &db.Payment{
		ID:            uuid.New(),
		AmountInCents: 1000, // Mock amount
		Status:        "completed",
	}, nil
}

func (m *mockPaymentService) CreateComprehensivePayment(ctx context.Context, params params.CreateComprehensivePaymentParams) (*db.Payment, error) {
	return &db.Payment{
		ID:            uuid.New(),
		AmountInCents: params.AmountCents,
		Status:        "completed",
	}, nil
}

func (m *mockPaymentService) GetPayment(ctx context.Context, params params.GetPaymentParams) (*db.Payment, error) {
	return &db.Payment{
		ID:            params.PaymentID,
		AmountInCents: 1000,
		Status:        "completed",
	}, nil
}

func (m *mockPaymentService) GetPaymentByTransactionHash(ctx context.Context, txHash string) (*db.Payment, error) {
	return &db.Payment{
		ID:            uuid.New(),
		AmountInCents: 1000,
		Status:        "completed",
	}, nil
}

func (m *mockPaymentService) ListPayments(ctx context.Context, params params.ListPaymentsParams) ([]db.Payment, error) {
	return []db.Payment{
		{ID: uuid.New(), AmountInCents: 1000, Status: "completed"},
	}, nil
}

func (m *mockPaymentService) UpdatePaymentStatus(ctx context.Context, params params.UpdatePaymentStatusParams) (*db.Payment, error) {
	return &db.Payment{
		ID:            params.PaymentID,
		AmountInCents: 1000,
		Status:        params.Status,
	}, nil
}

func (m *mockPaymentService) GetPaymentMetrics(ctx context.Context, workspaceID uuid.UUID, startTime, endTime time.Time, currency string) (*db.GetPaymentMetricsRow, error) {
	return &db.GetPaymentMetricsRow{}, nil
}

func (m *mockPaymentService) CreateManualPayment(ctx context.Context, params params.CreateManualPaymentParams) (*db.Payment, error) {
	return &db.Payment{
		ID:            uuid.New(),
		AmountInCents: params.AmountInCents,
		Status:        "completed",
	}, nil
}

type mockEmailService struct {
	ctrl *gomock.Controller
}

func newMockEmailService(ctrl *gomock.Controller) *mockEmailService {
	return &mockEmailService{ctrl: ctrl}
}

func (m *mockEmailService) SendTransactionalEmail(ctx context.Context, emailParams params.TransactionalEmailParams) error {
	return nil
}

func (m *mockEmailService) SendBatchEmails(ctx context.Context, requests []requests.BatchEmailRequest) ([]responses.BatchEmailResult, error) {
	results := make([]responses.BatchEmailResult, len(requests))
	for i := range requests {
		results[i] = responses.BatchEmailResult{
			Index:   i,
			Success: true,
			Error:   "",
		}
	}
	return results, nil
}

func (m *mockEmailService) SendDunningEmail(ctx context.Context, template *db.DunningEmailTemplate, data map[string]business.EmailData, toEmail string) error {
	return nil
}

func TestSubscriptionManagementService_UpgradeSubscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockCalculator := newMockProrationCalculator(ctrl)
	mockPaymentService := newMockPaymentService(ctrl)
	mockEmailService := newMockEmailService(ctrl)

	service := services.NewSubscriptionManagementServiceWithDependencies(
		mockQuerier,
		mockCalculator,
		mockPaymentService,
		mockEmailService,
		zap.NewNop(),
	)

	ctx := context.Background()
	subscriptionID := uuid.New()

	subscription := db.Subscription{
		ID:                 subscriptionID,
		Status:             db.SubscriptionStatusActive,
		ProductID:          uuid.New(),
		PriceID:            uuid.New(),
		WorkspaceID:        uuid.New(),
		CustomerID:         uuid.New(),
		TotalAmountInCents: 1000,
		CurrentPeriodStart: pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -10), Valid: true},
		CurrentPeriodEnd:   pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 20), Valid: true},
	}

	tests := []struct {
		name           string
		subscriptionID uuid.UUID
		newLineItems   []requests.LineItemUpdate
		reason         string
		setupMocks     func()
		wantErr        bool
		errorString    string
	}{
		{
			name:           "successfully upgrades subscription",
			subscriptionID: subscriptionID,
			newLineItems: []requests.LineItemUpdate{
				{
					Action:     "update",
					ProductID:  subscription.ProductID,
					PriceID:    subscription.PriceID,
					Quantity:   1,
					UnitAmount: 2000,
				},
			},
			reason: "Customer upgrade request",
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)

				scheduleChange := db.SubscriptionScheduleChange{
					ID:             uuid.New(),
					SubscriptionID: subscriptionID,
					ChangeType:     "upgrade",
					Status:         "processing",
				}
				mockQuerier.EXPECT().CreateScheduleChange(ctx, gomock.Any()).Return(scheduleChange, nil)

				updatedSub := subscription
				updatedSub.TotalAmountInCents = 2000
				mockQuerier.EXPECT().UpdateSubscriptionForUpgrade(ctx, gomock.Any()).Return(updatedSub, nil)

				mockQuerier.EXPECT().CreateProrationRecord(ctx, gomock.Any()).Return(db.SubscriptionProration{}, nil)
				mockQuerier.EXPECT().RecordStateChange(ctx, gomock.Any()).Return(db.SubscriptionStateHistory{}, nil)
				mockQuerier.EXPECT().UpdateScheduleChangeStatus(ctx, gomock.Any()).Return(scheduleChange, nil)

				// Mock email dependencies
				customer := db.Customer{Name: pgtype.Text{String: "John Doe", Valid: true}, Email: pgtype.Text{String: "john@example.com", Valid: true}}
				product := db.Product{Name: "Test Product"}
				workspace := db.Workspace{Name: "Test Workspace"}

				mockQuerier.EXPECT().GetCustomer(ctx, subscription.CustomerID).Return(customer, nil)
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(ctx, subscription.ProductID).Return(product, nil)
				mockQuerier.EXPECT().GetWorkspace(ctx, subscription.WorkspaceID).Return(workspace, nil)
			},
			wantErr: false,
		},
		{
			name:           "fails when subscription not found",
			subscriptionID: subscriptionID,
			newLineItems:   []requests.LineItemUpdate{},
			reason:         "Test upgrade",
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(db.Subscription{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "failed to get subscription",
		},
		{
			name:           "fails when subscription not active",
			subscriptionID: subscriptionID,
			newLineItems:   []requests.LineItemUpdate{},
			reason:         "Test upgrade",
			setupMocks: func() {
				inactiveSub := subscription
				inactiveSub.Status = db.SubscriptionStatusCanceled
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(inactiveSub, nil)
			},
			wantErr:     true,
			errorString: "can only upgrade active subscriptions",
		},
		{
			name:           "handles database error during schedule change creation",
			subscriptionID: subscriptionID,
			newLineItems: []requests.LineItemUpdate{
				{Action: "update", UnitAmount: 2000, Quantity: 1},
			},
			reason: "Test upgrade",
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)
				mockQuerier.EXPECT().CreateScheduleChange(ctx, gomock.Any()).Return(db.SubscriptionScheduleChange{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to create schedule change",
		},
		{
			name:           "handles database error during subscription update",
			subscriptionID: subscriptionID,
			newLineItems: []requests.LineItemUpdate{
				{Action: "update", UnitAmount: 2000, Quantity: 1},
			},
			reason: "Test upgrade",
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)

				scheduleChange := db.SubscriptionScheduleChange{ID: uuid.New()}
				mockQuerier.EXPECT().CreateScheduleChange(ctx, gomock.Any()).Return(scheduleChange, nil)
				mockQuerier.EXPECT().UpdateSubscriptionForUpgrade(ctx, gomock.Any()).Return(db.Subscription{}, errors.New("update error"))
			},
			wantErr:     true,
			errorString: "failed to update subscription",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.UpgradeSubscription(ctx, tt.subscriptionID, tt.newLineItems, tt.reason)

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

func TestSubscriptionManagementService_DowngradeSubscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockCalculator := newMockProrationCalculator(ctrl)
	mockPaymentService := newMockPaymentService(ctrl)
	mockEmailService := newMockEmailService(ctrl)

	service := services.NewSubscriptionManagementServiceWithDependencies(
		mockQuerier,
		mockCalculator,
		mockPaymentService,
		mockEmailService,
		zap.NewNop(),
	)

	ctx := context.Background()
	subscriptionID := uuid.New()

	subscription := db.Subscription{
		ID:                 subscriptionID,
		Status:             db.SubscriptionStatusActive,
		ProductID:          uuid.New(),
		PriceID:            uuid.New(),
		WorkspaceID:        uuid.New(),
		CustomerID:         uuid.New(),
		TotalAmountInCents: 2000,
		CurrentPeriodEnd:   pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 20), Valid: true},
	}

	tests := []struct {
		name           string
		subscriptionID uuid.UUID
		newLineItems   []requests.LineItemUpdate
		reason         string
		setupMocks     func()
		wantErr        bool
		errorString    string
	}{
		{
			name:           "successfully schedules downgrade",
			subscriptionID: subscriptionID,
			newLineItems: []requests.LineItemUpdate{
				{
					Action:     "update",
					ProductID:  subscription.ProductID,
					PriceID:    subscription.PriceID,
					Quantity:   1,
					UnitAmount: 1000,
				},
			},
			reason: "Customer downgrade request",
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)

				scheduleChange := db.SubscriptionScheduleChange{
					ID:             uuid.New(),
					SubscriptionID: subscriptionID,
					ChangeType:     "downgrade",
					Status:         "scheduled",
				}
				mockQuerier.EXPECT().CreateScheduleChange(ctx, gomock.Any()).Return(scheduleChange, nil)

				// Mock email dependencies
				customer := db.Customer{Name: pgtype.Text{String: "Jane Doe", Valid: true}, Email: pgtype.Text{String: "jane@example.com", Valid: true}}
				product := db.Product{Name: "Test Product"}
				workspace := db.Workspace{Name: "Test Workspace"}

				mockQuerier.EXPECT().GetCustomer(ctx, subscription.CustomerID).Return(customer, nil)
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(ctx, subscription.ProductID).Return(product, nil)
				mockQuerier.EXPECT().GetWorkspace(ctx, subscription.WorkspaceID).Return(workspace, nil)
			},
			wantErr: false,
		},
		{
			name:           "fails when subscription not found",
			subscriptionID: subscriptionID,
			newLineItems:   []requests.LineItemUpdate{},
			reason:         "Test downgrade",
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(db.Subscription{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "failed to get subscription",
		},
		{
			name:           "fails when subscription not active",
			subscriptionID: subscriptionID,
			newLineItems:   []requests.LineItemUpdate{},
			reason:         "Test downgrade",
			setupMocks: func() {
				inactiveSub := subscription
				inactiveSub.Status = db.SubscriptionStatusCanceled
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(inactiveSub, nil)
			},
			wantErr:     true,
			errorString: "can only downgrade active subscriptions",
		},
		{
			name:           "handles database error during schedule change creation",
			subscriptionID: subscriptionID,
			newLineItems: []requests.LineItemUpdate{
				{Action: "update", UnitAmount: 1000, Quantity: 1},
			},
			reason: "Test downgrade",
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)
				mockQuerier.EXPECT().CreateScheduleChange(ctx, gomock.Any()).Return(db.SubscriptionScheduleChange{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to create schedule change",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.DowngradeSubscription(ctx, tt.subscriptionID, tt.newLineItems, tt.reason)

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

func TestSubscriptionManagementService_CancelSubscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockCalculator := newMockProrationCalculator(ctrl)
	mockPaymentService := newMockPaymentService(ctrl)
	mockEmailService := newMockEmailService(ctrl)

	service := services.NewSubscriptionManagementServiceWithDependencies(
		mockQuerier,
		mockCalculator,
		mockPaymentService,
		mockEmailService,
		zap.NewNop(),
	)

	ctx := context.Background()
	subscriptionID := uuid.New()

	subscription := db.Subscription{
		ID:                 subscriptionID,
		Status:             db.SubscriptionStatusActive,
		ProductID:          uuid.New(),
		WorkspaceID:        uuid.New(),
		CustomerID:         uuid.New(),
		TotalAmountInCents: 1000,
		CurrentPeriodEnd:   pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 20), Valid: true},
		CancelAt:           pgtype.Timestamptz{Valid: false},
	}

	tests := []struct {
		name           string
		subscriptionID uuid.UUID
		reason         string
		feedback       string
		setupMocks     func()
		wantErr        bool
		errorString    string
	}{
		{
			name:           "successfully schedules cancellation",
			subscriptionID: subscriptionID,
			reason:         "Customer request",
			feedback:       "Too expensive",
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)

				updatedSub := subscription
				updatedSub.CancelAt = pgtype.Timestamptz{Time: subscription.CurrentPeriodEnd.Time, Valid: true}
				mockQuerier.EXPECT().ScheduleSubscriptionCancellation(ctx, gomock.Any()).Return(updatedSub, nil)

				scheduleChange := db.SubscriptionScheduleChange{
					ID:             uuid.New(),
					SubscriptionID: subscriptionID,
					ChangeType:     "cancel",
					Status:         "scheduled",
				}
				mockQuerier.EXPECT().CreateScheduleChange(ctx, gomock.Any()).Return(scheduleChange, nil)

				// Mock email dependencies
				customer := db.Customer{Name: pgtype.Text{String: "John Doe", Valid: true}, Email: pgtype.Text{String: "john@example.com", Valid: true}}
				product := db.Product{Name: "Test Product"}
				workspace := db.Workspace{Name: "Test Workspace"}

				mockQuerier.EXPECT().GetCustomer(ctx, subscription.CustomerID).Return(customer, nil)
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(ctx, subscription.ProductID).Return(product, nil)
				mockQuerier.EXPECT().GetWorkspace(ctx, subscription.WorkspaceID).Return(workspace, nil)
			},
			wantErr: false,
		},
		{
			name:           "fails when subscription not found",
			subscriptionID: subscriptionID,
			reason:         "Test cancel",
			feedback:       "Test feedback",
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(db.Subscription{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "failed to get subscription",
		},
		{
			name:           "fails when subscription already cancelled",
			subscriptionID: subscriptionID,
			reason:         "Test cancel",
			feedback:       "Test feedback",
			setupMocks: func() {
				cancelledSub := subscription
				cancelledSub.Status = db.SubscriptionStatusCanceled
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(cancelledSub, nil)
			},
			wantErr:     true,
			errorString: "subscription already cancelled",
		},
		{
			name:           "fails when subscription already scheduled for cancellation",
			subscriptionID: subscriptionID,
			reason:         "Test cancel",
			feedback:       "Test feedback",
			setupMocks: func() {
				scheduledSub := subscription
				scheduledSub.CancelAt = pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 10), Valid: true}
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(scheduledSub, nil)
			},
			wantErr:     true,
			errorString: "subscription already cancelled",
		},
		{
			name:           "handles database error during cancellation scheduling",
			subscriptionID: subscriptionID,
			reason:         "Test cancel",
			feedback:       "Test feedback",
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)
				mockQuerier.EXPECT().ScheduleSubscriptionCancellation(ctx, gomock.Any()).Return(db.Subscription{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to schedule cancellation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.CancelSubscription(ctx, tt.subscriptionID, tt.reason, tt.feedback)

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

func TestSubscriptionManagementService_PauseSubscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockCalculator := newMockProrationCalculator(ctrl)
	mockPaymentService := newMockPaymentService(ctrl)
	mockEmailService := newMockEmailService(ctrl)

	service := services.NewSubscriptionManagementServiceWithDependencies(
		mockQuerier,
		mockCalculator,
		mockPaymentService,
		mockEmailService,
		zap.NewNop(),
	)

	ctx := context.Background()
	subscriptionID := uuid.New()

	subscription := db.Subscription{
		ID:                 subscriptionID,
		Status:             db.SubscriptionStatusActive,
		ProductID:          uuid.New(),
		WorkspaceID:        uuid.New(),
		CustomerID:         uuid.New(),
		TotalAmountInCents: 1000,
		CurrentPeriodStart: pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -10), Valid: true},
		CurrentPeriodEnd:   pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 20), Valid: true},
	}

	futureDate := time.Now().AddDate(0, 1, 0)

	tests := []struct {
		name           string
		subscriptionID uuid.UUID
		pauseUntil     *time.Time
		reason         string
		setupMocks     func()
		wantErr        bool
		errorString    string
	}{
		{
			name:           "successfully pauses subscription without end date",
			subscriptionID: subscriptionID,
			pauseUntil:     nil,
			reason:         "Customer request",
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)

				pausedSub := subscription
				pausedSub.Status = db.SubscriptionStatusSuspended
				mockQuerier.EXPECT().PauseSubscription(ctx, gomock.Any()).Return(pausedSub, nil)
				mockQuerier.EXPECT().CreateProrationRecord(ctx, gomock.Any()).Return(db.SubscriptionProration{}, nil)

				// Mock email dependencies
				customer := db.Customer{Name: pgtype.Text{String: "John Doe", Valid: true}, Email: pgtype.Text{String: "john@example.com", Valid: true}}
				product := db.Product{Name: "Test Product"}
				workspace := db.Workspace{Name: "Test Workspace"}

				mockQuerier.EXPECT().GetCustomer(ctx, subscription.CustomerID).Return(customer, nil)
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(ctx, subscription.ProductID).Return(product, nil)
				mockQuerier.EXPECT().GetWorkspace(ctx, subscription.WorkspaceID).Return(workspace, nil)
			},
			wantErr: false,
		},
		{
			name:           "successfully pauses subscription with end date",
			subscriptionID: subscriptionID,
			pauseUntil:     &futureDate,
			reason:         "Temporary pause",
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)

				pausedSub := subscription
				pausedSub.Status = db.SubscriptionStatusSuspended
				mockQuerier.EXPECT().PauseSubscription(ctx, gomock.Any()).Return(pausedSub, nil)

				// Schedule automatic resume
				scheduleChange := db.SubscriptionScheduleChange{
					ID:             uuid.New(),
					SubscriptionID: subscriptionID,
					ChangeType:     "resume",
					Status:         "scheduled",
				}
				mockQuerier.EXPECT().CreateScheduleChange(ctx, gomock.Any()).Return(scheduleChange, nil)
				mockQuerier.EXPECT().CreateProrationRecord(ctx, gomock.Any()).Return(db.SubscriptionProration{}, nil)

				// Mock email dependencies
				customer := db.Customer{Name: pgtype.Text{String: "John Doe", Valid: true}, Email: pgtype.Text{String: "john@example.com", Valid: true}}
				product := db.Product{Name: "Test Product"}
				workspace := db.Workspace{Name: "Test Workspace"}

				mockQuerier.EXPECT().GetCustomer(ctx, subscription.CustomerID).Return(customer, nil)
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(ctx, subscription.ProductID).Return(product, nil)
				mockQuerier.EXPECT().GetWorkspace(ctx, subscription.WorkspaceID).Return(workspace, nil)
			},
			wantErr: false,
		},
		{
			name:           "fails when subscription not found",
			subscriptionID: subscriptionID,
			pauseUntil:     nil,
			reason:         "Test pause",
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(db.Subscription{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "failed to get subscription",
		},
		{
			name:           "fails when subscription not active",
			subscriptionID: subscriptionID,
			pauseUntil:     nil,
			reason:         "Test pause",
			setupMocks: func() {
				inactiveSub := subscription
				inactiveSub.Status = db.SubscriptionStatusCanceled
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(inactiveSub, nil)
			},
			wantErr:     true,
			errorString: "can only pause active subscriptions",
		},
		{
			name:           "handles database error during pause",
			subscriptionID: subscriptionID,
			pauseUntil:     nil,
			reason:         "Test pause",
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)
				mockQuerier.EXPECT().PauseSubscription(ctx, gomock.Any()).Return(db.Subscription{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to pause subscription",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.PauseSubscription(ctx, tt.subscriptionID, tt.pauseUntil, tt.reason)

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

func TestSubscriptionManagementService_ResumeSubscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockCalculator := newMockProrationCalculator(ctrl)
	mockPaymentService := newMockPaymentService(ctrl)
	mockEmailService := newMockEmailService(ctrl)

	service := services.NewSubscriptionManagementServiceWithDependencies(
		mockQuerier,
		mockCalculator,
		mockPaymentService,
		mockEmailService,
		zap.NewNop(),
	)

	ctx := context.Background()
	subscriptionID := uuid.New()

	subscription := db.Subscription{
		ID:                 subscriptionID,
		Status:             db.SubscriptionStatusSuspended,
		ProductID:          uuid.New(),
		WorkspaceID:        uuid.New(),
		CustomerID:         uuid.New(),
		TotalAmountInCents: 1000,
		CurrentPeriodStart: pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -10), Valid: true},
		CurrentPeriodEnd:   pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 20), Valid: true},
	}

	tests := []struct {
		name           string
		subscriptionID uuid.UUID
		setupMocks     func()
		wantErr        bool
		errorString    string
	}{
		{
			name:           "successfully resumes subscription",
			subscriptionID: subscriptionID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)

				resumedSub := subscription
				resumedSub.Status = db.SubscriptionStatusActive
				mockQuerier.EXPECT().ResumeSubscription(ctx, gomock.Any()).Return(resumedSub, nil)
				mockQuerier.EXPECT().RecordStateChange(ctx, gomock.Any()).Return(db.SubscriptionStateHistory{}, nil)

				// Mock email dependencies
				customer := db.Customer{Name: pgtype.Text{String: "John Doe", Valid: true}, Email: pgtype.Text{String: "john@example.com", Valid: true}}
				product := db.Product{Name: "Test Product"}
				workspace := db.Workspace{Name: "Test Workspace"}

				mockQuerier.EXPECT().GetCustomer(ctx, subscription.CustomerID).Return(customer, nil)
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(ctx, subscription.ProductID).Return(product, nil)
				mockQuerier.EXPECT().GetWorkspace(ctx, subscription.WorkspaceID).Return(workspace, nil)
			},
			wantErr: false,
		},
		{
			name:           "fails when subscription not found",
			subscriptionID: subscriptionID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(db.Subscription{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "failed to get subscription",
		},
		{
			name:           "fails when subscription not paused",
			subscriptionID: subscriptionID,
			setupMocks: func() {
				activeSub := subscription
				activeSub.Status = db.SubscriptionStatusActive
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(activeSub, nil)
			},
			wantErr:     true,
			errorString: "can only resume paused subscriptions",
		},
		{
			name:           "handles database error during resume",
			subscriptionID: subscriptionID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)
				mockQuerier.EXPECT().ResumeSubscription(ctx, gomock.Any()).Return(db.Subscription{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to resume subscription",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.ResumeSubscription(ctx, tt.subscriptionID)

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

func TestSubscriptionManagementService_ReactivateCancelledSubscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockCalculator := newMockProrationCalculator(ctrl)
	mockPaymentService := newMockPaymentService(ctrl)
	mockEmailService := newMockEmailService(ctrl)

	service := services.NewSubscriptionManagementServiceWithDependencies(
		mockQuerier,
		mockCalculator,
		mockPaymentService,
		mockEmailService,
		zap.NewNop(),
	)

	ctx := context.Background()
	subscriptionID := uuid.New()

	tests := []struct {
		name           string
		subscriptionID uuid.UUID
		setupMocks     func()
		wantErr        bool
		errorString    string
	}{
		{
			name:           "successfully reactivates cancelled subscription",
			subscriptionID: subscriptionID,
			setupMocks: func() {
				reactivatedSub := db.Subscription{
					ID:                 subscriptionID,
					Status:             db.SubscriptionStatusActive,
					ProductID:          uuid.New(),
					WorkspaceID:        uuid.New(),
					CustomerID:         uuid.New(),
					TotalAmountInCents: 1000,
					CancelAt:           pgtype.Timestamptz{Valid: false},
				}
				mockQuerier.EXPECT().ReactivateScheduledCancellation(ctx, subscriptionID).Return(reactivatedSub, nil)

				changes := []db.SubscriptionScheduleChange{
					{
						ID:             uuid.New(),
						SubscriptionID: subscriptionID,
						ChangeType:     "cancel",
						Status:         "scheduled",
					},
				}
				mockQuerier.EXPECT().GetSubscriptionScheduledChanges(ctx, subscriptionID).Return(changes, nil)
				mockQuerier.EXPECT().CancelScheduledChange(ctx, changes[0].ID).Return(changes[0], nil)

				// Mock email dependencies
				customer := db.Customer{Name: pgtype.Text{String: "John Doe", Valid: true}, Email: pgtype.Text{String: "john@example.com", Valid: true}}
				product := db.Product{Name: "Test Product"}
				workspace := db.Workspace{Name: "Test Workspace"}

				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(reactivatedSub, nil)
				mockQuerier.EXPECT().GetCustomer(ctx, reactivatedSub.CustomerID).Return(customer, nil)
				mockQuerier.EXPECT().GetProductWithoutWorkspaceId(ctx, reactivatedSub.ProductID).Return(product, nil)
				mockQuerier.EXPECT().GetWorkspace(ctx, reactivatedSub.WorkspaceID).Return(workspace, nil)
			},
			wantErr: false,
		},
		{
			name:           "fails when subscription not scheduled for cancellation",
			subscriptionID: subscriptionID,
			setupMocks: func() {
				mockQuerier.EXPECT().ReactivateScheduledCancellation(ctx, subscriptionID).Return(db.Subscription{}, sql.ErrNoRows)
			},
			wantErr:     true,
			errorString: "subscription is not scheduled for cancellation",
		},
		{
			name:           "handles database error during reactivation",
			subscriptionID: subscriptionID,
			setupMocks: func() {
				mockQuerier.EXPECT().ReactivateScheduledCancellation(ctx, subscriptionID).Return(db.Subscription{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to reactivate subscription",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.ReactivateCancelledSubscription(ctx, tt.subscriptionID)

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

func TestSubscriptionManagementService_PreviewChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockCalculator := newMockProrationCalculator(ctrl)
	mockPaymentService := newMockPaymentService(ctrl)
	mockEmailService := newMockEmailService(ctrl)

	service := services.NewSubscriptionManagementServiceWithDependencies(
		mockQuerier,
		mockCalculator,
		mockPaymentService,
		mockEmailService,
		zap.NewNop(),
	)

	ctx := context.Background()
	subscriptionID := uuid.New()

	subscription := db.Subscription{
		ID:                 subscriptionID,
		Status:             db.SubscriptionStatusActive,
		TotalAmountInCents: 1000,
		CurrentPeriodStart: pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -10), Valid: true},
		CurrentPeriodEnd:   pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 20), Valid: true},
	}

	tests := []struct {
		name           string
		subscriptionID uuid.UUID
		changeType     string
		newLineItems   []requests.LineItemUpdate
		setupMocks     func()
		wantErr        bool
		errorString    string
		validateResult func(*business.ChangePreview)
	}{
		{
			name:           "successfully previews upgrade",
			subscriptionID: subscriptionID,
			changeType:     "upgrade",
			newLineItems: []requests.LineItemUpdate{
				{Action: "update", UnitAmount: 2000, Quantity: 1},
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)
			},
			wantErr: false,
			validateResult: func(preview *business.ChangePreview) {
				assert.Equal(t, int64(1000), preview.CurrentAmount)
				assert.Equal(t, int64(2000), preview.NewAmount)
				assert.Equal(t, int64(1000), preview.ProrationCredit)
				assert.Equal(t, int64(1000), preview.ImmediateCharge)
				assert.NotNil(t, preview.ProrationDetails)
				assert.Equal(t, "Proration explanation", preview.Message)
			},
		},
		{
			name:           "successfully previews downgrade",
			subscriptionID: subscriptionID,
			changeType:     "downgrade",
			newLineItems: []requests.LineItemUpdate{
				{Action: "update", UnitAmount: 500, Quantity: 1},
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)
			},
			wantErr: false,
			validateResult: func(preview *business.ChangePreview) {
				assert.Equal(t, int64(1000), preview.CurrentAmount)
				assert.Equal(t, int64(500), preview.NewAmount)
				assert.Equal(t, subscription.CurrentPeriodEnd.Time, preview.EffectiveDate)
				assert.Contains(t, preview.Message, "end of your current billing period")
			},
		},
		{
			name:           "successfully previews cancellation",
			subscriptionID: subscriptionID,
			changeType:     "cancel",
			newLineItems:   []requests.LineItemUpdate{},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)
			},
			wantErr: false,
			validateResult: func(preview *business.ChangePreview) {
				assert.Equal(t, int64(1000), preview.CurrentAmount)
				assert.Equal(t, int64(0), preview.NewAmount)
				assert.Equal(t, subscription.CurrentPeriodEnd.Time, preview.EffectiveDate)
				assert.Contains(t, preview.Message, "cancelled at the end")
			},
		},
		{
			name:           "fails when subscription not found",
			subscriptionID: subscriptionID,
			changeType:     "upgrade",
			newLineItems:   []requests.LineItemUpdate{},
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(db.Subscription{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "failed to get subscription",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			preview, err := service.PreviewChange(ctx, tt.subscriptionID, tt.changeType, tt.newLineItems)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, preview)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, preview)
				if tt.validateResult != nil {
					tt.validateResult(preview)
				}
			}
		})
	}
}

func TestSubscriptionManagementService_ProcessScheduledChanges(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockCalculator := newMockProrationCalculator(ctrl)
	mockPaymentService := newMockPaymentService(ctrl)
	mockEmailService := newMockEmailService(ctrl)

	service := services.NewSubscriptionManagementServiceWithDependencies(
		mockQuerier,
		mockCalculator,
		mockPaymentService,
		mockEmailService,
		zap.NewNop(),
	)

	ctx := context.Background()

	tests := []struct {
		name       string
		setupMocks func()
		wantErr    bool
	}{
		{
			name: "successfully processes scheduled changes",
			setupMocks: func() {
				changes := []db.SubscriptionScheduleChange{
					{
						ID:             uuid.New(),
						SubscriptionID: uuid.New(),
						ChangeType:     "downgrade",
						Status:         "scheduled",
					},
					{
						ID:             uuid.New(),
						SubscriptionID: uuid.New(),
						ChangeType:     "cancel",
						Status:         "scheduled",
					},
				}

				mockQuerier.EXPECT().GetDueScheduledChanges(ctx, gomock.Any()).Return(changes, nil)

				for _, change := range changes {
					mockQuerier.EXPECT().UpdateScheduleChangeStatus(ctx, db.UpdateScheduleChangeStatusParams{
						ID:           change.ID,
						Status:       "processing",
						ErrorMessage: pgtype.Text{Valid: false},
					}).Return(change, nil)

					if change.ChangeType == "cancel" {
						mockQuerier.EXPECT().CancelSubscriptionImmediately(ctx, gomock.Any()).Return(db.Subscription{}, nil)
					}

					mockQuerier.EXPECT().UpdateScheduleChangeStatus(ctx, db.UpdateScheduleChangeStatusParams{
						ID:           change.ID,
						Status:       "completed",
						ErrorMessage: pgtype.Text{Valid: false},
					}).Return(change, nil)
				}
			},
			wantErr: false,
		},
		{
			name: "handles database error when getting scheduled changes",
			setupMocks: func() {
				mockQuerier.EXPECT().GetDueScheduledChanges(ctx, gomock.Any()).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
		{
			name: "returns empty when no scheduled changes",
			setupMocks: func() {
				mockQuerier.EXPECT().GetDueScheduledChanges(ctx, gomock.Any()).Return([]db.SubscriptionScheduleChange{}, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.ProcessScheduledChanges(ctx)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSubscriptionManagementService_GetSubscriptionHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewSubscriptionManagementServiceWithDependencies(
		mockQuerier,
		nil,
		nil,
		nil,
		zap.NewNop(),
	)

	ctx := context.Background()
	subscriptionID := uuid.New()

	tests := []struct {
		name           string
		subscriptionID uuid.UUID
		limit          int32
		setupMocks     func()
		wantErr        bool
		errorString    string
		wantCount      int
	}{
		{
			name:           "successfully gets subscription history",
			subscriptionID: subscriptionID,
			limit:          10,
			setupMocks: func() {
				history := []db.SubscriptionStateHistory{
					{ID: uuid.New(), SubscriptionID: subscriptionID},
					{ID: uuid.New(), SubscriptionID: subscriptionID},
				}
				mockQuerier.EXPECT().GetSubscriptionStateHistory(ctx, db.GetSubscriptionStateHistoryParams{
					SubscriptionID: subscriptionID,
					Limit:          10,
				}).Return(history, nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:           "handles database error",
			subscriptionID: subscriptionID,
			limit:          10,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionStateHistory(ctx, gomock.Any()).Return(nil, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to get subscription history",
		},
		{
			name:           "returns empty history",
			subscriptionID: subscriptionID,
			limit:          10,
			setupMocks: func() {
				mockQuerier.EXPECT().GetSubscriptionStateHistory(ctx, gomock.Any()).Return([]db.SubscriptionStateHistory{}, nil)
			},
			wantErr:   false,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			history, err := service.GetSubscriptionHistory(ctx, tt.subscriptionID, tt.limit)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, history)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, history, tt.wantCount)
			}
		})
	}
}

func TestSubscriptionManagementService_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewSubscriptionManagementServiceWithDependencies(
		mockQuerier,
		nil,
		nil,
		nil,
		zap.NewNop(),
	)

	tests := []struct {
		name      string
		operation func() error
		wantErr   bool
		errorMsg  string
	}{
		{
			name: "nil context handling",
			operation: func() error {
				// Mock the database call that will happen even with nil context
				mockQuerier.EXPECT().GetSubscription(nil, gomock.Any()).Return(db.Subscription{}, assert.AnError)
				_, err := service.PreviewChange(nil, uuid.New(), "upgrade", []requests.LineItemUpdate{})
				return err
			},
			wantErr: true,
		},
		{
			name: "empty UUID handling",
			operation: func() error {
				mockQuerier.EXPECT().GetSubscription(gomock.Any(), uuid.Nil).Return(db.Subscription{}, pgx.ErrNoRows)
				return service.UpgradeSubscription(context.Background(), uuid.Nil, []requests.LineItemUpdate{}, "test")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()

			if tt.wantErr {
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
