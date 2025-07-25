package helpers

import (
	"github.com/cyphera/cyphera-api/libs/go/db"
)

// TokenResponse represents the standardized API response for token operations
type TokenResponse struct {
	ID              string `json:"id"`
	Object          string `json:"object"`
	NetworkID       string `json:"network_id"`
	GasToken        bool   `json:"gas_token"`
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	ContractAddress string `json:"contract_address"`
	Decimals        int32  `json:"decimals"`
	Active          bool   `json:"active"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
	DeletedAt       *int64 `json:"deleted_at,omitempty"`
}

// ToTokenResponse converts database model to API response
func ToTokenResponse(t db.Token) TokenResponse {
	var deletedAt *int64
	if t.DeletedAt.Valid {
		unixTime := t.DeletedAt.Time.Unix()
		deletedAt = &unixTime
	}

	return TokenResponse{
		ID:              t.ID.String(),
		Object:          "token",
		NetworkID:       t.NetworkID.String(),
		GasToken:        t.GasToken,
		Name:            t.Name,
		Symbol:          t.Symbol,
		ContractAddress: t.ContractAddress,
		Decimals:        t.Decimals,
		Active:          t.Active,
		CreatedAt:       t.CreatedAt.Time.Unix(),
		UpdatedAt:       t.UpdatedAt.Time.Unix(),
		DeletedAt:       deletedAt,
	}
}
