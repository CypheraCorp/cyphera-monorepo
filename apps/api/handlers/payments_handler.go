package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PaymentsHandler manages payment-related HTTP endpoints
type PaymentsHandler struct {
	common         *CommonServices
	paymentService interfaces.PaymentService
	logger         *zap.Logger
}

// NewPaymentsHandler creates a new payments handler with the required dependencies
func NewPaymentsHandler(
	common *CommonServices,
	paymentService interfaces.PaymentService,
	logger *zap.Logger,
) *PaymentsHandler {
	if logger == nil {
		logger = zap.L()
	}
	return &PaymentsHandler{
		common:         common,
		paymentService: paymentService,
		logger:         logger,
	}
}

// ListPayments godoc
// @Summary List payments
// @Description Get a paginated list of payments for the workspace
// @Tags payments
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Param status query string false "Filter by payment status"
// @Param customer_id query string false "Filter by customer ID"
// @Param payment_method query string false "Filter by payment method"
// @Param start_date query string false "Filter by start date (RFC3339)"
// @Param end_date query string false "Filter by end date (RFC3339)"
// @Success 200 {object} responses.PaymentListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/payments [get]
func (h *PaymentsHandler) ListPayments(c *gin.Context) {
	workspaceIDStr := c.GetHeader("X-Workspace-ID")
	if workspaceIDStr == "" {
		sendError(c, http.StatusBadRequest, "X-Workspace-ID header is required", nil)
		return
	}
	
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID", nil)
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	
	offset := (page - 1) * limit

	// Build list parameters
	listParams := params.ListPaymentsParams{
		WorkspaceID: workspaceID,
		Limit:       int32(limit),
		Offset:      int32(offset),
	}

	// Apply filters if provided
	if status := c.Query("status"); status != "" {
		listParams.Status = &status
	}
	
	if customerID := c.Query("customer_id"); customerID != "" {
		parsedCustomerID, err := uuid.Parse(customerID)
		if err != nil {
			sendError(c, http.StatusBadRequest, "Invalid customer ID", nil)
			return
		}
		listParams.CustomerID = &parsedCustomerID
	}
	
	if startDate := c.Query("start_date"); startDate != "" {
		listParams.StartDate = &startDate
	}
	
	if endDate := c.Query("end_date"); endDate != "" {
		listParams.EndDate = &endDate
	}

	// Get payments from service
	payments, err := h.paymentService.ListPayments(c.Request.Context(), listParams)
	if err != nil {
		h.logger.Error("Failed to list payments", 
			zap.String("workspace_id", workspaceID.String()),
			zap.Error(err))
		sendError(c, http.StatusInternalServerError, "Failed to retrieve payments", err)
		return
	}

	// Get total count for pagination
	totalCount, err := h.common.GetDB().CountPaymentsByWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		h.logger.Error("Failed to count payments", 
			zap.String("workspace_id", workspaceID.String()),
			zap.Error(err))
		totalCount = int64(len(payments)) // Fallback to current page count
	}

	// Convert to response format
	paymentResponses := make([]responses.PaymentResponse, 0, len(payments))
	for _, payment := range payments {
		paymentResponse, err := h.convertToPaymentResponse(c.Request.Context(), payment)
		if err != nil {
			h.logger.Warn("Failed to convert payment to response", 
				zap.String("payment_id", payment.ID.String()),
				zap.Error(err))
			continue
		}
		paymentResponses = append(paymentResponses, *paymentResponse)
	}

	// Build pagination metadata
	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))
	pagination := responses.PaginationMeta{
		Page:       page,
		PerPage:    limit,
		Total:      int(totalCount),
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	response := responses.PaymentListResponse{
		Data:       paymentResponses,
		Pagination: pagination,
	}

	c.JSON(http.StatusOK, response)
}

// GetPayment godoc
// @Summary Get a payment by ID
// @Description Get detailed information about a specific payment
// @Tags payments
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param id path string true "Payment ID"
// @Success 200 {object} responses.PaymentResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/payments/{id} [get]
func (h *PaymentsHandler) GetPayment(c *gin.Context) {
	workspaceIDStr := c.GetHeader("X-Workspace-ID")
	if workspaceIDStr == "" {
		sendError(c, http.StatusBadRequest, "X-Workspace-ID header is required", nil)
		return
	}
	
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID", nil)
		return
	}

	paymentIDStr := c.Param("id")
	paymentID, err := uuid.Parse(paymentIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid payment ID", nil)
		return
	}

	// Get payment from service
	payment, err := h.paymentService.GetPayment(c.Request.Context(), params.GetPaymentParams{
		PaymentID:   paymentID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if err.Error() == "payment not found" {
			sendError(c, http.StatusNotFound, "Payment not found", nil)
			return
		}
		h.logger.Error("Failed to get payment", 
			zap.String("payment_id", paymentID.String()),
			zap.String("workspace_id", workspaceID.String()),
			zap.Error(err))
		sendError(c, http.StatusInternalServerError, "Failed to retrieve payment", err)
		return
	}

	// Convert to response format
	paymentResponse, err := h.convertToPaymentResponse(c.Request.Context(), *payment)
	if err != nil {
		h.logger.Error("Failed to convert payment to response", 
			zap.String("payment_id", payment.ID.String()),
			zap.Error(err))
		sendError(c, http.StatusInternalServerError, "Failed to format payment response", err)
		return
	}

	c.JSON(http.StatusOK, paymentResponse)
}

// convertToPaymentResponse converts a database payment to an API response
func (h *PaymentsHandler) convertToPaymentResponse(ctx context.Context, payment db.Payment) (*responses.PaymentResponse, error) {
	response := &responses.PaymentResponse{
		ID:                  payment.ID.String(),
		WorkspaceID:         payment.WorkspaceID.String(),
		CustomerID:          payment.CustomerID.String(),
		AmountInCents:       payment.AmountInCents,
		FormattedAmount:     helpers.FormatMoney(payment.AmountInCents, payment.Currency),
		Currency:            payment.Currency,
		Status:              payment.Status,
		PaymentMethod:       payment.PaymentMethod,
		ProductAmountCents:  payment.ProductAmountCents,
		TaxAmountCents:      payment.TaxAmountCents.Int64,
		GasAmountCents:      payment.GasAmountCents.Int64,
		DiscountAmountCents: payment.DiscountAmountCents.Int64,
		HasGasFee:           payment.HasGasFee.Bool,
		GasSponsored:        payment.GasSponsored.Bool,
		InitiatedAt:         payment.InitiatedAt.Time,
		CreatedAt:           payment.CreatedAt.Time,
		UpdatedAt:           payment.UpdatedAt.Time,
	}

	// Add optional fields
	if payment.TransactionHash.Valid {
		response.TransactionHash = &payment.TransactionHash.String
	}
	
	if payment.InvoiceID.Valid {
		invoiceID := uuid.UUID(payment.InvoiceID.Bytes).String()
		response.InvoiceID = &invoiceID
	}
	
	if payment.SubscriptionID.Valid {
		subID := uuid.UUID(payment.SubscriptionID.Bytes).String()
		response.SubscriptionID = &subID
	}
	
	if payment.ExternalPaymentID.Valid {
		response.ExternalPaymentID = &payment.ExternalPaymentID.String
	}
	
	if payment.PaymentProvider.Valid {
		response.PaymentProvider = &payment.PaymentProvider.String
	}
	
	if payment.CryptoAmount.Valid {
		// For now, convert to a simple string representation
		cryptoAmount := fmt.Sprintf("%.8f", 0.0) // TODO: Implement proper numeric conversion
		response.CryptoAmount = &cryptoAmount
	}
	
	if payment.ExchangeRate.Valid {
		// For now, convert to a simple string representation
		exchangeRate := fmt.Sprintf("%.6f", 1.0) // TODO: Implement proper numeric conversion
		response.ExchangeRate = &exchangeRate
	}
	
	if payment.GasFeeUsdCents.Valid {
		gasFee := payment.GasFeeUsdCents.Int64
		response.GasFeeUSDCents = &gasFee
	}
	
	if payment.CompletedAt.Valid {
		response.CompletedAt = &payment.CompletedAt.Time
	}
	
	if payment.FailedAt.Valid {
		response.FailedAt = &payment.FailedAt.Time
	}
	
	if payment.ErrorMessage.Valid {
		response.ErrorMessage = &payment.ErrorMessage.String
	}

	// Fetch related data
	// Get customer info
	customer, err := h.common.GetDB().GetCustomer(ctx, payment.CustomerID)
	if err == nil {
		response.Customer = &responses.CustomerBasic{
			ID:    customer.ID.String(),
			Name:  customer.Name.String,
			Email: customer.Email.String,
		}
	}

	// Get network info if available
	if payment.NetworkID.Valid {
		network, err := h.common.GetDB().GetNetwork(ctx, payment.NetworkID.Bytes)
		if err == nil {
			response.Network = &responses.NetworkBasic{
				ID:          network.ID.String(),
				Name:        network.Name,
				ChainID:     int64(network.ChainID),
				DisplayName: network.DisplayName.String,
			}
		}
	}

	// Get token info if available
	if payment.TokenID.Valid {
		token, err := h.common.GetDB().GetToken(ctx, payment.TokenID.Bytes)
		if err == nil {
			response.Token = &responses.TokenBasic{
				ID:              token.ID.String(),
				Symbol:          token.Symbol,
				Name:            token.Name,
				ContractAddress: token.ContractAddress,
				Decimals:        int(token.Decimals),
			}
		}
	}

	// Get product name if subscription is linked
	if payment.SubscriptionID.Valid {
		subscription, err := h.common.GetDB().GetSubscription(ctx, payment.SubscriptionID.Bytes)
		if err == nil {
			product, err := h.common.GetDB().GetProduct(ctx, db.GetProductParams{
				ID:          subscription.ProductID,
				WorkspaceID: payment.WorkspaceID,
			})
			if err == nil {
				response.ProductName = &product.Name
				productID := product.ID.String()
				response.ProductID = &productID
			}
		}
	}

	// Parse metadata if available
	if len(payment.Metadata) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(payment.Metadata, &metadata); err == nil {
			response.Metadata = payment.Metadata
		}
	}

	return response, nil
}