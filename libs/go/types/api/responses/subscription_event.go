package responses

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionEventPriceInfo contains essential price information relevant to a subscription event
type SubscriptionEventPriceInfo struct {
	ID                  uuid.UUID `json:"id"`
	Type                string    `json:"type"`
	Currency            string    `json:"currency"`
	UnitAmountInPennies int64     `json:"unit_amount_in_pennies"`
	IntervalType        string    `json:"interval_type,omitempty"`
	TermLength          int32     `json:"term_length,omitempty"`
	CreatedAt           int64     `json:"created_at"`
	UpdatedAt           int64     `json:"updated_at"`
}

// SubscriptionEventResponse represents a detailed view of a subscription event
type SubscriptionEventResponse struct {
	ID                 uuid.UUID                  `json:"id"`
	SubscriptionID     uuid.UUID                  `json:"subscription_id"`
	EventType          string                     `json:"event_type"`
	TransactionHash    string                     `json:"transaction_hash,omitempty"`
	EventAmountInCents int32                      `json:"event_amount_in_cents"`
	EventOccurredAt    time.Time                  `json:"event_occurred_at"`
	ErrorMessage       string                     `json:"error_message,omitempty"`
	EventMetadata      []byte                     `json:"event_metadata,omitempty"`
	EventCreatedAt     time.Time                  `json:"event_created_at"`
	CustomerID         uuid.UUID                  `json:"customer_id,omitempty"`
	SubscriptionStatus string                     `json:"subscription_status,omitempty"`
	ProductID          uuid.UUID                  `json:"product_id,omitempty"`
	ProductName        string                     `json:"product_name,omitempty"`
	Customer           CustomerResponse           `json:"customer"`
	Product            ProductResponse            `json:"product"`
	PriceInfo          SubscriptionEventPriceInfo `json:"price_info"`
	ProductToken       ProductTokenResponse       `json:"product_token"`
	Network            NetworkResponse            `json:"network"`
}

type SubscriptionEventFullResponse struct {
	ID                 uuid.UUID                  `json:"id"`
	SubscriptionID     uuid.UUID                  `json:"subscription_id"`
	EventType          string                     `json:"event_type"`
	TransactionHash    string                     `json:"transaction_hash,omitempty"`
	EventAmountInCents int32                      `json:"event_amount_in_cents"`
	EventOccurredAt    time.Time                  `json:"event_occurred_at"`
	ErrorMessage       string                     `json:"error_message,omitempty"`
	EventMetadata      []byte                     `json:"event_metadata,omitempty"`
	EventCreatedAt     time.Time                  `json:"event_created_at"`
	CustomerID         uuid.UUID                  `json:"customer_id,omitempty"`
	SubscriptionStatus string                     `json:"subscription_status,omitempty"`
	ProductID          uuid.UUID                  `json:"product_id,omitempty"`
	ProductName        string                     `json:"product_name,omitempty"`
	Customer           CustomerResponse           `json:"customer"`
	Product            ProductResponse            `json:"product"`
	PriceInfo          SubscriptionEventPriceInfo `json:"price_info"`
	ProductToken       ProductTokenResponse       `json:"product_token"`
	Network            NetworkResponse            `json:"network"`
}

// ListSubscriptionEventsResult represents the result of listing subscription events
type ListSubscriptionEventsResult struct {
	Events []SubscriptionEventFullResponse `json:"events"`
	Total  int64                           `json:"total"`
}
