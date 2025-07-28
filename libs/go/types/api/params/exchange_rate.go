package params

import "github.com/google/uuid"

// ExchangeRateParams contains parameters for fetching exchange rates
type ExchangeRateParams struct {
	FromSymbol string     // e.g., "ETH", "BTC"
	ToSymbol   string     // e.g., "USD", "EUR"
	TokenID    *uuid.UUID // For database tracking
	NetworkID  *uuid.UUID // For network-specific rates
}
