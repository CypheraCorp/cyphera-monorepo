package requests

// CreateEventRequest represents the request body for creating a subscription event
type CreateEventRequest struct {
	SubscriptionID  string `json:"subscription_id" binding:"required"`
	EventType       string `json:"event_type" binding:"required"`
	TransactionHash string `json:"transaction_hash"`
	AmountInCents   int32  `json:"amount_in_cents"`
	OccurredAt      int64  `json:"occurred_at"`
	ErrorMessage    string `json:"error_message"`
	Metadata        []byte `json:"metadata"`
}
