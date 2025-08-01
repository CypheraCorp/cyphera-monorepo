package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// InvoiceService handles invoice creation and management
type InvoiceService struct {
	queries               db.Querier
	logger                *zap.Logger
	taxService            *TaxService
	discountService       *DiscountService
	gasSponsorshipService *GasSponsorshipService
	currencyService       *CurrencyService
	exchangeRateService   *ExchangeRateService
}

// NewInvoiceService creates a new invoice service
func NewInvoiceService(
	queries db.Querier,
	logger *zap.Logger,
	taxService *TaxService,
	discountService *DiscountService,
	gasSponsorshipService *GasSponsorshipService,
	currencyService *CurrencyService,
	exchangeRateService *ExchangeRateService,
) *InvoiceService {
	return &InvoiceService{
		queries:               queries,
		logger:                logger,
		taxService:            taxService,
		discountService:       discountService,
		gasSponsorshipService: gasSponsorshipService,
		currencyService:       currencyService,
		exchangeRateService:   exchangeRateService,
	}
}

// CreateInvoice creates a new invoice with line items
func (s *InvoiceService) CreateInvoice(ctx context.Context, invoiceParams params.InvoiceCreateParams) (*responses.InvoiceResponse, error) {
	// If subscription ID is provided and no line items, generate from subscription
	if invoiceParams.SubscriptionID != nil && len(invoiceParams.LineItems) == 0 {
		// Determine period dates
		periodStart := time.Now()
		periodEnd := periodStart.AddDate(0, 1, 0) // Default to one month
		if invoiceParams.PeriodStart != nil && invoiceParams.PeriodEnd != nil {
			periodStart = *invoiceParams.PeriodStart
			periodEnd = *invoiceParams.PeriodEnd
		}
		
		// Generate invoice from subscription
		return s.GenerateInvoiceFromSubscription(ctx, *invoiceParams.SubscriptionID, periodStart, periodEnd, invoiceParams.Status == "draft")
	}

	// Generate invoice number
	invoiceNumber, err := s.generateInvoiceNumber(ctx, invoiceParams.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate invoice number: %w", err)
	}

	// Get customer for tax calculation
	customer, err := s.queries.GetCustomer(ctx, invoiceParams.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	// Calculate subtotal from line items
	var subtotalCents int64
	for _, item := range invoiceParams.LineItems {
		if item.LineItemType == "product" {
			subtotalCents += int64(item.Quantity * float64(item.UnitAmountCents))
		}
	}

	// Apply discount if provided
	var discountCents int64
	if invoiceParams.DiscountCode != nil && *invoiceParams.DiscountCode != "" {
		discount, err := s.discountService.ApplyDiscount(ctx, params.DiscountApplicationParams{
			WorkspaceID:  invoiceParams.WorkspaceID,
			CustomerID:   invoiceParams.CustomerID,
			DiscountCode: *invoiceParams.DiscountCode,
			AmountCents:  subtotalCents,
		})
		if err != nil {
			s.logger.Warn("Failed to apply discount",
				zap.String("discount_code", *invoiceParams.DiscountCode),
				zap.Error(err))
		} else {
			discountCents = discount.DiscountAmountCents
		}
	}

	// Calculate tax
	taxableAmount := subtotalCents - discountCents
	taxCalculation, err := s.taxService.CalculateTax(ctx, params.TaxCalculationParams{
		WorkspaceID:     invoiceParams.WorkspaceID,
		CustomerID:      invoiceParams.CustomerID,
		AmountCents:     taxableAmount,
		Currency:        invoiceParams.Currency,
		TransactionType: "subscription",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to calculate tax: %w", err)
	}

	// Calculate total
	totalAmount := subtotalCents - discountCents + taxCalculation.TotalTaxCents

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(invoiceParams.Metadata)
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
		WorkspaceID:            invoiceParams.WorkspaceID,
		CustomerID:             pgtype.UUID{Bytes: invoiceParams.CustomerID, Valid: true},
		SubscriptionID:         uuidToPgtype(invoiceParams.SubscriptionID),
		InvoiceNumber:          pgtype.Text{String: invoiceNumber, Valid: true},
		Status:                 "draft",
		AmountDue:              int32(totalAmount),
		Currency:               invoiceParams.Currency,
		SubtotalCents:          pgtype.Int8{Int64: subtotalCents, Valid: true},
		DiscountCents:          pgtype.Int8{Int64: discountCents, Valid: true},
		TaxAmountCents:         taxCalculation.TotalTaxCents,
		TaxDetails:             taxDetailsJSON,
		DueDate:                timeToPgtype(invoiceParams.DueDate),
		CustomerTaxID:          pgtype.Text{String: customer.TaxID.String, Valid: customer.TaxID.Valid},
		CustomerJurisdictionID: pgtype.UUID{Valid: false},             // TODO: Convert jurisdiction to UUID
		ReverseChargeApplies:   pgtype.Bool{Bool: false, Valid: true}, // TODO: Get from tax calculation
		Metadata:               metadataJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	// Create line items
	var lineItems []db.InvoiceLineItem

	// Add product line items
	for _, item := range invoiceParams.LineItems {
		if item.LineItemType != "product" {
			continue
		}

		lineItem, err := s.createLineItem(ctx, invoice.ID, invoiceParams.Currency, item)
		if err != nil {
			return nil, fmt.Errorf("failed to create product line item: %w", err)
		}
		lineItems = append(lineItems, lineItem)
	}

	// Add discount line item if applicable
	if discountCents > 0 {
		quantity := pgtype.Numeric{}
		if err := quantity.Scan("1"); err != nil {
			return nil, fmt.Errorf("failed to convert discount quantity: %w", err)
		}

		discountLineItem, err := s.queries.CreateInvoiceLineItem(ctx, db.CreateInvoiceLineItemParams{
			InvoiceID:         invoice.ID,
			Description:       fmt.Sprintf("Discount: %s", *invoiceParams.DiscountCode),
			Quantity:          quantity,
			UnitAmountInCents: -discountCents,
			AmountInCents:     -discountCents,
			FiatCurrency:      invoiceParams.Currency,
			LineItemType:      pgtype.Text{String: "discount", Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create discount line item: %w", err)
		}
		lineItems = append(lineItems, discountLineItem)
	}

	// Add tax line items
	for _, taxDetail := range taxCalculation.TaxBreakdown {
		quantity := pgtype.Numeric{}
		if err := quantity.Scan("1"); err != nil {
			return nil, fmt.Errorf("failed to convert tax quantity: %w", err)
		}

		taxLineItem, err := s.queries.CreateInvoiceLineItem(ctx, db.CreateInvoiceLineItemParams{
			InvoiceID:         invoice.ID,
			Description:       fmt.Sprintf("Tax (%s)", taxDetail.Description),
			Quantity:          quantity,
			UnitAmountInCents: taxDetail.TaxAmountCents,
			AmountInCents:     taxDetail.TaxAmountCents,
			FiatCurrency:      invoiceParams.Currency,
			LineItemType:      pgtype.Text{String: "tax", Valid: true},
			TaxRate:           pgtype.Numeric{Valid: false},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create tax line item: %w", err)
		}
		lineItems = append(lineItems, taxLineItem)
	}

	// Add gas fee line items if any
	gasFeesTotal, sponsoredGasFees, err := s.addGasFeeLineItems(ctx, invoice.ID, invoiceParams)
	if err != nil {
		return nil, fmt.Errorf("failed to add gas fee line items: %w", err)
	}

	// Record invoice creation activity
	activityMetadata, _ := json.Marshal(map[string]interface{}{
		"line_items_count": len(lineItems),
		"has_discount":     discountCents > 0,
		"has_tax":          taxCalculation.TotalTaxCents > 0,
	})
	
	s.recordActivity(ctx, db.CreateInvoiceActivityParams{
		InvoiceID:    invoice.ID,
		WorkspaceID:  invoice.WorkspaceID,
		ActivityType: "created",
		ToStatus:     pgtype.Text{String: "draft", Valid: true},
		Description:  pgtype.Text{String: fmt.Sprintf("Invoice %s created", invoice.InvoiceNumber.String), Valid: true},
		Metadata:     activityMetadata,
	})

	// Convert to InvoiceResponse
	invoiceDetails := &business.InvoiceWithDetails{
		Invoice:          invoice,
		LineItems:        lineItems,
		ProductSubtotal:  subtotalCents,
		GasFeesSubtotal:  gasFeesTotal,
		SponsoredGasFees: sponsoredGasFees,
		TaxAmount:        taxCalculation.TotalTaxCents,
		DiscountAmount:   discountCents,
		TotalAmount:      totalAmount + gasFeesTotal,
		CustomerTotal:    totalAmount + gasFeesTotal - sponsoredGasFees,
		TaxDetails:       convertTaxBreakdownToDetails(taxCalculation.TaxBreakdown),
	}

	return convertToInvoiceResponse(invoiceDetails), nil
}

// GetInvoiceWithDetails retrieves an invoice with all its line items and calculations
func (s *InvoiceService) GetInvoiceWithDetails(ctx context.Context, workspaceID, invoiceID uuid.UUID) (*responses.InvoiceResponse, error) {
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
	var taxDetails []business.TaxDetail
	if len(invoice.TaxDetails) > 0 {
		if err := json.Unmarshal(invoice.TaxDetails, &taxDetails); err != nil {
			s.logger.Error("Failed to unmarshal tax details", zap.Error(err))
		}
	}

	invoiceDetails := &business.InvoiceWithDetails{
		Invoice:          invoice,
		LineItems:        lineItems,
		ProductSubtotal:  subtotals.ProductSubtotal,
		GasFeesSubtotal:  subtotals.CustomerGasFees,
		SponsoredGasFees: subtotals.SponsoredGasFees,
		TaxAmount:        subtotals.TotalTax,
		DiscountAmount:   subtotals.TotalDiscount,
		TotalAmount:      int64(invoice.AmountDue),
		CustomerTotal:    int64(subtotals.CustomerTotal),
		TaxDetails:       taxDetails,
		CryptoAmounts:    convertCryptoAmounts(cryptoAmounts),
	}

	return convertToInvoiceResponse(invoiceDetails), nil
}

// GenerateInvoiceFromSubscription creates an invoice from a subscription and its line items
func (s *InvoiceService) GenerateInvoiceFromSubscription(ctx context.Context, subscriptionID uuid.UUID, periodStart, periodEnd time.Time, isDraft bool) (*responses.InvoiceResponse, error) {
	// Get subscription with line items
	subscriptionRows, err := s.queries.GetSubscriptionWithLineItems(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription with line items: %w", err)
	}

	if len(subscriptionRows) == 0 {
		return nil, fmt.Errorf("subscription %s has no active line items", subscriptionID)
	}

	// Get subscription and customer details
	subscriptionDetails, err := s.queries.GetSubscriptionForInvoicing(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription details: %w", err)
	}

	// Check if invoice already exists for this period
	exists, err := s.queries.CheckInvoiceExistsForPeriod(ctx, db.CheckInvoiceExistsForPeriodParams{
		SubscriptionID: pgtype.UUID{Bytes: subscriptionID, Valid: true},
		WorkspaceID:    subscriptionDetails.WorkspaceID,
		PeriodStart:    timeToPgtype(&periodStart),
		PeriodEnd:      timeToPgtype(&periodEnd),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check existing invoice: %w", err)
	}

	if exists {
		return nil, fmt.Errorf("invoice already exists for subscription %s for period %s to %s", 
			subscriptionID, periodStart.Format("2006-01-02"), periodEnd.Format("2006-01-02"))
	}

	// Determine currency from base line item
	var currency string
	var subtotalCents int64
	for _, row := range subscriptionRows {
		if row.LineItemType == "base" {
			currency = row.LineItemCurrency
		}
		subtotalCents += int64(row.TotalAmountInPennies)
	}

	// Generate invoice number
	invoiceNumber, err := s.generateInvoiceNumber(ctx, subscriptionDetails.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate invoice number: %w", err)
	}

	// Calculate tax
	taxCalculation, err := s.taxService.CalculateTax(ctx, params.TaxCalculationParams{
		WorkspaceID:     subscriptionDetails.WorkspaceID,
		CustomerID:      subscriptionDetails.CustomerID,
		AmountCents:     subtotalCents,
		Currency:        currency,
		TransactionType: "subscription",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to calculate tax: %w", err)
	}

	// Calculate total
	totalAmount := subtotalCents + taxCalculation.TotalTaxCents

	// Set due date (30 days from period end)
	dueDate := periodEnd.AddDate(0, 0, 30)

	// Determine status
	status := "draft"
	if !isDraft {
		status = "open"
	}

	// Convert tax details to JSON
	taxDetailsJSON, err := json.Marshal(taxCalculation.TaxBreakdown)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tax details: %w", err)
	}

	// Create invoice
	invoice, err := s.queries.CreateInvoiceWithDetails(ctx, db.CreateInvoiceWithDetailsParams{
		WorkspaceID:            subscriptionDetails.WorkspaceID,
		CustomerID:             pgtype.UUID{Bytes: subscriptionDetails.CustomerID, Valid: true},
		SubscriptionID:         pgtype.UUID{Bytes: subscriptionID, Valid: true},
		InvoiceNumber:          pgtype.Text{String: invoiceNumber, Valid: true},
		Status:                 status,
		AmountDue:              int32(totalAmount),
		Currency:               currency,
		SubtotalCents:          pgtype.Int8{Int64: subtotalCents, Valid: true},
		DiscountCents:          pgtype.Int8{Int64: 0, Valid: true},
		TaxAmountCents:         taxCalculation.TotalTaxCents,
		TaxDetails:             taxDetailsJSON,
		DueDate:                timeToPgtype(&dueDate),
		CustomerTaxID:          pgtype.Text{String: subscriptionDetails.TaxID.String, Valid: subscriptionDetails.TaxID.Valid},
		CustomerJurisdictionID: pgtype.UUID{Valid: false},
		ReverseChargeApplies:   pgtype.Bool{Bool: false, Valid: true},
		Metadata:               []byte(`{"generated_from": "subscription"}`),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	// Create line items from subscription line items
	for _, row := range subscriptionRows {
		description := fmt.Sprintf("%s - %s", row.ProductName.String, row.ProductDescription.String)
		if row.LineItemType == "addon" {
			description = fmt.Sprintf("Add-on: %s", description)
		}

		quantity := pgtype.Numeric{}
		if err := quantity.Scan(fmt.Sprintf("%d", row.Quantity)); err != nil {
			return nil, fmt.Errorf("failed to convert quantity: %w", err)
		}

		_, err = s.queries.CreateInvoiceLineItemFromSubscription(ctx, db.CreateInvoiceLineItemFromSubscriptionParams{
			InvoiceID:         invoice.ID,
			SubscriptionID:    pgtype.UUID{Bytes: subscriptionID, Valid: true},
			ProductID:         pgtype.UUID{Bytes: row.ProductID, Valid: true},
			Description:       description,
			Quantity:          quantity,
			UnitAmountInCents: int64(row.UnitAmountInPennies),
			AmountInCents:     int64(row.TotalAmountInPennies),
			FiatCurrency:      row.LineItemCurrency,
			LineItemType:      pgtype.Text{String: "product", Valid: true},
			PeriodStart:       timeToPgtype(&periodStart),
			PeriodEnd:         timeToPgtype(&periodEnd),
			Metadata:          []byte(fmt.Sprintf(`{"line_item_type": "%s"}`, row.LineItemType)),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create line item for product %s: %w", row.ProductID, err)
		}
	}

	// Add tax line items
	for _, taxDetail := range taxCalculation.TaxBreakdown {
		quantity := pgtype.Numeric{}
		if err := quantity.Scan("1"); err != nil {
			return nil, fmt.Errorf("failed to convert tax quantity: %w", err)
		}

		_, err = s.queries.CreateInvoiceLineItem(ctx, db.CreateInvoiceLineItemParams{
			InvoiceID:         invoice.ID,
			Description:       fmt.Sprintf("Tax (%s)", taxDetail.Description),
			Quantity:          quantity,
			UnitAmountInCents: taxDetail.TaxAmountCents,
			AmountInCents:     taxDetail.TaxAmountCents,
			FiatCurrency:      currency,
			LineItemType:      pgtype.Text{String: "tax", Valid: true},
			SubscriptionID:    pgtype.UUID{Bytes: subscriptionID, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create tax line item: %w", err)
		}
	}

	// Get the created invoice with line items
	return s.GetInvoiceWithDetails(ctx, invoice.WorkspaceID, invoice.ID)
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
		ID:                 invoiceID,
		WorkspaceID:        workspaceID,
		Status:             "open",
		CustomerID:         invoice.CustomerID,
		SubscriptionID:     invoice.SubscriptionID,
		CollectionMethod:   invoice.CollectionMethod,
		AmountDue:          invoice.AmountDue,
		AmountPaid:         invoice.AmountPaid,
		AmountRemaining:    invoice.AmountRemaining,
		Currency:           invoice.Currency,
		DueDate:            invoice.DueDate,
		PaidAt:             invoice.PaidAt,
		InvoicePdf:         invoice.InvoicePdf,
		HostedInvoiceUrl:   invoice.HostedInvoiceUrl,
		ChargeID:           invoice.ChargeID,
		PaymentIntentID:    invoice.PaymentIntentID,
		LineItems:          invoice.LineItems,
		TaxAmount:          invoice.TaxAmount,
		TotalTaxAmounts:    invoice.TotalTaxAmounts,
		BillingReason:      invoice.BillingReason,
		PaidOutOfBand:      invoice.PaidOutOfBand,
		AttemptCount:       invoice.AttemptCount,
		NextPaymentAttempt: invoice.NextPaymentAttempt,
		Metadata:           invoice.Metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to finalize invoice: %w", err)
	}

	// Record status change activity
	s.recordActivity(ctx, db.CreateInvoiceActivityParams{
		InvoiceID:    invoiceID,
		WorkspaceID:  workspaceID,
		ActivityType: "status_changed",
		FromStatus:   pgtype.Text{String: "draft", Valid: true},
		ToStatus:     pgtype.Text{String: "open", Valid: true},
		Description:  pgtype.Text{String: "Invoice finalized and ready for payment", Valid: true},
		Metadata:     []byte("{}"),
	})

	return &updatedInvoice, nil
}

// VoidInvoice marks an invoice as void
func (s *InvoiceService) VoidInvoice(ctx context.Context, workspaceID, invoiceID uuid.UUID) (*db.Invoice, error) {
	// Void the invoice
	invoice, err := s.queries.VoidInvoice(ctx, db.VoidInvoiceParams{
		ID:          invoiceID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to void invoice: %w", err)
	}

	s.logger.Info("Invoice voided",
		zap.String("invoice_id", invoiceID.String()),
		zap.String("workspace_id", workspaceID.String()))

	// Record status change activity
	s.recordActivity(ctx, db.CreateInvoiceActivityParams{
		InvoiceID:    invoiceID,
		WorkspaceID:  workspaceID,
		ActivityType: "status_changed",
		FromStatus:   pgtype.Text{String: invoice.Status, Valid: true},
		ToStatus:     pgtype.Text{String: "void", Valid: true},
		Description:  pgtype.Text{String: "Invoice voided", Valid: true},
		Metadata:     []byte("{}"),
	})

	return &invoice, nil
}

// MarkInvoicePaid marks an invoice as paid
func (s *InvoiceService) MarkInvoicePaid(ctx context.Context, workspaceID, invoiceID uuid.UUID) (*db.Invoice, error) {
	// Mark the invoice as paid
	invoice, err := s.queries.MarkInvoicePaid(ctx, db.MarkInvoicePaidParams{
		ID:          invoiceID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to mark invoice as paid: %w", err)
	}

	s.logger.Info("Invoice marked as paid",
		zap.String("invoice_id", invoiceID.String()),
		zap.String("workspace_id", workspaceID.String()),
		zap.Time("paid_at", invoice.PaidAt.Time))

	// Record status change activity
	s.recordActivity(ctx, db.CreateInvoiceActivityParams{
		InvoiceID:    invoiceID,
		WorkspaceID:  workspaceID,
		ActivityType: "status_changed",
		FromStatus:   pgtype.Text{String: "open", Valid: true},
		ToStatus:     pgtype.Text{String: "paid", Valid: true},
		Description:  pgtype.Text{String: "Invoice marked as paid manually", Valid: true},
		Metadata:     []byte(`{"paid_out_of_band": true}`),
	})

	return &invoice, nil
}

// MarkInvoiceUncollectible marks an invoice as uncollectible (bad debt)
func (s *InvoiceService) MarkInvoiceUncollectible(ctx context.Context, workspaceID, invoiceID uuid.UUID) (*db.Invoice, error) {
	// Get current invoice
	invoice, err := s.queries.GetInvoiceByID(ctx, db.GetInvoiceByIDParams{
		ID:          invoiceID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	// Check if invoice can be marked as uncollectible
	if invoice.Status != "open" && invoice.Status != "past_due" {
		return nil, fmt.Errorf("invoice cannot be marked as uncollectible: current status is %s", invoice.Status)
	}

	// Update invoice status to uncollectible
	updatedInvoice, err := s.queries.UpdateInvoiceStatus(ctx, db.UpdateInvoiceStatusParams{
		ID:          invoiceID,
		WorkspaceID: workspaceID,
		Status:      "uncollectible",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to mark invoice as uncollectible: %w", err)
	}

	s.logger.Info("Invoice marked as uncollectible",
		zap.String("invoice_id", invoiceID.String()),
		zap.String("workspace_id", workspaceID.String()),
		zap.String("previous_status", invoice.Status))

	// Record status change activity
	s.recordActivity(ctx, db.CreateInvoiceActivityParams{
		InvoiceID:    invoiceID,
		WorkspaceID:  workspaceID,
		ActivityType: "status_changed",
		FromStatus:   pgtype.Text{String: invoice.Status, Valid: true},
		ToStatus:     pgtype.Text{String: "uncollectible", Valid: true},
		Description:  pgtype.Text{String: "Invoice marked as uncollectible", Valid: true},
		Metadata:     []byte("{}"),
	})

	return &updatedInvoice, nil
}

// DuplicateInvoice creates a copy of an existing invoice
func (s *InvoiceService) DuplicateInvoice(ctx context.Context, workspaceID, invoiceID uuid.UUID) (*responses.InvoiceResponse, error) {
	// Get original invoice with line items
	originalInvoice, err := s.GetInvoiceWithDetails(ctx, workspaceID, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original invoice: %w", err)
	}

	// Create new invoice params from original
	invoiceParams := params.InvoiceCreateParams{
		WorkspaceID:    workspaceID,
		CustomerID:     originalInvoice.CustomerID,
		SubscriptionID: originalInvoice.SubscriptionID,
		Currency:       originalInvoice.Currency,
		Status:         "draft", // Always create duplicates as draft
		DueDate:        nil,     // Reset due date
		DiscountCode:   nil,     // Don't copy discount
		Metadata: map[string]interface{}{
			"duplicated_from": originalInvoice.ID.String(),
			"duplicated_at":   time.Now().Format(time.RFC3339),
		},
	}

	// Copy line items (only product line items, not tax/discount)
	for _, item := range originalInvoice.LineItems {
		if item.LineItemType == "product" || item.LineItemType == "gas_fee" {
			invoiceParams.LineItems = append(invoiceParams.LineItems, params.LineItemCreateParams{
				Description:     item.Description,
				Quantity:        item.Quantity,
				UnitAmountCents: item.UnitAmountCents,
				ProductID:       item.ProductID,
				LineItemType:    item.LineItemType,
				Metadata:        item.Metadata,
			})
		}
	}

	// Create the duplicate invoice
	duplicatedInvoice, err := s.CreateInvoice(ctx, invoiceParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create duplicate invoice: %w", err)
	}

	s.logger.Info("Invoice duplicated",
		zap.String("original_invoice_id", invoiceID.String()),
		zap.String("duplicate_invoice_id", duplicatedInvoice.ID.String()),
		zap.String("workspace_id", workspaceID.String()))

	return duplicatedInvoice, nil
}

// GetInvoiceActivity retrieves the activity history for an invoice
func (s *InvoiceService) GetInvoiceActivity(ctx context.Context, workspaceID, invoiceID uuid.UUID, limit, offset int32) ([]db.InvoiceActivity, error) {
	// Verify invoice exists and belongs to workspace
	_, err := s.queries.GetInvoiceByID(ctx, db.GetInvoiceByIDParams{
		ID:          invoiceID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	// Get activities
	activities, err := s.queries.GetInvoiceActivities(ctx, db.GetInvoiceActivitiesParams{
		InvoiceID: invoiceID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice activities: %w", err)
	}

	return activities, nil
}

// BulkGenerateInvoices generates invoices for all due subscriptions
func (s *InvoiceService) BulkGenerateInvoices(ctx context.Context, workspaceID uuid.UUID, endDate time.Time, maxInvoices int32) (*responses.BulkInvoiceGenerationResult, error) {
	result := &responses.BulkInvoiceGenerationResult{
		Success: []responses.InvoiceResponse{},
		Failed:  []responses.BulkInvoiceError{},
	}

	// Get subscriptions that need invoicing
	subscriptions, err := s.queries.GetSubscriptionsForBulkInvoicing(ctx, db.GetSubscriptionsForBulkInvoicingParams{
		WorkspaceID:      workspaceID,
		CurrentPeriodEnd: pgtype.Timestamptz{Time: endDate, Valid: true},
		Limit:            maxInvoices,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriptions for bulk invoicing: %w", err)
	}

	// Process each subscription
	for _, sub := range subscriptions {
		// Generate invoice for subscription
		invoice, genErr := s.GenerateInvoiceFromSubscription(ctx, sub.ID, sub.CurrentPeriodStart.Time, sub.CurrentPeriodEnd.Time, false)
		if genErr != nil {
			s.logger.Error("Failed to generate invoice for subscription",
				zap.String("subscription_id", sub.ID.String()),
				zap.Error(genErr))
			
			result.Failed = append(result.Failed, responses.BulkInvoiceError{
				SubscriptionID: sub.ID,
				CustomerID:     sub.CustomerID,
				Error:          genErr.Error(),
			})
			continue
		}

		result.Success = append(result.Success, *invoice)
	}

	result.TotalProcessed = len(subscriptions)
	result.SuccessCount = len(result.Success)
	result.FailedCount = len(result.Failed)

	return result, nil
}

// GetInvoiceStats retrieves invoice statistics for a workspace
func (s *InvoiceService) GetInvoiceStats(ctx context.Context, workspaceID uuid.UUID, startDate, endDate time.Time) (*responses.InvoiceStatsResponse, error) {
	stats, err := s.queries.GetInvoiceStatsByWorkspace(ctx, db.GetInvoiceStatsByWorkspaceParams{
		WorkspaceID: workspaceID,
		CreatedAt:   pgtype.Timestamptz{Time: startDate, Valid: true},
		CreatedAt_2: pgtype.Timestamptz{Time: endDate, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice stats: %w", err)
	}

	// Get default currency for the workspace
	currency := "USD" // Default
	if s.currencyService != nil {
		settings, err := s.currencyService.GetWorkspaceCurrencySettings(ctx, workspaceID)
		if err == nil && settings != nil {
			currency = settings.DefaultCurrency
		}
	}

	// Type assert the interface{} values to int64
	totalOutstanding, _ := stats.TotalOutstandingCents.(int64)
	totalPaid, _ := stats.TotalPaidCents.(int64)
	totalUncollectible, _ := stats.TotalUncollectibleCents.(int64)

	return &responses.InvoiceStatsResponse{
		DraftCount:              stats.DraftCount,
		OpenCount:               stats.OpenCount,
		PaidCount:               stats.PaidCount,
		VoidCount:               stats.VoidCount,
		UncollectibleCount:      stats.UncollectibleCount,
		TotalCount:              stats.TotalCount,
		TotalOutstandingCents:   totalOutstanding,
		TotalPaidCents:          totalPaid,
		TotalUncollectibleCents: totalUncollectible,
		Currency:                currency,
		PeriodStart:             startDate,
		PeriodEnd:               endDate,
	}, nil
}

// recordActivity is a helper method to record invoice activities
func (s *InvoiceService) recordActivity(ctx context.Context, params db.CreateInvoiceActivityParams) error {
	_, err := s.queries.CreateInvoiceActivity(ctx, params)
	if err != nil {
		s.logger.Error("Failed to record invoice activity",
			zap.String("invoice_id", params.InvoiceID.String()),
			zap.String("activity_type", params.ActivityType),
			zap.Error(err))
		// Don't fail the main operation if activity recording fails
		return nil
	}
	return nil
}

// CreateInvoiceLineItemsFromSubscription creates invoice line items from subscription line items
func (s *InvoiceService) CreateInvoiceLineItemsFromSubscription(ctx context.Context, invoiceID, subscriptionID uuid.UUID, periodStart, periodEnd time.Time) error {
	// Get subscription with line items
	subscriptionRows, err := s.queries.GetSubscriptionWithLineItems(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription with line items: %w", err)
	}

	if len(subscriptionRows) == 0 {
		return fmt.Errorf("subscription %s has no active line items", subscriptionID)
	}

	// Get invoice to verify it exists and get currency
	invoice, err := s.queries.GetInvoiceByID(ctx, db.GetInvoiceByIDParams{
		ID:          invoiceID,
		WorkspaceID: subscriptionRows[0].WorkspaceID,
	})
	if err != nil {
		return fmt.Errorf("failed to get invoice: %w", err)
	}

	// Create line items from subscription line items
	for _, row := range subscriptionRows {
		description := fmt.Sprintf("%s - %s", row.ProductName.String, row.ProductDescription.String)
		if row.LineItemType == "addon" {
			description = fmt.Sprintf("Add-on: %s", description)
		}

		quantity := pgtype.Numeric{}
		if err := quantity.Scan(fmt.Sprintf("%d", row.Quantity)); err != nil {
			return fmt.Errorf("failed to convert quantity: %w", err)
		}

		_, err = s.queries.CreateInvoiceLineItemFromSubscription(ctx, db.CreateInvoiceLineItemFromSubscriptionParams{
			InvoiceID:         invoiceID,
			SubscriptionID:    pgtype.UUID{Bytes: subscriptionID, Valid: true},
			ProductID:         pgtype.UUID{Bytes: row.ProductID, Valid: true},
			Description:       description,
			Quantity:          quantity,
			UnitAmountInCents: int64(row.UnitAmountInPennies),
			AmountInCents:     int64(row.TotalAmountInPennies),
			FiatCurrency:      invoice.Currency,
			LineItemType:      pgtype.Text{String: "product", Valid: true},
			PeriodStart:       timeToPgtype(&periodStart),
			PeriodEnd:         timeToPgtype(&periodEnd),
			Metadata:          []byte(fmt.Sprintf(`{"line_item_type": "%s"}`, row.LineItemType)),
		})
		if err != nil {
			return fmt.Errorf("failed to create line item for product %s: %w", row.ProductID, err)
		}
	}

	return nil
}

// GeneratePendingInvoices generates invoices for all subscriptions that are due
func (s *InvoiceService) GeneratePendingInvoices(ctx context.Context, lookAheadDays int) ([]uuid.UUID, error) {
	// Calculate look-ahead date
	lookAheadDate := time.Now().AddDate(0, 0, lookAheadDays)
	
	// Get pending subscriptions
	pendingSubscriptions, err := s.queries.GetPendingInvoicesForGeneration(ctx, pgtype.Timestamptz{
		Time:  lookAheadDate,
		Valid: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get pending subscriptions: %w", err)
	}

	var generatedInvoiceIDs []uuid.UUID
	for _, sub := range pendingSubscriptions {
		// Calculate period dates
		periodStart := sub.CurrentPeriodEnd.Time
		periodEnd := sub.NextRedemptionDate.Time

		// Generate invoice
		invoice, err := s.GenerateInvoiceFromSubscription(ctx, sub.SubscriptionID, periodStart, periodEnd, false)
		if err != nil {
			s.logger.Error("Failed to generate invoice for subscription",
				zap.String("subscription_id", sub.SubscriptionID.String()),
				zap.Error(err))
			continue
		}

		generatedInvoiceIDs = append(generatedInvoiceIDs, invoice.ID)
		
		s.logger.Info("Generated invoice for subscription",
			zap.String("subscription_id", sub.SubscriptionID.String()),
			zap.String("invoice_id", invoice.ID.String()),
			zap.String("invoice_number", invoice.InvoiceNumber))
	}

	return generatedInvoiceIDs, nil
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

func (s *InvoiceService) createLineItem(ctx context.Context, invoiceID uuid.UUID, currency string, params params.LineItemCreateParams) (db.InvoiceLineItem, error) {
	// Convert quantity to pgtype.Numeric
	quantity := pgtype.Numeric{}
	if err := quantity.Scan(fmt.Sprintf("%.10f", params.Quantity)); err != nil {
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
		PeriodStart:       timeToPgtype(params.PeriodStart),
		PeriodEnd:         timeToPgtype(params.PeriodEnd),
		LineItemType:      pgtype.Text{String: params.LineItemType, Valid: true},
		GasFeePaymentID:   uuidToPgtype(params.GasFeePaymentID),
		Metadata:          metadataJSON,
	})
}

func (s *InvoiceService) addGasFeeLineItems(ctx context.Context, invoiceID uuid.UUID, invoiceParams params.InvoiceCreateParams) (gasFeesTotal, sponsoredGasFees int64, err error) {
	// Check for gas fee line items in params
	for _, item := range invoiceParams.LineItems {
		if item.LineItemType != "gas_fee" {
			continue
		}

		// Check if this gas fee is sponsored
		isSponsored := false
		var sponsorType, sponsorName string

		if s.gasSponsorshipService != nil && item.GasFeePaymentID != nil {
			// Check sponsorship eligibility
			decision, err := s.gasSponsorshipService.ShouldSponsorGas(ctx, params.SponsorshipCheckParams{
				WorkspaceID:     invoiceParams.WorkspaceID,
				CustomerID:      invoiceParams.CustomerID,
				ProductID:       uuid.Nil,   // Would come from product if applicable
				CustomerTier:    "standard", // TODO: Get actual customer tier
				GasCostUSDCents: item.UnitAmountCents,
				TransactionType: "invoice",
			})
			if err != nil {
				s.logger.Warn("Failed to check gas sponsorship eligibility",
					zap.Error(err),
					zap.String("customer_id", invoiceParams.CustomerID.String()))
			} else if decision.ShouldSponsor {
				isSponsored = true
				sponsorType = decision.SponsorType
				sponsorName = decision.Reason
			}
		}

		// Create gas fee line item
		quantity := pgtype.Numeric{}
		if err := quantity.Scan(fmt.Sprintf("%.10f", item.Quantity)); err != nil {
			return 0, 0, fmt.Errorf("failed to convert quantity: %w", err)
		}

		amountCents := int64(item.Quantity * float64(item.UnitAmountCents))

		// Convert metadata
		metadataJSON, err := json.Marshal(item.Metadata)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to marshal metadata: %w", err)
		}

		lineItem, err := s.queries.CreateInvoiceLineItem(ctx, db.CreateInvoiceLineItemParams{
			InvoiceID:         invoiceID,
			Description:       item.Description,
			Quantity:          quantity,
			UnitAmountInCents: item.UnitAmountCents,
			AmountInCents:     amountCents,
			FiatCurrency:      invoiceParams.Currency,
			LineItemType:      pgtype.Text{String: "gas_fee", Valid: true},
			GasFeePaymentID:   uuidToPgtype(item.GasFeePaymentID),
			IsGasSponsored:    pgtype.Bool{Bool: isSponsored, Valid: true},
			GasSponsorType:    pgtype.Text{String: sponsorType, Valid: sponsorType != ""},
			GasSponsorName:    pgtype.Text{String: sponsorName, Valid: sponsorName != ""},
			Metadata:          metadataJSON,
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
			if err := s.gasSponsorshipService.RecordSponsoredTransaction(ctx, business.SponsorshipRecord{
				WorkspaceID:     invoiceParams.WorkspaceID,
				PaymentID:       uuid.Nil, // Would be set when payment is processed
				GasCostUSDCents: amountCents,
				SponsorType:     sponsorType,
				SponsorID:       invoiceParams.WorkspaceID, // Using workspace as sponsor for now
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

func convertTaxBreakdownToDetails(breakdown []business.TaxLineItem) []business.TaxDetail {
	var result []business.TaxDetail
	for _, item := range breakdown {
		result = append(result, business.TaxDetail{
			JurisdictionID:   item.Jurisdiction,
			JurisdictionName: item.Description,
			TaxRate:          item.Rate,
			TaxAmountCents:   item.TaxAmountCents,
			TaxType:          item.TaxType,
		})
	}
	return result
}

func convertCryptoAmounts(amounts []db.GetInvoiceCryptoAmountsRow) []business.CryptoAmount {
	var result []business.CryptoAmount
	for _, a := range amounts {
		result = append(result, business.CryptoAmount{
			TokenID:         a.TokenID.Bytes,
			NetworkID:       a.NetworkID.Bytes,
			CryptoAmount:    fmt.Sprintf("%d", a.TotalCryptoAmount),
			TaxCryptoAmount: fmt.Sprintf("%d", a.TotalTaxCryptoAmount),
		})
	}
	return result
}

func convertToInvoiceResponse(details *business.InvoiceWithDetails) *responses.InvoiceResponse {
	var dueDate *time.Time
	if details.Invoice.DueDate.Valid {
		dueDate = &details.Invoice.DueDate.Time
	}

	var subscriptionID *uuid.UUID
	if details.Invoice.SubscriptionID.Valid {
		id := uuid.UUID(details.Invoice.SubscriptionID.Bytes)
		subscriptionID = &id
	}

	// Convert line items
	lineItems := make([]responses.InvoiceLineItemResponse, len(details.LineItems))
	for i, item := range details.LineItems {
		lineItems[i] = responses.InvoiceLineItemResponse{
			ID:              item.ID,
			Description:     item.Description,
			Quantity:        1.0, // Default quantity
			UnitAmountCents: item.UnitAmountInCents,
			AmountCents:     item.AmountInCents,
			Currency:        item.FiatCurrency,
			LineItemType:    item.LineItemType.String,
			ProductID:       nil, // TODO: Add if available
			PriceID:         nil, // TODO: Add if available
		}
	}

	// Convert tax details
	taxDetails := make([]responses.TaxDetail, len(details.TaxDetails))
	for i, tax := range details.TaxDetails {
		taxDetails[i] = responses.TaxDetail{
			JurisdictionID:   tax.JurisdictionID,
			JurisdictionName: tax.JurisdictionName,
			TaxRate:          tax.TaxRate,
			TaxAmountCents:   tax.TaxAmountCents,
			TaxType:          tax.TaxType,
		}
	}

	return &responses.InvoiceResponse{
		ID:               details.Invoice.ID,
		WorkspaceID:      details.Invoice.WorkspaceID,
		CustomerID:       details.Invoice.CustomerID.Bytes,
		SubscriptionID:   subscriptionID,
		InvoiceNumber:    details.Invoice.InvoiceNumber.String,
		Status:           details.Invoice.Status,
		Currency:         details.Invoice.Currency,
		DueDate:          dueDate,
		ProductSubtotal:  details.ProductSubtotal,
		GasFeesSubtotal:  details.GasFeesSubtotal,
		SponsoredGasFees: details.SponsoredGasFees,
		TaxAmount:        details.TaxAmount,
		DiscountAmount:   details.DiscountAmount,
		TotalAmount:      details.TotalAmount,
		CustomerTotal:    details.CustomerTotal,
		LineItems:        lineItems,
		TaxDetails:       taxDetails,
		PaymentLinkID:    nil, // TODO: Add if available
		PaymentLinkURL:   nil, // TODO: Add if available
		CreatedAt:        details.Invoice.CreatedAt.Time,
		UpdatedAt:        details.Invoice.UpdatedAt.Time,
	}
}
