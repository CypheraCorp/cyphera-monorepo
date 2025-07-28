package responses

import (
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// DunningCampaignResponse represents a dunning campaign
type DunningCampaignResponse struct {
	ID                    uuid.UUID          `json:"id"`
	WorkspaceID           uuid.UUID          `json:"workspace_id"`
	ConfigurationID       uuid.UUID          `json:"configuration_id"`
	SubscriptionID        pgtype.UUID        `json:"subscription_id"`
	PaymentID             pgtype.UUID        `json:"payment_id"`
	CustomerID            uuid.UUID          `json:"customer_id"`
	Status                string             `json:"status"`
	StartedAt             pgtype.Timestamptz `json:"started_at"`
	CompletedAt           pgtype.Timestamptz `json:"completed_at"`
	CurrentAttempt        int32              `json:"current_attempt"`
	NextRetryAt           pgtype.Timestamptz `json:"next_retry_at"`
	LastRetryAt           pgtype.Timestamptz `json:"last_retry_at"`
	Recovered             pgtype.Bool        `json:"recovered"`
	RecoveredAt           pgtype.Timestamptz `json:"recovered_at"`
	RecoveredAmountCents  pgtype.Int8        `json:"recovered_amount_cents"`
	FinalActionTaken      pgtype.Text        `json:"final_action_taken"`
	FinalActionAt         pgtype.Timestamptz `json:"final_action_at"`
	OriginalFailureReason pgtype.Text        `json:"original_failure_reason"`
	OriginalAmountCents   int64              `json:"original_amount_cents"`
	Currency              string             `json:"currency"`
	Metadata              []byte             `json:"metadata"`
	CreatedAt             pgtype.Timestamptz `json:"created_at"`
	UpdatedAt             pgtype.Timestamptz `json:"updated_at"`
	CustomerEmail         string             `json:"customer_email"`
	CustomerName          string             `json:"customer_name"`
	SubscriptionProductID uuid.UUID          `json:"subscription_product_id,omitempty"`
}

// DunningCampaignDetailResponse represents detailed dunning campaign information
type DunningCampaignDetailResponse struct {
	db.GetDunningCampaignRow
	ConfigurationName string              `json:"configuration_name"`
	MaxRetryAttempts  int32               `json:"max_retry_attempts"`
	RetryIntervalDays []int32             `json:"retry_interval_days"`
	CustomerEmail     string              `json:"customer_email"`
	CustomerName      string              `json:"customer_name"`
	Attempts          []db.DunningAttempt `json:"attempts"`
}

// CampaignStatsResponse represents dunning campaign statistics
type CampaignStatsResponse struct {
	ActiveCampaigns      int64   `json:"active_campaigns"`
	RecoveredCampaigns   int64   `json:"recovered_campaigns"`
	LostCampaigns        int64   `json:"lost_campaigns"`
	AtRiskAmountCents    int64   `json:"at_risk_amount_cents"`
	RecoveredAmountCents int64   `json:"recovered_amount_cents"`
	LostAmountCents      int64   `json:"lost_amount_cents"`
	RecoveryRate         float64 `json:"recovery_rate"`
}

// LocalProcessPaymentResponse contains the response from processing a payment
type LocalProcessPaymentResponse struct {
	TransactionHash string
	Status          string
	GasUsed         string
	BlockNumber     uint64
}
