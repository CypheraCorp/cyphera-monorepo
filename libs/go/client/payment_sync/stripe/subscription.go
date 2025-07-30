package stripe

import (
	"context"
	"fmt"

	ps "github.com/cyphera/cyphera-api/libs/go/client/payment_sync"

	"github.com/stripe/stripe-go/v82"
	"go.uber.org/zap"
)

// mapStripeSubscriptionItemToPSSubscriptionItem converts a Stripe SubscriptionItem to ps.SubscriptionItem.
func mapStripeSubscriptionItemToPSSubscriptionItem(stripeItem *stripe.SubscriptionItem) ps.SubscriptionItem {
	if stripeItem == nil {
		return ps.SubscriptionItem{}
	}

	var taxRateIDs []string
	if len(stripeItem.TaxRates) > 0 {
		taxRateIDs = make([]string, len(stripeItem.TaxRates))
		for i, tr := range stripeItem.TaxRates {
			if tr != nil {
				taxRateIDs[i] = tr.ID
			}
		}
	}

	return ps.SubscriptionItem{
		ExternalID: stripeItem.ID,
		PriceID:    stripeItem.Price.ID, // Assuming Price is at least an object with ID
		Quantity:   int(stripeItem.Quantity),
		Metadata:   stripeItem.Metadata,
		TaxRateIDs: taxRateIDs,
		// ps.SubscriptionItem.ID would be Cyphera's internal ID if needed
	}
}

// mapStripeSubscriptionToPSSubscription converts a Stripe Subscription object to the canonical ps.Subscription.
func mapStripeSubscriptionToPSSubscription(stripeSub *stripe.Subscription) ps.Subscription {
	if stripeSub == nil {
		return ps.Subscription{}
	}

	var items []ps.SubscriptionItem
	var primaryCurrentPeriodStart int64
	var primaryCurrentPeriodEnd int64

	if len(stripeSub.Items.Data) > 0 {
		items = make([]ps.SubscriptionItem, len(stripeSub.Items.Data))
		for i, item := range stripeSub.Items.Data {
			items[i] = mapStripeSubscriptionItemToPSSubscriptionItem(item)
		}
		if stripeSub.Items.Data[0] != nil {
			primaryCurrentPeriodStart = stripeSub.Items.Data[0].CurrentPeriodStart
			primaryCurrentPeriodEnd = stripeSub.Items.Data[0].CurrentPeriodEnd
		}
	}

	var latestInvoiceID string
	if stripeSub.LatestInvoice != nil {
		latestInvoiceID = stripeSub.LatestInvoice.ID
	}

	var defaultPaymentMethodID string
	if stripeSub.DefaultPaymentMethod != nil {
		defaultPaymentMethodID = stripeSub.DefaultPaymentMethod.ID
	}

	var defaultTaxRateIDs []string
	if len(stripeSub.DefaultTaxRates) > 0 {
		defaultTaxRateIDs = make([]string, len(stripeSub.DefaultTaxRates))
		for i, tr := range stripeSub.DefaultTaxRates {
			if tr != nil {
				defaultTaxRateIDs[i] = tr.ID
			}
		}
	}

	return ps.Subscription{
		ExternalID:             stripeSub.ID,
		CustomerID:             stripeSub.Customer.ID,
		Status:                 string(stripeSub.Status),
		CurrentPeriodStart:     primaryCurrentPeriodStart,
		CurrentPeriodEnd:       primaryCurrentPeriodEnd,
		TrialStartDate:         stripeSub.TrialStart,
		TrialEndDate:           stripeSub.TrialEnd,
		CancelAtPeriodEnd:      stripeSub.CancelAtPeriodEnd,
		CanceledAt:             stripeSub.CanceledAt,
		EndedAt:                stripeSub.EndedAt,
		DefaultPaymentMethodID: defaultPaymentMethodID,
		Items:                  items,
		Metadata:               stripeSub.Metadata,
		LatestInvoiceID:        latestInvoiceID,
		BillingCycleAnchor:     stripeSub.BillingCycleAnchor,
		CollectionMethod:       string(stripeSub.CollectionMethod),
		DefaultTaxRateIDs:      defaultTaxRateIDs,
		// ps.Subscription.ID would be Cyphera's internal ID
	}
}

// CreateSubscription creates a new subscription in Stripe.
func (s *StripeService) CreateSubscription(ctx context.Context, subData ps.Subscription) (ps.Subscription, error) {
	if s.client == nil {
		return ps.Subscription{}, fmt.Errorf("stripe client not configured")
	}

	if len(subData.Items) == 0 {
		return ps.Subscription{}, fmt.Errorf("subscription must have at least one item")
	}

	stripeItems := make([]*stripe.SubscriptionCreateItemParams, len(subData.Items))
	for i, item := range subData.Items {
		itemP := &stripe.SubscriptionCreateItemParams{
			Price:    stripe.String(item.PriceID),
			Quantity: stripe.Int64(int64(item.Quantity)),
			Metadata: item.Metadata,
		}
		if len(item.TaxRateIDs) > 0 {
			itemP.TaxRates = stripe.StringSlice(item.TaxRateIDs)
		}
		stripeItems[i] = itemP
	}

	params := &stripe.SubscriptionCreateParams{
		Customer: stripe.String(subData.CustomerID),
		Items:    stripeItems,
		Metadata: subData.Metadata,
	}

	if subData.TrialEndDate > 0 {
		params.TrialEnd = stripe.Int64(subData.TrialEndDate)
	}
	if subData.DefaultPaymentMethodID != "" {
		params.DefaultPaymentMethod = stripe.String(subData.DefaultPaymentMethodID)
	}
	if subData.CancelAtPeriodEnd {
		params.CancelAtPeriodEnd = stripe.Bool(subData.CancelAtPeriodEnd)
	}
	if subData.BillingCycleAnchor > 0 {
		params.BillingCycleAnchor = stripe.Int64(subData.BillingCycleAnchor)
	}
	if subData.CollectionMethod != "" {
		params.CollectionMethod = stripe.String(string(stripe.SubscriptionCollectionMethod(subData.CollectionMethod)))
	}
	if len(subData.DefaultTaxRateIDs) > 0 {
		params.DefaultTaxRates = stripe.StringSlice(subData.DefaultTaxRateIDs)
	}

	params.AddExpand("latest_invoice")
	params.AddExpand("default_payment_method")
	params.AddExpand("default_tax_rates")
	params.AddExpand("items.data.tax_rates")

	s.logger.Info("Creating Stripe subscription", zap.String("customer_id", subData.CustomerID), zap.Int("item_count", len(stripeItems)))

	newStripeSub, err := s.client.V1Subscriptions.Create(ctx, params)
	if err != nil {
		s.logger.Error("Failed to create Stripe subscription", zap.Error(err), zap.Any("params", params))
		return ps.Subscription{}, fmt.Errorf("stripe_service.CreateSubscription: %w", err)
	}

	mappedSub := mapStripeSubscriptionToPSSubscription(newStripeSub)
	s.logger.Info("Successfully created Stripe subscription", zap.String("stripe_subscription_id", newStripeSub.ID))
	return mappedSub, nil
}

// GetSubscription retrieves a subscription by its external ID from Stripe.
func (s *StripeService) GetSubscription(ctx context.Context, externalID string) (ps.Subscription, error) {
	if s.client == nil {
		return ps.Subscription{}, fmt.Errorf("stripe client not configured")
	}

	s.logger.Info("Fetching Stripe subscription", zap.String("stripe_subscription_id", externalID))

	params := &stripe.SubscriptionRetrieveParams{}
	params.AddExpand("latest_invoice")
	params.AddExpand("default_payment_method")
	params.AddExpand("default_tax_rates")
	params.AddExpand("items.data.tax_rates")

	stripeSub, err := s.client.V1Subscriptions.Retrieve(ctx, externalID, params)
	if err != nil {
		s.logger.Error("Failed to fetch Stripe subscription", zap.Error(err), zap.String("stripe_subscription_id", externalID))
		return ps.Subscription{}, fmt.Errorf("stripe_service.GetSubscription: %w", err)
	}

	mappedSub := mapStripeSubscriptionToPSSubscription(stripeSub)
	s.logger.Info("Successfully fetched and mapped Stripe subscription", zap.String("stripe_subscription_id", externalID))

	return mappedSub, nil
}

// UpdateSubscription updates an existing subscription in Stripe.
// Note: Updating subscription items can be complex (adding, removing, changing quantities).
// This initial implementation will focus on replacing items if provided, and updating top-level fields.
func (s *StripeService) UpdateSubscription(
	ctx context.Context,
	externalID string,
	itemsToUpdate []ps.SubscriptionItem, // ps.SubscriptionItem here should have PriceID and Quantity for new/updated items.
	// If updating an existing item, its ps.SubscriptionItem.ExternalID should be the Stripe Subscription Item ID.
	prorationBehavior string,
	metadata map[string]string,
	otherUpdateFields map[string]interface{}, // Basic support for CancelAtPeriodEnd, DefaultPaymentMethodID from this map for now.
) (ps.Subscription, error) {
	if s.client == nil {
		return ps.Subscription{}, fmt.Errorf("stripe client not configured")
	}

	params := &stripe.SubscriptionUpdateParams{}

	if len(itemsToUpdate) > 0 {
		stripeItems := make([]*stripe.SubscriptionUpdateItemParams, len(itemsToUpdate))
		for i, item := range itemsToUpdate {
			updateItem := &stripe.SubscriptionUpdateItemParams{
				Metadata: item.Metadata,
			}

			if item.ExternalID != "" {
				updateItem.ID = stripe.String(item.ExternalID)
				if item.Quantity > 0 {
					updateItem.Quantity = stripe.Int64(int64(item.Quantity))
				}
			} else {
				updateItem.Price = stripe.String(item.PriceID)
				updateItem.Quantity = stripe.Int64(int64(item.Quantity))
			}
			if len(item.TaxRateIDs) > 0 {
				updateItem.TaxRates = stripe.StringSlice(item.TaxRateIDs)
			}
			stripeItems[i] = updateItem
		}
		params.Items = stripeItems
	}

	if prorationBehavior != "" {
		params.ProrationBehavior = stripe.String(prorationBehavior)
	}

	if metadata != nil {
		params.Metadata = metadata
	}

	if val, ok := otherUpdateFields["cancel_at_period_end"]; ok {
		if cancel, okBool := val.(bool); okBool {
			params.CancelAtPeriodEnd = stripe.Bool(cancel)
		}
	}
	if val, ok := otherUpdateFields["default_payment_method"]; ok {
		if pmID, okStr := val.(string); okStr && pmID != "" {
			params.DefaultPaymentMethod = stripe.String(pmID)
		}
	}
	if val, ok := otherUpdateFields["collection_method"]; ok {
		if cm, okStr := val.(string); okStr && cm != "" {
			params.CollectionMethod = stripe.String(string(stripe.SubscriptionCollectionMethod(cm)))
		}
	}
	if val, ok := otherUpdateFields["billing_cycle_anchor_now"]; ok {
		if bcaNow, okBool := val.(bool); okBool && bcaNow {
			params.BillingCycleAnchorNow = stripe.Bool(true)
		}
	}
	if val, ok := otherUpdateFields["default_tax_rate_ids"]; ok {
		if dtrIDs, okSlice := val.([]string); okSlice && len(dtrIDs) > 0 {
			params.DefaultTaxRates = stripe.StringSlice(dtrIDs)
		}
	}

	params.AddExpand("latest_invoice")
	params.AddExpand("default_payment_method")
	params.AddExpand("default_tax_rates")
	params.AddExpand("items.data.tax_rates")

	s.logger.Info("Updating Stripe subscription", zap.String("stripe_subscription_id", externalID), zap.Any("params", params))

	updatedStripeSub, err := s.client.V1Subscriptions.Update(ctx, externalID, params)
	if err != nil {
		s.logger.Error("Failed to update Stripe subscription", zap.Error(err), zap.String("stripe_subscription_id", externalID))
		return ps.Subscription{}, fmt.Errorf("stripe_service.UpdateSubscription: %w", err)
	}

	mappedSub := mapStripeSubscriptionToPSSubscription(updatedStripeSub)
	s.logger.Info("Successfully updated and mapped Stripe subscription", zap.String("stripe_subscription_id", externalID))

	return mappedSub, nil
}

// CancelSubscription cancels a subscription in Stripe, either immediately or at period end.
func (s *StripeService) CancelSubscription(ctx context.Context, externalID string, cancelAtPeriodEnd bool, invoiceNow bool) (ps.Subscription, error) {
	if s.client == nil {
		return ps.Subscription{}, fmt.Errorf("stripe client not configured")
	}

	var canceledStripeSub *stripe.Subscription
	var err error

	params := &stripe.SubscriptionCancelParams{}
	if invoiceNow {
		params.InvoiceNow = stripe.Bool(true)
	}
	// Prorate could also be a parameter if needed: params.Prorate = stripe.Bool(true)

	// Always expand latest_invoice and default_payment_method for consistent mapping
	// For Update operation (cancelAtPeriodEnd=true)
	updateParams := &stripe.SubscriptionUpdateParams{}
	updateParams.AddExpand("latest_invoice")
	updateParams.AddExpand("default_payment_method")
	updateParams.AddExpand("default_tax_rates")
	updateParams.AddExpand("items.data.tax_rates")

	// For Cancel operation (cancelAtPeriodEnd=false)
	// The Cancel method itself doesn't have AddExpand directly on its params struct in the same way Create/Update often do.
	// The returned subscription object from Cancel should have its fields populated.
	// If specific fields are missing, a subsequent Get might be needed, or check if CancelParams supports expansion.
	// For now, assume returned object from Cancel is sufficient or use a Get if mapping is sparse.

	if cancelAtPeriodEnd {
		s.logger.Info("Updating Stripe subscription to cancel at period end", zap.String("stripe_subscription_id", externalID))
		updateParams.CancelAtPeriodEnd = stripe.Bool(true)
		canceledStripeSub, err = s.client.V1Subscriptions.Update(ctx, externalID, updateParams)
	} else {
		s.logger.Info("Canceling Stripe subscription immediately", zap.String("stripe_subscription_id", externalID), zap.Bool("invoice_now", invoiceNow))
		canceledStripeSub, err = s.client.V1Subscriptions.Cancel(ctx, externalID, params)
	}

	if err != nil {
		s.logger.Error("Failed to cancel Stripe subscription", zap.Error(err), zap.String("stripe_subscription_id", externalID))
		return ps.Subscription{}, fmt.Errorf("stripe_service.CancelSubscription: %w", err)
	}

	mappedSub := mapStripeSubscriptionToPSSubscription(canceledStripeSub)
	s.logger.Info("Successfully canceled Stripe subscription", zap.String("stripe_subscription_id", externalID), zap.Bool("at_period_end", cancelAtPeriodEnd))

	return mappedSub, nil
}

// ListSubscriptions retrieves a list of subscriptions from Stripe.
// It uses ps.ListParams and maps relevant fields and filters to Stripe's SubscriptionListParams.
func (s *StripeService) ListSubscriptions(ctx context.Context, params ps.ListParams) ([]ps.Subscription, string, error) {
	if s.client == nil {
		return nil, "", fmt.Errorf("stripe client not configured")
	}

	s.logger.Info("Listing Stripe subscriptions", zap.Any("params", params))

	stripeParams := &stripe.SubscriptionListParams{}

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

	// Handle specific filters from params.Filters
	if params.Filters != nil {
		if customerID, ok := params.Filters["customer_id"].(string); ok && customerID != "" {
			stripeParams.Customer = stripe.String(customerID)
		}
		if priceID, ok := params.Filters["price_id"].(string); ok && priceID != "" {
			stripeParams.Price = stripe.String(priceID)
		}
		if status, ok := params.Filters["status"].(string); ok && status != "" {
			stripeParams.Status = stripe.String(status)
		}
		if collectionMethod, ok := params.Filters["collection_method"].(string); ok && collectionMethod != "" {
			stripeParams.CollectionMethod = stripe.String(collectionMethod)
		}

		currentPeriodStartRange := &stripe.RangeQueryParams{}
		hasCurrentPeriodStartFilter := false
		if startAfter, ok := params.Filters["current_period_start_after"].(int64); ok && startAfter > 0 {
			currentPeriodStartRange.GreaterThan = startAfter
			hasCurrentPeriodStartFilter = true
		}
		if startBefore, ok := params.Filters["current_period_start_before"].(int64); ok && startBefore > 0 {
			currentPeriodStartRange.LesserThan = startBefore
			hasCurrentPeriodStartFilter = true
		}
		if hasCurrentPeriodStartFilter {
			stripeParams.CurrentPeriodStartRange = currentPeriodStartRange
		}

		currentPeriodEndRange := &stripe.RangeQueryParams{}
		hasCurrentPeriodEndFilter := false
		if endAfter, ok := params.Filters["current_period_end_after"].(int64); ok && endAfter > 0 {
			currentPeriodEndRange.GreaterThan = endAfter
			hasCurrentPeriodEndFilter = true
		}
		if endBefore, ok := params.Filters["current_period_end_before"].(int64); ok && endBefore > 0 {
			currentPeriodEndRange.LesserThan = endBefore
			hasCurrentPeriodEndFilter = true
		}
		if hasCurrentPeriodEndFilter {
			stripeParams.CurrentPeriodEndRange = currentPeriodEndRange
		}
	}

	stripeParams.AddExpand("latest_invoice")
	stripeParams.AddExpand("default_payment_method")
	stripeParams.AddExpand("default_tax_rates")
	stripeParams.AddExpand("items.data.tax_rates")
	// Add other expansions if commonly needed for mapStripeSubscriptionToPSSubscription

	var subscriptions []ps.Subscription
	var lastID string

	// Use the new iter.Seq2 pattern for listing.
	for stripeSub, err := range s.client.V1Subscriptions.List(ctx, stripeParams) {
		if err != nil {
			s.logger.Error("Error iterating Stripe subscriptions list", zap.Error(err))
			// For Seq2, errors are per-item. We can choose to stop or collect errors.
			// For now, we stop on the first error encountered during iteration.
			return nil, "", fmt.Errorf("stripe_service.ListSubscriptions: error during iteration: %w", err)
		}
		if stripeSub == nil { // Should not happen if err is nil, but good practice
			continue
		}
		subscriptions = append(subscriptions, mapStripeSubscriptionToPSSubscription(stripeSub))
		lastID = stripeSub.ID
	}

	// No separate iter.Err() check is typically needed or available from Seq2 directly in this pattern.

	nextPageCursor := ""
	if params.Limit > 0 && len(subscriptions) == params.Limit {
		nextPageCursor = lastID
	}

	s.logger.Info("Successfully listed Stripe subscriptions", zap.Int("count", len(subscriptions)), zap.String("next_cursor", nextPageCursor))
	return subscriptions, nextPageCursor, nil
}

// ReactivateSubscription reactivates a subscription in Stripe that was previously canceled with cancel_at_period_end = true.
// It does this by setting cancel_at_period_end to false.
func (s *StripeService) ReactivateSubscription(ctx context.Context, externalID string) (ps.Subscription, error) {
	if s.client == nil {
		return ps.Subscription{}, fmt.Errorf("stripe client not configured")
	}

	s.logger.Info("Reactivating Stripe subscription", zap.String("stripe_subscription_id", externalID))

	updateParams := &stripe.SubscriptionUpdateParams{
		CancelAtPeriodEnd: stripe.Bool(false),
	}

	// Add expansions to ensure the returned object can be mapped comprehensively
	updateParams.AddExpand("latest_invoice")
	updateParams.AddExpand("default_payment_method")
	updateParams.AddExpand("default_tax_rates")
	updateParams.AddExpand("items.data.tax_rates")
	// Consider other expansions if mapStripeSubscriptionToPSSubscription needs them

	updatedStripeSub, err := s.client.V1Subscriptions.Update(ctx, externalID, updateParams)
	if err != nil {
		s.logger.Error("Failed to reactivate Stripe subscription", zap.Error(err), zap.String("stripe_subscription_id", externalID))
		return ps.Subscription{}, fmt.Errorf("stripe_service.ReactivateSubscription: failed to update subscription %s: %w", externalID, err)
	}

	mappedSub := mapStripeSubscriptionToPSSubscription(updatedStripeSub)
	s.logger.Info("Successfully reactivated Stripe subscription", zap.String("stripe_subscription_id", externalID))

	return mappedSub, nil
}
