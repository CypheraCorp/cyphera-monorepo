package params

import (
	"time"

	"github.com/google/uuid"
)

// InvoiceCreateParams contains parameters for creating an invoice
type InvoiceCreateParams struct {
	WorkspaceID    uuid.UUID
	CustomerID     uuid.UUID
	SubscriptionID *uuid.UUID
	Currency       string
	DueDate        *time.Time
	LineItems      []LineItemCreateParams
	DiscountCode   *string
	Metadata       map[string]interface{}
}

// LineItemCreateParams contains parameters for creating a line item
type LineItemCreateParams struct {
	Description     string
	Quantity        float64
	UnitAmountCents int64
	ProductID       *uuid.UUID
	PriceID         *uuid.UUID
	SubscriptionID  *uuid.UUID
	PeriodStart     *time.Time
	PeriodEnd       *time.Time
	LineItemType    string // "product", "gas_fee", "tax", "discount"
	GasFeePaymentID *uuid.UUID
	Metadata        map[string]interface{}
}
