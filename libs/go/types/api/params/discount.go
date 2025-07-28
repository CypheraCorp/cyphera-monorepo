package params

import "github.com/google/uuid"

// DiscountApplicationParams contains parameters for applying a discount
type DiscountApplicationParams struct {
	WorkspaceID     uuid.UUID
	CustomerID      uuid.UUID
	ProductID       *uuid.UUID
	SubscriptionID  *uuid.UUID
	InvoiceID       *uuid.UUID
	DiscountCode    string
	AmountCents     int64
	Currency        string
	IsNewCustomer   bool
	CustomerEmail   string
	Metadata        map[string]interface{}
}