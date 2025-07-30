package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	// "github.com/jackc/pgx/v5/pgtype" // Commented out: unused after commenting out paymentMonitorUuidToPgtype
	"go.uber.org/zap"

	"github.com/cyphera/cyphera-api/libs/go/constants"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
)

const (
	// Error messages
	errMsgFailedToGetPayments     = "failed to get failed payments"
	errMsgFailedToCreateCampaign  = "failed to create dunning campaign"
	errMsgFailedToGetSubscription = "failed to get subscription"
	errMsgFailedToGetSubEvents    = "failed to get failed subscription events"
)

// PaymentFailureMonitor monitors for failed payments and creates dunning campaigns
type PaymentFailureMonitor struct {
	queries        *db.Queries
	logger         *zap.Logger
	dunningService *DunningService
}

func NewPaymentFailureMonitor(queries *db.Queries, logger *zap.Logger, dunningService *DunningService) *PaymentFailureMonitor {
	return &PaymentFailureMonitor{
		queries:        queries,
		logger:         logger,
		dunningService: dunningService,
	}
}

// MonitorFailedPayments checks for failed payments that need dunning campaigns
func (m *PaymentFailureMonitor) MonitorFailedPayments(ctx context.Context) error {
	// Get recent failed payments that don't have dunning campaigns
	failedPayments, err := m.getFailedPaymentsNeedingDunning(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsgFailedToGetPayments, err)
	}

	m.logger.Info("monitoring failed payments", zap.Int("count", len(failedPayments)))

	for _, payment := range failedPayments {
		if err := m.createDunningCampaignForPayment(ctx, &payment); err != nil {
			m.logger.Error("failed to create dunning campaign",
				zap.String("payment_id", payment.ID.String()),
				zap.Error(err))
		}
	}

	return nil
}

// getFailedPaymentsNeedingDunning gets failed payments that don't have active dunning campaigns
func (m *PaymentFailureMonitor) getFailedPaymentsNeedingDunning(ctx context.Context) ([]db.Payment, error) {
	// TODO: Create a specific query for this
	// For now, we'll use a simplified approach

	// Get all payments with status 'failed' from the last 24 hours
	// that don't already have a dunning campaign
	var failedPayments []db.Payment

	// This is a placeholder - you'll need to create the actual query
	m.logger.Info("checking for failed payments needing dunning")

	return failedPayments, nil
}

// createDunningCampaignForPayment creates a dunning campaign for a failed payment
func (m *PaymentFailureMonitor) createDunningCampaignForPayment(ctx context.Context, payment *db.Payment) error {
	// Check if campaign already exists
	existing, err := m.queries.GetActiveDunningCampaignForPayment(ctx, uuidToPgtype(&payment.ID))
	if err == nil && existing.ID != uuid.Nil {
		m.logger.Debug("dunning campaign already exists for payment",
			zap.String("payment_id", payment.ID.String()))
		return nil
	}

	// Get the default dunning configuration for the workspace
	config, err := m.queries.GetDefaultDunningConfiguration(ctx, payment.WorkspaceID)
	if err != nil {
		// If no default config, skip dunning for this payment
		m.logger.Warn("no default dunning configuration for workspace",
			zap.String("workspace_id", payment.WorkspaceID.String()))
		return nil
	}

	// Create the campaign
	// Note: DunningCampaignParams only supports subscription-based campaigns in the current implementation
	// For payment-based campaigns, we would need to enhance the params structure
	campaign, err := m.dunningService.CreateCampaign(ctx, params.DunningCampaignParams{
		SubscriptionID:    uuid.Nil, // Payment-based, not subscription-based
		ConfigurationID:   config.ID,
		InitialPaymentID:  &payment.ID,
		TriggerReason:     getPaymentFailureReason(payment),
		OutstandingAmount: payment.AmountInCents,
		Currency:          payment.Currency,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", errMsgFailedToCreateCampaign, err)
	}

	m.logger.Info("created dunning campaign for failed payment",
		zap.String("payment_id", payment.ID.String()),
		zap.String("campaign_id", campaign.ID.String()))

	return nil
}

// MonitorFailedSubscriptions checks for failed subscription events that need dunning
func (m *PaymentFailureMonitor) MonitorFailedSubscriptions(ctx context.Context) error {
	// Get recent failed subscription events
	failedEvents, err := m.getFailedSubscriptionEvents(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsgFailedToGetSubEvents, err)
	}

	m.logger.Info("monitoring failed subscriptions", zap.Int("count", len(failedEvents)))

	for _, event := range failedEvents {
		if err := m.createDunningCampaignForSubscription(ctx, &event); err != nil {
			m.logger.Error("failed to create dunning campaign for subscription",
				zap.String("subscription_id", event.SubscriptionID.String()),
				zap.Error(err))
		}
	}

	return nil
}

// getFailedSubscriptionEvents gets failed subscription events needing dunning
func (m *PaymentFailureMonitor) getFailedSubscriptionEvents(ctx context.Context) ([]db.SubscriptionEvent, error) {
	// TODO: Create a query to get failed subscription events
	// that don't have active dunning campaigns
	var failedEvents []db.SubscriptionEvent

	m.logger.Info("checking for failed subscription events needing dunning")

	return failedEvents, nil
}

// createDunningCampaignForSubscription creates a dunning campaign for a failed subscription
func (m *PaymentFailureMonitor) createDunningCampaignForSubscription(ctx context.Context, event *db.SubscriptionEvent) error {
	// Get subscription details
	subscription, err := m.queries.GetSubscription(ctx, event.SubscriptionID)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsgFailedToGetSubscription, err)
	}

	// Check if campaign already exists
	existing, err := m.queries.GetActiveDunningCampaignForSubscription(ctx, uuidToPgtype(&subscription.ID))
	if err == nil && existing.ID != uuid.Nil {
		m.logger.Debug("dunning campaign already exists for subscription",
			zap.String("subscription_id", subscription.ID.String()))
		return nil
	}

	// Get the default dunning configuration for the workspace
	config, err := m.queries.GetDefaultDunningConfiguration(ctx, subscription.WorkspaceID)
	if err != nil {
		m.logger.Warn("no default dunning configuration for workspace",
			zap.String("workspace_id", subscription.WorkspaceID.String()))
		return nil
	}

	// Create the campaign
	campaign, err := m.dunningService.CreateCampaign(ctx, params.DunningCampaignParams{
		SubscriptionID:    subscription.ID,
		ConfigurationID:   config.ID,
		TriggerReason:     getEventFailureReason(event),
		OutstandingAmount: int64(event.AmountInCents),
		Currency:          constants.USDCurrency, // TODO: Get currency from price
		InitialPaymentID:  nil,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", errMsgFailedToCreateCampaign, err)
	}

	m.logger.Info("created dunning campaign for failed subscription",
		zap.String("subscription_id", subscription.ID.String()),
		zap.String("campaign_id", campaign.ID.String()))

	return nil
}

// Helper functions

func getPaymentFailureReason(payment *db.Payment) string {
	if payment.ErrorMessage.Valid {
		return payment.ErrorMessage.String
	}
	return "Payment failed"
}

func getEventFailureReason(event *db.SubscriptionEvent) string {
	if event.ErrorMessage.Valid {
		return event.ErrorMessage.String
	}
	return fmt.Sprintf("Subscription event failed: %s", event.EventType)
}

// Commented out: unused function
/*
func paymentMonitorUuidToPgtype(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}
*/
