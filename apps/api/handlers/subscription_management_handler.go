package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type SubscriptionManagementHandler struct {
	service interfaces.SubscriptionManagementService
	logger  *zap.Logger
}

// NewSubscriptionManagementHandler creates a handler with an interface
func NewSubscriptionManagementHandler(
	service interfaces.SubscriptionManagementService,
	logger *zap.Logger,
) *SubscriptionManagementHandler {
	if logger == nil {
		logger = zap.L()
	}
	return &SubscriptionManagementHandler{
		service: service,
		logger:  logger,
	}
}

// Use types from the centralized packages
type UpgradeSubscriptionRequest = requests.UpgradeSubscriptionRequest
type DowngradeSubscriptionRequest = requests.DowngradeSubscriptionRequest
type CancelSubscriptionRequest = requests.CancelSubscriptionRequest
type PauseSubscriptionRequest = requests.PauseSubscriptionRequest
type PreviewChangeRequest = requests.PreviewChangeRequest

// Helper functions to convert between request and service types
func toServicesLineItemUpdate(reqItem requests.LineItemUpdate) requests.LineItemUpdate {
	return requests.LineItemUpdate{
		Action:         reqItem.Action,
		LineItemID:     reqItem.LineItemID,
		ProductID:      reqItem.ProductID,
		PriceID:        reqItem.PriceID,
		ProductTokenID: reqItem.ProductTokenID,
		Quantity:       reqItem.Quantity,
		UnitAmount:     reqItem.UnitAmount,
	}
}

func toServicesLineItemUpdateList(reqItems []requests.LineItemUpdate) []requests.LineItemUpdate {
	result := make([]requests.LineItemUpdate, len(reqItems))
	for i, reqItem := range reqItems {
		result[i] = toServicesLineItemUpdate(reqItem)
	}
	return result
}

// @Summary Upgrade a subscription
// @Description Upgrade a subscription immediately with proration
// @Tags subscriptions
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param subscription_id path string true "Subscription ID"
// @Param request body UpgradeSubscriptionRequest true "Upgrade details"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/subscriptions/{subscription_id}/upgrade [post]
func (h *SubscriptionManagementHandler) UpgradeSubscription(c *gin.Context) {
	ctx := c.Request.Context()

	subscriptionIDStr := c.Param("subscription_id")
	subscriptionID, err := uuid.Parse(subscriptionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid subscription ID"})
		return
	}

	var req UpgradeSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	// TODO: Implement proper ownership verification
	// userID := c.GetString("user_id")
	// if !h.verifySubscriptionOwnership(ctx, subscriptionID, userID) {
	//     c.JSON(http.StatusForbidden, ErrorResponse{Error: "Access denied"})
	//     return
	// }

	err = h.service.UpgradeSubscription(ctx, subscriptionID, toServicesLineItemUpdateList(req.LineItems), req.Reason)
	if err != nil {
		h.logger.Error("Failed to upgrade subscription",
			zap.String("subscription_id", subscriptionID.String()),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to upgrade subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Subscription upgraded successfully",
		"subscription_id": subscriptionID,
		"status":          "upgraded",
	})
}

// @Summary Downgrade a subscription
// @Description Schedule a subscription downgrade for the end of the billing period
// @Tags subscriptions
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param subscription_id path string true "Subscription ID"
// @Param request body DowngradeSubscriptionRequest true "Downgrade details"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/subscriptions/{subscription_id}/downgrade [post]
func (h *SubscriptionManagementHandler) DowngradeSubscription(c *gin.Context) {
	ctx := c.Request.Context()

	subscriptionIDStr := c.Param("subscription_id")
	subscriptionID, err := uuid.Parse(subscriptionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid subscription ID"})
		return
	}

	var req DowngradeSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	err = h.service.DowngradeSubscription(ctx, subscriptionID, toServicesLineItemUpdateList(req.LineItems), req.Reason)
	if err != nil {
		h.logger.Error("Failed to downgrade subscription",
			zap.String("subscription_id", subscriptionID.String()),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to schedule downgrade"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Subscription downgrade scheduled",
		"subscription_id": subscriptionID,
		"status":          "downgrade_scheduled",
	})
}

// @Summary Cancel a subscription
// @Description Schedule subscription cancellation for the end of the billing period
// @Tags subscriptions
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param subscription_id path string true "Subscription ID"
// @Param request body CancelSubscriptionRequest true "Cancellation details"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/subscriptions/{subscription_id}/cancel [post]
func (h *SubscriptionManagementHandler) CancelSubscription(c *gin.Context) {
	ctx := c.Request.Context()

	subscriptionIDStr := c.Param("subscription_id")
	subscriptionID, err := uuid.Parse(subscriptionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid subscription ID"})
		return
	}

	var req CancelSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	err = h.service.CancelSubscription(ctx, subscriptionID, req.Reason, req.Feedback)
	if err != nil {
		h.logger.Error("Failed to cancel subscription",
			zap.String("subscription_id", subscriptionID.String()),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to cancel subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Subscription cancellation scheduled",
		"subscription_id": subscriptionID,
		"status":          "cancellation_scheduled",
	})
}

// @Summary Pause a subscription
// @Description Pause a subscription immediately or until a specific date
// @Tags subscriptions
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param subscription_id path string true "Subscription ID"
// @Param request body PauseSubscriptionRequest true "Pause details"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/subscriptions/{subscription_id}/pause [post]
func (h *SubscriptionManagementHandler) PauseSubscription(c *gin.Context) {
	ctx := c.Request.Context()

	subscriptionIDStr := c.Param("subscription_id")
	subscriptionID, err := uuid.Parse(subscriptionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid subscription ID"})
		return
	}

	var req PauseSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	var pauseUntil *time.Time
	if req.PauseUntil != "" {
		t, err := time.Parse(time.RFC3339, req.PauseUntil)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid pause_until timestamp"})
			return
		}
		pauseUntil = &t
	}

	err = h.service.PauseSubscription(ctx, subscriptionID, pauseUntil, req.Reason)
	if err != nil {
		h.logger.Error("Failed to pause subscription",
			zap.String("subscription_id", subscriptionID.String()),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to pause subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Subscription paused",
		"subscription_id": subscriptionID,
		"status":          "paused",
		"pause_until":     pauseUntil,
	})
}

// @Summary Resume a paused subscription
// @Description Resume a paused subscription immediately
// @Tags subscriptions
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param subscription_id path string true "Subscription ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/subscriptions/{subscription_id}/resume [post]
func (h *SubscriptionManagementHandler) ResumeSubscription(c *gin.Context) {
	ctx := c.Request.Context()

	subscriptionIDStr := c.Param("subscription_id")
	subscriptionID, err := uuid.Parse(subscriptionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid subscription ID"})
		return
	}

	err = h.service.ResumeSubscription(ctx, subscriptionID)
	if err != nil {
		h.logger.Error("Failed to resume subscription",
			zap.String("subscription_id", subscriptionID.String()),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to resume subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Subscription resumed",
		"subscription_id": subscriptionID,
		"status":          "active",
	})
}

// @Summary Reactivate a cancelled subscription
// @Description Remove a scheduled cancellation and keep the subscription active
// @Tags subscriptions
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param subscription_id path string true "Subscription ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/subscriptions/{subscription_id}/reactivate [post]
func (h *SubscriptionManagementHandler) ReactivateSubscription(c *gin.Context) {
	ctx := c.Request.Context()

	subscriptionIDStr := c.Param("subscription_id")
	subscriptionID, err := uuid.Parse(subscriptionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid subscription ID"})
		return
	}

	err = h.service.ReactivateCancelledSubscription(ctx, subscriptionID)
	if err != nil {
		h.logger.Error("Failed to reactivate subscription",
			zap.String("subscription_id", subscriptionID.String()),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to reactivate subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Subscription reactivated",
		"subscription_id": subscriptionID,
		"status":          "active",
	})
}

// @Summary Preview a subscription change
// @Description Preview what will happen with a subscription change without committing
// @Tags subscriptions
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param subscription_id path string true "Subscription ID"
// @Param request body PreviewChangeRequest true "Change preview details"
// @Success 200 {object} services.ChangePreview
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/subscriptions/{subscription_id}/preview-change [post]
func (h *SubscriptionManagementHandler) PreviewChange(c *gin.Context) {
	ctx := c.Request.Context()

	subscriptionIDStr := c.Param("subscription_id")
	subscriptionID, err := uuid.Parse(subscriptionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid subscription ID"})
		return
	}

	var req PreviewChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	preview, err := h.service.PreviewChange(ctx, subscriptionID, req.ChangeType, toServicesLineItemUpdateList(req.LineItems))
	if err != nil {
		h.logger.Error("Failed to preview change",
			zap.String("subscription_id", subscriptionID.String()),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to preview change"})
		return
	}

	c.JSON(http.StatusOK, preview)
}

// @Summary Get subscription history
// @Description Get the state change history for a subscription
// @Tags subscriptions
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param subscription_id path string true "Subscription ID"
// @Param limit query int false "Number of records to return (default: 50)"
// @Success 200 {object} []db.SubscriptionStateHistory
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/subscriptions/{subscription_id}/history [get]
func (h *SubscriptionManagementHandler) GetSubscriptionHistory(c *gin.Context) {
	ctx := c.Request.Context()

	subscriptionIDStr := c.Param("subscription_id")
	subscriptionID, err := uuid.Parse(subscriptionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid subscription ID"})
		return
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	history, err := h.service.GetSubscriptionHistory(ctx, subscriptionID, int32(limit)) // #nosec G115 -- limit is already validated to be <= 100
	if err != nil {
		h.logger.Error("Failed to get subscription history",
			zap.String("subscription_id", subscriptionID.String()),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get subscription history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"count":   len(history),
	})
}

// Helper method to verify subscription ownership
func (h *SubscriptionManagementHandler) verifySubscriptionOwnership(ctx context.Context, subscriptionID uuid.UUID, userID string) bool {
	// In a real implementation, this would check if the user owns the subscription
	// through their workspace or customer record
	return true
}
