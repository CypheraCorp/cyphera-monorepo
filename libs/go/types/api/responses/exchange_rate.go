package responses

import (
	"time"

	"github.com/google/uuid"
)

// ExchangeRateResult contains the result of an exchange rate lookup
type ExchangeRateResult struct {
	Rate            float64    `json:"rate"`
	FromSymbol      string     `json:"from_symbol"`
	ToSymbol        string     `json:"to_symbol"`
	TokenID         *uuid.UUID `json:"token_id,omitempty"`
	NetworkID       *uuid.UUID `json:"network_id,omitempty"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Source          string     `json:"source"` // "coingecko", "chainlink", "manual"
	Confidence      float64    `json:"confidence"` // 0.0 to 1.0
}