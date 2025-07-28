package requests

import "github.com/google/uuid"

// CreatePaymentLinkRequest represents the request to create a payment link
type CreatePaymentLinkRequest struct {
	ProductID       *uuid.UUID             `json:"product_id,omitempty"`
	PriceID         *uuid.UUID             `json:"price_id,omitempty"`
	AmountCents     *int64                 `json:"amount_cents,omitempty"`
	Currency        string                 `json:"currency" binding:"required_without=PriceID"`
	PaymentType     string                 `json:"payment_type,omitempty"` // defaults to "one_time"
	CollectEmail    *bool                  `json:"collect_email,omitempty"`
	CollectShipping *bool                  `json:"collect_shipping,omitempty"`
	CollectName     *bool                  `json:"collect_name,omitempty"`
	ExpiresIn       *int                   `json:"expires_in_hours,omitempty"` // Hours until expiration
	MaxUses         *int32                 `json:"max_uses,omitempty"`
	RedirectURL     *string                `json:"redirect_url,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// UpdatePaymentLinkRequest represents the request to update a payment link
type UpdatePaymentLinkRequest struct {
	Status      *string                `json:"status,omitempty" binding:"omitempty,oneof=active inactive"`
	ExpiresIn   *int                   `json:"expires_in_hours,omitempty"`
	MaxUses     *int32                 `json:"max_uses,omitempty"`
	RedirectURL *string                `json:"redirect_url,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
