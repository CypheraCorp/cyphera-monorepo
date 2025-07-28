package requests

import "time"

// PaymentFailureWebhookRequest represents a payment failure webhook
type PaymentFailureWebhookRequest struct {
	Provider       string                 `json:"provider" binding:"required,oneof=stripe chargebee circle blockchain"`
	SubscriptionID string                 `json:"subscription_id" binding:"required,uuid"`
	CustomerID     string                 `json:"customer_id" binding:"required,uuid"`
	AmountCents    int64                  `json:"amount_cents" binding:"required,min=0"`
	Currency       string                 `json:"currency" binding:"required,len=3"`
	FailureReason  string                 `json:"failure_reason" binding:"required"`
	FailedAt       time.Time              `json:"failed_at" binding:"required"`
	Metadata       map[string]interface{} `json:"metadata"`
}
