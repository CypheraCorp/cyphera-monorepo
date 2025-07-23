package stripe

import (
	"context"
	"fmt"

	ps "github.com/cyphera/cyphera-api/libs/go/client/payment_sync"

	"github.com/stripe/stripe-go/v82"
	"go.uber.org/zap"
)

// mapStripePriceToPSPrice converts a Stripe Price object to the canonical ps.Price.
func mapStripePriceToPSPrice(stripePrice *stripe.Price) ps.Price {
	if stripePrice == nil {
		return ps.Price{}
	}

	var recurring *ps.RecurringInterval
	if stripePrice.Recurring != nil {
		recurring = &ps.RecurringInterval{
			Interval:      string(stripePrice.Recurring.Interval),
			IntervalCount: int(stripePrice.Recurring.IntervalCount),
		}
	}

	var psTiers []ps.PriceTier
	if len(stripePrice.Tiers) > 0 {
		psTiers = make([]ps.PriceTier, len(stripePrice.Tiers))
		for i, t := range stripePrice.Tiers {
			if t != nil { // Stripe tiers are pointers
				psTiers[i] = ps.PriceTier{
					UpTo:       t.UpTo,
					UnitAmount: t.UnitAmount,
					FlatAmount: t.FlatAmount,
				}
			}
		}
	}

	var psTransformQuantity *ps.TransformQuantity
	if stripePrice.TransformQuantity != nil {
		psTransformQuantity = &ps.TransformQuantity{
			DivideBy: stripePrice.TransformQuantity.DivideBy,
			Round:    string(stripePrice.TransformQuantity.Round),
		}
	}

	return ps.Price{
		ExternalID:            stripePrice.ID,
		ProductID:             stripePrice.Product.ID, // Assuming Product is expanded or just ID is needed
		Active:                stripePrice.Active,
		Amount:                stripePrice.UnitAmount, // UnitAmount is in the smallest currency unit
		Currency:              string(stripePrice.Currency),
		Recurring:             recurring,
		BillingScheme:         string(stripePrice.BillingScheme),
		Type:                  string(stripePrice.Type),
		Metadata:              stripePrice.Metadata,
		TaxBehavior:           string(stripePrice.TaxBehavior),
		Tiers:                 psTiers,
		TiersMode:             string(stripePrice.TiersMode),
		TransformQuantityData: psTransformQuantity,
		// ps.Price.ID (Cyphera's internal ID) would be populated elsewhere.
	}
}

// CreatePrice creates a new price in Stripe using the new stripe.Client API.
func (s *StripeService) CreatePrice(ctx context.Context, priceData ps.Price) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("stripe client not configured")
	}

	params := &stripe.PriceCreateParams{
		Product:    stripe.String(priceData.ProductID), // This should be the Stripe Product ID
		UnitAmount: stripe.Int64(priceData.Amount),     // Only if BillingScheme is per_unit or not tiered
		Currency:   stripe.String(string(stripe.Currency(priceData.Currency))),
		Active:     stripe.Bool(priceData.Active),
		Metadata:   priceData.Metadata,
	}

	// Handle BillingScheme, default to per_unit if not specified by `priceData` but Amount is present.
	// If Tiers are present, UnitAmount should generally not be set at the top level (unless it's a base fee for some models).
	if priceData.BillingScheme != "" {
		params.BillingScheme = stripe.String(string(stripe.PriceBillingScheme(priceData.BillingScheme)))
	}

	if priceData.Recurring != nil {
		params.Recurring = &stripe.PriceCreateRecurringParams{
			Interval:      stripe.String(string(stripe.PriceRecurringInterval(priceData.Recurring.Interval))),
			IntervalCount: stripe.Int64(int64(priceData.Recurring.IntervalCount)),
		}
	}

	if priceData.TaxBehavior != "" {
		params.TaxBehavior = stripe.String(string(stripe.PriceTaxBehavior(priceData.TaxBehavior)))
	}

	if len(priceData.Tiers) > 0 {
		stripeTiers := make([]*stripe.PriceCreateTierParams, len(priceData.Tiers))
		for i, t := range priceData.Tiers {
			stripeTiers[i] = &stripe.PriceCreateTierParams{
				UpTo:       stripe.Int64(t.UpTo),
				UnitAmount: stripe.Int64(t.UnitAmount),
				FlatAmount: stripe.Int64(t.FlatAmount),
			}
		}
		params.Tiers = stripeTiers
		// If tiers are used, UnitAmount on PriceCreateParams should typically be nil or represent a base fee.
		// The SDK/API might enforce this. For safety, if tiers are present, let's nullify top-level UnitAmount unless billing scheme implies otherwise.
		if params.BillingScheme == nil || stripe.StringValue(params.BillingScheme) == string(stripe.PriceBillingSchemeTiered) {
			params.UnitAmount = nil // Stripe typically expects this to be nil if tiers are defined, amount is in tiers.
		}

		if priceData.TiersMode != "" {
			params.TiersMode = stripe.String(string(stripe.PriceTiersMode(priceData.TiersMode)))
		}
	}

	if priceData.TransformQuantityData != nil {
		params.TransformQuantity = &stripe.PriceCreateTransformQuantityParams{
			DivideBy: stripe.Int64(priceData.TransformQuantityData.DivideBy),
			Round:    stripe.String(string(stripe.PriceTransformQuantityRound(priceData.TransformQuantityData.Round))),
		}
	}

	// If not recurring, it implies a one_time price. Type might need to be explicitly set if not inferred.
	// However, PriceCreateParams usually doesn't take 'Type'. It's often inferred from 'Recurring' presence.
	// If priceData.Type is explicitly "one_time", ensure this aligns with Stripe's expectation when Recurring is nil.
	// The ps.Price.Type field exists, let's see if Stripe's PriceCreateParams uses it.
	// Stripe infers price type from `Recurring` param. If `Recurring` is absent, it's `one_time`.
	// If `Recurring` is present, it's `recurring`.
	// The `Type` field on `PriceCreateParams` is typically not directly set by users.

	s.logger.Info("Creating Stripe price", zap.String("product_id", priceData.ProductID), zap.Int64("amount", priceData.Amount))

	newStripePrice, err := s.client.V1Prices.Create(ctx, params)
	if err != nil {
		s.logger.Error("Failed to create Stripe price", zap.Error(err), zap.Any("params", params))
		return "", fmt.Errorf("stripe_service.CreatePrice: failed to create price: %w", err)
	}

	s.logger.Info("Successfully created Stripe price", zap.String("stripe_price_id", newStripePrice.ID))
	return newStripePrice.ID, nil
}

// GetPrice retrieves a price by its external ID from Stripe using the new stripe.Client API.
func (s *StripeService) GetPrice(ctx context.Context, externalID string) (ps.Price, error) {
	if s.client == nil {
		return ps.Price{}, fmt.Errorf("stripe client not configured")
	}

	s.logger.Info("Fetching Stripe price", zap.String("stripe_price_id", externalID))

	params := &stripe.PriceRetrieveParams{}

	stripePrice, err := s.client.V1Prices.Retrieve(ctx, externalID, params)
	if err != nil {
		s.logger.Error("Failed to fetch Stripe price", zap.Error(err), zap.String("stripe_price_id", externalID))
		return ps.Price{}, fmt.Errorf("stripe_service.GetPrice: failed to fetch price %s: %w", externalID, err)
	}

	mappedPrice := mapStripePriceToPSPrice(stripePrice)
	s.logger.Info("Successfully fetched and mapped Stripe price", zap.String("stripe_price_id", externalID))

	return mappedPrice, nil
}

// UpdatePrice updates an existing price in Stripe using the new stripe.Client API.
// Typically, only fields like active status and metadata are updatable for a price.
// Amount, currency, recurring details, product are generally immutable.
func (s *StripeService) UpdatePrice(ctx context.Context, externalID string, priceData ps.Price) (ps.Price, error) {
	if s.client == nil {
		return ps.Price{}, fmt.Errorf("stripe client not configured")
	}

	params := &stripe.PriceUpdateParams{}

	// Only set fields that are typically updatable for a price.
	// Active status can usually be updated.
	params.Active = stripe.Bool(priceData.Active) // ps.Price.Active is bool, so always set.

	if priceData.Metadata != nil {
		params.Metadata = priceData.Metadata
	}

	// TaxBehavior is updatable
	if priceData.TaxBehavior != "" {
		params.TaxBehavior = stripe.String(string(stripe.PriceTaxBehavior(priceData.TaxBehavior)))
	}

	// Other fields like UnitAmount, Currency, Recurring, Product are generally not updatable.
	// If specific fields are found to be updatable, they can be added here with checks.

	s.logger.Info("Updating Stripe price", zap.String("stripe_price_id", externalID), zap.Any("update_params", params))

	updatedStripePrice, err := s.client.V1Prices.Update(ctx, externalID, params)
	if err != nil {
		s.logger.Error("Failed to update Stripe price", zap.Error(err), zap.String("stripe_price_id", externalID))
		return ps.Price{}, fmt.Errorf("stripe_service.UpdatePrice: failed to update price %s: %w", externalID, err)
	}

	mappedPrice := mapStripePriceToPSPrice(updatedStripePrice)
	s.logger.Info("Successfully updated and mapped Stripe price", zap.String("stripe_price_id", externalID))

	return mappedPrice, nil
}

// ListPrices retrieves a list of prices from Stripe using the new stripe.Client API.
// It can be filtered by productExternalID (optional) and other params from ps.ListParams.
func (s *StripeService) ListPrices(ctx context.Context, productExternalID string, params ps.ListParams) ([]ps.Price, string, error) {
	if s.client == nil {
		return nil, "", fmt.Errorf("stripe client not configured")
	}

	s.logger.Info("Listing Stripe prices", zap.String("product_id_filter", productExternalID), zap.Any("params", params))

	stripeParams := &stripe.PriceListParams{}

	if productExternalID != "" {
		stripeParams.Product = stripe.String(productExternalID)
	}

	if params.Limit > 0 {
		stripeParams.Limit = stripe.Int64(int64(params.Limit))
	}
	if params.StartingAfter != "" {
		stripeParams.StartingAfter = stripe.String(params.StartingAfter)
	}
	if params.EndingBefore != "" {
		stripeParams.EndingBefore = stripe.String(params.EndingBefore)
	}

	// Handle Active filter from ps.ListParams.Filters or a direct field if we add it to ps.ListParams for prices.
	if activeFilter, ok := params.Filters["active"]; ok {
		if activeBool, ok := activeFilter.(bool); ok {
			stripeParams.Active = stripe.Bool(activeBool)
		}
	}

	// Handle Currency filter
	if currencyFilter, ok := params.Filters["currency"].(string); ok && currencyFilter != "" {
		stripeParams.Currency = stripe.String(string(stripe.Currency(currencyFilter)))
	}

	// Handle Type filter (e.g., "one_time", "recurring")
	if typeFilter, ok := params.Filters["type"].(string); ok && typeFilter != "" {
		stripeParams.Type = stripe.String(string(stripe.PriceType(typeFilter)))
	}

	// Note: Stripe's PriceListParams also supports `Recurring` directly for more granular filtering on recurring prices.
	// e.g. stripeParams.Recurring = &stripe.PriceListRecurringParams{ Interval: stripe.String("month") }
	// This could be added if ps.ListParams.Filters needs to support it.

	var prices []ps.Price
	var lastID string

	for stripePrice, err := range s.client.V1Prices.List(ctx, stripeParams) {
		if err != nil {
			s.logger.Error("Error iterating Stripe prices list", zap.Error(err))
			return nil, "", fmt.Errorf("stripe_service.ListPrices: error during iteration: %w", err)
		}
		if stripePrice == nil {
			continue
		}
		prices = append(prices, mapStripePriceToPSPrice(stripePrice))
		lastID = stripePrice.ID
	}

	nextPageCursor := ""
	if params.Limit > 0 && len(prices) == params.Limit {
		nextPageCursor = lastID
	}

	s.logger.Info("Successfully listed Stripe prices", zap.Int("count", len(prices)), zap.String("next_cursor", nextPageCursor))
	return prices, nextPageCursor, nil
}
