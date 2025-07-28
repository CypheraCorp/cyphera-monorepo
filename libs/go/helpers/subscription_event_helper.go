package helpers

import (
	"context"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
)

// ToSubscriptionEventResponse converts a db.SubscriptionEvent to a SubscriptionEventResponse
func ToSubscriptionEventResponse(ctx context.Context, eventDetails db.SubscriptionEvent) (responses.SubscriptionEventResponse, error) {
	return responses.SubscriptionEventResponse{
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
func ToSubscriptionEventResponsePagination(ctx context.Context, eventDetails db.ListSubscriptionEventDetailsWithPaginationRow) (responses.SubscriptionEventFullResponse, error) {
	resp := responses.SubscriptionEventFullResponse{
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
		Customer: responses.CustomerResponse{
			ID:    eventDetails.CustomerID.String(),
			Email: eventDetails.CustomerEmail.String,
			Name:  eventDetails.CustomerName.String,
		},
		PriceInfo: responses.SubscriptionEventPriceInfo{
			ID:                  eventDetails.PriceID,
			Type:                string(eventDetails.PriceType),
			Currency:            string(eventDetails.PriceCurrency),
			UnitAmountInPennies: int64(eventDetails.PriceUnitAmountInPennies),
			IntervalType:        string(eventDetails.PriceIntervalType),
			TermLength:          int32(eventDetails.PriceTermLength),
		},
		ProductToken: responses.ProductTokenResponse{
			ID:          eventDetails.ProductTokenID.String(),
			TokenID:     eventDetails.ProductTokenTokenID.String(),
			TokenSymbol: eventDetails.ProductTokenSymbol,
		},
		Network: responses.NetworkResponse{
			ID:      eventDetails.NetworkID.String(),
			ChainID: eventDetails.NetworkChainID,
			Name:    eventDetails.NetworkName,
		},
	}
	return resp, nil
}
