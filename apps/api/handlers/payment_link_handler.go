package handlers

import (
	"net/http"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

const (
	// Error messages
	errMsgWorkspaceIDRequired = "Workspace ID required"
	errMsgInvalidLinkID       = "Invalid link ID"
	errMsgInvalidProductID    = "Invalid product ID"
	errMsgInvalidInvoiceID    = "Invalid invoice ID"
)

// PaymentLinkHandler handles payment link-related HTTP requests
type PaymentLinkHandler struct {
	common             *CommonServices
	paymentLinkService interfaces.PaymentLinkService
}

// NewPaymentLinkHandler creates a handler with interface dependency
func NewPaymentLinkHandler(
	common *CommonServices,
	paymentLinkService interfaces.PaymentLinkService,
) *PaymentLinkHandler {
	return &PaymentLinkHandler{
		common:             common,
		paymentLinkService: paymentLinkService,
	}
}

// Use types from the centralized packages
type CreatePaymentLinkRequest = requests.CreatePaymentLinkRequest
type UpdatePaymentLinkRequest = requests.UpdatePaymentLinkRequest

// CreatePaymentLink creates a new payment link
// @Summary Create a payment link
// @Description Create a new payment link for accepting payments
// @Tags payment-links
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param request body CreatePaymentLinkRequest true "Payment link creation request"
// @Success 201 {object} responses.PaymentLinkResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payment-links [post]
func (h *PaymentLinkHandler) CreatePaymentLink(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		h.common.HandleError(c, err, errMsgWorkspaceIDRequired, http.StatusBadRequest, h.common.logger)
		return
	}

	var req CreatePaymentLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.common.HandleError(c, err, "Invalid request body", http.StatusBadRequest, h.common.logger)
		return
	}

	// Set defaults
	paymentType := "one_time"
	if req.PaymentType != "" {
		paymentType = req.PaymentType
	}

	collectEmail := true
	if req.CollectEmail != nil {
		collectEmail = *req.CollectEmail
	}

	collectName := true
	if req.CollectName != nil {
		collectName = *req.CollectName
	}

	collectShipping := false
	if req.CollectShipping != nil {
		collectShipping = *req.CollectShipping
	}

	// Calculate expiration time if provided
	var expiresAtStr *string
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(*req.ExpiresIn) * time.Hour).Format(time.RFC3339)
		expiresAtStr = &exp
	}

	// Convert AmountCents to non-pointer if provided
	var amountCents int64
	if req.AmountCents != nil {
		amountCents = *req.AmountCents
	}

	// Build metadata with additional fields for backward compatibility
	metadata := req.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["payment_type"] = paymentType
	metadata["collect_email"] = collectEmail
	metadata["collect_shipping"] = collectShipping
	metadata["collect_name"] = collectName

	// Create payment link
	link, err := h.paymentLinkService.CreatePaymentLink(ctx, params.PaymentLinkCreateParams{
		WorkspaceID:         workspaceID,
		ProductID:           req.ProductID,
		PriceID:             req.PriceID,
		AmountCents:         amountCents,
		Currency:            req.Currency,
		ExpiresAt:           expiresAtStr,
		MaxRedemptions:      req.MaxUses,
		RedirectURL:         req.RedirectURL,
		Metadata:            metadata,
		RequireCustomerInfo: collectEmail || collectName,
	})
	if err != nil {
		h.common.HandleError(c, err, "Failed to create payment link", http.StatusInternalServerError, h.common.logger)
		return
	}

	c.JSON(http.StatusCreated, link)
}

// GetPaymentLink retrieves a payment link by ID
// @Summary Get payment link
// @Description Get a payment link by ID
// @Tags payment-links
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param link_id path string true "Payment Link ID"
// @Success 200 {object} responses.PaymentLinkResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payment-links/{link_id} [get]
func (h *PaymentLinkHandler) GetPaymentLink(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		h.common.HandleError(c, err, errMsgWorkspaceIDRequired, http.StatusBadRequest, h.common.logger)
		return
	}

	linkIDStr := c.Param("link_id")
	linkID, err := uuid.Parse(linkIDStr)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidLinkID, http.StatusBadRequest, h.common.logger)
		return
	}

	link, err := h.paymentLinkService.GetPaymentLink(ctx, workspaceID, linkID)
	if err != nil {
		h.common.HandleError(c, err, "Payment link not found", http.StatusNotFound, h.common.logger)
		return
	}

	c.JSON(http.StatusOK, link)
}

// GetPaymentLinkBySlug retrieves a payment link by slug (public endpoint)
// @Summary Get payment link by slug
// @Description Get a payment link by slug for payment processing
// @Tags payment-links
// @Accept json
// @Produce json
// @Param slug path string true "Payment Link Slug"
// @Success 200 {object} responses.PaymentLinkResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payment-links/slug/{slug} [get]
func (h *PaymentLinkHandler) GetPaymentLinkBySlug(c *gin.Context) {
	ctx := c.Request.Context()
	slug := c.Param("slug")

	link, err := h.paymentLinkService.GetPaymentLinkBySlug(ctx, slug)
	if err != nil {
		h.common.HandleError(c, err, "Payment link not found or inactive", http.StatusNotFound, h.common.logger)
		return
	}

	c.JSON(http.StatusOK, link)
}

// UpdatePaymentLink updates a payment link
// @Summary Update payment link
// @Description Update an existing payment link
// @Tags payment-links
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param link_id path string true "Payment Link ID"
// @Param request body UpdatePaymentLinkRequest true "Update request"
// @Success 200 {object} responses.PaymentLinkResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payment-links/{link_id} [put]
func (h *PaymentLinkHandler) UpdatePaymentLink(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		h.common.HandleError(c, err, errMsgWorkspaceIDRequired, http.StatusBadRequest, h.common.logger)
		return
	}

	linkIDStr := c.Param("link_id")
	linkID, err := uuid.Parse(linkIDStr)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidLinkID, http.StatusBadRequest, h.common.logger)
		return
	}

	var req UpdatePaymentLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.common.HandleError(c, err, "Invalid request body", http.StatusBadRequest, h.common.logger)
		return
	}

	// Calculate expiration time if provided
	var expiresAtStr *string
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(*req.ExpiresIn) * time.Hour).Format(time.RFC3339)
		expiresAtStr = &exp
	}

	// Convert status to IsActive boolean
	var isActive *bool
	if req.Status != nil {
		active := *req.Status == "active"
		isActive = &active
	}

	link, err := h.paymentLinkService.UpdatePaymentLink(ctx, workspaceID, linkID, params.PaymentLinkUpdateParams{
		IsActive:       isActive,
		ExpiresAt:      expiresAtStr,
		MaxRedemptions: req.MaxUses,
		RedirectURL:    req.RedirectURL,
		Metadata:       req.Metadata,
	})
	if err != nil {
		h.common.HandleError(c, err, "Failed to update payment link", http.StatusInternalServerError, h.common.logger)
		return
	}

	c.JSON(http.StatusOK, link)
}

// DeactivatePaymentLink deactivates a payment link
// @Summary Deactivate payment link
// @Description Deactivate a payment link so it can no longer be used
// @Tags payment-links
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param link_id path string true "Payment Link ID"
// @Success 200 {object} map[string]interface{} "message: Payment link deactivated"
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payment-links/{link_id}/deactivate [post]
func (h *PaymentLinkHandler) DeactivatePaymentLink(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		h.common.HandleError(c, err, errMsgWorkspaceIDRequired, http.StatusBadRequest, h.common.logger)
		return
	}

	linkIDStr := c.Param("link_id")
	linkID, err := uuid.Parse(linkIDStr)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidLinkID, http.StatusBadRequest, h.common.logger)
		return
	}

	err = h.paymentLinkService.DeactivatePaymentLink(ctx, workspaceID, linkID)
	if err != nil {
		h.common.HandleError(c, err, "Failed to deactivate payment link", http.StatusInternalServerError, h.common.logger)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment link deactivated successfully",
	})
}

// ListPaymentLinks lists payment links for a workspace
// @Summary List payment links
// @Description List all payment links for a workspace with pagination
// @Tags payment-links
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Param product_id query string false "Filter by product ID"
// @Success 200 {object} map[string]interface{} "links array and pagination"
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payment-links [get]
func (h *PaymentLinkHandler) ListPaymentLinks(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		h.common.HandleError(c, err, errMsgWorkspaceIDRequired, http.StatusBadRequest, h.common.logger)
		return
	}

	// Parse pagination parameters
	limit, offset, err := helpers.ParsePaginationParamsAsInt(c)
	if err != nil {
		h.common.HandleError(c, err, "Invalid pagination parameters", http.StatusBadRequest, h.common.logger)
		return
	}

	// Parse query parameters
	productIDStr := c.Query("product_id")

	var links []db.PaymentLink

	if productIDStr != "" {
		// Filter by product
		productID, err := uuid.Parse(productIDStr)
		if err != nil {
			h.common.HandleError(c, err, errMsgInvalidProductID, http.StatusBadRequest, h.common.logger)
			return
		}

		links, err = h.common.db.GetPaymentLinksByProduct(ctx, db.GetPaymentLinksByProductParams{
			WorkspaceID: workspaceID,
			ProductID:   pgtype.UUID{Bytes: productID, Valid: true},
		})
		if err != nil {
			h.common.HandleError(c, err, "Failed to list payment links", http.StatusInternalServerError, h.common.logger)
			return
		}
	} else {
		// Get all links with pagination
		links, err = h.common.db.GetPaymentLinksByWorkspace(ctx, db.GetPaymentLinksByWorkspaceParams{
			WorkspaceID: workspaceID,
			Limit:       int32(limit),  // #nosec G115 -- ParsePaginationParamsAsInt validates limit <= 100
			Offset:      int32(offset), // #nosec G115 -- ParsePaginationParamsAsInt validates offset >= 0
		})
		if err != nil {
			h.common.HandleError(c, err, "Failed to list payment links", http.StatusInternalServerError, h.common.logger)
			return
		}
	}

	// Convert to response format
	baseURL := h.paymentLinkService.GetBaseURL()
	var linkResponses []map[string]interface{}
	for _, link := range links {
		linkResponses = append(linkResponses, map[string]interface{}{
			"id":         link.ID,
			"slug":       link.Slug,
			"url":        baseURL + "/pay/" + link.Slug,
			"status":     link.Status,
			"product_id": link.ProductID.Bytes,
			"used_count": link.UsedCount,
			"max_uses":   link.MaxUses.Int32,
			"expires_at": link.ExpiresAt.Time,
			"created_at": link.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"links": linkResponses,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetPaymentLinkStats gets statistics for payment links
// @Summary Get payment link statistics
// @Description Get usage statistics for payment links in a workspace
// @Tags payment-links
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Success 200 {object} map[string]interface{} "Payment link statistics"
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payment-links/stats [get]
func (h *PaymentLinkHandler) GetPaymentLinkStats(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		h.common.HandleError(c, err, errMsgWorkspaceIDRequired, http.StatusBadRequest, h.common.logger)
		return
	}

	stats, err := h.common.db.GetPaymentLinkStats(ctx, workspaceID)
	if err != nil {
		h.common.HandleError(c, err, "Failed to get payment link statistics", http.StatusInternalServerError, h.common.logger)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_links":    stats.TotalLinks,
		"active_links":   stats.ActiveLinks,
		"inactive_links": stats.InactiveLinks,
		"expired_links":  stats.ExpiredLinks,
		"total_uses":     stats.TotalUses,
	})
}

// CreateInvoicePaymentLink creates a payment link for an invoice
// @Summary Create payment link for invoice
// @Description Create a payment link specifically for an invoice
// @Tags payment-links
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param invoice_id path string true "Invoice ID"
// @Success 201 {object} responses.PaymentLinkResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/invoices/{invoice_id}/payment-link [post]
func (h *PaymentLinkHandler) CreateInvoicePaymentLink(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := GetWorkspaceID(c)
	if err != nil {
		h.common.HandleError(c, err, errMsgWorkspaceIDRequired, http.StatusBadRequest, h.common.logger)
		return
	}

	invoiceIDStr := c.Param("invoice_id")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		h.common.HandleError(c, err, errMsgInvalidInvoiceID, http.StatusBadRequest, h.common.logger)
		return
	}

	// Get invoice
	invoice, err := h.common.db.GetInvoiceByID(ctx, db.GetInvoiceByIDParams{
		ID:          invoiceID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		h.common.HandleError(c, err, "Invoice not found", http.StatusNotFound, h.common.logger)
		return
	}

	// Create payment link for invoice
	link, err := h.paymentLinkService.CreatePaymentLinkForInvoice(ctx, invoice)
	if err != nil {
		h.common.HandleError(c, err, "Failed to create payment link", http.StatusInternalServerError, h.common.logger)
		return
	}

	// Update invoice with payment link ID
	_, err = h.common.db.LinkInvoiceToPaymentLink(ctx, db.LinkInvoiceToPaymentLinkParams{
		ID:            invoiceID,
		WorkspaceID:   workspaceID,
		PaymentLinkID: pgtype.UUID{Bytes: link.ID, Valid: true},
	})
	if err != nil {
		h.common.logger.Error("Failed to link invoice to payment link",
			zap.Error(err),
			zap.String("invoice_id", invoiceID.String()),
			zap.String("payment_link_id", link.ID.String()))
	}

	c.JSON(http.StatusCreated, link)
}
