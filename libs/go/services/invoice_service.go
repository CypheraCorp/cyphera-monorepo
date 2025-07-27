package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// InvoiceService handles invoice creation and management
type InvoiceService struct {
	queries              *db.Queries
	logger              *zap.Logger
	taxService          *TaxService
	discountService     *DiscountService
	gasSponsorshipService *GasSponsorshipService
	currencyService     *CurrencyService
	exchangeRateService *ExchangeRateService
}

// NewInvoiceService creates a new invoice service
func NewInvoiceService(
	queries *db.Queries,
	logger *zap.Logger,
	taxService *TaxService,
	discountService *DiscountService,
	gasSponsorshipService *GasSponsorshipService,
	currencyService *CurrencyService,
	exchangeRateService *ExchangeRateService,
) *InvoiceService {
	return &InvoiceService{
		queries:              queries,
		logger:              logger,
		taxService:          taxService,
		discountService:     discountService,
		gasSponsorshipService: gasSponsorshipService,
		currencyService:     currencyService,
		exchangeRateService: exchangeRateService,
	}
}

// InvoiceCreateParams contains parameters for creating an invoice
type InvoiceCreateParams struct {
	WorkspaceID    uuid.UUID
	CustomerID     uuid.UUID
	SubscriptionID *uuid.UUID
	Currency       string
	DueDate        *time.Time
	LineItems      []LineItemCreateParams
	DiscountCode   *string
	Metadata       map[string]interface{}
}

// LineItemCreateParams contains parameters for creating a line item
type LineItemCreateParams struct {
	Description       string
	Quantity          float64
	UnitAmountCents   int64
	ProductID         *uuid.UUID
	PriceID          *uuid.UUID
	SubscriptionID    *uuid.UUID
	PeriodStart      *time.Time
	PeriodEnd        *time.Time
	LineItemType      string // "product", "gas_fee", "tax", "discount"
	GasFeePaymentID  *uuid.UUID
	Metadata         map[string]interface{}
}

// InvoiceWithDetails contains invoice with all line items and calculations
type InvoiceWithDetails struct {
	Invoice            db.Invoice
	LineItems          []db.InvoiceLineItem
	ProductSubtotal    int64
	GasFeesSubtotal    int64
	SponsoredGasFees   int64
	TaxAmount          int64
	DiscountAmount     int64
	TotalAmount        int64
	CustomerTotal      int64 // Total after gas sponsorship
	TaxDetails         []TaxDetail
	CryptoAmounts      []CryptoAmount
}

// TaxDetail contains tax calculation details
type TaxDetail struct {
	JurisdictionID string  `json:"jurisdiction_id"`
	JurisdictionName string `json:"jurisdiction_name"`
	TaxRate        float64 `json:"tax_rate"`
	TaxAmountCents int64   `json:"tax_amount_cents"`
	TaxType        string  `json:"tax_type"` // "vat", "sales_tax", etc.
}

// CryptoAmount contains crypto payment amounts by token
type CryptoAmount struct {
	TokenID       uuid.UUID `json:"token_id"`
	NetworkID     uuid.UUID `json:"network_id"`
	CryptoAmount  string    `json:"crypto_amount"`
	TaxCryptoAmount string  `json:"tax_crypto_amount"`
}

// CreateInvoice creates a new invoice with line items
func (s *InvoiceService) CreateInvoice(ctx context.Context, params InvoiceCreateParams) (*InvoiceWithDetails, error) {
	// Generate invoice number
	invoiceNumber, err := s.generateInvoiceNumber(ctx, params.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate invoice number: %w", err)
	}

	// Get customer for tax calculation
	customer, err := s.queries.GetCustomer(ctx, params.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	// Calculate subtotal from line items
	var subtotalCents int64
	for _, item := range params.LineItems {
		if item.LineItemType == "product" {
			subtotalCents += int64(item.Quantity * float64(item.UnitAmountCents))
		}
	}

	// Apply discount if provided
	var discountCents int64
	if params.DiscountCode != nil && *params.DiscountCode != "" {
		discount, err := s.discountService.ApplyDiscount(ctx, DiscountApplicationParams{
			WorkspaceID:  params.WorkspaceID,
			CustomerID:   params.CustomerID,
			DiscountCode: *params.DiscountCode,
			AmountCents:  subtotalCents,
		})
		if err != nil {
			s.logger.Warn("Failed to apply discount", 
				zap.String("discount_code", *params.DiscountCode),
				zap.Error(err))
		} else {
			discountCents = discount.DiscountAmountCents
		}
	}

	// Calculate tax
	taxableAmount := subtotalCents - discountCents
	taxCalculation, err := s.taxService.CalculateTax(ctx, TaxCalculationParams{
		WorkspaceID:    params.WorkspaceID,
		CustomerID:     params.CustomerID,
		AmountCents:    taxableAmount,
		Currency:       params.Currency,
		TransactionType: "subscription",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to calculate tax: %w", err)
	}

	// Calculate total
	totalAmount := subtotalCents - discountCents + taxCalculation.TotalTaxCents

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(params.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Convert tax details to JSON
	taxDetailsJSON, err := json.Marshal(taxCalculation.TaxBreakdown)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tax details: %w", err)
	}

	// Create invoice
	invoice, err := s.queries.CreateInvoiceWithDetails(ctx, db.CreateInvoiceWithDetailsParams{
		WorkspaceID:             params.WorkspaceID,
		CustomerID:              pgtype.UUID{Bytes: params.CustomerID, Valid: true},
		SubscriptionID:          uuidToPgtype(params.SubscriptionID),
		InvoiceNumber:           pgtype.Text{String: invoiceNumber, Valid: true},
		Status:                  "draft",
		AmountDue:               int32(totalAmount),
		Currency:                params.Currency,
		SubtotalCents:           pgtype.Int8{Int64: subtotalCents, Valid: true},
		DiscountCents:           pgtype.Int8{Int64: discountCents, Valid: true},
		TaxAmountCents:          taxCalculation.TotalTaxCents,
		TaxDetails:              taxDetailsJSON,
		DueDate:                 timeToPgtype(params.DueDate),
		CustomerTaxID:           pgtype.Text{String: customer.TaxID.String, Valid: customer.TaxID.Valid},
		CustomerJurisdictionID:  pgtype.UUID{Valid: false}, // TODO: Convert jurisdiction to UUID
		ReverseChargeApplies:    pgtype.Bool{Bool: false, Valid: true}, // TODO: Get from tax calculation
		Metadata:                metadataJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	// Create line items
	var lineItems []db.InvoiceLineItem

	// Add product line items
	for _, item := range params.LineItems {
		if item.LineItemType != "product" {
			continue
		}

		lineItem, err := s.createLineItem(ctx, invoice.ID, params.Currency, item)
		if err != nil {
			return nil, fmt.Errorf("failed to create product line item: %w", err)
		}
		lineItems = append(lineItems, lineItem)
	}

	// Add discount line item if applicable
	if discountCents > 0 {
		discountLineItem, err := s.queries.CreateInvoiceLineItem(ctx, db.CreateInvoiceLineItemParams{
			InvoiceID:          invoice.ID,
			Description:        fmt.Sprintf("Discount: %s", *params.DiscountCode),
			Quantity:           pgtype.Numeric{Valid: true}, // Will be set to 1
			UnitAmountInCents:  -discountCents,
			AmountInCents:      -discountCents,
			FiatCurrency:       params.Currency,
			LineItemType:       pgtype.Text{String: "discount", Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create discount line item: %w", err)
		}
		lineItems = append(lineItems, discountLineItem)
	}

	// Add tax line items
	for _, taxDetail := range taxCalculation.TaxBreakdown {
		taxLineItem, err := s.queries.CreateInvoiceLineItem(ctx, db.CreateInvoiceLineItemParams{
			InvoiceID:          invoice.ID,
			Description:        fmt.Sprintf("Tax (%s)", taxDetail.Description),
			Quantity:           pgtype.Numeric{Valid: true}, // Will be set to 1
			UnitAmountInCents:  taxDetail.TaxAmountCents,
			AmountInCents:      taxDetail.TaxAmountCents,
			FiatCurrency:       params.Currency,
			LineItemType:       pgtype.Text{String: "tax", Valid: true},
			TaxRate:            pgtype.Numeric{Valid: true}, // Set to tax rate
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create tax line item: %w", err)
		}
		lineItems = append(lineItems, taxLineItem)
	}

	// Add gas fee line items if any
	gasFeesTotal, sponsoredGasFees, err := s.addGasFeeLineItems(ctx, invoice.ID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to add gas fee line items: %w", err)
	}

	// Return invoice with details
	return &InvoiceWithDetails{
		Invoice:           invoice,
		LineItems:         lineItems,
		ProductSubtotal:   subtotalCents,
		GasFeesSubtotal:   gasFeesTotal,
		SponsoredGasFees:  sponsoredGasFees,
		TaxAmount:         taxCalculation.TotalTaxCents,
		DiscountAmount:    discountCents,
		TotalAmount:       totalAmount + gasFeesTotal,
		CustomerTotal:     totalAmount + gasFeesTotal - sponsoredGasFees,
		TaxDetails:        convertTaxBreakdownToDetails(taxCalculation.TaxBreakdown),
	}, nil
}

// GetInvoiceWithDetails retrieves an invoice with all its line items and calculations
func (s *InvoiceService) GetInvoiceWithDetails(ctx context.Context, workspaceID, invoiceID uuid.UUID) (*InvoiceWithDetails, error) {
	// Get invoice
	invoice, err := s.queries.GetInvoiceByID(ctx, db.GetInvoiceByIDParams{
		ID:          invoiceID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	// Get line items
	lineItems, err := s.queries.GetInvoiceLineItems(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get line items: %w", err)
	}

	// Get subtotals
	subtotals, err := s.queries.GetInvoiceSubtotal(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice subtotals: %w", err)
	}

	// Get crypto amounts
	cryptoAmounts, err := s.queries.GetInvoiceCryptoAmounts(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get crypto amounts: %w", err)
	}

	// Parse tax details
	var taxDetails []TaxDetail
	if len(invoice.TaxDetails) > 0 {
		if err := json.Unmarshal(invoice.TaxDetails, &taxDetails); err != nil {
			s.logger.Error("Failed to unmarshal tax details", zap.Error(err))
		}
	}

	return &InvoiceWithDetails{
		Invoice:           invoice,
		LineItems:         lineItems,
		ProductSubtotal:   subtotals.ProductSubtotal,
		GasFeesSubtotal:   subtotals.CustomerGasFees,
		SponsoredGasFees:  subtotals.SponsoredGasFees,
		TaxAmount:         subtotals.TotalTax,
		DiscountAmount:    subtotals.TotalDiscount,
		TotalAmount:       int64(invoice.AmountDue),
		CustomerTotal:     int64(subtotals.CustomerTotal),
		TaxDetails:        taxDetails,
		CryptoAmounts:     convertCryptoAmounts(cryptoAmounts),
	}, nil
}

// FinalizeInvoice marks an invoice as finalized and ready for payment
func (s *InvoiceService) FinalizeInvoice(ctx context.Context, workspaceID, invoiceID uuid.UUID) (*db.Invoice, error) {
	// Get current invoice
	invoice, err := s.queries.GetInvoiceByID(ctx, db.GetInvoiceByIDParams{
		ID:          invoiceID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	// Check if invoice can be finalized
	if invoice.Status != "draft" {
		return nil, fmt.Errorf("invoice cannot be finalized: current status is %s", invoice.Status)
	}

	// Update invoice status
	updatedInvoice, err := s.queries.UpdateInvoice(ctx, db.UpdateInvoiceParams{
		ID:          invoiceID,
		WorkspaceID: workspaceID,
		Status:      "open",
		CustomerID:  invoice.CustomerID,
		SubscriptionID: invoice.SubscriptionID,
		CollectionMethod: invoice.CollectionMethod,
		AmountDue: invoice.AmountDue,
		AmountPaid: invoice.AmountPaid,
		AmountRemaining: invoice.AmountRemaining,
		Currency: invoice.Currency,
		DueDate: invoice.DueDate,
		PaidAt: invoice.PaidAt,
		InvoicePdf: invoice.InvoicePdf,
		HostedInvoiceUrl: invoice.HostedInvoiceUrl,
		ChargeID: invoice.ChargeID,
		PaymentIntentID: invoice.PaymentIntentID,
		LineItems: invoice.LineItems,
		TaxAmount: invoice.TaxAmount,
		TotalTaxAmounts: invoice.TotalTaxAmounts,
		BillingReason: invoice.BillingReason,
		PaidOutOfBand: invoice.PaidOutOfBand,
		AttemptCount: invoice.AttemptCount,
		NextPaymentAttempt: invoice.NextPaymentAttempt,
		Metadata: invoice.Metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to finalize invoice: %w", err)
	}

	return &updatedInvoice, nil
}

// Helper functions

func (s *InvoiceService) generateInvoiceNumber(ctx context.Context, workspaceID uuid.UUID) (string, error) {
	// Get next invoice number
	nextNumber, err := s.queries.GetNextInvoiceNumber(ctx, workspaceID)
	if err != nil {
		return "", fmt.Errorf("failed to get next invoice number: %w", err)
	}

	// Format invoice number (e.g., INV-2024-0001)
	year := time.Now().Year()
	invoiceNumber := fmt.Sprintf("INV-%d-%04d", year, nextNumber)

	return invoiceNumber, nil
}

func (s *InvoiceService) createLineItem(ctx context.Context, invoiceID uuid.UUID, currency string, params LineItemCreateParams) (db.InvoiceLineItem, error) {
	// Convert quantity to pgtype.Numeric
	quantity := pgtype.Numeric{}
	if err := quantity.Scan(params.Quantity); err != nil {
		return db.InvoiceLineItem{}, fmt.Errorf("failed to convert quantity: %w", err)
	}

	// Calculate amount
	amountCents := int64(params.Quantity * float64(params.UnitAmountCents))

	// Convert metadata
	metadataJSON, err := json.Marshal(params.Metadata)
	if err != nil {
		return db.InvoiceLineItem{}, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return s.queries.CreateInvoiceLineItem(ctx, db.CreateInvoiceLineItemParams{
		InvoiceID:         invoiceID,
		Description:       params.Description,
		Quantity:          quantity,
		UnitAmountInCents: params.UnitAmountCents,
		AmountInCents:     amountCents,
		FiatCurrency:      currency,
		SubscriptionID:    uuidToPgtype(params.SubscriptionID),
		ProductID:         uuidToPgtype(params.ProductID),
		PriceID:           uuidToPgtype(params.PriceID),
		PeriodStart:       timeToPgtype(params.PeriodStart),
		PeriodEnd:         timeToPgtype(params.PeriodEnd),
		LineItemType:      pgtype.Text{String: params.LineItemType, Valid: true},
		GasFeePaymentID:   uuidToPgtype(params.GasFeePaymentID),
		Metadata:          metadataJSON,
	})
}

func (s *InvoiceService) addGasFeeLineItems(ctx context.Context, invoiceID uuid.UUID, params InvoiceCreateParams) (gasFeesTotal, sponsoredGasFees int64, err error) {
	// Check for gas fee line items in params
	for _, item := range params.LineItems {
		if item.LineItemType != "gas_fee" {
			continue
		}

		// Check if this gas fee is sponsored
		isSponsored := false
		var sponsorType, sponsorName string
		
		if s.gasSponsorshipService != nil && item.GasFeePaymentID != nil {
			// Check sponsorship eligibility
			decision, err := s.gasSponsorshipService.ShouldSponsorGas(ctx, SponsorshipCheckParams{
				WorkspaceID:      params.WorkspaceID,
				CustomerID:       params.CustomerID,
				ProductID:        uuid.Nil, // Would come from product if applicable
				CustomerTier:     "standard", // TODO: Get actual customer tier
				GasCostUSDCents:  item.UnitAmountCents,
				TransactionType:  "invoice",
			})
			if err != nil {
				s.logger.Warn("Failed to check gas sponsorship eligibility",
					zap.Error(err),
					zap.String("customer_id", params.CustomerID.String()))
			} else if decision.ShouldSponsor {
				isSponsored = true
				sponsorType = decision.SponsorType
				sponsorName = decision.Reason
			}
		}

		// Create gas fee line item
		quantity := pgtype.Numeric{}
		if err := quantity.Scan(item.Quantity); err != nil {
			return 0, 0, fmt.Errorf("failed to convert quantity: %w", err)
		}

		amountCents := int64(item.Quantity * float64(item.UnitAmountCents))
		
		// Convert metadata
		metadataJSON, err := json.Marshal(item.Metadata)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to marshal metadata: %w", err)
		}

		lineItem, err := s.queries.CreateInvoiceLineItem(ctx, db.CreateInvoiceLineItemParams{
			InvoiceID:          invoiceID,
			Description:        item.Description,
			Quantity:           quantity,
			UnitAmountInCents:  item.UnitAmountCents,
			AmountInCents:      amountCents,
			FiatCurrency:       params.Currency,
			LineItemType:       pgtype.Text{String: "gas_fee", Valid: true},
			GasFeePaymentID:    uuidToPgtype(item.GasFeePaymentID),
			IsGasSponsored:     pgtype.Bool{Bool: isSponsored, Valid: true},
			GasSponsorType:     pgtype.Text{String: sponsorType, Valid: sponsorType != ""},
			GasSponsorName:     pgtype.Text{String: sponsorName, Valid: sponsorName != ""},
			Metadata:           metadataJSON,
		})
		if err != nil {
			return 0, 0, fmt.Errorf("failed to create gas fee line item: %w", err)
		}

		// Track totals
		gasFeesTotal += amountCents
		if isSponsored {
			sponsoredGasFees += amountCents
		}

		// If sponsored, record sponsorship transaction
		if isSponsored && s.gasSponsorshipService != nil {
			if err := s.gasSponsorshipService.RecordSponsoredTransaction(ctx, SponsorshipRecord{
				WorkspaceID:     params.WorkspaceID,
				PaymentID:       uuid.Nil, // Would be set when payment is processed
				GasCostUSDCents: amountCents,
				SponsorType:     sponsorType,
				SponsorID:       params.WorkspaceID, // Using workspace as sponsor for now
			}); err != nil {
				s.logger.Error("Failed to record gas sponsorship usage",
					zap.Error(err),
					zap.String("invoice_id", invoiceID.String()),
					zap.String("line_item_id", lineItem.ID.String()))
			}
		}
	}

	return gasFeesTotal, sponsoredGasFees, nil
}

// Utility functions

func uuidToPgtype(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func timeToPgtype(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func getFirstJurisdiction(jurisdictions []string) string {
	if len(jurisdictions) > 0 {
		return jurisdictions[0]
	}
	return ""
}

func convertTaxBreakdownToDetails(breakdown []TaxLineItem) []TaxDetail {
	var result []TaxDetail
	for _, item := range breakdown {
		result = append(result, TaxDetail{
			JurisdictionID:   item.Jurisdiction,
			JurisdictionName: item.Description,
			TaxRate:         item.Rate,
			TaxAmountCents:  item.TaxAmountCents,
			TaxType:         item.TaxType,
		})
	}
	return result
}

func convertCryptoAmounts(amounts []db.GetInvoiceCryptoAmountsRow) []CryptoAmount {
	var result []CryptoAmount
	for _, a := range amounts {
		result = append(result, CryptoAmount{
			TokenID:      a.TokenID.Bytes,
			NetworkID:    a.NetworkID.Bytes,
			CryptoAmount: fmt.Sprintf("%d", a.TotalCryptoAmount),
			TaxCryptoAmount: fmt.Sprintf("%d", a.TotalTaxCryptoAmount),
		})
	}
	return result
}

func convertNumericToString(n pgtype.Numeric) string {
	if !n.Valid {
		return "0"
	}
	// Convert pgtype.Numeric to string
	// This is a simplified version - in production you'd want proper decimal handling
	return fmt.Sprintf("%v", n)
}