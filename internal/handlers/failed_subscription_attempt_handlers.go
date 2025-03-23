package handlers

import (
	"cyphera-api/internal/db"
	"cyphera-api/internal/logger"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// FailedSubscriptionAttemptHandler handles API endpoints for failed subscription attempts
type FailedSubscriptionAttemptHandler struct {
	common *CommonServices
}

// NewFailedSubscriptionAttemptHandler creates a new instance of FailedSubscriptionAttemptHandler
func NewFailedSubscriptionAttemptHandler(common *CommonServices) *FailedSubscriptionAttemptHandler {
	return &FailedSubscriptionAttemptHandler{
		common: common,
	}
}

// ListFailedSubscriptionAttempts godoc
// @Summary List all failed subscription attempts
// @Description Get a list of all failed subscription attempts
// @Tags Failed Subscription Attempts
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Number of items per page"
// @Success 200 {object} []db.FailedSubscriptionAttempt
// @Failure 500 {object} ErrorResponse
// @Router /failed-subscription-attempts [get]
func (h *FailedSubscriptionAttemptHandler) ListFailedSubscriptionAttempts(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse pagination parameters
	page, limit := 1, 50
	if pageParam := c.Query("page"); pageParam != "" {
		if parsedPage, err := strconv.Atoi(pageParam); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	// Get all failed attempts first, then manually apply pagination
	// Note: This is not efficient for large datasets and should be improved with a proper paginated query
	allAttempts, err := h.common.db.ListFailedSubscriptionAttempts(ctx)
	if err != nil {
		logger.Error("Failed to list failed subscription attempts", zap.Error(err))
		sendError(c, http.StatusInternalServerError, "Failed to retrieve failed subscription attempts", err)
		return
	}

	// Get total count for pagination
	count := len(allAttempts)

	// Manual pagination
	offset := (page - 1) * limit
	end := offset + limit
	if end > count {
		end = count
	}

	var pagedAttempts []db.FailedSubscriptionAttempt
	if offset < count {
		pagedAttempts = allAttempts[offset:end]
	} else {
		pagedAttempts = []db.FailedSubscriptionAttempt{}
	}

	sendPaginatedSuccess(c, http.StatusOK, pagedAttempts, page, limit, count)
}

// GetFailedSubscriptionAttempt godoc
// @Summary Get a specific failed subscription attempt
// @Description Get details of a specific failed subscription attempt by ID
// @Tags Failed Subscription Attempts
// @Accept json
// @Produce json
// @Param attempt_id path string true "Failed Subscription Attempt ID"
// @Success 200 {object} db.FailedSubscriptionAttempt
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /failed-subscription-attempts/{attempt_id} [get]
func (h *FailedSubscriptionAttemptHandler) GetFailedSubscriptionAttempt(c *gin.Context) {
	ctx := c.Request.Context()
	attemptID := c.Param("attempt_id")

	parsedID, err := uuid.Parse(attemptID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid failed subscription attempt ID", err)
		return
	}

	attempt, err := h.common.db.GetFailedSubscriptionAttempt(ctx, parsedID)
	if err != nil {
		handleDBError(c, err, "Failed subscription attempt not found")
		return
	}

	sendSuccess(c, http.StatusOK, attempt)
}

// ListFailedSubscriptionAttemptsByCustomer godoc
// @Summary List failed subscription attempts for a customer
// @Description Get all failed subscription attempts for a specific customer
// @Tags Failed Subscription Attempts
// @Accept json
// @Produce json
// @Param customer_id path string true "Customer ID"
// @Success 200 {array} db.FailedSubscriptionAttempt
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /failed-subscription-attempts/customer/{customer_id} [get]
func (h *FailedSubscriptionAttemptHandler) ListFailedSubscriptionAttemptsByCustomer(c *gin.Context) {
	ctx := c.Request.Context()
	customerID := c.Param("customer_id")

	parsedID, err := uuid.Parse(customerID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid customer ID", err)
		return
	}

	// Convert to pgtype.UUID
	pgUUID := pgtype.UUID{
		Bytes: parsedID,
		Valid: true,
	}

	attempts, err := h.common.db.ListFailedSubscriptionAttemptsByCustomer(ctx, pgUUID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve failed subscription attempts", err)
		return
	}

	sendList(c, attempts)
}

// ListFailedSubscriptionAttemptsByProduct godoc
// @Summary List failed subscription attempts for a product
// @Description Get all failed subscription attempts for a specific product
// @Tags Failed Subscription Attempts
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Success 200 {array} db.FailedSubscriptionAttempt
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /failed-subscription-attempts/product/{product_id} [get]
func (h *FailedSubscriptionAttemptHandler) ListFailedSubscriptionAttemptsByProduct(c *gin.Context) {
	ctx := c.Request.Context()
	productID := c.Param("product_id")

	parsedID, err := uuid.Parse(productID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid product ID", err)
		return
	}

	attempts, err := h.common.db.ListFailedSubscriptionAttemptsByProduct(ctx, parsedID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve failed subscription attempts", err)
		return
	}

	sendList(c, attempts)
}

// ListFailedSubscriptionAttemptsByErrorType godoc
// @Summary List failed subscription attempts by error type
// @Description Get all failed subscription attempts with a specific error type
// @Tags Failed Subscription Attempts
// @Accept json
// @Produce json
// @Param error_type path string true "Error Type"
// @Success 200 {array} db.FailedSubscriptionAttempt
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /failed-subscription-attempts/error-type/{error_type} [get]
func (h *FailedSubscriptionAttemptHandler) ListFailedSubscriptionAttemptsByErrorType(c *gin.Context) {
	ctx := c.Request.Context()
	errorType := c.Param("error_type")

	// Validate error type
	if !isValidErrorType(errorType) {
		sendError(c, http.StatusBadRequest, "Invalid error type", nil)
		return
	}

	attempts, err := h.common.db.ListFailedSubscriptionAttemptsByErrorType(ctx, db.SubscriptionEventType(errorType))
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve failed subscription attempts", err)
		return
	}

	sendList(c, attempts)
}

// isValidErrorType checks if the provided error type is valid
func isValidErrorType(errorType string) bool {
	validTypes := []db.SubscriptionEventType{
		db.SubscriptionEventTypeFailed,
		db.SubscriptionEventTypeFailedValidation,
		db.SubscriptionEventTypeFailedCustomerCreation,
		db.SubscriptionEventTypeFailedWalletCreation,
		db.SubscriptionEventTypeFailedDelegationStorage,
		db.SubscriptionEventTypeFailedSubscriptionDb,
		db.SubscriptionEventTypeFailedRedemption,
		db.SubscriptionEventTypeFailedTransaction,
		db.SubscriptionEventTypeFailedDuplicate,
	}

	for _, validType := range validTypes {
		if db.SubscriptionEventType(errorType) == validType {
			return true
		}
	}

	return false
}
