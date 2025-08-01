package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	dsClient "github.com/cyphera/cyphera-api/libs/go/client/delegation_server"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// SubscriptionService handles subscription business logic
type SubscriptionService struct {
	queries              db.Querier
	delegationClient     *dsClient.DelegationClient
	paymentService       *PaymentService
	customerService      *CustomerService
	invoiceService       interfaces.InvoiceService
	logger               *zap.Logger
	lastRedemptionTxHash string // Stores the transaction hash from the last successful redemption
}

// NewSubscriptionService creates a new subscription service
func NewSubscriptionService(queries db.Querier, delegationClient *dsClient.DelegationClient, paymentService *PaymentService, customerService *CustomerService, invoiceService interfaces.InvoiceService) *SubscriptionService {
	return &SubscriptionService{
		queries:          queries,
		delegationClient: delegationClient,
		paymentService:   paymentService,
		customerService:  customerService,
		invoiceService:   invoiceService,
		logger:           logger.Log,
	}
}

// WithTransaction creates a new subscription service instance with transaction-based queries
func (s *SubscriptionService) WithTransaction(tx pgx.Tx) *SubscriptionService {
	return &SubscriptionService{
		queries:          db.New(tx),
		delegationClient: s.delegationClient,
		paymentService:   s.paymentService,
		customerService:  s.customerService,
		invoiceService:   s.invoiceService,
		logger:           s.logger,
	}
}

// SubscriptionExistsError is a custom error for when a subscription already exists
type SubscriptionExistsError struct {
	Subscription *db.Subscription
}

func (e *SubscriptionExistsError) Error() string {
	return "subscription already exists for this customer and product"
}

// ProcessSubscriptionParams represents parameters for processing a subscription
type ProcessSubscriptionParams struct {
	Subscription         db.Subscription
	Product              db.Product // Product now contains pricing info
	Customer             db.Customer
	MerchantWallet       db.Wallet
	CustomerWallet       db.Wallet
	ProductToken         db.ProductsToken
	Token                db.Token
	Network              db.Network
	PaymentAmount        string
	RedemptionID         string
	PaymentDescription   string
	LastAttemptedAt      time.Time
	DelegationSignature  string
	DelegationExpiry     string
	AuthenticatedMessage string
	RedemptionAttempts   int32
	RedemptionTxHash     string
}

// ProcessSubscriptionResult represents the result of processing a subscription
type ProcessSubscriptionResult struct {
	TransactionHash string
	PaymentID       uuid.UUID
	Success         bool
	ErrorMessage    string
}

// GetSubscription retrieves a subscription by ID
func (s *SubscriptionService) GetSubscription(ctx context.Context, workspaceID, subscriptionID uuid.UUID) (*db.Subscription, error) {
	subscription, err := s.queries.GetSubscriptionWithWorkspace(ctx, db.GetSubscriptionWithWorkspaceParams{
		ID:          subscriptionID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, err
	}

	return &subscription, nil
}

// ListSubscriptions retrieves subscriptions with pagination
func (s *SubscriptionService) ListSubscriptions(ctx context.Context, workspaceID uuid.UUID, limit, offset int32) ([]responses.SubscriptionResponse, int64, error) {
	params := db.ListSubscriptionDetailsWithPaginationParams{
		WorkspaceID: workspaceID,
		Limit:       limit,
		Offset:      offset,
	}

	subscriptions, err := s.queries.ListSubscriptionDetailsWithPagination(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve subscriptions: %w", err)
	}

	// Get the total count for pagination metadata
	totalCount, err := s.queries.CountSubscriptions(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count subscriptions: %w", err)
	}

	subscriptionResponses := make([]responses.SubscriptionResponse, len(subscriptions))
	for i, sub := range subscriptions {
		subscription, err := helpers.ToSubscriptionResponse(sub)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert subscription to response: %w", err)
		}
		subscriptionResponses[i] = subscription
	}

	return subscriptionResponses, totalCount, nil
}

// ListSubscriptionsByCustomer retrieves subscriptions for a specific customer
func (s *SubscriptionService) ListSubscriptionsByCustomer(ctx context.Context, workspaceID, customerID uuid.UUID) ([]responses.SubscriptionResponse, error) {
	subscriptions, err := s.queries.ListSubscriptionsByCustomer(ctx, db.ListSubscriptionsByCustomerParams{
		CustomerID:  customerID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve customer subscriptions: %w", err)
	}

	subscriptionResponses := make([]responses.SubscriptionResponse, len(subscriptions))
	for i, sub := range subscriptions {
		subscription, err := helpers.ToSubscriptionResponseFromDBSubscription(sub)
		if err != nil {
			return nil, fmt.Errorf("failed to convert subscription to response: %w", err)
		}
		subscriptionResponses[i] = subscription
	}

	return subscriptionResponses, nil
}

// ListSubscriptionsByProduct retrieves subscriptions for a specific product
func (s *SubscriptionService) ListSubscriptionsByProduct(ctx context.Context, workspaceID, productID uuid.UUID) ([]db.Subscription, error) {
	subscriptions, err := s.queries.ListSubscriptionsByProduct(ctx, db.ListSubscriptionsByProductParams{
		ProductID:   productID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve product subscriptions: %w", err)
	}

	return subscriptions, nil
}

// UpdateSubscription updates a subscription
func (s *SubscriptionService) UpdateSubscription(ctx context.Context, subscriptionID uuid.UUID, req requests.UpdateSubscriptionRequest) (*db.Subscription, error) {
	// Check if subscription exists
	existingSubscription, err := s.queries.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("subscription not found: %w", err)
	}

	// Initialize update params with existing values
	params := db.UpdateSubscriptionParams{
		ID:                 subscriptionID,
		CustomerID:         existingSubscription.CustomerID,
		ProductID:          existingSubscription.ProductID,
		ProductTokenID:     existingSubscription.ProductTokenID,
		DelegationID:       existingSubscription.DelegationID,
		CustomerWalletID:   existingSubscription.CustomerWalletID,
		Status:             existingSubscription.Status,
		CurrentPeriodStart: existingSubscription.CurrentPeriodStart,
		CurrentPeriodEnd:   existingSubscription.CurrentPeriodEnd,
		NextRedemptionDate: existingSubscription.NextRedemptionDate,
		TotalRedemptions:   existingSubscription.TotalRedemptions,
		TotalAmountInCents: existingSubscription.TotalAmountInCents,
		Metadata:           existingSubscription.Metadata,
	}

	// Update with provided values
	if req.CustomerID != "" {
		parsedCustomerID, err := uuid.Parse(req.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("invalid customer ID format: %w", err)
		}
		params.CustomerID = parsedCustomerID
	}

	if req.ProductID != "" {
		parsedProductID, err := uuid.Parse(req.ProductID)
		if err != nil {
			return nil, fmt.Errorf("invalid product ID format: %w", err)
		}
		params.ProductID = parsedProductID
	}

	if req.ProductTokenID != "" {
		parsedProductTokenID, err := uuid.Parse(req.ProductTokenID)
		if err != nil {
			return nil, fmt.Errorf("invalid product token ID format: %w", err)
		}
		params.ProductTokenID = parsedProductTokenID
	}

	if req.DelegationID != "" {
		parsedDelegationID, err := uuid.Parse(req.DelegationID)
		if err != nil {
			return nil, fmt.Errorf("invalid delegation ID format: %w", err)
		}
		params.DelegationID = parsedDelegationID
	}

	if req.CustomerWalletID != "" {
		parsedCustomerWalletID, err := uuid.Parse(req.CustomerWalletID)
		if err != nil {
			return nil, fmt.Errorf("invalid customer wallet ID format: %w", err)
		}
		params.CustomerWalletID = pgtype.UUID{
			Bytes: parsedCustomerWalletID,
			Valid: true,
		}
	}

	if req.Status != "" {
		switch req.Status {
		case string(db.SubscriptionStatusActive), string(db.SubscriptionStatusCanceled), string(db.SubscriptionStatusExpired), string(db.SubscriptionStatusSuspended), string(db.SubscriptionStatusFailed):
			params.Status = db.SubscriptionStatus(req.Status)
		default:
			return nil, fmt.Errorf("invalid status value: %s", req.Status)
		}
	}

	if req.StartDate > 0 {
		params.CurrentPeriodStart = pgtype.Timestamptz{
			Time:  time.Unix(req.StartDate, 0),
			Valid: true,
		}
	}

	if req.EndDate > 0 {
		params.CurrentPeriodEnd = pgtype.Timestamptz{
			Time:  time.Unix(req.EndDate, 0),
			Valid: true,
		}
	}

	if req.NextRedemption > 0 {
		params.NextRedemptionDate = pgtype.Timestamptz{
			Time:  time.Unix(req.NextRedemption, 0),
			Valid: true,
		}
	}

	if req.Metadata != nil {
		params.Metadata = req.Metadata
	}

	// Update subscription
	subscription, err := s.queries.UpdateSubscription(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	return &subscription, nil
}

// DeleteSubscription deletes a subscription
func (s *SubscriptionService) DeleteSubscription(ctx context.Context, workspaceID, subscriptionID uuid.UUID) error {
	subscription, err := s.queries.GetSubscriptionWithWorkspace(ctx, db.GetSubscriptionWithWorkspaceParams{
		ID:          subscriptionID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	if subscription.WorkspaceID != workspaceID {
		return fmt.Errorf("subscription does not belong to this workspace")
	}

	if subscription.Status != db.SubscriptionStatusCanceled && subscription.Status != db.SubscriptionStatusExpired {
		return fmt.Errorf("subscription is not canceled or expired")
	}

	err = s.queries.DeleteSubscription(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	return nil
}

// StoreDelegationData stores delegation data in the database
func (s *SubscriptionService) StoreDelegationData(ctx context.Context, tx pgx.Tx, params params.StoreDelegationDataParams) (*db.DelegationDatum, error) {
	var qtx db.Querier
	if tx != nil {
		qtx = db.New(tx)
	} else {
		qtx = s.queries
	}

	s.logger.Info("Storing delegation data",
		zap.String("delegate", params.Delegate),
		zap.String("delegator", params.Delegator))

	caveatsJSON, err := json.Marshal(params.Caveats)
	if err != nil {
		s.logger.Error("Failed to marshal caveats", zap.Error(err))
		return nil, fmt.Errorf("failed to marshal caveats: %w", err)
	}

	delegationData, err := qtx.CreateDelegationData(ctx, db.CreateDelegationDataParams{
		Delegate:  params.Delegate,
		Delegator: params.Delegator,
		Authority: params.Authority,
		Caveats:   caveatsJSON,
		Salt:      params.Salt,
		Signature: params.Signature,
	})
	if err != nil {
		s.logger.Error("Failed to create delegation data", zap.Error(err))
		return nil, fmt.Errorf("failed to create delegation data: %w", err)
	}

	s.logger.Info("Delegation data stored successfully",
		zap.String("delegation_id", delegationData.ID.String()))

	return &delegationData, nil
}

// CreateSubscription creates a new subscription
func (s *SubscriptionService) CreateSubscription(ctx context.Context, tx pgx.Tx, params params.CreateSubscriptionParams) (*db.Subscription, error) {
	var qtx db.Querier
	if tx != nil {
		qtx = db.New(tx)
	} else {
		qtx = s.queries
	}

	s.logger.Info("Creating subscription",
		zap.String("customer_id", params.Customer.ID.String()),
		zap.String("product_id", params.ProductID.String()))

	// Check if subscription already exists
	existingSubscriptions, err := qtx.ListSubscriptionsByCustomer(ctx, db.ListSubscriptionsByCustomerParams{
		CustomerID:  params.Customer.ID,
		WorkspaceID: params.WorkspaceID,
	})
	if err != nil {
		s.logger.Error("Failed to check existing subscriptions", zap.Error(err))
		return nil, fmt.Errorf("failed to check existing subscriptions: %w", err)
	}

	// Check for existing active subscription for the same price
	for i, existingSub := range existingSubscriptions {
		if existingSub.Status == db.SubscriptionStatusActive {
			// Note: We can't check PriceID directly on the subscription model, would need to join with price table
			// For now, just check if any active subscription exists for this product
			s.logger.Warn("Active subscription already exists",
				zap.String("subscription_id", existingSub.ID.String()))
			return nil, &SubscriptionExistsError{Subscription: &existingSubscriptions[i]}
		}
	}

	// Create subscription
	subscription, err := qtx.CreateSubscription(ctx, db.CreateSubscriptionParams{
		WorkspaceID:        params.WorkspaceID,
		CustomerID:         params.Customer.ID,
		ProductID:          params.ProductID,
		ProductTokenID:     params.ProductTokenID,
		TokenAmount:        int32(params.TokenAmount),
		DelegationID:       params.DelegationData.ID,
		CustomerWalletID:   pgtype.UUID{Bytes: params.CustomerWallet.ID, Valid: true},
		Status:             db.SubscriptionStatusActive,
		CurrentPeriodStart: pgtype.Timestamptz{Time: params.PeriodStart, Valid: true},
		CurrentPeriodEnd:   pgtype.Timestamptz{Time: params.PeriodEnd, Valid: true},
		NextRedemptionDate: pgtype.Timestamptz{Time: params.NextRedemption, Valid: true},
		TotalRedemptions:   0,
		TotalAmountInCents: 0,
		Metadata: func() []byte {
			// Get the product for metadata
			product := params.Product

			metadata := map[string]interface{}{
				"product_id":   product.ID.String(),
				"token_amount": params.TokenAmount,
				"term_length":  product.TermLength.Int32,
				"interval_type": func() string {
					if product.IntervalType.Valid {
						return string(product.IntervalType.IntervalType)
					}
					return ""
				}(),
				"unit_amount_in_pennies": product.UnitAmountInPennies,
				"wallet_address":         params.CustomerWallet.WalletAddress,
				"delegation_id":          params.DelegationData.ID.String(),
			}
			metadataBytes, err := json.Marshal(metadata)
			if err != nil {
				s.logger.Error("Failed to marshal subscription metadata", zap.Error(err))
				return []byte("{}")
			}
			return metadataBytes
		}(),
		PaymentProvider: pgtype.Text{String: "", Valid: false},
	})
	if err != nil {
		s.logger.Error("Failed to create subscription", zap.Error(err))
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	s.logger.Info("Subscription created successfully",
		zap.String("subscription_id", subscription.ID.String()))

	// Create subscription line items
	// First, create the base product line item
	baseProduct := params.Product

	// Create base product line item
	_, err = qtx.CreateSubscriptionLineItem(ctx, db.CreateSubscriptionLineItemParams{
		SubscriptionID:       subscription.ID,
		ProductID:            baseProduct.ID,
		LineItemType:         "base",
		Quantity:             1,
		UnitAmountInPennies:  baseProduct.UnitAmountInPennies,
		Currency:             baseProduct.Currency,
		PriceType:            baseProduct.PriceType,
		IntervalType:         baseProduct.IntervalType,
		TotalAmountInPennies: baseProduct.UnitAmountInPennies,
		IsActive:             pgtype.Bool{Bool: true, Valid: true},
		Metadata:             []byte("{}"),
	})
	if err != nil {
		s.logger.Error("Failed to create base product line item", zap.Error(err))
		return nil, fmt.Errorf("failed to create base product line item: %w", err)
	}

	// Create addon line items if provided
	if params.Addons != nil && len(params.Addons) > 0 {
		for _, addon := range params.Addons {
			// Get addon product details
			addonProduct, err := qtx.GetProductWithoutWorkspaceId(ctx, addon.ProductID)
			if err != nil {
				s.logger.Error("Failed to get addon product",
					zap.String("addon_product_id", addon.ProductID.String()),
					zap.Error(err))
				return nil, fmt.Errorf("failed to get addon product: %w", err)
			}

			// Validate addon is actually an addon type
			if !addonProduct.ProductType.Valid || addonProduct.ProductType.String != "addon" {
				return nil, fmt.Errorf("product %s is not an addon type", addon.ProductID.String())
			}

			// Create addon line item
			totalAmount := addonProduct.UnitAmountInPennies * addon.Quantity
			_, err = qtx.CreateSubscriptionLineItem(ctx, db.CreateSubscriptionLineItemParams{
				SubscriptionID:       subscription.ID,
				ProductID:            addon.ProductID,
				LineItemType:         "addon",
				Quantity:             addon.Quantity,
				UnitAmountInPennies:  addonProduct.UnitAmountInPennies,
				Currency:             addonProduct.Currency,
				PriceType:            addonProduct.PriceType,
				IntervalType:         addonProduct.IntervalType,
				TotalAmountInPennies: totalAmount,
				IsActive:             pgtype.Bool{Bool: true, Valid: true},
				Metadata:             []byte("{}"),
			})
			if err != nil {
				s.logger.Error("Failed to create addon line item",
					zap.String("addon_product_id", addon.ProductID.String()),
					zap.Error(err))
				return nil, fmt.Errorf("failed to create addon line item: %w", err)
			}
		}
	}

	// Update subscription total amount based on line items
	totalRow, err := qtx.CalculateSubscriptionTotal(ctx, subscription.ID)
	if err != nil {
		s.logger.Error("Failed to calculate subscription total", zap.Error(err))
		// Don't fail the whole operation, just log the error
	} else {
		// Update the subscription's total amount
		// The query returns BIGINT which is int64 in Go
		if totalAmount, ok := totalRow.(int64); ok {
			subscription.TotalAmountInCents = int32(totalAmount)
		}
	}

	return &subscription, nil
}

// ProcessInitialRedemption executes the initial token redemption for a new subscription
func (s *SubscriptionService) ProcessInitialRedemption(ctx context.Context, tx pgx.Tx, redemptionParams params.InitialRedemptionParams) (*db.Subscription, error) {
	var qtx db.Querier
	if tx != nil {
		qtx = db.New(tx)
	} else {
		qtx = s.queries
	}

	s.logger.Info("Processing initial redemption",
		zap.String("subscription_id", redemptionParams.Subscription.ID.String()),
		zap.String("customer_id", redemptionParams.Customer.ID.String()))

	// Prepare delegation data for redemption
	caveatsJSON, err := json.Marshal(redemptionParams.DelegationData.Caveats)
	if err != nil {
		s.logger.Error("Failed to marshal caveats", zap.Error(err))
		return nil, fmt.Errorf("failed to marshal caveats: %w", err)
	}

	delegationData := dsClient.DelegationData{
		Delegate:  redemptionParams.DelegationData.Delegate,
		Delegator: redemptionParams.DelegationData.Delegator,
		Authority: redemptionParams.DelegationData.Authority,
		Caveats:   caveatsJSON,
		Salt:      redemptionParams.DelegationData.Salt,
		Signature: redemptionParams.DelegationData.Signature,
	}

	delegationBytes, err := json.Marshal(delegationData)
	if err != nil {
		s.logger.Error("Failed to marshal delegation data", zap.Error(err))
		return nil, fmt.Errorf("failed to marshal delegation data: %w", err)
	}

	// Create execution object for redemption
	executionObject := dsClient.ExecutionObject{
		MerchantAddress:      redemptionParams.MerchantWallet.WalletAddress,
		TokenContractAddress: redemptionParams.Token.ContractAddress,
		TokenAmount:          redemptionParams.TokenAmount,
		TokenDecimals:        redemptionParams.Token.Decimals,
		ChainID:              uint32(redemptionParams.Network.ChainID),
		NetworkName:          redemptionParams.Network.Name,
	}

	// Execute the redemption
	if s.delegationClient == nil {
		return nil, fmt.Errorf("delegation client is not configured")
	}
	txHash, err := s.delegationClient.RedeemDelegation(ctx, delegationBytes, executionObject)
	if err != nil {
		s.logger.Error("Delegation redemption failed",
			zap.Error(err),
			zap.String("subscription_id", redemptionParams.Subscription.ID.String()))
		return nil, fmt.Errorf("delegation redemption failed: %w", err)
	}

	// Store the transaction hash for future reference
	s.lastRedemptionTxHash = txHash

	// Record successful redemption event
	eventMetadata := map[string]interface{}{
		"product_id":        redemptionParams.Product.ID.String(),
		"product_name":      redemptionParams.Product.Name,
		"product_id_old":    redemptionParams.Product.ID.String(),
		"price_type":        string(redemptionParams.Product.PriceType),
		"product_token_id":  redemptionParams.ProductToken.ID.String(),
		"token_symbol":      redemptionParams.ProductToken.TokenSymbol,
		"network_name":      redemptionParams.ProductToken.NetworkName,
		"wallet_address":    redemptionParams.CustomerWallet.WalletAddress,
		"customer_id":       redemptionParams.Customer.ID.String(),
		"redemption_time":   time.Now().Unix(),
		"subscription_type": string(redemptionParams.Product.PriceType),
		"tx_hash":           txHash,
	}

	metadataBytes, err := json.Marshal(eventMetadata)
	if err != nil {
		s.logger.Error("Failed to marshal event metadata", zap.Error(err))
		metadataBytes = []byte("{}")
	}

	subscriptionEvent, eventErr := qtx.CreateSubscriptionEvent(ctx, db.CreateSubscriptionEventParams{
		SubscriptionID:  redemptionParams.Subscription.ID,
		EventType:       db.SubscriptionEventTypeRedeemed,
		OccurredAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
		TransactionHash: pgtype.Text{String: txHash, Valid: true},
		AmountInCents:   redemptionParams.Product.UnitAmountInPennies,
		Metadata:        metadataBytes,
	})
	if eventErr != nil {
		s.logger.Error("Failed to record successful redemption event",
			zap.Error(eventErr),
			zap.String("subscription_id", redemptionParams.Subscription.ID.String()))
	} else {
		// Create payment record for the successful redemption
		s.logger.Info("Creating payment record for subscription redemption",
			zap.String("subscription_id", redemptionParams.Subscription.ID.String()),
			zap.String("event_id", subscriptionEvent.ID.String()),
			zap.String("transaction_hash", txHash))

		// Create payment record using a transaction-aware payment service
		// We need to create a PaymentService instance with transaction-aware queries
		paymentServiceTx := &PaymentService{
			queries:               qtx,
			logger:                s.logger,
			gasFeeService:         nil, // Not needed for basic payment creation
			taxService:            nil, // Not needed for basic payment creation
			discountService:       nil, // Not needed for basic payment creation
			exchangeRateService:   nil, // Not needed for basic payment creation
			gasSponsorshipService: nil, // Not needed for basic payment creation
		}

		_, paymentErr := paymentServiceTx.CreatePaymentFromSubscriptionEvent(ctx, params.CreatePaymentFromSubscriptionEventParams{
			SubscriptionEvent: &subscriptionEvent,
			Subscription:      &redemptionParams.Subscription,
			Product:           &redemptionParams.Product,
			Customer:          &redemptionParams.Customer,
			TransactionHash:   txHash,
			NetworkID:         redemptionParams.Network.ID,
			TokenID:           redemptionParams.Token.ID,
			CryptoAmount:      fmt.Sprintf("%.*f", int(redemptionParams.Token.Decimals), float64(redemptionParams.TokenAmount)/math.Pow(10, float64(redemptionParams.Token.Decimals))), // Convert based on token decimals
			// Note: ExchangeRate and gas fees would need additional data from external sources
		})
		if paymentErr != nil {
			s.logger.Error("Failed to create payment record for subscription redemption",
				zap.Error(paymentErr),
				zap.String("subscription_id", redemptionParams.Subscription.ID.String()),
				zap.String("transaction_hash", txHash))
		} else {
			s.logger.Info("Payment record created successfully for subscription redemption",
				zap.String("subscription_id", redemptionParams.Subscription.ID.String()),
				zap.String("transaction_hash", txHash))

			// Create invoice for the initial subscription payment
			if s.invoiceService != nil {
				s.logger.Info("Creating invoice for initial subscription payment",
					zap.String("subscription_id", redemptionParams.Subscription.ID.String()),
					zap.String("customer_id", redemptionParams.Customer.ID.String()))

				// Skip invoice generation during transaction - line items not yet committed
				// TODO: Create invoice after transaction commits
				invoice, invoiceErr := (*responses.InvoiceResponse)(nil), error(nil)
				invoiceErr = fmt.Errorf("invoice creation skipped during transaction")

				if invoiceErr != nil {
					s.logger.Info("Skipping invoice creation during initial redemption",
						zap.String("reason", "transaction not yet committed"),
						zap.String("subscription_id", redemptionParams.Subscription.ID.String()))
				} else {
					s.logger.Info("Invoice created successfully for subscription",
						zap.String("subscription_id", redemptionParams.Subscription.ID.String()),
						zap.String("invoice_id", invoice.ID.String()),
						zap.String("invoice_status", invoice.Status))

					// Mark the invoice as paid since payment was already processed
					if invoice.Status == "open" {
						_, markPaidErr := s.invoiceService.MarkInvoicePaid(ctx, 
							redemptionParams.Product.WorkspaceID,
							invoice.ID)
						
						if markPaidErr != nil {
							s.logger.Error("Failed to mark invoice as paid",
								zap.Error(markPaidErr),
								zap.String("invoice_id", invoice.ID.String()))
						} else {
							s.logger.Info("Invoice marked as paid",
								zap.String("invoice_id", invoice.ID.String()))
						}
					}
				}
			}
		}
	}

	// Calculate next redemption date
	var nextRedemptionDate pgtype.Timestamptz
	if redemptionParams.Product.PriceType == db.PriceTypeRecurring {
		intervalType := ""
		if redemptionParams.Product.IntervalType.Valid {
			intervalType = string(redemptionParams.Product.IntervalType.IntervalType)
		}
		nextDate := helpers.CalculateNextRedemption(intervalType, time.Now())
		nextRedemptionDate = pgtype.Timestamptz{
			Time:  nextDate,
			Valid: true,
		}
	} else {
		nextRedemptionDate = pgtype.Timestamptz{
			Valid: false,
		}
	}

	// Update subscription with redemption details
	updatedSubscription, err := qtx.IncrementSubscriptionRedemption(ctx, db.IncrementSubscriptionRedemptionParams{
		ID:                 redemptionParams.Subscription.ID,
		TotalAmountInCents: redemptionParams.Product.UnitAmountInPennies,
		NextRedemptionDate: nextRedemptionDate,
	})
	if err != nil {
		s.logger.Error("Failed to update subscription redemption details",
			zap.Error(err),
			zap.String("subscription_id", redemptionParams.Subscription.ID.String()))
		return &redemptionParams.Subscription, err
	}

	// Update wallet usage time
	_, walletErr := qtx.UpdateCustomerWalletUsageTime(ctx, redemptionParams.CustomerWallet.ID)
	if walletErr != nil {
		s.logger.Warn("Failed to update wallet last used timestamp",
			zap.Error(walletErr),
			zap.String("wallet_id", redemptionParams.CustomerWallet.ID.String()))
	}

	s.logger.Info("Initial redemption successful",
		zap.String("subscription_id", redemptionParams.Subscription.ID.String()),
		zap.String("transaction_hash", txHash),
		zap.Int32("amount_in_cents", int32(redemptionParams.Product.UnitAmountInPennies)))

	return &updatedSubscription, nil
}

// CreateSubscriptionWithDelegation creates a subscription with delegation in a transaction
func (s *SubscriptionService) CreateSubscriptionWithDelegation(ctx context.Context, tx pgx.Tx, createParams params.CreateSubscriptionWithDelegationParams) (*params.SubscriptionCreationResult, error) {
	s.logger.Info("Creating subscription with delegation",
		zap.String("product_id", createParams.Product.ID.String()),
		zap.String("subscriber_address", createParams.SubscriberAddress))

	var result params.SubscriptionCreationResult

	// Process customer and wallet
	customer, customerWallet, err := s.customerService.ProcessCustomerAndWallet(ctx, tx, params.ProcessCustomerWalletParams{
		WalletAddress: createParams.SubscriberAddress,
		WorkspaceID:   createParams.Product.WorkspaceID,
		ProductID:     createParams.Product.ID,
		NetworkType:   string(createParams.Network.NetworkType),
	})
	if err != nil {
		s.logger.Error("Failed to process customer and wallet", zap.Error(err))
		return nil, fmt.Errorf("failed to process customer and wallet: %w", err)
	}

	result.Customer = customer
	result.CustomerWallet = customerWallet

	// Store delegation data
	delegationData, err := s.StoreDelegationData(ctx, tx, createParams.DelegationData)
	if err != nil {
		s.logger.Error("Failed to store delegation data", zap.Error(err))
		return nil, fmt.Errorf("failed to store delegation data: %w", err)
	}

	// Calculate subscription periods
	periodStart, periodEnd, nextRedemption := helpers.CalculateSubscriptionPeriods(createParams.Product)

	// Create subscription
	subscription, err := s.CreateSubscription(ctx, tx, params.CreateSubscriptionParams{
		Customer:       *customer,
		CustomerWallet: *customerWallet,
		WorkspaceID:    createParams.Product.WorkspaceID,
		ProductID:      createParams.Product.ID,
		ProductTokenID: createParams.ProductTokenID,
		Product:        createParams.Product,
		TokenAmount:    createParams.TokenAmount,
		DelegationData: *delegationData,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		NextRedemption: nextRedemption,
	})
	if err != nil {
		s.logger.Error("Failed to create subscription", zap.Error(err))
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	result.Subscription = subscription

	// Create subscription event
	eventMetadata := map[string]interface{}{
		"product_name":   createParams.Product.Name,
		"price_type":     string(createParams.Product.PriceType),
		"wallet_address": customerWallet.WalletAddress,
		"network_type":   createParams.Network.Type,
	}

	eventMetadataBytes, err := json.Marshal(eventMetadata)
	if err != nil {
		s.logger.Error("Failed to marshal event metadata", zap.Error(err))
		eventMetadataBytes = []byte("{}")
	}

	qtx := db.New(tx)
	_, err = qtx.CreateSubscriptionEvent(ctx, db.CreateSubscriptionEventParams{
		SubscriptionID: subscription.ID,
		EventType:      db.SubscriptionEventTypeCreated,
		OccurredAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		AmountInCents:  createParams.Product.UnitAmountInPennies,
		Metadata:       eventMetadataBytes,
	})
	if err != nil {
		s.logger.Error("Failed to create subscription event", zap.Error(err))
		return nil, fmt.Errorf("failed to create subscription event: %w", err)
	}

	// NOTE: Transaction should be committed by the caller before performing initial redemption
	// The initial redemption needs to happen after the transaction commits

	// Set a flag to indicate initial redemption should be performed after commit
	updatedSubscription, err := s.ProcessInitialRedemption(ctx, tx, params.InitialRedemptionParams{
		Customer:       *customer,
		CustomerWallet: *customerWallet,
		Subscription:   *subscription,
		Product:        createParams.Product,
		ProductToken:   createParams.ProductToken,
		DelegationData: createParams.DelegationData,
		MerchantWallet: createParams.MerchantWallet,
		Token:          createParams.Token,
		Network:        createParams.Network,
		TokenAmount:    createParams.TokenAmount,
	})
	if err != nil {
		s.logger.Error("Initial redemption failed",
			zap.Error(err),
			zap.String("subscription_id", subscription.ID.String()))

		// Update subscription status to failed
		qtx := db.New(tx)
		_, updateErr := qtx.UpdateSubscriptionStatus(ctx, db.UpdateSubscriptionStatusParams{
			ID:     subscription.ID,
			Status: db.SubscriptionStatusFailed,
		})
		if updateErr != nil {
			s.logger.Error("Failed to update subscription status after redemption failure",
				zap.Error(updateErr),
				zap.String("subscription_id", subscription.ID.String()))
		}

		// Soft delete the subscription
		deleteErr := qtx.DeleteSubscription(ctx, subscription.ID)
		if deleteErr != nil {
			s.logger.Error("Failed to soft-delete subscription after redemption failure",
				zap.Error(deleteErr),
				zap.String("subscription_id", subscription.ID.String()))
		}

		return nil, fmt.Errorf("initial redemption failed: %w", err)
	}

	result.Subscription = updatedSubscription
	result.TransactionHash = s.lastRedemptionTxHash
	result.InitialRedemption = true

	s.logger.Info("Subscription created successfully with initial redemption",
		zap.String("subscription_id", subscription.ID.String()),
		zap.String("transaction_hash", result.TransactionHash))

	return &result, nil
}

// ProcessDueSubscriptions finds and processes all subscriptions that are due for redemption
// This method expects to be called within a transaction context
func (s *SubscriptionService) ProcessDueSubscriptions(ctx context.Context) (*responses.ProcessDueSubscriptionsResult, error) {
	s.logger.Info("Processing due subscriptions")
	result := &responses.ProcessDueSubscriptionsResult{}
	now := time.Now()

	// Query for subscriptions due for redemption
	nowPgType := pgtype.Timestamptz{Time: now, Valid: true}
	subscriptions, err := s.queries.ListSubscriptionsDueForRedemption(ctx, nowPgType)
	if err != nil {
		s.logger.Error("Failed to fetch subscriptions due for redemption", zap.Error(err))
		return nil, fmt.Errorf("failed to fetch subscriptions: %w", err)
	}

	result.ProcessedCount = len(subscriptions)
	if result.ProcessedCount == 0 {
		s.logger.Info("No subscriptions found due for renewal")
		return result, nil
	}

	s.logger.Info("Found subscriptions due for redemption",
		zap.Int("count", result.ProcessedCount))

	// Process each subscription
	for i, subscription := range subscriptions {
		s.logger.Info("Processing subscription",
			zap.Int("current", i+1),
			zap.Int("total", result.ProcessedCount),
			zap.String("subscription_id", subscription.ID.String()),
			zap.String("status", string(subscription.Status)))

		// Skip non-processable statuses
		if !(subscription.Status == db.SubscriptionStatusActive || subscription.Status == db.SubscriptionStatusOverdue) {
			s.logger.Info("Skipping subscription with non-processable status",
				zap.String("subscription_id", subscription.ID.String()),
				zap.String("status", string(subscription.Status)))
			continue
		}

		// Process the subscription
		err := s.processSingleSubscription(ctx, s.queries, subscription)
		if err != nil {
			s.logger.Error("Failed to process subscription",
				zap.String("subscription_id", subscription.ID.String()),
				zap.Error(err))
			result.FailedCount++
		} else {
			result.SuccessfulCount++
		}
	}

	s.logger.Info("Completed processing due subscriptions",
		zap.Int("total", result.ProcessedCount),
		zap.Int("succeeded", result.SuccessfulCount),
		zap.Int("failed", result.FailedCount))

	return result, nil
}

// processSingleSubscription processes a single subscription for redemption
func (s *SubscriptionService) processSingleSubscription(ctx context.Context, qtx db.Querier, subscription db.ListSubscriptionsDueForRedemptionRow) error {
	var subEvent db.SubscriptionEvent

	// Re-fetch subscription for idempotency check
	currentSub, err := qtx.GetSubscription(ctx, subscription.ID)
	if err != nil {
		return fmt.Errorf("failed to re-fetch subscription: %w", err)
	}

	// Check if already processed
	if currentSub.Status == db.SubscriptionStatusCompleted {
		s.logger.Info("Subscription already completed",
			zap.String("subscription_id", currentSub.ID.String()))
		return nil
	}

	// Get required data
	product, err := qtx.GetProductWithoutWorkspaceId(ctx, subscription.ProductID)
	if err != nil {
		return fmt.Errorf("failed to get product: %w", err)
	}

	// Price is now embedded in product - no need to fetch separately
	price := product // Use product as price for code that expects price variable

	customer, err := qtx.GetCustomer(ctx, subscription.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to get customer: %w", err)
	}

	// Get delegation data
	delegationData, err := qtx.GetDelegationData(ctx, subscription.DelegationID)
	if err != nil {
		return fmt.Errorf("failed to get delegation data: %w", err)
	}

	// Get merchant wallet for the product
	merchantWallet, err := qtx.GetWalletByID(ctx, db.GetWalletByIDParams{
		ID:          product.WalletID,
		WorkspaceID: product.WorkspaceID,
	})
	if err != nil {
		return fmt.Errorf("failed to get merchant wallet: %w", err)
	}

	customerWallet, err := qtx.GetCustomerWallet(ctx, subscription.CustomerWalletID.Bytes)
	if err != nil {
		return fmt.Errorf("failed to get customer wallet: %w", err)
	}

	// Get product token info
	productToken, err := qtx.GetProductToken(ctx, subscription.ProductTokenID)
	if err != nil {
		return fmt.Errorf("failed to get product token: %w", err)
	}

	// Prepare for redemption
	caveatsJSON, err := json.Marshal(delegationData.Caveats)
	if err != nil {
		return fmt.Errorf("failed to marshal caveats: %w", err)
	}

	delegationForRedemption := dsClient.DelegationData{
		Delegate:  delegationData.Delegate,
		Delegator: delegationData.Delegator,
		Authority: delegationData.Authority,
		Caveats:   caveatsJSON,
		Salt:      delegationData.Salt,
		Signature: delegationData.Signature,
	}

	delegationBytes, err := json.Marshal(delegationForRedemption)
	if err != nil {
		return fmt.Errorf("failed to marshal delegation: %w", err)
	}

	// Get token details
	token, err := qtx.GetToken(ctx, productToken.TokenID)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// Create execution object
	executionObject := dsClient.ExecutionObject{
		MerchantAddress:      merchantWallet.WalletAddress,
		TokenContractAddress: token.ContractAddress,
		TokenAmount:          int64(subscription.TokenAmount),
		TokenDecimals:        token.Decimals,
		ChainID:              uint32(productToken.ChainID),
		NetworkName:          productToken.NetworkName,
	}

	// Execute redemption
	if s.delegationClient == nil {
		return fmt.Errorf("delegation client is not configured")
	}
	txHash, err := s.delegationClient.RedeemDelegation(ctx, delegationBytes, executionObject)
	if err != nil {
		// Update subscription status to overdue on failure
		_, updateErr := qtx.UpdateSubscriptionStatus(ctx, db.UpdateSubscriptionStatusParams{
			ID:     subscription.ID,
			Status: db.SubscriptionStatusOverdue,
		})
		if updateErr != nil {
			s.logger.Error("Failed to update subscription status",
				zap.Error(updateErr),
				zap.String("subscription_id", subscription.ID.String()))
		}
		return fmt.Errorf("redemption failed: %w", err)
	}

	// Note: Subscription event will be created later before payment creation

	// Update subscription with proper term length validation
	var nextRedemptionDate pgtype.Timestamptz
	if price.PriceType == db.PriceTypeRecurring {
		// CRITICAL BUG FIX: Check if subscription has reached its term limit
		// currentSub.TotalRedemptions will be incremented by IncrementSubscriptionRedemption below
		// so we check if the NEXT redemption would exceed the limit
		if currentSub.TotalRedemptions+1 >= price.TermLength.Int32 {
			s.logger.Info("Subscription reached maximum periods, marking as completed",
				zap.String("subscription_id", subscription.ID.String()),
				zap.Int32("current_redemptions", currentSub.TotalRedemptions),
				zap.Int32("max_periods", price.TermLength.Int32))

			// Mark as completed - reached maximum periods
			_, err = qtx.UpdateSubscriptionStatus(ctx, db.UpdateSubscriptionStatusParams{
				ID:     subscription.ID,
				Status: db.SubscriptionStatusCompleted,
			})
			if err != nil {
				s.logger.Error("Failed to mark subscription as completed after reaching term limit",
					zap.Error(err),
					zap.String("subscription_id", subscription.ID.String()))
			}

			// No next redemption date - subscription is complete
			nextRedemptionDate = pgtype.Timestamptz{Valid: false}
		} else {
			// Continue with next redemption - subscription still has periods remaining
			intervalType := ""
			if price.IntervalType.Valid {
				intervalType = string(price.IntervalType.IntervalType)
			}
			nextDate := helpers.CalculateNextRedemption(intervalType, time.Now())
			nextRedemptionDate = pgtype.Timestamptz{
				Time:  nextDate,
				Valid: true,
			}

			s.logger.Info("Subscription continuing to next period",
				zap.String("subscription_id", subscription.ID.String()),
				zap.Int32("current_redemptions", currentSub.TotalRedemptions),
				zap.Int32("max_periods", price.TermLength.Int32),
				zap.Time("next_redemption", nextDate))
		}
	} else {
		// One-time price, mark as completed
		_, err = qtx.UpdateSubscriptionStatus(ctx, db.UpdateSubscriptionStatusParams{
			ID:     subscription.ID,
			Status: db.SubscriptionStatusCompleted,
		})
		if err != nil {
			s.logger.Error("Failed to mark one-time subscription as completed",
				zap.Error(err),
				zap.String("subscription_id", subscription.ID.String()))
		}

		// No next redemption date for one-time subscriptions
		nextRedemptionDate = pgtype.Timestamptz{Valid: false}
	}

	_, err = qtx.IncrementSubscriptionRedemption(ctx, db.IncrementSubscriptionRedemptionParams{
		ID:                 subscription.ID,
		TotalAmountInCents: price.UnitAmountInPennies,
		NextRedemptionDate: nextRedemptionDate,
	})
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	// Create subscription event in database first (similar to ProcessInitialRedemption)
	eventMetadata := map[string]interface{}{
		"product_id":        product.ID.String(),
		"product_name":      product.Name,
		"product_id_old":    product.ID.String(),
		"price_type":        string(product.PriceType),
		"product_token_id":  productToken.ID.String(),
		"token_symbol":      productToken.TokenSymbol,
		"network_name":      productToken.NetworkName,
		"wallet_address":    customerWallet.WalletAddress,
		"customer_id":       customer.ID.String(),
		"redemption_time":   time.Now().Unix(),
		"subscription_type": string(product.PriceType),
		"tx_hash":           txHash,
	}

	metadataBytes, err := json.Marshal(eventMetadata)
	if err != nil {
		s.logger.Error("Failed to marshal event metadata", zap.Error(err))
		metadataBytes = []byte("{}")
	}

	// Actually create the subscription event in the database
	subEvent, err = qtx.CreateSubscriptionEvent(ctx, db.CreateSubscriptionEventParams{
		SubscriptionID:  subscription.ID,
		EventType:       db.SubscriptionEventTypeRedeemed,
		OccurredAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
		TransactionHash: pgtype.Text{String: txHash, Valid: true},
		AmountInCents:   product.UnitAmountInPennies,
		Metadata:        metadataBytes,
	})
	if err != nil {
		s.logger.Error("Failed to create subscription event",
			zap.Error(err),
			zap.String("subscription_id", subscription.ID.String()))
		return fmt.Errorf("failed to create subscription event: %w", err)
	}

	_, err = s.paymentService.CreatePaymentFromSubscriptionEvent(ctx, params.CreatePaymentFromSubscriptionEventParams{
		SubscriptionEvent: &subEvent,
		Subscription:      &currentSub,
		Product:           &product,
		Customer:          &customer,
		TransactionHash:   txHash,
		NetworkID:         productToken.NetworkID,
		TokenID:           productToken.TokenID,
		CryptoAmount:      fmt.Sprintf("%d", subscription.TokenAmount),
		ExchangeRate:      "1.0", // TODO: Get actual exchange rate
		GasFeeUSDCents:    0,     // TODO: Calculate gas fee
		GasSponsored:      false,
	})
	if err != nil {
		s.logger.Error("Failed to create payment record",
			zap.Error(err),
			zap.String("subscription_id", subscription.ID.String()))
	}

	// Check if subscription was completed and create completion event
	updatedSub, err := qtx.GetSubscription(ctx, subscription.ID)
	if err != nil {
		s.logger.Error("Failed to get updated subscription for completion check",
			zap.Error(err),
			zap.String("subscription_id", subscription.ID.String()))
	} else if updatedSub.Status == db.SubscriptionStatusCompleted {
		// Create completion event after redemption and payment
		completionReason := "term_limit_reached"
		if price.PriceType == db.PriceTypeOneTime {
			completionReason = "one_time_purchase"
		}

		completionMetadata := map[string]interface{}{
			"product_id":        product.ID.String(),
			"product_name":      product.Name,
			"product_id_old":    price.ID.String(),
			"total_redemptions": updatedSub.TotalRedemptions,
			"max_periods":       price.TermLength.Int32,
			"completion_reason": completionReason,
			"final_tx_hash":     txHash,
		}

		completionMetadataBytes, err := json.Marshal(completionMetadata)
		if err != nil {
			s.logger.Error("Failed to marshal completion metadata", zap.Error(err))
			completionMetadataBytes = []byte("{}")
		}

		_, err = qtx.CreateSubscriptionEvent(ctx, db.CreateSubscriptionEventParams{
			SubscriptionID:  subscription.ID,
			EventType:       db.SubscriptionEventTypeCompleted,
			OccurredAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
			TransactionHash: pgtype.Text{String: txHash, Valid: true},
			AmountInCents:   0, // No additional amount for completion event
			Metadata:        completionMetadataBytes,
		})
		if err != nil {
			s.logger.Error("Failed to create subscription completion event",
				zap.Error(err),
				zap.String("subscription_id", subscription.ID.String()))
		} else {
			s.logger.Info("Created subscription completion event",
				zap.String("subscription_id", subscription.ID.String()),
				zap.Int32("total_redemptions", updatedSub.TotalRedemptions))
		}
	}

	s.logger.Info("Successfully processed subscription",
		zap.String("subscription_id", subscription.ID.String()),
		zap.String("tx_hash", txHash))

	return nil
}

// SubscribeToProductByPriceID handles the complete workflow for subscribing to a product
// Deprecated: The name is kept for backward compatibility but it now uses product ID directly
func (s *SubscriptionService) SubscribeToProductByPriceID(ctx context.Context, subscribeParams params.SubscribeToProductByPriceIDParams) (*responses.SubscribeToProductByPriceIDResult, error) {
	// Since prices are now embedded in products, ProductID is what was formerly PriceID
	productID := subscribeParams.ProductID

	// Get product and validate it's active
	product, err := s.queries.GetProductWithoutWorkspaceId(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	if !product.Active {
		return &responses.SubscribeToProductByPriceIDResult{
			Success:      false,
			ErrorMessage: "Cannot subscribe to inactive product",
		}, nil
	}

	// Parse and validate product token ID
	parsedProductTokenID, err := uuid.Parse(subscribeParams.ProductTokenID)
	if err != nil {
		return &responses.SubscribeToProductByPriceIDResult{
			Success:      false,
			ErrorMessage: "Invalid product token ID format",
		}, nil
	}

	// Get merchant wallet
	merchantWallet, err := s.queries.GetWalletByID(ctx, db.GetWalletByIDParams{
		ID:          product.WalletID,
		WorkspaceID: product.WorkspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get merchant wallet: %w", err)
	}

	// Get product token
	productToken, err := s.queries.GetProductToken(ctx, parsedProductTokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product token: %w", err)
	}

	// Validate product token belongs to product
	if productToken.ProductID != product.ID {
		return &responses.SubscribeToProductByPriceIDResult{
			Success:      false,
			ErrorMessage: "Product token does not belong to the specified product",
		}, nil
	}

	// Get token
	token, err := s.queries.GetToken(ctx, productToken.TokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Get network
	network, err := s.queries.GetNetwork(ctx, token.NetworkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get network details: %w", err)
	}

	// Parse token amount
	tokenAmount, err := strconv.ParseInt(subscribeParams.TokenAmount, 10, 64)
	if err != nil {
		return &responses.SubscribeToProductByPriceIDResult{
			Success:      false,
			ErrorMessage: "Invalid token amount format",
		}, nil
	}

	// Normalize wallet address
	normalizedAddress := helpers.NormalizeWalletAddress(subscribeParams.SubscriberAddress, helpers.DetermineNetworkType(productToken.NetworkType))

	// Create delegation params
	caveatsJSON, err := json.Marshal(subscribeParams.DelegationData.Caveats)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal caveats: %w", err)
	}

	delegationParams := params.StoreDelegationDataParams{
		Delegate:  subscribeParams.DelegationData.Delegate,
		Delegator: subscribeParams.DelegationData.Delegator,
		Authority: subscribeParams.DelegationData.Authority,
		Caveats:   caveatsJSON,
		Salt:      subscribeParams.DelegationData.Salt,
		Signature: subscribeParams.DelegationData.Signature,
	}

	// Since this method is called with a transaction-aware service (via WithTransaction),
	// we can directly use the service's existing methods that work with transactions

	// Process customer and wallet first
	customer, customerWallet, err := s.customerService.ProcessCustomerAndWallet(ctx, nil, params.ProcessCustomerWalletParams{
		WalletAddress: normalizedAddress,
		WorkspaceID:   product.WorkspaceID,
		ProductID:     product.ID,
		NetworkType:   string(network.NetworkType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to process customer and wallet: %w", err)
	}

	// Store delegation data (this needs a transaction but service should handle it)
	delegationData, err := s.StoreDelegationData(ctx, nil, delegationParams)
	if err != nil {
		return nil, fmt.Errorf("failed to store delegation data: %w", err)
	}

	// Calculate subscription periods
	periodStart, periodEnd, nextRedemption := helpers.CalculateSubscriptionPeriods(product)

	// Create subscription using the service method
	subscription, err := s.CreateSubscription(ctx, nil, params.CreateSubscriptionParams{
		Customer:       *customer,
		CustomerWallet: *customerWallet,
		WorkspaceID:    product.WorkspaceID,
		ProductID:      product.ID,
		ProductTokenID: parsedProductTokenID,
		Product:        product,
		TokenAmount:    tokenAmount,
		DelegationData: *delegationData,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		NextRedemption: nextRedemption,
	})
	if err != nil {
		var subExistsErr *SubscriptionExistsError
		if errors.As(err, &subExistsErr) {
			return &responses.SubscribeToProductByPriceIDResult{
				Success:      false,
				ErrorMessage: "Subscription already exists for this customer and product",
			}, nil
		}
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// Create subscription creation event
	eventMetadata, err := json.Marshal(map[string]interface{}{
		"product_name":   product.Name,
		"price_type":     string(product.PriceType),
		"wallet_address": customerWallet.WalletAddress,
		"network_type":   customerWallet.NetworkType,
	})
	if err != nil {
		s.logger.Error("Failed to marshal event metadata", zap.Error(err))
		eventMetadata = []byte("{}")
	}

	_, err = s.queries.CreateSubscriptionEvent(ctx, db.CreateSubscriptionEventParams{
		SubscriptionID: subscription.ID,
		EventType:      db.SubscriptionEventTypeCreated,
		OccurredAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		AmountInCents:  product.UnitAmountInPennies,
		Metadata:       eventMetadata,
	})
	if err != nil {
		s.logger.Error("Failed to create subscription event", zap.Error(err))
		// Don't fail the whole operation for event creation failure
	}

	// Perform initial redemption
	updatedSubscription, err := s.ProcessInitialRedemption(ctx, nil, params.InitialRedemptionParams{
		Subscription:   *subscription,
		Customer:       *customer,
		CustomerWallet: *customerWallet,
		Product:        product,
		ProductToken:   productToken,
		DelegationData: delegationParams,
		MerchantWallet: merchantWallet,
		Token:          token,
		Network:        network,
		TokenAmount:    tokenAmount,
	})
	if err != nil {
		// Update subscription status to failed and soft delete
		_, updateErr := s.queries.UpdateSubscriptionStatus(ctx, db.UpdateSubscriptionStatusParams{
			ID:     subscription.ID,
			Status: db.SubscriptionStatusFailed,
		})
		if updateErr != nil {
			s.logger.Error("Failed to update subscription status after redemption failure",
				zap.Error(updateErr),
				zap.String("subscription_id", subscription.ID.String()))
		}

		deleteErr := s.queries.DeleteSubscription(ctx, subscription.ID)
		if deleteErr != nil {
			s.logger.Error("Failed to soft-delete subscription after redemption failure",
				zap.Error(deleteErr),
				zap.String("subscription_id", subscription.ID.String()))
		}

		// Create failed redemption event
		errorMsg := fmt.Sprintf("Initial redemption failed: %v", err)
		_, eventErr := s.queries.CreateSubscriptionEvent(ctx, db.CreateSubscriptionEventParams{
			SubscriptionID:  subscription.ID,
			EventType:       db.SubscriptionEventTypeFailedRedemption,
			TransactionHash: pgtype.Text{String: "", Valid: false},
			AmountInCents:   product.UnitAmountInPennies,
			ErrorMessage:    pgtype.Text{String: errorMsg, Valid: true},
			OccurredAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
			Metadata:        json.RawMessage(`{}`),
		})

		if eventErr != nil {
			s.logger.Error("Failed to create subscription event after redemption failure",
				zap.Error(eventErr),
				zap.String("subscription_id", subscription.ID.String()))
		}

		return &responses.SubscribeToProductByPriceIDResult{
			Success:      false,
			ErrorMessage: "Initial redemption failed, subscription marked as failed and soft-deleted",
		}, nil
	}

	return &responses.SubscribeToProductByPriceIDResult{
		Subscription: updatedSubscription,
		Success:      true,
	}, nil
}
