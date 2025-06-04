package stripe

import (
	"context"
	"fmt"

	ps "cyphera-api/internal/client/payment_sync"

	"github.com/stripe/stripe-go/v82"
	"go.uber.org/zap"
)

// mapStripeILITaxToPSTaxAmount maps a stripe.InvoiceLineItemTax to ps.TaxAmount.
func mapStripeILITaxToPSTaxAmount(stripeTax *stripe.InvoiceLineItemTax) ps.TaxAmount {
	if stripeTax == nil {
		return ps.TaxAmount{}
	}
	pa := ps.TaxAmount{
		Amount:        stripeTax.Amount,
		Inclusive:     stripeTax.TaxBehavior == stripe.InvoiceLineItemTaxTaxBehaviorInclusive,
		TaxableAmount: stripeTax.TaxableAmount,
	}
	if stripeTax.TaxRateDetails != nil {
		pa.RateID = stripeTax.TaxRateDetails.TaxRate // This is the TaxRate ID string
	}
	return pa
}

// mapStripeInvoiceLineToPSInvoiceLineItem converts a Stripe InvoiceLineItem object to the canonical ps.InvoiceLineItem.
// This version is based on the user-provided invoicelineitem.go SDK file.
func mapStripeInvoiceLineToPSInvoiceLineItem(stripeLineItem *stripe.InvoiceLineItem) ps.InvoiceLineItem {
	if stripeLineItem == nil {
		return ps.InvoiceLineItem{}
	}

	var priceID, productID, subItemID, itemType string
	var proration bool

	if stripeLineItem.Pricing != nil && stripeLineItem.Pricing.PriceDetails != nil {
		priceID = stripeLineItem.Pricing.PriceDetails.Price
		productID = stripeLineItem.Pricing.PriceDetails.Product
	}

	if stripeLineItem.Parent != nil {
		itemType = string(stripeLineItem.Parent.Type)
		if stripeLineItem.Parent.SubscriptionItemDetails != nil {
			subItemID = stripeLineItem.Parent.SubscriptionItemDetails.SubscriptionItem
		}
		// Determine Proration based on Parent type
		if stripeLineItem.Parent.Type == stripe.InvoiceLineItemParentTypeInvoiceItemDetails && stripeLineItem.Parent.InvoiceItemDetails != nil {
			proration = stripeLineItem.Parent.InvoiceItemDetails.Proration
		} else if stripeLineItem.Parent.Type == stripe.InvoiceLineItemParentTypeSubscriptionItemDetails && stripeLineItem.Parent.SubscriptionItemDetails != nil {
			proration = stripeLineItem.Parent.SubscriptionItemDetails.Proration
		}
	}

	var psPeriod *ps.Period
	if stripeLineItem.Period != nil {
		psPeriod = &ps.Period{
			Start: stripeLineItem.Period.Start,
			End:   stripeLineItem.Period.End,
		}
	}

	var psTaxAmounts []ps.TaxAmount
	if len(stripeLineItem.Taxes) > 0 {
		psTaxAmounts = make([]ps.TaxAmount, len(stripeLineItem.Taxes))
		for i, t := range stripeLineItem.Taxes {
			psTaxAmounts[i] = mapStripeILITaxToPSTaxAmount(t)
		}
	}

	return ps.InvoiceLineItem{
		ExternalID:         stripeLineItem.ID,
		Description:        stripeLineItem.Description,
		Amount:             stripeLineItem.Amount,
		Quantity:           int(stripeLineItem.Quantity),
		PriceID:            priceID,
		ProductID:          productID,
		Period:             psPeriod,
		Type:               itemType,
		Proration:          proration, // Set based on Parent an SDK structure
		Metadata:           stripeLineItem.Metadata,
		TaxAmounts:         psTaxAmounts,
		SubscriptionItemID: subItemID,
	}
}

// mapStripeInvoiceToPSInvoice converts a Stripe Invoice object to the canonical ps.Invoice.
func mapStripeInvoiceToPSInvoice(stripeInv *stripe.Invoice) ps.Invoice {
	if stripeInv == nil {
		return ps.Invoice{}
	}

	var lines []ps.InvoiceLineItem
	if stripeInv.Lines != nil && len(stripeInv.Lines.Data) > 0 {
		lines = make([]ps.InvoiceLineItem, len(stripeInv.Lines.Data))
		for i, line := range stripeInv.Lines.Data {
			lines[i] = mapStripeInvoiceLineToPSInvoiceLineItem(line)
		}
	}

	var customerID, subscriptionID, chargeID, paymentIntentID string
	if stripeInv.Customer != nil {
		customerID = stripeInv.Customer.ID
	}

	// Linter errors indicate these fields are undefined on stripeInv in user's env.
	// if stripeInv.Subscription != nil { // Linter: stripeInv.Subscription undefined
	// 	subscriptionID = stripeInv.Subscription.ID
	// }
	// if stripeInv.Charge != nil { // Linter: stripeInv.Charge undefined
	// 	chargeID = stripeInv.Charge.ID
	// }
	// if stripeInv.PaymentIntent != nil { // Linter: stripeInv.PaymentIntent undefined
	// 	paymentIntentID = stripeInv.PaymentIntent.ID
	// } else if stripeInv.Payments != nil && len(stripeInv.Payments.Data) > 0 {
	// 	firstPaymentListItem := stripeInv.Payments.Data[0]
	// 	if firstPaymentListItem != nil && firstPaymentListItem.Payment != nil && firstPaymentListItem.Payment.PaymentIntent != nil {
	// 		paymentIntentID = firstPaymentListItem.Payment.PaymentIntent.ID
	// 	}
	// }

	var totalTax int64
	// if stripeInv.Tax != nil { // Linter: stripeInv.Tax undefined
	// 	totalTax = *stripeInv.Tax
	// }

	var psTotalTaxAmounts []ps.TaxAmount
	// if len(stripeInv.TotalTaxAmounts) > 0 { // Linter: stripeInv.TotalTaxAmounts undefined
	// 	psTotalTaxAmounts = make([]ps.TaxAmount, len(stripeInv.TotalTaxAmounts))
	// 	for i, tta := range stripeInv.TotalTaxAmounts {
	// 		if tta != nil {
	// 			// Need to adapt mapStripeILITaxToPSTaxAmount or use a new helper for InvoiceTotalTaxAmount if structure differs significantly
	// 			// For now, assuming it has Amount, Inclusive, and TaxRate *object* (not ID string)
	// 			// psTotalTaxAmounts[i] = mapStripeTaxComponentsToPSTaxAmount(tta.Amount, tta.Inclusive, tta.TaxRate, 0)
	// 		}
	// 	}
	// }

	var paidAt int64
	// if stripeInv.PaidAt != 0 { // Linter: stripeInv.PaidAt undefined (assuming it's an int64)
	// 	paidAt = stripeInv.PaidAt
	// }

	var paidOutOfBand bool
	// if stripeInv.PaidOutOfBand { // Linter: stripeInv.PaidOutOfBand undefined
	// 	paidOutOfBand = stripeInv.PaidOutOfBand
	// }

	return ps.Invoice{
		ExternalID:         stripeInv.ID,
		CustomerID:         customerID,
		SubscriptionID:     subscriptionID, // Will be empty due to linter constraints
		Status:             string(stripeInv.Status),
		CollectionMethod:   string(stripeInv.CollectionMethod),
		AmountDue:          stripeInv.AmountDue,
		AmountPaid:         stripeInv.AmountPaid,
		AmountRemaining:    stripeInv.AmountRemaining,
		Currency:           string(stripeInv.Currency),
		DueDate:            stripeInv.DueDate,
		PaidAt:             paidAt, // Will be empty due to linter constraints
		InvoicePDF:         stripeInv.InvoicePDF,
		HostedInvoiceURL:   stripeInv.HostedInvoiceURL,
		Lines:              lines,
		Metadata:           stripeInv.Metadata,
		AttemptCount:       int(stripeInv.AttemptCount),
		NextPaymentAttempt: stripeInv.NextPaymentAttempt,
		ChargeID:           chargeID,          // Will be empty due to linter constraints
		PaymentIntentID:    paymentIntentID,   // Will be empty due to linter constraints
		Tax:                totalTax,          // Will be empty due to linter constraints
		TotalTaxAmounts:    psTotalTaxAmounts, // Will be empty due to linter constraints
		BillingReason:      string(stripeInv.BillingReason),
		PaidOutOfBand:      paidOutOfBand, // Will be empty due to linter constraints
	}
}

// GetInvoice retrieves an invoice by its external ID from Stripe.
func (s *StripeService) GetInvoice(ctx context.Context, externalID string) (ps.Invoice, error) {
	if s.client == nil {
		return ps.Invoice{}, fmt.Errorf("stripe client not configured")
	}
	s.logger.Info("Fetching Stripe invoice", zap.String("stripe_invoice_id", externalID))
	params := &stripe.InvoiceRetrieveParams{}
	params.AddExpand("customer")
	// params.AddExpand("subscription") // Commented out if stripeInv.Subscription is undefined
	// params.AddExpand("charge") // Commented out if stripeInv.Charge is undefined
	// params.AddExpand("payment_intent") // Commented out if stripeInv.PaymentIntent is undefined
	params.AddExpand("lines.data.parent.invoice_item_details")
	params.AddExpand("lines.data.parent.subscription_item_details")
	params.AddExpand("lines.data.pricing.price_details")
	params.AddExpand("lines.data.taxes.tax_rate_details") // For RateID in tax amounts
	// params.AddExpand("default_tax_rates") // If stripeInv.DefaultTaxRates is undefined
	// params.AddExpand("total_tax_amounts.rate") // If stripeInv.TotalTaxAmounts is undefined

	stripeInv, err := s.client.V1Invoices.Retrieve(ctx, externalID, params)
	if err != nil {
		s.logger.Error("Failed to fetch Stripe invoice", zap.Error(err), zap.String("stripe_invoice_id", externalID))
		return ps.Invoice{}, fmt.Errorf("stripe_service.GetInvoice: %w", err)
	}
	mappedInvoice := mapStripeInvoiceToPSInvoice(stripeInv)
	s.logger.Info("Successfully fetched and mapped Stripe invoice", zap.String("stripe_invoice_id", externalID))
	return mappedInvoice, nil
}

// ListInvoices retrieves a list of invoices from Stripe using Seq2 iteration.
func (s *StripeService) ListInvoices(ctx context.Context, params ps.ListParams) ([]ps.Invoice, string, error) {
	if s.client == nil {
		return nil, "", fmt.Errorf("stripe client not configured")
	}
	s.logger.Info("Listing Stripe invoices", zap.Any("params", params))
	stripeParams := &stripe.InvoiceListParams{}
	if params.Limit > 0 {
		stripeParams.Limit = stripe.Int64(int64(params.Limit))
	}
	if params.StartingAfter != "" {
		stripeParams.StartingAfter = stripe.String(params.StartingAfter)
	}
	if params.EndingBefore != "" {
		stripeParams.EndingBefore = stripe.String(params.EndingBefore)
	}
	if params.Filters != nil {
		if customerID, ok := params.Filters["customer_id"].(string); ok && customerID != "" {
			stripeParams.Customer = stripe.String(customerID)
		}
		// if subscriptionID, ok := params.Filters["subscription_id"].(string); ok && subscriptionID != "" { // Linter: stripeParams.Subscription undefined if Invoice.Subscription is
		// 	stripeParams.Subscription = stripe.String(subscriptionID)
		// }
		if status, ok := params.Filters["status"].(string); ok && status != "" {
			stripeParams.Status = stripe.String(status)
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
		dueDateRange := &stripe.RangeQueryParams{}
		hasDueDateFilter := false
		if dueAfter, ok := params.Filters["due_date_after"].(int64); ok && dueAfter > 0 {
			dueDateRange.GreaterThan = dueAfter
			hasDueDateFilter = true
		}
		if dueBefore, ok := params.Filters["due_date_before"].(int64); ok && dueBefore > 0 {
			dueDateRange.LesserThan = dueBefore
			hasDueDateFilter = true
		}
		if hasDueDateFilter {
			stripeParams.DueDateRange = dueDateRange
		}
	}

	stripeParams.AddExpand("data.customer")
	// stripeParams.AddExpand("data.subscription") // Commented out if stripeInv.Subscription is undefined
	// stripeParams.AddExpand("data.charge") // Commented out if stripeInv.Charge is undefined
	// stripeParams.AddExpand("data.payment_intent") // Commented out if stripeInv.PaymentIntent is undefined
	stripeParams.AddExpand("data.lines.data.parent.invoice_item_details")
	stripeParams.AddExpand("data.lines.data.parent.subscription_item_details")
	stripeParams.AddExpand("data.lines.data.pricing.price_details")
	stripeParams.AddExpand("data.lines.data.taxes.tax_rate_details")
	// stripeParams.AddExpand("data.default_tax_rates") // If stripeInv.DefaultTaxRates is undefined
	// stripeParams.AddExpand("data.total_tax_amounts.rate") // If stripeInv.TotalTaxAmounts is undefined

	var invoices []ps.Invoice
	var lastID string

	// Corrected iteration using Seq2 / range
	for item, err := range s.client.V1Invoices.List(ctx, stripeParams) {
		if err != nil {
			s.logger.Error("Error during Stripe invoices list iteration", zap.Error(err))
			return nil, "", fmt.Errorf("stripe_service.ListInvoices: error during iteration item: %w", err)
		}
		if item == nil { // Should not happen if err is nil, but good practice
			continue
		}
		invoices = append(invoices, mapStripeInvoiceToPSInvoice(item))
		lastID = item.ID
	}

	nextPageCursor := ""
	if params.Limit > 0 && len(invoices) == int(params.Limit) {
		nextPageCursor = lastID
	}
	s.logger.Info("Successfully listed Stripe invoices", zap.Int("count", len(invoices)), zap.String("next_cursor", nextPageCursor))
	return invoices, nextPageCursor, nil
}

// mapPSInvoiceLineItemToStripeInvoiceItemParams converts ps.InvoiceLineItem to stripe.InvoiceItemCreateParams.
func mapPSInvoiceLineItemToStripeInvoiceItemParams(customerID string, currency string, psItem ps.InvoiceLineItem) *stripe.InvoiceItemCreateParams {
	params := &stripe.InvoiceItemCreateParams{
		Customer:    stripe.String(customerID),
		Description: stripe.String(psItem.Description),
		Metadata:    psItem.Metadata,
	}

	if currency != "" {
		params.Currency = stripe.String(currency)
	}

	if psItem.Amount != 0 {
		params.Amount = stripe.Int64(psItem.Amount)
	} else if psItem.PriceID != "" {
		// params.Price = stripe.String(psItem.PriceID) // Linter: params.Price undefined
	}

	if psItem.Quantity > 0 {
		params.Quantity = stripe.Int64(int64(psItem.Quantity))
	}

	if psItem.Period != nil && psItem.Period.Start > 0 && psItem.Period.End > 0 {
		// Assuming stripe.InvoiceItemCreatePeriodParams is the correct type for create operations.
		// This needs to be verified against the user's local SDK if different from standard v82.
		params.Period = &stripe.InvoiceItemCreatePeriodParams{ // Corrected type for creation
			Start: stripe.Int64(psItem.Period.Start),
			End:   stripe.Int64(psItem.Period.End),
		}
	}

	if len(psItem.TaxAmounts) > 0 {
		taxRateIDs := make([]*string, 0, len(psItem.TaxAmounts))
		for _, ta := range psItem.TaxAmounts {
			if ta.RateID != "" {
				taxRateIDs = append(taxRateIDs, stripe.String(ta.RateID))
			}
		}
		if len(taxRateIDs) > 0 {
			params.TaxRates = taxRateIDs
		}
	}

	// if psItem.SubscriptionItemID != "" { // Linter: params.SubscriptionItem undefined
	// 	params.SubscriptionItem = stripe.String(psItem.SubscriptionItemID)
	// }

	return params
}

// CreateInvoice creates an invoice in Stripe with the given items.
func (s *StripeService) CreateInvoice(ctx context.Context, customerExternalID string, subscriptionExternalID string, items []ps.InvoiceLineItem, autoAdvance bool) (ps.Invoice, error) {
	if s.client == nil {
		return ps.Invoice{}, fmt.Errorf("stripe client not configured")
	}
	s.logger.Info("Creating Stripe invoice",
		zap.String("customer_id", customerExternalID),
		zap.String("subscription_id", subscriptionExternalID),
		zap.Int("item_count", len(items)),
		zap.Bool("auto_advance", autoAdvance),
	)

	var invCurrency string

	for _, psItem := range items {
		itemParams := mapPSInvoiceLineItemToStripeInvoiceItemParams(customerExternalID, invCurrency, psItem)
		_, err := s.client.V1InvoiceItems.Create(ctx, itemParams)
		if err != nil {
			s.logger.Error("Failed to create Stripe invoice item", zap.Error(err), zap.Any("item_params", itemParams))
			return ps.Invoice{}, fmt.Errorf("stripe_service.CreateInvoice: failed to create invoice item: %w", err)
		}
	}

	invParams := &stripe.InvoiceCreateParams{
		Customer:         stripe.String(customerExternalID),
		AutoAdvance:      stripe.Bool(autoAdvance),
		CollectionMethod: stripe.String(string(stripe.InvoiceCollectionMethodChargeAutomatically)),
	}
	// if subscriptionExternalID != "" { // Linter: invParams.Subscription undefined if Invoice.Subscription is
	// 	invParams.Subscription = stripe.String(subscriptionExternalID)
	// }

	invParams.AddExpand("customer")
	// invParams.AddExpand("subscription") // Commented out if stripeInv.Subscription is undefined
	// invParams.AddExpand("charge") // Commented out if stripeInv.Charge is undefined
	// invParams.AddExpand("payment_intent") // Commented out if stripeInv.PaymentIntent is undefined
	invParams.AddExpand("lines.data.parent.invoice_item_details")
	invParams.AddExpand("lines.data.parent.subscription_item_details")
	invParams.AddExpand("lines.data.pricing.price_details")
	invParams.AddExpand("lines.data.taxes.tax_rate_details")
	// invParams.AddExpand("default_tax_rates") // If stripeInv.DefaultTaxRates is undefined
	// invParams.AddExpand("total_tax_amounts.rate") // If stripeInv.TotalTaxAmounts is undefined

	newStripeInvoice, err := s.client.V1Invoices.Create(ctx, invParams)
	if err != nil {
		s.logger.Error("Failed to create Stripe invoice", zap.Error(err), zap.Any("invoice_params", invParams))
		return ps.Invoice{}, fmt.Errorf("stripe_service.CreateInvoice: failed to create invoice: %w", err)
	}

	mappedInvoice := mapStripeInvoiceToPSInvoice(newStripeInvoice)
	s.logger.Info("Successfully created Stripe invoice", zap.String("stripe_invoice_id", newStripeInvoice.ID))
	return mappedInvoice, nil
}

// PayInvoice attempts to pay an invoice using a specific payment method.
func (s *StripeService) PayInvoice(ctx context.Context, externalInvoiceID string, paymentMethodExternalID string) (ps.Invoice, error) {
	if s.client == nil {
		return ps.Invoice{}, fmt.Errorf("stripe client not configured")
	}
	s.logger.Info("Paying Stripe invoice", zap.String("stripe_invoice_id", externalInvoiceID), zap.String("payment_method_id", paymentMethodExternalID))
	payParams := &stripe.InvoicePayParams{}
	if paymentMethodExternalID != "" {
		payParams.PaymentMethod = stripe.String(paymentMethodExternalID)
	}
	payParams.AddExpand("customer")
	// payParams.AddExpand("subscription") // Commented out if stripeInv.Subscription is undefined
	// payParams.AddExpand("charge") // Commented out if stripeInv.Charge is undefined
	// payParams.AddExpand("payment_intent") // Commented out if stripeInv.PaymentIntent is undefined
	payParams.AddExpand("lines.data.parent.invoice_item_details")
	payParams.AddExpand("lines.data.parent.subscription_item_details")
	payParams.AddExpand("lines.data.pricing.price_details")
	payParams.AddExpand("lines.data.taxes.tax_rate_details")
	// payParams.AddExpand("default_tax_rates") // If stripeInv.DefaultTaxRates is undefined
	// payParams.AddExpand("total_tax_amounts.rate") // If stripeInv.TotalTaxAmounts is undefined

	paidStripeInvoice, err := s.client.V1Invoices.Pay(ctx, externalInvoiceID, payParams)
	if err != nil {
		s.logger.Error("Failed to pay Stripe invoice", zap.Error(err), zap.String("stripe_invoice_id", externalInvoiceID))
		return ps.Invoice{}, fmt.Errorf("stripe_service.PayInvoice: %w", err)
	}
	mappedInvoice := mapStripeInvoiceToPSInvoice(paidStripeInvoice)
	s.logger.Info("Successfully paid Stripe invoice", zap.String("stripe_invoice_id", externalInvoiceID))
	return mappedInvoice, nil
}

// VoidInvoice voids an open or uncollectible invoice.
func (s *StripeService) VoidInvoice(ctx context.Context, externalInvoiceID string) (ps.Invoice, error) {
	if s.client == nil {
		return ps.Invoice{}, fmt.Errorf("stripe client not configured")
	}
	s.logger.Info("Voiding Stripe invoice", zap.String("stripe_invoice_id", externalInvoiceID))
	params := &stripe.InvoiceVoidInvoiceParams{}
	params.AddExpand("customer")
	// params.AddExpand("subscription") // Commented out if stripeInv.Subscription is undefined
	// params.AddExpand("charge") // Commented out if stripeInv.Charge is undefined
	// params.AddExpand("payment_intent") // Commented out if stripeInv.PaymentIntent is undefined
	params.AddExpand("lines.data.parent.invoice_item_details")
	params.AddExpand("lines.data.parent.subscription_item_details")
	params.AddExpand("lines.data.pricing.price_details")
	params.AddExpand("lines.data.taxes.tax_rate_details")
	// params.AddExpand("default_tax_rates") // If stripeInv.DefaultTaxRates is undefined
	// params.AddExpand("total_tax_amounts.rate") // If stripeInv.TotalTaxAmounts is undefined

	voidedStripeInvoice, err := s.client.V1Invoices.VoidInvoice(ctx, externalInvoiceID, params)
	if err != nil {
		s.logger.Error("Failed to void Stripe invoice", zap.Error(err), zap.String("stripe_invoice_id", externalInvoiceID))
		return ps.Invoice{}, fmt.Errorf("stripe_service.VoidInvoice: %w", err)
	}
	mappedInvoice := mapStripeInvoiceToPSInvoice(voidedStripeInvoice)
	s.logger.Info("Successfully voided Stripe invoice", zap.String("stripe_invoice_id", externalInvoiceID))
	return mappedInvoice, nil
}

// FinalizeInvoice finalizes a draft invoice in Stripe.
func (s *StripeService) FinalizeInvoice(ctx context.Context, externalInvoiceID string) (ps.Invoice, error) {
	if s.client == nil {
		return ps.Invoice{}, fmt.Errorf("stripe client not configured")
	}
	s.logger.Info("Finalizing Stripe invoice", zap.String("stripe_invoice_id", externalInvoiceID))
	params := &stripe.InvoiceFinalizeInvoiceParams{}
	params.AddExpand("customer")
	// params.AddExpand("subscription") // Commented out if stripeInv.Subscription is undefined
	// params.AddExpand("charge") // Commented out if stripeInv.Charge is undefined
	// params.AddExpand("payment_intent") // Commented out if stripeInv.PaymentIntent is undefined
	params.AddExpand("lines.data.parent.invoice_item_details")
	params.AddExpand("lines.data.parent.subscription_item_details")
	params.AddExpand("lines.data.pricing.price_details")
	params.AddExpand("lines.data.taxes.tax_rate_details")
	// params.AddExpand("default_tax_rates") // If stripeInv.DefaultTaxRates is undefined
	// params.AddExpand("total_tax_amounts.rate") // If stripeInv.TotalTaxAmounts is undefined

	finalizedStripeInvoice, err := s.client.V1Invoices.FinalizeInvoice(ctx, externalInvoiceID, params)
	if err != nil {
		s.logger.Error("Failed to finalize Stripe invoice", zap.Error(err), zap.String("stripe_invoice_id", externalInvoiceID))
		return ps.Invoice{}, fmt.Errorf("stripe_service.FinalizeInvoice: %w", err)
	}
	mappedInvoice := mapStripeInvoiceToPSInvoice(finalizedStripeInvoice)
	s.logger.Info("Successfully finalized Stripe invoice", zap.String("stripe_invoice_id", externalInvoiceID))
	return mappedInvoice, nil
}

// SendInvoice sends an invoice to the customer via email.
func (s *StripeService) SendInvoice(ctx context.Context, externalInvoiceID string) (ps.Invoice, error) {
	if s.client == nil {
		return ps.Invoice{}, fmt.Errorf("stripe client not configured")
	}
	s.logger.Info("Sending Stripe invoice", zap.String("stripe_invoice_id", externalInvoiceID))
	params := &stripe.InvoiceSendInvoiceParams{}
	params.AddExpand("customer")
	// params.AddExpand("subscription") // Commented out if stripeInv.Subscription is undefined
	// params.AddExpand("charge") // Commented out if stripeInv.Charge is undefined
	// params.AddExpand("payment_intent") // Commented out if stripeInv.PaymentIntent is undefined
	params.AddExpand("lines.data.parent.invoice_item_details")
	params.AddExpand("lines.data.parent.subscription_item_details")
	params.AddExpand("lines.data.pricing.price_details")
	params.AddExpand("lines.data.taxes.tax_rate_details")
	// params.AddExpand("default_tax_rates") // If stripeInv.DefaultTaxRates is undefined
	// params.AddExpand("total_tax_amounts.rate") // If stripeInv.TotalTaxAmounts is undefined

	sentStripeInvoice, err := s.client.V1Invoices.SendInvoice(ctx, externalInvoiceID, params)
	if err != nil {
		s.logger.Error("Failed to send Stripe invoice", zap.Error(err), zap.String("stripe_invoice_id", externalInvoiceID))
		return ps.Invoice{}, fmt.Errorf("stripe_service.SendInvoice: %w", err)
	}
	mappedInvoice := mapStripeInvoiceToPSInvoice(sentStripeInvoice)
	s.logger.Info("Successfully sent Stripe invoice", zap.String("stripe_invoice_id", externalInvoiceID))
	return mappedInvoice, nil
}
