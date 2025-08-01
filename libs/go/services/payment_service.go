package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/constants"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// PaymentService handles business logic for payment operations
type PaymentService struct {
	queries               db.Querier
	logger                *zap.Logger
	gasFeeService         *GasFeeService
	taxService            *TaxService
	discountService       *DiscountService
	exchangeRateService   *ExchangeRateService
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

// CreatePaymentFromSubscriptionEvent creates a payment record when a subscription redemption occurs
func (s *PaymentService) CreatePaymentFromSubscriptionEvent(ctx context.Context, params params.CreatePaymentFromSubscriptionEventParams) (*db.Payment, error) {
	event := params.SubscriptionEvent
	subscription := params.Subscription
	product := params.Product
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
		WorkspaceID:        subscription.WorkspaceID,
		SubscriptionID:     pgtype.UUID{Bytes: subscription.ID, Valid: true},
		SubscriptionEvent:  pgtype.UUID{Bytes: event.ID, Valid: true},
		CustomerID:         customer.ID,
		AmountInCents:      int64(event.AmountInCents),
		Currency:           string(product.Currency),
		Status:             "completed", // Subscription events are already completed
		PaymentMethod:      "crypto",
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
		// Parse the decimal string into a pgtype.Numeric
		if err := paymentParams.CryptoAmount.Scan(params.CryptoAmount); err != nil {
			s.logger.Warn("Failed to parse crypto amount, skipping field",
				zap.String("crypto_amount", params.CryptoAmount),
				zap.Error(err))
			// Leave as invalid/null if parsing fails
			paymentParams.CryptoAmount = pgtype.Numeric{Valid: false}
		}
	}

	if params.ExchangeRate != "" {
		// Parse the decimal string into a pgtype.Numeric
		if err := paymentParams.ExchangeRate.Scan(params.ExchangeRate); err != nil {
			s.logger.Warn("Failed to parse exchange rate, skipping field",
				zap.String("exchange_rate", params.ExchangeRate),
				zap.Error(err))
			// Leave as invalid/null if parsing fails
			paymentParams.ExchangeRate = pgtype.Numeric{Valid: false}
		}
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

// CreateComprehensivePayment creates a payment with full business logic integration
func (s *PaymentService) CreateComprehensivePayment(ctx context.Context, paymentParams params.CreateComprehensivePaymentParams) (*db.Payment, error) {
	s.logger.Info("Creating comprehensive payment",
		zap.String("workspace_id", paymentParams.WorkspaceID.String()),
		zap.String("customer_id", paymentParams.CustomerID.String()),
		zap.Int64("amount_cents", paymentParams.AmountCents),
		zap.String("currency", paymentParams.Currency))

	// Step 1: Calculate taxes
	var taxResult *responses.TaxCalculationResult
	var err error

	if paymentParams.CustomerAddress != nil || paymentParams.BusinessAddress != nil {
		taxParams := params.TaxCalculationParams{
			WorkspaceID:       paymentParams.WorkspaceID,
			CustomerID:        paymentParams.CustomerID,
			ProductID:         paymentParams.ProductID,
			SubscriptionID:    paymentParams.SubscriptionID,
			AmountCents:       paymentParams.AmountCents,
			Currency:          paymentParams.Currency,
			CustomerAddress:   convertPaymentToTaxAddress(paymentParams.CustomerAddress),
			BusinessAddress:   convertPaymentToTaxAddress(paymentParams.BusinessAddress),
			TransactionType:   paymentParams.TransactionType,
			ProductType:       paymentParams.ProductType,
			IsB2B:             paymentParams.IsB2B,
			CustomerVATNumber: paymentParams.CustomerVATNumber,
		}

		taxResult, err = s.taxService.CalculateTax(ctx, taxParams)
		if err != nil {
			s.logger.Warn("Failed to calculate tax, proceeding without", zap.Error(err))
			taxResult = &responses.TaxCalculationResult{
				SubtotalCents:    paymentParams.AmountCents,
				TotalTaxCents:    0,
				TotalAmountCents: paymentParams.AmountCents,
			}
		}
	} else {
		// No tax calculation needed
		taxResult = &responses.TaxCalculationResult{
			SubtotalCents:    paymentParams.AmountCents,
			TotalTaxCents:    0,
			TotalAmountCents: paymentParams.AmountCents,
		}
	}

	// Step 2: Apply discounts if provided
	var discountResult *responses.DiscountApplicationResult
	finalAmount := taxResult.TotalAmountCents

	if paymentParams.DiscountCode != nil && *paymentParams.DiscountCode != "" {
		// Get customer info for discount validation
		customer, err := s.queries.GetCustomer(ctx, paymentParams.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("failed to get customer for discount validation: %w", err)
		}

		discountParams := params.DiscountApplicationParams{
			WorkspaceID:    paymentParams.WorkspaceID,
			CustomerID:     paymentParams.CustomerID,
			ProductID:      paymentParams.ProductID,
			SubscriptionID: paymentParams.SubscriptionID,
			DiscountCode:   *paymentParams.DiscountCode,
			AmountCents:    taxResult.TotalAmountCents,
			Currency:       paymentParams.Currency,
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
	var gasFeeResult *responses.GasFeeResult
	var gasCostCents int64 = 0

	if paymentParams.PaymentMethod == "crypto" && paymentParams.NetworkID != nil {
		// Estimate gas fee for the transaction
		estimateParams := params.EstimateGasFeeParams{
			NetworkID:         *paymentParams.NetworkID,
			TransactionType:   paymentParams.TransactionType,
			EstimatedGasLimit: 21000, // Standard ETH transfer
			Currency:          paymentParams.Currency,
		}

		gasEstimate, err := s.gasFeeService.EstimateGasFee(ctx, estimateParams)
		if err != nil {
			s.logger.Warn("Failed to estimate gas fee", zap.Error(err))
		} else {
			gasCostCents = gasEstimate.EstimatedCostUSDCents
		}
	}

	// Step 4: Check gas sponsorship
	var gasSponsored bool
	var sponsorID *uuid.UUID

	if gasCostCents > 0 && paymentParams.ProductID != nil {
		sponsorshipParams := params.SponsorshipCheckParams{
			WorkspaceID:     paymentParams.WorkspaceID,
			CustomerID:      paymentParams.CustomerID,
			ProductID:       *paymentParams.ProductID,
			GasCostUSDCents: gasCostCents,
			TransactionType: paymentParams.TransactionType,
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
	paymentParamsObj := db.CreatePaymentParams{
		WorkspaceID:        paymentParams.WorkspaceID,
		CustomerID:         paymentParams.CustomerID,
		AmountInCents:      finalAmount,
		Currency:           paymentParams.Currency,
		Status:             "pending", // Will be updated when confirmed
		PaymentMethod:      paymentParams.PaymentMethod,
		ProductAmountCents: paymentParams.AmountCents,
		HasGasFee:          pgtype.Bool{Bool: gasCostCents > 0, Valid: true},
		GasSponsored:       pgtype.Bool{Bool: gasSponsored, Valid: true},
	}

	// Set optional fields
	if paymentParams.InvoiceID != nil {
		paymentParamsObj.InvoiceID = pgtype.UUID{Bytes: *paymentParams.InvoiceID, Valid: true}
	}

	if paymentParams.SubscriptionID != nil {
		paymentParamsObj.SubscriptionID = pgtype.UUID{Bytes: *paymentParams.SubscriptionID, Valid: true}
	}

	if paymentParams.TransactionHash != nil {
		paymentParamsObj.TransactionHash = pgtype.Text{String: *paymentParams.TransactionHash, Valid: true}
	}

	if paymentParams.NetworkID != nil {
		paymentParamsObj.NetworkID = pgtype.UUID{Bytes: *paymentParams.NetworkID, Valid: true}
	}

	if paymentParams.TokenID != nil {
		paymentParamsObj.TokenID = pgtype.UUID{Bytes: *paymentParams.TokenID, Valid: true}
	}

	// Set tax and discount amounts
	if taxResult.TotalTaxCents > 0 {
		paymentParamsObj.TaxAmountCents = pgtype.Int8{Int64: taxResult.TotalTaxCents, Valid: true}
	}

	if gasCostCents > 0 {
		paymentParamsObj.GasAmountCents = pgtype.Int8{Int64: gasCostCents, Valid: true}
		paymentParamsObj.GasFeeUsdCents = pgtype.Int8{Int64: gasCostCents, Valid: true}
	}

	if discountResult != nil && discountResult.IsValid {
		paymentParamsObj.DiscountAmountCents = pgtype.Int8{Int64: discountResult.DiscountAmountCents, Valid: true}
	}

	// Create the payment
	payment, err := s.queries.CreatePayment(ctx, paymentParamsObj)
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
		sponsorshipRecord := business.SponsorshipRecord{
			WorkspaceID:     paymentParams.WorkspaceID,
			PaymentID:       payment.ID,
			GasCostUSDCents: gasCostCents,
			SponsorType:     constants.MerchantSponsorType,
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

// GetPayment retrieves a payment by ID with workspace validation
func (s *PaymentService) GetPayment(ctx context.Context, params params.GetPaymentParams) (*db.Payment, error) {
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

// ListPayments retrieves a paginated list of payments
func (s *PaymentService) ListPayments(ctx context.Context, params params.ListPaymentsParams) ([]db.Payment, error) {
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

	if params.Status != nil {
		return s.queries.GetPaymentsByStatus(ctx, db.GetPaymentsByStatusParams{
			WorkspaceID: params.WorkspaceID,
			Status:      *params.Status,
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

// UpdatePaymentStatus updates the status of a payment
func (s *PaymentService) UpdatePaymentStatus(ctx context.Context, paymentParams params.UpdatePaymentStatusParams) (*db.Payment, error) {
	var errorMsg pgtype.Text
	if paymentParams.FailureReason != nil {
		errorMsg = pgtype.Text{String: *paymentParams.FailureReason, Valid: true}
	}

	payment, err := s.queries.UpdatePaymentStatus(ctx, db.UpdatePaymentStatusParams{
		ID:           paymentParams.PaymentID,
		WorkspaceID:  paymentParams.WorkspaceID,
		Status:       paymentParams.Status,
		ErrorMessage: errorMsg,
	})
	if err != nil {
		s.logger.Error("Failed to update payment status",
			zap.String("payment_id", paymentParams.PaymentID.String()),
			zap.String("status", paymentParams.Status),
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

// CreateManualPayment creates a payment manually (not from subscription event)
func (s *PaymentService) CreateManualPayment(ctx context.Context, paymentParams params.CreateManualPaymentParams) (*db.Payment, error) {
	// Prepare metadata
	var metadataBytes []byte
	if paymentParams.Metadata != nil {
		var err error
		metadataBytes, err = json.Marshal(paymentParams.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	} else {
		metadataBytes = []byte("{}")
	}

	// Prepare payment parameters
	paymentParamsObj := db.CreatePaymentParams{
		WorkspaceID:        paymentParams.WorkspaceID,
		CustomerID:         paymentParams.CustomerID,
		AmountInCents:      paymentParams.AmountInCents,
		Currency:           paymentParams.Currency,
		Status:             "completed", // Manual payments are typically already completed
		PaymentMethod:      paymentParams.PaymentMethod,
		ProductAmountCents: paymentParams.AmountInCents, // Assume full amount is for product
		Metadata:           metadataBytes,
	}

	// Add optional fields
	if paymentParams.SubscriptionID != nil {
		paymentParamsObj.SubscriptionID = pgtype.UUID{Bytes: *paymentParams.SubscriptionID, Valid: true}
	}

	if paymentParams.InvoiceID != nil {
		paymentParamsObj.InvoiceID = pgtype.UUID{Bytes: *paymentParams.InvoiceID, Valid: true}
	}

	if paymentParams.TransactionHash != "" {
		paymentParamsObj.TransactionHash = pgtype.Text{String: paymentParams.TransactionHash, Valid: true}
	}

	if paymentParams.NetworkID != nil {
		paymentParamsObj.NetworkID = pgtype.UUID{Bytes: *paymentParams.NetworkID, Valid: true}
	}

	if paymentParams.TokenID != nil {
		paymentParamsObj.TokenID = pgtype.UUID{Bytes: *paymentParams.TokenID, Valid: true}
	}

	if paymentParams.ExternalPaymentID != "" {
		paymentParamsObj.ExternalPaymentID = pgtype.Text{String: paymentParams.ExternalPaymentID, Valid: true}
	}

	if paymentParams.PaymentProvider != "" {
		paymentParamsObj.PaymentProvider = pgtype.Text{String: paymentParams.PaymentProvider, Valid: true}
	}

	// Create the payment
	payment, err := s.queries.CreatePayment(ctx, paymentParamsObj)
	if err != nil {
		s.logger.Error("Failed to create manual payment",
			zap.String("customer_id", paymentParamsObj.CustomerID.String()),
			zap.String("payment_method", paymentParamsObj.PaymentMethod),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	s.logger.Info("Manual payment created successfully",
		zap.String("payment_id", payment.ID.String()),
		zap.String("payment_method", payment.PaymentMethod),
		zap.Int64("amount_cents", payment.AmountInCents))

	return &payment, nil
}

// Helper function to convert PaymentAddress to tax service Address type
func convertPaymentToTaxAddress(addr *params.PaymentAddress) *business.Address {
	if addr == nil {
		return nil
	}
	// Convert PaymentAddress to tax service Address
	return &business.Address{
		Street1:    addr.Street1,
		Street2:    addr.Street2,
		City:       addr.City,
		State:      addr.State,
		PostalCode: addr.PostalCode,
		Country:    addr.Country,
	}
}
