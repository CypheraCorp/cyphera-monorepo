package responses

import (
	"encoding/json"
)

// ProductResponse represents the standardized API response for product operations
type ProductResponse struct {
	ID                  string                 `json:"id"`
	Object              string                 `json:"object"`
	WorkspaceID         string                 `json:"workspace_id"`
	WalletID            string                 `json:"wallet_id"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description,omitempty"`
	ImageURL            string                 `json:"image_url,omitempty"`
	URL                 string                 `json:"url,omitempty"`
	Active              bool                   `json:"active"`
	ProductType         string                 `json:"product_type,omitempty"`  // 'base' or 'addon'
	ProductGroup        string                 `json:"product_group,omitempty"` // Groups related products
	PriceType           string                 `json:"price_type"`              // 'recurring' or 'one_time'
	Currency            string                 `json:"currency"`
	UnitAmountInPennies int64                  `json:"unit_amount_in_pennies"`
	IntervalType        string                 `json:"interval_type,omitempty"` // 'month', 'year', etc.
	TermLength          int32                  `json:"term_length,omitempty"`   // Number of intervals
	PriceNickname       string                 `json:"price_nickname,omitempty"`
	PriceExternalID     string                 `json:"price_external_id,omitempty"`
	Metadata            json.RawMessage        `json:"metadata,omitempty" swaggertype:"object"`
	CreatedAt           int64                  `json:"created_at"`
	UpdatedAt           int64                  `json:"updated_at"`
	ProductTokens       []ProductTokenResponse `json:"product_tokens,omitempty"`
	AvailableAddons     []ProductAddonResponse `json:"available_addons,omitempty"`
}

// PublicProductResponse represents a product in public API responses (no auth required)
type PublicProductResponse struct {
	ID                  string                 `json:"id"`
	AccountID           string                 `json:"account_id"`
	WorkspaceID         string                 `json:"workspace_id"`
	WalletAddress       string                 `json:"wallet_address"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description,omitempty"`
	ImageURL            string                 `json:"image_url,omitempty"`
	URL                 string                 `json:"url,omitempty"`
	ProductType         string                 `json:"product_type,omitempty"`
	ProductGroup        string                 `json:"product_group,omitempty"`
	PriceType           string                 `json:"price_type"`
	Currency            string                 `json:"currency"`
	UnitAmountInPennies int64                  `json:"unit_amount_in_pennies"`
	IntervalType        string                 `json:"interval_type,omitempty"`
	TermLength          int32                  `json:"term_length,omitempty"`
	ProductTokens       []ProductTokenResponse `json:"product_tokens,omitempty"`
	AvailableAddons     []ProductAddonResponse `json:"available_addons,omitempty"`
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
	ID                  string                 `json:"id"`
	Object              string                 `json:"object"`
	WorkspaceID         string                 `json:"workspace_id"`
	WalletID            string                 `json:"wallet_id"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description,omitempty"`
	ImageURL            string                 `json:"image_url,omitempty"`
	URL                 string                 `json:"url,omitempty"`
	Active              bool                   `json:"active"`
	ProductType         string                 `json:"product_type,omitempty"`
	ProductGroup        string                 `json:"product_group,omitempty"`
	PriceType           string                 `json:"price_type"`
	Currency            string                 `json:"currency"`
	UnitAmountInPennies int64                  `json:"unit_amount_in_pennies"`
	IntervalType        string                 `json:"interval_type,omitempty"`
	TermLength          int32                  `json:"term_length,omitempty"`
	PriceNickname       string                 `json:"price_nickname,omitempty"`
	PriceExternalID     string                 `json:"price_external_id,omitempty"`
	Metadata            json.RawMessage        `json:"metadata,omitempty" swaggertype:"object"`
	CreatedAt           int64                  `json:"created_at"`
	UpdatedAt           int64                  `json:"updated_at"`
	ProductTokens       []ProductTokenResponse `json:"product_tokens"`
	AvailableAddons     []ProductAddonResponse `json:"available_addons,omitempty"`
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

// ProductAddonRelationshipResponse represents a product addon relationship in API responses
type ProductAddonRelationshipResponse struct {
	ID             string          `json:"id"`
	Object         string          `json:"object"`
	BaseProductID  string          `json:"base_product_id"`
	AddonProductID string          `json:"addon_product_id"`
	IsRequired     bool            `json:"is_required"`
	MaxQuantity    *int32          `json:"max_quantity"`
	MinQuantity    int32           `json:"min_quantity"`
	DisplayOrder   int32           `json:"display_order"`
	Metadata       json.RawMessage `json:"metadata,omitempty" swaggertype:"object"`
	CreatedAt      int64           `json:"created_at"`
	UpdatedAt      int64           `json:"updated_at"`
}

// ProductAddonResponse represents an addon product with its relationship details
type ProductAddonResponse struct {
	ProductAddonRelationshipResponse
	AddonProduct ProductResponse `json:"addon_product"`
}

// ProductWithAddonsResponse represents a product with its available addons
type ProductWithAddonsResponse struct {
	ProductResponse
	AvailableAddons []ProductAddonResponse `json:"available_addons,omitempty"`
}

// ListProductAddonsResponse represents a list of addons for a product
type ListProductAddonsResponse struct {
	Object string                 `json:"object"`
	Data   []ProductAddonResponse `json:"data"`
	Total  int64                  `json:"total"`
}
