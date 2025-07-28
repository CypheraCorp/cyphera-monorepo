package params

import (
	"encoding/json"

	"github.com/google/uuid"
)

// CreateProductParams contains parameters for creating a product
type CreateProductParams struct {
	WorkspaceID   uuid.UUID
	WalletID      uuid.UUID
	Name          string
	Description   string
	ImageURL      string
	URL           string
	Active        bool
	Metadata      json.RawMessage
	Prices        []CreatePriceParams
	ProductTokens []CreateProductTokenParams
}

// CreatePriceParams contains parameters for creating a price
type CreatePriceParams struct {
	Active              bool
	Type                string
	Nickname            string
	Currency            string
	UnitAmountInPennies int64 // Using int64 for consistency
	IntervalType        string
	IntervalCount       int32
	TermLength          int32
	Metadata            json.RawMessage
}

// CreateProductTokenParams contains parameters for creating a product token
type CreateProductTokenParams struct {
	NetworkID uuid.UUID
	TokenID   uuid.UUID
	Active    bool
	Metadata  json.RawMessage
}

// GetProductParams contains parameters for getting a product
type GetProductParams struct {
	ProductID   uuid.UUID
	WorkspaceID uuid.UUID
}

// ListProductsParams contains parameters for listing products
type ListProductsParams struct {
	WorkspaceID uuid.UUID
	Limit       int32
	Offset      int32
	Active      *bool
}

// UpdateProductParams contains parameters for updating a product
type UpdateProductParams struct {
	ProductID     uuid.UUID
	WorkspaceID   uuid.UUID
	Name          *string
	WalletID      *uuid.UUID
	Description   *string
	ImageURL      *string
	URL           *string
	Active        *bool
	Metadata      json.RawMessage
	ProductTokens []CreateProductTokenParams
}

// Product represents the combined product data structure used in service results
type Product struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	WalletID    uuid.UUID
	Name        string
	Description string
	ImageURL    string
	URL         string
	Active      bool
	Metadata    json.RawMessage
	CreatedAt   int64
	UpdatedAt   int64
}

// ValidateSubscriptionParams contains parameters for validating a subscription request
type ValidateSubscriptionParams struct {
	SubscriberAddress         string
	PriceID                   string
	ProductTokenID            string
	TokenAmount               string
	ProductID                 uuid.UUID
	Delegation                DelegationParams
	CypheraSmartWalletAddress string // The expected delegate address
}
