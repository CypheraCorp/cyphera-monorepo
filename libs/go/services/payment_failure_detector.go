package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// PaymentFailureDetector detects failed payments and creates dunning campaigns
type PaymentFailureDetector struct {
	queries        *db.Queries
	logger         *zap.Logger
	dunningService *DunningService
}

// uuidFromPgtype converts pgtype.UUID to uuid.UUID
func uuidFromPgtype(pgUUID pgtype.UUID) (uuid.UUID, error) {
	if !pgUUID.Valid {
		return uuid.Nil, fmt.Errorf("invalid UUID")
	}
	return uuid.UUID(pgUUID.Bytes), nil
}

// NewPaymentFailureDetector creates a new payment failure detector
func NewPaymentFailureDetector(queries *db.Queries, logger *zap.Logger, dunningService *DunningService) *PaymentFailureDetector {
	return &PaymentFailureDetector{
		queries:        queries,
		logger:         logger,
		dunningService: dunningService,
	}
}

// DetectionResult holds the result of failure detection
type DetectionResult struct {
	FailedEvents      []db.SubscriptionEvent
	CampaignsCreated  int
	CampaignsSkipped  int
	Errors            []error
}

// DetectAndCreateCampaigns detects failed payments and creates dunning campaigns
func (d *PaymentFailureDetector) DetectAndCreateCampaigns(ctx context.Context, lookbackMinutes int) (*DetectionResult, error) {
	result := &DetectionResult{
		FailedEvents:     []db.SubscriptionEvent{},
		CampaignsCreated: 0,
		CampaignsSkipped: 0,
		Errors:           []error{},
	}

	// Get failed subscription events from the last N minutes
	since := time.Now().Add(-time.Duration(lookbackMinutes) * time.Minute)
	sincePgTime := pgtype.Timestamptz{
		Time:  since,
		Valid: true,
	}
	failedEvents, err := d.queries.ListRecentSubscriptionEventsByType(ctx, db.ListRecentSubscriptionEventsByTypeParams{
		EventType:   db.SubscriptionEventTypeFailed,
		OccurredAt:  sincePgTime,
	})
	if err != nil {
		return result, fmt.Errorf("failed to get recent failed events: %w", err)
	}

	result.FailedEvents = failedEvents
	d.logger.Info("Found failed subscription events",
		zap.Int("count", len(failedEvents)),
		zap.Time("since", since),
	)

	// Process each failed event
	for _, event := range failedEvents {
		err := d.processFailedEvent(ctx, event, result)
		if err != nil {
			result.Errors = append(result.Errors, err)
			d.logger.Error("Failed to process failed event",
				zap.String("event_id", event.ID.String()),
				zap.Error(err),
			)
		}
	}

	d.logger.Info("Payment failure detection completed",
		zap.Int("failed_events", len(result.FailedEvents)),
		zap.Int("campaigns_created", result.CampaignsCreated),
		zap.Int("campaigns_skipped", result.CampaignsSkipped),
		zap.Int("errors", len(result.Errors)),
	)

	return result, nil
}

// processFailedEvent processes a single failed payment event
func (d *PaymentFailureDetector) processFailedEvent(ctx context.Context, event db.SubscriptionEvent, result *DetectionResult) error {
	// Get subscription details
	subscription, err := d.queries.GetSubscription(ctx, event.SubscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Check if there's already an active campaign for this subscription
	existingCampaigns, err := d.queries.ListDunningCampaigns(ctx, db.ListDunningCampaignsParams{
		WorkspaceID: subscription.WorkspaceID,
		Limit:       100,
		Offset:      0,
	})
	if err != nil {
		return fmt.Errorf("failed to check existing campaigns: %w", err)
	}

	// Check if there's already an active campaign for this subscription
	for _, campaign := range existingCampaigns {
		// Convert pgtype.UUID to uuid.UUID for comparison
		campaignSubID, err := uuidFromPgtype(campaign.SubscriptionID)
		if err != nil {
			continue
		}
		
		if campaignSubID == subscription.ID && 
		   (campaign.Status == "active" || campaign.Status == "paused") {
			d.logger.Info("Skipping campaign creation - active campaign exists",
				zap.String("subscription_id", subscription.ID.String()),
				zap.String("existing_campaign_id", campaign.ID.String()),
			)
			result.CampaignsSkipped++
			return nil
		}
	}

	// Get or create default dunning configuration for the workspace
	config, err := d.getOrCreateDefaultConfiguration(ctx, subscription.WorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to get dunning configuration: %w", err)
	}

	// Get customer details
	customer, err := d.queries.GetCustomer(ctx, subscription.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to get customer: %w", err)
	}

	// Get price details for the amount
	price, err := d.queries.GetPrice(ctx, subscription.PriceID)
	if err != nil {
		return fmt.Errorf("failed to get price: %w", err)
	}

	// Create campaign parameters
	campaignParams := params.DunningCampaignParams{
		SubscriptionID:    subscription.ID,
		ConfigurationID:   config.ID,
		TriggerReason:     "Payment failed - subscription event",
		OutstandingAmount: int64(price.UnitAmountInPennies) * 100, // Convert pennies to cents
		Currency:          price.Currency,
		InitialPaymentID:  nil,
	}

	// Determine campaign strategy based on customer history
	strategy := d.determineCampaignStrategy(ctx, &customer, &subscription, int64(price.UnitAmountInPennies)*100)

	// Create the campaign
	campaign, err := d.dunningService.CreateCampaign(ctx, campaignParams)
	if err != nil {
		return fmt.Errorf("failed to create dunning campaign: %w", err)
	}

	// Extract customer email safely
	customerEmail := ""
	if customer.Email.Valid {
		customerEmail = customer.Email.String
	}
	
	d.logger.Info("Created dunning campaign for failed payment",
		zap.String("campaign_id", campaign.ID.String()),
		zap.String("subscription_id", subscription.ID.String()),
		zap.String("customer_email", customerEmail),
		zap.String("strategy", strategy),
		zap.Int64("amount_cents", int64(price.UnitAmountInPennies)*100),
	)

	result.CampaignsCreated++
	return nil
}

// getOrCreateDefaultConfiguration gets or creates a default dunning configuration
func (d *PaymentFailureDetector) getOrCreateDefaultConfiguration(ctx context.Context, workspaceID uuid.UUID) (*db.DunningConfiguration, error) {
	// Try to get existing configurations
	configs, err := d.queries.ListDunningConfigurations(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list configurations: %w", err)
	}

	// Return the first active configuration if exists
	for _, config := range configs {
		if config.IsActive.Valid && config.IsActive.Bool {
			return &config, nil
		}
	}

	// Create default configuration if none exists
	desc := "Automatically created configuration for failed payment recovery"
	config, err := d.dunningService.CreateConfiguration(ctx, params.DunningConfigParams{
		WorkspaceID:        workspaceID,
		Name:              "Default Auto-Created Configuration",
		Description:       &desc,
		MaxRetryAttempts:  4,
		RetryIntervalDays: []int32{3, 7, 7, 7},
		GracePeriodHours:  24,
		IsActive:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create default configuration: %w", err)
	}

	d.logger.Info("Created default dunning configuration",
		zap.String("workspace_id", workspaceID.String()),
		zap.String("config_id", config.ID.String()),
	)

	return config, nil
}

// determineCampaignStrategy determines the campaign strategy based on customer history
func (d *PaymentFailureDetector) determineCampaignStrategy(ctx context.Context, customer *db.Customer, subscription *db.Subscription, amountCents int64) string {
	// Get customer payment history
	events, err := d.queries.ListSubscriptionEventsBySubscription(ctx, subscription.ID)
	if err != nil {
		d.logger.Error("Failed to get customer payment history", zap.Error(err))
		return "standard"
	}

	// Count successful and failed payments
	var successfulPayments, failedPayments int
	for _, event := range events {
		switch event.EventType {
		case db.SubscriptionEventTypeRedeemed:
			successfulPayments++
		case db.SubscriptionEventTypeFailed, db.SubscriptionEventTypeFailedRedemption:
			failedPayments++
		}
	}

	// Calculate success rate
	totalPayments := successfulPayments + failedPayments
	if totalPayments == 0 {
		return "new_customer"
	}

	successRate := float64(successfulPayments) / float64(totalPayments)

	// Determine strategy based on history and amount
	switch {
	case totalPayments < 3:
		return "new_customer"
	case successRate >= 0.9 && amountCents >= 10000: // Good history, high value
		return "premium"
	case successRate >= 0.8: // Good history
		return "standard"
	case successRate >= 0.5: // Mixed history
		return "cautious"
	default: // Poor history
		return "high_risk"
	}
}

// ProcessFailedPaymentWebhook processes a payment failure webhook event
func (d *PaymentFailureDetector) ProcessFailedPaymentWebhook(ctx context.Context, workspaceID uuid.UUID, subscriptionID uuid.UUID, failureData map[string]interface{}) error {
	// Create a failed subscription event
	eventData := map[string]interface{}{
		"webhook_source": "payment_processor",
		"failure_data":   failureData,
		"processed_at":   time.Now().Format(time.RFC3339),
	}

	// Get subscription to determine amount
	subscription, err := d.queries.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	price, err := d.queries.GetPrice(ctx, subscription.PriceID)
	if err != nil {
		return fmt.Errorf("failed to get price: %w", err)
	}

	// Create the failed event
	eventDataJSON, _ := json.Marshal(eventData)
	event, err := d.queries.CreateFailedRedemptionEvent(ctx, db.CreateFailedRedemptionEventParams{
		SubscriptionID:  subscriptionID,
		AmountInCents:   int32(price.UnitAmountInPennies),
		Metadata:        eventDataJSON,
	})
	if err != nil {
		return fmt.Errorf("failed to create failed event: %w", err)
	}

	// Process the failed event to create campaign
	result := &DetectionResult{
		FailedEvents:     []db.SubscriptionEvent{},
		CampaignsCreated: 0,
		CampaignsSkipped: 0,
		Errors:           []error{},
	}

	err = d.processFailedEvent(ctx, event, result)
	if err != nil {
		return fmt.Errorf("failed to process webhook failure: %w", err)
	}

	return nil
}