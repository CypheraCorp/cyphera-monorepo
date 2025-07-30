package handlers

import (
	"fmt"
	"net/http"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"

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

// Type aliases for centralized types
type (
	CreateInvoiceRequest         = requests.CreateInvoiceRequest
	CreateInvoiceLineItemRequest = requests.CreateInvoiceLineItemRequest
	InvoiceResponse              = responses.InvoiceResponse
	InvoiceLineItemResponse      = responses.InvoiceLineItemResponse
)

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
	invoiceCreateParams := params.InvoiceCreateParams{
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
		invoiceCreateParams.LineItems = append(invoiceCreateParams.LineItems, params.LineItemCreateParams{
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
	invoiceDetails, err := h.invoiceService.CreateInvoice(ctx, invoiceCreateParams)
	if err != nil {
		h.common.HandleError(c, err, "Failed to create invoice", http.StatusInternalServerError, h.common.logger)
		return
	}

	c.JSON(http.StatusCreated, invoiceDetails)
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

	c.JSON(http.StatusOK, invoiceDetails)
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

	c.JSON(http.StatusOK, invoiceDetails)
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
	limit, offset, err := helpers.ParsePaginationParamsAsInt(c)
	if err != nil {
		h.common.HandleError(c, err, "Invalid pagination parameters", http.StatusBadRequest, h.common.logger)
		return
	}
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
			Limit:       int32(limit),  // ParsePaginationParamsAsInt validates limit <= 100
			Offset:      int32(offset), // ParsePaginationParamsAsInt validates offset >= 0
		})
	} else if status != "" {
		// Filter by status
		invoices, err = h.common.db.ListInvoicesByStatus(ctx, db.ListInvoicesByStatusParams{
			WorkspaceID: workspaceID,
			Status:      status,
			Limit:       int32(limit),  // ParsePaginationParamsAsInt validates limit <= 100
			Offset:      int32(offset), // ParsePaginationParamsAsInt validates offset >= 0
		})
	} else {
		// Get all invoices
		invoices, err = h.common.db.ListInvoicesByWorkspace(ctx, db.ListInvoicesByWorkspaceParams{
			WorkspaceID: workspaceID,
			Limit:       int32(limit),  // ParsePaginationParamsAsInt validates limit <= 100
			Offset:      int32(offset), // ParsePaginationParamsAsInt validates offset >= 0
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
		if err != nil {
			h.common.logger.Error("Failed to count invoices by status", zap.Error(err))
		} else {
			count = countResult
		}
	} else {
		countResult, err := h.common.db.CountInvoicesByWorkspace(ctx, workspaceID)
		if err != nil {
			h.common.logger.Error("Failed to count invoices", zap.Error(err))
		} else {
			count = countResult
		}
	}

	// Convert invoices to response format
	invoiceResponses := make([]map[string]interface{}, 0, len(invoices))
	for _, invoice := range invoices {
		invoiceResponses = append(invoiceResponses, map[string]interface{}{
			"id":             invoice.ID,
			"invoice_number": invoice.InvoiceNumber.String,
			"customer_id":    uuid.UUID(invoice.CustomerID.Bytes),
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
