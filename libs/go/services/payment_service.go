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
	queries             db.Querier
	logger              *zap.Logger
	gasFeeService       *GasFeeService
	taxService          *TaxService
	discountService     *DiscountService
	exchangeRateService *ExchangeRateService
	gasSponsorshipService *GasSponsorshipService
}

// NewPaymentService creates a new payment service
func NewPaymentService(queries db.Querier, cmcAPIKey string) *PaymentService {
	exchangeRateService := NewExchangeRateService(queries, cmcAPIKey)
	gasFeeService := NewGasFeeService(queries, exchangeRateService)
	taxService := NewTaxService(queries)
	discountService := NewDiscountService(queries)
	gasSponsorshipService := NewGasSponsorshipService(queries)

	return &PaymentService{
		queries:               queries,
		logger:                logger.Log,
		gasFeeService:         gasFeeService,
		taxService:            taxService,
		discountService:       discountService,
		exchangeRateService:   exchangeRateService,
		gasSponsorshipService: gasSponsorshipService,
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

// CreateComprehensivePaymentParams contains all parameters for a comprehensive payment
type CreateComprehensivePaymentParams struct {
	WorkspaceID         uuid.UUID
	CustomerID          uuid.UUID
	InvoiceID           *uuid.UUID
	SubscriptionID      *uuid.UUID
	ProductID           *uuid.UUID
	AmountCents         int64
	Currency            string
	PaymentMethod       string
	TransactionHash     *string
	NetworkID           *uuid.UUID
	TokenID             *uuid.UUID
	GasUsed             *int64
	GasPriceWei         *string
	CryptoAmount        *string
	DiscountCode        *string
	CustomerAddress     *PaymentAddress
	BusinessAddress     *PaymentAddress
	IsB2B               bool
	CustomerVATNumber   *string
	ProductType         string
	TransactionType     string
}

// PaymentAddress represents an address for payment processing
type PaymentAddress struct {
	Street1    string `json:"street1"`
	Street2    string `json:"street2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// CreateComprehensivePayment creates a payment with full business logic integration
func (s *PaymentService) CreateComprehensivePayment(ctx context.Context, params CreateComprehensivePaymentParams) (*db.Payment, error) {
	s.logger.Info("Creating comprehensive payment",
		zap.String("workspace_id", params.WorkspaceID.String()),
		zap.String("customer_id", params.CustomerID.String()),
		zap.Int64("amount_cents", params.AmountCents),
		zap.String("currency", params.Currency))

	// Step 1: Calculate taxes
	var taxResult *TaxCalculationResult
	var err error
	
	if params.CustomerAddress != nil || params.BusinessAddress != nil {
		taxParams := TaxCalculationParams{
			WorkspaceID:       params.WorkspaceID,
			CustomerID:        params.CustomerID,
			ProductID:         params.ProductID,
			SubscriptionID:    params.SubscriptionID,
			AmountCents:       params.AmountCents,
			Currency:          params.Currency,
			CustomerAddress:   convertToTaxAddress(params.CustomerAddress),
			BusinessAddress:   convertToTaxAddress(params.BusinessAddress),
			TransactionType:   params.TransactionType,
			ProductType:       params.ProductType,
			IsB2B:             params.IsB2B,
			CustomerVATNumber: params.CustomerVATNumber,
		}
		
		taxResult, err = s.taxService.CalculateTax(ctx, taxParams)
		if err != nil {
			s.logger.Warn("Failed to calculate tax, proceeding without", zap.Error(err))
			taxResult = &TaxCalculationResult{
				SubtotalCents:    params.AmountCents,
				TotalTaxCents:    0,
				TotalAmountCents: params.AmountCents,
			}
		}
	} else {
		// No tax calculation needed
		taxResult = &TaxCalculationResult{
			SubtotalCents:    params.AmountCents,
			TotalTaxCents:    0,
			TotalAmountCents: params.AmountCents,
		}
	}

	// Step 2: Apply discounts if provided
	var discountResult *DiscountApplicationResult
	finalAmount := taxResult.TotalAmountCents
	
	if params.DiscountCode != nil && *params.DiscountCode != "" {
		// Get customer info for discount validation
		customer, err := s.queries.GetCustomer(ctx, params.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("failed to get customer for discount validation: %w", err)
		}

		discountParams := DiscountApplicationParams{
			WorkspaceID:    params.WorkspaceID,
			CustomerID:     params.CustomerID,
			ProductID:      params.ProductID,
			SubscriptionID: params.SubscriptionID,
			DiscountCode:   *params.DiscountCode,
			AmountCents:    taxResult.TotalAmountCents,
			Currency:       params.Currency,
			IsNewCustomer:  customer.CreatedAt.Time.After(time.Now().AddDate(0, 0, -1)), // Rough new customer check
			CustomerEmail:  customer.Email.String,
		}

		discountResult, err = s.discountService.ApplyDiscount(ctx, discountParams)
		if err != nil {
			s.logger.Warn("Failed to apply discount, proceeding without", zap.Error(err))
		} else if discountResult.IsValid {
			finalAmount = discountResult.FinalAmountCents
		}
	}

	// Step 3: Calculate gas fees if this is a crypto transaction
	var gasFeeResult *GasFeeResult
	var gasCostCents int64 = 0
	
	if params.PaymentMethod == "crypto" && params.NetworkID != nil {
		// Estimate gas fee for the transaction
		estimateParams := EstimateGasFeeParams{
			NetworkID:         *params.NetworkID,
			TransactionType:   params.TransactionType,
			EstimatedGasLimit: 21000, // Standard ETH transfer
			Currency:          params.Currency,
		}

		gasEstimate, err := s.gasFeeService.EstimateGasFee(ctx, estimateParams)
		if err != nil {
			s.logger.Warn("Failed to estimate gas fee", zap.Error(err))
		} else {
			gasCostCents = gasEstimate.EstimatedCostCents
		}
	}

	// Step 4: Check gas sponsorship
	var gasSponsored bool
	var sponsorID *uuid.UUID
	
	if gasCostCents > 0 && params.ProductID != nil {
		sponsorshipParams := SponsorshipCheckParams{
			WorkspaceID:     params.WorkspaceID,
			CustomerID:      params.CustomerID,
			ProductID:       *params.ProductID,
			GasCostUSDCents: gasCostCents,
			TransactionType: params.TransactionType,
		}

		sponsorshipDecision, err := s.gasSponsorshipService.ShouldSponsorGas(ctx, sponsorshipParams)
		if err != nil {
			s.logger.Warn("Failed to check gas sponsorship", zap.Error(err))
		} else if sponsorshipDecision.ShouldSponsor {
			gasSponsored = true
			sponsorID = &sponsorshipDecision.SponsorID
		}
	}

	// Step 5: Create the payment record
	paymentParams := db.CreatePaymentParams{
		WorkspaceID:        params.WorkspaceID,
		CustomerID:         params.CustomerID,
		AmountInCents:      finalAmount,
		Currency:           params.Currency,
		Status:             "pending", // Will be updated when confirmed
		PaymentMethod:      params.PaymentMethod,
		ProductAmountCents: params.AmountCents,
		HasGasFee:          pgtype.Bool{Bool: gasCostCents > 0, Valid: true},
		GasSponsored:       pgtype.Bool{Bool: gasSponsored, Valid: true},
	}

	// Set optional fields
	if params.InvoiceID != nil {
		paymentParams.InvoiceID = pgtype.UUID{Bytes: *params.InvoiceID, Valid: true}
	}

	if params.SubscriptionID != nil {
		paymentParams.SubscriptionID = pgtype.UUID{Bytes: *params.SubscriptionID, Valid: true}
	}

	if params.TransactionHash != nil {
		paymentParams.TransactionHash = pgtype.Text{String: *params.TransactionHash, Valid: true}
	}

	if params.NetworkID != nil {
		paymentParams.NetworkID = pgtype.UUID{Bytes: *params.NetworkID, Valid: true}
	}

	if params.TokenID != nil {
		paymentParams.TokenID = pgtype.UUID{Bytes: *params.TokenID, Valid: true}
	}

	// Set tax and discount amounts
	if taxResult.TotalTaxCents > 0 {
		paymentParams.TaxAmountCents = pgtype.Int8{Int64: taxResult.TotalTaxCents, Valid: true}
	}

	if gasCostCents > 0 {
		paymentParams.GasAmountCents = pgtype.Int8{Int64: gasCostCents, Valid: true}
		paymentParams.GasFeeUsdCents = pgtype.Int8{Int64: gasCostCents, Valid: true}
	}

	if discountResult != nil && discountResult.IsValid {
		paymentParams.DiscountAmountCents = pgtype.Int8{Int64: discountResult.DiscountAmountCents, Valid: true}
	}

	// Create the payment
	payment, err := s.queries.CreatePayment(ctx, paymentParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Step 6: Create related records
	
	// Store tax calculation if applicable
	if taxResult != nil && taxResult.TotalTaxCents > 0 {
		if err := s.taxService.StoreTaxCalculation(ctx, payment.ID, taxResult); err != nil {
			s.logger.Warn("Failed to store tax calculation", zap.Error(err))
		}
	}

	// Create gas fee payment record if applicable
	if gasFeeResult != nil {
		if err := s.gasFeeService.CreateGasFeePaymentRecord(ctx, payment.ID, gasFeeResult, gasSponsored, sponsorID); err != nil {
			s.logger.Warn("Failed to create gas fee payment record", zap.Error(err))
		}
	}

	// Record gas sponsorship if applicable
	if gasSponsored && sponsorID != nil {
		sponsorshipRecord := SponsorshipRecord{
			WorkspaceID:     params.WorkspaceID,
			PaymentID:       payment.ID,
			GasCostUSDCents: gasCostCents,
			SponsorType:     "merchant",
			SponsorID:       *sponsorID,
		}

		if err := s.gasSponsorshipService.RecordSponsoredTransaction(ctx, sponsorshipRecord); err != nil {
			s.logger.Warn("Failed to record sponsored transaction", zap.Error(err))
		}
	}

	s.logger.Info("Successfully created comprehensive payment",
		zap.String("payment_id", payment.ID.String()),
		zap.Int64("final_amount", finalAmount),
		zap.Int64("tax_cents", taxResult.TotalTaxCents),
		zap.Int64("gas_cents", gasCostCents),
		zap.Bool("gas_sponsored", gasSponsored))

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

// Helper function to convert PaymentAddress to tax service Address type
func convertToTaxAddress(addr *PaymentAddress) *Address {
	if addr == nil {
		return nil
	}
	// Convert PaymentAddress to tax service Address
	return &Address{
		Street1:    addr.Street1,
		Street2:    addr.Street2,
		City:       addr.City,
		State:      addr.State,
		PostalCode: addr.PostalCode,
		Country:    addr.Country,
	}
}