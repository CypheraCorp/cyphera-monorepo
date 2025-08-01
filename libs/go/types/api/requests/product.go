package requests

import (
	"encoding/json"
)

// CreatePriceRequest represents the request body for creating a new price
type CreatePriceRequest struct {
	Active              bool            `json:"active"`
	Type                string          `json:"type" binding:"required"`
	Nickname            string          `json:"nickname"`
	Currency            string          `json:"currency" binding:"required"`
	UnitAmountInPennies int64           `json:"unit_amount_in_pennies" binding:"required"` // Using int64 for consistency
	IntervalType        string          `json:"interval_type"`
	IntervalCount       int32           `json:"interval_count"`
	TermLength          int32           `json:"term_length"`
	Metadata            json.RawMessage `json:"metadata" swaggertype:"object"`
}

// CreateProductRequest represents the request body for creating a product
// Updated to support embedded pricing (prices table merged into products)
type CreateProductRequest struct {
	Name          string                      `json:"name" binding:"required"`
	WalletID      string                      `json:"wallet_id" binding:"required"`
	Description   string                      `json:"description"`
	ImageURL      string                      `json:"image_url"`
	URL           string                      `json:"url"`
	Active        bool                        `json:"active"`
	Metadata      json.RawMessage             `json:"metadata" swaggertype:"object"`
	ProductTokens []CreateProductTokenRequest `json:"product_tokens,omitempty"`

	// Embedded price fields (required since prices table was merged into products)
	PriceType           string `json:"price_type" binding:"required"`
	Currency            string `json:"currency" binding:"required"`
	UnitAmountInPennies int64  `json:"unit_amount_in_pennies" binding:"required"`
	IntervalType        string `json:"interval_type"`
	TermLength          int32  `json:"term_length"`
	PriceNickname       string `json:"price_nickname"`
}

// UpdateProductRequest represents the request body for updating a product
type UpdateProductRequest struct {
	Name          string                      `json:"name,omitempty"`
	WalletID      string                      `json:"wallet_id,omitempty"`
	Description   string                      `json:"description,omitempty"`
	ImageURL      string                      `json:"image_url,omitempty"`
	URL           string                      `json:"url,omitempty"`
	Active        *bool                       `json:"active,omitempty"`
	Metadata      json.RawMessage             `json:"metadata,omitempty" swaggertype:"object"`
	ProductTokens []CreateProductTokenRequest `json:"product_tokens,omitempty"`
}

// CreateProductTokenRequest represents the request body for creating a product token
type CreateProductTokenRequest struct {
	NetworkID string          `json:"network_id" binding:"required"`
	TokenID   string          `json:"token_id" binding:"required"`
	Active    bool            `json:"active"`
	Metadata  json.RawMessage `json:"metadata" swaggertype:"object"`
}

// UpdateProductTokenRequest represents the request body for updating a product token
type UpdateProductTokenRequest struct {
	NetworkID string          `json:"network_id,omitempty"`
	TokenID   string          `json:"token_id,omitempty"`
	Active    *bool           `json:"active,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty" swaggertype:"object"`
}
