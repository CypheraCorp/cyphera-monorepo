package requests

import "github.com/google/uuid"

// LineItemUpdate represents a change to subscription line items
type LineItemUpdate struct {
	Action         string    `json:"action"` // add, update, remove
	LineItemID     uuid.UUID `json:"line_item_id,omitempty"`
	ProductID      uuid.UUID `json:"product_id,omitempty"`
	PriceID        uuid.UUID `json:"price_id,omitempty"`
	ProductTokenID uuid.UUID `json:"product_token_id,omitempty"`
	Quantity       int       `json:"quantity"`
	UnitAmount     int64     `json:"unit_amount,omitempty"`
}

// UpgradeSubscriptionRequest represents the request to upgrade a subscription
type UpgradeSubscriptionRequest struct {
	LineItems []LineItemUpdate `json:"line_items" binding:"required,min=1"`
	Reason    string           `json:"reason" binding:"max=500"`
}

// DowngradeSubscriptionRequest represents the request to downgrade a subscription
type DowngradeSubscriptionRequest struct {
	LineItems []LineItemUpdate `json:"line_items" binding:"required,min=1"`
	Reason    string           `json:"reason" binding:"max=500"`
}

// CancelSubscriptionRequest represents the request to cancel a subscription
type CancelSubscriptionRequest struct {
	Reason   string `json:"reason" binding:"required,max=500"`
	Feedback string `json:"feedback" binding:"max=1000"`
}

// PauseSubscriptionRequest represents the request to pause a subscription
type PauseSubscriptionRequest struct {
	PauseUntil string `json:"pause_until,omitempty"` // RFC3339 timestamp
	Reason     string `json:"reason" binding:"required,max=500"`
}

// PreviewChangeRequest represents the request to preview a subscription change
type PreviewChangeRequest struct {
	ChangeType string           `json:"change_type" binding:"required,oneof=upgrade downgrade cancel"`
	LineItems  []LineItemUpdate `json:"line_items,omitempty"`
}
