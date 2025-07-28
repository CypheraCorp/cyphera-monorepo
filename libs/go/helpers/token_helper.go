package helpers

import (
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
)


// ToTokenResponse converts database model to API response
func ToTokenResponse(t db.Token) responses.TokenResponse {
	var deletedAt *int64
	if t.DeletedAt.Valid {
		unixTime := t.DeletedAt.Time.Unix()
		deletedAt = &unixTime
	}

	return responses.TokenResponse{
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
