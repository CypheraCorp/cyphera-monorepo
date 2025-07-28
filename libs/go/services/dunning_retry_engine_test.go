package services_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func init() {
	logger.InitLogger("test")
}

// MockDelegationClient for testing
type MockDelegationClient struct {
	processPaymentFunc func(ctx context.Context, params params.LocalProcessPaymentParams) (*responses.LocalProcessPaymentResponse, error)
}

func (m *MockDelegationClient) ProcessPayment(ctx context.Context, params params.LocalProcessPaymentParams) (*responses.LocalProcessPaymentResponse, error) {
	if m.processPaymentFunc != nil {
		return m.processPaymentFunc(ctx, params)
	}
	return &responses.LocalProcessPaymentResponse{
		TransactionHash: "0x123abc",
		Status:          "success",
		GasUsed:         "21000",
		BlockNumber:     123456,
	}, nil
}

func TestDunningRetryEngine_ProcessDueCampaigns(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockDunningService := services.NewDunningService(mockQuerier, zap.NewNop())
	mockEmailService := (*services.EmailService)(nil) // Can be nil for testing
	mockDelegationClient := &MockDelegationClient{}

	engine := services.NewDunningRetryEngine(
		mockQuerier,
		zap.NewNop(),
		mockDunningService,
		mockEmailService,
		mockDelegationClient,
	)
	ctx := context.Background()

	campaignID := uuid.New()
	configID := uuid.New()
	customerID := uuid.New()
	workspaceID := uuid.New()

	tests := []struct {
		name       string
		limit      int32
		setupMocks func()
		wantErr    bool
		errString  string
	}{
		{
			name:  "successfully processes campaigns",
			limit: 10,
			setupMocks: func() {
				campaigns := []db.DunningCampaign{
					{
						ID:                  campaignID,
						WorkspaceID:         workspaceID,
						CustomerID:          customerID,
						ConfigurationID:     configID,
						CurrentAttempt:      1,
						OriginalAmountCents: 2000,
						Currency:            "USD",
						Status:              "active",
						CreatedAt:           pgtype.Timestamptz{Time: time.Now(), Valid: true},
						LastRetryAt:         pgtype.Timestamptz{Time: time.Now().Add(-24 * time.Hour), Valid: true},
						NextRetryAt:         pgtype.Timestamptz{Time: time.Now().Add(-1 * time.Hour), Valid: true},
					},
				}

				mockQuerier.EXPECT().
					ListDunningCampaignsForRetry(ctx, int32(10)).
					Return(campaigns, nil).
					Times(1)

				// Mock for processCampaign calls
				mockQuerier.EXPECT().
					GetDunningCampaign(ctx, campaignID).
					Return(db.GetDunningCampaignRow{
						ID:                  campaignID,
						WorkspaceID:         workspaceID,
						CustomerID:          customerID,
						ConfigurationID:     configID,
						CurrentAttempt:      1,
						MaxRetryAttempts:    3,
						OriginalAmountCents: 2000,
						Currency:            "USD",
						Status:              "active",
						CustomerName:        pgtype.Text{String: "John Doe", Valid: true},
						CustomerEmail:       pgtype.Text{String: "john@example.com", Valid: true},
					}, nil).
					Times(2)

				// Mock configuration retrieval
				type LocalAttemptAction struct {
					Attempt         int32      `json:"attempt"`
					Actions         []string   `json:"actions"`
					EmailTemplateID *uuid.UUID `json:"email_template_id,omitempty"`
				}
				attemptActions, _ := json.Marshal([]LocalAttemptAction{
					{
						Attempt: 2,
						Actions: []string{"retry_payment", "email"},
					},
				})

				mockQuerier.EXPECT().
					GetDunningConfiguration(ctx, configID).
					Return(db.DunningConfiguration{
						ID:                configID,
						WorkspaceID:       workspaceID,
						MaxRetryAttempts:  3,
						RetryIntervalDays: []int32{1, 3, 7},
						AttemptActions:    attemptActions,
						FinalAction:       "cancel_subscription",
					}, nil).
					Times(1)

				// Mock email template retrieval (may be called multiple times for different actions)
				mockQuerier.EXPECT().
					GetDunningEmailTemplateByType(ctx, gomock.Any()).
					Return(db.DunningEmailTemplate{ID: uuid.New()}, nil).
					AnyTimes()

				// Mock attempt creation (called by DunningService.CreateAttempt)
				mockQuerier.EXPECT().
					CreateDunningAttempt(ctx, gomock.Any()).
					Return(db.DunningAttempt{ID: uuid.New()}, nil).
					AnyTimes()

				// Mock attempt update (success)
				mockQuerier.EXPECT().
					UpdateDunningAttempt(ctx, gomock.Any()).
					Return(db.DunningAttempt{}, nil).
					AnyTimes()

				// Mock campaign update
				mockQuerier.EXPECT().
					UpdateDunningCampaign(ctx, gomock.Any()).
					Return(db.DunningCampaign{}, nil).
					AnyTimes()
			},
			wantErr: false,
		},
		{
			name:  "fails when listing campaigns fails",
			limit: 10,
			setupMocks: func() {
				mockQuerier.EXPECT().
					ListDunningCampaignsForRetry(ctx, int32(10)).
					Return(nil, assert.AnError).
					Times(1)
			},
			wantErr:   true,
			errString: "failed to list campaigns for retry",
		},
		{
			name:  "processes empty campaign list",
			limit: 10,
			setupMocks: func() {
				mockQuerier.EXPECT().
					ListDunningCampaignsForRetry(ctx, int32(10)).
					Return([]db.DunningCampaign{}, nil).
					Times(1)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := engine.ProcessDueCampaigns(ctx, tt.limit)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDunningRetryEngine_ProcessCampaign(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockDunningService := services.NewDunningService(mockQuerier, zap.NewNop())
	mockEmailService := (*services.EmailService)(nil) // Can be nil for testing
	mockDelegationClient := &MockDelegationClient{}

	engine := services.NewDunningRetryEngine(
		mockQuerier,
		zap.NewNop(),
		mockDunningService,
		mockEmailService,
		mockDelegationClient,
	)
	ctx := context.Background()

	campaignID := uuid.New()
	configID := uuid.New()
	customerID := uuid.New()
	workspaceID := uuid.New()

	tests := []struct {
		name       string
		campaign   *db.DunningCampaign
		setupMocks func()
		wantErr    bool
		errString  string
	}{
		{
			name: "successfully processes campaign within max attempts",
			campaign: &db.DunningCampaign{
				ID:                  campaignID,
				WorkspaceID:         workspaceID,
				CustomerID:          customerID,
				ConfigurationID:     configID,
				CurrentAttempt:      1,
				OriginalAmountCents: 2000,
				Currency:            "USD",
			},
			setupMocks: func() {
				// Mock listing campaigns first (ProcessDueCampaigns calls this)
				campaigns := []db.DunningCampaign{
					{
						ID:                  campaignID,
						WorkspaceID:         workspaceID,
						CustomerID:          customerID,
						ConfigurationID:     configID,
						CurrentAttempt:      1,
						OriginalAmountCents: 2000,
						Currency:            "USD",
						Status:              "active",
						CreatedAt:           pgtype.Timestamptz{Time: time.Now(), Valid: true},
						LastRetryAt:         pgtype.Timestamptz{Time: time.Now().Add(-24 * time.Hour), Valid: true},
						NextRetryAt:         pgtype.Timestamptz{Time: time.Now().Add(-1 * time.Hour), Valid: true},
					},
				}
				mockQuerier.EXPECT().
					ListDunningCampaignsForRetry(ctx, int32(1)).
					Return(campaigns, nil).
					Times(1)

				// Mock getting campaign details
				mockQuerier.EXPECT().
					GetDunningCampaign(ctx, campaignID).
					Return(db.GetDunningCampaignRow{
						ID:                  campaignID,
						WorkspaceID:         workspaceID,
						CustomerID:          customerID,
						ConfigurationID:     configID,
						CurrentAttempt:      1,
						MaxRetryAttempts:    3,
						OriginalAmountCents: 2000,
						Currency:            "USD",
						Status:              "active",
						CustomerName:        pgtype.Text{String: "John Doe", Valid: true},
						CustomerEmail:       pgtype.Text{String: "john@example.com", Valid: true},
					}, nil).
					Times(1)

				// Mock configuration retrieval
				type LocalAttemptAction struct {
					Attempt         int32      `json:"attempt"`
					Actions         []string   `json:"actions"`
					EmailTemplateID *uuid.UUID `json:"email_template_id,omitempty"`
				}
				attemptActions, _ := json.Marshal([]LocalAttemptAction{
					{
						Attempt: 2,
						Actions: []string{"retry_payment"},
					},
				})

				mockQuerier.EXPECT().
					GetDunningConfiguration(ctx, configID).
					Return(db.DunningConfiguration{
						ID:                configID,
						MaxRetryAttempts:  3,
						RetryIntervalDays: []int32{1, 3, 7},
						AttemptActions:    attemptActions,
						FinalAction:       "cancel_subscription",
					}, nil).
					Times(1)

				// Mock email template retrieval (may be called multiple times for different actions)
				mockQuerier.EXPECT().
					GetDunningEmailTemplateByType(ctx, gomock.Any()).
					Return(db.DunningEmailTemplate{ID: uuid.New()}, nil).
					AnyTimes()

				// Mock attempt creation (called by DunningService.CreateAttempt)
				mockQuerier.EXPECT().
					CreateDunningAttempt(ctx, gomock.Any()).
					Return(db.DunningAttempt{ID: uuid.New()}, nil).
					AnyTimes()

				// Mock attempt update (success)
				mockQuerier.EXPECT().
					UpdateDunningAttempt(ctx, gomock.Any()).
					Return(db.DunningAttempt{}, nil).
					AnyTimes()

				// Mock campaign update
				mockQuerier.EXPECT().
					UpdateDunningCampaign(ctx, gomock.Any()).
					Return(db.DunningCampaign{}, nil).
					AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "handles campaign processing failure gracefully",
			campaign: &db.DunningCampaign{
				ID:             campaignID,
				CurrentAttempt: 1,
			},
			setupMocks: func() {
				// Mock listing campaigns first (ProcessDueCampaigns calls this)
				campaigns := []db.DunningCampaign{
					{
						ID:                  campaignID,
						WorkspaceID:         workspaceID,
						CustomerID:          customerID,
						ConfigurationID:     configID,
						CurrentAttempt:      1,
						OriginalAmountCents: 2000,
						Currency:            "USD",
						Status:              "active",
					},
				}
				mockQuerier.EXPECT().
					ListDunningCampaignsForRetry(ctx, int32(1)).
					Return(campaigns, nil).
					Times(1)

				// This will cause processCampaign to fail, but ProcessDueCampaigns continues
				mockQuerier.EXPECT().
					GetDunningCampaign(ctx, campaignID).
					Return(db.GetDunningCampaignRow{}, assert.AnError).
					Times(1)
			},
			wantErr:   false, // ProcessDueCampaigns doesn't return errors from individual campaign failures
			errString: "",
		},
		{
			name: "handles configuration retrieval failure gracefully",
			campaign: &db.DunningCampaign{
				ID:              campaignID,
				ConfigurationID: configID,
				CurrentAttempt:  1,
			},
			setupMocks: func() {
				// Mock listing campaigns first (ProcessDueCampaigns calls this)
				campaigns := []db.DunningCampaign{
					{
						ID:                  campaignID,
						WorkspaceID:         workspaceID,
						CustomerID:          customerID,
						ConfigurationID:     configID,
						CurrentAttempt:      1,
						OriginalAmountCents: 2000,
						Currency:            "USD",
						Status:              "active",
					},
				}
				mockQuerier.EXPECT().
					ListDunningCampaignsForRetry(ctx, int32(1)).
					Return(campaigns, nil).
					Times(1)

				mockQuerier.EXPECT().
					GetDunningCampaign(ctx, campaignID).
					Return(db.GetDunningCampaignRow{
						ID:               campaignID,
						ConfigurationID:  configID,
						MaxRetryAttempts: 3,
					}, nil).
					Times(1)

				// This will cause processCampaign to fail, but ProcessDueCampaigns continues
				mockQuerier.EXPECT().
					GetDunningConfiguration(ctx, configID).
					Return(db.DunningConfiguration{}, assert.AnError).
					Times(1)
			},
			wantErr:   false, // ProcessDueCampaigns doesn't return errors from individual campaign failures
			errString: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			// Use reflection to call the private method
			// Since processCampaign is private, we test it through ProcessDueCampaigns
			// For this test, we'll focus on the public interface and mock the dependencies
			err := engine.ProcessDueCampaigns(ctx, 1)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
			} else {
				// This test structure would need adjustment for testing private methods
				// For now, we verify no error occurs when processing campaigns
				assert.NoError(t, err)
			}
		})
	}
}

func TestDunningRetryEngine_MonitorFailedPayments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockDunningService := services.NewDunningService(mockQuerier, zap.NewNop())
	mockEmailService := (*services.EmailService)(nil) // Can be nil for testing
	mockDelegationClient := &MockDelegationClient{}

	engine := services.NewDunningRetryEngine(
		mockQuerier,
		zap.NewNop(),
		mockDunningService,
		mockEmailService,
		mockDelegationClient,
	)
	ctx := context.Background()

	tests := []struct {
		name       string
		setupMocks func()
		wantErr    bool
		errString  string
	}{
		{
			name: "successfully monitors failed payments",
			setupMocks: func() {
				// Currently this is a placeholder method, so no mocks needed
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := engine.MonitorFailedPayments(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDunningRetryEngine_NewDunningRetryEngine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	mockDunningService := services.NewDunningService(mockQuerier, zap.NewNop())
	mockEmailService := (*services.EmailService)(nil) // Can be nil for testing
	mockDelegationClient := &MockDelegationClient{}

	tests := []struct {
		name             string
		queries          db.Querier
		logger           *zap.Logger
		dunningService   *services.DunningService
		emailService     *services.EmailService
		delegationClient services.DelegationClientInterface
		expectNonNil     bool
	}{
		{
			name:             "successfully creates engine with all dependencies",
			queries:          mockQuerier,
			logger:           zap.NewNop(),
			dunningService:   mockDunningService,
			emailService:     mockEmailService,
			delegationClient: mockDelegationClient,
			expectNonNil:     true,
		},
		{
			name:             "successfully creates engine with nil delegation client",
			queries:          mockQuerier,
			logger:           zap.NewNop(),
			dunningService:   mockDunningService,
			emailService:     mockEmailService,
			delegationClient: nil,
			expectNonNil:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := services.NewDunningRetryEngine(
				tt.queries,
				tt.logger,
				tt.dunningService,
				tt.emailService,
				tt.delegationClient,
			)

			if tt.expectNonNil {
				assert.NotNil(t, engine)
			} else {
				assert.Nil(t, engine)
			}
		})
	}
}

// Helper function for testing (renamed to avoid conflict)
func dunningStringPtr(s string) *string {
	return &s
}
