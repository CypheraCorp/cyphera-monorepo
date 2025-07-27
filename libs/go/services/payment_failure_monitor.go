package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/cyphera/cyphera-api/libs/go/db"
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
		return fmt.Errorf("failed to get failed payments: %w", err)
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
	existing, _ := m.queries.GetActiveDunningCampaignForPayment(ctx, paymentMonitorUuidToPgtype(&payment.ID))
	if existing.ID != uuid.Nil {
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
	campaign, err := m.dunningService.CreateCampaign(ctx, DunningCampaignParams{
		WorkspaceID:           payment.WorkspaceID,
		ConfigurationID:       config.ID,
		PaymentID:             &payment.ID,
		CustomerID:            payment.CustomerID,
		OriginalFailureReason: getPaymentFailureReason(payment),
		OriginalAmountCents:   payment.AmountInCents,
		Currency:              payment.Currency,
		Metadata: json.RawMessage(fmt.Sprintf(
			`{"payment_method":"%s","network_id":"%s","token_id":"%s"}`,
			payment.PaymentMethod,
			payment.NetworkID.Bytes,
			payment.TokenID.Bytes,
		)),
	})
	if err != nil {
		return fmt.Errorf("failed to create dunning campaign: %w", err)
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
		return fmt.Errorf("failed to get failed subscription events: %w", err)
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
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Check if campaign already exists
	existing, _ := m.queries.GetActiveDunningCampaignForSubscription(ctx, paymentMonitorUuidToPgtype(&subscription.ID))
	if existing.ID != uuid.Nil {
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
	campaign, err := m.dunningService.CreateCampaign(ctx, DunningCampaignParams{
		WorkspaceID:           subscription.WorkspaceID,
		ConfigurationID:       config.ID,
		SubscriptionID:        &subscription.ID,
		CustomerID:            subscription.CustomerID,
		OriginalFailureReason: getEventFailureReason(event),
		OriginalAmountCents:   int64(event.AmountInCents),
		Currency:              "USD", // TODO: Get currency from price
		Metadata: json.RawMessage(fmt.Sprintf(
			`{"product_id":"%s","price_id":"%s","event_type":"%s"}`,
			subscription.ProductID,
			subscription.PriceID,
			event.EventType,
		)),
	})
	if err != nil {
		return fmt.Errorf("failed to create dunning campaign: %w", err)
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

func paymentMonitorUuidToPgtype(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}