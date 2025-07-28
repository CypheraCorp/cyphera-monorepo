package params

import (
	"encoding/json"

	"github.com/google/uuid"
)

// DunningCampaignParams contains parameters for creating a dunning campaign
type DunningCampaignParams struct {
	SubscriptionID    uuid.UUID
	ConfigurationID   uuid.UUID
	TriggerReason     string
	OutstandingAmount int64
	Currency          string
	InitialPaymentID  *uuid.UUID
}

// DunningAttemptParams contains parameters for creating a dunning attempt
type DunningAttemptParams struct {
	CampaignID      uuid.UUID
	AttemptNumber   int32
	AttemptType     string
	ScheduledFor    string
	EmailTemplateID *uuid.UUID
	WebhookURL      *string
	RetryAmount     int64
}

// EmailTemplateParams contains parameters for creating email templates
type EmailTemplateParams struct {
	WorkspaceID     uuid.UUID
	ConfigurationID uuid.UUID
	TemplateName    string
	TemplateType    string
	Subject         string
	BodyHtml        string
	BodyText        string
	Variables       []string
	IsActive        bool
}

// LocalProcessPaymentParams contains parameters for processing a payment
type LocalProcessPaymentParams struct {
	DelegationID     string
	RecipientAddress string
	Amount           string
	TokenAddress     string
	NetworkID        uuid.UUID
}
type DunningConfigParams struct {
	WorkspaceID            uuid.UUID       `json:"workspace_id"`
	Name                   string          `json:"name"`
	Description            *string         `json:"description"`
	IsActive               bool            `json:"is_active"`
	IsDefault              bool            `json:"is_default"`
	MaxRetryAttempts       int32           `json:"max_retry_attempts"`
	RetryIntervalDays      []int32         `json:"retry_interval_days"`
	AttemptActions         json.RawMessage `json:"attempt_actions"`
	FinalAction            string          `json:"final_action"`
	FinalActionConfig      json.RawMessage `json:"final_action_config"`
	SendPreDunningReminder bool            `json:"send_pre_dunning_reminder"`
	PreDunningDays         int32           `json:"pre_dunning_days"`
	AllowCustomerRetry     bool            `json:"allow_customer_retry"`
	GracePeriodHours       int32           `json:"grace_period_hours"`
}
