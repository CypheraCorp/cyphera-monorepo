package helpers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/cyphera/cyphera-api/libs/go/constants"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/google/uuid"
)

// ProductDetailResponse represents detailed product response with extended fields
type ProductDetailResponse struct {
	ID            string                    `json:"id"`
	Object        string                    `json:"object"`
	WorkspaceID   string                    `json:"workspace_id"`
	WalletID      string                    `json:"wallet_id"`
	Name          string                    `json:"name"`
	Description   string                    `json:"description,omitempty"`
	ImageURL      string                    `json:"image_url,omitempty"`
	URL           string                    `json:"url,omitempty"`
	Active        bool                      `json:"active"`
	Metadata      json.RawMessage           `json:"metadata,omitempty" swaggertype:"object"`
	CreatedAt     int64                     `json:"created_at"`
	UpdatedAt     int64                     `json:"updated_at"`
	Prices        []PriceResponse           `json:"prices,omitempty"`
	ProductTokens []ProductTokenResponse    `json:"product_tokens,omitempty"`
}

// PublicProductTokenResponse represents a product token in public API responses
type PublicProductTokenResponse struct {
	ProductTokenID string `json:"product_token_id"`
	NetworkID      string `json:"network_id"`
	NetworkName    string `json:"network_name"`
	NetworkChainID string `json:"network_chain_id"`
	TokenID        string `json:"token_id"`
	TokenName      string `json:"token_name"`
	TokenSymbol    string `json:"token_symbol"`
	TokenDecimals  int32  `json:"token_decimals"`
	TokenAddress   string `json:"token_address"`
}

// PublicProductResponse represents a product in public API responses (no auth required)
type PublicProductResponse struct {
	ID            string                       `json:"id"`
	AccountID     string                       `json:"account_id"`
	WorkspaceID   string                       `json:"workspace_id"`
	WalletAddress string                       `json:"wallet_address"`
	Name          string                       `json:"name"`
	Description   string                       `json:"description,omitempty"`
	ImageURL      string                       `json:"image_url,omitempty"`
	URL           string                       `json:"url,omitempty"`
	ProductTokens []PublicProductTokenResponse `json:"product_tokens"`
	Price         PriceResponse                `json:"price"`
}

// ToProductDetailResponse converts database product model to detailed API response
func ToProductDetailResponse(p db.Product, dbPrices []db.Price) ProductDetailResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(p.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling product metadata: %v", err)
	}

	apiPrices := make([]PriceResponse, len(dbPrices))
	for i, dbPrice := range dbPrices {
		apiPrices[i] = ToPriceResponseFromDB(dbPrice)
	}

	return ProductDetailResponse{
		ID:          p.ID.String(),
		Object:      "product",
		WorkspaceID: p.WorkspaceID.String(),
		WalletID:    p.WalletID.String(),
		Name:        p.Name,
		Description: p.Description.String,
		ImageURL:    p.ImageUrl.String,
		URL:         p.Url.String,
		Active:      p.Active,
		Metadata:    p.Metadata,
		Prices:      apiPrices,
		CreatedAt:   p.CreatedAt.Time.Unix(),
		UpdatedAt:   p.UpdatedAt.Time.Unix(),
	}
}

// ToPriceResponseFromDB converts database price model to API response
func ToPriceResponseFromDB(p db.Price) PriceResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(p.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling price metadata: %v", err)
	}
	return PriceResponse{
		ID:                  p.ID.String(),
		Object:              "price",
		ProductID:           p.ProductID.String(),
		Active:              p.Active,
		Type:                string(p.Type),
		Nickname:            p.Nickname.String,
		Currency:            string(p.Currency),
		UnitAmountInPennies: int64(p.UnitAmountInPennies), // Convert int32 to int64 to match existing type
		IntervalType:        string(p.IntervalType),
		TermLength:          p.TermLength,
		Metadata:            p.Metadata,
		CreatedAt:           p.CreatedAt.Time.Unix(),
		UpdatedAt:           p.UpdatedAt.Time.Unix(),
	}
}

// ToPublicProductResponse converts database models to public API response
func ToPublicProductResponse(workspace db.Workspace, product db.Product, price db.Price, productTokens []db.GetActiveProductTokensByProductRow, wallet db.Wallet) PublicProductResponse {
	publicProductTokens := make([]PublicProductTokenResponse, len(productTokens))
	for i, pt := range productTokens {
		publicProductTokens[i] = PublicProductTokenResponse{
			ProductTokenID: pt.ID.String(),
			NetworkID:      pt.NetworkID.String(),
			NetworkName:    pt.NetworkName,
			NetworkChainID: strconv.Itoa(int(pt.ChainID)),
			TokenID:        pt.TokenID.String(),
			TokenName:      pt.TokenName,
			TokenSymbol:    pt.TokenSymbol,
			TokenDecimals:  int32(pt.Decimals),
		}
	}
	return PublicProductResponse{
		ID:            product.ID.String(),
		AccountID:     workspace.AccountID.String(),
		WorkspaceID:   workspace.ID.String(),
		WalletAddress: wallet.WalletAddress,
		Name:          product.Name,
		Description:   product.Description.String,
		ImageURL:      product.ImageUrl.String,
		URL:           product.Url.String,
		ProductTokens: publicProductTokens,
		Price:         ToPriceResponseFromDB(price),
	}
}

// ValidatePriceType validates the price type and returns a db.PriceType if valid
func ValidatePriceType(priceTypeStr string) (db.PriceType, error) {
	if priceTypeStr == "" {
		return "", fmt.Errorf("price type is required")
	}
	if priceTypeStr != string(db.PriceTypeRecurring) && priceTypeStr != string(db.PriceTypeOneOff) {
		return "", fmt.Errorf("invalid price type. Must be '%s' or '%s'", db.PriceTypeRecurring, db.PriceTypeOneOff)
	}
	return db.PriceType(priceTypeStr), nil
}

// ValidateIntervalType validates the interval type and returns a db.IntervalType if valid
func ValidateIntervalType(intervalType string) (db.IntervalType, error) {
	if intervalType == "" {
		return "", nil
	}
	validIntervalTypes := map[string]bool{
		constants.IntervalType1Minute:  true,
		constants.IntervalType5Minutes: true,
		constants.IntervalTypeDaily:    true,
		constants.IntervalTypeWeekly:   true,
		constants.IntervalTypeMonthly:  true,
		constants.IntervalTypeYearly:   true,
	}
	if !validIntervalTypes[intervalType] {
		return "", fmt.Errorf("invalid interval type")
	}
	return db.IntervalType(intervalType), nil
}

// ValidateWalletID validates and parses the wallet ID
func ValidateWalletID(walletID string) (uuid.UUID, error) {
	if walletID == "" {
		return uuid.Nil, fmt.Errorf("wallet ID is required")
	}
	parsed, err := uuid.Parse(walletID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid wallet ID format: %w", err)
	}
	return parsed, nil
}

// ValidatePriceTermLength validates price term length based on price type
func ValidatePriceTermLength(priceType db.PriceType, termLength int32, intervalType db.IntervalType, intervalCount int32) error {
	if priceType == db.PriceTypeRecurring {
		if intervalType == "" || intervalCount <= 0 {
			return fmt.Errorf("interval_type and interval_count are required for recurring prices")
		}
		if termLength <= 0 {
			return fmt.Errorf("term length must be greater than 0 for recurring prices")
		}
	} else if priceType == db.PriceTypeOneOff {
		if intervalType != "" || intervalCount != 0 || termLength != 0 {
			return fmt.Errorf("interval_type, interval_count, and term_length must not be set for one_off prices")
		}
	}
	return nil
}

// ValidatePriceInPennies validates that the price value is non-negative
func ValidatePriceInPennies(price int32) error {
	if price < 0 {
		return fmt.Errorf("unit_amount_in_pennies cannot be negative")
	}
	return nil
}

// ValidateProductName validates product name length
func ValidateProductName(name string) error {
	if name == "" {
		return fmt.Errorf("product name is required")
	}
	if len(name) > 255 {
		return fmt.Errorf("name must be less than 255 characters")
	}
	return nil
}

// ValidateProductDescription validates product description length
func ValidateProductDescription(description string) error {
	if description != "" && len(description) > 1000 {
		return fmt.Errorf("description must be less than 1000 characters")
	}
	return nil
}

// ValidateImageURL validates image URL format
func ValidateImageURL(imageURL string) error {
	if imageURL != "" {
		if _, err := url.ParseRequestURI(imageURL); err != nil {
			return fmt.Errorf("invalid image URL format: %w", err)
		}
	}
	return nil
}

// ValidateProductURL validates product URL format
func ValidateProductURL(productURL string) error {
	if productURL != "" {
		if _, err := url.ParseRequestURI(productURL); err != nil {
			return fmt.Errorf("invalid URL format: %w", err)
		}
	}
	return nil
}

// ValidateMetadata validates metadata JSON format
func ValidateMetadata(metadata json.RawMessage) error {
	if metadata != nil && !json.Valid(metadata) {
		return fmt.Errorf("invalid metadata JSON format")
	}
	return nil
}

// NormalizeWalletAddress normalizes wallet address based on network type
func NormalizeWalletAddress(address, networkType string) string {
	if networkType == string(db.NetworkTypeEvm) {
		return strings.ToLower(address)
	}
	return address
}

// DetermineNetworkType maps network names to their network types
func DetermineNetworkType(networkTypeStr string) string {
	networkType := strings.ToLower(networkTypeStr)
	switch networkType {
	case "ethereum", "sepolia", "goerli", "arbitrum", "optimism", "polygon", "base", "linea":
		return string(db.NetworkTypeEvm)
	case "solana":
		return string(db.NetworkTypeSolana)
	case "cosmos":
		return string(db.NetworkTypeCosmos)
	case "bitcoin":
		return string(db.NetworkTypeBitcoin)
	case "polkadot":
		return string(db.NetworkTypePolkadot)
	default:
		return string(db.NetworkTypeEvm)
	}
}

// MarshalCaveats converts the caveats array to JSON for storage
func MarshalCaveats(caveats interface{}) json.RawMessage {
	bytes, err := json.Marshal(caveats)
	if err != nil {
		return json.RawMessage("{}")
	}
	return bytes
}