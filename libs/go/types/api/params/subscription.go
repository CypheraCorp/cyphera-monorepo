package params

import (
	"encoding/json"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/google/uuid"
)

// ProcessSubscriptionParams represents parameters for processing a subscription
type ProcessSubscriptionParams struct {
	Subscription         db.Subscription
	Product              db.Product // Product now contains pricing
	Customer             db.Customer
	MerchantWallet       db.Wallet
	CustomerWallet       db.Wallet
	ProductToken         db.ProductsToken
	Token                db.Token
	Network              db.Network
	PaymentAmount        string
	RedemptionID         string
	PaymentDescription   string
	LastAttemptedAt      time.Time
	DelegationSignature  string
	DelegationExpiry     string
	AuthenticatedMessage string
	RedemptionAttempts   int32
	RedemptionTxHash     string
	// Deprecated: Use Product fields instead
	Price db.Product
}

// ListSubscriptionEventsParams contains parameters for listing subscription events
type ListSubscriptionEventsParams struct {
	WorkspaceID    uuid.UUID
	SubscriptionID *uuid.UUID
	CustomerID     *uuid.UUID
	EventType      *string
	StartDate      *string
	EndDate        *string
	Limit          int32
	Offset         int32
}

// CreateSubscriptionEventParams contains parameters for creating a subscription event
type CreateSubscriptionEventParams struct {
	WorkspaceID         uuid.UUID
	SubscriptionID      uuid.UUID
	EventType           db.SubscriptionEventType
	AmountInCents       int32
	TransactionHash     *string
	PreviousPeriodStart *time.Time
	PreviousPeriodEnd   *time.Time
	CurrentPeriodStart  *time.Time
	CurrentPeriodEnd    *time.Time
	FailureReason       *string
	Metadata            map[string]interface{}
}

// UpdateSubscriptionRequest represents the request body for updating a subscription
type UpdateSubscriptionRequest struct {
	CustomerID       string          `json:"customer_id"`
	ProductID        string          `json:"product_id"`
	ProductTokenID   string          `json:"product_token_id"`
	DelegationID     string          `json:"delegation_id"`
	CustomerWalletID string          `json:"customer_wallet_id"`
	Status           string          `json:"status"`
	StartDate        int64           `json:"start_date"`
	EndDate          int64           `json:"end_date"`
	NextRedemption   int64           `json:"next_redemption"`
	Metadata         json.RawMessage `json:"metadata"`
}

type CreateSubscriptionParams struct {
	Customer       db.Customer
	CustomerWallet db.CustomerWallet
	WorkspaceID    uuid.UUID
	ProductID      uuid.UUID
	ProductTokenID uuid.UUID
	Product        db.Product // Product now contains pricing
	TokenAmount    int64
	DelegationData db.DelegationDatum
	PeriodStart    time.Time
	PeriodEnd      time.Time
	NextRedemption time.Time
	Addons         []SubscriptionAddonParams
}

// SubscriptionAddonParams contains parameters for a subscription addon
type SubscriptionAddonParams struct {
	ProductID uuid.UUID
	Quantity  int32
}

// StoreDelegationDataParams contains parameters for storing delegation data
type StoreDelegationDataParams struct {
	Delegate  string
	Delegator string
	Authority string
	Caveats   json.RawMessage
	Salt      string
	Signature string
}

// InitialRedemptionParams contains parameters for performing initial redemption
type InitialRedemptionParams struct {
	Customer       db.Customer
	CustomerWallet db.CustomerWallet
	Subscription   db.Subscription
	Product        db.Product // Product now contains pricing
	ProductToken   db.GetProductTokenRow
	DelegationData StoreDelegationDataParams
	MerchantWallet db.Wallet
	Token          db.Token
	Network        db.Network
	TokenAmount    int64
}

// CreateSubscriptionWithDelegationParams contains parameters for creating subscription with delegation
type CreateSubscriptionWithDelegationParams struct {
	Product           db.Product // Product now contains pricing
	ProductToken      db.GetProductTokenRow
	MerchantWallet    db.Wallet
	Token             db.Token
	Network           db.Network
	DelegationData    StoreDelegationDataParams
	SubscriberAddress string
	ProductTokenID    uuid.UUID
	TokenAmount       int64
}

// SubscriptionCreationResult represents the result of subscription creation
type SubscriptionCreationResult struct {
	Subscription      *db.Subscription
	Customer          *db.Customer
	CustomerWallet    *db.CustomerWallet
	TransactionHash   string
	InitialRedemption bool
}

// SubscribeToProductParams contains all parameters for creating a subscription
type SubscribeToProductParams struct {
	ProductID                 uuid.UUID
	SubscriberAddress         string
	ProductTokenID            string
	TokenAmount               string
	DelegationData            DelegationParams
	CypheraSmartWalletAddress string
}

// SubscribeToProductByPriceIDParams deprecated alias for backward compatibility
type SubscribeToProductByPriceIDParams = SubscribeToProductParams

// DelegationParams contains delegation data for validation
type DelegationParams struct {
	Delegate  string
	Delegator string
	Authority string
	Salt      string
	Signature string
	Caveats   json.RawMessage
}
