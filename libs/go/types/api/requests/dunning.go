package requests

import (
	"encoding/json"
)

// CreateDunningConfigurationRequest represents the request to create a dunning configuration
type CreateDunningConfigurationRequest struct {
	Name                   string          `json:"name" binding:"required"`
	Description            *string         `json:"description"`
	IsActive               bool            `json:"is_active"`
	IsDefault              bool            `json:"is_default"`
	MaxRetryAttempts       int32           `json:"max_retry_attempts" binding:"required,min=1,max=10"`
	RetryIntervalDays      []int32         `json:"retry_interval_days" binding:"required"`
	AttemptActions         json.RawMessage `json:"attempt_actions" binding:"required"`
	FinalAction            string          `json:"final_action" binding:"required,oneof=cancel pause downgrade"`
	FinalActionConfig      json.RawMessage `json:"final_action_config"`
	SendPreDunningReminder bool            `json:"send_pre_dunning_reminder"`
	PreDunningDays         int32           `json:"pre_dunning_days"`
	AllowCustomerRetry     bool            `json:"allow_customer_retry"`
	GracePeriodHours       int32           `json:"grace_period_hours"`
}

// CreateEmailTemplateRequest represents the request to create an email template
type CreateEmailTemplateRequest struct {
	Name               string          `json:"name" binding:"required"`
	TemplateType       string          `json:"template_type" binding:"required"`
	Subject            string          `json:"subject" binding:"required"`
	BodyHTML           string          `json:"body_html" binding:"required"`
	BodyText           *string         `json:"body_text"`
	AvailableVariables json.RawMessage `json:"available_variables"`
	IsActive           bool            `json:"is_active"`
}
