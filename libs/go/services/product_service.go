package services

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// ProductService handles business logic for product operations
type ProductService struct {
	queries db.Querier
	logger  *zap.Logger
}

// NewProductService creates a new product service
func NewProductService(queries db.Querier) *ProductService {
	return &ProductService{
		queries: queries,
		logger:  logger.Log,
	}
}

// CreateProduct creates a new product with embedded pricing and product tokens
func (s *ProductService) CreateProduct(ctx context.Context, createParams params.CreateProductParams) (*db.Product, error) {
	// Validate required fields
	if createParams.Name == "" {
		return nil, fmt.Errorf("product name is required")
	}
	if createParams.WorkspaceID == uuid.Nil {
		return nil, fmt.Errorf("workspace ID is required")
	}
	if createParams.WalletID == uuid.Nil {
		return nil, fmt.Errorf("wallet ID is required")
	}
	if createParams.PriceType == "" {
		return nil, fmt.Errorf("price type is required")
	}
	if createParams.Currency == "" {
		return nil, fmt.Errorf("currency is required")
	}

	// Validate product fields using helpers
	if err := helpers.ValidateProductName(createParams.Name); err != nil {
		return nil, err
	}
	if err := helpers.ValidateProductDescription(createParams.Description); err != nil {
		return nil, err
	}
	if err := helpers.ValidateImageURL(createParams.ImageURL); err != nil {
		return nil, err
	}
	if err := helpers.ValidateProductURL(createParams.URL); err != nil {
		return nil, err
	}
	if err := helpers.ValidateMetadata(createParams.Metadata); err != nil {
		return nil, err
	}

	// Validate pricing fields
	if err := helpers.ValidateProductPricing(createParams.PriceType, createParams.IntervalType, createParams.UnitAmountInPennies, createParams.TermLength); err != nil {
		return nil, err
	}

	// Validate wallet ownership
	if err := helpers.ValidateWalletOwnership(ctx, s.queries, createParams.WalletID, createParams.WorkspaceID); err != nil {
		return nil, err
	}

	// Prepare interval type for nullable field
	var intervalType db.NullIntervalType
	if createParams.IntervalType != "" {
		intervalType = db.NullIntervalType{
			IntervalType: db.IntervalType(createParams.IntervalType),
			Valid:        true,
		}
	}

	// Create the product with embedded pricing
	product, err := s.queries.CreateProduct(ctx, db.CreateProductParams{
		WorkspaceID:         createParams.WorkspaceID,
		WalletID:            createParams.WalletID,
		Name:                createParams.Name,
		Description:         pgtype.Text{String: createParams.Description, Valid: createParams.Description != ""},
		ImageUrl:            pgtype.Text{String: createParams.ImageURL, Valid: createParams.ImageURL != ""},
		Url:                 pgtype.Text{String: createParams.URL, Valid: createParams.URL != ""},
		Active:              createParams.Active,
		ProductType:         pgtype.Text{String: createParams.ProductType, Valid: createParams.ProductType != ""},
		ProductGroup:        pgtype.Text{String: createParams.ProductGroup, Valid: createParams.ProductGroup != ""},
		PriceType:           db.PriceType(createParams.PriceType),
		Currency:            createParams.Currency,
		UnitAmountInPennies: createParams.UnitAmountInPennies,
		IntervalType:        intervalType,
		TermLength:          pgtype.Int4{Int32: createParams.TermLength, Valid: createParams.TermLength > 0},
		PriceNickname:       pgtype.Text{String: createParams.PriceNickname, Valid: createParams.PriceNickname != ""},
		PriceExternalID:     pgtype.Text{String: createParams.PriceExternalID, Valid: createParams.PriceExternalID != ""},
		Metadata:            createParams.Metadata,
		PaymentProvider:     pgtype.Text{String: createParams.PaymentProvider, Valid: createParams.PaymentProvider != ""},
	})
	if err != nil {
		s.logger.Error("Failed to create product",
			zap.String("name", createParams.Name),
			zap.String("workspace_id", createParams.WorkspaceID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	// Create product tokens if provided
	if len(createParams.ProductTokens) > 0 {
		if err := helpers.CreateProductTokens(ctx, s.queries, product.ID, createParams.ProductTokens); err != nil {
			s.logger.Error("Failed to create product tokens",
				zap.String("product_id", product.ID.String()),
				zap.Error(err))
			return nil, fmt.Errorf("failed to create product tokens: %w", err)
		}
	}

	s.logger.Info("Product created successfully",
		zap.String("product_id", product.ID.String()),
		zap.String("name", product.Name),
		zap.String("price_type", string(product.PriceType)),
		zap.Int32("amount_in_pennies", product.UnitAmountInPennies),
		zap.Int("product_tokens_count", len(createParams.ProductTokens)))

	return &product, nil
}

// GetProduct retrieves a product by ID
func (s *ProductService) GetProduct(ctx context.Context, getParams params.GetProductParams) (*db.Product, error) {
	if getParams.ProductID == uuid.Nil {
		return nil, fmt.Errorf("product ID is required")
	}

	// Get the product
	product, err := s.queries.GetProduct(ctx, db.GetProductParams{
		ID:          getParams.ProductID,
		WorkspaceID: getParams.WorkspaceID,
	})
	if err != nil {
		s.logger.Error("Failed to get product",
			zap.String("product_id", getParams.ProductID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("product not found: %w", err)
	}

	// Validate workspace access if provided
	if getParams.WorkspaceID != uuid.Nil && product.WorkspaceID != getParams.WorkspaceID {
		return nil, fmt.Errorf("product not found in workspace")
	}

	return &product, nil
}

// ListProducts retrieves a paginated list of products for a workspace
func (s *ProductService) ListProducts(ctx context.Context, listParams params.ListProductsParams) (*responses.ListProductsResult, error) {
	if listParams.WorkspaceID == uuid.Nil {
		return nil, fmt.Errorf("workspace ID is required")
	}

	// Set default pagination
	if listParams.Limit <= 0 {
		listParams.Limit = 10
	}
	if listParams.Limit > 100 {
		listParams.Limit = 100
	}

	// Get products with pagination
	products, err := s.queries.ListProductsWithPagination(ctx, db.ListProductsWithPaginationParams{
		WorkspaceID: listParams.WorkspaceID,
		Limit:       listParams.Limit,
		Offset:      listParams.Offset,
	})

	if err != nil {
		s.logger.Error("Failed to list products",
			zap.String("workspace_id", listParams.WorkspaceID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve products: %w", err)
	}

	// Get total count
	total, err := s.queries.CountProducts(ctx, listParams.WorkspaceID)

	if err != nil {
		s.logger.Error("Failed to count products",
			zap.String("workspace_id", listParams.WorkspaceID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to count products: %w", err)
	}

	hasMore := int64(listParams.Offset+listParams.Limit) < total

	// Convert products to ProductDetailResponse
	productResponses := make([]responses.ProductDetailResponse, len(products))
	for i, product := range products {
		// Convert to response format (pricing is now embedded in product)
		productResponses[i] = helpers.ToProductDetailResponse(product)
	}

	return &responses.ListProductsResult{
		Products: productResponses,
		Total:    total,
		HasMore:  hasMore,
	}, nil
}

// UpdateProduct updates an existing product
func (s *ProductService) UpdateProduct(ctx context.Context, updateParams params.UpdateProductParams) (*db.Product, error) {
	if updateParams.ProductID == uuid.Nil {
		return nil, fmt.Errorf("product ID is required")
	}

	// Get existing product
	existingProduct, err := s.queries.GetProduct(ctx, db.GetProductParams{
		ID:          updateParams.ProductID,
		WorkspaceID: updateParams.WorkspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	// Validate workspace access
	if updateParams.WorkspaceID != uuid.Nil && existingProduct.WorkspaceID != updateParams.WorkspaceID {
		return nil, fmt.Errorf("product not found in workspace")
	}

	// Validate updates
	if err := helpers.ValidateProductUpdate(ctx, s.queries, updateParams, existingProduct); err != nil {
		return nil, err
	}

	// Build update parameters
	dbUpdateParams := helpers.BuildUpdateParams(updateParams, existingProduct)

	// Update the product
	updatedProduct, err := s.queries.UpdateProduct(ctx, dbUpdateParams)
	if err != nil {
		s.logger.Error("Failed to update product",
			zap.String("product_id", updateParams.ProductID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	s.logger.Info("Product updated successfully",
		zap.String("product_id", updatedProduct.ID.String()))

	return &updatedProduct, nil
}

// DeleteProduct deletes a product
func (s *ProductService) DeleteProduct(ctx context.Context, productID uuid.UUID, workspaceID uuid.UUID) error {
	if productID == uuid.Nil {
		return fmt.Errorf("product ID is required")
	}

	// Get existing product to validate ownership
	product, err := s.queries.GetProduct(ctx, db.GetProductParams{
		ID:          productID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("product not found: %w", err)
	}

	// Validate workspace access
	if workspaceID != uuid.Nil && product.WorkspaceID != workspaceID {
		return fmt.Errorf("product not found in workspace")
	}

	// TODO: Add check for active subscriptions when the method is available
	// For now, allow deletion without checking subscriptions

	// Delete the product (soft delete)
	err = s.queries.DeleteProduct(ctx, db.DeleteProductParams{
		ID:          productID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		s.logger.Error("Failed to delete product",
			zap.String("product_id", productID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to delete product: %w", err)
	}

	s.logger.Info("Product deleted successfully",
		zap.String("product_id", productID.String()))

	return nil
}

// GetPublicProductByPriceID retrieves a product and its details for public access via price ID
// Deprecated: Use GetPublicProductByID instead. Price IDs are now product IDs.
func (s *ProductService) GetPublicProductByPriceID(ctx context.Context, priceID uuid.UUID) (*responses.PublicProductResponse, error) {
	// Since prices are now embedded in products, price ID is actually product ID
	return s.GetPublicProductByID(ctx, priceID)
}

// GetPublicProductByID retrieves a product and its details for public access
func (s *ProductService) GetPublicProductByID(ctx context.Context, productID uuid.UUID) (*responses.PublicProductResponse, error) {
	if productID == uuid.Nil {
		return nil, fmt.Errorf("product ID is required")
	}

	// Get the product
	product, err := s.queries.GetProductWithoutWorkspaceId(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	// Check if product is active
	if !product.Active {
		return nil, fmt.Errorf("product is not active")
	}

	// Get the wallet
	wallet, err := s.queries.GetWalletByID(ctx, db.GetWalletByIDParams{
		ID:          product.WalletID,
		WorkspaceID: product.WorkspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("wallet not found for the product: %w", err)
	}

	// Get the workspace
	workspace, err := s.queries.GetWorkspace(ctx, product.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("workspace not found for the product: %w", err)
	}

	// Get product tokens
	productTokens, err := s.queries.GetActiveProductTokensByProduct(ctx, product.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve product tokens: %w", err)
	}

	// Build response
	response := helpers.ToPublicProductResponse(workspace, product, productTokens, wallet)

	// Enrich product tokens with complete token and network details
	for i, pt := range response.ProductTokens {
		tokenID, err := uuid.Parse(pt.TokenID)
		if err != nil {
			s.logger.Warn("Invalid token ID format in product token",
				zap.String("token_id", pt.TokenID),
				zap.String("product_token_id", pt.ProductTokenID))
			continue
		}

		// Get token details
		token, err := s.queries.GetToken(ctx, tokenID)
		if err != nil {
			s.logger.Warn("Failed to retrieve token details",
				zap.String("token_id", pt.TokenID),
				zap.Error(err))
			continue
		}

		// Get network details
		network, err := s.queries.GetNetwork(ctx, token.NetworkID)
		if err != nil {
			s.logger.Warn("Failed to retrieve network details",
				zap.String("network_id", token.NetworkID.String()),
				zap.Error(err))
			continue
		}

		// Populate all the missing fields that the frontend needs
		response.ProductTokens[i].TokenAddress = token.ContractAddress
		response.ProductTokens[i].ContractAddress = token.ContractAddress
		response.ProductTokens[i].TokenDecimals = token.Decimals
		response.ProductTokens[i].TokenName = token.Name
		response.ProductTokens[i].GasToken = token.GasToken
		response.ProductTokens[i].NetworkName = network.DisplayName.String
		response.ProductTokens[i].NetworkType = string(network.NetworkType)
		response.ProductTokens[i].ChainID = network.ChainID
	}

	return &response, nil
}

// ValidateSubscriptionRequest validates the subscription request parameters
func (s *ProductService) ValidateSubscriptionRequest(ctx context.Context, params params.ValidateSubscriptionParams) error {
	// Validate basic fields
	if params.SubscriberAddress == "" {
		return fmt.Errorf("subscriber address is required")
	}

	if params.ProductID == "" {
		return fmt.Errorf("product ID is required")
	}

	if _, err := uuid.Parse(params.ProductID); err != nil {
		return fmt.Errorf("invalid product ID format")
	}

	if _, err := uuid.Parse(params.ProductTokenID); err != nil {
		return fmt.Errorf("invalid product token ID format")
	}

	tokenAmount, err := strconv.ParseInt(params.TokenAmount, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid token amount format")
	}

	if tokenAmount <= 0 {
		return fmt.Errorf("token amount must be greater than zero")
	}

	// Validate delegation data
	if err := helpers.ValidateDelegationData(params.Delegation, params.CypheraSmartWalletAddress); err != nil {
		return fmt.Errorf("delegation validation failed: %w", err)
	}

	return nil
}

// ValidateProductForSubscription validates that a product is valid for subscription
func (s *ProductService) ValidateProductForSubscription(ctx context.Context, productID uuid.UUID) (*db.Product, error) {
	product, err := s.queries.GetProductWithoutWorkspaceId(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	if !product.Active {
		return nil, fmt.Errorf("product is not active")
	}

	return &product, nil
}

// ValidatePriceForSubscription validates that a product (formerly price) is valid for subscription
// Deprecated: Use ValidateProductForSubscription directly
func (s *ProductService) ValidatePriceForSubscription(ctx context.Context, priceID uuid.UUID) (*db.Product, *db.Product, error) {
	// Since prices are now embedded in products, price ID is actually product ID
	product, err := s.ValidateProductForSubscription(ctx, priceID)
	if err != nil {
		return nil, nil, err
	}

	// Return product twice for backward compatibility
	return product, product, nil
}

// GetProductTokenWithValidation retrieves and validates a product token
func (s *ProductService) GetProductTokenWithValidation(ctx context.Context, productTokenID uuid.UUID, productID uuid.UUID) (*db.GetProductTokenRow, error) {
	productToken, err := s.queries.GetProductToken(ctx, productTokenID)
	if err != nil {
		return nil, fmt.Errorf("product token not found: %w", err)
	}

	if productToken.ProductID != productID {
		return nil, fmt.Errorf("product token does not belong to the specified product")
	}

	if !productToken.Active {
		return nil, fmt.Errorf("product token is not active")
	}

	// Verify the token exists and is active
	token, err := s.queries.GetToken(ctx, productToken.TokenID)
	if err != nil {
		return nil, fmt.Errorf("token not found: %w", err)
	}

	if !token.Active {
		return nil, fmt.Errorf("token is not active")
	}

	return &productToken, nil
}
