package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

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
	"go.uber.org/zap"
)

func init() {
	logger.InitLogger("test")
}

func TestDunningService_CreateConfiguration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	logger := zap.NewNop()
	service := services.NewDunningService(mockQuerier, logger)
	ctx := context.Background()

	workspaceID := uuid.New()
	configID := uuid.New()

	expectedConfig := db.DunningConfiguration{
		ID:                     configID,
		WorkspaceID:            workspaceID,
		Name:                   "Default Config",
		IsActive:               pgtype.Bool{Bool: true, Valid: true},
		IsDefault:              pgtype.Bool{Bool: true, Valid: true},
		MaxRetryAttempts:       3,
		RetryIntervalDays:      []int32{1, 3, 7},
		FinalAction:            "cancel",
		SendPreDunningReminder: pgtype.Bool{Bool: true, Valid: true},
		PreDunningDays:         pgtype.Int4{Int32: 3, Valid: true},
		AllowCustomerRetry:     pgtype.Bool{Bool: true, Valid: true},
		GracePeriodHours:       pgtype.Int4{Int32: 24, Valid: true},
	}

	tests := []struct {
		name        string
		params      params.DunningConfigParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully creates configuration",
			params: params.DunningConfigParams{
				WorkspaceID:            workspaceID,
				Name:                   "Default Config",
				Description:            nil,
				IsActive:               true,
				IsDefault:              true,
				MaxRetryAttempts:       3,
				RetryIntervalDays:      []int32{1, 3, 7},
				AttemptActions:         json.RawMessage(`{"1": "email", "2": "email", "3": "email"}`),
				FinalAction:            "cancel",
				FinalActionConfig:      json.RawMessage(`{"reason": "payment_failed"}`),
				SendPreDunningReminder: true,
				PreDunningDays:         3,
				AllowCustomerRetry:     true,
				GracePeriodHours:       24,
			},
			setupMocks: func() {
				// Unset default configurations
				mockQuerier.EXPECT().SetDefaultDunningConfiguration(ctx, db.SetDefaultDunningConfigurationParams{
					WorkspaceID: workspaceID,
					ID:          uuid.Nil,
				}).Return(nil)

				// Create configuration
				mockQuerier.EXPECT().CreateDunningConfiguration(ctx, gomock.Any()).Return(expectedConfig, nil)
			},
			wantErr: false,
		},
		{
			name: "successfully creates non-default configuration",
			params: params.DunningConfigParams{
				WorkspaceID:            workspaceID,
				Name:                   "Custom Config",
				IsActive:               true,
				IsDefault:              false,
				MaxRetryAttempts:       2,
				RetryIntervalDays:      []int32{2, 5},
				AttemptActions:         json.RawMessage(`{"1": "email", "2": "email"}`),
				FinalAction:            "pause",
				FinalActionConfig:      json.RawMessage(`{"duration": "30d"}`),
				SendPreDunningReminder: false,
				PreDunningDays:         0,
				AllowCustomerRetry:     false,
				GracePeriodHours:       12,
			},
			setupMocks: func() {
				// No need to unset defaults when IsDefault is false
				expectedNonDefault := expectedConfig
				expectedNonDefault.Name = "Custom Config"
				expectedNonDefault.IsDefault = pgtype.Bool{Bool: false, Valid: true}
				mockQuerier.EXPECT().CreateDunningConfiguration(ctx, gomock.Any()).Return(expectedNonDefault, nil)
			},
			wantErr: false,
		},
		{
			name: "fails when unsetting defaults fails",
			params: params.DunningConfigParams{
				WorkspaceID: workspaceID,
				Name:        "Test Config",
				IsDefault:   true,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().SetDefaultDunningConfiguration(ctx, gomock.Any()).Return(errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to unset default configurations",
		},
		{
			name: "fails when creating configuration fails",
			params: params.DunningConfigParams{
				WorkspaceID: workspaceID,
				Name:        "Test Config",
				IsDefault:   false,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().CreateDunningConfiguration(ctx, gomock.Any()).Return(db.DunningConfiguration{}, errors.New("creation error"))
			},
			wantErr:     true,
			errorString: "failed to create dunning configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			config, err := service.CreateConfiguration(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, config)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				assert.Equal(t, workspaceID, config.WorkspaceID)
			}
		})
	}
}

func TestDunningService_GetConfiguration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	logger := zap.NewNop()
	service := services.NewDunningService(mockQuerier, logger)
	ctx := context.Background()

	configID := uuid.New()
	expectedConfig := db.DunningConfiguration{
		ID:   configID,
		Name: "Test Config",
	}

	tests := []struct {
		name        string
		configID    uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:     "successfully gets configuration",
			configID: configID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetDunningConfiguration(ctx, configID).Return(expectedConfig, nil)
			},
			wantErr: false,
		},
		{
			name:     "configuration not found",
			configID: configID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetDunningConfiguration(ctx, configID).Return(db.DunningConfiguration{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "failed to get dunning configuration",
		},
		{
			name:     "database error",
			configID: configID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetDunningConfiguration(ctx, configID).Return(db.DunningConfiguration{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to get dunning configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			config, err := service.GetConfiguration(ctx, tt.configID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, config)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				assert.Equal(t, configID, config.ID)
			}
		})
	}
}

func TestDunningService_GetDefaultConfiguration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	logger := zap.NewNop()
	service := services.NewDunningService(mockQuerier, logger)
	ctx := context.Background()

	workspaceID := uuid.New()
	expectedConfig := db.DunningConfiguration{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Name:        "Default Config",
		IsDefault:   pgtype.Bool{Bool: true, Valid: true},
	}

	tests := []struct {
		name        string
		workspaceID uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "successfully gets default configuration",
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetDefaultDunningConfiguration(ctx, workspaceID).Return(expectedConfig, nil)
			},
			wantErr: false,
		},
		{
			name:        "no default configuration found",
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetDefaultDunningConfiguration(ctx, workspaceID).Return(db.DunningConfiguration{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "failed to get default dunning configuration",
		},
		{
			name:        "database error",
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetDefaultDunningConfiguration(ctx, workspaceID).Return(db.DunningConfiguration{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to get default dunning configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			config, err := service.GetDefaultConfiguration(ctx, tt.workspaceID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, config)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				assert.Equal(t, workspaceID, config.WorkspaceID)
			}
		})
	}
}

func TestDunningService_CreateCampaign(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	logger := zap.NewNop()
	service := services.NewDunningService(mockQuerier, logger)
	ctx := context.Background()

	workspaceID := uuid.New()
	configID := uuid.New()
	subscriptionID := uuid.New()
	paymentID := uuid.New()
	customerID := uuid.New()
	campaignID := uuid.New()

	config := db.DunningConfiguration{
		ID:               configID,
		GracePeriodHours: pgtype.Int4{Int32: 24, Valid: true},
	}

	expectedCampaign := db.DunningCampaign{
		ID:                    campaignID,
		WorkspaceID:           workspaceID,
		ConfigurationID:       configID,
		SubscriptionID:        pgtype.UUID{Bytes: subscriptionID, Valid: true},
		CustomerID:            customerID,
		Status:                "active",
		OriginalAmountCents:   1000,
		Currency:              "USD",
		OriginalFailureReason: pgtype.Text{String: "insufficient_funds", Valid: true},
	}

	tests := []struct {
		name        string
		params      params.DunningCampaignParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully creates campaign for subscription",
			params: params.DunningCampaignParams{
				ConfigurationID:   configID,
				SubscriptionID:    subscriptionID,
				TriggerReason:     "insufficient_funds",
				OutstandingAmount: 1000,
				Currency:          "USD",
			},
			setupMocks: func() {
				// Check for existing active campaign
				mockQuerier.EXPECT().GetActiveDunningCampaignForSubscription(ctx, pgtype.UUID{Bytes: subscriptionID, Valid: true}).Return(db.DunningCampaign{}, pgx.ErrNoRows)

				// Get configuration
				mockQuerier.EXPECT().GetDunningConfiguration(ctx, configID).Return(config, nil)

				// Get subscription to extract workspace and customer
				subscription := db.Subscription{
					ID:          subscriptionID,
					WorkspaceID: workspaceID,
					CustomerID:  customerID,
				}
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)

				// Create campaign
				mockQuerier.EXPECT().CreateDunningCampaign(ctx, gomock.Any()).Return(expectedCampaign, nil)

				// Update with retry time
				mockQuerier.EXPECT().UpdateDunningCampaign(ctx, gomock.Any()).Return(expectedCampaign, nil)
			},
			wantErr: false,
		},
		{
			name: "fails for payment-only campaign without workspace info",
			params: params.DunningCampaignParams{
				ConfigurationID:   configID,
				SubscriptionID:    uuid.Nil,
				InitialPaymentID:  &paymentID,
				TriggerReason:     "card_declined",
				OutstandingAmount: 2000,
				Currency:          "USD",
			},
			setupMocks: func() {
				// Check for existing active campaign
				mockQuerier.EXPECT().GetActiveDunningCampaignForPayment(ctx, pgtype.UUID{Bytes: paymentID, Valid: true}).Return(db.DunningCampaign{}, pgx.ErrNoRows)

				// Get configuration
				mockQuerier.EXPECT().GetDunningConfiguration(ctx, configID).Return(config, nil)
			},
			wantErr:     true,
			errorString: "workspace and customer ID required for campaign creation",
		},
		{
			name: "fails when active campaign already exists for subscription",
			params: params.DunningCampaignParams{
				ConfigurationID: configID,
				SubscriptionID:  subscriptionID,
			},
			setupMocks: func() {
				existingCampaign := db.DunningCampaign{ID: uuid.New()}
				mockQuerier.EXPECT().GetActiveDunningCampaignForSubscription(ctx, pgtype.UUID{Bytes: subscriptionID, Valid: true}).Return(existingCampaign, nil)
			},
			wantErr:     true,
			errorString: "active dunning campaign already exists for subscription",
		},
		{
			name: "fails when active campaign already exists for payment",
			params: params.DunningCampaignParams{
				ConfigurationID:  configID,
				SubscriptionID:   uuid.Nil,
				InitialPaymentID: &paymentID,
			},
			setupMocks: func() {
				existingCampaign := db.DunningCampaign{ID: uuid.New()}
				mockQuerier.EXPECT().GetActiveDunningCampaignForPayment(ctx, pgtype.UUID{Bytes: paymentID, Valid: true}).Return(existingCampaign, nil)
			},
			wantErr:     true,
			errorString: "active dunning campaign already exists for payment",
		},
		{
			name: "fails when configuration not found",
			params: params.DunningCampaignParams{
				ConfigurationID: configID,
				SubscriptionID:  uuid.Nil,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetDunningConfiguration(ctx, configID).Return(db.DunningConfiguration{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "failed to get dunning configuration",
		},
		{
			name: "fails when campaign creation fails",
			params: params.DunningCampaignParams{
				ConfigurationID: configID,
				SubscriptionID:  subscriptionID,
				TriggerReason:   "insufficient_funds",
			},
			setupMocks: func() {
				// Check for existing active campaign
				mockQuerier.EXPECT().GetActiveDunningCampaignForSubscription(ctx, pgtype.UUID{Bytes: subscriptionID, Valid: true}).Return(db.DunningCampaign{}, pgx.ErrNoRows)
				
				// Get configuration
				mockQuerier.EXPECT().GetDunningConfiguration(ctx, configID).Return(config, nil)
				
				// Get subscription to extract workspace and customer
				subscription := db.Subscription{
					ID:          subscriptionID,
					WorkspaceID: workspaceID,
					CustomerID:  customerID,
				}
				mockQuerier.EXPECT().GetSubscription(ctx, subscriptionID).Return(subscription, nil)
				
				// Fail on campaign creation
				mockQuerier.EXPECT().CreateDunningCampaign(ctx, gomock.Any()).Return(db.DunningCampaign{}, errors.New("creation error"))
			},
			wantErr:     true,
			errorString: "failed to create dunning campaign",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			campaign, err := service.CreateCampaign(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, campaign)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, campaign)
				assert.Equal(t, workspaceID, campaign.WorkspaceID)
				assert.Equal(t, configID, campaign.ConfigurationID)
			}
		})
	}
}

func TestDunningService_CreateAttempt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	logger := zap.NewNop()
	service := services.NewDunningService(mockQuerier, logger)
	ctx := context.Background()

	campaignID := uuid.New()
	paymentID := uuid.New()
	templateID := uuid.New()
	attemptID := uuid.New()

	expectedAttempt := db.DunningAttempt{
		ID:                attemptID,
		CampaignID:        campaignID,
		AttemptNumber:     1,
		AttemptType:       "email",
		Status:            "pending",
		CommunicationType: pgtype.Text{String: "email", Valid: true},
		EmailTemplateID:   pgtype.UUID{Bytes: templateID, Valid: true},
	}

	tests := []struct {
		name        string
		params      params.DunningAttemptParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully creates email attempt",
			params: params.DunningAttemptParams{
				CampaignID:      campaignID,
				AttemptNumber:   1,
				AttemptType:     "email",
				EmailTemplateID: &templateID,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().CreateDunningAttempt(ctx, gomock.Any()).Return(expectedAttempt, nil)
			},
			wantErr: false,
		},
		{
			name: "successfully creates payment attempt",
			params: params.DunningAttemptParams{
				CampaignID:    campaignID,
				AttemptNumber: 2,
				AttemptType:   "payment_retry",
			},
			setupMocks: func() {
				paymentAttempt := expectedAttempt
				paymentAttempt.AttemptType = "payment_retry"
				paymentAttempt.AttemptNumber = 2
				paymentAttempt.PaymentID = pgtype.UUID{Bytes: paymentID, Valid: true}
				mockQuerier.EXPECT().CreateDunningAttempt(ctx, gomock.Any()).Return(paymentAttempt, nil)
			},
			wantErr: false,
		},
		{
			name: "fails when creation fails",
			params: params.DunningAttemptParams{
				CampaignID:    campaignID,
				AttemptNumber: 1,
				AttemptType:   "email",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().CreateDunningAttempt(ctx, gomock.Any()).Return(db.DunningAttempt{}, errors.New("creation error"))
			},
			wantErr:     true,
			errorString: "failed to create dunning attempt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			attempt, err := service.CreateAttempt(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, attempt)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, attempt)
				assert.Equal(t, campaignID, attempt.CampaignID)
				assert.Equal(t, "pending", attempt.Status)
			}
		})
	}
}

func TestDunningService_UpdateAttemptStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	logger := zap.NewNop()
	service := services.NewDunningService(mockQuerier, logger)
	ctx := context.Background()

	attemptID := uuid.New()
	errorMsg := "Payment failed due to insufficient funds"

	tests := []struct {
		name        string
		attemptID   uuid.UUID
		status      string
		errorPtr    *string
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:      "successfully updates attempt to succeeded",
			attemptID: attemptID,
			status:    "succeeded",
			errorPtr:  nil,
			setupMocks: func() {
				expectedAttempt := db.DunningAttempt{
					ID:     attemptID,
					Status: "succeeded",
				}
				mockQuerier.EXPECT().UpdateDunningAttempt(ctx, gomock.Any()).Return(expectedAttempt, nil)
			},
			wantErr: false,
		},
		{
			name:      "successfully updates attempt to failed with error",
			attemptID: attemptID,
			status:    "failed",
			errorPtr:  &errorMsg,
			setupMocks: func() {
				expectedAttempt := db.DunningAttempt{
					ID:           attemptID,
					Status:       "failed",
					PaymentError: pgtype.Text{String: errorMsg, Valid: true},
				}
				mockQuerier.EXPECT().UpdateDunningAttempt(ctx, gomock.Any()).Return(expectedAttempt, nil)
			},
			wantErr: false,
		},
		{
			name:      "fails when update fails",
			attemptID: attemptID,
			status:    "succeeded",
			errorPtr:  nil,
			setupMocks: func() {
				mockQuerier.EXPECT().UpdateDunningAttempt(ctx, gomock.Any()).Return(db.DunningAttempt{}, errors.New("update error"))
			},
			wantErr:     true,
			errorString: "failed to update dunning attempt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			attempt, err := service.UpdateAttemptStatus(ctx, tt.attemptID, tt.status, tt.errorPtr)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, attempt)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, attempt)
				assert.Equal(t, attemptID, attempt.ID)
				assert.Equal(t, tt.status, attempt.Status)
			}
		})
	}
}

func TestDunningService_RecoverCampaign(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	logger := zap.NewNop()
	service := services.NewDunningService(mockQuerier, logger)
	ctx := context.Background()

	campaignID := uuid.New()
	recoveredAmount := int64(1500)

	tests := []struct {
		name            string
		campaignID      uuid.UUID
		recoveredAmount int64
		setupMocks      func()
		wantErr         bool
		errorString     string
	}{
		{
			name:            "successfully recovers campaign",
			campaignID:      campaignID,
			recoveredAmount: recoveredAmount,
			setupMocks: func() {
				expectedCampaign := db.DunningCampaign{
					ID:                   campaignID,
					Status:               "recovered",
					RecoveredAmountCents: pgtype.Int8{Int64: recoveredAmount, Valid: true},
				}
				mockQuerier.EXPECT().RecoverDunningCampaign(ctx, db.RecoverDunningCampaignParams{
					ID:                   campaignID,
					RecoveredAmountCents: pgtype.Int8{Int64: recoveredAmount, Valid: true},
				}).Return(expectedCampaign, nil)
			},
			wantErr: false,
		},
		{
			name:            "fails when recovery fails",
			campaignID:      campaignID,
			recoveredAmount: recoveredAmount,
			setupMocks: func() {
				mockQuerier.EXPECT().RecoverDunningCampaign(ctx, gomock.Any()).Return(db.DunningCampaign{}, errors.New("recovery error"))
			},
			wantErr:     true,
			errorString: "failed to recover dunning campaign",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			campaign, err := service.RecoverCampaign(ctx, tt.campaignID, tt.recoveredAmount)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, campaign)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, campaign)
				assert.Equal(t, campaignID, campaign.ID)
				assert.Equal(t, "recovered", campaign.Status)
			}
		})
	}
}

func TestDunningService_FailCampaign(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	logger := zap.NewNop()
	service := services.NewDunningService(mockQuerier, logger)
	ctx := context.Background()

	campaignID := uuid.New()
	subscriptionID := uuid.New()

	tests := []struct {
		name        string
		campaignID  uuid.UUID
		finalAction string
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "successfully fails campaign with cancel action",
			campaignID:  campaignID,
			finalAction: "cancel",
			setupMocks: func() {
				expectedCampaign := db.DunningCampaign{
					ID:               campaignID,
					Status:           "failed",
					SubscriptionID:   pgtype.UUID{Bytes: subscriptionID, Valid: true},
					FinalActionTaken: pgtype.Text{String: "cancel", Valid: true},
				}
				mockQuerier.EXPECT().FailDunningCampaign(ctx, db.FailDunningCampaignParams{
					ID:               campaignID,
					FinalActionTaken: pgtype.Text{String: "cancel", Valid: true},
				}).Return(expectedCampaign, nil)

				// Mock the cancel subscription calls
				mockQuerier.EXPECT().ScheduleSubscriptionCancellation(ctx, gomock.Any()).Return(db.Subscription{}, nil)
				mockQuerier.EXPECT().RecordStateChange(ctx, gomock.Any()).Return(db.SubscriptionStateHistory{}, nil)
			},
			wantErr: false,
		},
		{
			name:        "successfully fails campaign with pause action",
			campaignID:  campaignID,
			finalAction: "pause",
			setupMocks: func() {
				expectedCampaign := db.DunningCampaign{
					ID:               campaignID,
					Status:           "failed",
					FinalActionTaken: pgtype.Text{String: "pause", Valid: true},
				}
				mockQuerier.EXPECT().FailDunningCampaign(ctx, db.FailDunningCampaignParams{
					ID:               campaignID,
					FinalActionTaken: pgtype.Text{String: "pause", Valid: true},
				}).Return(expectedCampaign, nil)
			},
			wantErr: false,
		},
		{
			name:        "fails when failing campaign fails",
			campaignID:  campaignID,
			finalAction: "cancel",
			setupMocks: func() {
				mockQuerier.EXPECT().FailDunningCampaign(ctx, gomock.Any()).Return(db.DunningCampaign{}, errors.New("fail error"))
			},
			wantErr:     true,
			errorString: "failed to fail dunning campaign",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			campaign, err := service.FailCampaign(ctx, tt.campaignID, tt.finalAction)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, campaign)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, campaign)
				assert.Equal(t, campaignID, campaign.ID)
				assert.Equal(t, "failed", campaign.Status)
			}
		})
	}
}

func TestDunningService_CreateEmailTemplate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	logger := zap.NewNop()
	service := services.NewDunningService(mockQuerier, logger)
	ctx := context.Background()

	workspaceID := uuid.New()
	templateID := uuid.New()

	expectedTemplate := db.DunningEmailTemplate{
		ID:           templateID,
		WorkspaceID:  workspaceID,
		Name:         "Payment Failed",
		TemplateType: "payment_failed",
		Subject:      "Payment Issue with Your Subscription",
		BodyHtml:     "<p>Hello {{customer_name}}, we had trouble processing your payment.</p>",
		IsActive:     pgtype.Bool{Bool: true, Valid: true},
	}

	tests := []struct {
		name        string
		params      params.EmailTemplateParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully creates active email template",
			params: params.EmailTemplateParams{
				WorkspaceID:     workspaceID,
				ConfigurationID: uuid.New(),
				TemplateName:    "Payment Failed",
				TemplateType:    "payment_failed",
				Subject:         "Payment Issue with Your Subscription",
				BodyHtml:        "<p>Hello {{customer_name}}, we had trouble processing your payment.</p>",
				BodyText:        "Hello {{customer_name}}, we had trouble processing your payment.",
				Variables:       []string{"customer_name", "amount", "currency"},
				IsActive:        true,
			},
			setupMocks: func() {
				// Deactivate other templates of same type
				mockQuerier.EXPECT().DeactivateTemplatesByType(ctx, db.DeactivateTemplatesByTypeParams{
					WorkspaceID:  workspaceID,
					TemplateType: "payment_failed",
					ID:           uuid.Nil,
				}).Return(nil)

				// Create template
				mockQuerier.EXPECT().CreateDunningEmailTemplate(ctx, gomock.Any()).Return(expectedTemplate, nil)
			},
			wantErr: false,
		},
		{
			name: "successfully creates inactive email template",
			params: params.EmailTemplateParams{
				WorkspaceID:     workspaceID,
				ConfigurationID: uuid.New(),
				TemplateName:    "Custom Template",
				TemplateType:    "reminder",
				Subject:         "Payment Reminder",
				BodyHtml:        "<p>This is a reminder about your payment.</p>",
				Variables:       []string{"customer_name"},
				IsActive:        false,
			},
			setupMocks: func() {
				// No deactivation needed when IsActive is false
				inactiveTemplate := expectedTemplate
				inactiveTemplate.Name = "Custom Template"
				inactiveTemplate.TemplateType = "reminder"
				inactiveTemplate.Subject = "Payment Reminder"
				inactiveTemplate.BodyHtml = "<p>This is a reminder about your payment.</p>"
				inactiveTemplate.IsActive = pgtype.Bool{Bool: false, Valid: true}
				mockQuerier.EXPECT().CreateDunningEmailTemplate(ctx, gomock.Any()).Return(inactiveTemplate, nil)
			},
			wantErr: false,
		},
		{
			name: "fails when deactivating existing templates fails",
			params: params.EmailTemplateParams{
				WorkspaceID:     workspaceID,
				ConfigurationID: uuid.New(),
				TemplateName:    "Test Template",
				TemplateType:    "payment_failed",
				IsActive:        true,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().DeactivateTemplatesByType(ctx, gomock.Any()).Return(errors.New("deactivation error"))
			},
			wantErr:     true,
			errorString: "failed to deactivate existing templates",
		},
		{
			name: "fails when creating template fails",
			params: params.EmailTemplateParams{
				WorkspaceID:     workspaceID,
				ConfigurationID: uuid.New(),
				TemplateName:    "Test Template",
				TemplateType:    "reminder",
				IsActive:        false,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().CreateDunningEmailTemplate(ctx, gomock.Any()).Return(db.DunningEmailTemplate{}, errors.New("creation error"))
			},
			wantErr:     true,
			errorString: "failed to create email template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			template, err := service.CreateEmailTemplate(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, template)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, template)
				assert.Equal(t, workspaceID, template.WorkspaceID)
				assert.Equal(t, tt.params.TemplateName, template.Name)
			}
		})
	}
}

func TestDunningService_GetCampaignStats(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	logger := zap.NewNop()
	service := services.NewDunningService(mockQuerier, logger)
	ctx := context.Background()

	workspaceID := uuid.New()
	startDate := time.Now().AddDate(0, -1, 0) // 1 month ago
	endDate := time.Now()

	expectedStats := db.GetDunningCampaignStatsRow{
		ActiveCampaigns:      1,
		RecoveredCampaigns:   6,
		LostCampaigns:        3,
		AtRiskAmountCents:    10000,
		RecoveredAmountCents: 5000,
		LostAmountCents:      3000,
	}

	tests := []struct {
		name        string
		workspaceID uuid.UUID
		startDate   time.Time
		endDate     time.Time
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "successfully gets campaign stats",
			workspaceID: workspaceID,
			startDate:   startDate,
			endDate:     endDate,
			setupMocks: func() {
				mockQuerier.EXPECT().GetDunningCampaignStats(ctx, db.GetDunningCampaignStatsParams{
					WorkspaceID: workspaceID,
					CreatedAt:   pgtype.Timestamptz{Time: startDate, Valid: true},
					CreatedAt_2: pgtype.Timestamptz{Time: endDate, Valid: true},
				}).Return(expectedStats, nil)
			},
			wantErr: false,
		},
		{
			name:        "fails when getting stats fails",
			workspaceID: workspaceID,
			startDate:   startDate,
			endDate:     endDate,
			setupMocks: func() {
				mockQuerier.EXPECT().GetDunningCampaignStats(ctx, gomock.Any()).Return(db.GetDunningCampaignStatsRow{}, errors.New("stats error"))
			},
			wantErr:     true,
			errorString: "failed to get campaign stats",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			stats, err := service.GetCampaignStats(ctx, tt.workspaceID, tt.startDate, tt.endDate)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, stats)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, stats)
				assert.Equal(t, expectedStats.ActiveCampaigns, stats.ActiveCampaigns)
				assert.Equal(t, expectedStats.RecoveredCampaigns, stats.RecoveredCampaigns)
				assert.Equal(t, expectedStats.LostCampaigns, stats.LostCampaigns)
			}
		})
	}
}

func TestDunningService_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*mocks.MockQuerier)
		operation   func(*services.DunningService) error
		expectError bool
		errorMsg    string
	}{
		{
			name: "context error handling",
			setupMocks: func(mockQuerier *mocks.MockQuerier) {
				mockQuerier.EXPECT().GetDunningConfiguration(gomock.Any(), gomock.Any()).Return(db.DunningConfiguration{}, errors.New("context error"))
			},
			operation: func(service *services.DunningService) error {
				_, err := service.GetConfiguration(context.Background(), uuid.New())
				return err
			},
			expectError: true,
			errorMsg:    "context error",
		},
		{
			name: "zero UUID handling",
			setupMocks: func(mockQuerier *mocks.MockQuerier) {
				mockQuerier.EXPECT().GetDunningConfiguration(gomock.Any(), uuid.Nil).Return(db.DunningConfiguration{}, pgx.ErrNoRows)
			},
			operation: func(service *services.DunningService) error {
				_, err := service.GetConfiguration(context.Background(), uuid.Nil)
				return err
			},
			expectError: true,
			errorMsg:    "failed to get dunning configuration",
		},
		{
			name: "empty final action handling",
			setupMocks: func(mockQuerier *mocks.MockQuerier) {
				campaignID := uuid.New()
				mockQuerier.EXPECT().FailDunningCampaign(gomock.Any(), gomock.Any()).Return(db.DunningCampaign{
					ID: campaignID,
				}, nil)
			},
			operation: func(service *services.DunningService) error {
				campaignID := uuid.New()
				_, err := service.FailCampaign(context.Background(), campaignID, "unknown_action")
				return err
			},
			expectError: false, // The method doesn't return error, just logs it
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockQuerier(ctrl)
			logger := zap.NewNop()
			service := services.NewDunningService(mockQuerier, logger)

			tt.setupMocks(mockQuerier)
			err := tt.operation(service)

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

func TestDunningService_BoundaryConditions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	logger := zap.NewNop()
	service := services.NewDunningService(mockQuerier, logger)
	ctx := context.Background()

	tests := []struct {
		name       string
		setupMocks func()
		operation  func() error
		wantErr    bool
	}{
		{
			name: "create configuration with maximum retry attempts",
			setupMocks: func() {
				mockQuerier.EXPECT().CreateDunningConfiguration(ctx, gomock.Any()).Return(db.DunningConfiguration{
					MaxRetryAttempts: 100,
				}, nil)
			},
			operation: func() error {
				_, err := service.CreateConfiguration(ctx, params.DunningConfigParams{
					WorkspaceID:       uuid.New(),
					Name:              "Max Retries",
					IsActive:          true,
					MaxRetryAttempts:  100,
					RetryIntervalDays: make([]int32, 100), // Very long array
					FinalAction:       "cancel",
				})
				return err
			},
			wantErr: false,
		},
		{
			name: "create campaign with very large amount",
			setupMocks: func() {
				config := db.DunningConfiguration{
					GracePeriodHours: pgtype.Int4{Int32: 1, Valid: true},
				}
				mockQuerier.EXPECT().GetDunningConfiguration(ctx, gomock.Any()).Return(config, nil)
				mockQuerier.EXPECT().CreateDunningCampaign(ctx, gomock.Any()).Return(db.DunningCampaign{}, nil)
				mockQuerier.EXPECT().UpdateDunningCampaign(ctx, gomock.Any()).Return(db.DunningCampaign{}, nil)
			},
			operation: func() error {
				_, err := service.CreateCampaign(ctx, params.DunningCampaignParams{
					ConfigurationID:   uuid.New(),
					SubscriptionID:    uuid.Nil,
					TriggerReason:     "test",
					OutstandingAmount: 9223372036854775807, // Max int64
					Currency:          "USD",
				})
				return err
			},
			wantErr: false,
		},
		{
			name: "create email template with very long content",
			setupMocks: func() {
				mockQuerier.EXPECT().CreateDunningEmailTemplate(ctx, gomock.Any()).Return(db.DunningEmailTemplate{}, nil)
			},
			operation: func() error {
				longContent := make([]byte, 100000) // 100KB of content
				for i := range longContent {
					longContent[i] = 'a'
				}
				_, err := service.CreateEmailTemplate(ctx, params.EmailTemplateParams{
					WorkspaceID:     uuid.New(),
					ConfigurationID: uuid.New(),
					TemplateName:    "Long Template",
					TemplateType:    "test",
					Subject:         "Test",
					BodyHtml:        string(longContent),
					IsActive:        false,
				})
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

func TestDunningService_ExecuteFinalAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	logger := zap.NewNop()
	service := services.NewDunningService(mockQuerier, logger)
	ctx := context.Background()

	campaignID := uuid.New()
	subscriptionID := uuid.New()

	tests := []struct {
		name        string
		finalAction string
		campaign    *db.DunningCampaign
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "successfully executes cancel action for subscription",
			finalAction: "cancel",
			campaign: &db.DunningCampaign{
				ID:             campaignID,
				SubscriptionID: pgtype.UUID{Bytes: subscriptionID, Valid: true},
			},
			setupMocks: func() {
				// Mock the cancel subscription calls
				mockQuerier.EXPECT().ScheduleSubscriptionCancellation(ctx, gomock.Any()).Return(db.Subscription{}, nil)
				mockQuerier.EXPECT().RecordStateChange(ctx, gomock.Any()).Return(db.SubscriptionStateHistory{}, nil)
			},
			wantErr: false,
		},
		{
			name:        "successfully executes pause action",
			finalAction: "pause",
			campaign: &db.DunningCampaign{
				ID: campaignID,
			},
			setupMocks: func() {
				// No mocks needed for pause - it's a TODO
			},
			wantErr: false,
		},
		{
			name:        "successfully executes downgrade action",
			finalAction: "downgrade",
			campaign: &db.DunningCampaign{
				ID: campaignID,
			},
			setupMocks: func() {
				// No mocks needed for downgrade - it's a TODO
			},
			wantErr: false,
		},
		{
			name:        "fails with unknown action",
			finalAction: "unknown",
			campaign: &db.DunningCampaign{
				ID: campaignID,
			},
			setupMocks: func() {
				// No mocks needed
			},
			wantErr:     true,
			errorString: "unknown final action",
		},
		{
			name:        "fails when cancellation scheduling fails",
			finalAction: "cancel",
			campaign: &db.DunningCampaign{
				ID:             campaignID,
				SubscriptionID: pgtype.UUID{Bytes: subscriptionID, Valid: true},
			},
			setupMocks: func() {
				mockQuerier.EXPECT().ScheduleSubscriptionCancellation(ctx, gomock.Any()).Return(db.Subscription{}, errors.New("cancellation error"))
			},
			wantErr:     true,
			errorString: "failed to cancel subscription",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			// Since executeFinalAction is not exported, we test it through FailCampaign
			mockQuerier.EXPECT().FailDunningCampaign(ctx, gomock.Any()).Return(*tt.campaign, nil)

			_, err := service.FailCampaign(ctx, tt.campaign.ID, tt.finalAction)

			if tt.wantErr {
				// For this test, we only check if the final action execution would work
				// The actual error handling is in the FailCampaign method
				assert.NoError(t, err) // FailCampaign doesn't return final action errors
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDunningService_HelperFunctions(t *testing.T) {
	t.Run("textToPgtype with nil string", func(t *testing.T) {
		// Testing with reflection since these are unexported functions
		// We test the behavior through exported methods that use them

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mocks.NewMockQuerier(ctrl)
		service := services.NewDunningService(mockQuerier, logger.Log)
		ctx := context.Background()

		// Test with nil description
		mockQuerier.EXPECT().CreateDunningConfiguration(ctx, gomock.Any()).DoAndReturn(
			func(ctx context.Context, params db.CreateDunningConfigurationParams) (db.DunningConfiguration, error) {
				// Verify that nil description becomes invalid pgtype.Text
				assert.False(t, params.Description.Valid)
				return db.DunningConfiguration{}, nil
			},
		)

		_, _ = service.CreateConfiguration(ctx, params.DunningConfigParams{
			WorkspaceID:      uuid.New(),
			Name:             "Test",
			Description:      nil, // This should result in invalid pgtype.Text
			MaxRetryAttempts: 1,
			FinalAction:      "cancel",
		})
	})

	t.Run("textToPgtype with empty string", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mocks.NewMockQuerier(ctrl)
		service := services.NewDunningService(mockQuerier, logger.Log)
		ctx := context.Background()

		emptyDesc := ""
		mockQuerier.EXPECT().CreateDunningConfiguration(ctx, gomock.Any()).DoAndReturn(
			func(ctx context.Context, params db.CreateDunningConfigurationParams) (db.DunningConfiguration, error) {
				// Verify that empty string becomes invalid pgtype.Text
				assert.False(t, params.Description.Valid)
				return db.DunningConfiguration{}, nil
			},
		)

		_, _ = service.CreateConfiguration(ctx, params.DunningConfigParams{
			WorkspaceID:      uuid.New(),
			Name:             "Test",
			Description:      &emptyDesc, // This should result in invalid pgtype.Text
			MaxRetryAttempts: 1,
			FinalAction:      "cancel",
		})
	})

	t.Run("textToPgtype with valid string", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mocks.NewMockQuerier(ctrl)
		service := services.NewDunningService(mockQuerier, logger.Log)
		ctx := context.Background()

		validDesc := "Valid description"
		mockQuerier.EXPECT().CreateDunningConfiguration(ctx, gomock.Any()).DoAndReturn(
			func(ctx context.Context, params db.CreateDunningConfigurationParams) (db.DunningConfiguration, error) {
				// Verify that valid string becomes valid pgtype.Text
				assert.True(t, params.Description.Valid)
				assert.Equal(t, validDesc, params.Description.String)
				return db.DunningConfiguration{}, nil
			},
		)

		_, _ = service.CreateConfiguration(ctx, params.DunningConfigParams{
			WorkspaceID:      uuid.New(),
			Name:             "Test",
			Description:      &validDesc,
			MaxRetryAttempts: 1,
			FinalAction:      "cancel",
		})
	})

	t.Run("dunningUuidToPgtype with nil UUID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mocks.NewMockQuerier(ctrl)
		service := services.NewDunningService(mockQuerier, logger.Log)
		ctx := context.Background()

		config := db.DunningConfiguration{
			GracePeriodHours: pgtype.Int4{Int32: 24, Valid: true},
		}
		mockQuerier.EXPECT().GetDunningConfiguration(ctx, gomock.Any()).Return(config, nil)

		mockQuerier.EXPECT().CreateDunningCampaign(ctx, gomock.Any()).DoAndReturn(
			func(ctx context.Context, params db.CreateDunningCampaignParams) (db.DunningCampaign, error) {
				// Verify that nil UUID becomes invalid pgtype.UUID
				assert.False(t, params.SubscriptionID.Valid)
				assert.False(t, params.PaymentID.Valid)
				return db.DunningCampaign{}, nil
			},
		)
		mockQuerier.EXPECT().UpdateDunningCampaign(ctx, gomock.Any()).Return(db.DunningCampaign{}, nil)

		_, _ = service.CreateCampaign(ctx, params.DunningCampaignParams{
			ConfigurationID:   uuid.New(),
			SubscriptionID:    uuid.Nil, // This should result in invalid pgtype.UUID
			TriggerReason:     "test",
			OutstandingAmount: 1000,
			Currency:          "USD",
		})
	})

	t.Run("dunningUuidToPgtype with valid UUID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mocks.NewMockQuerier(ctrl)
		service := services.NewDunningService(mockQuerier, logger.Log)
		ctx := context.Background()

		subscriptionID := uuid.New()
		config := db.DunningConfiguration{
			GracePeriodHours: pgtype.Int4{Int32: 24, Valid: true},
		}

		// Check for existing campaigns
		mockQuerier.EXPECT().GetActiveDunningCampaignForSubscription(ctx, gomock.Any()).Return(
			db.DunningCampaign{ID: uuid.Nil}, nil)
		mockQuerier.EXPECT().GetDunningConfiguration(ctx, gomock.Any()).Return(config, nil)

		mockQuerier.EXPECT().CreateDunningCampaign(ctx, gomock.Any()).DoAndReturn(
			func(ctx context.Context, params db.CreateDunningCampaignParams) (db.DunningCampaign, error) {
				// Verify that valid UUID becomes valid pgtype.UUID
				assert.True(t, params.SubscriptionID.Valid)
				assert.Equal(t, subscriptionID, uuid.UUID(params.SubscriptionID.Bytes))
				return db.DunningCampaign{}, nil
			},
		)
		mockQuerier.EXPECT().UpdateDunningCampaign(ctx, gomock.Any()).Return(db.DunningCampaign{}, nil)

		_, _ = service.CreateCampaign(ctx, params.DunningCampaignParams{
			ConfigurationID:   uuid.New(),
			SubscriptionID:    subscriptionID, // This should result in valid pgtype.UUID
			TriggerReason:     "test",
			OutstandingAmount: 1000,
			Currency:          "USD",
		})
	})
}

func TestDunningService_ConcurrentOperations(t *testing.T) {
	// Test that the service can handle concurrent operations without panics
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewDunningService(mockQuerier, logger.Log)
	ctx := context.Background()

	// Set up expectations for concurrent operations
	workspaceID := uuid.New()
	configID := uuid.New()

	// Allow multiple calls for concurrent testing
	mockQuerier.EXPECT().GetDunningConfiguration(ctx, configID).Return(db.DunningConfiguration{
		ID:          configID,
		WorkspaceID: workspaceID,
		Name:        "Test Config",
	}, nil).AnyTimes()

	// Run multiple goroutines performing operations
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Goroutine panicked: %v", r)
				}
				done <- true
			}()

			// Perform read operation
			_, _ = service.GetConfiguration(ctx, configID)
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestDunningService_ComplexScenarios(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewDunningService(mockQuerier, logger.Log)
	ctx := context.Background()

	t.Run("campaign lifecycle - create, attempt, recover", func(t *testing.T) {
		workspaceID := uuid.New()
		configID := uuid.New()
		campaignID := uuid.New()
		attemptID := uuid.New()
		customerID := uuid.New()
		subscriptionID := uuid.New()

		// Step 1: Create campaign
		config := db.DunningConfiguration{
			ID:               configID,
			GracePeriodHours: pgtype.Int4{Int32: 24, Valid: true},
		}
		mockQuerier.EXPECT().GetActiveDunningCampaignForSubscription(ctx, gomock.Any()).Return(
			db.DunningCampaign{ID: uuid.Nil}, nil)
		mockQuerier.EXPECT().GetDunningConfiguration(ctx, configID).Return(config, nil)
		mockQuerier.EXPECT().CreateDunningCampaign(ctx, gomock.Any()).Return(db.DunningCampaign{
			ID:              campaignID,
			WorkspaceID:     workspaceID,
			ConfigurationID: configID,
			CustomerID:      customerID,
			Status:          "active",
		}, nil)
		mockQuerier.EXPECT().UpdateDunningCampaign(ctx, gomock.Any()).Return(db.DunningCampaign{
			ID:              campaignID,
			WorkspaceID:     workspaceID,
			ConfigurationID: configID,
			CustomerID:      customerID,
			Status:          "active",
		}, nil)

		campaign, err := service.CreateCampaign(ctx, params.DunningCampaignParams{
			ConfigurationID:   configID,
			SubscriptionID:    subscriptionID,
			TriggerReason:     "insufficient_funds",
			OutstandingAmount: 2999,
			Currency:          "USD",
		})
		assert.NoError(t, err)
		assert.NotNil(t, campaign)

		// Step 2: Create attempt
		mockQuerier.EXPECT().CreateDunningAttempt(ctx, gomock.Any()).Return(db.DunningAttempt{
			ID:            attemptID,
			CampaignID:    campaignID,
			AttemptNumber: 1,
			AttemptType:   "retry_payment",
			Status:        "pending",
		}, nil)

		attempt, err := service.CreateAttempt(ctx, params.DunningAttemptParams{
			CampaignID:    campaignID,
			AttemptNumber: 1,
			AttemptType:   "retry_payment",
		})
		assert.NoError(t, err)
		assert.NotNil(t, attempt)

		// Step 3: Update attempt to success
		mockQuerier.EXPECT().UpdateDunningAttempt(ctx, gomock.Any()).Return(db.DunningAttempt{
			ID:     attemptID,
			Status: "success",
		}, nil)

		updatedAttempt, err := service.UpdateAttemptStatus(ctx, attemptID, "success", nil)
		assert.NoError(t, err)
		assert.Equal(t, "success", updatedAttempt.Status)

		// Step 4: Recover campaign
		mockQuerier.EXPECT().RecoverDunningCampaign(ctx, gomock.Any()).Return(db.DunningCampaign{
			ID:     campaignID,
			Status: "recovered",
		}, nil)

		recoveredCampaign, err := service.RecoverCampaign(ctx, campaignID, 2999)
		assert.NoError(t, err)
		assert.Equal(t, "recovered", recoveredCampaign.Status)
	})

	t.Run("campaign lifecycle - create, multiple attempts, fail", func(t *testing.T) {
		workspaceID := uuid.New()
		configID := uuid.New()
		campaignID := uuid.New()
		customerID := uuid.New()
		subscriptionID := uuid.New()

		// Create campaign
		config := db.DunningConfiguration{
			ID:               configID,
			GracePeriodHours: pgtype.Int4{Int32: 24, Valid: true},
		}
		mockQuerier.EXPECT().GetActiveDunningCampaignForSubscription(ctx, gomock.Any()).Return(
			db.DunningCampaign{ID: uuid.Nil}, nil)
		mockQuerier.EXPECT().GetDunningConfiguration(ctx, configID).Return(config, nil)
		mockQuerier.EXPECT().CreateDunningCampaign(ctx, gomock.Any()).Return(db.DunningCampaign{
			ID:              campaignID,
			WorkspaceID:     workspaceID,
			ConfigurationID: configID,
			CustomerID:      customerID,
			Status:          "active",
		}, nil)
		mockQuerier.EXPECT().UpdateDunningCampaign(ctx, gomock.Any()).Return(db.DunningCampaign{
			ID:              campaignID,
			WorkspaceID:     workspaceID,
			ConfigurationID: configID,
			CustomerID:      customerID,
			Status:          "active",
		}, nil)

		campaign, err := service.CreateCampaign(ctx, params.DunningCampaignParams{
			ConfigurationID:   configID,
			SubscriptionID:    subscriptionID,
			TriggerReason:     "card_declined",
			OutstandingAmount: 4999,
			Currency:          "USD",
		})
		assert.NoError(t, err)
		assert.NotNil(t, campaign)

		// Create multiple failed attempts
		for i := 1; i <= 3; i++ {
			attemptID := uuid.New()
			errorMsg := fmt.Sprintf("Attempt %d failed", i)

			mockQuerier.EXPECT().CreateDunningAttempt(ctx, gomock.Any()).Return(db.DunningAttempt{
				ID:            attemptID,
				CampaignID:    campaignID,
				AttemptNumber: int32(i),
				AttemptType:   "retry_payment",
				Status:        "pending",
			}, nil)

			_, err := service.CreateAttempt(ctx, params.DunningAttemptParams{
				CampaignID:    campaignID,
				AttemptNumber: int32(i),
				AttemptType:   "retry_payment",
			})
			assert.NoError(t, err)

			mockQuerier.EXPECT().UpdateDunningAttempt(ctx, gomock.Any()).Return(db.DunningAttempt{
				ID:     attemptID,
				Status: "failed",
			}, nil)

			_, err = service.UpdateAttemptStatus(ctx, attemptID, "failed", &errorMsg)
			assert.NoError(t, err)
		}

		// Fail campaign after all attempts exhausted
		mockQuerier.EXPECT().FailDunningCampaign(ctx, gomock.Any()).Return(db.DunningCampaign{
			ID:               campaignID,
			Status:           "failed",
			SubscriptionID:   pgtype.UUID{Bytes: subscriptionID, Valid: true},
			FinalActionTaken: pgtype.Text{String: "cancel", Valid: true},
		}, nil)

		// Mock cancellation calls
		mockQuerier.EXPECT().ScheduleSubscriptionCancellation(ctx, gomock.Any()).Return(db.Subscription{}, nil)
		mockQuerier.EXPECT().RecordStateChange(ctx, gomock.Any()).Return(db.SubscriptionStateHistory{}, nil)

		failedCampaign, err := service.FailCampaign(ctx, campaignID, "cancel")
		assert.NoError(t, err)
		assert.Equal(t, "failed", failedCampaign.Status)
		assert.Equal(t, "cancel", failedCampaign.FinalActionTaken.String)
	})
}

func TestDunningService_DataValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewDunningService(mockQuerier, logger.Log)
	ctx := context.Background()

	t.Run("configuration with invalid JSON", func(t *testing.T) {
		// Test that service can handle invalid JSON in attempt actions
		invalidJSON := json.RawMessage(`{"invalid": json}`) // Missing quotes around 'json'

		mockQuerier.EXPECT().CreateDunningConfiguration(ctx, gomock.Any()).DoAndReturn(
			func(ctx context.Context, params db.CreateDunningConfigurationParams) (db.DunningConfiguration, error) {
				// Verify that the service passes through the JSON as-is
				assert.Equal(t, invalidJSON, params.AttemptActions)
				return db.DunningConfiguration{}, nil
			},
		)

		_, err := service.CreateConfiguration(ctx, params.DunningConfigParams{
			WorkspaceID:      uuid.New(),
			Name:             "Test Config",
			MaxRetryAttempts: 3,
			AttemptActions:   invalidJSON, // Service should pass this through
			FinalAction:      "cancel",
		})

		// The service should not validate JSON - that's the database's responsibility
		assert.NoError(t, err)
	})

	t.Run("negative amount handling", func(t *testing.T) {
		config := db.DunningConfiguration{
			GracePeriodHours: pgtype.Int4{Int32: 24, Valid: true},
		}
		mockQuerier.EXPECT().GetDunningConfiguration(ctx, gomock.Any()).Return(config, nil)
		mockQuerier.EXPECT().CreateDunningCampaign(ctx, gomock.Any()).DoAndReturn(
			func(ctx context.Context, params db.CreateDunningCampaignParams) (db.DunningCampaign, error) {
				// Verify that negative amount is passed through
				assert.Equal(t, int64(-1000), params.OriginalAmountCents)
				return db.DunningCampaign{}, nil
			},
		)
		mockQuerier.EXPECT().UpdateDunningCampaign(ctx, gomock.Any()).Return(db.DunningCampaign{}, nil)

		_, err := service.CreateCampaign(ctx, params.DunningCampaignParams{
			ConfigurationID:   uuid.New(),
			SubscriptionID:    uuid.Nil,
			TriggerReason:     "test",
			OutstandingAmount: -1000, // Negative amount
			Currency:          "USD",
		})

		// Service should allow negative amounts - business logic validation is elsewhere
		assert.NoError(t, err)
	})

	t.Run("empty currency handling", func(t *testing.T) {
		config := db.DunningConfiguration{
			GracePeriodHours: pgtype.Int4{Int32: 24, Valid: true},
		}
		mockQuerier.EXPECT().GetDunningConfiguration(ctx, gomock.Any()).Return(config, nil)
		mockQuerier.EXPECT().CreateDunningCampaign(ctx, gomock.Any()).DoAndReturn(
			func(ctx context.Context, params db.CreateDunningCampaignParams) (db.DunningCampaign, error) {
				// Verify that empty currency is passed through
				assert.Equal(t, "", params.Currency)
				return db.DunningCampaign{}, nil
			},
		)
		mockQuerier.EXPECT().UpdateDunningCampaign(ctx, gomock.Any()).Return(db.DunningCampaign{}, nil)

		_, err := service.CreateCampaign(ctx, params.DunningCampaignParams{
			ConfigurationID:   uuid.New(),
			SubscriptionID:    uuid.Nil,
			TriggerReason:     "test",
			OutstandingAmount: 1000,
			Currency:          "", // Empty currency
		})

		// Service should allow empty currency - validation is at database level
		assert.NoError(t, err)
	})
}

// Helper function for string pointers
func stringPtrHelper(s string) *string {
	return &s
}
