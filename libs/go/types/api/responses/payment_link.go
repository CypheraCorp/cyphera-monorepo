package responses

import (
	"time"

	"github.com/google/uuid"
)

// PaymentLinkResponse represents a payment link with full details
type PaymentLinkResponse struct {
	ID              uuid.UUID              `json:"id"`
	WorkspaceID     uuid.UUID              `json:"workspace_id"`
	Slug            string                 `json:"slug"`
	URL             string                 `json:"url"`
	Status          string                 `json:"status"`
	ProductID       *uuid.UUID             `json:"product_id,omitempty"`
	PriceID         *uuid.UUID             `json:"price_id,omitempty"`
	AmountCents     *int64                 `json:"amount_cents,omitempty"`
	Currency        string                 `json:"currency"`
	PaymentType     string                 `json:"payment_type"`
	CollectEmail    bool                   `json:"collect_email"`
	CollectShipping bool                   `json:"collect_shipping"`
	CollectName     bool                   `json:"collect_name"`
	ExpiresAt       *time.Time             `json:"expires_at,omitempty"`
	MaxUses         *int32                 `json:"max_uses,omitempty"`
	UsedCount       int32                  `json:"used_count"`
	RedirectURL     *string                `json:"redirect_url,omitempty"`
	QRCodeURL       *string                `json:"qr_code_url,omitempty"`
	QRCodeData      *string                `json:"qr_code_data,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ListPaymentLinksResponse represents a list of payment links
type ListPaymentLinksResponse struct {
	Object     string                `json:"object"`
	Data       []PaymentLinkResponse `json:"data"`
	HasMore    bool                  `json:"has_more,omitempty"`
	TotalCount int64                 `json:"total_count,omitempty"`
}
