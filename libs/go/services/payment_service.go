package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// PaymentService handles business logic for payment operations
type PaymentService struct {
	queries db.Querier
	logger  *zap.Logger
}

// NewPaymentService creates a new payment service
func NewPaymentService(queries db.Querier) *PaymentService {
	return &PaymentService{
		queries: queries,
		logger:  logger.Log,
	}
}

// CreatePaymentFromSubscriptionEventParams contains parameters for creating a payment from a subscription event
type CreatePaymentFromSubscriptionEventParams struct {
	SubscriptionEvent   *db.SubscriptionEvent
	Subscription        *db.Subscription
	Product             *db.Product
	Price               *db.Price
	Customer            *db.Customer
	TransactionHash     string
	NetworkID           uuid.UUID
	TokenID             uuid.UUID
	CryptoAmount        string // Decimal as string
	ExchangeRate        string // Decimal as string
	GasFeeUSDCents      int64
	GasSponsored        bool
}

// CreatePaymentFromSubscriptionEvent creates a payment record when a subscription redemption occurs
func (s *PaymentService) CreatePaymentFromSubscriptionEvent(ctx context.Context, params CreatePaymentFromSubscriptionEventParams) (*db.Payment, error) {
	event := params.SubscriptionEvent
	subscription := params.Subscription
	price := params.Price
	customer := params.Customer

	// Validate that this is a redeemed event
	if event.EventType != db.SubscriptionEventTypeRedeemed {
		return nil, fmt.Errorf("can only create payments for redeemed subscription events")
	}

	// Check if payment already exists for this subscription event
	existingPayment, err := s.queries.GetPaymentBySubscriptionEvent(ctx, pgtype.UUID{
		Bytes: event.ID,
		Valid: true,
	})
	if err == nil {
		s.logger.Info("Payment already exists for subscription event",
			zap.String("payment_id", existingPayment.ID.String()),
			zap.String("event_id", event.ID.String()))
		return &existingPayment, nil
	}

	// Prepare payment parameters
	paymentParams := db.CreatePaymentParams{
		WorkspaceID:    subscription.WorkspaceID,
		SubscriptionID: pgtype.UUID{Bytes: subscription.ID, Valid: true},
		SubscriptionEvent: pgtype.UUID{Bytes: event.ID, Valid: true},
		CustomerID:     customer.ID,
		AmountInCents:  int64(event.AmountInCents),
		Currency:       string(price.Currency),
		Status:         "completed", // Subscription events are already completed
		PaymentMethod:  "crypto",
		ProductAmountCents: int64(event.AmountInCents),
	}

	// Add blockchain data if available
	if params.TransactionHash != "" {
		paymentParams.TransactionHash = pgtype.Text{
			String: params.TransactionHash,
			Valid:  true,
		}
	}

	if params.NetworkID != uuid.Nil {
		paymentParams.NetworkID = pgtype.UUID{
			Bytes: params.NetworkID,
			Valid: true,
		}
	}

	if params.TokenID != uuid.Nil {
		paymentParams.TokenID = pgtype.UUID{
			Bytes: params.TokenID,
			Valid: true,
		}
	}

	if params.CryptoAmount != "" {
		paymentParams.CryptoAmount = pgtype.Numeric{
			Int:   nil,
			Exp:   0,
			Valid: true,
		}
		// Note: You may need to properly parse the decimal string here
		// For now, just marking as valid
	}

	if params.ExchangeRate != "" {
		paymentParams.ExchangeRate = pgtype.Numeric{
			Int:   nil,
			Exp:   0,
			Valid: true,
		}
		// Note: You may need to properly parse the decimal string here
	}

	// Handle gas fees
	if params.GasFeeUSDCents > 0 {
		paymentParams.HasGasFee = pgtype.Bool{Bool: true, Valid: true}
		paymentParams.GasFeeUsdCents = pgtype.Int8{Int64: params.GasFeeUSDCents, Valid: true}
		paymentParams.GasSponsored = pgtype.Bool{Bool: params.GasSponsored, Valid: true}
		
		// If customer pays gas, add to total amount
		if !params.GasSponsored {
			paymentParams.GasAmountCents = pgtype.Int8{Int64: params.GasFeeUSDCents, Valid: true}
			paymentParams.AmountInCents += params.GasFeeUSDCents
		}
	}

	// Set payment provider as internal since this is from our subscription system
	paymentParams.PaymentProvider = pgtype.Text{String: "internal", Valid: true}

	// Add metadata from subscription event
	if event.Metadata != nil {
		paymentParams.Metadata = event.Metadata
	} else {
		paymentParams.Metadata = []byte("{}")
	}

	// Create the payment
	payment, err := s.queries.CreatePayment(ctx, paymentParams)
	if err != nil {
		s.logger.Error("Failed to create payment from subscription event",
			zap.String("event_id", event.ID.String()),
			zap.String("subscription_id", subscription.ID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	s.logger.Info("Payment created successfully from subscription event",
		zap.String("payment_id", payment.ID.String()),
		zap.String("event_id", event.ID.String()),
		zap.String("transaction_hash", params.TransactionHash),
		zap.Int64("amount_cents", payment.AmountInCents))

	return &payment, nil
}

// GetPaymentParams contains parameters for retrieving a payment
type GetPaymentParams struct {
	PaymentID   uuid.UUID
	WorkspaceID uuid.UUID
}

// GetPayment retrieves a payment by ID with workspace validation
func (s *PaymentService) GetPayment(ctx context.Context, params GetPaymentParams) (*db.Payment, error) {
	payment, err := s.queries.GetPayment(ctx, db.GetPaymentParams{
		ID:          params.PaymentID,
		WorkspaceID: params.WorkspaceID,
	})
	if err != nil {
		s.logger.Error("Failed to get payment",
			zap.String("payment_id", params.PaymentID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("payment not found: %w", err)
	}

	return &payment, nil
}

// GetPaymentByTransactionHash retrieves a payment by blockchain transaction hash
func (s *PaymentService) GetPaymentByTransactionHash(ctx context.Context, txHash string) (*db.Payment, error) {
	if txHash == "" {
		return nil, fmt.Errorf("transaction hash is required")
	}

	payment, err := s.queries.GetPaymentByTransactionHash(ctx, pgtype.Text{
		String: txHash,
		Valid:  true,
	})
	if err != nil {
		s.logger.Error("Failed to get payment by transaction hash",
			zap.String("tx_hash", txHash),
			zap.Error(err))
		return nil, fmt.Errorf("payment not found: %w", err)
	}

	return &payment, nil
}

// ListPaymentsParams contains parameters for listing payments
type ListPaymentsParams struct {
	WorkspaceID uuid.UUID
	CustomerID  *uuid.UUID
	Status      string
	Limit       int32
	Offset      int32
}

// ListPayments retrieves a paginated list of payments
func (s *PaymentService) ListPayments(ctx context.Context, params ListPaymentsParams) ([]db.Payment, error) {
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	if params.CustomerID != nil {
		return s.queries.GetPaymentsByCustomer(ctx, db.GetPaymentsByCustomerParams{
			CustomerID:  *params.CustomerID,
			WorkspaceID: params.WorkspaceID,
			Limit:       params.Limit,
			Offset:      params.Offset,
		})
	}

	if params.Status != "" {
		return s.queries.GetPaymentsByStatus(ctx, db.GetPaymentsByStatusParams{
			WorkspaceID: params.WorkspaceID,
			Status:      params.Status,
			Limit:       params.Limit,
			Offset:      params.Offset,
		})
	}

	return s.queries.GetPaymentsByWorkspace(ctx, db.GetPaymentsByWorkspaceParams{
		WorkspaceID: params.WorkspaceID,
		Limit:       params.Limit,
		Offset:      params.Offset,
	})
}

// UpdatePaymentStatusParams contains parameters for updating payment status
type UpdatePaymentStatusParams struct {
	PaymentID    uuid.UUID
	WorkspaceID  uuid.UUID
	Status       string
	ErrorMessage string
}

// UpdatePaymentStatus updates the status of a payment
func (s *PaymentService) UpdatePaymentStatus(ctx context.Context, params UpdatePaymentStatusParams) (*db.Payment, error) {
	errorMsg := pgtype.Text{String: params.ErrorMessage, Valid: params.ErrorMessage != ""}
	
	payment, err := s.queries.UpdatePaymentStatus(ctx, db.UpdatePaymentStatusParams{
		ID:           params.PaymentID,
		WorkspaceID:  params.WorkspaceID,
		Status:       params.Status,
		ErrorMessage: errorMsg,
	})
	if err != nil {
		s.logger.Error("Failed to update payment status",
			zap.String("payment_id", params.PaymentID.String()),
			zap.String("status", params.Status),
			zap.Error(err))
		return nil, fmt.Errorf("failed to update payment status: %w", err)
	}

	s.logger.Info("Payment status updated",
		zap.String("payment_id", payment.ID.String()),
		zap.String("status", payment.Status))

	return &payment, nil
}

// GetPaymentMetrics retrieves payment metrics for a workspace within a date range
func (s *PaymentService) GetPaymentMetrics(ctx context.Context, workspaceID uuid.UUID, startTime, endTime time.Time, currency string) (*db.GetPaymentMetricsRow, error) {
	metrics, err := s.queries.GetPaymentMetrics(ctx, db.GetPaymentMetricsParams{
		WorkspaceID: workspaceID,
		CreatedAt:   pgtype.Timestamptz{Time: startTime, Valid: true},
		CreatedAt_2: pgtype.Timestamptz{Time: endTime, Valid: true},
		Currency:    currency,
	})
	if err != nil {
		s.logger.Error("Failed to get payment metrics",
			zap.String("workspace_id", workspaceID.String()),
			zap.String("currency", currency),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get payment metrics: %w", err)
	}

	return &metrics, nil
}

// CreateManualPaymentParams contains parameters for creating a manual payment
type CreateManualPaymentParams struct {
	WorkspaceID         uuid.UUID
	CustomerID          uuid.UUID
	SubscriptionID      *uuid.UUID
	InvoiceID           *uuid.UUID
	AmountInCents       int64
	Currency            string
	PaymentMethod       string
	TransactionHash     string
	NetworkID           *uuid.UUID
	TokenID             *uuid.UUID
	CryptoAmount        string
	ExchangeRate        string
	ExternalPaymentID   string
	PaymentProvider     string
	Metadata            map[string]interface{}
}

// CreateManualPayment creates a payment manually (not from subscription event)
func (s *PaymentService) CreateManualPayment(ctx context.Context, params CreateManualPaymentParams) (*db.Payment, error) {
	// Prepare metadata
	var metadataBytes []byte
	if params.Metadata != nil {
		var err error
		metadataBytes, err = json.Marshal(params.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	} else {
		metadataBytes = []byte("{}")
	}

	// Prepare payment parameters
	paymentParams := db.CreatePaymentParams{
		WorkspaceID:        params.WorkspaceID,
		CustomerID:         params.CustomerID,
		AmountInCents:      params.AmountInCents,
		Currency:           params.Currency,
		Status:             "completed", // Manual payments are typically already completed
		PaymentMethod:      params.PaymentMethod,
		ProductAmountCents: params.AmountInCents, // Assume full amount is for product
		Metadata:           metadataBytes,
	}

	// Add optional fields
	if params.SubscriptionID != nil {
		paymentParams.SubscriptionID = pgtype.UUID{Bytes: *params.SubscriptionID, Valid: true}
	}

	if params.InvoiceID != nil {
		paymentParams.InvoiceID = pgtype.UUID{Bytes: *params.InvoiceID, Valid: true}
	}

	if params.TransactionHash != "" {
		paymentParams.TransactionHash = pgtype.Text{String: params.TransactionHash, Valid: true}
	}

	if params.NetworkID != nil {
		paymentParams.NetworkID = pgtype.UUID{Bytes: *params.NetworkID, Valid: true}
	}

	if params.TokenID != nil {
		paymentParams.TokenID = pgtype.UUID{Bytes: *params.TokenID, Valid: true}
	}

	if params.ExternalPaymentID != "" {
		paymentParams.ExternalPaymentID = pgtype.Text{String: params.ExternalPaymentID, Valid: true}
	}

	if params.PaymentProvider != "" {
		paymentParams.PaymentProvider = pgtype.Text{String: params.PaymentProvider, Valid: true}
	}

	// Create the payment
	payment, err := s.queries.CreatePayment(ctx, paymentParams)
	if err != nil {
		s.logger.Error("Failed to create manual payment",
			zap.String("customer_id", params.CustomerID.String()),
			zap.String("payment_method", params.PaymentMethod),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	s.logger.Info("Manual payment created successfully",
		zap.String("payment_id", payment.ID.String()),
		zap.String("payment_method", params.PaymentMethod),
		zap.Int64("amount_cents", payment.AmountInCents))

	return &payment, nil
}