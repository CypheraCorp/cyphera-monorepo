package stripe

import (
	"context"
	"fmt"

	ps "cyphera-api/internal/client/payment_sync"

	"github.com/stripe/stripe-go/v82"
	"go.uber.org/zap"
)

// mapStripePaymentIntentToPSTransaction converts a Stripe PaymentIntent object to the canonical ps.Transaction.
func mapStripePaymentIntentToPSTransaction(pi *stripe.PaymentIntent) ps.Transaction {
	if pi == nil {
		return ps.Transaction{}
	}

	var customerID, invoiceID string
	var providerType, providerStatus string

	if pi.Customer != nil {
		customerID = pi.Customer.ID
	}

	providerType = "stripe"
	providerStatus = string(pi.Status)

	return ps.Transaction{
		ExternalID:            pi.ID,
		UserID:                customerID,
		Type:                  "fiat_on_ramp_stripe_pi",
		Status:                string(pi.Status),
		Amount:                pi.Amount,
		Currency:              string(pi.Currency),
		CreatedAt:             pi.Created,
		Description:           pi.Description,
		Metadata:              pi.Metadata,
		CustomerID:            customerID,
		InvoiceID:             invoiceID,
		ProviderTransactionID: pi.ID,
		ProviderType:          providerType,
		ProviderFiatAmount:    pi.Amount,
		ProviderFiatCurrency:  string(pi.Currency),
		ProviderStatus:        providerStatus,
	}
}

// mapStripeChargeToPSTransaction converts a Stripe Charge object to the canonical ps.Transaction.
func mapStripeChargeToPSTransaction(ch *stripe.Charge) ps.Transaction {
	if ch == nil {
		return ps.Transaction{}
	}
	var customerID, invoiceIDString string
	var providerType, providerStatus string

	if ch.Customer != nil {
		customerID = ch.Customer.ID
	}

	providerType = "stripe"
	providerStatus = string(ch.Status)

	return ps.Transaction{
		ExternalID:            ch.ID,
		UserID:                customerID,
		Type:                  "stripe_charge",
		Status:                string(ch.Status),
		Amount:                ch.Amount,
		Currency:              string(ch.Currency),
		CreatedAt:             ch.Created,
		Description:           ch.Description,
		Metadata:              ch.Metadata,
		CustomerID:            customerID,
		InvoiceID:             invoiceIDString,
		ProviderTransactionID: ch.ID,
		ProviderType:          providerType,
		ProviderFiatAmount:    ch.Amount,
		ProviderFiatCurrency:  string(ch.Currency),
		ProviderStatus:        providerStatus,
	}
}

// GetTransaction retrieves a transaction by its external ID from Stripe.
func (s *StripeService) GetTransaction(ctx context.Context, transactionID string) (ps.Transaction, error) {
	if s.client == nil {
		return ps.Transaction{}, fmt.Errorf("stripe client not configured")
	}
	s.logger.Info("Fetching Stripe transaction", zap.String("transaction_id_ps", transactionID))

	piParams := &stripe.PaymentIntentRetrieveParams{}
	piParams.AddExpand("customer")
	piParams.AddExpand("payment_method")
	piParams.AddExpand("latest_charge.customer")
	piParams.AddExpand("invoice")

	stripePI, errPI := s.client.V1PaymentIntents.Retrieve(ctx, transactionID, piParams)
	if errPI == nil && stripePI != nil {
		s.logger.Info("Found Stripe transaction as PaymentIntent", zap.String("stripe_pi_id", stripePI.ID))
		return mapStripePaymentIntentToPSTransaction(stripePI), nil
	}

	chParams := &stripe.ChargeRetrieveParams{}
	chParams.AddExpand("customer")
	chParams.AddExpand("payment_intent")

	stripeCh, errCh := s.client.V1Charges.Retrieve(ctx, transactionID, chParams)
	if errCh == nil && stripeCh != nil {
		s.logger.Info("Found Stripe transaction as Charge", zap.String("stripe_ch_id", stripeCh.ID))
		return mapStripeChargeToPSTransaction(stripeCh), nil
	}

	s.logger.Error("Failed to fetch Stripe transaction as PaymentIntent or Charge",
		zap.String("transaction_id_ps", transactionID),
		zap.NamedError("payment_intent_error", errPI),
		zap.NamedError("charge_error", errCh),
	)
	return ps.Transaction{}, fmt.Errorf("stripe_service.GetTransaction: transaction %s not found as PI or Charge", transactionID)
}

// ListTransactions retrieves a list of transactions from Stripe.
func (s *StripeService) ListTransactions(ctx context.Context, params ps.ListParams) ([]ps.Transaction, string, error) {
	if s.client == nil {
		return nil, "", fmt.Errorf("stripe client not configured")
	}
	s.logger.Info("Listing Stripe transactions (PaymentIntents)", zap.Any("params", params))

	stripeParams := &stripe.PaymentIntentListParams{}
	if params.Limit > 0 {
		stripeParams.Limit = stripe.Int64(int64(params.Limit))
	}
	if params.StartingAfter != "" {
		stripeParams.StartingAfter = stripe.String(params.StartingAfter)
	}
	if params.EndingBefore != "" {
		stripeParams.EndingBefore = stripe.String(params.EndingBefore)
	}
	if params.CreatedAfter > 0 || params.CreatedBefore > 0 {
		createdRange := &stripe.RangeQueryParams{}
		if params.CreatedAfter > 0 {
			createdRange.GreaterThan = params.CreatedAfter
		}
		if params.CreatedBefore > 0 {
			createdRange.LesserThan = params.CreatedBefore
		}
		stripeParams.CreatedRange = createdRange
	}
	if params.Filters != nil {
		if customerStripeID, ok := params.Filters["customer_stripe_id"].(string); ok && customerStripeID != "" {
			stripeParams.Customer = stripe.String(customerStripeID)
		}
	}

	stripeParams.AddExpand("data.customer")
	stripeParams.AddExpand("data.payment_method")
	stripeParams.AddExpand("data.latest_charge.customer")
	stripeParams.AddExpand("data.invoice")

	var transactions []ps.Transaction
	var lastID string

	for pi, err := range s.client.V1PaymentIntents.List(ctx, stripeParams) {
		if err != nil {
			s.logger.Error("Error during Stripe PaymentIntents list iteration", zap.Error(err))
			return nil, "", fmt.Errorf("stripe_service.ListTransactions: error during iteration: %w", err)
		}
		if pi == nil {
			continue
		}
		transactions = append(transactions, mapStripePaymentIntentToPSTransaction(pi))
		lastID = pi.ID
	}

	nextPageCursor := ""
	if params.Limit > 0 && len(transactions) == int(params.Limit) {
		nextPageCursor = lastID
	}

	s.logger.Info("Successfully listed Stripe PaymentIntents", zap.Int("count", len(transactions)), zap.String("next_cursor", nextPageCursor))
	return transactions, nextPageCursor, nil
}

// CreateStripePaymentIntentForOnRamp creates a Stripe PaymentIntent, e.g., for fiat on-ramping.
func (s *StripeService) CreateStripePaymentIntentForOnRamp(ctx context.Context, amountFiat int64, currencyFiat string, customerStripeID string, description string, metadata map[string]string) (ps.Transaction, error) {
	if s.client == nil {
		return ps.Transaction{}, fmt.Errorf("stripe client not configured")
	}

	params := &stripe.PaymentIntentCreateParams{
		Amount:                  stripe.Int64(amountFiat),
		Currency:                stripe.String(currencyFiat),
		Customer:                stripe.String(customerStripeID),
		Description:             stripe.String(description),
		Metadata:                metadata,
		AutomaticPaymentMethods: &stripe.PaymentIntentCreateAutomaticPaymentMethodsParams{Enabled: stripe.Bool(true)},
	}

	params.AddExpand("customer")
	params.AddExpand("payment_method")
	params.AddExpand("latest_charge.customer")
	params.AddExpand("invoice")

	s.logger.Info("Creating Stripe PaymentIntent for on-ramp", zap.Any("params", params))
	newPI, err := s.client.V1PaymentIntents.Create(ctx, params)
	if err != nil {
		s.logger.Error("Failed to create Stripe PaymentIntent for on-ramp", zap.Error(err), zap.Any("params", params))
		return ps.Transaction{}, fmt.Errorf("stripe_service.CreateStripePaymentIntentForOnRamp: %w", err)
	}

	s.logger.Info("Successfully created Stripe PaymentIntent for on-ramp", zap.String("stripe_pi_id", newPI.ID))
	return mapStripePaymentIntentToPSTransaction(newPI), nil
}

// GetStripePaymentIntentStatus retrieves a Stripe PaymentIntent and maps it to ps.Transaction.
func (s *StripeService) GetStripePaymentIntentStatus(ctx context.Context, stripePaymentIntentID string) (ps.Transaction, error) {
	if s.client == nil {
		return ps.Transaction{}, fmt.Errorf("stripe client not configured")
	}
	s.logger.Info("Fetching Stripe PaymentIntent status", zap.String("stripe_pi_id", stripePaymentIntentID))

	piParams := &stripe.PaymentIntentRetrieveParams{}
	piParams.AddExpand("customer")
	piParams.AddExpand("payment_method")
	piParams.AddExpand("latest_charge.customer")
	piParams.AddExpand("invoice")

	stripePI, err := s.client.V1PaymentIntents.Retrieve(ctx, stripePaymentIntentID, piParams)
	if err != nil {
		s.logger.Error("Failed to fetch Stripe PaymentIntent for status", zap.Error(err), zap.String("stripe_pi_id", stripePaymentIntentID))
		return ps.Transaction{}, fmt.Errorf("stripe_service.GetStripePaymentIntentStatus: %w", err)
	}
	return mapStripePaymentIntentToPSTransaction(stripePI), nil
}

// CapturePaymentIntent captures a previously created Stripe payment intent.
func (s *StripeService) CapturePaymentIntent(ctx context.Context, stripePaymentIntentID string, amountToCapture int64) (ps.Transaction, error) {
	if s.client == nil {
		return ps.Transaction{}, fmt.Errorf("stripe client not configured")
	}

	params := &stripe.PaymentIntentCaptureParams{}
	if amountToCapture > 0 {
		params.AmountToCapture = stripe.Int64(amountToCapture)
	}

	params.AddExpand("customer")
	params.AddExpand("payment_method")
	params.AddExpand("latest_charge.customer")
	params.AddExpand("invoice")

	s.logger.Info("Capturing Stripe PaymentIntent", zap.String("stripe_pi_id", stripePaymentIntentID), zap.Int64("amount_to_capture", amountToCapture))
	capturedPI, err := s.client.V1PaymentIntents.Capture(ctx, stripePaymentIntentID, params)
	if err != nil {
		s.logger.Error("Failed to capture Stripe PaymentIntent", zap.Error(err), zap.String("stripe_pi_id", stripePaymentIntentID))
		return ps.Transaction{}, fmt.Errorf("stripe_service.CapturePaymentIntent: %w", err)
	}

	s.logger.Info("Successfully captured Stripe PaymentIntent", zap.String("stripe_pi_id", capturedPI.ID))
	return mapStripePaymentIntentToPSTransaction(capturedPI), nil
}

// CreateRefund creates a refund for a charge in Stripe.
func (s *StripeService) CreateRefund(ctx context.Context, chargeExternalID string, paymentIntentExternalID string, amount int64, reason string, metadata map[string]string) (ps.Transaction, error) {
	if s.client == nil {
		return ps.Transaction{}, fmt.Errorf("stripe client not configured")
	}

	params := &stripe.RefundCreateParams{
		Metadata: metadata,
	}
	if chargeExternalID != "" {
		params.Charge = stripe.String(chargeExternalID)
	} else if paymentIntentExternalID != "" {
		params.PaymentIntent = stripe.String(paymentIntentExternalID)
	} else {
		return ps.Transaction{}, fmt.Errorf("either chargeExternalID or paymentIntentExternalID is required to create a refund")
	}

	if amount > 0 {
		params.Amount = stripe.Int64(amount)
	}
	if reason != "" {
		params.Reason = stripe.String(reason)
	}

	params.AddExpand("payment_intent.customer")
	params.AddExpand("payment_intent.payment_method")
	params.AddExpand("payment_intent.invoice")
	params.AddExpand("charge.customer")

	s.logger.Info("Creating Stripe refund", zap.Any("params", params))
	stripeRefund, err := s.client.V1Refunds.Create(ctx, params)
	if err != nil {
		s.logger.Error("Failed to create Stripe refund", zap.Error(err), zap.Any("params", params))
		return ps.Transaction{}, fmt.Errorf("stripe_service.CreateRefund: %w", err)
	}

	if stripeRefund.PaymentIntent != nil {
		s.logger.Info("Refund created for PaymentIntent, fetching PI for mapping", zap.String("refund_id", stripeRefund.ID), zap.String("pi_id", stripeRefund.PaymentIntent.ID))
		updatedPI, piErr := s.GetStripePaymentIntentStatus(ctx, stripeRefund.PaymentIntent.ID)
		if piErr == nil {
			updatedPI.Status = string(stripeRefund.Status)
			updatedPI.Type = "refund_for_stripe_pi"
			return updatedPI, nil
		}
		s.logger.Warn("Failed to fetch PaymentIntent associated with refund for detailed mapping", zap.Error(piErr), zap.String("refund_id", stripeRefund.ID))
	}

	if stripeRefund.Charge != nil {
		s.logger.Info("Refund created for Charge, fetching Charge for mapping", zap.String("refund_id", stripeRefund.ID), zap.String("charge_id", stripeRefund.Charge.ID))
		chParams := &stripe.ChargeRetrieveParams{}
		chParams.AddExpand("customer")
		chParams.AddExpand("payment_intent")
		retrievedCh, chErr := s.client.V1Charges.Retrieve(ctx, stripeRefund.Charge.ID, chParams)
		if chErr == nil && retrievedCh != nil {
			mappedTx := mapStripeChargeToPSTransaction(retrievedCh)
			mappedTx.Status = string(stripeRefund.Status)
			mappedTx.Type = "refund_for_stripe_charge"
			return mappedTx, nil
		}
		s.logger.Warn("Failed to fetch Charge associated with refund for detailed mapping", zap.Error(chErr), zap.String("refund_id", stripeRefund.ID))
	}

	s.logger.Info("Successfully created Stripe refund, mapping directly from refund object (simplified)", zap.String("refund_id", stripeRefund.ID))
	return ps.Transaction{
		ExternalID:            stripeRefund.ID,
		Amount:                stripeRefund.Amount,
		Currency:              string(stripeRefund.Currency),
		Status:                string(stripeRefund.Status),
		Type:                  "stripe_refund_direct",
		CreatedAt:             stripeRefund.Created,
		Metadata:              stripeRefund.Metadata,
		ProviderTransactionID: stripeRefund.ID,
		ProviderType:          "stripe",
		ProviderFiatAmount:    stripeRefund.Amount,
		ProviderFiatCurrency:  string(stripeRefund.Currency),
		ProviderStatus:        string(stripeRefund.Status),
	}, nil
}
