package stripe

import (
	"context"
	"fmt"

	ps "cyphera-api/internal/client/payment_sync"

	"github.com/stripe/stripe-go/v82"
	// We will use s.client.Customers.New, so specific customer import for New might not be strictly needed
	// but stripe.CustomerParams is from "github.com/stripe/stripe-go/v82"
	"go.uber.org/zap"
)

// mapStripeAddressToPSAddress converts a Stripe Address to ps.Address.
func mapStripeAddressToPSAddress(stripeAddr *stripe.Address) *ps.Address {
	if stripeAddr == nil {
		return nil
	}
	return &ps.Address{
		Line1:      stripeAddr.Line1,
		Line2:      stripeAddr.Line2,
		City:       stripeAddr.City,
		State:      stripeAddr.State,
		PostalCode: stripeAddr.PostalCode,
		Country:    stripeAddr.Country,
	}
}

// mapPSAddressToStripeAddressParams converts a ps.Address to stripe.AddressParams.
func mapPSAddressToStripeAddressParams(psAddr *ps.Address) *stripe.AddressParams {
	if psAddr == nil {
		return nil
	}
	return &stripe.AddressParams{
		Line1:      stripe.String(psAddr.Line1),
		Line2:      stripe.String(psAddr.Line2),
		City:       stripe.String(psAddr.City),
		State:      stripe.String(psAddr.State),
		PostalCode: stripe.String(psAddr.PostalCode),
		Country:    stripe.String(psAddr.Country),
	}
}

// mapStripeCustomerToPSCustomer converts a Stripe Customer object to the canonical ps.Customer.
func mapStripeCustomerToPSCustomer(stripeCust *stripe.Customer) ps.Customer {
	if stripeCust == nil {
		return ps.Customer{}
	}

	var psTaxIDs []ps.TaxID
	if stripeCust.TaxIDs != nil {
		for _, stripeTaxID := range stripeCust.TaxIDs.Data {
			if stripeTaxID != nil {
				psTaxIDs = append(psTaxIDs, ps.TaxID{
					Type:  string(stripeTaxID.Type),
					Value: stripeTaxID.Value,
					// ps.TaxID does not have Country directly, Stripe's does. Included in comment in ps.TaxID.
				})
			}
		}
	}

	var preferredLocale string
	if len(stripeCust.PreferredLocales) > 0 {
		preferredLocale = stripeCust.PreferredLocales[0]
	}

	var shippingAddress *ps.Address
	if stripeCust.Shipping != nil {
		shippingAddress = mapStripeAddressToPSAddress(stripeCust.Shipping.Address)
		// ps.Customer doesn't have distinct ShippingName and ShippingPhone.
		// If needed, these could be added to ps.Address or ps.Customer.ShippingAddress structure.
	}

	return ps.Customer{
		ExternalID:      stripeCust.ID,
		Email:           stripeCust.Email,
		Name:            stripeCust.Name,
		Phone:           stripeCust.Phone,
		Metadata:        stripeCust.Metadata,
		BillingAddress:  mapStripeAddressToPSAddress(stripeCust.Address),
		ShippingAddress: shippingAddress,
		TaxIDs:          psTaxIDs,
		PreferredLocale: preferredLocale,
		// ps.Customer.ID (Cyphera's internal ID) would be populated elsewhere,
		// this mapping is primarily for data received from Stripe.
	}
}

// CreateCustomer creates a new customer in Stripe using the new stripe.Client API.
func (s *StripeService) CreateCustomer(ctx context.Context, customerData ps.Customer) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("stripe client not configured")
	}

	params := &stripe.CustomerCreateParams{
		Email:    stripe.String(customerData.Email),
		Name:     stripe.String(customerData.Name),
		Phone:    stripe.String(customerData.Phone),
		Metadata: customerData.Metadata,
	}

	if customerData.BillingAddress != nil {
		params.Address = mapPSAddressToStripeAddressParams(customerData.BillingAddress)
	}

	if customerData.ShippingAddress != nil {
		shippingDetails := &stripe.CustomerCreateShippingParams{
			Address: mapPSAddressToStripeAddressParams(customerData.ShippingAddress),
			Name:    stripe.String(customerData.Name),  // Use main customer name for shipping name
			Phone:   stripe.String(customerData.Phone), // Use main customer phone for shipping phone
		}
		params.Shipping = shippingDetails
	}

	if len(customerData.TaxIDs) > 0 {
		params.TaxIDData = make([]*stripe.CustomerCreateTaxIDDataParams, len(customerData.TaxIDs))
		for i, psTaxID := range customerData.TaxIDs {
			params.TaxIDData[i] = &stripe.CustomerCreateTaxIDDataParams{
				Type:  stripe.String(psTaxID.Type),
				Value: stripe.String(psTaxID.Value),
			}
		}
	}

	if customerData.PreferredLocale != "" {
		params.PreferredLocales = []*string{stripe.String(customerData.PreferredLocale)}
	}

	s.logger.Info("Creating Stripe customer", zap.String("email", customerData.Email))

	newStripeCustomer, err := s.client.V1Customers.Create(ctx, params)
	if err != nil {
		s.logger.Error("Failed to create Stripe customer", zap.Error(err), zap.Any("params", params))
		return "", fmt.Errorf("stripe_service.CreateCustomer: failed to create customer: %w", err)
	}

	s.logger.Info("Successfully created Stripe customer", zap.String("stripe_customer_id", newStripeCustomer.ID))
	return newStripeCustomer.ID, nil
}

// GetCustomer retrieves a customer by their external ID from Stripe using the new stripe.Client API.
func (s *StripeService) GetCustomer(ctx context.Context, externalID string) (ps.Customer, error) {
	if s.client == nil {
		return ps.Customer{}, fmt.Errorf("stripe client not configured")
	}

	s.logger.Info("Fetching Stripe customer", zap.String("stripe_customer_id", externalID))

	params := &stripe.CustomerRetrieveParams{}
	// Note: Context is now the first argument to the Retrieve method.
	// params.Params.Context = ctx // This line is removed

	stripeCust, err := s.client.V1Customers.Retrieve(ctx, externalID, params)
	if err != nil {
		s.logger.Error("Failed to fetch Stripe customer", zap.Error(err), zap.String("stripe_customer_id", externalID))
		return ps.Customer{}, fmt.Errorf("stripe_service.GetCustomer: failed to fetch customer %s: %w", externalID, err)
	}

	if stripeCust.Deleted {
		s.logger.Warn("Fetched Stripe customer is marked as deleted", zap.String("stripe_customer_id", externalID))
		return ps.Customer{}, fmt.Errorf("stripe_service.GetCustomer: customer %s is deleted", externalID)
	}

	mappedCustomer := mapStripeCustomerToPSCustomer(stripeCust)
	s.logger.Info("Successfully fetched and mapped Stripe customer", zap.String("stripe_customer_id", externalID))

	return mappedCustomer, nil
}

// UpdateCustomer updates an existing customer in Stripe using the new stripe.Client API.
func (s *StripeService) UpdateCustomer(ctx context.Context, externalID string, customerData ps.Customer) (ps.Customer, error) {
	if s.client == nil {
		return ps.Customer{}, fmt.Errorf("stripe client not configured")
	}

	params := &stripe.CustomerUpdateParams{}

	if customerData.Email != "" {
		params.Email = stripe.String(customerData.Email)
	}
	if customerData.Name != "" {
		params.Name = stripe.String(customerData.Name)
	}
	if customerData.Phone != "" {
		params.Phone = stripe.String(customerData.Phone)
	}
	if customerData.Metadata != nil {
		params.Metadata = customerData.Metadata // Handles clearing metadata if customerData.Metadata is an empty map.
	}

	if customerData.BillingAddress != nil {
		params.Address = mapPSAddressToStripeAddressParams(customerData.BillingAddress)
	} else {
		// To clear an address, you might need to pass an empty AddressParams or specific clear instructions based on API.
		// For now, not providing it means no update to address. If ps.Customer.BillingAddress being nil means "clear",
		// this would need stripe.EmptyAddressParams or similar if available and supported by the API for clearing.
		// Current behavior: if nil, no change. If non-nil, update/set.
	}

	if customerData.ShippingAddress != nil {
		params.Shipping = &stripe.CustomerUpdateShippingParams{
			Address: mapPSAddressToStripeAddressParams(customerData.ShippingAddress),
			Name:    stripe.String(customerData.Name),  // Assuming shipping name updates with main name
			Phone:   stripe.String(customerData.Phone), // Assuming shipping phone updates with main phone
		}
	} else {
		// Similar to BillingAddress, clearing shipping might require explicit params.
		// Current behavior: if nil, no change. If non-nil, update/set.
	}

	if customerData.PreferredLocale != "" {
		params.PreferredLocales = []*string{stripe.String(customerData.PreferredLocale)}
	} else {
		// To clear preferred_locales, pass an empty slice: params.PreferredLocales = []*string{}
		// For now, an empty string in ps.Customer.PreferredLocale will not clear it on Stripe unless explicitly handled.
	}

	// Note: TaxIDs cannot be updated directly via CustomerUpdateParams.
	// They must be managed using the Tax ID API methods (e.g., creating/deleting tax IDs for a customer).
	// Syncing TaxID changes would require fetching existing TaxIDs, diffing, and making separate API calls.

	s.logger.Info("Updating Stripe customer", zap.String("stripe_customer_id", externalID), zap.Any("update_params", params))

	updatedStripeCustomer, err := s.client.V1Customers.Update(ctx, externalID, params)
	if err != nil {
		s.logger.Error("Failed to update Stripe customer", zap.Error(err), zap.String("stripe_customer_id", externalID))
		return ps.Customer{}, fmt.Errorf("stripe_service.UpdateCustomer: failed to update customer %s: %w", externalID, err)
	}

	mappedCustomer := mapStripeCustomerToPSCustomer(updatedStripeCustomer)
	s.logger.Info("Successfully updated and mapped Stripe customer", zap.String("stripe_customer_id", externalID))

	return mappedCustomer, nil
}

// DeleteCustomer deletes a customer in Stripe using the new stripe.Client API.
func (s *StripeService) DeleteCustomer(ctx context.Context, externalID string) error {
	if s.client == nil {
		return fmt.Errorf("stripe client not configured")
	}

	s.logger.Info("Deleting Stripe customer", zap.String("stripe_customer_id", externalID))

	params := &stripe.CustomerDeleteParams{}
	// Note: Context is now the first argument to the Delete method.
	// params.Params.Context = ctx // This line is removed

	_, err := s.client.V1Customers.Delete(ctx, externalID, params)
	if err != nil {
		s.logger.Error("Failed to delete Stripe customer", zap.Error(err), zap.String("stripe_customer_id", externalID))
		return fmt.Errorf("stripe_service.DeleteCustomer: failed to delete customer %s: %w", externalID, err)
	}

	s.logger.Info("Successfully marked Stripe customer as deleted", zap.String("stripe_customer_id", externalID))
	return nil
}

// ListCustomers retrieves a list of customers from Stripe using the new stripe.Client API and iter.Seq2.
func (s *StripeService) ListCustomers(ctx context.Context, params ps.ListParams) ([]ps.Customer, string, error) {
	if s.client == nil {
		return nil, "", fmt.Errorf("stripe client not configured")
	}

	s.logger.Info("Listing Stripe customers", zap.Any("params", params))

	stripeParams := &stripe.CustomerListParams{} // CustomerListParams remains the same type

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

	if emailFilter, ok := params.Filters["email"].(string); ok && emailFilter != "" {
		stripeParams.Email = stripe.String(emailFilter)
	}
	// Note: stripeParams.ListParams.Context = ctx is removed as ctx is passed directly.

	var customers []ps.Customer
	var lastID string

	// Use the new iter.Seq2 pattern for listing as per the migration guide.
	// s.client.V1Customers.List(ctx, stripeParams) directly returns an iter.Seq2 compatible type.
	for stripeCust, err := range s.client.V1Customers.List(ctx, stripeParams) {
		if err != nil {
			s.logger.Error("Error iterating Stripe customers list", zap.Error(err))
			return nil, "", fmt.Errorf("stripe_service.ListCustomers: error during iteration: %w", err)
		}
		if stripeCust == nil {
			continue
		}
		if stripeCust.Deleted {
			continue
		}
		customers = append(customers, mapStripeCustomerToPSCustomer(stripeCust))
		lastID = stripeCust.ID
	}

	// The range loop over iter.Seq2 handles errors per item.
	// No separate iter.Err() check is typically needed or available from Seq2 directly.

	nextPageCursor := ""
	// Determine nextPageCursor: If the number of items fetched equals the limit,
	// it's likely there are more. This is a heuristic.
	if params.Limit > 0 && len(customers) == params.Limit {
		nextPageCursor = lastID
	}

	s.logger.Info("Successfully listed Stripe customers", zap.Int("count", len(customers)), zap.String("next_cursor", nextPageCursor))
	return customers, nextPageCursor, nil
}
