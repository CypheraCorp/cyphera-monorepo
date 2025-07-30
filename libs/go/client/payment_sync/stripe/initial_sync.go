package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	ps "github.com/cyphera/cyphera-api/libs/go/client/payment_sync"
	"github.com/cyphera/cyphera-api/libs/go/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
	"github.com/stripe/stripe-go/v82/subscription"
	"go.uber.org/zap"
)

// runSyncProcess executes the actual sync process
func (s *StripeService) runSyncProcess(ctx context.Context, session *db.PaymentSyncSession, config ps.InitialSyncConfig) error {
	totalProcessed := 0
	var lastError error

	progress := map[string]interface{}{
		"total_processed": 0,
		"current_entity":  "",
		"started_at":      time.Now(),
	}

	// Process each entity type
	for _, entityType := range config.EntityTypes {
		s.logger.Info("Processing entity type",
			zap.String("entity_type", entityType),
			zap.String("session_id", session.ID.String()))

		progress["current_entity"] = entityType

		// Marshal progress to JSON
		progressJSON, _ := json.Marshal(progress)
		if _, err := s.db.UpdateSyncSessionProgress(ctx, db.UpdateSyncSessionProgressParams{
			ID:       session.ID,
			Progress: progressJSON,
		}); err != nil {
			s.logger.Error("Failed to update sync session progress", zap.Error(err))
		}

		var processed int
		var err error

		switch entityType {
		case "customers":
			processed, err = s.syncCustomers(ctx, session, config)
		case "products":
			processed, err = s.syncProducts(ctx, session, config)
		case "prices":
			processed, err = s.syncPrices(ctx, session, config)
		case "subscriptions":
			processed, err = s.syncSubscriptions(ctx, session, config)
		default:
			s.logger.Warn("Unknown entity type", zap.String("entity_type", entityType))
			continue
		}

		if err != nil {
			lastError = err
			s.logger.Error("Failed to sync entity type",
				zap.String("entity_type", entityType),
				zap.Error(err))

			// Log the error event but continue with other entities
			s.logSyncEvent(ctx, session, entityType, "", "sync_failed", err.Error(), nil)
			continue
		}

		totalProcessed += processed
		progress["total_processed"] = totalProcessed
		progress[entityType+"_processed"] = processed

		s.logger.Info("Completed entity type sync",
			zap.String("entity_type", entityType),
			zap.Int("processed", processed))
	}

	// Update final session status
	finalStatus := "completed"
	if lastError != nil {
		finalStatus = "failed"
		progress["final_error"] = lastError.Error()
	}

	progress["completed_at"] = time.Now()
	progress["total_processed"] = totalProcessed

	_, err := s.db.UpdateSyncSessionStatus(ctx, db.UpdateSyncSessionStatusParams{
		ID:     session.ID,
		Status: finalStatus,
	})
	if err != nil {
		s.logger.Error("Failed to update final session status", zap.Error(err))
	}

	// Marshal final progress
	progressJSON, _ := json.Marshal(progress)
	if _, err := s.db.UpdateSyncSessionProgress(ctx, db.UpdateSyncSessionProgressParams{
		ID:       session.ID,
		Progress: progressJSON,
	}); err != nil {
		s.logger.Error("Failed to update final sync session progress", zap.Error(err))
	}

	s.logger.Info("Initial sync completed",
		zap.String("session_id", session.ID.String()),
		zap.String("status", finalStatus),
		zap.Int("total_processed", totalProcessed))

	return lastError
}

// syncCustomers syncs all customers from Stripe
func (s *StripeService) syncCustomers(ctx context.Context, session *db.PaymentSyncSession, config ps.InitialSyncConfig) (int, error) {
	s.logSyncEvent(ctx, session, "customer", "", "sync_started", "Starting customer sync", nil)

	processed := 0
	params := &stripe.CustomerListParams{
		ListParams: stripe.ListParams{
			Limit: stripe.Int64(int64(config.BatchSize)),
		},
	}

	if config.StartingAfter != "" {
		params.StartingAfter = stripe.String(config.StartingAfter)
	}

	iter := customer.List(params)
	for iter.Next() {
		stripeCustomer := iter.Customer()

		// Convert to canonical format
		psCustomer := mapStripeCustomerToPSCustomer(stripeCustomer)

		// Create or update customer in database
		err := s.UpsertCustomer(ctx, session, psCustomer)
		if err != nil {
			s.logger.Error("Failed to upsert customer",
				zap.String("stripe_id", stripeCustomer.ID),
				zap.Error(err))

			s.logSyncEvent(ctx, session, "customer", "", "sync_failed",
				fmt.Sprintf("Failed to upsert customer %s: %v", stripeCustomer.ID, err),
				map[string]interface{}{"stripe_id": stripeCustomer.ID})
			continue
		}

		s.logSyncEvent(ctx, session, "customer", "", "sync_completed",
			fmt.Sprintf("Successfully synced customer %s", stripeCustomer.ID),
			map[string]interface{}{"stripe_id": stripeCustomer.ID})

		processed++
	}

	if err := iter.Err(); err != nil {
		return processed, fmt.Errorf("stripe customer iteration error: %w", err)
	}

	return processed, nil
}

// syncProducts syncs all products from Stripe
func (s *StripeService) syncProducts(ctx context.Context, session *db.PaymentSyncSession, config ps.InitialSyncConfig) (int, error) {
	s.logSyncEvent(ctx, session, "product", "", "sync_started", "Starting product sync", nil)

	processed := 0
	params := &stripe.ProductListParams{
		ListParams: stripe.ListParams{
			Limit: stripe.Int64(int64(config.BatchSize)),
		},
	}

	iter := product.List(params)
	for iter.Next() {
		stripeProduct := iter.Product()

		// Convert to canonical format
		psProduct := mapStripeProductToPSProduct(stripeProduct)

		// Create or update product in database
		err := s.UpsertProduct(ctx, session, psProduct)
		if err != nil {
			s.logger.Error("Failed to upsert product",
				zap.String("stripe_id", stripeProduct.ID),
				zap.Error(err))

			s.logSyncEvent(ctx, session, "product", "", "sync_failed",
				fmt.Sprintf("Failed to upsert product %s: %v", stripeProduct.ID, err),
				map[string]interface{}{"stripe_id": stripeProduct.ID})
			continue
		}

		s.logSyncEvent(ctx, session, "product", "", "sync_completed",
			fmt.Sprintf("Successfully synced product %s", stripeProduct.ID),
			map[string]interface{}{"stripe_id": stripeProduct.ID})

		processed++
	}

	if err := iter.Err(); err != nil {
		return processed, fmt.Errorf("stripe product iteration error: %w", err)
	}

	return processed, nil
}

// syncPrices syncs all prices from Stripe
func (s *StripeService) syncPrices(ctx context.Context, session *db.PaymentSyncSession, config ps.InitialSyncConfig) (int, error) {
	s.logSyncEvent(ctx, session, "price", "", "sync_started", "Starting price sync", nil)

	processed := 0
	params := &stripe.PriceListParams{
		ListParams: stripe.ListParams{
			Limit: stripe.Int64(int64(config.BatchSize)),
		},
	}

	iter := price.List(params)
	for iter.Next() {
		stripePrice := iter.Price()

		// Convert to canonical format
		psPrice := mapStripePriceToPSPrice(stripePrice)

		// Create or update price in database
		err := s.UpsertPrice(ctx, session, psPrice)
		if err != nil {
			s.logger.Error("Failed to upsert price",
				zap.String("stripe_id", stripePrice.ID),
				zap.Error(err))

			s.logSyncEvent(ctx, session, "price", "", "sync_failed",
				fmt.Sprintf("Failed to upsert price %s: %v", stripePrice.ID, err),
				map[string]interface{}{"stripe_id": stripePrice.ID})
			continue
		}

		s.logSyncEvent(ctx, session, "price", "", "sync_completed",
			fmt.Sprintf("Successfully synced price %s", stripePrice.ID),
			map[string]interface{}{"stripe_id": stripePrice.ID})

		processed++
	}

	if err := iter.Err(); err != nil {
		return processed, fmt.Errorf("stripe price iteration error: %w", err)
	}

	return processed, nil
}

// syncSubscriptions syncs all subscriptions from Stripe
func (s *StripeService) syncSubscriptions(ctx context.Context, session *db.PaymentSyncSession, config ps.InitialSyncConfig) (int, error) {
	s.logSyncEvent(ctx, session, "subscription", "", "sync_started", "Starting subscription sync", nil)

	processed := 0
	params := &stripe.SubscriptionListParams{
		ListParams: stripe.ListParams{
			Limit: stripe.Int64(int64(config.BatchSize)),
		},
	}

	iter := subscription.List(params)
	for iter.Next() {
		stripeSubscription := iter.Subscription()

		// Convert to canonical format
		psSubscription := mapStripeSubscriptionToPSSubscription(stripeSubscription)

		// Create or update subscription in database
		err := s.UpsertSubscription(ctx, session, psSubscription)
		if err != nil {
			s.logger.Error("Failed to upsert subscription",
				zap.String("stripe_id", stripeSubscription.ID),
				zap.Error(err))

			s.logSyncEvent(ctx, session, "subscription", "", "sync_failed",
				fmt.Sprintf("Failed to upsert subscription %s: %v", stripeSubscription.ID, err),
				map[string]interface{}{"stripe_id": stripeSubscription.ID})
			continue
		}

		s.logSyncEvent(ctx, session, "subscription", "", "sync_completed",
			fmt.Sprintf("Successfully synced subscription %s", stripeSubscription.ID),
			map[string]interface{}{"stripe_id": stripeSubscription.ID})

		processed++
	}

	if err := iter.Err(); err != nil {
		return processed, fmt.Errorf("stripe subscription iteration error: %w", err)
	}

	return processed, nil
}

// logSyncEvent logs a sync event to the database
func (s *StripeService) logSyncEvent(ctx context.Context, session *db.PaymentSyncSession, entityType, entityID, eventType, message string, details map[string]interface{}) {
	var entityUUID pgtype.UUID
	if entityID != "" {
		if parsed, err := uuid.Parse(entityID); err == nil {
			entityUUID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	var eventMessage pgtype.Text
	if message != "" {
		eventMessage = pgtype.Text{String: message, Valid: true}
	}

	var eventDetails []byte
	if details != nil {
		eventDetails, _ = json.Marshal(details)
	}

	_, err := s.db.CreateSyncEvent(ctx, db.CreateSyncEventParams{
		SessionID:    session.ID,
		WorkspaceID:  session.WorkspaceID,
		ProviderName: session.ProviderName,
		EntityType:   entityType,
		EntityID:     entityUUID,
		EventType:    eventType,
		EventMessage: eventMessage,
		EventDetails: eventDetails,
	})
	if err != nil {
		s.logger.Error("Failed to log sync event", zap.Error(err))
	}
}

// UpsertCustomer creates or updates a customer from payment sync data
func (s *StripeService) UpsertCustomer(ctx context.Context, session *db.PaymentSyncSession, customer ps.Customer) error {
	s.logger.Debug("Upserting customer",
		zap.String("external_id", customer.ExternalID),
		zap.String("email", customer.Email))

	// Marshal metadata to JSON
	metadata, err := json.Marshal(customer.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal customer metadata: %w", err)
	}

	// Try to find existing customer by external ID (now workspace-independent)
	existingCustomer, err := s.db.GetCustomerByExternalID(ctx, pgtype.Text{String: customer.ExternalID, Valid: true})

	if err != nil {
		// Customer doesn't exist, create new one
		s.logger.Debug("Creating new customer", zap.String("external_id", customer.ExternalID))

		newCustomer, err := s.db.CreateCustomerWithSync(ctx, db.CreateCustomerWithSyncParams{
			ExternalID:        pgtype.Text{String: customer.ExternalID, Valid: true},
			Email:             pgtype.Text{String: customer.Email, Valid: customer.Email != ""},
			Name:              pgtype.Text{String: customer.Name, Valid: customer.Name != ""},
			Phone:             pgtype.Text{String: customer.Phone, Valid: customer.Phone != ""},
			Metadata:          metadata,
			PaymentSyncStatus: pgtype.Text{String: "synced", Valid: true},
			PaymentProvider:   pgtype.Text{String: "stripe", Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to create customer: %w", err)
		}

		// Associate customer with workspace using the new association table
		_, err = s.db.AddCustomerToWorkspace(ctx, db.AddCustomerToWorkspaceParams{
			WorkspaceID: session.WorkspaceID,
			CustomerID:  newCustomer.ID,
		})
		if err != nil {
			return fmt.Errorf("failed to associate customer with workspace: %w", err)
		}

		s.logger.Debug("Successfully created customer", zap.String("external_id", customer.ExternalID))
	} else {
		// Customer exists, update it
		s.logger.Debug("Updating existing customer",
			zap.String("external_id", customer.ExternalID),
			zap.String("existing_id", existingCustomer.ID.String()))

		_, err = s.db.UpdateCustomer(ctx, db.UpdateCustomerParams{
			ID:       existingCustomer.ID,
			Email:    pgtype.Text{String: customer.Email, Valid: customer.Email != ""},
			Name:     pgtype.Text{String: customer.Name, Valid: customer.Name != ""},
			Phone:    pgtype.Text{String: customer.Phone, Valid: customer.Phone != ""},
			Metadata: metadata,
		})
		if err != nil {
			return fmt.Errorf("failed to update customer: %w", err)
		}

		// Ensure customer is associated with this workspace
		_, err = s.db.AddCustomerToWorkspace(ctx, db.AddCustomerToWorkspaceParams{
			WorkspaceID: session.WorkspaceID,
			CustomerID:  existingCustomer.ID,
		})
		if err != nil {
			// Ignore error if association already exists
			s.logger.Debug("Customer already associated with workspace or association failed",
				zap.String("customer_id", existingCustomer.ID.String()),
				zap.String("workspace_id", session.WorkspaceID.String()))
		}

		// Update sync status
		_, err = s.db.UpdateCustomerSyncStatus(ctx, db.UpdateCustomerSyncStatusParams{
			ID:                existingCustomer.ID,
			PaymentSyncStatus: pgtype.Text{String: "synced", Valid: true},
			PaymentProvider:   pgtype.Text{String: "stripe", Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update customer sync status: %w", err)
		}

		s.logger.Debug("Successfully updated customer", zap.String("external_id", customer.ExternalID))
	}

	return nil
}

// UpsertProduct creates or updates a product from payment sync data
func (s *StripeService) UpsertProduct(ctx context.Context, session *db.PaymentSyncSession, product ps.Product) error {
	s.logger.Debug("Upserting product",
		zap.String("external_id", product.ExternalID),
		zap.String("name", product.Name))

	// Marshal metadata to JSON
	metadata, err := json.Marshal(product.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal product metadata: %w", err)
	}

	// Try to find existing product by external ID
	existingProduct, err := s.db.GetProductByExternalID(ctx, db.GetProductByExternalIDParams{
		WorkspaceID:     session.WorkspaceID,
		ExternalID:      pgtype.Text{String: product.ExternalID, Valid: true},
		PaymentProvider: pgtype.Text{String: "stripe", Valid: true},
	})

	if err != nil {
		// Product doesn't exist, create new one
		s.logger.Debug("Creating new product", zap.String("external_id", product.ExternalID))

		// We need a wallet_id for the product - let's get the first wallet for this workspace
		// This is a simplification - in a real implementation you might want to handle this differently
		wallets, err := s.db.ListWalletsByWorkspaceID(ctx, session.WorkspaceID)
		if err != nil || len(wallets) == 0 {
			return fmt.Errorf("no wallets found for workspace: %w", err)
		}

		_, err = s.db.CreateProductWithSync(ctx, db.CreateProductWithSyncParams{
			WorkspaceID:        session.WorkspaceID,
			WalletID:           wallets[0].ID, // Use first wallet
			ExternalID:         pgtype.Text{String: product.ExternalID, Valid: true},
			Name:               product.Name,
			Description:        pgtype.Text{String: product.Description, Valid: product.Description != ""},
			Active:             product.Active,
			Metadata:           metadata,
			PaymentSyncStatus:  pgtype.Text{String: "synced", Valid: true},
			PaymentSyncedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
			PaymentSyncVersion: pgtype.Int4{Int32: 1, Valid: true},
			PaymentProvider:    pgtype.Text{String: "stripe", Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to create product: %w", err)
		}

		s.logger.Debug("Successfully created product", zap.String("external_id", product.ExternalID))
	} else {
		// Product exists, update it
		s.logger.Debug("Updating existing product",
			zap.String("external_id", product.ExternalID),
			zap.String("existing_id", existingProduct.ID.String()))

		_, err = s.db.UpdateProduct(ctx, db.UpdateProductParams{
			ID:          existingProduct.ID,
			WorkspaceID: session.WorkspaceID,
			Name:        product.Name,
			WalletID:    existingProduct.WalletID, // Keep existing wallet
			Description: pgtype.Text{String: product.Description, Valid: product.Description != ""},
			Active:      product.Active,
			Metadata:    metadata,
		})
		if err != nil {
			return fmt.Errorf("failed to update product: %w", err)
		}

		// Update sync status
		_, err = s.db.UpdateProductPaymentSyncStatus(ctx, db.UpdateProductPaymentSyncStatusParams{
			ID:                existingProduct.ID,
			WorkspaceID:       session.WorkspaceID,
			PaymentSyncStatus: pgtype.Text{String: "synced", Valid: true},
			PaymentProvider:   pgtype.Text{String: "stripe", Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update product sync status: %w", err)
		}

		s.logger.Debug("Successfully updated product", zap.String("external_id", product.ExternalID))
	}

	return nil
}

// UpsertPrice creates or updates a price from payment sync data
func (s *StripeService) UpsertPrice(ctx context.Context, session *db.PaymentSyncSession, price ps.Price) error {
	s.logger.Debug("Upserting price",
		zap.String("external_id", price.ExternalID),
		zap.String("product_id", price.ProductID))

	// Marshal metadata to JSON
	metadata, err := json.Marshal(price.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal price metadata: %w", err)
	}

	// Find the corresponding product first
	product, err := s.db.GetProductByExternalID(ctx, db.GetProductByExternalIDParams{
		WorkspaceID:     session.WorkspaceID,
		ExternalID:      pgtype.Text{String: price.ProductID, Valid: true},
		PaymentProvider: pgtype.Text{String: "stripe", Valid: true},
	})
	if err != nil {
		return fmt.Errorf("product not found for external_id: %s", price.ProductID)
	}

	// Convert price type to enum
	var priceType db.PriceType
	switch price.Type {
	case "recurring":
		priceType = db.PriceTypeRecurring
	case "one_time":
		priceType = db.PriceTypeOneOff
	default:
		priceType = db.PriceTypeOneOff // default
	}

	// Convert currency to uppercase string for database
	currency := strings.ToUpper(price.Currency)

	// Handle interval type for recurring prices
	var intervalType db.IntervalType
	var termLength int32 = 1 // default
	if price.Recurring != nil {
		switch price.Recurring.Interval {
		case "day":
			intervalType = db.IntervalTypeDaily
		case "week":
			intervalType = db.IntervalTypeWeek
		case "month":
			intervalType = db.IntervalTypeMonth
		case "year":
			intervalType = db.IntervalTypeYear
		default:
			intervalType = db.IntervalTypeMonth // default
		}
		if price.Recurring.IntervalCount > 0 {
			termLength = int32(price.Recurring.IntervalCount)
		}
	} else {
		// For one-time prices, we need to set interval_type to something valid
		intervalType = db.IntervalTypeDaily // This will be ignored due to the CHECK constraint
		termLength = 1
	}

	// Try to find existing price by external ID
	existingPrice, err := s.db.GetPriceByExternalID(ctx, db.GetPriceByExternalIDParams{
		ExternalID:      pgtype.Text{String: price.ExternalID, Valid: true},
		PaymentProvider: pgtype.Text{String: "stripe", Valid: true},
	})

	if err != nil {
		// Price doesn't exist, create new one
		s.logger.Debug("Creating new price", zap.String("external_id", price.ExternalID))

		_, err = s.db.CreatePriceWithSync(ctx, db.CreatePriceWithSyncParams{
			ProductID:           product.ID,
			ExternalID:          pgtype.Text{String: price.ExternalID, Valid: true},
			Active:              price.Active,
			Type:                priceType,
			Currency:            currency,
			UnitAmountInPennies: int32(price.Amount),
			IntervalType:        intervalType,
			TermLength:          termLength,
			Metadata:            metadata,
			PaymentSyncStatus:   pgtype.Text{String: "synced", Valid: true},
			PaymentSyncedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
			PaymentSyncVersion:  pgtype.Int4{Int32: 1, Valid: true},
			PaymentProvider:     pgtype.Text{String: "stripe", Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to create price: %w", err)
		}

		s.logger.Debug("Successfully created price", zap.String("external_id", price.ExternalID))
	} else {
		// Price exists, update it
		s.logger.Debug("Updating existing price",
			zap.String("external_id", price.ExternalID),
			zap.String("existing_id", existingPrice.ID.String()))

		_, err = s.db.UpdatePrice(ctx, db.UpdatePriceParams{
			ID:       existingPrice.ID,
			Active:   price.Active,
			Metadata: metadata,
		})
		if err != nil {
			return fmt.Errorf("failed to update price: %w", err)
		}

		// Update sync status
		_, err = s.db.UpdatePricePaymentSyncStatus(ctx, db.UpdatePricePaymentSyncStatusParams{
			ID:                existingPrice.ID,
			PaymentSyncStatus: pgtype.Text{String: "synced", Valid: true},
			PaymentProvider:   pgtype.Text{String: "stripe", Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update price sync status: %w", err)
		}

		s.logger.Debug("Successfully updated price", zap.String("external_id", price.ExternalID))
	}

	return nil
}

// UpsertSubscription creates or updates a subscription from payment sync data
func (s *StripeService) UpsertSubscription(ctx context.Context, session *db.PaymentSyncSession, subscription ps.Subscription) error {
	s.logger.Debug("Upserting subscription",
		zap.String("external_id", subscription.ExternalID),
		zap.String("customer_id", subscription.CustomerID))

	// Marshal metadata to JSON
	metadata, err := json.Marshal(subscription.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription metadata: %w", err)
	}

	// Find the corresponding customer
	existingCustomer, err := s.db.GetCustomerByExternalID(ctx, pgtype.Text{String: subscription.CustomerID, Valid: true})
	if err != nil {
		return fmt.Errorf("failed to find customer for subscription: %w", err)
	}

	// For simplicity, we'll handle the first subscription item
	if len(subscription.Items) == 0 {
		return fmt.Errorf("subscription has no items")
	}

	priceItem := subscription.Items[0]

	// Find the price
	price, err := s.db.GetPriceByExternalID(ctx, db.GetPriceByExternalIDParams{
		ExternalID:      pgtype.Text{String: priceItem.PriceID, Valid: true},
		PaymentProvider: pgtype.Text{String: "stripe", Valid: true},
	})
	if err != nil {
		return fmt.Errorf("price not found for external_id: %s", priceItem.PriceID)
	}

	// Find the product
	product, err := s.db.GetProduct(ctx, db.GetProductParams{
		ID:          price.ProductID,
		WorkspaceID: session.WorkspaceID,
	})
	if err != nil {
		return fmt.Errorf("product not found for price: %w", err)
	}

	// Convert subscription status to enum
	var status db.SubscriptionStatus
	switch subscription.Status {
	case "active":
		status = db.SubscriptionStatusActive
	case "canceled":
		status = db.SubscriptionStatusCanceled
	default:
		status = db.SubscriptionStatusActive // default
	}

	// Convert timestamps
	currentPeriodStart := pgtype.Timestamptz{
		Time:  time.Unix(subscription.CurrentPeriodStart, 0),
		Valid: subscription.CurrentPeriodStart > 0,
	}
	currentPeriodEnd := pgtype.Timestamptz{
		Time:  time.Unix(subscription.CurrentPeriodEnd, 0),
		Valid: subscription.CurrentPeriodEnd > 0,
	}

	// Try to find existing subscription by external ID
	existingSubscription, err := s.db.GetSubscriptionByExternalID(ctx, db.GetSubscriptionByExternalIDParams{
		WorkspaceID:     session.WorkspaceID,
		ExternalID:      pgtype.Text{String: subscription.ExternalID, Valid: true},
		PaymentProvider: pgtype.Text{String: "stripe", Valid: true},
	})

	if err != nil {
		// Subscription doesn't exist, create new one
		s.logger.Debug("Creating new subscription", zap.String("external_id", subscription.ExternalID))

		// For now, we'll create a dummy delegation and product token since these are required
		// In a real implementation, you'd need to handle these properly

		// Create a dummy delegation (this is a simplification)
		delegationData, err := s.db.CreateDelegationData(ctx, db.CreateDelegationDataParams{
			Delegate:  "dummy_delegate",
			Delegator: "dummy_delegator",
			Authority: "dummy_authority",
			Caveats:   []byte("[]"),
			Salt:      "dummy_salt",
			Signature: "dummy_signature",
		})
		if err != nil {
			return fmt.Errorf("failed to create delegation data: %w", err)
		}

		// Get the first product token for this product (this is a simplification)
		productTokens, err := s.db.GetActiveProductTokensByProduct(ctx, product.ID)
		if err != nil || len(productTokens) == 0 {
			return fmt.Errorf("no product tokens found for product: %w", err)
		}

		_, err = s.db.CreateSubscriptionWithSync(ctx, db.CreateSubscriptionWithSyncParams{
			CustomerID:         existingCustomer.ID,
			ProductID:          product.ID,
			WorkspaceID:        session.WorkspaceID,
			PriceID:            price.ID,
			ProductTokenID:     productTokens[0].ID,
			ExternalID:         pgtype.Text{String: subscription.ExternalID, Valid: true},
			TokenAmount:        1, // Default token amount
			DelegationID:       delegationData.ID,
			Status:             status,
			CurrentPeriodStart: currentPeriodStart,
			CurrentPeriodEnd:   currentPeriodEnd,
			TotalRedemptions:   0,
			TotalAmountInCents: 0,
			Metadata:           metadata,
			PaymentSyncStatus:  pgtype.Text{String: "synced", Valid: true},
			PaymentSyncedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
			PaymentSyncVersion: pgtype.Int4{Int32: 1, Valid: true},
			PaymentProvider:    pgtype.Text{String: "stripe", Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to create subscription: %w", err)
		}

		s.logger.Debug("Successfully created subscription", zap.String("external_id", subscription.ExternalID))
	} else {
		// Subscription exists, update it
		s.logger.Debug("Updating existing subscription",
			zap.String("external_id", subscription.ExternalID),
			zap.String("existing_id", existingSubscription.ID.String()))

		_, err = s.db.UpdateSubscription(ctx, db.UpdateSubscriptionParams{
			ID:                 existingSubscription.ID,
			CustomerID:         existingSubscription.CustomerID,
			ProductID:          existingSubscription.ProductID,
			WorkspaceID:        existingSubscription.WorkspaceID,
			PriceID:            existingSubscription.PriceID,
			ProductTokenID:     existingSubscription.ProductTokenID,
			TokenAmount:        existingSubscription.TokenAmount,
			DelegationID:       existingSubscription.DelegationID,
			CustomerWalletID:   existingSubscription.CustomerWalletID,
			Status:             status,
			CurrentPeriodStart: currentPeriodStart,
			CurrentPeriodEnd:   currentPeriodEnd,
			NextRedemptionDate: existingSubscription.NextRedemptionDate,
			TotalRedemptions:   existingSubscription.TotalRedemptions,
			TotalAmountInCents: existingSubscription.TotalAmountInCents,
			Metadata:           metadata,
		})
		if err != nil {
			return fmt.Errorf("failed to update subscription: %w", err)
		}

		// Update sync status
		_, err = s.db.UpdateSubscriptionPaymentSyncStatus(ctx, db.UpdateSubscriptionPaymentSyncStatusParams{
			ID:                existingSubscription.ID,
			WorkspaceID:       session.WorkspaceID,
			PaymentSyncStatus: pgtype.Text{String: "synced", Valid: true},
			PaymentProvider:   pgtype.Text{String: "stripe", Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update subscription sync status: %w", err)
		}

		s.logger.Debug("Successfully updated subscription", zap.String("external_id", subscription.ExternalID))
	}

	return nil
}

// UpsertInvoice creates or updates an invoice from payment sync data
func (s *StripeService) UpsertInvoice(ctx context.Context, session *db.PaymentSyncSession, invoice ps.Invoice) error {
	s.logger.Debug("Upserting invoice",
		zap.String("external_id", invoice.ExternalID),
		zap.String("customer_id", invoice.CustomerID),
		zap.String("status", invoice.Status))

	// Marshal line items to JSON
	lineItemsJSON, err := json.Marshal(invoice.Lines)
	if err != nil {
		return fmt.Errorf("failed to marshal invoice line items: %w", err)
	}

	// Marshal total tax amounts to JSON
	totalTaxAmountsJSON, err := json.Marshal(invoice.TotalTaxAmounts)
	if err != nil {
		return fmt.Errorf("failed to marshal total tax amounts: %w", err)
	}

	// Marshal metadata to JSON
	metadataJSON, err := json.Marshal(invoice.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal invoice metadata: %w", err)
	}

	// Try to find corresponding customer by external ID
	var customerID *uuid.UUID
	if invoice.CustomerID != "" {
		existingCustomer, err := s.db.GetCustomerByExternalID(ctx, pgtype.Text{String: invoice.CustomerID, Valid: true})
		if err == nil {
			customerID = &existingCustomer.ID
		}
		// If customer not found, we continue without linking (customerID remains nil)
	}

	// Try to find corresponding subscription by external ID
	var subscriptionID *uuid.UUID
	if invoice.SubscriptionID != "" {
		existingSubscription, err := s.db.GetSubscriptionByExternalID(ctx, db.GetSubscriptionByExternalIDParams{
			WorkspaceID:     session.WorkspaceID,
			ExternalID:      pgtype.Text{String: invoice.SubscriptionID, Valid: true},
			PaymentProvider: pgtype.Text{String: "stripe", Valid: true},
		})
		if err == nil {
			subscriptionID = &existingSubscription.ID
		}
		// If subscription not found, we continue without linking
	}

	// Convert timestamps
	dueDate := pgtype.Timestamptz{Valid: false}
	if invoice.DueDate > 0 {
		dueDate = pgtype.Timestamptz{
			Time:  time.Unix(invoice.DueDate, 0),
			Valid: true,
		}
	}

	paidAt := pgtype.Timestamptz{Valid: false}
	if invoice.PaidAt > 0 {
		paidAt = pgtype.Timestamptz{
			Time:  time.Unix(invoice.PaidAt, 0),
			Valid: true,
		}
	}

	createdDate := time.Now() // Default to now
	// Note: ps.Invoice doesn't have CreatedAt, might need to be added to the interface
	// For now, we'll use the current time

	nextPaymentAttempt := pgtype.Timestamptz{Valid: false}
	if invoice.NextPaymentAttempt > 0 {
		nextPaymentAttempt = pgtype.Timestamptz{
			Time:  time.Unix(invoice.NextPaymentAttempt, 0),
			Valid: true,
		}
	}

	// Convert optional customer and subscription IDs to pgtype.UUID
	customerUUID := pgtype.UUID{Valid: false}
	if customerID != nil {
		customerUUID = pgtype.UUID{Bytes: *customerID, Valid: true}
	}

	subscriptionUUID := pgtype.UUID{Valid: false}
	if subscriptionID != nil {
		subscriptionUUID = pgtype.UUID{Bytes: *subscriptionID, Valid: true}
	}

	// Convert optional string fields to pgtype.Text
	externalCustomerID := pgtype.Text{String: invoice.CustomerID, Valid: invoice.CustomerID != ""}
	externalSubscriptionID := pgtype.Text{String: invoice.SubscriptionID, Valid: invoice.SubscriptionID != ""}
	collectionMethod := pgtype.Text{String: invoice.CollectionMethod, Valid: invoice.CollectionMethod != ""}
	invoicePDF := pgtype.Text{String: invoice.InvoicePDF, Valid: invoice.InvoicePDF != ""}
	hostedInvoiceURL := pgtype.Text{String: invoice.HostedInvoiceURL, Valid: invoice.HostedInvoiceURL != ""}
	chargeID := pgtype.Text{String: invoice.ChargeID, Valid: invoice.ChargeID != ""}
	paymentIntentID := pgtype.Text{String: invoice.PaymentIntentID, Valid: invoice.PaymentIntentID != ""}
	billingReason := pgtype.Text{String: invoice.BillingReason, Valid: invoice.BillingReason != ""}
	paymentProvider := pgtype.Text{String: "stripe", Valid: true}
	paymentSyncStatus := pgtype.Text{String: "synced", Valid: true}

	// Call UpsertInvoice function
	_, err = s.db.UpsertInvoice(ctx, db.UpsertInvoiceParams{
		WorkspaceID:            session.WorkspaceID,                                          // $1
		CustomerID:             customerUUID,                                                 // $2
		SubscriptionID:         subscriptionUUID,                                             // $3
		ExternalID:             invoice.ExternalID,                                           // $4
		ExternalCustomerID:     externalCustomerID,                                           // $5
		ExternalSubscriptionID: externalSubscriptionID,                                       // $6
		Status:                 invoice.Status,                                               // $7
		CollectionMethod:       collectionMethod,                                             // $8
		AmountDue:              int32(invoice.AmountDue),                                     // $9
		AmountPaid:             int32(invoice.AmountPaid),                                    // $10
		AmountRemaining:        int32(invoice.AmountRemaining),                               // $11
		Currency:               invoice.Currency,                                             // $12
		DueDate:                dueDate,                                                      // $13
		PaidAt:                 paidAt,                                                       // $14
		CreatedDate:            pgtype.Timestamptz{Time: createdDate, Valid: true},           // $15
		InvoicePdf:             invoicePDF,                                                   // $16
		HostedInvoiceUrl:       hostedInvoiceURL,                                             // $17
		ChargeID:               chargeID,                                                     // $18
		PaymentIntentID:        paymentIntentID,                                              // $19
		LineItems:              lineItemsJSON,                                                // $20
		TaxAmount:              pgtype.Int4{Int32: int32(invoice.Tax), Valid: true},          // $21
		TotalTaxAmounts:        totalTaxAmountsJSON,                                          // $22
		BillingReason:          billingReason,                                                // $23
		PaidOutOfBand:          pgtype.Bool{Bool: invoice.PaidOutOfBand, Valid: true},        // $24
		PaymentProvider:        paymentProvider,                                              // $25
		PaymentSyncStatus:      paymentSyncStatus,                                            // $26
		PaymentSyncedAt:        pgtype.Timestamptz{Time: time.Now(), Valid: true},            // $27
		AttemptCount:           pgtype.Int4{Int32: int32(invoice.AttemptCount), Valid: true}, // $28
		NextPaymentAttempt:     nextPaymentAttempt,                                           // $29
		Metadata:               metadataJSON,                                                 // $30
	})

	if err != nil {
		return fmt.Errorf("failed to upsert invoice: %w", err)
	}

	s.logger.Debug("Successfully upserted invoice",
		zap.String("external_id", invoice.ExternalID),
		zap.String("status", invoice.Status))

	return nil
}
