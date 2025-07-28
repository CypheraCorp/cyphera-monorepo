package business

import "github.com/google/uuid"

// LocalProcessPaymentParams contains parameters for processing a payment
type LocalProcessPaymentParams struct {
	DelegationID     string
	RecipientAddress string
	Amount           string
	TokenAddress     string
	NetworkID        uuid.UUID
}

// LocalProcessPaymentResponse contains the response from processing a payment
type LocalProcessPaymentResponse struct {
	TransactionHash string
	Status          string
	GasUsed         string
	BlockNumber     uint64
}
