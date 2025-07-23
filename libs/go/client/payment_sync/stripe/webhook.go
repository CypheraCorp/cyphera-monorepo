package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	ps "github.com/cyphera/cyphera-api/libs/go/client/payment_sync"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
	"go.uber.org/zap"
)

// HandleWebhook processes incoming webhook events from Stripe.
// It validates the signature, unmarshals the event data, and maps it to a canonical ps.WebhookEvent.
func (s *StripeService) HandleWebhook(ctx context.Context, requestBody []byte, signatureHeader string) (ps.WebhookEvent, error) {
	if s.client == nil || s.webhookSecret == "" {
		return ps.WebhookEvent{}, fmt.Errorf("stripe service not configured for webhooks (client or secret missing)")
	}

	event, err := webhook.ConstructEvent(requestBody, signatureHeader, s.webhookSecret)
	if err != nil {
		s.logger.Error("Webhook signature verification failed", zap.Error(err))
		return ps.WebhookEvent{SignatureValid: false, RawData: requestBody}, fmt.Errorf("webhook signature verification failed: %w", err)
	}

	s.logger.Info("Received Stripe webhook event", zap.String("event_id", event.ID), zap.String("event_type", string(event.Type)))

	psEvent := ps.WebhookEvent{
		ProviderEventID: event.ID,
		Provider:        s.GetServiceName(),
		EventType:       string(event.Type),
		ReceivedAt:      time.Now().Unix(), // Timestamp when our system processed it
		// CreatedAt:    event.Created, // Timestamp from Stripe when the event was created
		RawData:        requestBody,
		SignatureValid: true,
	}

	// Attempt to unmarshal the event data object into the appropriate Stripe struct
	// and then map it to the canonical ps struct.
	switch event.Type {
	case stripe.EventTypeCustomerCreated,
		stripe.EventTypeCustomerUpdated,
		stripe.EventTypeCustomerDeleted:
		var customer stripe.Customer
		if err := json.Unmarshal(event.Data.Raw, &customer); err != nil {
			s.logger.Error("Failed to unmarshal webhook event data for customer", zap.String("event_type", string(event.Type)), zap.Error(err))
			return psEvent, fmt.Errorf("failed to unmarshal %s data: %w", event.Type, err)
		}
		// For customer.deleted, the customer object is still provided.
		psEvent.Data = mapStripeCustomerToPSCustomer(&customer)

	case stripe.EventTypeInvoiceCreated,
		stripe.EventTypeInvoiceUpdated,
		stripe.EventTypeInvoicePaid,
		stripe.EventTypeInvoicePaymentFailed,
		stripe.EventTypeInvoiceFinalized,
		stripe.EventTypeInvoiceVoided,
		stripe.EventTypeInvoiceMarkedUncollectible:
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			s.logger.Error("Failed to unmarshal webhook event data for invoice", zap.String("event_type", string(event.Type)), zap.Error(err))
			return psEvent, fmt.Errorf("failed to unmarshal %s data: %w", event.Type, err)
		}
		psEvent.Data = mapStripeInvoiceToPSInvoice(&invoice)

	case stripe.EventTypeCustomerSubscriptionCreated,
		stripe.EventTypeCustomerSubscriptionUpdated,
		stripe.EventTypeCustomerSubscriptionDeleted,      // This means the subscription was canceled and fully ended.
		stripe.EventTypeCustomerSubscriptionTrialWillEnd: // Useful for notifications
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			s.logger.Error("Failed to unmarshal webhook event data for subscription", zap.String("event_type", string(event.Type)), zap.Error(err))
			return psEvent, fmt.Errorf("failed to unmarshal %s data: %w", event.Type, err)
		}
		psEvent.Data = mapStripeSubscriptionToPSSubscription(&subscription)

	case stripe.EventTypeProductCreated,
		stripe.EventTypeProductUpdated,
		stripe.EventTypeProductDeleted:
		var product stripe.Product
		if err := json.Unmarshal(event.Data.Raw, &product); err != nil {
			s.logger.Error("Failed to unmarshal webhook event data for product", zap.String("event_type", string(event.Type)), zap.Error(err))
			return psEvent, fmt.Errorf("failed to unmarshal %s data: %w", event.Type, err)
		}
		psEvent.Data = mapStripeProductToPSProduct(&product)

	case stripe.EventTypePriceCreated,
		stripe.EventTypePriceUpdated,
		stripe.EventTypePriceDeleted:
		var price stripe.Price
		if err := json.Unmarshal(event.Data.Raw, &price); err != nil {
			s.logger.Error("Failed to unmarshal webhook event data for price", zap.String("event_type", string(event.Type)), zap.Error(err))
			return psEvent, fmt.Errorf("failed to unmarshal %s data: %w", event.Type, err)
		}
		psEvent.Data = mapStripePriceToPSPrice(&price)

	case stripe.EventTypePaymentIntentSucceeded,
		stripe.EventTypePaymentIntentPaymentFailed,
		stripe.EventTypePaymentIntentProcessing,
		stripe.EventTypePaymentIntentRequiresAction,
		stripe.EventTypePaymentIntentCanceled,
		stripe.EventTypePaymentIntentCreated: // useful for knowing about it early
		var paymentIntent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			s.logger.Error("Failed to unmarshal webhook event data for PaymentIntent", zap.String("event_type", string(event.Type)), zap.Error(err))
			return psEvent, fmt.Errorf("failed to unmarshal %s data: %w", event.Type, err)
		}
		psEvent.Data = mapStripePaymentIntentToPSTransaction(&paymentIntent)

	case stripe.EventTypeChargeSucceeded,
		stripe.EventTypeChargeFailed,
		stripe.EventTypeChargeRefunded,
		stripe.EventTypeChargePending,
		stripe.EventTypeChargeExpired,
		stripe.EventTypeChargeDisputeCreated: // And other dispute events
		var charge stripe.Charge
		if err := json.Unmarshal(event.Data.Raw, &charge); err != nil {
			s.logger.Error("Failed to unmarshal webhook event data for Charge", zap.String("event_type", string(event.Type)), zap.Error(err))
			return psEvent, fmt.Errorf("failed to unmarshal %s data: %w", event.Type, err)
		}
		psEvent.Data = mapStripeChargeToPSTransaction(&charge)

	// TODO: Add cases for other relevant event types like:
	// - TaxRate events: stripe.EventTypeTaxRateCreated, stripe.EventTypeTaxRateUpdated
	// - Payout events (if Stripe Connect Payouts are used directly): stripe.EventTypePayoutCreated, stripe.EventTypePayoutPaid, stripe.EventTypePayoutFailed
	// - Refund events: stripe.EventTypeChargeRefundUpdated (more specific than just charge.refunded)
	// - SubscriptionSchedule events if used: stripe.EventTypeSubscriptionScheduleCreated, etc.
	// - InvoiceItem events: stripe.EventTypeInvoiceItemCreated, etc. (often these are precursors to invoice.created/updated)

	default:
		s.logger.Warn("Unhandled Stripe webhook event type", zap.String("event_type", string(event.Type)), zap.String("event_id", event.ID))
		// Store the raw data for unhandled events if needed for auditing or later processing
		psEvent.Data = event.Data.Raw
	}

	return psEvent, nil
}
