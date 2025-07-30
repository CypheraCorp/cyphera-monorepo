package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/cyphera/cyphera-api/libs/go/constants"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
)

// DelegationClientInterface is a local interface to avoid circular dependency
type DelegationClientInterface interface {
	ProcessPayment(ctx context.Context, params params.LocalProcessPaymentParams) (*responses.LocalProcessPaymentResponse, error)
}

type DunningRetryEngine struct {
	queries          db.Querier
	logger           *zap.Logger
	dunningService   *DunningService
	emailService     *EmailService
	delegationClient DelegationClientInterface
}

func NewDunningRetryEngine(queries db.Querier, logger *zap.Logger, dunningService *DunningService, emailService *EmailService, delegationClient DelegationClientInterface) *DunningRetryEngine {
	return &DunningRetryEngine{
		queries:          queries,
		logger:           logger,
		dunningService:   dunningService,
		emailService:     emailService,
		delegationClient: delegationClient,
	}
}

// ProcessDueCampaigns processes all campaigns that are due for retry
func (e *DunningRetryEngine) ProcessDueCampaigns(ctx context.Context, limit int32) error {
	campaigns, err := e.queries.ListDunningCampaignsForRetry(ctx, limit)
	if err != nil {
		return fmt.Errorf("failed to list campaigns for retry: %w", err)
	}

	e.logger.Info("processing dunning campaigns", zap.Int("count", len(campaigns)))

	for _, campaign := range campaigns {
		if err := e.processCampaign(ctx, &campaign); err != nil {
			e.logger.Error("failed to process campaign",
				zap.String("campaign_id", campaign.ID.String()),
				zap.Error(err))
			// Continue processing other campaigns
		}
	}

	return nil
}

// processCampaign handles a single dunning campaign
func (e *DunningRetryEngine) processCampaign(ctx context.Context, campaign *db.DunningCampaign) error {
	// Get the campaign with full details
	fullCampaign, err := e.queries.GetDunningCampaign(ctx, campaign.ID)
	if err != nil {
		return fmt.Errorf("failed to get campaign details: %w", err)
	}

	// Check if we've exceeded max attempts
	if campaign.CurrentAttempt >= fullCampaign.MaxRetryAttempts {
		return e.handleMaxAttemptsReached(ctx, campaign, &fullCampaign)
	}

	// Get the configuration
	config, err := e.queries.GetDunningConfiguration(ctx, campaign.ConfigurationID)
	if err != nil {
		return fmt.Errorf("failed to get configuration: %w", err)
	}

	// Process attempt actions
	attemptNumber := campaign.CurrentAttempt + 1
	if err := e.processAttemptActions(ctx, campaign, &config, attemptNumber); err != nil {
		return fmt.Errorf("failed to process attempt actions: %w", err)
	}

	// Calculate next retry time
	var nextRetryAt *time.Time
	if int(attemptNumber) < len(config.RetryIntervalDays) {
		days := config.RetryIntervalDays[attemptNumber]
		next := time.Now().Add(time.Duration(days) * 24 * time.Hour)
		nextRetryAt = &next
	}

	// Update campaign
	_, err = e.queries.UpdateDunningCampaign(ctx, db.UpdateDunningCampaignParams{
		ID:             campaign.ID,
		CurrentAttempt: attemptNumber,
		LastRetryAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		NextRetryAt:    pgtype.Timestamptz{Time: *nextRetryAt, Valid: nextRetryAt != nil},
	})
	if err != nil {
		return fmt.Errorf("failed to update campaign: %w", err)
	}

	return nil
}

// processAttemptActions executes all actions for a specific attempt
func (e *DunningRetryEngine) processAttemptActions(ctx context.Context, campaign *db.DunningCampaign, config *db.DunningConfiguration, attemptNumber int32) error {
	// Parse attempt actions from config
	var attemptActions []AttemptAction
	if err := json.Unmarshal(config.AttemptActions, &attemptActions); err != nil {
		return fmt.Errorf("failed to parse attempt actions: %w", err)
	}

	// Find actions for this attempt
	var actionsForAttempt *AttemptAction
	for _, action := range attemptActions {
		if action.Attempt == attemptNumber {
			actionsForAttempt = &action
			break
		}
	}

	if actionsForAttempt == nil {
		// No specific actions for this attempt, default to retry payment
		actionsForAttempt = &AttemptAction{
			Attempt: attemptNumber,
			Actions: []string{"retry_payment"},
		}
	}

	// Execute each action
	for _, actionType := range actionsForAttempt.Actions {
		switch actionType {
		case "retry_payment":
			if err := e.retryPayment(ctx, campaign, attemptNumber); err != nil {
				e.logger.Error("failed to retry payment",
					zap.String("campaign_id", campaign.ID.String()),
					zap.Error(err))
			}
		case "email":
			if err := e.sendEmail(ctx, campaign, attemptNumber, actionsForAttempt.EmailTemplateID); err != nil {
				e.logger.Error("failed to send email",
					zap.String("campaign_id", campaign.ID.String()),
					zap.Error(err))
			}
		case "in_app":
			if err := e.createInAppNotification(ctx, campaign, attemptNumber); err != nil {
				e.logger.Error("failed to create in-app notification",
					zap.String("campaign_id", campaign.ID.String()),
					zap.Error(err))
			}
		}
	}

	return nil
}

// retryPayment attempts to retry the failed payment
func (e *DunningRetryEngine) retryPayment(ctx context.Context, campaign *db.DunningCampaign, attemptNumber int32) error {
	// Create attempt record
	attempt, err := e.dunningService.CreateAttempt(ctx, params.DunningAttemptParams{
		CampaignID:    campaign.ID,
		AttemptNumber: attemptNumber,
		AttemptType:   "retry_payment",
	})
	if err != nil {
		return fmt.Errorf("failed to create attempt record: %w", err)
	}

	// Check if delegation client is available
	if e.delegationClient == nil {
		// If no delegation client, we can't actually retry the payment
		e.logger.Warn("delegation client not available, payment retry functionality not yet implemented",
			zap.String("campaign_id", campaign.ID.String()),
			zap.Int32("attempt_number", attemptNumber))

		// Mark attempt as failed with explanation
		errorMsg := "payment retry requires delegation server integration (not yet implemented)"
		_, err = e.dunningService.UpdateAttemptStatus(ctx, attempt.ID, constants.FailedStatus, &errorMsg)
		if err != nil {
			e.logger.Error("failed to update attempt status", zap.Error(err))
		}

		return nil
	}

	// Get subscription details for payment retry
	if campaign.SubscriptionID.Valid {
		// Get subscription
		subscription, err := e.queries.GetSubscription(ctx, campaign.SubscriptionID.Bytes)
		if err != nil {
			errorMsg := fmt.Sprintf("failed to get subscription: %v", err)
			_, _ = e.dunningService.UpdateAttemptStatus(ctx, attempt.ID, constants.FailedStatus, &errorMsg)
			return fmt.Errorf("failed to get subscription: %w", err)
		}

		// Get delegation data from delegation_data table using delegation_id
		delegationData, err := e.queries.GetDelegationData(ctx, subscription.DelegationID)
		if err != nil {
			errorMsg := fmt.Sprintf("failed to get delegation data: %v", err)
			_, _ = e.dunningService.UpdateAttemptStatus(ctx, attempt.ID, constants.FailedStatus, &errorMsg)
			return fmt.Errorf("failed to get delegation data: %w", err)
		}

		// Get product token info (contains network and token details)
		productToken, err := e.queries.GetProductToken(ctx, subscription.ProductTokenID)
		if err != nil {
			errorMsg := fmt.Sprintf("failed to get product token: %v", err)
			_, _ = e.dunningService.UpdateAttemptStatus(ctx, attempt.ID, constants.FailedStatus, &errorMsg)
			return fmt.Errorf("failed to get product token: %w", err)
		}

		// Get workspace wallet address
		workspaceWallets, err := e.queries.ListWalletsByWorkspaceID(ctx, campaign.WorkspaceID)
		if err != nil || len(workspaceWallets) == 0 {
			errorMsg := "no workspace wallet found"
			_, _ = e.dunningService.UpdateAttemptStatus(ctx, attempt.ID, constants.FailedStatus, &errorMsg)
			return fmt.Errorf("no workspace wallet found")
		}

		// Use the first wallet as merchant address
		merchantAddress := workspaceWallets[0].WalletAddress

		// Log payment retry attempt
		e.logger.Info("attempting payment retry",
			zap.String("campaign_id", campaign.ID.String()),
			zap.Int32("attempt_number", attemptNumber),
			zap.Int64("amount", campaign.OriginalAmountCents),
			zap.String("merchant_address", merchantAddress),
			zap.String("token_address", productToken.ContractAddress))

		// Attempt to process the payment
		paymentParams := params.LocalProcessPaymentParams{
			DelegationID:     delegationData.ID.String(),
			RecipientAddress: merchantAddress,
			Amount:           fmt.Sprintf("%d", campaign.OriginalAmountCents),
			TokenAddress:     productToken.ContractAddress,
			NetworkID:        productToken.NetworkID,
		}

		response, err := e.delegationClient.ProcessPayment(ctx, paymentParams)
		var txHash string
		if err == nil && response != nil {
			txHash = response.TransactionHash
		}
		if err != nil {
			// Payment failed
			errorMsg := fmt.Sprintf("payment retry failed: %v", err)
			_, updateErr := e.dunningService.UpdateAttemptStatus(ctx, attempt.ID, constants.FailedStatus, &errorMsg)
			if updateErr != nil {
				e.logger.Error("failed to update attempt status", zap.Error(updateErr))
			}

			e.logger.Error("payment retry failed",
				zap.String("campaign_id", campaign.ID.String()),
				zap.Int32("attempt_number", attemptNumber),
				zap.Error(err))

			return nil // Don't return error to allow other campaigns to be processed
		}

		// Payment successful!
		e.logger.Info("payment retry successful",
			zap.String("campaign_id", campaign.ID.String()),
			zap.String("tx_hash", txHash),
			zap.Int32("attempt_number", attemptNumber))

		// Update attempt as successful
		_, err = e.dunningService.UpdateAttemptStatus(ctx, attempt.ID, "success", nil)
		if err != nil {
			e.logger.Error("failed to update attempt status", zap.Error(err))
		}

		// Store the payment transaction hash in attempt metadata
		attemptMetadata, _ := json.Marshal(map[string]string{
			"transaction_hash": txHash,
		})
		_, err = e.queries.UpdateDunningAttempt(ctx, db.UpdateDunningAttemptParams{
			ID:       attempt.ID,
			Status:   "success",
			Metadata: attemptMetadata,
		})
		if err != nil {
			e.logger.Error("failed to update attempt metadata", zap.Error(err))
		}

		// Recover the campaign
		_, err = e.dunningService.RecoverCampaign(ctx, campaign.ID, campaign.OriginalAmountCents)
		if err != nil {
			e.logger.Error("failed to recover campaign", zap.Error(err))
			return fmt.Errorf("failed to recover campaign: %w", err)
		}
	} else if campaign.PaymentID.Valid {
		// For one-time payment retries
		errorMsg := "payment retry for one-time payments not yet implemented"
		_, _ = e.dunningService.UpdateAttemptStatus(ctx, attempt.ID, "failed", &errorMsg)
		e.logger.Warn("one-time payment retry not implemented",
			zap.String("campaign_id", campaign.ID.String()),
			zap.String("payment_id", uuid.UUID(campaign.PaymentID.Bytes).String()))
	}

	return nil
}

// sendEmail sends a dunning email to the customer
func (e *DunningRetryEngine) sendEmail(ctx context.Context, campaign *db.DunningCampaign, attemptNumber int32, templateID *uuid.UUID) error {
	// Get campaign details
	fullCampaign, err := e.queries.GetDunningCampaign(ctx, campaign.ID)
	if err != nil {
		return fmt.Errorf("failed to get campaign details: %w", err)
	}

	// Determine template type based on attempt number
	templateType := fmt.Sprintf("attempt_%d", attemptNumber)
	if attemptNumber == 1 {
		templateType = "attempt_1"
	} else if attemptNumber >= 3 {
		templateType = "final_notice"
	}

	// Get email template
	var template db.DunningEmailTemplate
	if templateID != nil {
		template, err = e.queries.GetDunningEmailTemplate(ctx, *templateID)
	} else {
		template, err = e.queries.GetDunningEmailTemplateByType(ctx, db.GetDunningEmailTemplateByTypeParams{
			WorkspaceID:  campaign.WorkspaceID,
			TemplateType: templateType,
		})
	}
	if err != nil {
		return fmt.Errorf("failed to get email template: %w", err)
	}

	// Create attempt record
	attempt, err := e.dunningService.CreateAttempt(ctx, params.DunningAttemptParams{
		CampaignID:      campaign.ID,
		AttemptNumber:   attemptNumber,
		AttemptType:     "send_email",
		EmailTemplateID: &template.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to create attempt record: %w", err)
	}

	// Prepare email data
	emailData := business.EmailData{
		CustomerName:      fullCampaign.CustomerName.String,
		CustomerEmail:     fullCampaign.CustomerEmail.String,
		Amount:            formatAmount(campaign.OriginalAmountCents, campaign.Currency),
		Currency:          campaign.Currency,
		ProductName:       "Subscription", // TODO: Get product name from subscription
		RetryDate:         calculateNextRetryDate(campaign, &fullCampaign).Format("January 2, 2006"),
		AttemptsRemaining: int(fullCampaign.MaxRetryAttempts - attemptNumber),
		PaymentLink:       fmt.Sprintf("https://app.cyphera.com/payment/retry/%s", campaign.ID.String()), // TODO: Get actual payment link
		SupportEmail:      "support@cyphera.com",                                                         // TODO: Get from workspace settings
		MerchantName:      "Cyphera",                                                                     // TODO: Get from workspace settings
		UnsubscribeLink:   fmt.Sprintf("https://app.cyphera.com/unsubscribe/%s", campaign.CustomerID.String()),
	}

	// Send email
	if e.emailService != nil {
		err = e.emailService.SendDunningEmail(ctx, &template, map[string]business.EmailData{"default": emailData}, fullCampaign.CustomerEmail.String)
		if err != nil {
			// Update attempt as failed
			_, updateErr := e.queries.UpdateDunningAttempt(ctx, db.UpdateDunningAttemptParams{
				ID:                 attempt.ID,
				Status:             constants.FailedStatus,
				CompletedAt:        pgtype.Timestamptz{Time: time.Now(), Valid: true},
				CommunicationSent:  pgtype.Bool{Bool: false, Valid: true},
				CommunicationError: pgtype.Text{String: err.Error(), Valid: true},
			})
			if updateErr != nil {
				e.logger.Error("failed to update attempt after email error", zap.Error(updateErr))
			}
			return fmt.Errorf("failed to send email: %w", err)
		}
	} else {
		// Fallback if email service not configured
		e.logger.Warn("email service not configured, skipping email send",
			zap.String("campaign_id", campaign.ID.String()),
			zap.String("customer_email", fullCampaign.CustomerEmail.String),
			zap.String("template_type", templateType))
	}

	// Update attempt as sent
	_, err = e.queries.UpdateDunningAttempt(ctx, db.UpdateDunningAttemptParams{
		ID:                attempt.ID,
		Status:            "success",
		CompletedAt:       pgtype.Timestamptz{Time: time.Now(), Valid: true},
		CommunicationSent: pgtype.Bool{Bool: true, Valid: true},
	})
	if err != nil {
		return err
	}

	return nil
}

// createInAppNotification creates an in-app notification for the customer
func (e *DunningRetryEngine) createInAppNotification(ctx context.Context, campaign *db.DunningCampaign, attemptNumber int32) error {
	// Create attempt record
	attempt, err := e.dunningService.CreateAttempt(ctx, params.DunningAttemptParams{
		CampaignID:    campaign.ID,
		AttemptNumber: attemptNumber,
		AttemptType:   "in_app_notification",
	})
	if err != nil {
		return fmt.Errorf("failed to create attempt record: %w", err)
	}

	// TODO: Create actual in-app notification
	e.logger.Info("creating in-app notification",
		zap.String("campaign_id", campaign.ID.String()),
		zap.String("customer_id", campaign.CustomerID.String()))

	// Update attempt as sent
	_, err = e.queries.UpdateDunningAttempt(ctx, db.UpdateDunningAttemptParams{
		ID:                attempt.ID,
		Status:            "success",
		CompletedAt:       pgtype.Timestamptz{Time: time.Now(), Valid: true},
		CommunicationSent: pgtype.Bool{Bool: true, Valid: true},
	})
	if err != nil {
		return err
	}

	return nil
}

// handleMaxAttemptsReached handles campaigns that have exhausted all retry attempts
func (e *DunningRetryEngine) handleMaxAttemptsReached(ctx context.Context, campaign *db.DunningCampaign, fullCampaign *db.GetDunningCampaignRow) error {
	// Get configuration to determine final action
	config, err := e.queries.GetDunningConfiguration(ctx, campaign.ConfigurationID)
	if err != nil {
		return fmt.Errorf("failed to get configuration: %w", err)
	}

	e.logger.Info("max attempts reached, executing final action",
		zap.String("campaign_id", campaign.ID.String()),
		zap.String("final_action", config.FinalAction))

	// Execute final action
	_, err = e.dunningService.FailCampaign(ctx, campaign.ID, config.FinalAction)
	if err != nil {
		return fmt.Errorf("failed to execute final action: %w", err)
	}

	return nil
}

// MonitorFailedPayments monitors for newly failed payments and creates dunning campaigns
func (e *DunningRetryEngine) MonitorFailedPayments(ctx context.Context) error {
	// TODO: Query for failed payments that don't have active dunning campaigns
	// This would typically be called by a background job or scheduler

	// For now, this is a placeholder
	e.logger.Info("monitoring for failed payments")

	return nil
}

// Helper types and functions

type AttemptAction struct {
	Attempt         int32      `json:"attempt"`
	Actions         []string   `json:"actions"`
	EmailTemplateID *uuid.UUID `json:"email_template_id,omitempty"`
}

// Commented out: unused function
/*
func stringPtr(s string) *string {
	return &s
}
*/

// formatAmount formats cents to a human-readable amount string
func formatAmount(cents int64, currency string) string {
	// Convert cents to dollars (or equivalent for other currencies)
	amount := float64(cents) / 100.0

	// Format based on currency
	switch currency {
	case constants.USDCurrency:
		return fmt.Sprintf("$%.2f", amount)
	case "EUR":
		return fmt.Sprintf("€%.2f", amount)
	case "GBP":
		return fmt.Sprintf("£%.2f", amount)
	default:
		return fmt.Sprintf("%.2f %s", amount, currency)
	}
}

// calculateNextRetryDate calculates when the next retry will occur
func calculateNextRetryDate(campaign *db.DunningCampaign, fullCampaign *db.GetDunningCampaignRow) time.Time {
	if campaign.NextRetryAt.Valid {
		return campaign.NextRetryAt.Time
	}

	// Default to 7 days from now if not set
	return time.Now().Add(7 * 24 * time.Hour)
}
