package responses

import (
	"encoding/json"
)

// PriceResponse represents a price object in API responses
type PriceResponse struct {
	ID                  string          `json:"id"`
	Object              string          `json:"object"`
	ProductID           string          `json:"product_id"`
	Active              bool            `json:"active"`
	Type                string          `json:"type"`
	Nickname            string          `json:"nickname,omitempty"`
	Currency            string          `json:"currency"`
	UnitAmountInPennies int64           `json:"unit_amount_in_pennies"` // Using int64 for better compatibility
	IntervalType        string          `json:"interval_type,omitempty"`
	IntervalCount       int32           `json:"interval_count,omitempty"`
	TermLength          int32           `json:"term_length,omitempty"`
	Metadata            json.RawMessage `json:"metadata,omitempty" swaggertype:"object"`
	CreatedAt           int64           `json:"created_at"`
	UpdatedAt           int64           `json:"updated_at"`
}

// ProductResponse represents the standardized API response for product operations
type ProductResponse struct {
	ID            string                 `json:"id"`
	Object        string                 `json:"object"`
	WorkspaceID   string                 `json:"workspace_id"`
	WalletID      string                 `json:"wallet_id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	ImageURL      string                 `json:"image_url,omitempty"`
	URL           string                 `json:"url,omitempty"`
	Active        bool                   `json:"active"`
	Metadata      json.RawMessage        `json:"metadata,omitempty" swaggertype:"object"`
	CreatedAt     int64                  `json:"created_at"`
	UpdatedAt     int64                  `json:"updated_at"`
	Prices        []PriceResponse        `json:"prices,omitempty"`
	ProductTokens []ProductTokenResponse `json:"product_tokens,omitempty"`
}

// PublicProductResponse represents a product in public API responses (no auth required)
type PublicProductResponse struct {
	ID            string                 `json:"id"`
	AccountID     string                 `json:"account_id"`
	WorkspaceID   string                 `json:"workspace_id"`
	WalletAddress string                 `json:"wallet_address"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	ImageURL      string                 `json:"image_url,omitempty"`
	URL           string                 `json:"url,omitempty"`
	ProductTokens []ProductTokenResponse `json:"product_tokens,omitempty"`
	Price         PriceResponse          `json:"price"`
}

// ProductTokenResponse represents a product token in API responses
type ProductTokenResponse struct {
	ID              string          `json:"id"`
	Object          string          `json:"object"`
	ProductID       string          `json:"product_id"`
	ProductTokenID  string          `json:"product_token_id"` // ID of the product_token record
	NetworkID       string          `json:"network_id"`
	TokenID         string          `json:"token_id"`
	TokenName       string          `json:"token_name,omitempty"`
	TokenSymbol     string          `json:"token_symbol,omitempty"`
	TokenAddress    string          `json:"token_address,omitempty"`
	ContractAddress string          `json:"contract_address,omitempty"`
	TokenDecimals   int32           `json:"token_decimals,omitempty"`
	GasToken        bool            `json:"gas_token,omitempty"`
	ChainID         int32           `json:"chain_id,omitempty"`
	NetworkName     string          `json:"network_name,omitempty"`
	NetworkType     string          `json:"network_type,omitempty"`
	Active          bool            `json:"active"`
	Metadata        json.RawMessage `json:"metadata,omitempty" swaggertype:"object"`
	CreatedAt       int64           `json:"created_at"`
	UpdatedAt       int64           `json:"updated_at"`
}

// ProductDetailResponse represents a detailed product response with computed fields
type ProductDetailResponse struct {
	ID            string                 `json:"id"`
	Object        string                 `json:"object"`
	WorkspaceID   string                 `json:"workspace_id"`
	WalletID      string                 `json:"wallet_id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	ImageURL      string                 `json:"image_url,omitempty"`
	URL           string                 `json:"url,omitempty"`
	Active        bool                   `json:"active"`
	Metadata      json.RawMessage        `json:"metadata,omitempty" swaggertype:"object"`
	CreatedAt     int64                  `json:"created_at"`
	UpdatedAt     int64                  `json:"updated_at"`
	Prices        []PriceResponse        `json:"prices"`
	ProductTokens []ProductTokenResponse `json:"product_tokens"`
}

// ListProductsResponse represents a paginated list of products
type ListProductsResponse struct {
	Object  string            `json:"object"`
	Data    []ProductResponse `json:"data"`
	HasMore bool              `json:"has_more"`
	Total   int64             `json:"total"`
}

// ListProductsResult represents the result of listing products from services
type ListProductsResult struct {
	Products []ProductDetailResponse `json:"products"`
	Total    int64                   `json:"total"`
	HasMore  bool                    `json:"has_more"`
}
