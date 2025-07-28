package business

import "github.com/google/uuid"

// RedemptionTask represents a task to be processed by the redemption processor
type RedemptionTask struct {
	SubscriptionID uuid.UUID              `json:"subscription_id"`
	DelegationID   uuid.UUID              `json:"delegation_id"`
	ProductID      uuid.UUID              `json:"product_id"`
	ProductTokenID uuid.UUID              `json:"product_token_id"`
	AmountInCents  int32                  `json:"amount_in_cents"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}
