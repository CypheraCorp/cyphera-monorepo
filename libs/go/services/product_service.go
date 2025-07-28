package services

import (
	"context"
	"fmt"

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

// CreateProduct creates a new product with associated prices and product tokens
func (s *ProductService) CreateProduct(ctx context.Context, createParams params.CreateProductParams) (*db.Product, []db.Price, error) {
	// Validate required fields
	if createParams.Name == "" {
		return nil, nil, fmt.Errorf("product name is required")
	}
	if createParams.WorkspaceID == uuid.Nil {
		return nil, nil, fmt.Errorf("workspace ID is required")
	}
	if createParams.WalletID == uuid.Nil {
		return nil, nil, fmt.Errorf("wallet ID is required")
	}

	// Validate product fields using helpers
	if err := helpers.ValidateProductName(createParams.Name); err != nil {
		return nil, nil, err
	}
	if err := helpers.ValidateProductDescription(createParams.Description); err != nil {
		return nil, nil, err
	}
	if err := helpers.ValidateImageURL(createParams.ImageURL); err != nil {
		return nil, nil, err
	}
	if err := helpers.ValidateProductURL(createParams.URL); err != nil {
		return nil, nil, err
	}
	if err := helpers.ValidateMetadata(createParams.Metadata); err != nil {
		return nil, nil, err
	}

	// Validate wallet ownership
	if err := s.validateWalletOwnership(ctx, createParams.WalletID, createParams.WorkspaceID); err != nil {
		return nil, nil, err
	}

	// Validate prices
	for i, price := range createParams.Prices {
		if err := s.validatePrice(price); err != nil {
			return nil, nil, fmt.Errorf("price %d validation failed: %w", i+1, err)
		}
	}

	// Create the product
	product, err := s.queries.CreateProduct(ctx, db.CreateProductParams{
		WorkspaceID: createParams.WorkspaceID,
		WalletID:    createParams.WalletID,
		Name:        createParams.Name,
		Description: pgtype.Text{String: createParams.Description, Valid: createParams.Description != ""},
		ImageUrl:    pgtype.Text{String: createParams.ImageURL, Valid: createParams.ImageURL != ""},
		Url:         pgtype.Text{String: createParams.URL, Valid: createParams.URL != ""},
		Active:      createParams.Active,
		Metadata:    createParams.Metadata,
	})
	if err != nil {
		s.logger.Error("Failed to create product",
			zap.String("name", createParams.Name),
			zap.String("workspace_id", createParams.WorkspaceID.String()),
			zap.Error(err))
		return nil, nil, fmt.Errorf("failed to create product: %w", err)
	}

	// Create associated prices
	prices := make([]db.Price, len(createParams.Prices))
	for i, priceParam := range createParams.Prices {
		dbPrice, err := s.queries.CreatePrice(ctx, db.CreatePriceParams{
			ProductID:           product.ID,
			Active:              priceParam.Active,
			Type:                db.PriceType(priceParam.Type),
			Nickname:            pgtype.Text{String: priceParam.Nickname, Valid: priceParam.Nickname != ""},
			Currency:            priceParam.Currency,
			UnitAmountInPennies: int32(priceParam.UnitAmountInPennies),
			IntervalType:        db.IntervalType(priceParam.IntervalType),
			TermLength:          priceParam.TermLength,
			Metadata:            priceParam.Metadata,
		})
		if err != nil {
			s.logger.Error("Failed to create price for product",
				zap.String("product_id", product.ID.String()),
				zap.Int("price_index", i),
				zap.Error(err))
			return nil, nil, fmt.Errorf("failed to create price %d: %w", i+1, err)
		}
		prices[i] = dbPrice
	}

	// Create product tokens if provided
	if len(createParams.ProductTokens) > 0 {
		if err := s.createProductTokens(ctx, product.ID, createParams.ProductTokens); err != nil {
			s.logger.Error("Failed to create product tokens",
				zap.String("product_id", product.ID.String()),
				zap.Error(err))
			return nil, nil, fmt.Errorf("failed to create product tokens: %w", err)
		}
	}

	s.logger.Info("Product created successfully",
		zap.String("product_id", product.ID.String()),
		zap.String("name", product.Name),
		zap.Int("prices_count", len(prices)),
		zap.Int("product_tokens_count", len(createParams.ProductTokens)))

	return &product, prices, nil
}

// GetProduct retrieves a product by ID with its associated prices
func (s *ProductService) GetProduct(ctx context.Context, getParams params.GetProductParams) (*db.Product, []db.Price, error) {
	if getParams.ProductID == uuid.Nil {
		return nil, nil, fmt.Errorf("product ID is required")
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
		return nil, nil, fmt.Errorf("product not found: %w", err)
	}

	// Validate workspace access if provided
	if getParams.WorkspaceID != uuid.Nil && product.WorkspaceID != getParams.WorkspaceID {
		return nil, nil, fmt.Errorf("product not found in workspace")
	}

	// Get associated prices
	prices, err := s.queries.ListPricesByProduct(ctx, product.ID)
	if err != nil {
		s.logger.Error("Failed to get prices for product",
			zap.String("product_id", product.ID.String()),
			zap.Error(err))
		return nil, nil, fmt.Errorf("failed to retrieve product prices: %w", err)
	}

	return &product, prices, nil
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
		// Get prices for the product
		prices, err := s.queries.ListPricesByProduct(ctx, product.ID)
		if err != nil {
			s.logger.Warn("Failed to get prices for product",
				zap.String("product_id", product.ID.String()),
				zap.Error(err))
			prices = []db.Price{} // Continue with empty prices
		}

		// Convert to response format
		productResponses[i] = helpers.ToProductDetailResponse(product, prices)
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
	if err := s.validateProductUpdate(ctx, updateParams, existingProduct); err != nil {
		return nil, err
	}

	// Build update parameters
	dbUpdateParams := s.buildUpdateParams(updateParams, existingProduct)

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

// validatePrice validates price parameters
func (s *ProductService) validatePrice(price params.CreatePriceParams) error {
	// Validate price type
	priceType, err := helpers.ValidatePriceType(price.Type)
	if err != nil {
		return err
	}

	// Validate interval type
	intervalType, err := helpers.ValidateIntervalType(price.IntervalType)
	if err != nil {
		return err
	}

	// Validate price amount
	if err := helpers.ValidatePriceInPennies(int32(price.UnitAmountInPennies)); err != nil {
		return err
	}

	// Validate term length based on price type
	if err := helpers.ValidatePriceTermLength(priceType, price.TermLength, intervalType, price.IntervalCount); err != nil {
		return err
	}

	// Validate metadata
	if err := helpers.ValidateMetadata(price.Metadata); err != nil {
		return err
	}

	return nil
}

// validateWalletOwnership validates that wallet belongs to the workspace
func (s *ProductService) validateWalletOwnership(ctx context.Context, walletID, workspaceID uuid.UUID) error {
	wallet, err := s.queries.GetWalletByID(ctx, db.GetWalletByIDParams{
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

// createProductTokens creates product tokens for a product
func (s *ProductService) createProductTokens(ctx context.Context, productID uuid.UUID, tokens []params.CreateProductTokenParams) error {
	for _, pt := range tokens {
		// NetworkID and TokenID are already UUIDs, no need to parse

		_, err := s.queries.CreateProductToken(ctx, db.CreateProductTokenParams{
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

// validateProductUpdate validates product update parameters
func (s *ProductService) validateProductUpdate(ctx context.Context, params params.UpdateProductParams, existingProduct db.Product) error {
	// Validate name if provided
	if params.Name != nil {
		if err := helpers.ValidateProductName(*params.Name); err != nil {
			return err
		}
	}

	// Validate description if provided
	if params.Description != nil {
		if err := helpers.ValidateProductDescription(*params.Description); err != nil {
			return err
		}
	}

	// Validate image URL if provided
	if params.ImageURL != nil {
		if err := helpers.ValidateImageURL(*params.ImageURL); err != nil {
			return err
		}
	}

	// Validate product URL if provided
	if params.URL != nil {
		if err := helpers.ValidateProductURL(*params.URL); err != nil {
			return err
		}
	}

	// Validate metadata if provided
	if params.Metadata != nil {
		if err := helpers.ValidateMetadata(params.Metadata); err != nil {
			return err
		}
	}

	// Validate wallet ownership if wallet is being changed
	if params.WalletID != nil && *params.WalletID != existingProduct.WalletID {
		if err := s.validateWalletOwnership(ctx, *params.WalletID, existingProduct.WorkspaceID); err != nil {
			return err
		}
	}

	return nil
}

// buildUpdateParams builds the database update parameters
func (s *ProductService) buildUpdateParams(params params.UpdateProductParams, existingProduct db.Product) db.UpdateProductParams {
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

// GetPublicProductByPriceID retrieves a product and its details for public access via price ID
func (s *ProductService) GetPublicProductByPriceID(ctx context.Context, priceID uuid.UUID) (*responses.PublicProductResponse, error) {
	if priceID == uuid.Nil {
		return nil, fmt.Errorf("price ID is required")
	}

	// Get the price
	price, err := s.queries.GetPrice(ctx, priceID)
	if err != nil {
		return nil, fmt.Errorf("price not found: %w", err)
	}

	// Check if price is active
	if !price.Active {
		return nil, fmt.Errorf("price is not active")
	}

	// Get the product
	product, err := s.queries.GetProductWithoutWorkspaceId(ctx, price.ProductID)
	if err != nil {
		return nil, fmt.Errorf("product not found for the given price: %w", err)
	}

	// Check if product is active
	if !product.Active {
		return nil, fmt.Errorf("product associated with this price is not active")
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
	response := helpers.ToPublicProductResponse(workspace, product, price, productTokens, wallet)

	// Enrich product tokens with token addresses
	for i, pt := range response.ProductTokens {
		tokenID, err := uuid.Parse(pt.TokenID)
		if err != nil {
			s.logger.Warn("Invalid token ID format in product token",
				zap.String("token_id", pt.TokenID),
				zap.String("product_token_id", pt.ProductTokenID))
			continue
		}

		token, err := s.queries.GetToken(ctx, tokenID)
		if err != nil {
			s.logger.Warn("Failed to retrieve token details",
				zap.String("token_id", pt.TokenID),
				zap.Error(err))
			continue
		}

		response.ProductTokens[i].TokenAddress = token.ContractAddress
	}

	return &response, nil
}
