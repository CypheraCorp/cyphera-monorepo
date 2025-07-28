package requests

import (
	"time"

	"github.com/google/uuid"
)

// CreateInvoiceRequest represents the request to create an invoice
type CreateInvoiceRequest struct {
	CustomerID     uuid.UUID                      `json:"customer_id" binding:"required"`
	SubscriptionID *uuid.UUID                     `json:"subscription_id,omitempty"`
	Currency       string                         `json:"currency" binding:"required,len=3"`
	DueDate        *time.Time                     `json:"due_date,omitempty"`
	LineItems      []CreateInvoiceLineItemRequest `json:"line_items" binding:"required,min=1,dive"`
	DiscountCode   *string                        `json:"discount_code,omitempty"`
	Metadata       map[string]interface{}         `json:"metadata,omitempty"`
}

// CreateInvoiceLineItemRequest represents a line item in the invoice creation request
type CreateInvoiceLineItemRequest struct {
	Description     string                 `json:"description" binding:"required"`
	Quantity        float64                `json:"quantity" binding:"required,gt=0"`
	UnitAmountCents int64                  `json:"unit_amount_cents" binding:"required,gte=0"`
	ProductID       *uuid.UUID             `json:"product_id,omitempty"`
	PriceID         *uuid.UUID             `json:"price_id,omitempty"`
	SubscriptionID  *uuid.UUID             `json:"subscription_id,omitempty"`
	PeriodStart     *time.Time             `json:"period_start,omitempty"`
	PeriodEnd       *time.Time             `json:"period_end,omitempty"`
	LineItemType    string                 `json:"line_item_type" binding:"required,oneof=product gas_fee"`
	GasFeePaymentID *uuid.UUID             `json:"gas_fee_payment_id,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}
