package params

import (
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/google/uuid"
)

// CreatePaymentFromSubscriptionEventParams contains parameters for creating a payment from a subscription event
type CreatePaymentFromSubscriptionEventParams struct {
	SubscriptionEvent *db.SubscriptionEvent
	Subscription      *db.Subscription
	Product           *db.Product // Product now contains pricing info
	Customer          *db.Customer
	TransactionHash   string
	NetworkID         uuid.UUID
	TokenID           uuid.UUID
	CryptoAmount      string // Decimal as string
	ExchangeRate      string // Decimal as string
	GasFeeUSDCents    int64
	GasSponsored      bool
	InvoiceID         *uuid.UUID // Optional invoice ID to link payment to invoice
}

// CreateComprehensivePaymentParams contains parameters for creating a comprehensive payment
type CreateComprehensivePaymentParams struct {
	// Core payment details
	WorkspaceID     uuid.UUID
	CustomerID      uuid.UUID
	AmountCents     int64
	Currency        string
	PaymentMethod   string // "crypto", "card", "bank"
	TransactionHash *string
	ExternalID      *string
	ProviderName    *string // "circle", "stripe", etc.

	// Related entities
	SubscriptionID      *uuid.UUID
	SubscriptionEventID *uuid.UUID
	InvoiceID           *uuid.UUID
	ProductID           *uuid.UUID

	// Crypto-specific
	NetworkID         *uuid.UUID
	TokenID           *uuid.UUID
	CryptoAmount      *string
	ExchangeRate      *string
	GasFeeUSDCents    *int64
	GasFeeSponsoredBy *string

	// Tax information
	TaxExempt         bool
	CustomerVATNumber *string
	CustomerAddress   *PaymentAddress
	BusinessAddress   *PaymentAddress

	// Discount
	DiscountCode *string

	// Metadata
	Description         string
	StatementDescriptor *string
	TransactionType     string // "purchase", "subscription", "refund"
	ProductType         string // For tax categorization
	IsB2B               bool
}

// PaymentAddress represents an address for payment processing
type PaymentAddress struct {
	Street1    string `json:"street1"`
	Street2    string `json:"street2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// GetPaymentParams contains parameters for retrieving a payment
type GetPaymentParams struct {
	PaymentID   uuid.UUID
	WorkspaceID uuid.UUID
}

// ListPaymentsParams contains parameters for listing payments
type ListPaymentsParams struct {
	WorkspaceID    uuid.UUID
	CustomerID     *uuid.UUID
	SubscriptionID *uuid.UUID
	Status         *string
	StartDate      *string
	EndDate        *string
	Limit          int32
	Offset         int32
}

// UpdatePaymentStatusParams contains parameters for updating payment status
type UpdatePaymentStatusParams struct {
	PaymentID       uuid.UUID
	WorkspaceID     uuid.UUID
	Status          string
	TransactionHash *string
	FailureReason   *string
	ProcessedAt     *string
}

// CreateManualPaymentParams contains parameters for creating a manual payment
type CreateManualPaymentParams struct {
	WorkspaceID       uuid.UUID
	CustomerID        uuid.UUID
	SubscriptionID    *uuid.UUID
	InvoiceID         *uuid.UUID
	AmountInCents     int64
	Currency          string
	PaymentMethod     string
	Description       string
	TransactionHash   string
	NetworkID         *uuid.UUID
	TokenID           *uuid.UUID
	CryptoAmount      string
	ExchangeRate      string
	ExternalPaymentID string
	PaymentProvider   string
	GasFeeUSDCents    int64
	GasSponsored      bool
	ProcessedAt       string
	CreatedByUserID   uuid.UUID
	Metadata          map[string]interface{}
}
