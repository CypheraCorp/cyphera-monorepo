package handlers

import (
	"cyphera-api/internal/db"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ProductTokenResponse represents the standardized API response for product token operations
type ProductTokenResponse struct {
	ID              string `json:"id"`
	ProductID       string `json:"product_id"`
	NetworkID       string `json:"network_id"`
	TokenID         string `json:"token_id"`
	TokenName       string `json:"token_name,omitempty"`
	TokenSymbol     string `json:"token_symbol,omitempty"`
	ContractAddress string `json:"contract_address,omitempty"`
	GasToken        bool   `json:"gas_token,omitempty"`
	ChainID         int32  `json:"chain_id,omitempty"`
	NetworkName     string `json:"network_name,omitempty"`
	NetworkType     string `json:"network_type,omitempty"`
	Active          bool   `json:"active"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

// CreateProductTokenRequest represents the request body for creating a product token
type CreateProductTokenRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	NetworkID string `json:"network_id" binding:"required"`
	TokenID   string `json:"token_id" binding:"required"`
	Active    bool   `json:"active"`
}

// UpdateProductTokenRequest represents the request body for updating a product token
type UpdateProductTokenRequest struct {
	Active bool `json:"active" binding:"required"`
}

// Helper functions to convert database models to API responses
func toBasicProductTokenResponse(pt db.ProductsToken) ProductTokenResponse {
	return ProductTokenResponse{
		ID:        pt.ID.String(),
		ProductID: pt.ProductID.String(),
		NetworkID: pt.NetworkID.String(),
		TokenID:   pt.TokenID.String(),
		Active:    pt.Active,
		CreatedAt: pt.CreatedAt.Time.Unix(),
		UpdatedAt: pt.UpdatedAt.Time.Unix(),
	}
}

func toProductTokenResponse(pt db.GetProductTokenRow) ProductTokenResponse {
	return ProductTokenResponse{
		ID:              pt.ID.String(),
		ProductID:       pt.ProductID.String(),
		NetworkID:       pt.NetworkID.String(),
		TokenID:         pt.TokenID.String(),
		TokenName:       pt.TokenName,
		TokenSymbol:     pt.TokenSymbol,
		ContractAddress: pt.ContractAddress,
		GasToken:        pt.GasToken,
		ChainID:         pt.ChainID,
		NetworkName:     pt.NetworkName,
		NetworkType:     pt.NetworkType,
		Active:          pt.Active,
		CreatedAt:       pt.CreatedAt.Time.Unix(),
		UpdatedAt:       pt.UpdatedAt.Time.Unix(),
	}
}

func toProductTokenByIdsResponse(pt db.GetProductTokenByIdsRow) ProductTokenResponse {
	return ProductTokenResponse{
		ID:              pt.ID.String(),
		ProductID:       pt.ProductID.String(),
		NetworkID:       pt.NetworkID.String(),
		TokenID:         pt.TokenID.String(),
		TokenName:       pt.TokenName,
		TokenSymbol:     pt.TokenSymbol,
		ContractAddress: pt.ContractAddress,
		GasToken:        pt.GasToken,
		Active:          pt.Active,
		CreatedAt:       pt.CreatedAt.Time.Unix(),
		UpdatedAt:       pt.UpdatedAt.Time.Unix(),
	}
}

func toProductTokenByNetworkResponse(pt db.GetProductTokensByNetworkRow) ProductTokenResponse {
	return ProductTokenResponse{
		ID:              pt.ID.String(),
		ProductID:       pt.ProductID.String(),
		NetworkID:       pt.NetworkID.String(),
		TokenID:         pt.TokenID.String(),
		TokenName:       pt.TokenName,
		TokenSymbol:     pt.TokenSymbol,
		ContractAddress: pt.ContractAddress,
		GasToken:        pt.GasToken,
		Active:          pt.Active,
		CreatedAt:       pt.CreatedAt.Time.Unix(),
		UpdatedAt:       pt.UpdatedAt.Time.Unix(),
	}
}

func toActiveProductTokenByNetworkResponse(pt db.GetActiveProductTokensByNetworkRow) ProductTokenResponse {
	return ProductTokenResponse{
		ID:              pt.ID.String(),
		ProductID:       pt.ProductID.String(),
		NetworkID:       pt.NetworkID.String(),
		TokenID:         pt.TokenID.String(),
		TokenName:       pt.TokenName,
		TokenSymbol:     pt.TokenSymbol,
		ContractAddress: pt.ContractAddress,
		GasToken:        pt.GasToken,
		Active:          pt.Active,
		CreatedAt:       pt.CreatedAt.Time.Unix(),
		UpdatedAt:       pt.UpdatedAt.Time.Unix(),
	}
}

func toActiveProductTokenByProductResponse(pt db.GetActiveProductTokensByProductRow) ProductTokenResponse {
	return ProductTokenResponse{
		ID:              pt.ID.String(),
		ProductID:       pt.ProductID.String(),
		NetworkID:       pt.NetworkID.String(),
		TokenID:         pt.TokenID.String(),
		TokenName:       pt.TokenName,
		TokenSymbol:     pt.TokenSymbol,
		ContractAddress: pt.ContractAddress,
		GasToken:        pt.GasToken,
		ChainID:         pt.ChainID,
		NetworkName:     pt.NetworkName,
		NetworkType:     pt.NetworkType,
		Active:          pt.Active,
		CreatedAt:       pt.CreatedAt.Time.Unix(),
		UpdatedAt:       pt.UpdatedAt.Time.Unix(),
	}
}

func toProductTokensByProductResponse(pt db.GetProductTokensByProductRow) ProductTokenResponse {
	return ProductTokenResponse{
		ID:              pt.ID.String(),
		ProductID:       pt.ProductID.String(),
		NetworkID:       pt.NetworkID.String(),
		TokenID:         pt.TokenID.String(),
		TokenName:       pt.TokenName,
		TokenSymbol:     pt.TokenSymbol,
		ContractAddress: pt.ContractAddress,
		GasToken:        pt.GasToken,
		ChainID:         pt.ChainID,
		NetworkName:     pt.NetworkName,
		NetworkType:     pt.NetworkType,
		Active:          pt.Active,
		CreatedAt:       pt.CreatedAt.Time.Unix(),
		UpdatedAt:       pt.UpdatedAt.Time.Unix(),
	}
}

// GetProductToken godoc
// @Summary Get a product token
// @Description Retrieves the details of an existing product token
// @Tags product-tokens
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param token_id path string true "Product Token ID"
// @Success 200 {object} ProductTokenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/{product_id}/tokens/{token_id} [get]
func (h *ProductHandler) GetProductToken(c *gin.Context) {
	productId := c.Param("product_id")
	parsedUUID, err := uuid.Parse(productId)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid product token ID format"})
		return
	}

	productToken, err := h.common.db.GetProductToken(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Product token not found"})
		return
	}

	c.JSON(http.StatusOK, toProductTokenResponse(productToken))
}

// GetProductTokenByIds godoc
// @Summary Get a product token by IDs
// @Description Retrieves the details of an existing product token by product, network, and token IDs
// @Tags product-tokens
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param network_id path string true "Network ID"
// @Param token_id path string true "Token ID"
// @Success 200 {object} ProductTokenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/{product_id}/networks/{network_id}/tokens/{token_id} [get]
func (h *ProductHandler) GetProductTokenByIds(c *gin.Context) {
	productID := c.Param("product_id")
	parsedProductID, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid product ID format"})
		return
	}

	networkID := c.Param("network_id")
	parsedNetworkID, err := uuid.Parse(networkID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid network ID format"})
		return
	}

	tokenID := c.Param("token_id")
	parsedTokenID, err := uuid.Parse(tokenID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid token ID format"})
		return
	}

	productToken, err := h.common.db.GetProductTokenByIds(c.Request.Context(), db.GetProductTokenByIdsParams{
		ProductID: parsedProductID,
		NetworkID: parsedNetworkID,
		TokenID:   parsedTokenID,
	})
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Product token not found"})
		return
	}

	c.JSON(http.StatusOK, toProductTokenByIdsResponse(productToken))
}

// GetProductTokensByNetwork godoc
// @Summary Get product tokens by network
// @Description Returns a list of all product tokens for a specific network
// @Tags product-tokens
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param network_id path string true "Network ID"
// @Success 200 {array} ProductTokenResponse
// @Failure 400 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/{product_id}/networks/{network_id}/tokens [get]
func (h *ProductHandler) GetProductTokensByNetwork(c *gin.Context) {
	productID := c.Param("product_id")
	parsedProductID, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid product ID format"})
		return
	}

	networkID := c.Param("network_id")
	parsedNetworkID, err := uuid.Parse(networkID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid network ID format"})
		return
	}

	productTokens, err := h.common.db.GetProductTokensByNetwork(c.Request.Context(), db.GetProductTokensByNetworkParams{
		ProductID: parsedProductID,
		NetworkID: parsedNetworkID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve product tokens"})
		return
	}

	response := make([]ProductTokenResponse, len(productTokens))
	for i, pt := range productTokens {
		response[i] = toProductTokenByNetworkResponse(pt)
	}

	c.JSON(http.StatusOK, response)
}

// GetActiveProductTokensByNetwork godoc
// @Summary Get active product tokens by network
// @Description Returns a list of all active product tokens for a specific network
// @Tags product-tokens
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param network_id path string true "Network ID"
// @Success 200 {array} ProductTokenResponse
// @Failure 400 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/{product_id}/networks/{network_id}/tokens/active [get]
func (h *ProductHandler) GetActiveProductTokensByNetwork(c *gin.Context) {
	productID := c.Param("product_id")
	parsedProductID, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid product ID format"})
		return
	}

	networkID := c.Param("network_id")
	parsedNetworkID, err := uuid.Parse(networkID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid network ID format"})
		return
	}

	productTokens, err := h.common.db.GetActiveProductTokensByNetwork(c.Request.Context(), db.GetActiveProductTokensByNetworkParams{
		ProductID: parsedProductID,
		NetworkID: parsedNetworkID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve active product tokens"})
		return
	}

	response := make([]ProductTokenResponse, len(productTokens))
	for i, pt := range productTokens {
		response[i] = toActiveProductTokenByNetworkResponse(pt)
	}

	c.JSON(http.StatusOK, response)
}

// GetProductTokensByProduct godoc
// @Summary Get product tokens by product
// @Description Returns a list of all product tokens for a specific product
// @Tags product-tokens
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Success 200 {array} ProductTokenResponse
// @Failure 400 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/{product_id}/tokens [get]
func (h *ProductHandler) GetProductTokensByProduct(c *gin.Context) {
	productID := c.Param("product_id")
	parsedProductID, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid product ID format"})
		return
	}

	productTokens, err := h.common.db.GetProductTokensByProduct(c.Request.Context(), parsedProductID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve product tokens"})
		return
	}

	response := make([]ProductTokenResponse, len(productTokens))
	for i, pt := range productTokens {
		response[i] = toProductTokensByProductResponse(pt)
	}

	c.JSON(http.StatusOK, response)
}

// GetActiveProductTokensByProduct godoc
// @Summary Get active product tokens by product
// @Description Returns a list of all active product tokens for a specific product
// @Tags product-tokens
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Success 200 {array} ProductTokenResponse
// @Failure 400 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/{product_id}/tokens/active [get]
func (h *ProductHandler) GetActiveProductTokensByProduct(c *gin.Context) {
	productID := c.Param("product_id")
	parsedProductID, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid product ID format"})
		return
	}

	productTokens, err := h.common.db.GetActiveProductTokensByProduct(c.Request.Context(), parsedProductID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve active product tokens"})
		return
	}

	response := make([]ProductTokenResponse, len(productTokens))
	for i, pt := range productTokens {
		response[i] = toActiveProductTokenByProductResponse(pt)
	}

	c.JSON(http.StatusOK, response)
}

// CreateProductToken godoc
// @Summary Create a product token
// @Description Creates a new product token
// @Tags product-tokens
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param token body CreateProductTokenRequest true "Product Token creation data"
// @Success 201 {object} ProductTokenResponse
// @Failure 400 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/{product_id}/tokens [post]
func (h *ProductHandler) CreateProductToken(c *gin.Context) {
	var req CreateProductTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid product ID format"})
		return
	}

	networkID, err := uuid.Parse(req.NetworkID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid network ID format"})
		return
	}

	tokenID, err := uuid.Parse(req.TokenID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid token ID format"})
		return
	}

	productToken, err := h.common.db.CreateProductToken(c.Request.Context(), db.CreateProductTokenParams{
		ProductID: productID,
		NetworkID: networkID,
		TokenID:   tokenID,
		Active:    req.Active,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create product token"})
		return
	}

	c.JSON(http.StatusCreated, toBasicProductTokenResponse(productToken))
}

// UpdateProductToken godoc
// @Summary Update a product token
// @Description Updates an existing product token
// @Tags product-tokens
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param network_id path string true "Network ID"
// @Param token_id path string true "Token ID"
// @Param token body UpdateProductTokenRequest true "Product Token update data"
// @Success 200 {object} ProductTokenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/{product_id}/networks/{network_id}/tokens/{token_id} [put]
func (h *ProductHandler) UpdateProductToken(c *gin.Context) {
	productID := c.Param("product_id")
	parsedProductID, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid product ID format"})
		return
	}

	networkID := c.Param("network_id")
	parsedNetworkID, err := uuid.Parse(networkID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid network ID format"})
		return
	}

	tokenID := c.Param("token_id")
	parsedTokenID, err := uuid.Parse(tokenID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid token ID format"})
		return
	}

	var req UpdateProductTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	productToken, err := h.common.db.UpdateProductToken(c.Request.Context(), db.UpdateProductTokenParams{
		ProductID: parsedProductID,
		NetworkID: parsedNetworkID,
		TokenID:   parsedTokenID,
		Active:    req.Active,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update product token"})
		return
	}

	c.JSON(http.StatusOK, toBasicProductTokenResponse(productToken))
}

// DeleteProductToken godoc
// @Summary Delete a product token
// @Description Soft deletes a product token
// @Tags product-tokens
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param network_id path string true "Network ID"
// @Param token_id path string true "Token ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /products/{product_id}/networks/{network_id}/tokens/{token_id} [delete]
func (h *ProductHandler) DeleteProductToken(c *gin.Context) {
	productID := c.Param("product_id")
	parsedProductID, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid product ID format"})
		return
	}

	networkID := c.Param("network_id")
	parsedNetworkID, err := uuid.Parse(networkID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid network ID format"})
		return
	}

	tokenID := c.Param("token_id")
	parsedTokenID, err := uuid.Parse(tokenID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid token ID format"})
		return
	}

	err = h.common.db.DeleteProductTokenByIds(c.Request.Context(), db.DeleteProductTokenByIdsParams{
		ProductID: parsedProductID,
		NetworkID: parsedNetworkID,
		TokenID:   parsedTokenID,
	})
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Product token not found"})
		return
	}

	c.Status(http.StatusNoContent)
}
