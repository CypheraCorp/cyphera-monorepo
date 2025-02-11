package handlers

import (
	"cyphera-api/internal/constants"
	"cyphera-api/internal/db"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// SwaggerMetadata is used to represent JSON metadata in Swagger docs
type SwaggerMetadata map[string]interface{}

// ProductHandler handles product-related operations
type ProductHandler struct {
	common *CommonServices
}

// NewProductHandler creates a new ProductHandler instance
func NewProductHandler(common *CommonServices) *ProductHandler {
	return &ProductHandler{common: common}
}

// ProductResponse represents the standardized API response for product operations
type ProductResponse struct {
	ID              string                 `json:"id"`
	Object          string                 `json:"object"`
	WorkspaceID     string                 `json:"workspace_id"`
	WalletID        string                 `json:"wallet_id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description,omitempty"`
	ProductType     string                 `json:"product_type"`
	IntervalType    string                 `json:"interval_type,omitempty"`
	TermLength      int32                  `json:"term_length,omitempty"`
	PriceInPennies  int32                  `json:"price_in_pennies"`
	ImageURL        string                 `json:"image_url,omitempty"`
	URL             string                 `json:"url,omitempty"`
	MerchantPaidGas bool                   `json:"merchant_paid_gas"`
	Active          bool                   `json:"active"`
	Metadata        json.RawMessage        `json:"metadata,omitempty" swaggertype:"object"`
	CreatedAt       int64                  `json:"created_at"`
	UpdatedAt       int64                  `json:"updated_at"`
	ProductTokens   []ProductTokenResponse `json:"product_tokens,omitempty"`
}

// CreateProductRequest represents the request body for creating a product
type CreateProductRequest struct {
	Name            string                      `json:"name" binding:"required"`
	WalletID        string                      `json:"wallet_id" binding:"required"`
	Description     string                      `json:"description"`
	ProductType     string                      `json:"product_type" binding:"required"`
	IntervalType    string                      `json:"interval_type"`
	TermLength      int32                       `json:"term_length"`
	PriceInPennies  int32                       `json:"price_in_pennies" binding:"required"`
	ImageURL        string                      `json:"image_url"`
	URL             string                      `json:"url"`
	MerchantPaidGas bool                        `json:"merchant_paid_gas"`
	Active          bool                        `json:"active"`
	Metadata        json.RawMessage             `json:"metadata" swaggertype:"object"`
	ProductTokens   []CreateProductTokenRequest `json:"product_tokens,omitempty"`
}

// UpdateProductRequest represents the request body for updating a product
type UpdateProductRequest struct {
	Name            string                      `json:"name,omitempty"`
	WalletID        string                      `json:"wallet_id,omitempty"`
	Description     string                      `json:"description,omitempty"`
	ProductType     string                      `json:"product_type,omitempty"`
	IntervalType    string                      `json:"interval_type,omitempty"`
	TermLength      *int32                      `json:"term_length,omitempty"`
	PriceInPennies  *int32                      `json:"price_in_pennies,omitempty"`
	ImageURL        string                      `json:"image_url,omitempty"`
	URL             string                      `json:"url,omitempty"`
	MerchantPaidGas *bool                       `json:"merchant_paid_gas,omitempty"`
	Active          *bool                       `json:"active,omitempty"`
	Metadata        json.RawMessage             `json:"metadata,omitempty" swaggertype:"object"`
	ProductTokens   []CreateProductTokenRequest `json:"product_tokens,omitempty"`
}

// ListProductsResponse represents the paginated response for product list operations
type ListProductsResponse struct {
	Object  string            `json:"object"`
	Data    []ProductResponse `json:"data"`
	HasMore bool              `json:"has_more"`
	Total   int64             `json:"total"`
}

// GetProduct godoc
// @Summary Get product by ID
// @Description Get product details by product ID
// @Tags products
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Success 200 {object} ProductResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/{product_id} [get]
func (h *ProductHandler) GetProduct(c *gin.Context) {
	productId := c.Param("product_id")
	parsedUUID, err := uuid.Parse(productId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid product ID format", err)
		return
	}

	product, err := h.common.db.GetProduct(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Product not found")
		return
	}

	sendSuccess(c, http.StatusOK, toProductResponse(product))
}

// GetPublicProductByID godoc
// @Summary Get public product by ID
// @Description Get public product details by product ID
// @Tags products
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Success 200 {object} ProductResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/public/{product_id} [get]
func (h *ProductHandler) GetPublicProductByID(c *gin.Context) {
	productId := c.Param("product_id")
	parsedUUID, err := uuid.Parse(productId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid product ID format", err)
		return
	}

	product, err := h.common.db.GetProduct(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Product not found")
		return
	}

	wallet, err := h.common.db.GetWalletByID(c.Request.Context(), product.WalletID)
	if err != nil {
		handleDBError(c, err, "Wallet not found")
		return
	}

	workspace, err := h.common.db.GetWorkspace(c.Request.Context(), product.WorkspaceID)
	if err != nil {
		handleDBError(c, err, "Workspace not found")
		return
	}

	productTokens, err := h.common.db.GetActiveProductTokensByProduct(c.Request.Context(), parsedUUID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve product tokens", err)
		return
	}

	sendSuccess(c, http.StatusOK, toPublicProductResponse(workspace, product, productTokens, wallet))
}

func toPublicProductResponse(workspace db.Workspace, product db.Product, productTokens []db.GetActiveProductTokensByProductRow, wallet db.Wallet) PublicProductResponse {
	// Convert product tokens to public response format
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
		}
	}

	return PublicProductResponse{
		ProductID:       product.ID.String(),
		AccountID:       workspace.AccountID.String(),
		WorkspaceID:     workspace.ID.String(),
		WalletAddress:   wallet.WalletAddress,
		Name:            product.Name,
		Description:     product.Description.String,
		ProductType:     string(product.ProductType),
		IntervalType:    string(product.IntervalType),
		TermLength:      product.TermLength.Int32,
		PriceInPennies:  product.PriceInPennies,
		ImageURL:        product.ImageUrl.String,
		MerchantPaidGas: product.MerchantPaidGas,
		ProductTokens:   publicProductTokens,
	}
}

type PublicProductResponse struct {
	ProductID       string                       `json:"product_id"`
	AccountID       string                       `json:"account_id"`
	WorkspaceID     string                       `json:"workspace_id"`
	WalletAddress   string                       `json:"wallet_address"`
	Name            string                       `json:"name"`
	Description     string                       `json:"description"`
	ProductType     string                       `json:"product_type"`
	IntervalType    string                       `json:"interval_type,omitempty"`
	TermLength      int32                        `json:"term_length,omitempty"`
	PriceInPennies  int32                        `json:"price_in_pennies"`
	ImageURL        string                       `json:"image_url,omitempty"`
	MerchantPaidGas bool                         `json:"merchant_paid_gas"`
	ProductTokens   []PublicProductTokenResponse `json:"product_tokens,omitempty"`
}

type PublicProductTokenResponse struct {
	ProductTokenID string `json:"product_token_id"`
	NetworkID      string `json:"network_id"`
	NetworkName    string `json:"network_name"`
	NetworkChainID string `json:"network_chain_id"`
	TokenID        string `json:"token_id"`
	TokenName      string `json:"token_name"`
	TokenSymbol    string `json:"token_symbol"`
	TokenImageURL  string `json:"token_image_url"`
}

// validateProductType validates the product type and returns a db.ProductType if valid
func validateProductType(productType string) (db.ProductType, error) {
	if productType == "" {
		return "", nil
	}
	if productType != constants.ProductTypeRecurring && productType != constants.ProductTypeOneOff {
		return "", fmt.Errorf("invalid product type. Must be '%s' or '%s'", constants.ProductTypeRecurring, constants.ProductTypeOneOff)
	}
	return db.ProductType(productType), nil
}

// validateIntervalType validates the interval type and returns a db.IntervalType if valid
func validateIntervalType(intervalType string) (db.IntervalType, error) {
	if intervalType == "" {
		return "", nil
	}

	validIntervalTypes := map[string]bool{
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

// validateWalletID validates and parses the wallet ID
func validateWalletID(walletID string) (uuid.UUID, error) {
	if walletID == "" {
		return uuid.Nil, nil
	}
	parsed, err := uuid.Parse(walletID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid wallet ID format: %w", err)
	}
	return parsed, nil
}

// updateTextFields updates text-based fields in the product parameters
func updateTextFields(params *db.UpdateProductParams, req UpdateProductRequest) {
	if req.Name != "" {
		params.Name = req.Name
	}
	if req.Description != "" {
		params.Description = pgtype.Text{String: req.Description, Valid: true}
	}
	if req.ImageURL != "" {
		params.ImageUrl = pgtype.Text{String: req.ImageURL, Valid: true}
	}
	if req.URL != "" {
		params.Url = pgtype.Text{String: req.URL, Valid: true}
	}
}

// updateNumericFields updates numeric fields in the product parameters
func updateNumericFields(params *db.UpdateProductParams, req UpdateProductRequest) {
	if req.TermLength != nil {
		params.TermLength = pgtype.Int4{Int32: *req.TermLength, Valid: true}
	}
	if req.PriceInPennies != nil {
		params.PriceInPennies = *req.PriceInPennies
	}
}

// updateBooleanFields updates boolean fields in the product parameters
func updateBooleanFields(params *db.UpdateProductParams, req UpdateProductRequest) {
	if req.MerchantPaidGas != nil {
		params.MerchantPaidGas = *req.MerchantPaidGas
	}
	if req.Active != nil {
		params.Active = *req.Active
	}
}

// updateProductParams creates the update parameters for a product
func (h *ProductHandler) updateProductParams(id uuid.UUID, req UpdateProductRequest) (db.UpdateProductParams, error) {
	params := db.UpdateProductParams{
		ID: id,
	}

	// Update basic text fields
	updateTextFields(&params, req)

	// Update product type if provided
	if req.ProductType != "" {
		productType, err := validateProductType(req.ProductType)
		if err != nil {
			return params, err
		}
		params.ProductType = productType
	}

	// Update interval type if provided
	if req.IntervalType != "" {
		intervalType, err := validateIntervalType(req.IntervalType)
		if err != nil {
			return params, err
		}
		params.IntervalType = intervalType
	}

	// Update numeric fields
	updateNumericFields(&params, req)

	// Update boolean fields
	updateBooleanFields(&params, req)

	// Update metadata if provided
	if req.Metadata != nil {
		params.Metadata = req.Metadata
	}

	// update wallet id if provided
	if req.WalletID != "" {
		parsedWalletID, err := uuid.Parse(req.WalletID)
		if err != nil {
			return params, fmt.Errorf("invalid wallet ID format: %w", err)
		}
		params.WalletID = parsedWalletID
	}
	return params, nil
}

// validatePaginationParams validates and returns pagination parameters
func validatePaginationParams(c *gin.Context) (limit int32, offset int32, err error) {
	const maxLimit int32 = 100
	limit = 10 // default limit

	if limitStr := c.Query("limit"); limitStr != "" {
		parsedLimit, err := strconv.ParseInt(limitStr, 10, 32)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid limit parameter")
		}
		if parsedLimit > int64(maxLimit) {
			limit = maxLimit
		} else if parsedLimit > 0 {
			limit = int32(parsedLimit)
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		parsedOffset, err := strconv.ParseInt(offsetStr, 10, 32)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid offset parameter")
		}
		if parsedOffset > 0 {
			offset = int32(parsedOffset)
		}
	}

	return limit, offset, nil
}

// validateProductUpdate validates all update parameters at once
func (h *ProductHandler) validateProductUpdate(c *gin.Context, req UpdateProductRequest, existingProduct db.Product) error {
	// Validate product type if provided
	if req.ProductType != "" {
		if _, err := validateProductType(req.ProductType); err != nil {
			return err
		}
	}

	// Validate interval type if provided
	if req.IntervalType != "" {
		if _, err := validateIntervalType(req.IntervalType); err != nil {
			return err
		}
	}

	// Validate term length if provided
	if req.TermLength != nil {
		if err := validateTermLength(string(existingProduct.ProductType), *req.TermLength); err != nil {
			return err
		}
	}

	// Validate price if provided
	if req.PriceInPennies != nil {
		if err := validatePriceInPennies(*req.PriceInPennies); err != nil {
			return err
		}
	}

	// Validate wallet if provided
	if req.WalletID != "" {
		parsedWalletID, err := uuid.Parse(req.WalletID)
		if err != nil {
			return fmt.Errorf("invalid wallet ID format: %w", err)
		}

		if err := h.validateWallet(c, parsedWalletID, existingProduct.WorkspaceID); err != nil {
			return err
		}
	}

	return nil
}

// ListProducts godoc
// @Summary List products
// @Description Retrieves paginated products for a workspace
// @Tags products
// @Accept json
// @Produce json
// @Param limit query int false "Number of products to return (default 10, max 100)"
// @Param offset query int false "Number of products to skip (default 0)"
// @Success 200 {object} ListProductsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products [get]
func (h *ProductHandler) ListProducts(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	// Get pagination parameters
	limit, offset, err := validatePaginationParams(c)
	if err != nil {
		sendError(c, http.StatusBadRequest, err.Error(), err)
		return
	}

	// Get total count
	total, err := h.common.db.CountProducts(c.Request.Context(), parsedWorkspaceID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to count products", err)
		return
	}

	// Get paginated products
	products, err := h.common.db.ListProductsWithPagination(c.Request.Context(), db.ListProductsWithPaginationParams{
		WorkspaceID: parsedWorkspaceID,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve products", err)
		return
	}

	responseList := make([]ProductResponse, len(products))
	// for each product, get the active product tokens
	for i, product := range products {
		productResponse := toProductResponse(product)
		productTokenList, err := h.common.db.GetActiveProductTokensByProduct(c.Request.Context(), product.ID)
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to retrieve product tokens", err)
			return
		}
		productTokenListResponse := make([]ProductTokenResponse, len(productTokenList))
		for j, productToken := range productTokenList {
			productTokenListResponse[j] = toActiveProductTokenByProductResponse(productToken)
		}
		productResponse.ProductTokens = productTokenListResponse
		responseList[i] = productResponse
	}

	// Calculate hasMore safely without integer overflow risk
	var hasMore bool
	if total > 0 {
		hasMore = (int64(offset) + int64(limit)) < total
	}

	listProductsResponse := ListProductsResponse{
		Object:  "list",
		Data:    responseList,
		HasMore: hasMore,
		Total:   total,
	}

	sendSuccess(c, http.StatusOK, listProductsResponse)
}

// ListActiveProducts godoc
// @Summary List active products
// @Description Retrieves all active products for a workspace
// @Tags products
// @Accept json
// @Produce json
// @Success 200 {object} ListProductsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/active [get]
func (h *ProductHandler) ListActiveProducts(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	products, err := h.common.db.ListActiveProducts(c.Request.Context(), parsedWorkspaceID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve active products", err)
		return
	}

	responseList := make([]ProductResponse, len(products))
	for i, product := range products {
		responseList[i] = toProductResponse(product)
	}

	response := ListProductsResponse{
		Object:  "list",
		Data:    responseList,
		HasMore: false,
		Total:   int64(len(products)),
	}

	sendSuccess(c, http.StatusOK, response)
}

// validateTermLength validates the term length for recurring products
func validateTermLength(productType string, termLength int32) error {
	if productType == constants.ProductTypeRecurring && termLength <= 0 {
		return fmt.Errorf("term length must be greater than 0 for recurring products")
	}
	return nil
}

// validateWallet validates the wallet exists and belongs to the workspace's account
func (h *ProductHandler) validateWallet(ctx *gin.Context, walletID uuid.UUID, workspaceID uuid.UUID) error {
	wallet, err := h.common.db.GetWalletByID(ctx.Request.Context(), walletID)
	if err != nil {
		return fmt.Errorf("failed to get wallet: %w", err)
	}

	workspace, err := h.common.db.GetWorkspace(ctx.Request.Context(), workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	if wallet.AccountID != workspace.AccountID {
		return fmt.Errorf("wallet does not belong to account")
	}

	return nil
}

// createProductTokens creates the associated product tokens for a product
func (h *ProductHandler) createProductTokens(c *gin.Context, productID uuid.UUID, tokens []CreateProductTokenRequest) error {
	for _, pt := range tokens {
		networkID, err := uuid.Parse(pt.NetworkID)
		if err != nil {
			return fmt.Errorf("invalid network ID format: %w", err)
		}

		tokenID, err := uuid.Parse(pt.TokenID)
		if err != nil {
			return fmt.Errorf("invalid token ID format: %w", err)
		}

		_, err = h.common.db.CreateProductToken(c.Request.Context(), db.CreateProductTokenParams{
			ProductID: productID,
			NetworkID: networkID,
			TokenID:   tokenID,
			Active:    pt.Active,
		})
		if err != nil {
			return fmt.Errorf("failed to create product token: %w", err)
		}
	}
	return nil
}

// validatePriceInPennies validates that the price value is within int32 bounds
func validatePriceInPennies(price int32) error {
	if price < 0 {
		return fmt.Errorf("price_in_pennies cannot be negative")
	}
	return nil
}

// CreateProduct godoc
// @Summary Create product
// @Description Creates a new product
// @Tags products
// @Accept json
// @Produce json
// @Param product body CreateProductRequest true "Product creation data"
// @Success 201 {object} ProductResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products [post]
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	// Validate product type
	productType, err := validateProductType(req.ProductType)
	if err != nil {
		sendError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Validate interval type
	intervalType, err := validateIntervalType(req.IntervalType)
	if err != nil {
		sendError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Validate wallet ID
	parsedWalletID, err := validateWalletID(req.WalletID)
	if err != nil {
		sendError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Validate term length
	if err := validateTermLength(req.ProductType, req.TermLength); err != nil {
		sendError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Validate price
	if err := validatePriceInPennies(req.PriceInPennies); err != nil {
		sendError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Validate wallet belongs to workspace
	if err := h.validateWallet(c, parsedWalletID, parsedWorkspaceID); err != nil {
		sendError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Create product
	product, err := h.common.db.CreateProduct(c.Request.Context(), db.CreateProductParams{
		WorkspaceID:     parsedWorkspaceID,
		WalletID:        parsedWalletID,
		Name:            req.Name,
		Description:     pgtype.Text{String: req.Description, Valid: req.Description != ""},
		ProductType:     productType,
		IntervalType:    intervalType,
		TermLength:      pgtype.Int4{Int32: req.TermLength, Valid: req.TermLength != 0},
		PriceInPennies:  req.PriceInPennies,
		ImageUrl:        pgtype.Text{String: req.ImageURL, Valid: req.ImageURL != ""},
		Url:             pgtype.Text{String: req.URL, Valid: req.URL != ""},
		MerchantPaidGas: req.MerchantPaidGas,
		Active:          req.Active,
		Metadata:        req.Metadata,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create product", err)
		return
	}

	// Create product tokens if provided
	if len(req.ProductTokens) > 0 {
		if err := h.createProductTokens(c, product.ID, req.ProductTokens); err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to create product tokens", err)
			return
		}
	}

	sendSuccess(c, http.StatusCreated, toProductResponse(product))
}

// UpdateProduct godoc
// @Summary Update product
// @Description Updates an existing product
// @Tags products
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param product body UpdateProductRequest true "Product update data"
// @Success 200 {object} ProductResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/{product_id} [put]
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	// Parse product ID
	productId := c.Param("product_id")
	parsedUUID, err := uuid.Parse(productId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid product ID format", err)
		return
	}

	// Parse request body
	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Get existing product to verify workspace ownership
	existingProduct, err := h.common.db.GetProduct(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Product not found")
		return
	}

	// Verify workspace ownership
	workspaceID := c.GetHeader("X-Workspace-ID")
	if existingProduct.WorkspaceID.String() != workspaceID {
		sendError(c, http.StatusForbidden, "Product does not belong to workspace", nil)
		return
	}

	// Validate all update parameters
	if err := h.validateProductUpdate(c, req, existingProduct); err != nil {
		sendError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Create update parameters
	params, err := h.updateProductParams(parsedUUID, req)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid update parameters", err)
		return
	}

	// Update the product
	product, err := h.common.db.UpdateProduct(c.Request.Context(), params)
	if err != nil {
		handleDBError(c, err, "Failed to update product")
		return
	}

	// Update product tokens if provided
	if len(req.ProductTokens) > 0 {
		// First, delete all existing product tokens
		err = h.common.db.DeleteProductTokensByProduct(c.Request.Context(), product.ID)
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to delete existing product tokens", err)
			return
		}

		// Create new product tokens
		if err := h.createProductTokens(c, product.ID, req.ProductTokens); err != nil {
			sendError(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
	}

	sendSuccess(c, http.StatusOK, toProductResponse(product))
}

// DeleteProduct godoc
// @Summary Delete product
// @Description Deletes a product
// @Tags products
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/{product_id} [delete]
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	productId := c.Param("product_id")
	parsedUUID, err := uuid.Parse(productId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid product ID format", err)
		return
	}

	err = h.common.db.DeleteProduct(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Failed to delete product")
		return
	}

	sendSuccess(c, http.StatusNoContent, nil)
}

// Helper function to convert database model to API response
func toProductResponse(p db.Product) ProductResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(p.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling product metadata: %v", err)
		metadata = make(map[string]interface{}) // Use empty map if unmarshal fails
	}

	return ProductResponse{
		ID:              p.ID.String(),
		Object:          "product",
		WorkspaceID:     p.WorkspaceID.String(),
		WalletID:        p.WalletID.String(),
		Name:            p.Name,
		Description:     p.Description.String,
		ProductType:     string(p.ProductType),
		IntervalType:    string(p.IntervalType),
		TermLength:      p.TermLength.Int32,
		PriceInPennies:  p.PriceInPennies,
		ImageURL:        p.ImageUrl.String,
		URL:             p.Url.String,
		MerchantPaidGas: p.MerchantPaidGas,
		Active:          p.Active,
		Metadata:        p.Metadata,
		CreatedAt:       p.CreatedAt.Time.Unix(),
		UpdatedAt:       p.UpdatedAt.Time.Unix(),
	}
}
