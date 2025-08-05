package responses

import (
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/google/uuid"
)

// ProcessSubscriptionResult represents the result of processing a subscription
type ProcessSubscriptionResult struct {
	TransactionHash string    `json:"transaction_hash"`
	PaymentID       uuid.UUID `json:"payment_id"`
	Success         bool      `json:"success"`
	Error           string    `json:"error,omitempty"`
}

// ProcessDueSubscriptionsResult represents the result of processing all due subscriptions
type ProcessDueSubscriptionsResult struct {
	ProcessedCount   int         `json:"processed_count"`
	SuccessfulCount  int         `json:"successful_count"`
	FailedCount      int         `json:"failed_count"`
	ProcessedIDs     []uuid.UUID `json:"processed_ids"`
	FailedIDs        []uuid.UUID `json:"failed_ids"`
	ProcessingErrors []string    `json:"processing_errors,omitempty"`
}

// SubscriptionCustomerResponse represents the customer data within a subscription response
type SubscriptionCustomerResponse struct {
	ID                 uuid.UUID              `json:"id"`
	NumID              int64                  `json:"num_id"`
	Name               string                 `json:"name,omitempty"`
	Email              string                 `json:"email"`
	Phone              string                 `json:"phone,omitempty"`
	Description        string                 `json:"description,omitempty"`
	FinishedOnboarding bool                   `json:"finished_onboarding"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// SubscriptionLineItemResponse represents a line item in a subscription
type SubscriptionLineItemResponse struct {
	ID                   uuid.UUID              `json:"id"`
	SubscriptionID       uuid.UUID              `json:"subscription_id"`
	ProductID            uuid.UUID              `json:"product_id"`
	LineItemType         string                 `json:"line_item_type"` // 'base' or 'addon'
	Quantity             int32                  `json:"quantity"`
	UnitAmountInPennies  int32                  `json:"unit_amount_in_pennies"`
	Currency             string                 `json:"currency"`
	PriceType            string                 `json:"price_type"`
	IntervalType         *string                `json:"interval_type,omitempty"`
	TotalAmountInPennies int32                  `json:"total_amount_in_pennies"`
	IsActive             bool                   `json:"is_active"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
	Product              *ProductResponse       `json:"product,omitempty"`
}

// SubscriptionResponse represents a subscription along with its associated product details.
type SubscriptionResponse struct {
	ID                     uuid.UUID                      `json:"id"`
	NumID                  int64                          `json:"num_id"`
	WorkspaceID            uuid.UUID                      `json:"workspace_id"`
	CustomerID             uuid.UUID                      `json:"customer_id,omitempty"`
	CustomerName           string                         `json:"customer_name,omitempty"`
	CustomerEmail          string                         `json:"customer_email,omitempty"`
	Customer               *SubscriptionCustomerResponse  `json:"customer,omitempty"`
	Status                 string                         `json:"status"`
	CurrentPeriodStart     time.Time                      `json:"current_period_start"`
	CurrentPeriodEnd       time.Time                      `json:"current_period_end"`
	NextRedemptionDate     *time.Time                     `json:"next_redemption_date,omitempty"`
	TotalRedemptions       int32                          `json:"total_redemptions"`
	TotalAmountInCents     int32                          `json:"total_amount_in_cents"`
	TokenAmount            int32                          `json:"token_amount"`
	DelegationID           uuid.UUID                      `json:"delegation_id"`
	CustomerWalletID       *uuid.UUID                     `json:"customer_wallet_id,omitempty"`
	ExternalID             string                         `json:"external_id,omitempty"`
	PaymentSyncStatus      string                         `json:"payment_sync_status,omitempty"`
	PaymentSyncedAt        *time.Time                     `json:"payment_synced_at,omitempty"`
	PaymentSyncVersion     int32                          `json:"payment_sync_version,omitempty"`
	PaymentProvider        string                         `json:"payment_provider,omitempty"`
	InitialTransactionHash string                         `json:"initial_transaction_hash,omitempty"`
	Metadata               map[string]interface{}         `json:"metadata,omitempty"`
	CreatedAt              time.Time                      `json:"created_at"`
	UpdatedAt              time.Time                      `json:"updated_at"`
	TrialStart             *time.Time                     `json:"trial_start,omitempty"`
	TrialEnd               *time.Time                     `json:"trial_end,omitempty"`
	CanceledAt             *time.Time                     `json:"canceled_at,omitempty"`
	CancelAt               *time.Time                     `json:"cancel_at,omitempty"`
	CancellationReason     string                         `json:"cancellation_reason,omitempty"`
	PausedAt               *time.Time                     `json:"paused_at,omitempty"`
	PauseEndsAt            *time.Time                     `json:"pause_ends_at,omitempty"`
	ProductID              uuid.UUID                      `json:"product_id"`
	ProductTokenID         uuid.UUID                      `json:"product_token_id"`
	Product                *ProductResponse               `json:"product,omitempty"`
	ProductToken           *ProductTokenResponse          `json:"product_token,omitempty"`
	LineItems              []SubscriptionLineItemResponse `json:"line_items,omitempty"`
}

// GetRedemptionStatusResponse represents the response for the redemption status endpoint
type GetRedemptionStatusResponse struct {
	SubscriptionID   string     `json:"subscription_id"`
	Status           string     `json:"status"` // pending, success, failed
	Message          string     `json:"message"`
	LastRedemptionAt *time.Time `json:"last_redemption_at,omitempty"`
	LastAttemptedAt  *time.Time `json:"last_attempted_at,omitempty"`
	TotalRedemptions int32      `json:"total_redemptions"`
	NextRedemptionAt time.Time  `json:"next_redemption_at"`
	TransactionHash  string     `json:"transaction_hash,omitempty"`
	FailureReason    string     `json:"failure_reason,omitempty"`
}

// SubscribeToProductParams contains all parameters for creating a subscription
type SubscribeToProductParams struct {
	ProductID                 uuid.UUID
	SubscriberAddress         string
	ProductTokenID            string
	TokenAmount               string
	DelegationData            params.DelegationParams
	CypheraSmartWalletAddress string
	Addons                    []SubscriptionAddonParams
}

// SubscriptionAddonParams contains parameters for a subscription addon
type SubscriptionAddonParams struct {
	ProductID uuid.UUID
	Quantity  int32
}

// SubscribeToProductResult contains the result of subscription creation
type SubscribeToProductResult struct {
	Subscription *db.Subscription
	Success      bool
	ErrorMessage string
}

// Deprecated: Use SubscribeToProductParams instead
type SubscribeToProductByPriceIDParams = SubscribeToProductParams

// Deprecated: Use SubscribeToProductResult instead
type SubscribeToProductByPriceIDResult = SubscribeToProductResult
