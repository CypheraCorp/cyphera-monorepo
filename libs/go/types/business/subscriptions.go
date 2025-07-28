package business

import (
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/google/uuid"
)

// ChangePreview shows what will happen with a subscription change
type ChangePreview struct {
	CurrentAmount    int64            `json:"current_amount"`
	NewAmount        int64            `json:"new_amount"`
	ProrationCredit  int64            `json:"proration_credit,omitempty"`
	ImmediateCharge  int64            `json:"immediate_charge,omitempty"`
	EffectiveDate    time.Time        `json:"effective_date"`
	ProrationDetails *ProrationResult `json:"proration_details,omitempty"`
	Message          string           `json:"message"`
}

// Note: ProrationResult will be imported from proration package

// SubscriptionExistsError is a custom error for when a subscription already exists
type SubscriptionExistsError struct {
	Subscription *db.Subscription
}

func (e *SubscriptionExistsError) Error() string {
	return "subscription already exists for this customer and product"
}

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

// ProcessSubscriptionResult represents the result of processing a subscription
type ProcessSubscriptionResult struct {
	TransactionHash string
	PaymentID       uuid.UUID
	Success         bool
	Error           error
}

// ProcessDueSubscriptionsResult represents the result of processing all due subscriptions
type ProcessDueSubscriptionsResult struct {
	ProcessedCount   int
	SuccessfulCount  int
	FailedCount      int
	ProcessedIDs     []uuid.UUID
	FailedIDs        []uuid.UUID
	ProcessingErrors []error
}
