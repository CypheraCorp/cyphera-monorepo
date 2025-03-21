package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"cyphera-api/internal/client"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DelegationHandler handles delegation-related operations
type DelegationHandler struct {
	common *CommonServices
}

// NewDelegationHandler creates a new DelegationHandler instance
func NewDelegationHandler(common *CommonServices) *DelegationHandler {
	return &DelegationHandler{common: common}
}

// RedeemDelegationDirectly sends a redemption request to the gRPC service
// and returns the transaction hash or an error
//
// Parameters:
//   - ctx: Context for the request
//   - delegationData: The delegation data as a byte array
//
// Returns:
//   - The transaction hash as a string
//   - Error if the redemption failed
func (h *DelegationHandler) RedeemDelegationDirectly(ctx context.Context, delegationData []byte) (string, error) {
	log.Printf("Attempting to redeem delegation, data size: %d bytes", len(delegationData))

	// Create client for the gRPC service
	delegationClient, err := client.NewDelegationClient()
	if err != nil {
		return "", fmt.Errorf("failed to create delegation service client: %w", err)
	}
	defer delegationClient.Close()

	// Call the client to redeem the delegation
	txHash, err := delegationClient.RedeemDelegation(ctx, delegationData)
	if err != nil {
		log.Printf("Delegation redemption failed: %v", err)
		return "", fmt.Errorf("delegation redemption failed: %w", err)
	}

	log.Printf("Delegation successfully redeemed, tx hash: %s", txHash)
	return txHash, nil
}

// RedeemDelegation godoc
// @Summary Redeem a delegation
// @Description Redeems a delegation using the Node.js gRPC service
// @Tags delegations
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param subscription_id path string true "Subscription ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/{product_id}/subscriptions/{subscription_id}/redeem [post]
func (h *DelegationHandler) RedeemDelegation(c *gin.Context) {
	productID := c.Param("product_id")
	_, err := uuid.Parse(productID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid product ID format", err)
		return
	}

	subscriptionID := c.Param("subscription_id")
	parsedSubscriptionID, err := uuid.Parse(subscriptionID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid subscription ID format", err)
		return
	}

	// Get the delegation from the database
	// TODO: Replace with actual database query once implemented
	delegation, err := h.getDelegationForSubscription(c.Request.Context(), parsedSubscriptionID)
	if err != nil {
		handleDBError(c, err, "Failed to fetch delegation")
		return
	}

	// Convert the delegation to a JSON string
	delegationJSON, err := json.Marshal(delegation)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to serialize delegation", err)
		return
	}

	// Call the method to redeem the delegation
	txHash, err := h.RedeemDelegationDirectly(c.Request.Context(), delegationJSON)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to redeem delegation", err)
		return
	}

	// Update the subscription status in the database
	// TODO: Implement this with your actual database structure
	// err = h.common.db.UpdateSubscriptionRedemption(c.Request.Context(), db.UpdateSubscriptionRedemptionParams{
	// 	ID:                parsedSubscriptionID,
	// 	TransactionHash:   txHash,
	// })
	// if err != nil {
	// 	sendError(c, http.StatusInternalServerError, "Failed to update subscription", err)
	// 	return
	// }

	sendSuccess(c, http.StatusOK, gin.H{
		"message":          "Delegation successfully redeemed",
		"transaction_hash": txHash,
	})
}

// DelegationData represents the delegation information stored in the database
type DelegationData struct {
	Delegate  string          `json:"delegate"`
	Delegator string          `json:"delegator"`
	Authority string          `json:"authority"`
	Caveats   json.RawMessage `json:"caveats"`
	Salt      string          `json:"salt"`
	Signature string          `json:"signature"`
}

// getDelegationForSubscription retrieves delegation data for a subscription
// TODO: Replace this with actual database query implementation
func (h *DelegationHandler) getDelegationForSubscription(ctx context.Context, subscriptionID uuid.UUID) (*DelegationData, error) {
	// Mock implementation - replace with real database query
	// This would be implemented in your database queries file
	return &DelegationData{
		Delegate:  "0x1234567890123456789012345678901234567890",
		Delegator: "0x0987654321098765432109876543210987654321",
		Authority: "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789",
		Caveats:   json.RawMessage("[]"),
		Salt:      "0x123456789",
		Signature: "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
	}, nil
}
