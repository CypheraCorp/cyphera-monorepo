package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// SubscriptionManagementService handles all subscription lifecycle operations
type SubscriptionManagementService struct {
	db             db.Querier
	calculator     IProrationCalculator
	paymentService IPaymentService
	emailService   IEmailService
	logger         *zap.Logger
}

// NewSubscriptionManagementService creates a new subscription management service
func NewSubscriptionManagementService(
	db db.Querier,
	paymentService IPaymentService,
	emailService IEmailService,
) *SubscriptionManagementService {
	var calculator IProrationCalculator = NewProrationCalculator()
	return &SubscriptionManagementService{
		db:             db,
		calculator:     calculator,
		paymentService: paymentService,
		emailService:   emailService,
		logger:         zap.L(),
	}
}

// NewSubscriptionManagementServiceWithDependencies creates a service with custom dependencies
func NewSubscriptionManagementServiceWithDependencies(
	db db.Querier,
	calculator IProrationCalculator,
	paymentService IPaymentService,
	emailService IEmailService,
	logger *zap.Logger,
) *SubscriptionManagementService {
	if logger == nil {
		logger = zap.L()
	}
	return &SubscriptionManagementService{
		db:             db,
		calculator:     calculator,
		paymentService: paymentService,
		emailService:   emailService,
		logger:         logger,
	}
}

// UpgradeSubscription handles immediate subscription upgrades with proration
func (sms *SubscriptionManagementService) UpgradeSubscription(
	ctx context.Context,
	subscriptionID uuid.UUID,
	newLineItems []requests.LineItemUpdate,
	reason string,
) error {
	// Get current subscription details
	sub, err := sms.db.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Validate subscription can be upgraded
	if sub.Status != db.SubscriptionStatusActive {
		return fmt.Errorf("can only upgrade active subscriptions, current status: %s", sub.Status)
	}

	// Calculate new total (simplified - in real implementation would calculate from line items)
	newTotal := sms.calculateNewTotal(ctx, newLineItems)
	oldTotal := int64(sub.TotalAmountInCents)

	// Calculate proration
	proration := sms.calculator.CalculateUpgradeProration(
		sub.CurrentPeriodStart.Time,
		sub.CurrentPeriodEnd.Time,
		oldTotal,
		newTotal,
		time.Now(),
	)

	// Create schedule change record
	fromLineItems, _ := json.Marshal(map[string]interface{}{
		"product_id": sub.ProductID,
		"price_id":   sub.PriceID,
		"amount":     oldTotal,
	})
	toLineItems, _ := json.Marshal(newLineItems)
	prorationCalc, _ := json.Marshal(proration.Calculation)

	scheduleChange, err := sms.db.CreateScheduleChange(ctx, db.CreateScheduleChangeParams{
		SubscriptionID:       subscriptionID,
		ChangeType:           "upgrade",
		ScheduledFor:         pgtype.Timestamptz{Time: time.Now(), Valid: true},
		FromLineItems:        fromLineItems,
		ToLineItems:          toLineItems,
		ProrationAmountCents: pgtype.Int8{Int64: proration.NetAmount, Valid: true},
		ProrationCalculation: prorationCalc,
		Status:               "processing",
		Reason:               pgtype.Text{String: reason, Valid: true},
		InitiatedBy:          pgtype.Text{String: "customer", Valid: true},
		Metadata:             []byte("{}"),
	})
	if err != nil {
		return fmt.Errorf("failed to create schedule change: %w", err)
	}

	// Update subscription immediately
	// In real implementation, would update line items properly
	_, err = sms.db.UpdateSubscriptionForUpgrade(ctx, db.UpdateSubscriptionForUpgradeParams{
		ID:                 subscriptionID,
		PriceID:            sub.PriceID, // Would be updated in real implementation
		TotalAmountInCents: int32(newTotal),
	})
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	// Create proration record
	_, err = sms.db.CreateProrationRecord(ctx, db.CreateProrationRecordParams{
		SubscriptionID:      subscriptionID,
		ScheduleChangeID:    pgtype.UUID{Bytes: scheduleChange.ID, Valid: true},
		ProrationType:       "upgrade_credit",
		PeriodStart:         pgtype.Timestamptz{Time: sub.CurrentPeriodStart.Time, Valid: true},
		PeriodEnd:           pgtype.Timestamptz{Time: sub.CurrentPeriodEnd.Time, Valid: true},
		DaysTotal:           int32(proration.DaysTotal),
		DaysUsed:            int32(proration.DaysUsed),
		DaysRemaining:       int32(proration.DaysRemaining),
		OriginalAmountCents: oldTotal,
		UsedAmountCents:     int64(float64(oldTotal) * float64(proration.DaysUsed) / float64(proration.DaysTotal)),
		CreditAmountCents:   proration.CreditAmount,
		AppliedToInvoiceID:  pgtype.UUID{Valid: false},
		AppliedToPaymentID:  pgtype.UUID{Valid: false},
	})
	if err != nil {
		sms.logger.Error("Failed to create proration record", zap.Error(err))
	}

	// Process immediate payment if there's a net charge
	if proration.NetAmount > 0 {
		// In real implementation, would process payment through payment service
		sms.logger.Info("Processing proration payment",
			zap.Int64("amount", proration.NetAmount),
			zap.String("subscription_id", subscriptionID.String()))
	}

	// Record state change
	_, err = sms.db.RecordStateChange(ctx, db.RecordStateChangeParams{
		SubscriptionID:    subscriptionID,
		FromStatus:        db.NullSubscriptionStatus{SubscriptionStatus: sub.Status, Valid: true},
		ToStatus:          sub.Status, // Status doesn't change on upgrade
		FromAmountCents:   pgtype.Int8{Int64: oldTotal, Valid: true},
		ToAmountCents:     pgtype.Int8{Int64: newTotal, Valid: true},
		LineItemsSnapshot: toLineItems,
		ChangeReason:      pgtype.Text{String: reason, Valid: true},
		ScheduleChangeID:  pgtype.UUID{Bytes: scheduleChange.ID, Valid: true},
		InitiatedBy:       pgtype.Text{String: "customer", Valid: true},
	})
	if err != nil {
		sms.logger.Error("Failed to record state change", zap.Error(err))
	}

	// Update schedule change status
	_, err = sms.db.UpdateScheduleChangeStatus(ctx, db.UpdateScheduleChangeStatusParams{
		ID:           scheduleChange.ID,
		Status:       "completed",
		ErrorMessage: pgtype.Text{Valid: false},
	})
	if err != nil {
		sms.logger.Error("Failed to update schedule change status", zap.Error(err))
	}

	// Send confirmation email
	if sms.emailService != nil {
		if err := sms.sendSubscriptionChangeEmail(ctx, sub, "upgrade", oldTotal, newTotal, proration.NetAmount); err != nil {
			sms.logger.Error("Failed to send upgrade email", zap.Error(err))
		}
	}

	sms.logger.Info("Subscription upgraded successfully",
		zap.String("subscription_id", subscriptionID.String()),
		zap.Int64("old_amount", oldTotal),
		zap.Int64("new_amount", newTotal),
		zap.Int64("proration_charge", proration.NetAmount))

	return nil
}

// DowngradeSubscription schedules a downgrade for the end of the billing period
func (sms *SubscriptionManagementService) DowngradeSubscription(
	ctx context.Context,
	subscriptionID uuid.UUID,
	newLineItems []requests.LineItemUpdate,
	reason string,
) error {
	// Get current subscription
	sub, err := sms.db.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Validate subscription can be downgraded
	if sub.Status != db.SubscriptionStatusActive {
		return fmt.Errorf("can only downgrade active subscriptions")
	}

	// Schedule for end of period
	scheduleResult := sms.calculator.ScheduleDowngrade(sub.CurrentPeriodEnd.Time, "downgrade")

	// Create schedule change record
	fromLineItems, _ := json.Marshal(map[string]interface{}{
		"product_id": sub.ProductID,
		"price_id":   sub.PriceID,
		"amount":     sub.TotalAmountInCents,
	})
	toLineItems, _ := json.Marshal(newLineItems)

	_, err = sms.db.CreateScheduleChange(ctx, db.CreateScheduleChangeParams{
		SubscriptionID:       subscriptionID,
		ChangeType:           "downgrade",
		ScheduledFor:         pgtype.Timestamptz{Time: scheduleResult.ScheduledFor, Valid: true},
		FromLineItems:        fromLineItems,
		ToLineItems:          toLineItems,
		ProrationAmountCents: pgtype.Int8{Valid: false}, // No proration for downgrades
		ProrationCalculation: []byte("{}"),
		Status:               "scheduled",
		Reason:               pgtype.Text{String: reason, Valid: true},
		InitiatedBy:          pgtype.Text{String: "customer", Valid: true},
		Metadata:             []byte("{}"),
	})
	if err != nil {
		return fmt.Errorf("failed to create schedule change: %w", err)
	}

	// Send confirmation email
	if sms.emailService != nil {
		// Calculate new total for email
		newTotal := sms.calculateNewTotal(ctx, newLineItems)
		if err := sms.sendSubscriptionChangeEmail(ctx, sub, "downgrade", int64(sub.TotalAmountInCents), newTotal, 0); err != nil {
			sms.logger.Error("Failed to send downgrade email", zap.Error(err))
		}
	}

	sms.logger.Info("Subscription downgrade scheduled",
		zap.String("subscription_id", subscriptionID.String()),
		zap.Time("effective_date", scheduleResult.ScheduledFor))

	return nil
}

// CancelSubscription schedules cancellation for the end of the billing period
func (sms *SubscriptionManagementService) CancelSubscription(
	ctx context.Context,
	subscriptionID uuid.UUID,
	reason string,
	feedback string,
) error {
	// Get subscription
	sub, err := sms.db.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Check if already cancelled
	if sub.Status == db.SubscriptionStatusCanceled || sub.CancelAt.Valid {
		return fmt.Errorf("subscription already cancelled")
	}

	// Set cancellation for end of period
	cancelDate := sub.CurrentPeriodEnd.Time

	// Update subscription
	_, err = sms.db.ScheduleSubscriptionCancellation(ctx, db.ScheduleSubscriptionCancellationParams{
		ID:                 subscriptionID,
		CancelAt:           pgtype.Timestamptz{Time: cancelDate, Valid: true},
		CancellationReason: pgtype.Text{String: reason, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to schedule cancellation: %w", err)
	}

	// Create schedule change
	metadata, _ := json.Marshal(map[string]string{"feedback": feedback})
	_, err = sms.db.CreateScheduleChange(ctx, db.CreateScheduleChangeParams{
		SubscriptionID:       subscriptionID,
		ChangeType:           "cancel",
		ScheduledFor:         pgtype.Timestamptz{Time: cancelDate, Valid: true},
		FromLineItems:        []byte("{}"),
		ToLineItems:          []byte("{}"),
		ProrationAmountCents: pgtype.Int8{Valid: false},
		ProrationCalculation: []byte("{}"),
		Status:               "scheduled",
		Reason:               pgtype.Text{String: reason, Valid: true},
		InitiatedBy:          pgtype.Text{String: "customer", Valid: true},
		Metadata:             metadata,
	})
	if err != nil {
		sms.logger.Error("Failed to create schedule change for cancellation", zap.Error(err))
	}

	// Send confirmation email
	if sms.emailService != nil {
		if err := sms.sendSubscriptionChangeEmail(ctx, sub, "cancel", int64(sub.TotalAmountInCents), 0, 0); err != nil {
			sms.logger.Error("Failed to send cancellation email", zap.Error(err))
		}
	}

	sms.logger.Info("Subscription cancellation scheduled",
		zap.String("subscription_id", subscriptionID.String()),
		zap.Time("cancel_date", cancelDate))

	return nil
}

// PauseSubscription pauses a subscription immediately or at a scheduled time
func (sms *SubscriptionManagementService) PauseSubscription(
	ctx context.Context,
	subscriptionID uuid.UUID,
	pauseUntil *time.Time,
	reason string,
) error {
	sub, err := sms.db.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Validate pause request
	if sub.Status != db.SubscriptionStatusActive {
		return fmt.Errorf("can only pause active subscriptions")
	}

	// Calculate any pause credit
	pauseCredit := sms.calculator.CalculatePauseCredit(
		sub.CurrentPeriodStart.Time,
		sub.CurrentPeriodEnd.Time,
		int64(sub.TotalAmountInCents),
		time.Now(),
	)

	// Update subscription status
	pauseEndsAt := pgtype.Timestamptz{Valid: false}
	if pauseUntil != nil {
		pauseEndsAt = pgtype.Timestamptz{Time: *pauseUntil, Valid: true}
	}

	_, err = sms.db.PauseSubscription(ctx, db.PauseSubscriptionParams{
		ID:          subscriptionID,
		PauseEndsAt: pauseEndsAt,
	})
	if err != nil {
		return fmt.Errorf("failed to pause subscription: %w", err)
	}

	// Create schedule change for automatic resume if pause end date provided
	if pauseUntil != nil {
		_, err = sms.db.CreateScheduleChange(ctx, db.CreateScheduleChangeParams{
			SubscriptionID:       subscriptionID,
			ChangeType:           "resume",
			ScheduledFor:         pgtype.Timestamptz{Time: *pauseUntil, Valid: true},
			FromLineItems:        []byte("{}"),
			ToLineItems:          []byte("{}"),
			ProrationAmountCents: pgtype.Int8{Valid: false},
			ProrationCalculation: []byte("{}"),
			Status:               "scheduled",
			Reason:               pgtype.Text{String: "Automatic resume after pause period", Valid: true},
			InitiatedBy:          pgtype.Text{String: "system", Valid: true},
			Metadata:             []byte("{}"),
		})
		if err != nil {
			sms.logger.Error("Failed to schedule automatic resume", zap.Error(err))
		}
	}

	// Create proration record if there's a credit
	if pauseCredit.CreditAmount > 0 {
		_, err = sms.db.CreateProrationRecord(ctx, db.CreateProrationRecordParams{
			SubscriptionID:      subscriptionID,
			ScheduleChangeID:    pgtype.UUID{Valid: false},
			ProrationType:       "pause_credit",
			PeriodStart:         pgtype.Timestamptz{Time: sub.CurrentPeriodStart.Time, Valid: true},
			PeriodEnd:           pgtype.Timestamptz{Time: sub.CurrentPeriodEnd.Time, Valid: true},
			DaysTotal:           int32(pauseCredit.DaysTotal),
			DaysUsed:            int32(pauseCredit.DaysUsed),
			DaysRemaining:       int32(pauseCredit.DaysRemaining),
			OriginalAmountCents: int64(sub.TotalAmountInCents),
			UsedAmountCents:     int64(float64(sub.TotalAmountInCents) * float64(pauseCredit.DaysUsed) / float64(pauseCredit.DaysTotal)),
			CreditAmountCents:   pauseCredit.CreditAmount,
			AppliedToInvoiceID:  pgtype.UUID{Valid: false},
			AppliedToPaymentID:  pgtype.UUID{Valid: false},
		})
		if err != nil {
			sms.logger.Error("Failed to create pause credit record", zap.Error(err))
		}
	}

	// Send confirmation email
	if sms.emailService != nil {
		if err := sms.sendSubscriptionChangeEmail(ctx, sub, "pause", int64(sub.TotalAmountInCents), 0, pauseCredit.CreditAmount); err != nil {
			sms.logger.Error("Failed to send pause email", zap.Error(err))
		}
	}

	sms.logger.Info("Subscription paused",
		zap.String("subscription_id", subscriptionID.String()),
		zap.Int64("credit_amount", pauseCredit.CreditAmount))

	return nil
}

// ResumeSubscription resumes a paused subscription
func (sms *SubscriptionManagementService) ResumeSubscription(
	ctx context.Context,
	subscriptionID uuid.UUID,
) error {
	sub, err := sms.db.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	if sub.Status != db.SubscriptionStatusSuspended {
		return fmt.Errorf("can only resume paused subscriptions")
	}

	// Calculate new billing cycle
	newPeriodStart := time.Now()
	// In real implementation, would use proper interval from price/product
	newPeriodEnd := sms.calculator.AddBillingPeriod(newPeriodStart, "monthly", 1)

	// Update subscription
	_, err = sms.db.ResumeSubscription(ctx, db.ResumeSubscriptionParams{
		ID:                 subscriptionID,
		CurrentPeriodStart: pgtype.Timestamptz{Time: newPeriodStart, Valid: true},
		CurrentPeriodEnd:   pgtype.Timestamptz{Time: newPeriodEnd, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to resume subscription: %w", err)
	}

	// Process immediate payment for new period
	// TODO: Implement payment processing

	// Record state change
	_, err = sms.db.RecordStateChange(ctx, db.RecordStateChangeParams{
		SubscriptionID:    subscriptionID,
		FromStatus:        db.NullSubscriptionStatus{SubscriptionStatus: db.SubscriptionStatusSuspended, Valid: true},
		ToStatus:          db.SubscriptionStatusActive,
		FromAmountCents:   pgtype.Int8{Int64: int64(sub.TotalAmountInCents), Valid: true},
		ToAmountCents:     pgtype.Int8{Int64: int64(sub.TotalAmountInCents), Valid: true},
		LineItemsSnapshot: []byte("{}"),
		ChangeReason:      pgtype.Text{String: "Subscription resumed", Valid: true},
		ScheduleChangeID:  pgtype.UUID{Valid: false},
		InitiatedBy:       pgtype.Text{String: "customer", Valid: true},
	})

	// Send confirmation email
	if sms.emailService != nil {
		if err := sms.sendSubscriptionChangeEmail(ctx, sub, "resume", 0, int64(sub.TotalAmountInCents), 0); err != nil {
			sms.logger.Error("Failed to send resume email", zap.Error(err))
		}
	}

	sms.logger.Info("Subscription resumed",
		zap.String("subscription_id", subscriptionID.String()),
		zap.Time("new_period_start", newPeriodStart),
		zap.Time("new_period_end", newPeriodEnd))

	return nil
}

// ReactivateCancelledSubscription removes a scheduled cancellation
func (sms *SubscriptionManagementService) ReactivateCancelledSubscription(
	ctx context.Context,
	subscriptionID uuid.UUID,
) error {
	_, err := sms.db.ReactivateScheduledCancellation(ctx, subscriptionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("subscription is not scheduled for cancellation")
		}
		return fmt.Errorf("failed to reactivate subscription: %w", err)
	}

	// Cancel any scheduled cancellation changes
	changes, err := sms.db.GetSubscriptionScheduledChanges(ctx, subscriptionID)
	if err == nil {
		for _, change := range changes {
			if change.ChangeType == "cancel" && change.Status == "scheduled" {
				_, err = sms.db.CancelScheduledChange(ctx, change.ID)
				if err != nil {
					sms.logger.Error("Failed to cancel scheduled change", zap.Error(err))
				}
			}
		}
	}

	// Send confirmation email
	if sms.emailService != nil {
		// Get subscription details for email
		sub, err := sms.db.GetSubscription(ctx, subscriptionID)
		if err == nil {
			if err := sms.sendSubscriptionChangeEmail(ctx, sub, "reactivate", int64(sub.TotalAmountInCents), int64(sub.TotalAmountInCents), 0); err != nil {
				sms.logger.Error("Failed to send reactivation email", zap.Error(err))
			}
		}
	}

	sms.logger.Info("Subscription reactivated",
		zap.String("subscription_id", subscriptionID.String()))

	return nil
}

// PreviewChange previews what will happen with a subscription change
func (sms *SubscriptionManagementService) PreviewChange(
	ctx context.Context,
	subscriptionID uuid.UUID,
	changeType string,
	newLineItems []requests.LineItemUpdate,
) (*business.ChangePreview, error) {
	sub, err := sms.db.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	currentAmount := int64(sub.TotalAmountInCents)
	newAmount := sms.calculateNewTotal(ctx, newLineItems)

	preview := &business.ChangePreview{
		CurrentAmount: currentAmount,
		NewAmount:     newAmount,
	}

	switch changeType {
	case "upgrade":
		proration := sms.calculator.CalculateUpgradeProration(
			sub.CurrentPeriodStart.Time,
			sub.CurrentPeriodEnd.Time,
			currentAmount,
			newAmount,
			time.Now(),
		)
		preview.ProrationCredit = proration.CreditAmount
		preview.ImmediateCharge = proration.NetAmount
		preview.EffectiveDate = time.Now()
		preview.ProrationDetails = proration
		preview.Message = sms.calculator.FormatProrationExplanation(proration)

	case "downgrade":
		preview.EffectiveDate = sub.CurrentPeriodEnd.Time
		preview.Message = "Downgrade will take effect at the end of your current billing period. You'll continue with your current plan until then."

	case "cancel":
		preview.EffectiveDate = sub.CurrentPeriodEnd.Time
		preview.Message = "Your subscription will be cancelled at the end of your current billing period. You'll have access until then."
	}

	return preview, nil
}

// ProcessScheduledChanges processes all due scheduled changes
func (sms *SubscriptionManagementService) ProcessScheduledChanges(ctx context.Context) error {
	// Get all due scheduled changes
	changes, err := sms.db.GetDueScheduledChanges(ctx, pgtype.Timestamptz{Time: time.Now(), Valid: true})
	if err != nil {
		return fmt.Errorf("failed to get due scheduled changes: %w", err)
	}

	for _, change := range changes {
		// Update status to processing
		_, err = sms.db.UpdateScheduleChangeStatus(ctx, db.UpdateScheduleChangeStatusParams{
			ID:           change.ID,
			Status:       "processing",
			ErrorMessage: pgtype.Text{Valid: false},
		})
		if err != nil {
			sms.logger.Error("Failed to update change status", zap.Error(err))
			continue
		}

		// Process based on change type
		var processErr error
		switch change.ChangeType {
		case "downgrade":
			processErr = sms.executeDowngrade(ctx, change)
		case "cancel":
			processErr = sms.executeCancellation(ctx, change)
		case "resume":
			processErr = sms.executeResume(ctx, change)
		}

		// Update final status
		if processErr != nil {
			sms.logger.Error("Failed to process scheduled change",
				zap.String("change_id", change.ID.String()),
				zap.String("change_type", change.ChangeType),
				zap.Error(processErr))

			_, err = sms.db.UpdateScheduleChangeStatus(ctx, db.UpdateScheduleChangeStatusParams{
				ID:           change.ID,
				Status:       "failed",
				ErrorMessage: pgtype.Text{String: processErr.Error(), Valid: true},
			})
		} else {
			_, err = sms.db.UpdateScheduleChangeStatus(ctx, db.UpdateScheduleChangeStatusParams{
				ID:           change.ID,
				Status:       "completed",
				ErrorMessage: pgtype.Text{Valid: false},
			})
		}
	}

	return nil
}

// Helper methods

func (sms *SubscriptionManagementService) calculateNewTotal(ctx context.Context, lineItems []requests.LineItemUpdate) int64 {
	// Simplified calculation - in real implementation would look up prices and calculate properly
	var total int64
	for _, item := range lineItems {
		if item.Action != "remove" {
			total += item.UnitAmount * int64(item.Quantity)
		}
	}
	return total
}

func (sms *SubscriptionManagementService) executeDowngrade(ctx context.Context, change db.SubscriptionScheduleChange) error {
	// In real implementation, would update line items based on to_line_items
	sms.logger.Info("Executing scheduled downgrade",
		zap.String("subscription_id", change.SubscriptionID.String()))
	return nil
}

func (sms *SubscriptionManagementService) executeCancellation(ctx context.Context, change db.SubscriptionScheduleChange) error {
	_, err := sms.db.CancelSubscriptionImmediately(ctx, db.CancelSubscriptionImmediatelyParams{
		ID:                 change.SubscriptionID,
		CancellationReason: change.Reason,
	})
	return err
}

func (sms *SubscriptionManagementService) executeResume(ctx context.Context, change db.SubscriptionScheduleChange) error {
	return sms.ResumeSubscription(ctx, change.SubscriptionID)
}

// GetSubscriptionHistory retrieves the state change history for a subscription
func (sms *SubscriptionManagementService) GetSubscriptionHistory(ctx context.Context, subscriptionID uuid.UUID, limit int32) ([]db.SubscriptionStateHistory, error) {
	history, err := sms.db.GetSubscriptionStateHistory(ctx, db.GetSubscriptionStateHistoryParams{
		SubscriptionID: subscriptionID,
		Limit:          limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription history: %w", err)
	}
	return history, nil
}

// sendSubscriptionChangeEmail sends email notifications for subscription changes
func (sms *SubscriptionManagementService) sendSubscriptionChangeEmail(ctx context.Context, sub db.Subscription, changeType string, oldAmount, newAmount, prorationAmount int64) error {
	// Get customer details
	customer, err := sms.db.GetCustomer(ctx, sub.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to get customer: %w", err)
	}

	// Get product details
	product, err := sms.db.GetProductWithoutWorkspaceId(ctx, sub.ProductID)
	if err != nil {
		return fmt.Errorf("failed to get product: %w", err)
	}

	// Get workspace details for merchant name
	workspace, err := sms.db.GetWorkspace(ctx, sub.WorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	// Format amounts
	formatAmount := func(cents int64) string {
		return fmt.Sprintf("$%.2f", float64(cents)/100)
	}

	// Prepare email based on change type
	var subject, htmlBody, textBody string

	switch changeType {
	case "upgrade":
		subject = fmt.Sprintf("Subscription Upgraded - %s", product.Name)
		htmlBody = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #28a745; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .details { background-color: #f8f9fa; padding: 15px; margin: 15px 0; border-radius: 5px; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>Subscription Upgraded Successfully</h2>
        </div>
        <div class="content">
            <p>Hi %s,</p>
            <p>Your subscription to <strong>%s</strong> has been successfully upgraded.</p>
            <div class="details">
                <p><strong>Previous Plan:</strong> %s/month</p>
                <p><strong>New Plan:</strong> %s/month</p>
                %s
            </div>
            <p>Your new plan is effective immediately. Thank you for upgrading!</p>
            <p>If you have any questions, please contact our support team.</p>
            <p>Best regards,<br>%s Team</p>
        </div>
    </div>
</body>
</html>`,
			customer.Name.String, product.Name, formatAmount(oldAmount), formatAmount(newAmount),
			func() string {
				if prorationAmount > 0 {
					return fmt.Sprintf("<p><strong>Proration Charge:</strong> %s</p>", formatAmount(prorationAmount))
				}
				return ""
			}(),
			workspace.Name)

	case "downgrade":
		subject = fmt.Sprintf("Subscription Downgrade Scheduled - %s", product.Name)
		htmlBody = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #ffc107; color: #333; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .details { background-color: #fff3cd; padding: 15px; margin: 15px 0; border-radius: 5px; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>Subscription Downgrade Scheduled</h2>
        </div>
        <div class="content">
            <p>Hi %s,</p>
            <p>Your subscription downgrade for <strong>%s</strong> has been scheduled.</p>
            <div class="details">
                <p><strong>Current Plan:</strong> %s/month</p>
                <p><strong>New Plan:</strong> %s/month</p>
                <p><strong>Effective Date:</strong> End of current billing period</p>
            </div>
            <p>You'll continue to enjoy your current plan benefits until the end of your billing period.</p>
            <p>If you change your mind, you can cancel this downgrade from your account settings.</p>
            <p>Best regards,<br>%s Team</p>
        </div>
    </div>
</body>
</html>`,
			customer.Name.String, product.Name, formatAmount(oldAmount), formatAmount(newAmount), workspace.Name)

	case "cancel":
		subject = fmt.Sprintf("Subscription Cancellation Scheduled - %s", product.Name)
		htmlBody = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #dc3545; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .warning { background-color: #f8d7da; padding: 15px; margin: 15px 0; border-radius: 5px; }
        .button { display: inline-block; padding: 10px 20px; background-color: #007bff; color: white; text-decoration: none; border-radius: 5px; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>Subscription Cancellation Scheduled</h2>
        </div>
        <div class="content">
            <p>Hi %s,</p>
            <p>We're sorry to see you go! Your subscription to <strong>%s</strong> has been scheduled for cancellation.</p>
            <div class="warning">
                <p><strong>Important:</strong> Your subscription will remain active until the end of your current billing period.</p>
                <p>You can continue using all features until then.</p>
            </div>
            <p>Changed your mind? You can reactivate your subscription anytime before it expires.</p>
            <p style="text-align: center;"><a href="#" class="button">Reactivate Subscription</a></p>
            <p>We'd love to hear your feedback about why you're leaving.</p>
            <p>Best regards,<br>%s Team</p>
        </div>
    </div>
</body>
</html>`,
			customer.Name.String, product.Name, workspace.Name)

	case "pause":
		subject = fmt.Sprintf("Subscription Paused - %s", product.Name)
		htmlBody = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #6c757d; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .info { background-color: #e9ecef; padding: 15px; margin: 15px 0; border-radius: 5px; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>Subscription Paused</h2>
        </div>
        <div class="content">
            <p>Hi %s,</p>
            <p>Your subscription to <strong>%s</strong> has been paused as requested.</p>
            <div class="info">
                <p><strong>Status:</strong> Paused</p>
                %s
            </div>
            <p>You can resume your subscription anytime from your account settings.</p>
            <p>Best regards,<br>%s Team</p>
        </div>
    </div>
</body>
</html>`,
			customer.Name.String, product.Name,
			func() string {
				if prorationAmount > 0 {
					return fmt.Sprintf("<p><strong>Credit Applied:</strong> %s</p>", formatAmount(prorationAmount))
				}
				return ""
			}(),
			workspace.Name)

	case "resume":
		subject = fmt.Sprintf("Subscription Resumed - %s", product.Name)
		htmlBody = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #28a745; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .success { background-color: #d4edda; padding: 15px; margin: 15px 0; border-radius: 5px; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>Subscription Resumed</h2>
        </div>
        <div class="content">
            <p>Hi %s,</p>
            <p>Great news! Your subscription to <strong>%s</strong> has been resumed.</p>
            <div class="success">
                <p><strong>Status:</strong> Active</p>
                <p><strong>Plan:</strong> %s/month</p>
            </div>
            <p>You now have full access to all features again. Welcome back!</p>
            <p>Best regards,<br>%s Team</p>
        </div>
    </div>
</body>
</html>`,
			customer.Name.String, product.Name, formatAmount(newAmount), workspace.Name)

	case "reactivate":
		subject = fmt.Sprintf("Subscription Reactivated - %s", product.Name)
		htmlBody = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #28a745; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .success { background-color: #d4edda; padding: 15px; margin: 15px 0; border-radius: 5px; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>Subscription Reactivated</h2>
        </div>
        <div class="content">
            <p>Hi %s,</p>
            <p>Good news! Your subscription to <strong>%s</strong> has been reactivated.</p>
            <div class="success">
                <p>Your scheduled cancellation has been removed and your subscription will continue as normal.</p>
                <p><strong>Plan:</strong> %s/month</p>
            </div>
            <p>Thank you for continuing with us!</p>
            <p>Best regards,<br>%s Team</p>
        </div>
    </div>
</body>
</html>`,
			customer.Name.String, product.Name, formatAmount(oldAmount), workspace.Name)

	default:
		return fmt.Errorf("unknown change type: %s", changeType)
	}

	// Send the email
	params := params.TransactionalEmailParams{
		To:          []string{customer.Email.String},
		Subject:     subject,
		HTMLContent: htmlBody,
		TextContent: textBody, // In production, would generate text version
		Tags: map[string]interface{}{
			"subscription_change": changeType,
			"subscription_id":     sub.ID.String(),
			"workspace_id":        sub.WorkspaceID.String(),
		},
	}

	return sms.emailService.SendTransactionalEmail(ctx, params)
}
