package services_test

import (
	"context"
	"testing"
	"time"

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

func TestErrorRecoveryService_ReplayWebhookEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewErrorRecoveryService(mockQuerier, logger.Log, nil)
	ctx := context.Background()

	workspaceID := uuid.New()
	eventID := uuid.New()
	webhookEventID := "webhook_123"

	tests := []struct {
		name       string
		request    requests.WebhookReplayRequest
		setupMocks func()
		wantErr    bool
		validate   func(*responses.WebhookReplayResponse)
	}{
		{
			name: "invalid workspace ID",
			request: requests.WebhookReplayRequest{
				WorkspaceID:    "invalid-uuid",
				ProviderName:   "stripe",
				WebhookEventID: webhookEventID,
			},
			setupMocks: func() {},
			wantErr:    false,
			validate: func(resp *responses.WebhookReplayResponse) {
				assert.False(t, resp.Success)
				assert.Equal(t, "invalid workspace ID format", resp.Error)
				assert.Equal(t, webhookEventID, resp.OriginalEventID)
			},
		},
		{
			name: "webhook event not found",
			request: requests.WebhookReplayRequest{
				WorkspaceID:    workspaceID.String(),
				ProviderName:   "stripe",
				WebhookEventID: webhookEventID,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetWebhookEventForReplay(ctx, db.GetWebhookEventForReplayParams{
						WorkspaceID:    workspaceID,
						ProviderName:   "stripe",
						WebhookEventID: pgtype.Text{String: webhookEventID, Valid: true},
					}).
					Return(db.GetWebhookEventForReplayRow{}, pgx.ErrNoRows)
			},
			wantErr: false,
			validate: func(resp *responses.WebhookReplayResponse) {
				assert.False(t, resp.Success)
				assert.Equal(t, "original webhook event not found", resp.Error)
			},
		},
		{
			name: "max retries exceeded without force",
			request: requests.WebhookReplayRequest{
				WorkspaceID:    workspaceID.String(),
				ProviderName:   "stripe",
				WebhookEventID: webhookEventID,
				ForceReplay:    false,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetWebhookEventForReplay(ctx, gomock.Any()).
					Return(db.GetWebhookEventForReplayRow{
						ID:                 eventID,
						ProcessingAttempts: pgtype.Int4{Int32: 3, Valid: true},
						EventDetails:       []byte(`{}`),
					}, nil)
			},
			wantErr: false,
			validate: func(resp *responses.WebhookReplayResponse) {
				assert.False(t, resp.Success)
				assert.Contains(t, resp.Error, "force_replay=true to override")
			},
		},
		{
			name: "invalid event details JSON",
			request: requests.WebhookReplayRequest{
				WorkspaceID:    workspaceID.String(),
				ProviderName:   "stripe",
				WebhookEventID: webhookEventID,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetWebhookEventForReplay(ctx, gomock.Any()).
					Return(db.GetWebhookEventForReplayRow{
						ID:                 eventID,
						ProcessingAttempts: pgtype.Int4{Int32: 1, Valid: true},
						EventDetails:       []byte(`invalid json`),
					}, nil)

				mockQuerier.EXPECT().
					ReplayWebhookEvent(ctx, gomock.Any()).
					Return(db.PaymentSyncEvent{
						ID: uuid.New(),
					}, nil)
			},
			wantErr: false,
			validate: func(resp *responses.WebhookReplayResponse) {
				assert.False(t, resp.Success)
				assert.Equal(t, "failed to parse event data", resp.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			result, err := service.ReplayWebhookEvent(ctx, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validate != nil {
					tt.validate(result)
				}
			}
		})
	}
}

func TestErrorRecoveryService_RecoverSyncSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewErrorRecoveryService(mockQuerier, logger.Log, nil)
	ctx := context.Background()

	workspaceID := uuid.New()
	sessionID := uuid.New()

	tests := []struct {
		name       string
		request    requests.SyncRecoveryRequest
		setupMocks func()
		wantErr    bool
		validate   func(*responses.SyncRecoveryResponse)
	}{
		{
			name: "invalid workspace ID",
			request: requests.SyncRecoveryRequest{
				WorkspaceID:  "invalid-uuid",
				SessionID:    sessionID.String(),
				RecoveryMode: "resume",
			},
			setupMocks: func() {},
			wantErr:    false,
			validate: func(resp *responses.SyncRecoveryResponse) {
				assert.False(t, resp.Success)
				assert.Equal(t, "invalid workspace ID format", resp.Error)
				assert.Equal(t, sessionID.String(), resp.SessionID)
			},
		},
		{
			name: "invalid session ID",
			request: requests.SyncRecoveryRequest{
				WorkspaceID:  workspaceID.String(),
				SessionID:    "invalid-uuid",
				RecoveryMode: "resume",
			},
			setupMocks: func() {},
			wantErr:    false,
			validate: func(resp *responses.SyncRecoveryResponse) {
				assert.False(t, resp.Success)
				assert.Equal(t, "invalid session ID format", resp.Error)
			},
		},
		{
			name: "session not found",
			request: requests.SyncRecoveryRequest{
				WorkspaceID:  workspaceID.String(),
				SessionID:    sessionID.String(),
				RecoveryMode: "resume",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetSyncSession(ctx, gomock.Any()).
					Return(db.PaymentSyncSession{}, pgx.ErrNoRows)
			},
			wantErr: false,
			validate: func(resp *responses.SyncRecoveryResponse) {
				assert.False(t, resp.Success)
				assert.Equal(t, "sync session not found", resp.Error)
			},
		},
		{
			name: "non-recoverable session status",
			request: requests.SyncRecoveryRequest{
				WorkspaceID:  workspaceID.String(),
				SessionID:    sessionID.String(),
				RecoveryMode: "resume",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetSyncSession(ctx, gomock.Any()).
					Return(db.PaymentSyncSession{
						ID:          sessionID,
						WorkspaceID: workspaceID,
						Status:      "completed",
					}, nil)
			},
			wantErr: false,
			validate: func(resp *responses.SyncRecoveryResponse) {
				assert.False(t, resp.Success)
				assert.Contains(t, resp.Error, "not recoverable")
			},
		},
		{
			name: "invalid recovery mode",
			request: requests.SyncRecoveryRequest{
				WorkspaceID:  workspaceID.String(),
				SessionID:    sessionID.String(),
				RecoveryMode: "invalid",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetSyncSession(ctx, gomock.Any()).
					Return(db.PaymentSyncSession{
						ID:          sessionID,
						WorkspaceID: workspaceID,
						Status:      "failed",
					}, nil)
			},
			wantErr: false,
			validate: func(resp *responses.SyncRecoveryResponse) {
				assert.False(t, resp.Success)
				assert.Equal(t, "invalid recovery mode, use 'resume' or 'restart'", resp.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			result, err := service.RecoverSyncSession(ctx, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validate != nil {
					tt.validate(result)
				}
			}
		})
	}
}

func TestErrorRecoveryService_GetDLQStats(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewErrorRecoveryService(mockQuerier, logger.Log, nil)
	ctx := context.Background()

	workspaceID := uuid.New()
	since := time.Now().AddDate(0, 0, -7)

	tests := []struct {
		name         string
		workspaceID  string
		providerName string
		since        time.Time
		setupMocks   func()
		wantErr      bool
		errString    string
		validate     func(*responses.DLQProcessingStats)
	}{
		{
			name:         "successful stats retrieval",
			workspaceID:  workspaceID.String(),
			providerName: "stripe",
			since:        since,
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetDLQProcessingStats(ctx, db.GetDLQProcessingStatsParams{
						WorkspaceID:  workspaceID,
						ProviderName: "stripe",
						OccurredAt:   pgtype.Timestamptz{Time: since, Valid: true},
					}).
					Return(db.GetDLQProcessingStatsRow{
						TotalDlqMessages:      100,
						SuccessfullyProcessed: 80,
						ProcessingFailed:      15,
						MaxRetriesExceeded:    5,
						LastProcessedAt:       time.Now(),
					}, nil)
			},
			wantErr: false,
			validate: func(stats *responses.DLQProcessingStats) {
				assert.Equal(t, int64(100), stats.TotalMessages)
				assert.Equal(t, int64(80), stats.SuccessfullyProcessed)
				assert.Equal(t, int64(15), stats.ProcessingFailed)
				assert.Equal(t, int64(5), stats.MaxRetriesExceeded)
				assert.Equal(t, float64(80), stats.SuccessRate)
			},
		},
		{
			name:         "invalid workspace ID",
			workspaceID:  "invalid-uuid",
			providerName: "stripe",
			since:        since,
			setupMocks:   func() {},
			wantErr:      true,
			errString:    "invalid workspace ID format",
		},
		{
			name:         "database error",
			workspaceID:  workspaceID.String(),
			providerName: "stripe",
			since:        since,
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetDLQProcessingStats(ctx, gomock.Any()).
					Return(db.GetDLQProcessingStatsRow{}, assert.AnError)
			},
			wantErr:   true,
			errString: "failed to get DLQ stats",
		},
		{
			name:         "zero total messages",
			workspaceID:  workspaceID.String(),
			providerName: "stripe",
			since:        since,
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetDLQProcessingStats(ctx, gomock.Any()).
					Return(db.GetDLQProcessingStatsRow{
						TotalDlqMessages:      0,
						SuccessfullyProcessed: 0,
						ProcessingFailed:      0,
						MaxRetriesExceeded:    0,
					}, nil)
			},
			wantErr: false,
			validate: func(stats *responses.DLQProcessingStats) {
				assert.Equal(t, int64(0), stats.TotalMessages)
				assert.Equal(t, float64(0), stats.SuccessRate)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			result, err := service.GetDLQStats(ctx, tt.workspaceID, tt.providerName, tt.since)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validate != nil {
					tt.validate(result)
				}
			}
		})
	}
}
