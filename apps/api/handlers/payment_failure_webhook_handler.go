package handlers

import (
	"net/http"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PaymentFailureWebhookHandler handles payment failure webhooks from payment providers
type PaymentFailureWebhookHandler struct {
	common          *CommonServices
	failureDetector interfaces.PaymentFailureDetector
}

// NewPaymentFailureWebhookHandler creates a handler with interface dependencies
func NewPaymentFailureWebhookHandler(
	common *CommonServices,
	failureDetector interfaces.PaymentFailureDetector,
) *PaymentFailureWebhookHandler {
	return &PaymentFailureWebhookHandler{
		common:          common,
		failureDetector: failureDetector,
	}
}

// Use types from the centralized packages
type PaymentFailureWebhookRequest = requests.PaymentFailureWebhookRequest

// HandlePaymentFailure processes a payment failure webhook
// @Summary Process payment failure webhook
// @Description Process a payment failure notification from a payment provider
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param webhook body PaymentFailureWebhookRequest true "Payment failure webhook data"
// @Success 200 {object} map[string]interface{} "Webhook processed successfully"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/webhooks/payment-failure [post]
func (h *PaymentFailureWebhookHandler) HandlePaymentFailure(c *gin.Context) {
	// Get workspace ID from context
	workspaceID, exists := c.Get("workspace_id")
	if !exists {
		h.common.logger.Error("workspace_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	workspaceUUID, err := uuid.Parse(workspaceID.(string))
	if err != nil {
		h.common.logger.Error("invalid workspace_id", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid workspace"})
		return
	}

	var req PaymentFailureWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.common.logger.Error("invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Parse subscription ID
	subscriptionID, err := uuid.Parse(req.SubscriptionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
		return
	}

	// Verify the subscription belongs to this workspace
	subscription, err := h.common.db.GetSubscription(c.Request.Context(), subscriptionID)
	if err != nil {
		h.common.logger.Error("failed to get subscription", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}

	if subscription.WorkspaceID != workspaceUUID {
		h.common.logger.Error("subscription workspace mismatch",
			zap.String("subscription_workspace", subscription.WorkspaceID.String()),
			zap.String("request_workspace", workspaceUUID.String()),
		)
		c.JSON(http.StatusForbidden, gin.H{"error": "Subscription does not belong to this workspace"})
		return
	}

	// Build failure data
	failureData := map[string]interface{}{
		"provider":            req.Provider,
		"amount_cents":        req.AmountCents,
		"currency":            req.Currency,
		"failure_reason":      req.FailureReason,
		"failed_at":           req.FailedAt.Format(time.RFC3339),
		"metadata":            req.Metadata,
		"webhook_received_at": time.Now().Format(time.RFC3339),
	}

	// Process the payment failure
	err = h.failureDetector.ProcessFailedPaymentWebhook(
		c.Request.Context(),
		workspaceUUID,
		subscriptionID,
		failureData,
	)
	if err != nil {
		h.common.logger.Error("failed to process payment failure webhook",
			zap.Error(err),
			zap.String("subscription_id", subscriptionID.String()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process payment failure"})
		return
	}

	h.common.logger.Info("Payment failure webhook processed successfully",
		zap.String("workspace_id", workspaceUUID.String()),
		zap.String("subscription_id", subscriptionID.String()),
		zap.String("provider", req.Provider),
		zap.Int64("amount_cents", req.AmountCents),
	)

	c.JSON(http.StatusOK, gin.H{
		"status":          "processed",
		"message":         "Payment failure recorded and dunning campaign created if applicable",
		"subscription_id": subscriptionID.String(),
	})
}

// HandleBatchPaymentFailures processes multiple payment failures at once
// @Summary Process batch payment failure webhooks
// @Description Process multiple payment failure notifications from a payment provider
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param webhooks body []PaymentFailureWebhookRequest true "Payment failure webhook data array"
// @Success 200 {object} map[string]interface{} "Batch processing results"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/webhooks/payment-failures/batch [post]
func (h *PaymentFailureWebhookHandler) HandleBatchPaymentFailures(c *gin.Context) {
	// Get workspace ID from context
	workspaceID, exists := c.Get("workspace_id")
	if !exists {
		h.common.logger.Error("workspace_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	workspaceUUID, err := uuid.Parse(workspaceID.(string))
	if err != nil {
		h.common.logger.Error("invalid workspace_id", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid workspace"})
		return
	}

	var requests []PaymentFailureWebhookRequest
	if err := c.ShouldBindJSON(&requests); err != nil {
		h.common.logger.Error("invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	results := []map[string]interface{}{}
	successCount := 0
	failureCount := 0

	for _, req := range requests {
		result := map[string]interface{}{
			"subscription_id": req.SubscriptionID,
			"status":          "pending",
		}

		// Parse subscription ID
		subscriptionID, err := uuid.Parse(req.SubscriptionID)
		if err != nil {
			result["status"] = "failed"
			result["error"] = "Invalid subscription ID"
			failureCount++
			results = append(results, result)
			continue
		}

		// Build failure data
		failureData := map[string]interface{}{
			"provider":            req.Provider,
			"amount_cents":        req.AmountCents,
			"currency":            req.Currency,
			"failure_reason":      req.FailureReason,
			"failed_at":           req.FailedAt.Format(time.RFC3339),
			"metadata":            req.Metadata,
			"webhook_received_at": time.Now().Format(time.RFC3339),
		}

		// Process the payment failure
		err = h.failureDetector.ProcessFailedPaymentWebhook(
			c.Request.Context(),
			workspaceUUID,
			subscriptionID,
			failureData,
		)
		if err != nil {
			result["status"] = "failed"
			result["error"] = err.Error()
			failureCount++
		} else {
			result["status"] = "processed"
			successCount++
		}

		results = append(results, result)
	}

	h.common.logger.Info("Batch payment failure webhook processing completed",
		zap.String("workspace_id", workspaceUUID.String()),
		zap.Int("total", len(requests)),
		zap.Int("success", successCount),
		zap.Int("failed", failureCount),
	)

	c.JSON(http.StatusOK, gin.H{
		"status":  "completed",
		"total":   len(requests),
		"success": successCount,
		"failed":  failureCount,
		"results": results,
	})
}
