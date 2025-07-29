package helpers

import (
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
)

// Helper functions to convert database models to API responses
func toBasicProductTokenResponse(pt db.ProductsToken) responses.ProductTokenResponse {
	return responses.ProductTokenResponse{
		ID:        pt.ID.String(),
		ProductID: pt.ProductID.String(),
		NetworkID: pt.NetworkID.String(),
		TokenID:   pt.TokenID.String(),
		Active:    pt.Active,
		CreatedAt: pt.CreatedAt.Time.Unix(),
		UpdatedAt: pt.UpdatedAt.Time.Unix(),
	}
}

func toProductTokenResponse(pt db.GetProductTokenRow) responses.ProductTokenResponse {
	return responses.ProductTokenResponse{
		ID:              pt.ID.String(),
		ProductID:       pt.ProductID.String(),
		NetworkID:       pt.NetworkID.String(),
		TokenID:         pt.TokenID.String(),
		TokenName:       pt.TokenName,
		TokenSymbol:     pt.TokenSymbol,
		ContractAddress: pt.ContractAddress,
		GasToken:        pt.GasToken,
		ChainID:         pt.ChainID,
		NetworkName:     pt.NetworkName,
		NetworkType:     pt.NetworkType,
		Active:          pt.Active,
		CreatedAt:       pt.CreatedAt.Time.Unix(),
		UpdatedAt:       pt.UpdatedAt.Time.Unix(),
	}
}

func toProductTokenByIdsResponse(pt db.GetProductTokenByIdsRow) responses.ProductTokenResponse {
	return responses.ProductTokenResponse{
		ID:              pt.ID.String(),
		ProductID:       pt.ProductID.String(),
		NetworkID:       pt.NetworkID.String(),
		TokenID:         pt.TokenID.String(),
		TokenName:       pt.TokenName,
		TokenSymbol:     pt.TokenSymbol,
		ContractAddress: pt.ContractAddress,
		GasToken:        pt.GasToken,
		Active:          pt.Active,
		CreatedAt:       pt.CreatedAt.Time.Unix(),
		UpdatedAt:       pt.UpdatedAt.Time.Unix(),
	}
}

func toProductTokenByNetworkResponse(pt db.GetProductTokensByNetworkRow) responses.ProductTokenResponse {
	return responses.ProductTokenResponse{
		ID:              pt.ID.String(),
		ProductID:       pt.ProductID.String(),
		NetworkID:       pt.NetworkID.String(),
		TokenID:         pt.TokenID.String(),
		TokenName:       pt.TokenName,
		TokenSymbol:     pt.TokenSymbol,
		ContractAddress: pt.ContractAddress,
		GasToken:        pt.GasToken,
		Active:          pt.Active,
		CreatedAt:       pt.CreatedAt.Time.Unix(),
		UpdatedAt:       pt.UpdatedAt.Time.Unix(),
	}
}

func toActiveProductTokenByNetworkResponse(pt db.GetActiveProductTokensByNetworkRow) responses.ProductTokenResponse {
	return responses.ProductTokenResponse{
		ID:              pt.ID.String(),
		ProductID:       pt.ProductID.String(),
		NetworkID:       pt.NetworkID.String(),
		TokenID:         pt.TokenID.String(),
		TokenName:       pt.TokenName,
		TokenSymbol:     pt.TokenSymbol,
		ContractAddress: pt.ContractAddress,
		GasToken:        pt.GasToken,
		Active:          pt.Active,
		CreatedAt:       pt.CreatedAt.Time.Unix(),
		UpdatedAt:       pt.UpdatedAt.Time.Unix(),
	}
}

func ToActiveProductTokenByProductResponse(pt db.GetActiveProductTokensByProductRow) responses.ProductTokenResponse {
	return responses.ProductTokenResponse{
		ID:              pt.ID.String(),
		ProductID:       pt.ProductID.String(),
		NetworkID:       pt.NetworkID.String(),
		TokenID:         pt.TokenID.String(),
		TokenName:       pt.TokenName,
		TokenSymbol:     pt.TokenSymbol,
		ContractAddress: pt.ContractAddress,
		GasToken:        pt.GasToken,
		ChainID:         pt.ChainID,
		TokenDecimals:   int32(pt.Decimals),
		NetworkName:     pt.NetworkName,
		NetworkType:     pt.NetworkType,
		Active:          pt.Active,
		CreatedAt:       pt.CreatedAt.Time.Unix(),
		UpdatedAt:       pt.UpdatedAt.Time.Unix(),
	}
}

func toProductTokensByProductResponse(pt db.GetProductTokensByProductRow) responses.ProductTokenResponse {
	return responses.ProductTokenResponse{
		ID:              pt.ID.String(),
		ProductID:       pt.ProductID.String(),
		NetworkID:       pt.NetworkID.String(),
		TokenID:         pt.TokenID.String(),
		TokenName:       pt.TokenName,
		TokenSymbol:     pt.TokenSymbol,
		ContractAddress: pt.ContractAddress,
		GasToken:        pt.GasToken,
		ChainID:         pt.ChainID,
		NetworkName:     pt.NetworkName,
		NetworkType:     pt.NetworkType,
		Active:          pt.Active,
		CreatedAt:       pt.CreatedAt.Time.Unix(),
		UpdatedAt:       pt.UpdatedAt.Time.Unix(),
	}
}
