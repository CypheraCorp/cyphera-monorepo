package responses

import (
	"time"

	"github.com/google/uuid"
)

// TaxDetail contains tax calculation details
type TaxDetail struct {
	JurisdictionID   string  `json:"jurisdiction_id"`
	JurisdictionName string  `json:"jurisdiction_name"`
	TaxRate          float64 `json:"tax_rate"`
	TaxAmountCents   int64   `json:"tax_amount_cents"`
	TaxType          string  `json:"tax_type"` // "vat", "sales_tax", etc.
}

// InvoiceResponse represents an invoice in API responses
type InvoiceResponse struct {
	ID               uuid.UUID                 `json:"id"`
	WorkspaceID      uuid.UUID                 `json:"workspace_id"`
	CustomerID       uuid.UUID                 `json:"customer_id"`
	SubscriptionID   *uuid.UUID                `json:"subscription_id,omitempty"`
	InvoiceNumber    string                    `json:"invoice_number"`
	Status           string                    `json:"status"`
	Currency         string                    `json:"currency"`
	DueDate          *time.Time                `json:"due_date,omitempty"`
	ProductSubtotal  int64                     `json:"product_subtotal"`
	GasFeesSubtotal  int64                     `json:"gas_fees_subtotal"`
	SponsoredGasFees int64                     `json:"sponsored_gas_fees"`
	TaxAmount        int64                     `json:"tax_amount"`
	DiscountAmount   int64                     `json:"discount_amount"`
	TotalAmount      int64                     `json:"total_amount"`
	CustomerTotal    int64                     `json:"customer_total"`
	LineItems        []InvoiceLineItemResponse `json:"line_items"`
	TaxDetails       []TaxDetail               `json:"tax_details"`
	PaymentLinkID    *uuid.UUID                `json:"payment_link_id,omitempty"`
	PaymentLinkURL   *string                   `json:"payment_link_url,omitempty"`
	CreatedAt        time.Time                 `json:"created_at"`
	UpdatedAt        time.Time                 `json:"updated_at"`
}

// InvoiceLineItemResponse represents a line item in API responses
type InvoiceLineItemResponse struct {
	ID              uuid.UUID              `json:"id"`
	Description     string                 `json:"description"`
	Quantity        float64                `json:"quantity"`
	UnitAmountCents int64                  `json:"unit_amount_cents"`
	AmountCents     int64                  `json:"amount_cents"`
	Currency        string                 `json:"currency"`
	LineItemType    string                 `json:"line_item_type"`
	IsGasSponsored  bool                   `json:"is_gas_sponsored,omitempty"`
	GasSponsorType  *string                `json:"gas_sponsor_type,omitempty"`
	GasSponsorName  *string                `json:"gas_sponsor_name,omitempty"`
	ProductID       *uuid.UUID             `json:"product_id,omitempty"`
	PriceID         *uuid.UUID             `json:"price_id,omitempty"`
	PeriodStart     *time.Time             `json:"period_start,omitempty"`
	PeriodEnd       *time.Time             `json:"period_end,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}
