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
	Price                db.Price
	Product              db.Product
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
	Price          db.Price
	TokenAmount    int64
	DelegationData db.DelegationDatum
	PeriodStart    time.Time
	PeriodEnd      time.Time
	NextRedemption time.Time
}
