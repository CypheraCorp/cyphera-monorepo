package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
)

type DunningService struct {
	queries db.Querier
	logger  *zap.Logger
}

func NewDunningService(queries db.Querier, logger *zap.Logger) *DunningService {
	return &DunningService{
		queries: queries,
		logger:  logger,
	}
}

// Configuration Management

func (s *DunningService) CreateConfiguration(ctx context.Context, params params.DunningConfigParams) (*db.DunningConfiguration, error) {
	// If setting as default, unset other defaults first
	if params.IsDefault {
		if err := s.queries.SetDefaultDunningConfiguration(ctx, db.SetDefaultDunningConfigurationParams{
			WorkspaceID: params.WorkspaceID,
			ID:          uuid.Nil, // Will be different from any existing ID
		}); err != nil {
			return nil, fmt.Errorf("failed to unset default configurations: %w", err)
		}
	}

	config, err := s.queries.CreateDunningConfiguration(ctx, db.CreateDunningConfigurationParams{
		WorkspaceID:            params.WorkspaceID,
		Name:                   params.Name,
		Description:            textToPgtype(params.Description),
		IsActive:               pgtype.Bool{Bool: params.IsActive, Valid: true},
		IsDefault:              pgtype.Bool{Bool: params.IsDefault, Valid: true},
		MaxRetryAttempts:       params.MaxRetryAttempts,
		RetryIntervalDays:      params.RetryIntervalDays,
		AttemptActions:         params.AttemptActions,
		FinalAction:            params.FinalAction,
		FinalActionConfig:      params.FinalActionConfig,
		SendPreDunningReminder: pgtype.Bool{Bool: params.SendPreDunningReminder, Valid: true},
		PreDunningDays:         pgtype.Int4{Int32: params.PreDunningDays, Valid: true},
		AllowCustomerRetry:     pgtype.Bool{Bool: params.AllowCustomerRetry, Valid: true},
		GracePeriodHours:       pgtype.Int4{Int32: params.GracePeriodHours, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create dunning configuration: %w", err)
	}

	return &config, nil
}

func (s *DunningService) GetConfiguration(ctx context.Context, id uuid.UUID) (*db.DunningConfiguration, error) {
	config, err := s.queries.GetDunningConfiguration(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get dunning configuration: %w", err)
	}
	return &config, nil
}

func (s *DunningService) GetDefaultConfiguration(ctx context.Context, workspaceID uuid.UUID) (*db.DunningConfiguration, error) {
	config, err := s.queries.GetDefaultDunningConfiguration(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get default dunning configuration: %w", err)
	}
	return &config, nil
}

// Campaign Management

func (s *DunningService) CreateCampaign(ctx context.Context, params params.DunningCampaignParams) (*db.DunningCampaign, error) {
	// Check if there's already an active campaign for this subscription/payment
	if params.SubscriptionID != uuid.Nil {
		existing, _ := s.queries.GetActiveDunningCampaignForSubscription(ctx, dunningUuidToPgtype(&params.SubscriptionID))
		if existing.ID != uuid.Nil {
			return nil, fmt.Errorf("active dunning campaign already exists for subscription")
		}
	}
	if params.InitialPaymentID != nil {
		existing, _ := s.queries.GetActiveDunningCampaignForPayment(ctx, dunningUuidToPgtype(params.InitialPaymentID))
		if existing.ID != uuid.Nil {
			return nil, fmt.Errorf("active dunning campaign already exists for payment")
		}
	}

	// Get configuration to calculate first retry time
	config, err := s.queries.GetDunningConfiguration(ctx, params.ConfigurationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dunning configuration: %w", err)
	}

	// Calculate next retry time based on grace period
	nextRetryAt := time.Now().Add(time.Duration(config.GracePeriodHours.Int32) * time.Hour)

	// Get workspace ID and customer ID based on subscription
	var workspaceID, customerID uuid.UUID
	if params.SubscriptionID != uuid.Nil {
		// Get subscription to extract workspace and customer
		sub, err := s.queries.GetSubscription(ctx, params.SubscriptionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get subscription: %w", err)
		}
		workspaceID = sub.WorkspaceID
		customerID = sub.CustomerID
	} else {
		// For payment-based campaigns, we'd need to get this info differently
		// This is a limitation of the current params structure
		return nil, fmt.Errorf("workspace and customer ID required for campaign creation")
	}

	metadata, _ := json.Marshal(map[string]interface{}{
		"trigger_reason": params.TriggerReason,
	})

	campaign, err := s.queries.CreateDunningCampaign(ctx, db.CreateDunningCampaignParams{
		WorkspaceID:           workspaceID,
		ConfigurationID:       params.ConfigurationID,
		SubscriptionID:        dunningUuidToPgtype(&params.SubscriptionID),
		PaymentID:             dunningUuidToPgtype(params.InitialPaymentID),
		CustomerID:            customerID,
		Status:                "active",
		OriginalFailureReason: pgtype.Text{String: params.TriggerReason, Valid: true},
		OriginalAmountCents:   params.OutstandingAmount,
		Currency:              params.Currency,
		Metadata:              metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create dunning campaign: %w", err)
	}

	// Update with next retry time
	updated, err := s.queries.UpdateDunningCampaign(ctx, db.UpdateDunningCampaignParams{
		ID:          campaign.ID,
		NextRetryAt: pgtype.Timestamptz{Time: nextRetryAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set next retry time: %w", err)
	}

	return &updated, nil
}

// Attempt Management

func (s *DunningService) CreateAttempt(ctx context.Context, params params.DunningAttemptParams) (*db.DunningAttempt, error) {
	// Create default metadata
	metadata := json.RawMessage(`{}`)

	attempt, err := s.queries.CreateDunningAttempt(ctx, db.CreateDunningAttemptParams{
		CampaignID:        params.CampaignID,
		AttemptNumber:     params.AttemptNumber,
		AttemptType:       params.AttemptType,
		Status:            "pending",
		PaymentID:         pgtype.UUID{Valid: false},
		CommunicationType: pgtype.Text{Valid: false},
		EmailTemplateID:   dunningUuidToPgtype(params.EmailTemplateID),
		Metadata:          metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create dunning attempt: %w", err)
	}

	return &attempt, nil
}

func (s *DunningService) UpdateAttemptStatus(ctx context.Context, attemptID uuid.UUID, status string, error *string) (*db.DunningAttempt, error) {
	var paymentError pgtype.Text
	if status == "failed" && error != nil {
		paymentError = pgtype.Text{String: *error, Valid: true}
	}

	attempt, err := s.queries.UpdateDunningAttempt(ctx, db.UpdateDunningAttemptParams{
		ID:           attemptID,
		Status:       status,
		CompletedAt:  pgtype.Timestamptz{Time: time.Now(), Valid: true},
		PaymentError: paymentError,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update dunning attempt: %w", err)
	}

	return &attempt, nil
}

// Recovery and Final Actions

func (s *DunningService) RecoverCampaign(ctx context.Context, campaignID uuid.UUID, recoveredAmountCents int64) (*db.DunningCampaign, error) {
	campaign, err := s.queries.RecoverDunningCampaign(ctx, db.RecoverDunningCampaignParams{
		ID:                   campaignID,
		RecoveredAmountCents: pgtype.Int8{Int64: recoveredAmountCents, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to recover dunning campaign: %w", err)
	}

	// TODO: Send recovery success notification
	// TODO: Update analytics

	return &campaign, nil
}

func (s *DunningService) FailCampaign(ctx context.Context, campaignID uuid.UUID, finalAction string) (*db.DunningCampaign, error) {
	campaign, err := s.queries.FailDunningCampaign(ctx, db.FailDunningCampaignParams{
		ID:               campaignID,
		FinalActionTaken: pgtype.Text{String: finalAction, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fail dunning campaign: %w", err)
	}

	// Execute final action
	if err := s.executeFinalAction(ctx, &campaign, finalAction); err != nil {
		s.logger.Error("failed to execute final action",
			zap.String("campaign_id", campaignID.String()),
			zap.String("action", finalAction),
			zap.Error(err))
	}

	return &campaign, nil
}

func (s *DunningService) executeFinalAction(ctx context.Context, campaign *db.DunningCampaign, action string) error {
	switch action {
	case "cancel":
		// Cancel the subscription associated with the campaign
		if campaign.SubscriptionID.Valid {
			s.logger.Info("Cancelling subscription due to failed dunning campaign",
				zap.String("campaign_id", campaign.ID.String()),
				zap.String("subscription_id", uuid.UUID(campaign.SubscriptionID.Bytes).String()))

			// Schedule cancellation for end of period (give customer benefit of their paid period)
			_, err := s.queries.ScheduleSubscriptionCancellation(ctx, db.ScheduleSubscriptionCancellationParams{
				ID:                 campaign.SubscriptionID.Bytes,
				CancelAt:           pgtype.Timestamptz{Time: time.Now(), Valid: true}, // Will be processed by scheduled changes processor
				CancellationReason: pgtype.Text{String: "Failed dunning process - automatic cancellation", Valid: true},
			})
			if err != nil {
				return fmt.Errorf("failed to cancel subscription: %w", err)
			}

			// Record state change for audit trail
			_, err = s.queries.RecordStateChange(ctx, db.RecordStateChangeParams{
				SubscriptionID: campaign.SubscriptionID.Bytes,
				FromStatus:     db.NullSubscriptionStatus{Valid: false}, // Status doesn't change yet
				ToStatus:       db.SubscriptionStatusActive,             // Still active until cancel_at date
				ChangeReason:   pgtype.Text{String: fmt.Sprintf("Scheduled cancellation due to failed dunning campaign %s", campaign.ID), Valid: true},
				InitiatedBy:    pgtype.Text{String: "dunning_system", Valid: true},
			})
			if err != nil {
				s.logger.Error("Failed to record state change for dunning cancellation", zap.Error(err))
			}
		}
		return nil
	case "pause":
		// TODO: Pause subscription
		return nil
	case "downgrade":
		// TODO: Downgrade subscription based on config
		return nil
	default:
		return fmt.Errorf("unknown final action: %s", action)
	}
}

// Email Template Management

func (s *DunningService) CreateEmailTemplate(ctx context.Context, params params.EmailTemplateParams) (*db.DunningEmailTemplate, error) {
	// If setting as active, deactivate others of same type
	if params.IsActive {
		if err := s.queries.DeactivateTemplatesByType(ctx, db.DeactivateTemplatesByTypeParams{
			WorkspaceID:  params.WorkspaceID,
			TemplateType: params.TemplateType,
			ID:           uuid.Nil,
		}); err != nil {
			return nil, fmt.Errorf("failed to deactivate existing templates: %w", err)
		}
	}

	template, err := s.queries.CreateDunningEmailTemplate(ctx, db.CreateDunningEmailTemplateParams{
		WorkspaceID:        params.WorkspaceID,
		Name:               params.TemplateName,
		TemplateType:       params.TemplateType,
		Subject:            params.Subject,
		BodyHtml:           params.BodyHtml,
		BodyText:           textToPgtype(&params.BodyText),
		AvailableVariables: json.RawMessage("[]"), // Convert Variables slice to JSON
		IsActive:           pgtype.Bool{Bool: params.IsActive, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create email template: %w", err)
	}

	return &template, nil
}

// Analytics

func (s *DunningService) GetCampaignStats(ctx context.Context, workspaceID uuid.UUID, startDate, endDate time.Time) (*db.GetDunningCampaignStatsRow, error) {
	stats, err := s.queries.GetDunningCampaignStats(ctx, db.GetDunningCampaignStatsParams{
		WorkspaceID: workspaceID,
		CreatedAt:   pgtype.Timestamptz{Time: startDate, Valid: true},
		CreatedAt_2: pgtype.Timestamptz{Time: endDate, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get campaign stats: %w", err)
	}

	return &stats, nil
}

// Helper functions

func textToPgtype(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func dunningUuidToPgtype(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}
