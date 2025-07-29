package params

import (
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
)

// TaxCalculationParams contains parameters for tax calculation
type TaxCalculationParams struct {
	WorkspaceID       uuid.UUID
	CustomerID        uuid.UUID
	ProductID         *uuid.UUID
	SubscriptionID    *uuid.UUID
	AmountCents       int64
	Currency          string
	CustomerAddress   *business.Address
	BusinessAddress   *business.Address
	TaxExempt         bool
	TaxExemptionCode  *string
	CustomerVATNumber *string
	TransactionType   string // "purchase", "subscription", "refund"
	ProductType       string // "digital_goods", "physical_goods", "service"
	IsB2B             bool
}

// ShippingInfo contains shipping information for tax calculation
type ShippingInfo struct {
	Amount      int64
	FromAddress *business.Address
	ToAddress   *business.Address
}
