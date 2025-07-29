package params

import "github.com/google/uuid"

// PaymentLinkCreateParams contains parameters for creating a payment link
type PaymentLinkCreateParams struct {
	WorkspaceID         uuid.UUID
	CustomerID          *uuid.UUID
	ProductID           *uuid.UUID
	PriceID             *uuid.UUID
	InvoiceID           *uuid.UUID
	AmountCents         int64
	Currency            string
	Title               string
	Description         string
	Metadata            map[string]interface{}
	ExpiresAt           *string
	MaxRedemptions      *int32
	RequireCustomerInfo bool
	AllowedNetworks     []uuid.UUID
	AllowedTokens       []uuid.UUID
	RedirectURL         *string
	CustomSlug          *string
}

// PaymentLinkUpdateParams contains parameters for updating a payment link
type PaymentLinkUpdateParams struct {
	Title               *string
	Description         *string
	ExpiresAt           *string
	MaxRedemptions      *int32
	IsActive            *bool
	Metadata            map[string]interface{}
	RequireCustomerInfo *bool
	AllowedNetworks     []uuid.UUID
	AllowedTokens       []uuid.UUID
	RedirectURL         *string
}
