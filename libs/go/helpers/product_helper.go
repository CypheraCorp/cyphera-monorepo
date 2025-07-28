package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/cyphera/cyphera-api/libs/go/constants"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ToProductDetailResponse converts database product model to detailed API response
func ToProductDetailResponse(p db.Product, dbPrices []db.Price) responses.ProductDetailResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(p.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling product metadata: %v", err)
	}

	apiPrices := make([]responses.PriceResponse, len(dbPrices))
	for i, dbPrice := range dbPrices {
		apiPrices[i] = ToPriceResponseFromDB(dbPrice)
	}

	return responses.ProductDetailResponse{
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

// ToPublicProductResponse converts database models to public API response
func ToPublicProductResponse(workspace db.Workspace, product db.Product, price db.Price, productTokens []db.GetActiveProductTokensByProductRow, wallet db.Wallet) responses.PublicProductResponse {
	publicProductTokens := make([]responses.ProductTokenResponse, len(productTokens))
	for i, pt := range productTokens {
		publicProductTokens[i] = responses.ProductTokenResponse{
			ID:             pt.ID.String(),
			Object:         "product_token",
			ProductID:      product.ID.String(),
			ProductTokenID: pt.ID.String(), // Same as ID for product_token record
			NetworkID:      pt.NetworkID.String(),
			TokenID:        pt.TokenID.String(),
			TokenSymbol:    pt.TokenSymbol,
			Active:         true,
			Metadata:       json.RawMessage("{}"), // GetActiveProductTokensByProductRow doesn't have metadata
			CreatedAt:      pt.CreatedAt.Time.Unix(),
			UpdatedAt:      pt.UpdatedAt.Time.Unix(),
		}
	}
	return responses.PublicProductResponse{
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

// ToPriceResponseFromDB converts database price model to API response
func ToPriceResponseFromDB(p db.Price) responses.PriceResponse {
	return responses.PriceResponse{
		ID:                  p.ID.String(),
		Object:              "price",
		ProductID:           p.ProductID.String(),
		Active:              p.Active,
		Type:                string(p.Type),
		Nickname:            p.Nickname.String,
		Currency:            p.Currency,
		UnitAmountInPennies: int64(p.UnitAmountInPennies),
		IntervalType:        string(p.IntervalType),
		IntervalCount:       0, // Price doesn't have IntervalCount in db, defaulting to 0
		TermLength:          p.TermLength,
		Metadata:            p.Metadata,
		CreatedAt:           p.CreatedAt.Time.Unix(),
		UpdatedAt:           p.UpdatedAt.Time.Unix(),
	}
}

// ValidatePrice validates price parameters
func ValidatePrice(price params.CreatePriceParams) error {
	// Validate price type
	priceType, err := ValidatePriceType(price.Type)
	if err != nil {
		return err
	}

	// Validate interval type
	intervalType, err := ValidateIntervalType(price.IntervalType)
	if err != nil {
		return err
	}

	// Validate price amount
	if err := ValidatePriceInPennies(int32(price.UnitAmountInPennies)); err != nil {
		return err
	}

	// Validate term length based on price type
	if err := ValidatePriceTermLength(priceType, price.TermLength, intervalType, price.IntervalCount); err != nil {
		return err
	}

	// Validate metadata
	if err := ValidateMetadata(price.Metadata); err != nil {
		return err
	}

	return nil
}

// ValidateWalletOwnership validates that wallet belongs to the workspace
func ValidateWalletOwnership(ctx context.Context, queries db.Querier, walletID, workspaceID uuid.UUID) error {
	wallet, err := queries.GetWalletByID(ctx, db.GetWalletByIDParams{
		ID:          walletID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("wallet not found or not accessible: %w", err)
	}

	if wallet.WorkspaceID != workspaceID {
		return fmt.Errorf("wallet does not belong to workspace")
	}

	return nil
}

// CreateProductTokens creates product tokens for a product
func CreateProductTokens(ctx context.Context, queries db.Querier, productID uuid.UUID, tokens []params.CreateProductTokenParams) error {
	for _, pt := range tokens {
		// NetworkID and TokenID are already UUIDs, no need to parse

		_, err := queries.CreateProductToken(ctx, db.CreateProductTokenParams{
			ProductID: productID,
			NetworkID: pt.NetworkID,
			TokenID:   pt.TokenID,
			Active:    pt.Active,
		})
		if err != nil {
			return fmt.Errorf("failed to create product token: %w", err)
		}
	}
	return nil
}

// ValidateProductUpdate validates product update parameters
func ValidateProductUpdate(ctx context.Context, queries db.Querier, params params.UpdateProductParams, existingProduct db.Product) error {
	// Validate name if provided
	if params.Name != nil {
		if err := ValidateProductName(*params.Name); err != nil {
			return err
		}
	}

	// Validate description if provided
	if params.Description != nil {
		if err := ValidateProductDescription(*params.Description); err != nil {
			return err
		}
	}

	// Validate image URL if provided
	if params.ImageURL != nil {
		if err := ValidateImageURL(*params.ImageURL); err != nil {
			return err
		}
	}

	// Validate product URL if provided
	if params.URL != nil {
		if err := ValidateProductURL(*params.URL); err != nil {
			return err
		}
	}

	// Validate metadata if provided
	if params.Metadata != nil {
		if err := ValidateMetadata(params.Metadata); err != nil {
			return err
		}
	}

	// Validate wallet ownership if wallet is being changed
	if params.WalletID != nil && *params.WalletID != existingProduct.WalletID {
		if err := ValidateWalletOwnership(ctx, queries, *params.WalletID, existingProduct.WorkspaceID); err != nil {
			return err
		}
	}

	return nil
}

// BuildUpdateParams builds the database update parameters
func BuildUpdateParams(params params.UpdateProductParams, existingProduct db.Product) db.UpdateProductParams {
	updateParams := db.UpdateProductParams{
		ID:          params.ProductID,
		Name:        existingProduct.Name,
		Description: existingProduct.Description,
		ImageUrl:    existingProduct.ImageUrl,
		Url:         existingProduct.Url,
		Active:      existingProduct.Active,
		Metadata:    existingProduct.Metadata,
		WalletID:    existingProduct.WalletID,
	}

	// Apply updates only for provided fields
	if params.Name != nil {
		updateParams.Name = *params.Name
	}
	if params.Description != nil {
		updateParams.Description = pgtype.Text{String: *params.Description, Valid: true}
	}
	if params.ImageURL != nil {
		updateParams.ImageUrl = pgtype.Text{String: *params.ImageURL, Valid: true}
	}
	if params.URL != nil {
		updateParams.Url = pgtype.Text{String: *params.URL, Valid: true}
	}
	if params.Active != nil {
		updateParams.Active = *params.Active
	}
	if params.Metadata != nil {
		updateParams.Metadata = params.Metadata
	}
	if params.WalletID != nil {
		updateParams.WalletID = *params.WalletID
	}

	return updateParams
}

// ValidateDelegationData validates the delegation data
func ValidateDelegationData(delegation params.DelegationParams, cypheraAddress string) error {
	if delegation.Delegate != cypheraAddress {
		return fmt.Errorf("delegate address does not match cyphera smart wallet address, %s != %s", delegation.Delegate, cypheraAddress)
	}

	if delegation.Delegate == "" || delegation.Delegator == "" ||
		delegation.Authority == "" || delegation.Salt == "" ||
		delegation.Signature == "" {
		return fmt.Errorf("incomplete delegation data")
	}

	return nil
}
