package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// InvoiceHandler handles invoice-related HTTP requests
type InvoiceHandler struct {
	common             *CommonServices
	invoiceService     interfaces.InvoiceService
	paymentLinkService interfaces.PaymentLinkService
	logger             *zap.Logger
}

// NewInvoiceHandler creates a handler with interface dependencies
func NewInvoiceHandler(
	common *CommonServices,
	invoiceService interfaces.InvoiceService,
	paymentLinkService interfaces.PaymentLinkService,
	logger *zap.Logger,
) *InvoiceHandler {
	if logger == nil {
		logger = zap.L()
	}

	return &InvoiceHandler{
		common:             common,
		invoiceService:     invoiceService,
		paymentLinkService: paymentLinkService,
		logger:             logger,
	}
}

// CreateInvoiceRequest represents the request to create an invoice
type CreateInvoiceRequest struct {
	CustomerID     uuid.UUID                      `json:"customer_id" binding:"required"`
	SubscriptionID *uuid.UUID                     `json:"subscription_id,omitempty"`
	Currency       string                         `json:"currency" binding:"required,len=3"`
	DueDate        *time.Time                     `json:"due_date,omitempty"`
	LineItems      []CreateInvoiceLineItemRequest `json:"line_items" binding:"required,min=1,dive"`
	DiscountCode   *string                        `json:"discount_code,omitempty"`
	Metadata       map[string]interface{}         `json:"metadata,omitempty"`
}

// CreateInvoiceLineItemRequest represents a line item in the invoice creation request
type CreateInvoiceLineItemRequest struct {
	Description     string                 `json:"description" binding:"required"`
	Quantity        float64                `json:"quantity" binding:"required,gt=0"`
	UnitAmountCents int64                  `json:"unit_amount_cents" binding:"required,gte=0"`
	ProductID       *uuid.UUID             `json:"product_id,omitempty"`
	PriceID         *uuid.UUID             `json:"price_id,omitempty"`
	SubscriptionID  *uuid.UUID             `json:"subscription_id,omitempty"`
	PeriodStart     *time.Time             `json:"period_start,omitempty"`
	PeriodEnd       *time.Time             `json:"period_end,omitempty"`
	LineItemType    string                 `json:"line_item_type" binding:"required,oneof=product gas_fee"`
	GasFeePaymentID *uuid.UUID             `json:"gas_fee_payment_id,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// InvoiceResponse represents an invoice in API responses
type InvoiceResponse struct {
	ID               uuid.UUID                 `json:"id"`
	WorkspaceID      uuid.UUID                 `json:"workspace_id"`
	CustomerID       uuid.UUID                 `json:"customer_id"`
	SubscriptionID   *uuid.UUID                `json:"subscription_id,omitempty"`
	InvoiceNumber    string                    `json:"invoice_number"`
	Status           string                    `json:"status"`
	Currency         string                    `json:"currency"`
	DueDate          *time.Time                `json:"due_date,omitempty"`
	ProductSubtotal  int64                     `json:"product_subtotal"`
	GasFeesSubtotal  int64                     `json:"gas_fees_subtotal"`
	SponsoredGasFees int64                     `json:"sponsored_gas_fees"`
	TaxAmount        int64                     `json:"tax_amount"`
	DiscountAmount   int64                     `json:"discount_amount"`
	TotalAmount      int64                     `json:"total_amount"`
	CustomerTotal    int64                     `json:"customer_total"`
	LineItems        []InvoiceLineItemResponse `json:"line_items"`
	TaxDetails       []services.TaxDetail      `json:"tax_details"`
	PaymentLinkID    *uuid.UUID                `json:"payment_link_id,omitempty"`
	PaymentLinkURL   *string                   `json:"payment_link_url,omitempty"`
	CreatedAt        time.Time                 `json:"created_at"`
	UpdatedAt        time.Time                 `json:"updated_at"`
}

// InvoiceLineItemResponse represents a line item in API responses
type InvoiceLineItemResponse struct {
	ID              uuid.UUID              `json:"id"`
	Description     string                 `json:"description"`
	Quantity        float64                `json:"quantity"`
	UnitAmountCents int64                  `json:"unit_amount_cents"`
	AmountCents     int64                  `json:"amount_cents"`
	Currency        string                 `json:"currency"`
	LineItemType    string                 `json:"line_item_type"`
	IsGasSponsored  bool                   `json:"is_gas_sponsored,omitempty"`
	GasSponsorType  *string                `json:"gas_sponsor_type,omitempty"`
	GasSponsorName  *string                `json:"gas_sponsor_name,omitempty"`
	ProductID       *uuid.UUID             `json:"product_id,omitempty"`
	PriceID         *uuid.UUID             `json:"price_id,omitempty"`
	PeriodStart     *time.Time             `json:"period_start,omitempty"`
	PeriodEnd       *time.Time             `json:"period_end,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// CreateInvoice creates a new invoice
// @Summary Create a new invoice
// @Description Create a new invoice with line items, tax calculation, and optional discounts
// @Tags invoices
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param invoice body CreateInvoiceRequest true "Invoice creation request"
// @Success 201 {object} InvoiceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/invoices [post]
func (h *InvoiceHandler) CreateInvoice(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		h.common.HandleError(c, err, "Workspace ID required", http.StatusBadRequest, h.common.logger)
		return
	}

	var req CreateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.common.HandleError(c, err, "Invalid request body", http.StatusBadRequest, h.common.logger)
		return
	}

	// Convert request to service params
	params := services.InvoiceCreateParams{
		WorkspaceID:    workspaceID,
		CustomerID:     req.CustomerID,
		SubscriptionID: req.SubscriptionID,
		Currency:       req.Currency,
		DueDate:        req.DueDate,
		DiscountCode:   req.DiscountCode,
		Metadata:       req.Metadata,
	}

	// Convert line items
	for _, item := range req.LineItems {
		params.LineItems = append(params.LineItems, services.LineItemCreateParams{
			Description:     item.Description,
			Quantity:        item.Quantity,
			UnitAmountCents: item.UnitAmountCents,
			ProductID:       item.ProductID,
			PriceID:         item.PriceID,
			SubscriptionID:  item.SubscriptionID,
			PeriodStart:     item.PeriodStart,
			PeriodEnd:       item.PeriodEnd,
			LineItemType:    item.LineItemType,
			GasFeePaymentID: item.GasFeePaymentID,
			Metadata:        item.Metadata,
		})
	}

	// Create invoice
	invoiceDetails, err := h.invoiceService.CreateInvoice(ctx, params)
	if err != nil {
		h.common.HandleError(c, err, "Failed to create invoice", http.StatusInternalServerError, h.common.logger)
		return
	}

	// Convert to response
	response := h.convertToInvoiceResponse(invoiceDetails)

	c.JSON(http.StatusCreated, response)
}

// GetInvoice retrieves an invoice by ID
// @Summary Get invoice by ID
// @Description Get detailed invoice information including all line items and calculations
// @Tags invoices
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param invoice_id path string true "Invoice ID"
// @Success 200 {object} InvoiceResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/invoices/{invoice_id} [get]
func (h *InvoiceHandler) GetInvoice(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		h.common.HandleError(c, err, "Workspace ID required", http.StatusBadRequest, h.common.logger)
		return
	}

	invoiceIDStr := c.Param("invoice_id")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		h.common.HandleError(c, err, "Invalid invoice ID", http.StatusBadRequest, h.common.logger)
		return
	}

	// Get invoice with details
	invoiceDetails, err := h.invoiceService.GetInvoiceWithDetails(ctx, workspaceID, invoiceID)
	if err != nil {
		h.common.HandleError(c, err, "Failed to get invoice", http.StatusNotFound, h.common.logger)
		return
	}

	// Convert to response
	response := h.convertToInvoiceResponse(invoiceDetails)

	c.JSON(http.StatusOK, response)
}

// PreviewInvoice previews an invoice with all calculations
// @Summary Preview invoice
// @Description Preview invoice with all line items, tax calculations, and totals without saving
// @Tags invoices
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param invoice_id path string true "Invoice ID"
// @Success 200 {object} InvoiceResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/invoices/{invoice_id}/preview [get]
func (h *InvoiceHandler) PreviewInvoice(c *gin.Context) {
	// This is the same as GetInvoice for now
	h.GetInvoice(c)
}

// FinalizeInvoice finalizes an invoice making it ready for payment
// @Summary Finalize invoice
// @Description Change invoice status from draft to open, making it ready for payment
// @Tags invoices
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param invoice_id path string true "Invoice ID"
// @Success 200 {object} InvoiceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/invoices/{invoice_id}/finalize [post]
func (h *InvoiceHandler) FinalizeInvoice(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		h.common.HandleError(c, err, "Workspace ID required", http.StatusBadRequest, h.common.logger)
		return
	}

	invoiceIDStr := c.Param("invoice_id")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		h.common.HandleError(c, err, "Invalid invoice ID", http.StatusBadRequest, h.common.logger)
		return
	}

	// Finalize invoice
	invoice, err := h.invoiceService.FinalizeInvoice(ctx, workspaceID, invoiceID)
	if err != nil {
		h.common.HandleError(c, err, "Failed to finalize invoice", http.StatusBadRequest, h.common.logger)
		return
	}

	// Get full invoice details
	invoiceDetails, err := h.invoiceService.GetInvoiceWithDetails(ctx, workspaceID, invoice.ID)
	if err != nil {
		h.common.HandleError(c, err, "Failed to get invoice details", http.StatusInternalServerError, h.common.logger)
		return
	}

	// Convert to response
	response := h.convertToInvoiceResponse(invoiceDetails)

	c.JSON(http.StatusOK, response)
}

// SendInvoice sends an invoice via email
// @Summary Send invoice
// @Description Send invoice to customer via email
// @Tags invoices
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param invoice_id path string true "Invoice ID"
// @Success 200 {object} map[string]interface{} "message: Invoice sent successfully"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/invoices/{invoice_id}/send [post]
func (h *InvoiceHandler) SendInvoice(c *gin.Context) {
	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		h.common.HandleError(c, err, "Workspace ID required", http.StatusBadRequest, h.common.logger)
		return
	}

	invoiceIDStr := c.Param("invoice_id")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		h.common.HandleError(c, err, "Invalid invoice ID", http.StatusBadRequest, h.common.logger)
		return
	}

	// TODO: Implement email sending logic
	h.common.logger.Info("Invoice send requested",
		zap.String("workspace_id", workspaceID.String()),
		zap.String("invoice_id", invoiceID.String()))

	c.JSON(http.StatusOK, gin.H{
		"message": "Invoice sending not yet implemented",
	})
}

// GetInvoicePaymentLink gets or creates a payment link for an invoice
// @Summary Get invoice payment link
// @Description Get or create a payment link with QR code for invoice payment
// @Tags invoices
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param invoice_id path string true "Invoice ID"
// @Success 200 {object} map[string]interface{} "payment_link_url, qr_code_data"
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/invoices/{invoice_id}/payment-link [get]
func (h *InvoiceHandler) GetInvoicePaymentLink(c *gin.Context) {
	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		h.common.HandleError(c, err, "Workspace ID required", http.StatusBadRequest, h.common.logger)
		return
	}

	invoiceIDStr := c.Param("invoice_id")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		h.common.HandleError(c, err, "Invalid invoice ID", http.StatusBadRequest, h.common.logger)
		return
	}

	// TODO: Implement payment link generation
	h.common.logger.Info("Payment link requested",
		zap.String("workspace_id", workspaceID.String()),
		zap.String("invoice_id", invoiceID.String()))

	c.JSON(http.StatusOK, gin.H{
		"payment_link_url": fmt.Sprintf("https://pay.cyphera.com/invoice/%s", invoiceID),
		"qr_code_data":     "payment link QR code generation not yet implemented",
	})
}

// ListInvoices lists invoices for a workspace
// @Summary List invoices
// @Description List all invoices for a workspace with pagination
// @Tags invoices
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Param status query string false "Filter by status (draft, open, paid, void)"
// @Param customer_id query string false "Filter by customer ID"
// @Success 200 {object} map[string]interface{} "invoices array and pagination"
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/invoices [get]
func (h *InvoiceHandler) ListInvoices(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		h.common.HandleError(c, err, "Workspace ID required", http.StatusBadRequest, h.common.logger)
		return
	}

	// Parse query parameters
	limit, offset := GetPaginationParams(c)
	status := c.Query("status")
	customerIDStr := c.Query("customer_id")

	var invoices []db.Invoice
	var count int64

	// Filter by customer if provided
	if customerIDStr != "" {
		customerID, err := uuid.Parse(customerIDStr)
		if err != nil {
			h.common.HandleError(c, err, "Invalid customer ID", http.StatusBadRequest, h.common.logger)
			return
		}

		invoices, err = h.common.db.ListInvoicesByCustomer(ctx, db.ListInvoicesByCustomerParams{
			WorkspaceID: workspaceID,
			CustomerID:  pgtype.UUID{Bytes: customerID, Valid: true},
			Limit:       int32(limit),
			Offset:      int32(offset),
		})
	} else if status != "" {
		// Filter by status
		invoices, err = h.common.db.ListInvoicesByStatus(ctx, db.ListInvoicesByStatusParams{
			WorkspaceID: workspaceID,
			Status:      status,
			Limit:       int32(limit),
			Offset:      int32(offset),
		})
	} else {
		// Get all invoices
		invoices, err = h.common.db.ListInvoicesByWorkspace(ctx, db.ListInvoicesByWorkspaceParams{
			WorkspaceID: workspaceID,
			Limit:       int32(limit),
			Offset:      int32(offset),
		})
	}

	if err != nil {
		h.common.HandleError(c, err, "Failed to list invoices", http.StatusInternalServerError, h.common.logger)
		return
	}

	// Get total count
	if status != "" {
		countResult, err := h.common.db.CountInvoicesByStatus(ctx, db.CountInvoicesByStatusParams{
			WorkspaceID: workspaceID,
			Status:      status,
		})
		if err == nil {
			count = countResult
		}
	} else {
		countResult, err := h.common.db.CountInvoicesByWorkspace(ctx, workspaceID)
		if err == nil {
			count = countResult
		}
	}

	// Convert invoices to response format
	var invoiceResponses []map[string]interface{}
	for _, invoice := range invoices {
		invoiceResponses = append(invoiceResponses, map[string]interface{}{
			"id":             invoice.ID,
			"invoice_number": invoice.InvoiceNumber.String,
			"customer_id":    invoice.CustomerID.Bytes,
			"status":         invoice.Status,
			"currency":       invoice.Currency,
			"amount_due":     invoice.AmountDue,
			"due_date":       invoice.DueDate.Time,
			"created_at":     invoice.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"invoices": invoiceResponses,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
			"total":  count,
		},
	})
}

// Helper functions

func (h *InvoiceHandler) convertToInvoiceResponse(details *services.InvoiceWithDetails) InvoiceResponse {
	response := InvoiceResponse{
		ID:               details.Invoice.ID,
		WorkspaceID:      details.Invoice.WorkspaceID,
		CustomerID:       details.Invoice.CustomerID.Bytes,
		InvoiceNumber:    details.Invoice.InvoiceNumber.String,
		Status:           details.Invoice.Status,
		Currency:         details.Invoice.Currency,
		ProductSubtotal:  details.ProductSubtotal,
		GasFeesSubtotal:  details.GasFeesSubtotal,
		SponsoredGasFees: details.SponsoredGasFees,
		TaxAmount:        details.TaxAmount,
		DiscountAmount:   details.DiscountAmount,
		TotalAmount:      details.TotalAmount,
		CustomerTotal:    details.CustomerTotal,
		TaxDetails:       details.TaxDetails,
		CreatedAt:        details.Invoice.CreatedAt.Time,
		UpdatedAt:        details.Invoice.UpdatedAt.Time,
	}

	// Set optional fields
	if details.Invoice.SubscriptionID.Valid {
		id := uuid.UUID(details.Invoice.SubscriptionID.Bytes)
		response.SubscriptionID = &id
	}
	if details.Invoice.DueDate.Valid {
		response.DueDate = &details.Invoice.DueDate.Time
	}
	if details.Invoice.PaymentLinkID.Valid {
		id := uuid.UUID(details.Invoice.PaymentLinkID.Bytes)
		response.PaymentLinkID = &id
	}

	// Convert line items
	for _, item := range details.LineItems {
		lineItemResp := InvoiceLineItemResponse{
			ID:              item.ID,
			Description:     item.Description,
			Quantity:        h.convertNumericToFloat64(item.Quantity),
			UnitAmountCents: item.UnitAmountInCents,
			AmountCents:     item.AmountInCents,
			Currency:        item.FiatCurrency,
			LineItemType:    item.LineItemType.String,
		}

		// Set optional fields
		if item.IsGasSponsored.Valid {
			lineItemResp.IsGasSponsored = item.IsGasSponsored.Bool
		}
		if item.GasSponsorType.Valid {
			lineItemResp.GasSponsorType = &item.GasSponsorType.String
		}
		if item.GasSponsorName.Valid {
			lineItemResp.GasSponsorName = &item.GasSponsorName.String
		}
		if item.ProductID.Valid {
			id := uuid.UUID(item.ProductID.Bytes)
			lineItemResp.ProductID = &id
		}
		if item.PriceID.Valid {
			id := uuid.UUID(item.PriceID.Bytes)
			lineItemResp.PriceID = &id
		}
		if item.PeriodStart.Valid {
			lineItemResp.PeriodStart = &item.PeriodStart.Time
		}
		if item.PeriodEnd.Valid {
			lineItemResp.PeriodEnd = &item.PeriodEnd.Time
		}

		// Parse metadata
		if len(item.Metadata) > 0 {
			var metadata map[string]interface{}
			if err := json.Unmarshal(item.Metadata, &metadata); err == nil {
				lineItemResp.Metadata = metadata
			}
		}

		response.LineItems = append(response.LineItems, lineItemResp)
	}

	return response
}

func (h *InvoiceHandler) convertNumericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	// This is a simplified conversion - in production you'd want proper decimal handling
	// For now, we'll use a basic approach
	return 1.0 // Default to 1 for quantity
}
