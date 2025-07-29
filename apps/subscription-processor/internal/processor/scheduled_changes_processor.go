package processor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// ScheduledChangesProcessor processes scheduled subscription changes
type ScheduledChangesProcessor struct {
	service        interfaces.SubscriptionManagementService
	db             *db.Queries
	logger         *zap.Logger
	interval       time.Duration
	stopCh         chan struct{}
	wg             sync.WaitGroup
	paymentService interfaces.PaymentService
	emailService   interfaces.EmailService
}

// emailServiceAdapter adapts interfaces.EmailService to services.IEmailService
type emailServiceAdapter struct {
	service interfaces.EmailService
}

func (a *emailServiceAdapter) SendTransactionalEmail(ctx context.Context, params params.TransactionalEmailParams) error {
	return a.service.SendTransactionalEmail(ctx, params)
}

func (a *emailServiceAdapter) SendBatchEmails(ctx context.Context, requests []requests.BatchEmailRequest) ([]responses.BatchEmailResult, error) {
	return a.service.SendBatchEmails(ctx, requests)
}

func (a *emailServiceAdapter) SendDunningEmail(ctx context.Context, template *db.DunningEmailTemplate, data map[string]business.EmailData, toEmail string) error {
	return a.service.SendDunningEmail(ctx, template, data, toEmail)
}

// paymentServiceAdapter adapts interfaces.PaymentService to services.IPaymentService
type paymentServiceAdapter struct {
	service interfaces.PaymentService
}

func (a *paymentServiceAdapter) CreatePaymentFromSubscriptionEvent(ctx context.Context, params params.CreatePaymentFromSubscriptionEventParams) (*db.Payment, error) {
	return a.service.CreatePaymentFromSubscriptionEvent(ctx, params)
}

func (a *paymentServiceAdapter) CreateComprehensivePayment(ctx context.Context, params params.CreateComprehensivePaymentParams) (*db.Payment, error) {
	return a.service.CreateComprehensivePayment(ctx, params)
}

func (a *paymentServiceAdapter) GetPayment(ctx context.Context, params params.GetPaymentParams) (*db.Payment, error) {
	return a.service.GetPayment(ctx, params)
}

func (a *paymentServiceAdapter) GetPaymentByTransactionHash(ctx context.Context, txHash string) (*db.Payment, error) {
	return a.service.GetPaymentByTransactionHash(ctx, txHash)
}

func (a *paymentServiceAdapter) ListPayments(ctx context.Context, params params.ListPaymentsParams) ([]db.Payment, error) {
	return a.service.ListPayments(ctx, params)
}

func (a *paymentServiceAdapter) UpdatePaymentStatus(ctx context.Context, params params.UpdatePaymentStatusParams) (*db.Payment, error) {
	return a.service.UpdatePaymentStatus(ctx, params)
}

func (a *paymentServiceAdapter) GetPaymentMetrics(ctx context.Context, workspaceID uuid.UUID, startTime, endTime time.Time, currency string) (*db.GetPaymentMetricsRow, error) {
	return a.service.GetPaymentMetrics(ctx, workspaceID, startTime, endTime, currency)
}

func (a *paymentServiceAdapter) CreateManualPayment(ctx context.Context, params params.CreateManualPaymentParams) (*db.Payment, error) {
	return a.service.CreateManualPayment(ctx, params)
}

// NewScheduledChangesProcessor creates a new scheduled changes processor
func NewScheduledChangesProcessor(
	dbQueries *db.Queries,
	paymentService interfaces.PaymentService,
	emailService interfaces.EmailService,
	interval time.Duration,
) *ScheduledChangesProcessor {
	// Create adapters to bridge interface differences
	emailAdapter := &emailServiceAdapter{service: emailService}
	paymentAdapter := &paymentServiceAdapter{service: paymentService}

	return &ScheduledChangesProcessor{
		service:        services.NewSubscriptionManagementService(dbQueries, paymentAdapter, emailAdapter),
		db:             dbQueries,
		logger:         logger.Log,
		interval:       interval,
		stopCh:         make(chan struct{}),
		paymentService: paymentService,
		emailService:   emailService,
	}
}

// Start begins processing scheduled changes
func (p *ScheduledChangesProcessor) Start() {
	p.wg.Add(1)
	go p.run()
	p.logger.Info("Scheduled changes processor started",
		zap.Duration("interval", p.interval))
}

// Stop gracefully shuts down the processor
func (p *ScheduledChangesProcessor) Stop() {
	p.logger.Info("Stopping scheduled changes processor...")
	close(p.stopCh)
	p.wg.Wait()
	p.logger.Info("Scheduled changes processor stopped")
}

// run is the main processing loop
func (p *ScheduledChangesProcessor) run() {
	defer p.wg.Done()

	// Process immediately on startup
	p.ProcessChanges()

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.ProcessChanges()
		case <-p.stopCh:
			return
		}
	}
}

// ProcessChanges processes all due scheduled changes
func (p *ScheduledChangesProcessor) ProcessChanges() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	startTime := time.Now()
	p.logger.Info("Starting scheduled changes processing run")

	// Process scheduled subscription changes
	if err := p.service.ProcessScheduledChanges(ctx); err != nil {
		p.logger.Error("Failed to process scheduled changes",
			zap.Error(err))
		return
	}

	// Also check for subscriptions that need to be cancelled based on cancel_at field
	if err := p.processCancellations(ctx); err != nil {
		p.logger.Error("Failed to process cancellations",
			zap.Error(err))
	}

	// Check for paused subscriptions that should resume
	if err := p.processResumptions(ctx); err != nil {
		p.logger.Error("Failed to process resumptions",
			zap.Error(err))
	}

	// Process dunning campaigns that need final action
	if err := p.processDunningFinalActions(ctx); err != nil {
		p.logger.Error("Failed to process dunning final actions",
			zap.Error(err))
	}

	duration := time.Since(startTime)
	p.logger.Info("Completed scheduled changes processing run",
		zap.Duration("duration", duration))
}

// processCancellations handles subscriptions with cancel_at dates that have passed
func (p *ScheduledChangesProcessor) processCancellations(ctx context.Context) error {
	// Get subscriptions due for cancellation
	subscriptions, err := p.db.GetSubscriptionsDueForCancellation(ctx, pgtype.Timestamptz{Time: time.Now(), Valid: true})
	if err != nil {
		return err
	}

	for _, sub := range subscriptions {
		p.logger.Info("Processing scheduled cancellation",
			zap.String("subscription_id", sub.ID.String()),
			zap.Time("cancel_at", sub.CancelAt.Time))

		// Update subscription status to cancelled
		_, err := p.db.CancelSubscriptionImmediately(ctx, db.CancelSubscriptionImmediatelyParams{
			ID:                 sub.ID,
			CancellationReason: sub.CancellationReason,
		})
		if err != nil {
			p.logger.Error("Failed to cancel subscription",
				zap.String("subscription_id", sub.ID.String()),
				zap.Error(err))
			continue
		}

		// Record state change
		_, err = p.db.RecordStateChange(ctx, db.RecordStateChangeParams{
			SubscriptionID:    sub.ID,
			FromStatus:        db.NullSubscriptionStatus{SubscriptionStatus: db.SubscriptionStatusActive, Valid: true},
			ToStatus:          db.SubscriptionStatusCanceled,
			FromAmountCents:   pgtype.Int8{Int64: int64(sub.TotalAmountInCents), Valid: true},
			ToAmountCents:     pgtype.Int8{Int64: 0, Valid: true},
			LineItemsSnapshot: []byte("{}"),
			ChangeReason:      pgtype.Text{String: "Scheduled cancellation processed", Valid: true},
			ScheduleChangeID:  pgtype.UUID{Valid: false},
			InitiatedBy:       pgtype.Text{String: "system", Valid: true},
		})
		if err != nil {
			p.logger.Error("Failed to record cancellation state change",
				zap.String("subscription_id", sub.ID.String()),
				zap.Error(err))
		}

		// Send cancellation email
		if p.emailService != nil {
			err := p.sendCancellationEmail(ctx, sub)
			if err != nil {
				p.logger.Error("Failed to send cancellation email",
					zap.String("subscription_id", sub.ID.String()),
					zap.Error(err))
			}
		}
	}

	return nil
}

// processResumptions handles paused subscriptions that should resume
func (p *ScheduledChangesProcessor) processResumptions(ctx context.Context) error {
	// Get paused subscriptions due for resumption
	subscriptions, err := p.db.GetSubscriptionsDueForResumption(ctx, pgtype.Timestamptz{Time: time.Now(), Valid: true})
	if err != nil {
		return err
	}

	for _, sub := range subscriptions {
		p.logger.Info("Processing scheduled resumption",
			zap.String("subscription_id", sub.ID.String()),
			zap.Time("pause_ends_at", sub.PauseEndsAt.Time))

		// Use the service to resume the subscription
		err := p.service.ResumeSubscription(ctx, sub.ID)
		if err != nil {
			p.logger.Error("Failed to resume subscription",
				zap.String("subscription_id", sub.ID.String()),
				zap.Error(err))
			continue
		}

		// Send resumption email
		if p.emailService != nil {
			err := p.sendResumptionEmail(ctx, sub)
			if err != nil {
				p.logger.Error("Failed to send resumption email",
					zap.String("subscription_id", sub.ID.String()),
					zap.Error(err))
			}
		}
	}

	return nil
}

// processDunningFinalActions handles dunning campaigns that have exhausted all retry attempts
func (p *ScheduledChangesProcessor) processDunningFinalActions(ctx context.Context) error {
	// Get campaigns that need final action
	campaigns, err := p.db.GetCampaignsNeedingFinalAction(ctx)
	if err != nil {
		return fmt.Errorf("failed to get campaigns needing final action: %w", err)
	}

	p.logger.Info("processDunningFinalActions()",
		zap.Int("campaign_count", len(campaigns)))

	for _, campaign := range campaigns {
		p.logger.Info("Processing final action for dunning campaign",
			zap.String("campaign_id", campaign.ID.String()),
			zap.String("subscription_id", uuid.UUID(campaign.SubscriptionID.Bytes).String()),
			zap.String("final_action", campaign.FinalAction))

		// Execute the final action based on configuration
		switch campaign.FinalAction {
		case "cancel":
			// Schedule cancellation for the subscription
			if campaign.SubscriptionID.Valid {
				_, err := p.db.ScheduleSubscriptionCancellation(ctx, db.ScheduleSubscriptionCancellationParams{
					ID:                 campaign.SubscriptionID.Bytes,
					CancelAt:           pgtype.Timestamptz{Time: time.Now(), Valid: true},
					CancellationReason: pgtype.Text{String: "Failed dunning process - automatic cancellation", Valid: true},
				})
				if err != nil {
					p.logger.Error("Failed to schedule subscription cancellation",
						zap.String("campaign_id", campaign.ID.String()),
						zap.String("subscription_id", uuid.UUID(campaign.SubscriptionID.Bytes).String()),
						zap.Error(err))
					continue
				}

				// Mark the campaign as having taken final action
				_, err = p.db.FailDunningCampaign(ctx, db.FailDunningCampaignParams{
					ID:               campaign.ID,
					FinalActionTaken: pgtype.Text{String: "cancel", Valid: true},
				})
				if err != nil {
					p.logger.Error("Failed to update campaign final action status",
						zap.String("campaign_id", campaign.ID.String()),
						zap.Error(err))
				}

				// Send cancellation notification email
				if p.emailService != nil {
					err := p.sendDunningCancellationEmail(ctx, campaign.SubscriptionID.Bytes)
					if err != nil {
						p.logger.Error("Failed to send dunning cancellation email",
							zap.String("subscription_id", uuid.UUID(campaign.SubscriptionID.Bytes).String()),
							zap.Error(err))
					}
				}
			}

		case "pause":
			// Pause the subscription
			if campaign.SubscriptionID.Valid {
				_, err := p.db.PauseSubscription(ctx, db.PauseSubscriptionParams{
					ID:          uuid.UUID(campaign.SubscriptionID.Bytes),
					PauseEndsAt: pgtype.Timestamptz{Valid: false}, // Indefinite pause
				})
				if err != nil {
					p.logger.Error("Failed to pause subscription",
						zap.String("campaign_id", campaign.ID.String()),
						zap.String("subscription_id", uuid.UUID(campaign.SubscriptionID.Bytes).String()),
						zap.Error(err))
					continue
				}

				// Mark the campaign as having taken final action
				_, err = p.db.FailDunningCampaign(ctx, db.FailDunningCampaignParams{
					ID:               campaign.ID,
					FinalActionTaken: pgtype.Text{String: "pause", Valid: true},
				})
				if err != nil {
					p.logger.Error("Failed to update campaign final action status",
						zap.String("campaign_id", campaign.ID.String()),
						zap.Error(err))
				}
			}

		case "downgrade":
			// TODO: Implement downgrade logic based on final_action_config
			p.logger.Warn("Downgrade final action not yet implemented",
				zap.String("campaign_id", campaign.ID.String()))

		default:
			p.logger.Warn("Unknown final action type",
				zap.String("campaign_id", campaign.ID.String()),
				zap.String("final_action", campaign.FinalAction))
		}
	}

	return nil
}

// sendCancellationEmail sends an email notification when a subscription is cancelled
func (p *ScheduledChangesProcessor) sendCancellationEmail(ctx context.Context, sub db.Subscription) error {
	// Get customer details
	customer, err := p.db.GetCustomer(ctx, sub.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to get customer: %w", err)
	}

	// Get product details
	product, err := p.db.GetProductWithoutWorkspaceId(ctx, sub.ProductID)
	if err != nil {
		return fmt.Errorf("failed to get product: %w", err)
	}

	// Get workspace details
	workspace, err := p.db.GetWorkspace(ctx, sub.WorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	// Prepare email
	subject := fmt.Sprintf("Subscription Cancelled - %s", product.Name)
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #dc3545; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>Subscription Cancelled</h2>
        </div>
        <div class="content">
            <p>Hi %s,</p>
            <p>Your subscription to <strong>%s</strong> has been cancelled as scheduled.</p>
            <p>We're sorry to see you go. If you'd like to resubscribe in the future, you can do so anytime from our website.</p>
            <p>Thank you for being a valued customer.</p>
            <p>Best regards,<br>%s Team</p>
        </div>
    </div>
</body>
</html>`, customer.Name.String, product.Name, workspace.Name)

	emailParams := params.TransactionalEmailParams{
		To:          []string{customer.Email.String},
		Subject:     subject,
		HTMLContent: htmlBody,
		Tags: map[string]interface{}{
			"category":        "subscription_cancelled",
			"subscription_id": sub.ID.String(),
		},
	}

	return p.emailService.SendTransactionalEmail(ctx, emailParams)
}

// sendResumptionEmail sends an email notification when a subscription is resumed
func (p *ScheduledChangesProcessor) sendResumptionEmail(ctx context.Context, sub db.Subscription) error {
	// Get customer details
	customer, err := p.db.GetCustomer(ctx, sub.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to get customer: %w", err)
	}

	// Get product details
	product, err := p.db.GetProductWithoutWorkspaceId(ctx, sub.ProductID)
	if err != nil {
		return fmt.Errorf("failed to get product: %w", err)
	}

	// Get workspace details
	workspace, err := p.db.GetWorkspace(ctx, sub.WorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	// Prepare email
	subject := fmt.Sprintf("Subscription Resumed - %s", product.Name)
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #28a745; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
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
            <p>Your subscription to <strong>%s</strong> has been automatically resumed as scheduled.</p>
            <p>You now have full access to all features. Welcome back!</p>
            <p>If you have any questions, please don't hesitate to contact us.</p>
            <p>Best regards,<br>%s Team</p>
        </div>
    </div>
</body>
</html>`, customer.Name.String, product.Name, workspace.Name)

	emailParams := params.TransactionalEmailParams{
		To:          []string{customer.Email.String},
		Subject:     subject,
		HTMLContent: htmlBody,
		Tags: map[string]interface{}{
			"category":        "subscription_resumed",
			"subscription_id": sub.ID.String(),
		},
	}

	return p.emailService.SendTransactionalEmail(ctx, emailParams)
}

// sendDunningCancellationEmail sends an email when a subscription is cancelled due to failed dunning
func (p *ScheduledChangesProcessor) sendDunningCancellationEmail(ctx context.Context, subscriptionID uuid.UUID) error {
	// Get subscription details
	sub, err := p.db.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Get customer details
	customer, err := p.db.GetCustomer(ctx, sub.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to get customer: %w", err)
	}

	// Get product details
	product, err := p.db.GetProductWithoutWorkspaceId(ctx, sub.ProductID)
	if err != nil {
		return fmt.Errorf("failed to get product: %w", err)
	}

	// Get workspace details
	workspace, err := p.db.GetWorkspace(ctx, sub.WorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	// Prepare email
	subject := fmt.Sprintf("Subscription Cancelled - Payment Issues - %s", product.Name)
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #dc3545; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .warning { background-color: #f8d7da; border: 1px solid #dc3545; padding: 15px; margin: 15px 0; }
        .button { display: inline-block; padding: 10px 20px; background-color: #007bff; color: white; text-decoration: none; border-radius: 5px; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>Subscription Cancelled</h2>
        </div>
        <div class="content">
            <p>Hi %s,</p>
            <p>We regret to inform you that your subscription to <strong>%s</strong> has been cancelled due to repeated payment failures.</p>
            <div class="warning">
                <p>Despite multiple attempts, we were unable to process your payment. As a result, your access to the service has been terminated.</p>
            </div>
            <p>If you'd like to reactivate your subscription, please update your payment method and resubscribe:</p>
            <p style="text-align: center;"><a href="#" class="button">Resubscribe Now</a></p>
            <p>We value you as a customer and would be happy to welcome you back anytime.</p>
            <p>If you believe this was an error or need assistance, please contact us at support@%s.</p>
            <p>Best regards,<br>%s Team</p>
        </div>
    </div>
</body>
</html>`, customer.Name.String, product.Name, workspace.Name, workspace.Name)

	emailParams := params.TransactionalEmailParams{
		To:          []string{customer.Email.String},
		Subject:     subject,
		HTMLContent: htmlBody,
		Tags: map[string]interface{}{
			"category":        "dunning_cancellation",
			"subscription_id": sub.ID.String(),
		},
	}

	return p.emailService.SendTransactionalEmail(ctx, emailParams)
}
