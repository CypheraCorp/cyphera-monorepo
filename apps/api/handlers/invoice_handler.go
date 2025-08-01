package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

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
		customerID, parseErr := uuid.Parse(customerIDStr)
		if parseErr != nil {
			h.common.HandleError(c, parseErr, "Invalid customer ID", http.StatusBadRequest, h.common.logger)
			return
		}

		invoices, err = h.common.db.ListInvoicesByCustomer(ctx, db.ListInvoicesByCustomerParams{
			WorkspaceID: workspaceID,
			CustomerID:  pgtype.UUID{Bytes: customerID, Valid: true},
			Limit:       int32(limit),  // #nosec G115 -- ParsePaginationParamsAsInt validates limit <= 100
			Offset:      int32(offset), // #nosec G115 -- ParsePaginationParamsAsInt validates offset >= 0
		})
	} else if status != "" {
		// Filter by status
		invoices, err = h.common.db.ListInvoicesByStatus(ctx, db.ListInvoicesByStatusParams{
			WorkspaceID: workspaceID,
			Status:      status,
			Limit:       int32(limit),  // #nosec G115 -- ParsePaginationParamsAsInt validates limit <= 100
			Offset:      int32(offset), // #nosec G115 -- ParsePaginationParamsAsInt validates offset >= 0
		})
	} else {
		// Get all invoices
		invoices, err = h.common.db.ListInvoicesByWorkspace(ctx, db.ListInvoicesByWorkspaceParams{
			WorkspaceID: workspaceID,
			Limit:       int32(limit),  // #nosec G115 -- ParsePaginationParamsAsInt validates limit <= 100
			Offset:      int32(offset), // #nosec G115 -- ParsePaginationParamsAsInt validates offset >= 0
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

// VoidInvoice voids an open or draft invoice
// @Summary Void invoice
// @Description Void an invoice, preventing it from being paid
// @Tags invoices
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param invoice_id path string true "Invoice ID"
// @Success 200 {object} InvoiceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/invoices/{invoice_id}/void [post]
func (h *InvoiceHandler) VoidInvoice(c *gin.Context) {
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

	// Void the invoice
	invoice, err := h.invoiceService.VoidInvoice(ctx, workspaceID, invoiceID)
	if err != nil {
		h.common.HandleError(c, err, "Failed to void invoice", http.StatusBadRequest, h.common.logger)
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

// MarkInvoicePaid manually marks an invoice as paid
// @Summary Mark invoice as paid
// @Description Manually mark an invoice as paid (for out-of-band payments)
// @Tags invoices
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param invoice_id path string true "Invoice ID"
// @Success 200 {object} InvoiceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/invoices/{invoice_id}/mark-paid [post]
func (h *InvoiceHandler) MarkInvoicePaid(c *gin.Context) {
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

	// Mark invoice as paid
	invoice, err := h.invoiceService.MarkInvoicePaid(ctx, workspaceID, invoiceID)
	if err != nil {
		h.common.HandleError(c, err, "Failed to mark invoice as paid", http.StatusBadRequest, h.common.logger)
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

// MarkInvoiceUncollectible marks an invoice as uncollectible (bad debt)
// @Summary Mark invoice as uncollectible
// @Description Mark an invoice as uncollectible when payment cannot be recovered
// @Tags invoices
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param invoice_id path string true "Invoice ID"
// @Success 200 {object} InvoiceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/invoices/{invoice_id}/mark-uncollectible [post]
func (h *InvoiceHandler) MarkInvoiceUncollectible(c *gin.Context) {
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

	// Mark invoice as uncollectible
	invoice, err := h.invoiceService.MarkInvoiceUncollectible(ctx, workspaceID, invoiceID)
	if err != nil {
		h.common.HandleError(c, err, "Failed to mark invoice as uncollectible", http.StatusBadRequest, h.common.logger)
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

// DuplicateInvoice creates a copy of an existing invoice
// @Summary Duplicate invoice
// @Description Create a copy of an existing invoice as a new draft
// @Tags invoices
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param invoice_id path string true "Invoice ID"
// @Success 201 {object} InvoiceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/invoices/{invoice_id}/duplicate [post]
func (h *InvoiceHandler) DuplicateInvoice(c *gin.Context) {
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

	// Duplicate the invoice
	duplicatedInvoice, err := h.invoiceService.DuplicateInvoice(ctx, workspaceID, invoiceID)
	if err != nil {
		h.common.HandleError(c, err, "Failed to duplicate invoice", http.StatusBadRequest, h.common.logger)
		return
	}

	c.JSON(http.StatusCreated, duplicatedInvoice)
}

// GetInvoiceActivity gets invoice activity history
// @Summary Get invoice activity history
// @Description Retrieves activity history for a specific invoice
// @Tags invoices
// @Accept json
// @Produce json
// @Param invoice_id path string true "Invoice ID"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{} "activities array and pagination info"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/invoices/{invoice_id}/activity [get]
func (h *InvoiceHandler) GetInvoiceActivity(c *gin.Context) {
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

	// Parse query parameters
	limit, offset, err := helpers.ParsePaginationParamsAsInt(c)
	if err != nil {
		h.common.HandleError(c, err, "Invalid pagination parameters", http.StatusBadRequest, h.common.logger)
		return
	}

	// Get invoice activities
	activities, err := h.invoiceService.GetInvoiceActivity(ctx, workspaceID, invoiceID, int32(limit), int32(offset))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.common.HandleError(c, err, "Invoice not found", http.StatusNotFound, h.common.logger)
			return
		}
		h.common.HandleError(c, err, "Failed to get invoice activities", http.StatusInternalServerError, h.common.logger)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activities": activities,
		"limit":      limit,
		"offset":     offset,
	})
}

// BulkGenerateInvoices generates invoices for all due subscriptions
// @Summary Bulk generate invoices
// @Description Generates invoices for all subscriptions with periods ending before the specified date
// @Tags invoices
// @Accept json
// @Produce json
// @Param body body map[string]interface{} true "Bulk generation parameters" Example({"end_date": "2024-01-31T23:59:59Z", "max_invoices": 100})
// @Success 200 {object} responses.BulkInvoiceGenerationResult
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/invoices/bulk-generate [post]
func (h *InvoiceHandler) BulkGenerateInvoices(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		h.common.HandleError(c, err, "Workspace ID required", http.StatusBadRequest, h.common.logger)
		return
	}

	var req struct {
		EndDate     time.Time `json:"end_date" binding:"required"`
		MaxInvoices int32     `json:"max_invoices,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.common.HandleError(c, err, "Invalid request body", http.StatusBadRequest, h.common.logger)
		return
	}

	// Default max invoices to 100 if not specified
	if req.MaxInvoices <= 0 {
		req.MaxInvoices = 100
	}

	// Cap at 500 to prevent excessive load
	if req.MaxInvoices > 500 {
		req.MaxInvoices = 500
	}

	// Bulk generate invoices
	result, err := h.invoiceService.BulkGenerateInvoices(ctx, workspaceID, req.EndDate, req.MaxInvoices)
	if err != nil {
		h.common.HandleError(c, err, "Failed to bulk generate invoices", http.StatusInternalServerError, h.common.logger)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetInvoiceStats gets invoice statistics
// @Summary Get invoice statistics
// @Description Retrieves invoice statistics for the specified date range
// @Tags invoices
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339)" Example("2024-01-01T00:00:00Z")
// @Param end_date query string false "End date (RFC3339)" Example("2024-01-31T23:59:59Z")
// @Success 200 {object} responses.InvoiceStatsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/invoices/stats [get]
func (h *InvoiceHandler) GetInvoiceStats(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		h.common.HandleError(c, err, "Workspace ID required", http.StatusBadRequest, h.common.logger)
		return
	}

	// Parse date parameters
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	var startDate, endDate time.Time

	// Default to last 30 days if not specified
	if startDateStr == "" {
		startDate = time.Now().AddDate(0, 0, -30).Truncate(24 * time.Hour)
	} else {
		startDate, err = time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			h.common.HandleError(c, err, "Invalid start_date format", http.StatusBadRequest, h.common.logger)
			return
		}
	}

	if endDateStr == "" {
		endDate = time.Now()
	} else {
		endDate, err = time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			h.common.HandleError(c, err, "Invalid end_date format", http.StatusBadRequest, h.common.logger)
			return
		}
	}

	// Get invoice stats
	stats, err := h.invoiceService.GetInvoiceStats(ctx, workspaceID, startDate, endDate)
	if err != nil {
		h.common.HandleError(c, err, "Failed to get invoice stats", http.StatusInternalServerError, h.common.logger)
		return
	}

	c.JSON(http.StatusOK, stats)
}