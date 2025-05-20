package stripe

import (
	"context"
	"fmt"

	ps "cyphera-api/internal/client/payment_sync"

	"github.com/stripe/stripe-go/v82"
	"go.uber.org/zap"
)

// mapStripeProductToPSProduct converts a Stripe Product object to the canonical ps.Product.
func mapStripeProductToPSProduct(stripeProd *stripe.Product) ps.Product {
	if stripeProd == nil {
		return ps.Product{}
	}

	var taxCodeID string
	if stripeProd.TaxCode != nil {
		taxCodeID = stripeProd.TaxCode.ID
	}

	return ps.Product{
		ExternalID:  stripeProd.ID,
		Name:        stripeProd.Name,
		Description: stripeProd.Description,
		Active:      stripeProd.Active,
		Type:        string(stripeProd.Type), // Stripe product type (service or good)
		Metadata:    stripeProd.Metadata,
		Shippable:   stripeProd.Shippable,
		UnitLabel:   stripeProd.UnitLabel,
		TaxCode:     taxCodeID,
		// ps.Product.ID (Cyphera's internal ID) would be populated elsewhere.
	}
}

// CreateProduct creates a new product in Stripe using the new stripe.Client API.
// It maps the canonical ps.Product to stripe.ProductCreateParams, calls the Stripe API,
// and returns the external Stripe Product ID.
func (s *StripeService) CreateProduct(ctx context.Context, productData ps.Product) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("stripe client not configured")
	}

	params := &stripe.ProductCreateParams{
		Name:        stripe.String(productData.Name),
		Description: stripe.String(productData.Description),
		Active:      stripe.Bool(productData.Active),
		Metadata:    productData.Metadata,
		Shippable:   stripe.Bool(productData.Shippable),
	}

	if productData.Type != "" {
		params.Type = stripe.String(string(stripe.ProductType(productData.Type))) // e.g., service or good
	}
	if productData.UnitLabel != "" {
		params.UnitLabel = stripe.String(productData.UnitLabel)
	}
	if productData.TaxCode != "" {
		params.TaxCode = stripe.String(productData.TaxCode)
	}

	s.logger.Info("Creating Stripe product", zap.String("name", productData.Name))

	newStripeProduct, err := s.client.V1Products.Create(ctx, params)
	if err != nil {
		s.logger.Error("Failed to create Stripe product", zap.Error(err), zap.Any("params", params))
		return "", fmt.Errorf("stripe_service.CreateProduct: failed to create product: %w", err)
	}

	s.logger.Info("Successfully created Stripe product", zap.String("stripe_product_id", newStripeProduct.ID))
	return newStripeProduct.ID, nil
}

// GetProduct retrieves a product by its external ID from Stripe using the new stripe.Client API.
func (s *StripeService) GetProduct(ctx context.Context, externalID string) (ps.Product, error) {
	if s.client == nil {
		return ps.Product{}, fmt.Errorf("stripe client not configured")
	}

	s.logger.Info("Fetching Stripe product", zap.String("stripe_product_id", externalID))

	// ProductRetrieveParams can be empty if no specific expansion or other params are needed.
	params := &stripe.ProductRetrieveParams{}

	stripeProd, err := s.client.V1Products.Retrieve(ctx, externalID, params)
	if err != nil {
		s.logger.Error("Failed to fetch Stripe product", zap.Error(err), zap.String("stripe_product_id", externalID))
		return ps.Product{}, fmt.Errorf("stripe_service.GetProduct: failed to fetch product %s: %w", externalID, err)
	}

	// Stripe's Retrieve does not typically return a 'deleted' flag like Customer.
	// If a product is not found, an error (typically resource_missing) is returned.
	// If it's found but inactive, its 'Active' field will be false.

	mappedProduct := mapStripeProductToPSProduct(stripeProd)
	s.logger.Info("Successfully fetched and mapped Stripe product", zap.String("stripe_product_id", externalID))

	return mappedProduct, nil
}

// UpdateProduct updates an existing product in Stripe using the new stripe.Client API.
func (s *StripeService) UpdateProduct(ctx context.Context, externalID string, productData ps.Product) (ps.Product, error) {
	if s.client == nil {
		return ps.Product{}, fmt.Errorf("stripe client not configured")
	}

	params := &stripe.ProductUpdateParams{}

	// Only set fields if they are intended to be updated.
	if productData.Name != "" {
		params.Name = stripe.String(productData.Name)
	}
	if productData.Description != "" {
		params.Description = stripe.String(productData.Description)
	}
	// For boolean fields like Active, we might need a more explicit way to update
	// if we want to distinguish between "false" and "not provided".
	// For now, if Active is in productData, we set it. ps.Product.Active is a bool so it always has a value.
	// Consider using a pointer to bool in ps.Product or a separate update mask if this becomes an issue.
	params.Active = stripe.Bool(productData.Active)
	params.Shippable = stripe.Bool(productData.Shippable)

	// Product type is typically not updatable after creation.
	// if productData.Type != "" { // This block is removed
	// 	params.Type = stripe.String(string(stripe.ProductType(productData.Type)))
	// }
	if productData.Metadata != nil {
		// To clear metadata pass stripe.ClearParams or an empty map, depending on API version specifics.
		// For simplicity, this replaces metadata if productData.Metadata is provided.
		params.Metadata = productData.Metadata
	}

	if productData.UnitLabel != "" {
		params.UnitLabel = stripe.String(productData.UnitLabel)
	}
	if productData.TaxCode != "" {
		params.TaxCode = stripe.String(productData.TaxCode)
	}

	s.logger.Info("Updating Stripe product", zap.String("stripe_product_id", externalID), zap.Any("update_params", params))

	updatedStripeProduct, err := s.client.V1Products.Update(ctx, externalID, params)
	if err != nil {
		s.logger.Error("Failed to update Stripe product", zap.Error(err), zap.String("stripe_product_id", externalID))
		return ps.Product{}, fmt.Errorf("stripe_service.UpdateProduct: failed to update product %s: %w", externalID, err)
	}

	mappedProduct := mapStripeProductToPSProduct(updatedStripeProduct)
	s.logger.Info("Successfully updated and mapped Stripe product", zap.String("stripe_product_id", externalID))

	return mappedProduct, nil
}

// DeleteProduct deletes a product in Stripe using the new stripe.Client API.
// Stripe's product deletion is a hard delete. The product object is returned upon successful deletion, marked as active: false.
func (s *StripeService) DeleteProduct(ctx context.Context, externalID string) error {
	if s.client == nil {
		return fmt.Errorf("stripe client not configured")
	}

	s.logger.Info("Deleting Stripe product", zap.String("stripe_product_id", externalID))

	// ProductDeleteParams can be empty if no specific params are needed for deletion.
	params := &stripe.ProductDeleteParams{}

	_, err := s.client.V1Products.Delete(ctx, externalID, params) // We get back the deleted product, but don't need to use it here.
	if err != nil {
		s.logger.Error("Failed to delete Stripe product", zap.Error(err), zap.String("stripe_product_id", externalID))
		return fmt.Errorf("stripe_service.DeleteProduct: failed to delete product %s: %w", externalID, err)
	}

	s.logger.Info("Successfully deleted Stripe product", zap.String("stripe_product_id", externalID))
	return nil
}

// ListProducts retrieves a list of products from Stripe using the new stripe.Client API.
func (s *StripeService) ListProducts(ctx context.Context, params ps.ListParams) ([]ps.Product, string, error) {
	if s.client == nil {
		return nil, "", fmt.Errorf("stripe client not configured")
	}

	s.logger.Info("Listing Stripe products", zap.Any("params", params))

	stripeParams := &stripe.ProductListParams{}

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

	if len(params.IDs) > 0 {
		stripeParams.IDs = stripe.StringSlice(params.IDs)
	}

	// Handle Active filter from ps.ListParams.Filters if present
	// Stripe ProductListParams has a direct Active field (pointer to bool).
	if activeFilter, ok := params.Filters["active"]; ok {
		if activeBool, ok := activeFilter.(bool); ok {
			stripeParams.Active = stripe.Bool(activeBool)
		}
	}

	// Example of another filter: type
	if typeFilter, okStr := params.Filters["type"].(string); okStr && typeFilter != "" {
		stripeParams.Type = stripe.String(typeFilter)
	}

	var products []ps.Product
	var lastID string

	for stripeProd, err := range s.client.V1Products.List(ctx, stripeParams) {
		if err != nil {
			s.logger.Error("Error iterating Stripe products list", zap.Error(err))
			return nil, "", fmt.Errorf("stripe_service.ListProducts: error during iteration: %w", err)
		}
		if stripeProd == nil {
			continue
		}
		// List already filters by active if stripeParams.Active is set. No need to double check stripeProd.Active here unless specifically desired.
		products = append(products, mapStripeProductToPSProduct(stripeProd))
		lastID = stripeProd.ID
	}

	nextPageCursor := ""
	if params.Limit > 0 && len(products) == params.Limit {
		nextPageCursor = lastID
	}

	s.logger.Info("Successfully listed Stripe products", zap.Int("count", len(products)), zap.String("next_cursor", nextPageCursor))
	return products, nextPageCursor, nil
}
