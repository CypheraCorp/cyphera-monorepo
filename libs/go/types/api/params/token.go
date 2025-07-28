package params

import "github.com/google/uuid"

// TokenQuoteParams contains parameters for getting a token quote
type TokenQuoteParams struct {
	TokenID    uuid.UUID
	NetworkID  uuid.UUID
	AmountWei  string
	ToCurrency string // "USD", "EUR", etc.
}

// CreateTokenParams contains parameters for creating a token
type CreateTokenParams struct {
	NetworkID       uuid.UUID
	Symbol          string
	Name            string
	ContractAddress string
	Decimals        int32
	IconURL         *string
	PriceUSD        *float64
	IsStablecoin    bool
	IsNative        bool
	CoingeckoID     *string
	Metadata        map[string]interface{}
}

// UpdateTokenParams contains parameters for updating a token
type UpdateTokenParams struct {
	ID              uuid.UUID
	IconURL         *string
	PriceUSD        *float64
	CoingeckoID     *string
	IsActive        *bool
	Metadata        map[string]interface{}
}