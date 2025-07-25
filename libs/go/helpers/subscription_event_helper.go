package helpers

import (
	"context"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/google/uuid"
)

// SubscriptionEventPriceInfo contains essential price information relevant to a subscription event.
type SubscriptionEventPriceInfo struct {
	ID                  uuid.UUID `json:"id"`
	Type                string    `json:"type"`
	Currency            string    `json:"currency"`
	UnitAmountInPennies int32     `json:"unit_amount_in_pennies"`
	IntervalType        string    `json:"interval_type,omitempty"`
	IntervalCount       int32     `json:"interval_count,omitempty"`
	TermLength          int32     `json:"term_length,omitempty"`
}

// SubscriptionEventResponse represents a detailed view of a subscription event.
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
	PriceInfo          SubscriptionEventPriceInfo `json:"price_info,omitempty"`
	Subscription       SubscriptionResponse       `json:"subscription,omitempty"`
	ProductToken       ProductTokenResponse       `json:"product_token,omitempty"`
	Network            NetworkResponse            `json:"network,omitempty"`
	Customer           CustomerResponse           `json:"customer,omitempty"`
}

// ToSubscriptionEventResponse converts a db.SubscriptionEvent to a SubscriptionEventResponse
func ToSubscriptionEventResponse(ctx context.Context, eventDetails db.SubscriptionEvent) (SubscriptionEventResponse, error) {
	return SubscriptionEventResponse{
		ID:                 eventDetails.ID,
		SubscriptionID:     eventDetails.SubscriptionID,
		EventType:          string(eventDetails.EventType),
		TransactionHash:    eventDetails.TransactionHash.String,
		EventAmountInCents: eventDetails.AmountInCents,
		EventOccurredAt:    eventDetails.OccurredAt.Time,
		ErrorMessage:       eventDetails.ErrorMessage.String,
		EventMetadata:      eventDetails.Metadata,
		EventCreatedAt:     eventDetails.CreatedAt.Time,
	}, nil
}

// ToSubscriptionEventResponsePagination converts a db.ListSubscriptionEventDetailsWithPaginationRow to a SubscriptionEventResponse
func ToSubscriptionEventResponsePagination(ctx context.Context, eventDetails db.ListSubscriptionEventDetailsWithPaginationRow) (SubscriptionEventResponse, error) {
	resp := SubscriptionEventResponse{
		ID:                 eventDetails.SubscriptionEventID,
		SubscriptionID:     eventDetails.SubscriptionID,
		EventType:          string(eventDetails.EventType),
		TransactionHash:    eventDetails.TransactionHash.String,
		EventAmountInCents: eventDetails.EventAmountInCents,
		EventOccurredAt:    eventDetails.EventOccurredAt.Time,
		ErrorMessage:       eventDetails.ErrorMessage.String,
		EventMetadata:      eventDetails.EventMetadata,
		EventCreatedAt:     eventDetails.EventCreatedAt.Time,
		SubscriptionStatus: string(eventDetails.SubscriptionStatus),
		ProductID:          eventDetails.ProductID,
		ProductName:        eventDetails.ProductName,
		Customer: CustomerResponse{
			ID:    eventDetails.CustomerID.String(),
			Email: eventDetails.CustomerEmail.String,
			Name:  eventDetails.CustomerName.String,
		},
		PriceInfo: SubscriptionEventPriceInfo{
			ID:                  eventDetails.PriceID,
			Type:                string(eventDetails.PriceType),
			Currency:            string(eventDetails.PriceCurrency),
			UnitAmountInPennies: eventDetails.PriceUnitAmountInPennies,
			IntervalType:        string(eventDetails.PriceIntervalType),
			TermLength:          int32(eventDetails.PriceTermLength),
		},
		ProductToken: ProductTokenResponse{
			ID:          eventDetails.ProductTokenID.String(),
			TokenID:     eventDetails.ProductTokenTokenID.String(),
			TokenSymbol: eventDetails.ProductTokenSymbol,
		},
		Network: NetworkResponse{
			ID:      eventDetails.NetworkID.String(),
			ChainID: eventDetails.NetworkChainID,
			Name:    eventDetails.NetworkName,
		},
	}
	return resp, nil
}
