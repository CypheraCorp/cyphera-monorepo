package helpers

import (
	"github.com/cyphera/cyphera-api/libs/go/db"
)

// ProductTokenResponse represents the standardized API response for product token operations
type ProductTokenResponse struct {
	ID              string `json:"id"`
	ProductID       string `json:"product_id"`
	NetworkID       string `json:"network_id"`
	TokenID         string `json:"token_id"`
	TokenName       string `json:"token_name,omitempty"`
	TokenSymbol     string `json:"token_symbol,omitempty"`
	TokenDecimals   int32  `json:"token_decimals,omitempty"`
	ContractAddress string `json:"contract_address,omitempty"`
	GasToken        bool   `json:"gas_token,omitempty"`
	ChainID         int32  `json:"chain_id,omitempty"`
	NetworkName     string `json:"network_name,omitempty"`
	NetworkType     string `json:"network_type,omitempty"`
	Active          bool   `json:"active"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

// CreateProductTokenRequest represents the request body for creating a product token
type CreateProductTokenRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	NetworkID string `json:"network_id" binding:"required"`
	TokenID   string `json:"token_id" binding:"required"`
	Active    bool   `json:"active"`
}

// UpdateProductTokenRequest represents the request body for updating a product token
type UpdateProductTokenRequest struct {
	Active bool `json:"active" binding:"required"`
}

// Helper functions to convert database models to API responses
func toBasicProductTokenResponse(pt db.ProductsToken) ProductTokenResponse {
	return ProductTokenResponse{
		ID:        pt.ID.String(),
		ProductID: pt.ProductID.String(),
		NetworkID: pt.NetworkID.String(),
		TokenID:   pt.TokenID.String(),
		Active:    pt.Active,
		CreatedAt: pt.CreatedAt.Time.Unix(),
		UpdatedAt: pt.UpdatedAt.Time.Unix(),
	}
}

func toProductTokenResponse(pt db.GetProductTokenRow) ProductTokenResponse {
	return ProductTokenResponse{
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

func toProductTokenByIdsResponse(pt db.GetProductTokenByIdsRow) ProductTokenResponse {
	return ProductTokenResponse{
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

func toProductTokenByNetworkResponse(pt db.GetProductTokensByNetworkRow) ProductTokenResponse {
	return ProductTokenResponse{
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

func toActiveProductTokenByNetworkResponse(pt db.GetActiveProductTokensByNetworkRow) ProductTokenResponse {
	return ProductTokenResponse{
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

func ToActiveProductTokenByProductResponse(pt db.GetActiveProductTokensByProductRow) ProductTokenResponse {
	return ProductTokenResponse{
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

func toProductTokensByProductResponse(pt db.GetProductTokensByProductRow) ProductTokenResponse {
	return ProductTokenResponse{
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
