package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	dsClient "github.com/cyphera/cyphera-api/libs/go/client/delegation_server"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// SubscriptionService handles subscription business logic
type SubscriptionService struct {
	queries              db.Querier
	delegationClient     *dsClient.DelegationClient
	paymentService       *PaymentService
	lastRedemptionTxHash string // Stores the transaction hash from the last successful redemption
}

// NewSubscriptionService creates a new subscription service
func NewSubscriptionService(queries db.Querier, delegationClient *dsClient.DelegationClient, paymentService *PaymentService) *SubscriptionService {
	return &SubscriptionService{
		queries:          queries,
		delegationClient: delegationClient,
		paymentService:   paymentService,
	}
}

// SubscriptionExistsError is a custom error for when a subscription already exists
type SubscriptionExistsError struct {
	Subscription *db.Subscription
}

func (e *SubscriptionExistsError) Error() string {
	return "subscription already exists for this customer and product"
}

// UpdateSubscriptionRequest represents the request structure for updating a subscription
type UpdateSubscriptionRequest struct {
	CustomerID       string          `json:"customer_id"`
	ProductID        string          `json:"product_id"`
	ProductTokenID   string          `json:"product_token_id"`
	DelegationID     string          `json:"delegation_id"`
	CustomerWalletID string          `json:"customer_wallet_id"`
	Status           string          `json:"status"`
	StartDate        int64           `json:"start_date"`
	EndDate          int64           `json:"end_date"`
	NextRedemption   int64           `json:"next_redemption"`
	Metadata         json.RawMessage `json:"metadata"`
}

// ProcessSubscriptionParams represents parameters for processing a subscription
type ProcessSubscriptionParams struct {
	Subscription         db.Subscription
	Price                db.Price
	Product              db.Product
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

// ProcessDueSubscriptionsResult contains statistics about the processing job
type ProcessDueSubscriptionsResult struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
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
func (s *SubscriptionService) ListSubscriptions(ctx context.Context, workspaceID uuid.UUID, limit, offset int32) ([]helpers.SubscriptionResponse, int64, error) {
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

	subscriptionResponses := make([]helpers.SubscriptionResponse, len(subscriptions))
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
func (s *SubscriptionService) ListSubscriptionsByCustomer(ctx context.Context, workspaceID, customerID uuid.UUID) ([]helpers.SubscriptionResponse, error) {
	subscriptions, err := s.queries.ListSubscriptionsByCustomer(ctx, db.ListSubscriptionsByCustomerParams{
		CustomerID:  customerID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve customer subscriptions: %w", err)
	}

	subscriptionResponses := make([]helpers.SubscriptionResponse, len(subscriptions))
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
func (s *SubscriptionService) UpdateSubscription(ctx context.Context, subscriptionID uuid.UUID, req UpdateSubscriptionRequest) (*db.Subscription, error) {
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
