package handlers

import (
	"cyphera-api/internal/db"
	"encoding/json"
	"net/http"

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
	ID              string          `json:"id"`
	Object          string          `json:"object"`
	WorkspaceID     string          `json:"workspace_id"`
	Name            string          `json:"name"`
	Description     string          `json:"description,omitempty"`
	ProductType     string          `json:"product_type"`
	IntervalType    string          `json:"interval_type,omitempty"`
	TermLength      int32           `json:"term_length,omitempty"`
	PriceInPennies  int32           `json:"price_in_pennies"`
	ImageURL        string          `json:"image_url,omitempty"`
	URL             string          `json:"url,omitempty"`
	MerchantPaidGas bool            `json:"merchant_paid_gas"`
	Active          bool            `json:"active"`
	Metadata        json.RawMessage `json:"metadata,omitempty" swaggertype:"object"`
	CreatedAt       int64           `json:"created_at"`
	UpdatedAt       int64           `json:"updated_at"`
}

// CreateProductRequest represents the request body for creating a product
type CreateProductRequest struct {
	WorkspaceID     string          `json:"workspace_id" binding:"required"`
	Name            string          `json:"name" binding:"required"`
	Description     string          `json:"description"`
	ProductType     string          `json:"product_type" binding:"required"`
	IntervalType    string          `json:"interval_type"`
	TermLength      int32           `json:"term_length"`
	PriceInPennies  int32           `json:"price_in_pennies" binding:"required"`
	ImageURL        string          `json:"image_url"`
	URL             string          `json:"url"`
	MerchantPaidGas bool            `json:"merchant_paid_gas"`
	Active          bool            `json:"active"`
	Metadata        json.RawMessage `json:"metadata" swaggertype:"object"`
}

// UpdateProductRequest represents the request body for updating a product
type UpdateProductRequest struct {
	Name            string          `json:"name,omitempty"`
	Description     string          `json:"description,omitempty"`
	ProductType     string          `json:"product_type,omitempty"`
	IntervalType    string          `json:"interval_type,omitempty"`
	TermLength      *int32          `json:"term_length,omitempty"`
	PriceInPennies  *int32          `json:"price_in_pennies,omitempty"`
	ImageURL        string          `json:"image_url,omitempty"`
	URL             string          `json:"url,omitempty"`
	MerchantPaidGas *bool           `json:"merchant_paid_gas,omitempty"`
	Active          *bool           `json:"active,omitempty"`
	Metadata        json.RawMessage `json:"metadata,omitempty" swaggertype:"object"`
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

// ListProducts godoc
// @Summary List products
// @Description Returns a list of all products for a workspace
// @Tags products
// @Accept json
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Success 200 {array} ProductResponse
// @Failure 400 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /workspaces/{workspace_id}/products [get]
func (h *ProductHandler) ListProducts(c *gin.Context) {
	workspaceID := c.Param("workspace_id")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid workspace ID format"})
		return
	}

	products, err := h.common.db.ListProducts(c.Request.Context(), parsedWorkspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve products"})
		return
	}

	response := make([]ProductResponse, len(products))
	for i, product := range products {
		response[i] = toProductResponse(product)
	}

	c.JSON(http.StatusOK, response)
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

	workspaceID, err := uuid.Parse(req.WorkspaceID)
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
		WorkspaceID:     workspaceID,
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

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid metadata format"})
		return
	}

	product, err := h.common.db.UpdateProduct(c.Request.Context(), db.UpdateProductParams{
		ID:              parsedUUID,
		Name:            req.Name,
		Description:     pgtype.Text{String: req.Description, Valid: req.Description != ""},
		ProductType:     db.ProductType(req.ProductType),
		IntervalType:    db.NullIntervalType{IntervalType: db.IntervalType(req.IntervalType), Valid: req.IntervalType != ""},
		TermLength:      pgtype.Int4{Int32: *req.TermLength, Valid: req.TermLength != nil},
		PriceInPennies:  int32(*req.PriceInPennies),
		ImageUrl:        pgtype.Text{String: req.ImageURL, Valid: req.ImageURL != ""},
		Url:             pgtype.Text{String: req.URL, Valid: req.URL != ""},
		MerchantPaidGas: *req.MerchantPaidGas,
		Active:          *req.Active,
		Metadata:        metadata,
	})
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
