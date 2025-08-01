package requests

import (
	"encoding/json"

	"github.com/cyphera/cyphera-api/libs/go/types/business"
)

// UpdateSubscriptionRequest represents the request structure for updating a subscription
type UpdateSubscriptionRequest struct {
	CustomerID       string          `json:"customer_id"`
	ProductID        string          `json:"product_id"`
	ProductTokenID   string          `json:"product_token_id"`
	DelegationID     string          `json:"delegation_id"`
	CustomerWalletID string          `json:"customer_wallet_id"`
	Status           string          `json:"status"`
	StartDate        int64           `json:"start_date"`
	EndDate          int64           `json:"end_date"`
	NextRedemption   int64           `json:"next_redemption"`
	Metadata         json.RawMessage `json:"metadata"`
}

// SubscribeRequest represents the request body for subscribing to a product
type SubscribeRequest struct {
	SubscriberAddress string                    `json:"subscriber_address" binding:"required"`
	ProductID         string                    `json:"product_id" binding:"required"`
	ProductTokenID    string                    `json:"product_token_id" binding:"required"`
	TokenAmount       string                    `json:"token_amount" binding:"required"`
	Delegation        business.DelegationStruct `json:"delegation" binding:"required"`
}
