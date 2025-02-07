package handlers

import (
	"cyphera-api/internal/db"
	"encoding/json"
	"fmt"
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
	WorkspaceID     string                      `json:"workspace_id"`
	Name            string                      `json:"name" binding:"required"`
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
// @Summary Get a product
// @Description Retrieves the details of an existing product
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid product ID format"})
		return
	}

	product, err := h.common.db.GetProduct(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Product not found"})
		return
	}

	c.JSON(http.StatusOK, toProductResponse(product))
}

// updateProductParams creates the update parameters for a product
func (h *ProductHandler) updateProductParams(id uuid.UUID, req UpdateProductRequest) (db.UpdateProductParams, error) {
	params := db.UpdateProductParams{
		ID: id,
	}

	if req.Name != "" {
		params.Name = req.Name
	}
	if req.Description != "" {
		params.Description = pgtype.Text{String: req.Description, Valid: true}
	}
	if req.ProductType != "" {
		params.ProductType = db.ProductType(req.ProductType)
	}
	if req.IntervalType != "" {
		params.IntervalType = db.NullIntervalType{IntervalType: db.IntervalType(req.IntervalType), Valid: true}
	}
	if req.TermLength != nil {
		params.TermLength = pgtype.Int4{Int32: *req.TermLength, Valid: true}
	}
	if req.PriceInPennies != nil {
		params.PriceInPennies = *req.PriceInPennies
	}
	if req.ImageURL != "" {
		params.ImageUrl = pgtype.Text{String: req.ImageURL, Valid: true}
	}
	if req.URL != "" {
		params.Url = pgtype.Text{String: req.URL, Valid: true}
	}
	if req.MerchantPaidGas != nil {
		params.MerchantPaidGas = *req.MerchantPaidGas
	}
	if req.Active != nil {
		params.Active = *req.Active
	}
	if req.Metadata != nil {
		metadata, err := json.Marshal(req.Metadata)
		if err != nil {
			return params, fmt.Errorf("invalid metadata format: %w", err)
		}
		params.Metadata = metadata
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

// ListProducts godoc
// @Summary List products
// @Description Returns a paginated list of all products for a workspace
// @Tags products
// @Accept json
// @Produce json
// @Param limit query int false "Number of products to return (default 10, max 100)"
// @Param offset query int false "Number of products to skip (default 0)"
// @Success 200 {object} ListProductsResponse
// @Failure 400 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products [get]
func (h *ProductHandler) ListProducts(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid workspace ID format"})
		return
	}

	// Get pagination parameters
	limit, offset, err := validatePaginationParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Get total count
	total, err := h.common.db.CountProducts(c.Request.Context(), parsedWorkspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to count products"})
		return
	}

	// Get paginated products
	products, err := h.common.db.ListProductsWithPagination(c.Request.Context(), db.ListProductsWithPaginationParams{
		WorkspaceID: parsedWorkspaceID,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve products"})
		return
	}

	responseList := make([]ProductResponse, len(products))
	// for each product, get the active product tokens
	for i, product := range products {
		productResponse := toProductResponse(product)
		productTokenList, err := h.common.db.GetActiveProductTokensByProduct(c.Request.Context(), product.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve product tokens"})
			return
		}
		productTokenListResponse := make([]ProductTokenResponse, len(productTokenList))
		for j, productToken := range productTokenList {
			productTokenListResponse[j] = toActiveProductTokenByProductResponse(productToken)
		}
		productResponse.ProductTokens = productTokenListResponse
		responseList[i] = productResponse
	}

	hasMore := int64(offset)+int64(len(responseList)) < total

	c.JSON(http.StatusOK, ListProductsResponse{
		Object:  "list",
		Data:    responseList,
		HasMore: hasMore,
		Total:   total,
	})
}

// ListActiveProducts godoc
// @Summary List active products
// @Description Returns a list of all active products for a workspace
// @Tags products
// @Accept json
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Success 200 {array} ProductResponse
// @Failure 400 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /workspaces/{workspace_id}/products/active [get]
func (h *ProductHandler) ListActiveProducts(c *gin.Context) {
	workspaceID := c.Param("workspace_id")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid workspace ID format"})
		return
	}

	products, err := h.common.db.ListActiveProducts(c.Request.Context(), parsedWorkspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve active products"})
		return
	}

	response := make([]ProductResponse, len(products))
	for i, product := range products {
		response[i] = toProductResponse(product)
	}

	c.JSON(http.StatusOK, response)
}

// CreateProduct godoc
// @Summary Create a product
// @Description Creates a new product
// @Tags products
// @Accept json
// @Produce json
// @Param product body CreateProductRequest true "Product creation data"
// @Success 201 {object} ProductResponse
// @Failure 400 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products [post]
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	parsedWorkspaceID, err := uuid.Parse(c.GetHeader("X-Workspace-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid workspace ID format"})
		return
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid metadata format"})
		return
	}

	product, err := h.common.db.CreateProduct(c.Request.Context(), db.CreateProductParams{
		WorkspaceID:     parsedWorkspaceID,
		Name:            req.Name,
		Description:     pgtype.Text{String: req.Description, Valid: req.Description != ""},
		ProductType:     db.ProductType(req.ProductType),
		IntervalType:    db.NullIntervalType{IntervalType: db.IntervalType(req.IntervalType), Valid: req.IntervalType != ""},
		TermLength:      pgtype.Int4{Int32: req.TermLength, Valid: req.TermLength != 0},
		PriceInPennies:  int32(req.PriceInPennies),
		ImageUrl:        pgtype.Text{String: req.ImageURL, Valid: req.ImageURL != ""},
		Url:             pgtype.Text{String: req.URL, Valid: req.URL != ""},
		MerchantPaidGas: req.MerchantPaidGas,
		Active:          req.Active,
		Metadata:        metadata,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create product"})
		return
	}

	for _, productToken := range req.ProductTokens {
		networkID, err := uuid.Parse(productToken.NetworkID)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid workspace ID format"})
			return
		}
		tokenID, err := uuid.Parse(productToken.TokenID)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid token ID format"})
			return
		}
		_, err = h.common.db.CreateProductToken(c.Request.Context(), db.CreateProductTokenParams{
			ProductID: product.ID,
			NetworkID: networkID,
			TokenID:   tokenID,
			Active:    productToken.Active,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create product token"})
			return
		}
	}

	c.JSON(http.StatusCreated, toProductResponse(product))
}

// UpdateProduct godoc
// @Summary Update a product
// @Description Updates an existing product
// @Tags products
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param product body UpdateProductRequest true "Product update data"
// @Success 200 {object} ProductResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/{product_id} [put]
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id := c.Param("product_id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid product ID format"})
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	params, err := h.updateProductParams(parsedUUID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	product, err := h.common.db.UpdateProduct(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update product"})
		return
	}

	c.JSON(http.StatusOK, toProductResponse(product))
}

// DeleteProduct godoc
// @Summary Delete a product
// @Description Soft deletes a product
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid product ID format"})
		return
	}

	err = h.common.db.DeleteProduct(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Product not found"})
		return
	}

	err = h.common.db.DeleteProductTokensByProduct(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete product tokens"})
		return
	}

	c.Status(http.StatusNoContent)
}

// Helper function to convert database model to API response
func toProductResponse(p db.Product) ProductResponse {
	return ProductResponse{
		ID:              p.ID.String(),
		Object:          "product",
		WorkspaceID:     p.WorkspaceID.String(),
		Name:            p.Name,
		Description:     p.Description.String,
		ProductType:     string(p.ProductType),
		IntervalType:    string(p.IntervalType.IntervalType),
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
