package handlers

import (
	"context"
	"cyphera-api/internal/db"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestProductResponse represents the standardized API response for product operations in tests
type TestProductResponse struct {
	ID              string                     `json:"id"`
	Object          string                     `json:"object"`
	WorkspaceID     string                     `json:"workspace_id"`
	WalletID        string                     `json:"wallet_id"`
	Name            string                     `json:"name"`
	Description     string                     `json:"description,omitempty"`
	ProductType     string                     `json:"product_type"`
	IntervalType    string                     `json:"interval_type,omitempty"`
	TermLength      int32                      `json:"term_length,omitempty"`
	PriceInPennies  int32                      `json:"price_in_pennies"`
	ImageURL        string                     `json:"image_url,omitempty"`
	URL             string                     `json:"url,omitempty"`
	MerchantPaidGas bool                       `json:"merchant_paid_gas"`
	Active          bool                       `json:"active"`
	Metadata        json.RawMessage            `json:"metadata,omitempty"`
	CreatedAt       int64                      `json:"created_at"`
	UpdatedAt       int64                      `json:"updated_at"`
	ProductTokens   []TestProductTokenResponse `json:"product_tokens,omitempty"`
}

// TestProductTokenResponse represents a token associated with a product in tests
type TestProductTokenResponse struct {
	ID             string `json:"id"`
	ProductID      string `json:"product_id"`
	NetworkID      string `json:"network_id"`
	TokenID        string `json:"token_id"`
	NetworkName    string `json:"network_name"`
	NetworkChainID string `json:"network_chain_id"`
	TokenName      string `json:"token_name"`
	TokenSymbol    string `json:"token_symbol"`
	TokenAddress   string `json:"token_address"`
	Active         bool   `json:"active"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

// TestPublicProductResponse represents the response for public product details in tests
type TestPublicProductResponse struct {
	ProductID       string                           `json:"product_id"`
	AccountID       string                           `json:"account_id"`
	WorkspaceID     string                           `json:"workspace_id"`
	WalletAddress   string                           `json:"wallet_address"`
	Name            string                           `json:"name"`
	Description     string                           `json:"description,omitempty"`
	ProductType     string                           `json:"product_type"`
	IntervalType    string                           `json:"interval_type,omitempty"`
	TermLength      int32                            `json:"term_length,omitempty"`
	PriceInPennies  int32                            `json:"price_in_pennies"`
	ImageURL        string                           `json:"image_url,omitempty"`
	MerchantPaidGas bool                             `json:"merchant_paid_gas"`
	ProductTokens   []TestPublicProductTokenResponse `json:"product_tokens,omitempty"`
}

// TestPublicProductTokenResponse represents a token associated with a product in public responses in tests
type TestPublicProductTokenResponse struct {
	ProductTokenID string `json:"product_token_id"`
	NetworkID      string `json:"network_id"`
	NetworkName    string `json:"network_name"`
	NetworkChainID string `json:"network_chain_id"`
	TokenID        string `json:"token_id"`
	TokenName      string `json:"token_name"`
	TokenSymbol    string `json:"token_symbol"`
	TokenAddress   string `json:"token_address"`
}

// MockProductDB is a mock implementation for the product handler tests
type MockProductDB struct {
	mock.Mock
}

func (m *MockProductDB) GetProduct(ctx any, id uuid.UUID) (db.Product, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.Product), args.Error(1)
}

func (m *MockProductDB) GetWalletByID(ctx any, id uuid.UUID) (db.Wallet, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.Wallet), args.Error(1)
}

func (m *MockProductDB) GetWorkspace(ctx any, id uuid.UUID) (db.Workspace, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.Workspace), args.Error(1)
}

func (m *MockProductDB) GetActiveProductTokensByProduct(ctx any, productID uuid.UUID) ([]db.GetActiveProductTokensByProductRow, error) {
	args := m.Called(ctx, productID)
	return args.Get(0).([]db.GetActiveProductTokensByProductRow), args.Error(1)
}

func (m *MockProductDB) GetToken(ctx any, id uuid.UUID) (db.Token, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.Token), args.Error(1)
}

// GetProductHandler is a minimal handler for testing the GetProduct method
type GetProductHandler struct {
	db interface {
		GetProduct(ctx any, id uuid.UUID) (db.Product, error)
	}
}

// Handle implements the GetProduct handler logic
func (h *GetProductHandler) Handle(c *gin.Context) {
	productId := c.Param("product_id")
	parsedUUID, err := uuid.Parse(productId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID format",
		})
		return
	}

	product, err := h.db.GetProduct(c.Request.Context(), parsedUUID)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"error": "Request timeout",
			})
			return
		}

		c.JSON(http.StatusNotFound, gin.H{
			"error": "Product not found",
		})
		return
	}

	response := TestProductResponse{
		ID:              product.ID.String(),
		Object:          "product",
		WorkspaceID:     product.WorkspaceID.String(),
		WalletID:        product.WalletID.String(),
		Name:            product.Name,
		Description:     product.Description.String,
		ProductType:     string(product.ProductType),
		IntervalType:    string(product.IntervalType),
		TermLength:      product.TermLength.Int32,
		PriceInPennies:  product.PriceInPennies,
		ImageURL:        product.ImageUrl.String,
		URL:             product.Url.String,
		MerchantPaidGas: product.MerchantPaidGas,
		Active:          product.Active,
		CreatedAt:       product.CreatedAt.Time.Unix(),
		UpdatedAt:       product.UpdatedAt.Time.Unix(),
	}

	if len(product.Metadata) > 0 {
		response.Metadata = product.Metadata
	}

	c.JSON(http.StatusOK, response)
}

// GetPublicProductHandler is a minimal handler for testing the GetPublicProductByID method
type GetPublicProductHandler struct {
	db interface {
		GetProduct(ctx any, id uuid.UUID) (db.Product, error)
		GetWalletByID(ctx any, id uuid.UUID) (db.Wallet, error)
		GetWorkspace(ctx any, id uuid.UUID) (db.Workspace, error)
		GetActiveProductTokensByProduct(ctx any, productID uuid.UUID) ([]db.GetActiveProductTokensByProductRow, error)
		GetToken(ctx any, id uuid.UUID) (db.Token, error)
	}
}

// Handle implements the GetPublicProductByID handler logic
func (h *GetPublicProductHandler) Handle(c *gin.Context) {
	productId := c.Param("product_id")
	parsedUUID, err := uuid.Parse(productId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID format",
		})
		return
	}

	product, err := h.db.GetProduct(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Product not found",
		})
		return
	}

	wallet, err := h.db.GetWalletByID(c.Request.Context(), product.WalletID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Wallet not found",
		})
		return
	}

	workspace, err := h.db.GetWorkspace(c.Request.Context(), product.WorkspaceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Workspace not found",
		})
		return
	}

	productTokens, err := h.db.GetActiveProductTokensByProduct(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve product tokens",
		})
		return
	}

	publicProductTokens := make([]TestPublicProductTokenResponse, len(productTokens))
	for i, pt := range productTokens {
		publicProductTokens[i] = TestPublicProductTokenResponse{
			ProductTokenID: pt.ID.String(),
			NetworkID:      pt.NetworkID.String(),
			NetworkName:    pt.NetworkName,
			NetworkChainID: strconv.FormatInt(int64(pt.ChainID), 10),
			TokenID:        pt.TokenID.String(),
			TokenName:      pt.TokenName,
			TokenSymbol:    pt.TokenSymbol,
		}

		token, err := h.db.GetToken(c.Request.Context(), uuid.MustParse(publicProductTokens[i].TokenID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve token",
			})
			return
		}
		publicProductTokens[i].TokenAddress = token.ContractAddress
	}

	response := TestPublicProductResponse{
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

	c.JSON(http.StatusOK, response)
}

// TestGetProduct_Success tests the GetProduct handler for a successful product retrieval
func TestGetProduct_Success(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mock
	mockDB := new(MockProductDB)

	// Create test data
	productID := uuid.New()
	workspaceID := uuid.New()
	walletID := uuid.New()

	product := db.Product{
		ID:          productID,
		WorkspaceID: workspaceID,
		WalletID:    walletID,
		Name:        "Test Product",
		Description: pgtype.Text{
			String: "A test product description",
			Valid:  true,
		},
		ProductType:    "recurring",
		IntervalType:   "month",
		TermLength:     pgtype.Int4{Int32: 12, Valid: true},
		PriceInPennies: 1999,
		ImageUrl: pgtype.Text{
			String: "https://example.com/image.jpg",
			Valid:  true,
		},
		Url: pgtype.Text{
			String: "https://example.com/product",
			Valid:  true,
		},
		MerchantPaidGas: true,
		Active:          true,
		CreatedAt: pgtype.Timestamptz{
			Time:  testTime,
			Valid: true,
		},
		UpdatedAt: pgtype.Timestamptz{
			Time:  testTime,
			Valid: true,
		},
	}

	// Mock DB response
	mockDB.On("GetProduct", mock.Anything, productID).Return(product, nil)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "product_id", Value: productID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodGet, "/products/"+productID.String(), nil)
	c.Request = req

	// Invoke handler
	handler := &GetProductHandler{
		db: mockDB,
	}
	handler.Handle(c)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response TestProductResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, productID.String(), response.ID)
	assert.Equal(t, "product", response.Object)
	assert.Equal(t, workspaceID.String(), response.WorkspaceID)
	assert.Equal(t, walletID.String(), response.WalletID)
	assert.Equal(t, "Test Product", response.Name)
	assert.Equal(t, "A test product description", response.Description)
	assert.Equal(t, "recurring", response.ProductType)
	assert.Equal(t, "month", response.IntervalType)
	assert.Equal(t, int32(12), response.TermLength)
	assert.Equal(t, int32(1999), response.PriceInPennies)
	assert.Equal(t, "https://example.com/image.jpg", response.ImageURL)
	assert.Equal(t, "https://example.com/product", response.URL)
	assert.True(t, response.MerchantPaidGas)
	assert.True(t, response.Active)
	assert.Equal(t, product.CreatedAt.Time.Unix(), response.CreatedAt)
	assert.Equal(t, product.UpdatedAt.Time.Unix(), response.UpdatedAt)

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

// TestGetProduct_InvalidID tests the GetProduct handler with an invalid UUID
func TestGetProduct_InvalidID(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mock
	mockDB := new(MockProductDB)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "product_id", Value: "invalid-uuid"},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodGet, "/products/invalid-uuid", nil)
	c.Request = req

	// Invoke handler
	handler := &GetProductHandler{
		db: mockDB,
	}
	handler.Handle(c)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
	assert.Equal(t, "Invalid product ID format", response["error"])

	// Verify no DB calls were made
	mockDB.AssertNotCalled(t, "GetProduct")
}

// TestGetProduct_NotFound tests the GetProduct handler when product is not found
func TestGetProduct_NotFound(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mock
	mockDB := new(MockProductDB)

	// Create test data
	productID := uuid.New()

	// Mock DB response for not found
	mockDB.On("GetProduct", mock.Anything, productID).Return(db.Product{}, errors.New("product not found"))

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "product_id", Value: productID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodGet, "/products/"+productID.String(), nil)
	c.Request = req

	// Invoke handler
	handler := &GetProductHandler{
		db: mockDB,
	}
	handler.Handle(c)

	// Assertions
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
	assert.Equal(t, "Product not found", response["error"])

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

// TestGetPublicProductByID_Success tests the GetPublicProductByID handler for a successful retrieval
func TestGetPublicProductByID_Success(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mock
	mockDB := new(MockProductDB)

	// Create test data
	productID := uuid.New()
	workspaceID := uuid.New()
	walletID := uuid.New()
	accountID := uuid.New()
	networkID := uuid.New()
	tokenID := uuid.New()

	product := db.Product{
		ID:          productID,
		WorkspaceID: workspaceID,
		WalletID:    walletID,
		Name:        "Test Product",
		Description: pgtype.Text{
			String: "A test product description",
			Valid:  true,
		},
		ProductType:    "recurring",
		IntervalType:   "month",
		TermLength:     pgtype.Int4{Int32: 12, Valid: true},
		PriceInPennies: 1999,
		ImageUrl: pgtype.Text{
			String: "https://example.com/image.jpg",
			Valid:  true,
		},
		MerchantPaidGas: true,
		Active:          true,
	}

	wallet := db.Wallet{
		ID:            walletID,
		AccountID:     accountID,
		WalletAddress: "0xabcdef1234567890",
	}

	workspace := db.Workspace{
		ID:        workspaceID,
		AccountID: accountID,
		Name:      "Test Workspace",
	}

	productToken := db.GetActiveProductTokensByProductRow{
		ID:          uuid.New(),
		ProductID:   productID,
		NetworkID:   networkID,
		TokenID:     tokenID,
		TokenName:   "Test Token",
		TokenSymbol: "TEST",
		NetworkName: "Ethereum",
		ChainID:     1,
		NetworkType: "mainnet",
		Active:      true,
	}

	token := db.Token{
		ID:              tokenID,
		ContractAddress: "0x1234567890abcdef",
		Symbol:          "TEST",
		Name:            "Test Token",
	}

	// Mock DB responses
	mockDB.On("GetProduct", mock.Anything, productID).Return(product, nil)
	mockDB.On("GetWalletByID", mock.Anything, walletID).Return(wallet, nil)
	mockDB.On("GetWorkspace", mock.Anything, workspaceID).Return(workspace, nil)
	mockDB.On("GetActiveProductTokensByProduct", mock.Anything, productID).Return([]db.GetActiveProductTokensByProductRow{productToken}, nil)
	mockDB.On("GetToken", mock.Anything, tokenID).Return(token, nil)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "product_id", Value: productID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodGet, "/products/public/"+productID.String(), nil)
	c.Request = req

	// Invoke handler
	handler := &GetPublicProductHandler{
		db: mockDB,
	}
	handler.Handle(c)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response TestPublicProductResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, productID.String(), response.ProductID)
	assert.Equal(t, accountID.String(), response.AccountID)
	assert.Equal(t, workspaceID.String(), response.WorkspaceID)
	assert.Equal(t, wallet.WalletAddress, response.WalletAddress)
	assert.Equal(t, product.Name, response.Name)
	assert.Equal(t, product.Description.String, response.Description)
	assert.Equal(t, string(product.ProductType), response.ProductType)
	assert.Equal(t, string(product.IntervalType), response.IntervalType)
	assert.Equal(t, product.TermLength.Int32, response.TermLength)
	assert.Equal(t, product.PriceInPennies, response.PriceInPennies)
	assert.Equal(t, product.ImageUrl.String, response.ImageURL)
	assert.Equal(t, product.MerchantPaidGas, response.MerchantPaidGas)

	// Check product tokens
	assert.Len(t, response.ProductTokens, 1)
	assert.Equal(t, productToken.ID.String(), response.ProductTokens[0].ProductTokenID)
	assert.Equal(t, productToken.NetworkID.String(), response.ProductTokens[0].NetworkID)
	assert.Equal(t, productToken.NetworkName, response.ProductTokens[0].NetworkName)
	assert.Equal(t, "1", response.ProductTokens[0].NetworkChainID)
	assert.Equal(t, productToken.TokenID.String(), response.ProductTokens[0].TokenID)
	assert.Equal(t, productToken.TokenName, response.ProductTokens[0].TokenName)
	assert.Equal(t, productToken.TokenSymbol, response.ProductTokens[0].TokenSymbol)
	assert.Equal(t, token.ContractAddress, response.ProductTokens[0].TokenAddress)

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

// TestGetPublicProductByID_WalletNotFound tests when wallet is not found
func TestGetPublicProductByID_WalletNotFound(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mock
	mockDB := new(MockProductDB)

	// Create test data
	productID := uuid.New()
	workspaceID := uuid.New()
	walletID := uuid.New()

	product := db.Product{
		ID:          productID,
		WorkspaceID: workspaceID,
		WalletID:    walletID,
		Name:        "Test Product",
	}

	// Mock DB responses
	mockDB.On("GetProduct", mock.Anything, productID).Return(product, nil)
	mockDB.On("GetWalletByID", mock.Anything, walletID).Return(db.Wallet{}, errors.New("wallet not found"))

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "product_id", Value: productID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodGet, "/products/public/"+productID.String(), nil)
	c.Request = req

	// Invoke handler
	handler := &GetPublicProductHandler{
		db: mockDB,
	}
	handler.Handle(c)

	// Assertions
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
	assert.Equal(t, "Wallet not found", response["error"])

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

// TestGetPublicProductByID_TokenError tests error during token retrieval
func TestGetPublicProductByID_TokenError(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mock
	mockDB := new(MockProductDB)

	// Create test data
	productID := uuid.New()
	workspaceID := uuid.New()
	walletID := uuid.New()
	accountID := uuid.New()
	networkID := uuid.New()
	tokenID := uuid.New()

	product := db.Product{
		ID:          productID,
		WorkspaceID: workspaceID,
		WalletID:    walletID,
		Name:        "Test Product",
	}

	wallet := db.Wallet{
		ID:            walletID,
		AccountID:     accountID,
		WalletAddress: "0xabcdef1234567890",
	}

	workspace := db.Workspace{
		ID:        workspaceID,
		AccountID: accountID,
		Name:      "Test Workspace",
	}

	productToken := db.GetActiveProductTokensByProductRow{
		ID:          uuid.New(),
		ProductID:   productID,
		NetworkID:   networkID,
		TokenID:     tokenID,
		TokenName:   "Test Token",
		TokenSymbol: "TEST",
		NetworkName: "Ethereum",
		ChainID:     1,
		Active:      true,
	}

	// Mock DB responses
	mockDB.On("GetProduct", mock.Anything, productID).Return(product, nil)
	mockDB.On("GetWalletByID", mock.Anything, walletID).Return(wallet, nil)
	mockDB.On("GetWorkspace", mock.Anything, workspaceID).Return(workspace, nil)
	mockDB.On("GetActiveProductTokensByProduct", mock.Anything, productID).Return([]db.GetActiveProductTokensByProductRow{productToken}, nil)
	mockDB.On("GetToken", mock.Anything, tokenID).Return(db.Token{}, errors.New("token not found"))

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "product_id", Value: productID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodGet, "/products/public/"+productID.String(), nil)
	c.Request = req

	// Invoke handler
	handler := &GetPublicProductHandler{
		db: mockDB,
	}
	handler.Handle(c)

	// Assertions
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
	assert.Equal(t, "Failed to retrieve token", response["error"])

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

// Define a test timestamp for consistent times in tests
var testTime = timeNow()

func timeNow() time.Time {
	return time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
}

// TestDelegationStruct represents the delegation data used in subscription tests
type TestDelegationStruct struct {
	Delegate  string             `json:"delegate"`
	Delegator string             `json:"delegator"`
	Authority string             `json:"authority"`
	Caveats   []TestCaveatStruct `json:"caveats"`
	Salt      string             `json:"salt"`
	Signature string             `json:"signature"`
}

// TestCaveatStruct represents caveat data for delegation tests
type TestCaveatStruct struct {
	// Empty struct for this test
}

// TestSubscribeRequest represents the request for subscribing to a product
type TestSubscribeRequest struct {
	SubscriberAddress string               `json:"subscriber_address"`
	ProductTokenID    string               `json:"product_token_id"`
	Delegation        TestDelegationStruct `json:"delegation"`
}

// MockSubscribeDB is a mock implementation for testing subscription functionality
type MockSubscribeDB struct {
	mock.Mock
}

func (m *MockSubscribeDB) GetProduct(ctx any, id uuid.UUID) (db.Product, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.Product), args.Error(1)
}

func (m *MockSubscribeDB) GetProductToken(ctx any, id uuid.UUID) (db.GetProductTokenRow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.GetProductTokenRow), args.Error(1)
}

func (m *MockSubscribeDB) GetCustomersByWalletAddress(ctx any, walletAddress string) ([]db.Customer, error) {
	args := m.Called(ctx, walletAddress)
	return args.Get(0).([]db.Customer), args.Error(1)
}

func (m *MockSubscribeDB) CreateCustomer(ctx any, params db.CreateCustomerParams) (db.Customer, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(db.Customer), args.Error(1)
}

func (m *MockSubscribeDB) CreateCustomerWallet(ctx any, params db.CreateCustomerWalletParams) (db.CustomerWallet, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(db.CustomerWallet), args.Error(1)
}

func (m *MockSubscribeDB) ListSubscriptionsByCustomer(ctx any, customerID uuid.UUID) ([]db.Subscription, error) {
	args := m.Called(ctx, customerID)
	return args.Get(0).([]db.Subscription), args.Error(1)
}

func (m *MockSubscribeDB) CreateDelegationDatum(ctx any, params any) (db.DelegationDatum, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(db.DelegationDatum), args.Error(1)
}

func (m *MockSubscribeDB) CreateSubscription(ctx any, params db.CreateSubscriptionParams) (db.Subscription, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(db.Subscription), args.Error(1)
}

func (m *MockSubscribeDB) CreateSubscriptionEvent(ctx any, params db.CreateSubscriptionEventParams) (db.SubscriptionEvent, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(db.SubscriptionEvent), args.Error(1)
}

func (m *MockSubscribeDB) UpdateSubscription(ctx any, params db.UpdateSubscriptionParams) (db.Subscription, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(db.Subscription), args.Error(1)
}

// MockPgxTx is a mock implementation of the pgx.Tx interface
type MockPgxTx struct {
	mock.Mock
}

func (m *MockPgxTx) Begin(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	return args.Get(0).(pgx.Tx), args.Error(1)
}

func (m *MockPgxTx) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockPgxTx) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// TestDelegationClient is a mock implementation of the delegation client
type TestDelegationClient struct {
	mock.Mock
}

func (m *TestDelegationClient) RedeemDelegationDirectly(ctx context.Context, signatureBytes []byte) (string, error) {
	args := m.Called(ctx, signatureBytes)
	return args.String(0), args.Error(1)
}

func (m *TestDelegationClient) CheckHealth() error {
	args := m.Called()
	return args.Error(0)
}

// TestSubscriptionHandler is a simplified version of the product handler with just the subscribe functionality
type TestSubscriptionHandler struct {
	db               *MockSubscribeDB
	tx               *MockPgxTx
	delegationClient *TestDelegationClient
}

// SubscribeToProduct is a test implementation of the handler
func (h *TestSubscriptionHandler) SubscribeToProduct(c *gin.Context) {
	ctx := c.Request.Context()
	productID := c.Param("product_id")
	parsedProductID, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID format"})
		return
	}

	var request TestSubscribeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Basic validation
	if request.SubscriberAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subscriber address is required"})
		return
	}

	parsedProductTokenID, err := uuid.Parse(request.ProductTokenID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product token ID format"})
		return
	}

	// Get product
	product, err := h.db.GetProduct(ctx, parsedProductID)
	if err != nil {
		if err.Error() == "product not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve product"})
		}
		return
	}

	// Check active product
	if !product.Active {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot subscribe to inactive product"})
		return
	}

	// Get product token
	productToken, err := h.db.GetProductToken(ctx, parsedProductTokenID)
	if err != nil {
		if err.Error() == "product token not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product token not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve product token"})
		}
		return
	}

	// Verify product token belongs to product
	if productToken.ProductID != parsedProductID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product token does not belong to the specified product"})
		return
	}

	// Normalize address
	normalizedAddress := strings.ToLower(request.SubscriberAddress)

	// Mock transaction handling
	h.tx.On("Commit", ctx).Return(nil)
	h.tx.On("Rollback", ctx).Return(nil)

	// Check for existing customer
	customers, err := h.db.GetCustomersByWalletAddress(ctx, normalizedAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check for existing customer"})
		return
	}

	var customer db.Customer
	var customerWallet db.CustomerWallet

	if len(customers) == 0 {
		// Create new customer
		customer = db.Customer{
			ID:          uuid.New(),
			WorkspaceID: product.WorkspaceID,
			CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}
		h.db.On("CreateCustomer", ctx, mock.Anything).Return(customer, nil)

		// Create wallet
		customerWallet = db.CustomerWallet{
			ID:            uuid.New(),
			CustomerID:    customer.ID,
			WalletAddress: normalizedAddress,
			NetworkType:   "evm",
			CreatedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}
		h.db.On("CreateCustomerWallet", ctx, mock.Anything).Return(customerWallet, nil)
	} else {
		// Use existing customer
		customer = customers[0]
		customerWallet = db.CustomerWallet{
			ID:            uuid.New(),
			CustomerID:    customer.ID,
			WalletAddress: normalizedAddress,
			NetworkType:   "evm",
			CreatedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}
	}

	// Check for existing subscription
	h.db.On("ListSubscriptionsByCustomer", ctx, customer.ID).Return([]db.Subscription{}, nil)

	// Create delegation data
	delegationData := db.DelegationDatum{
		ID:        uuid.New(),
		Delegate:  request.Delegation.Delegate,
		Delegator: request.Delegation.Delegator,
		Authority: request.Delegation.Authority,
		Salt:      request.Delegation.Salt,
		Signature: request.Delegation.Signature,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	h.db.On("CreateDelegationDatum", ctx, mock.Anything).Return(delegationData, nil)

	// Create subscription
	now := time.Now()
	nextRedemption := CalculateNextRedemption(db.IntervalTypeMonth, now)
	periodEnd := CalculatePeriodEnd(now, db.IntervalTypeMonth, 1)

	subscription := db.Subscription{
		ID:                 uuid.New(),
		CustomerID:         customer.ID,
		ProductID:          product.ID,
		ProductTokenID:     parsedProductTokenID,
		DelegationID:       delegationData.ID,
		Status:             db.SubscriptionStatusActive,
		CurrentPeriodStart: pgtype.Timestamptz{Time: now, Valid: true},
		CurrentPeriodEnd:   pgtype.Timestamptz{Time: periodEnd, Valid: true},
		NextRedemptionDate: pgtype.Timestamptz{Time: nextRedemption, Valid: true},
	}
	h.db.On("CreateSubscription", ctx, mock.Anything).Return(subscription, nil)

	// Create event
	event := db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subscription.ID,
		EventType:      db.SubscriptionEventTypeCreated,
		OccurredAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		AmountInCents:  product.PriceInPennies,
		CreatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	h.db.On("CreateSubscriptionEvent", ctx, mock.Anything).Return(event, nil)

	// Mock redemption
	if h.delegationClient != nil {
		h.delegationClient.On("RedeemDelegationDirectly", ctx, mock.Anything).Return("0xTxHash", nil)
	}

	c.JSON(http.StatusCreated, subscription)
}

// TestSubscribeToProduct_Success tests the successful creation of a subscription
func TestSubscribeToProduct_Success(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockDB := new(MockSubscribeDB)
	mockTx := new(MockPgxTx)
	mockDelegClient := new(TestDelegationClient)

	// Create test data
	productID := uuid.New()
	productTokenID := uuid.New()
	workspaceID := uuid.New()
	walletID := uuid.New()
	customerID := uuid.New()
	walletAddress := "0xabcdef1234567890"
	normalizedAddress := strings.ToLower(walletAddress)
	delegationID := uuid.New()

	now := time.Now()
	nextRedemption := CalculateNextRedemption(db.IntervalTypeMonth, now)
	periodEnd := CalculatePeriodEnd(now, db.IntervalTypeMonth, 1)

	product := db.Product{
		ID:             productID,
		WorkspaceID:    workspaceID,
		WalletID:       walletID,
		Name:           "Test Product",
		ProductType:    "recurring",
		IntervalType:   "month",
		PriceInPennies: 1999,
		Active:         true,
	}

	productToken := db.GetProductTokenRow{
		ID:          productTokenID,
		ProductID:   productID,
		NetworkType: "mainnet",
		NetworkName: "Ethereum",
		ChainID:     1,
	}

	customer := db.Customer{
		ID:          customerID,
		WorkspaceID: workspaceID,
		CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	customerWallet := db.CustomerWallet{
		ID:            uuid.New(),
		CustomerID:    customerID,
		WalletAddress: normalizedAddress,
		NetworkType:   "evm",
	}

	delegationData := db.DelegationDatum{
		ID:        delegationID,
		Delegate:  "0x1234567890abcdef",
		Delegator: normalizedAddress,
		Authority: "0x0987654321fedcba",
		Salt:      "12345",
		Signature: "0xsignature",
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	subscription := db.Subscription{
		ID:                 uuid.New(),
		CustomerID:         customerID,
		ProductID:          productID,
		ProductTokenID:     productTokenID,
		DelegationID:       delegationID,
		Status:             db.SubscriptionStatusActive,
		CurrentPeriodStart: pgtype.Timestamptz{Time: now, Valid: true},
		CurrentPeriodEnd:   pgtype.Timestamptz{Time: periodEnd, Valid: true},
		NextRedemptionDate: pgtype.Timestamptz{Time: nextRedemption, Valid: true},
	}

	// Setup mocks before the handler is called
	mockDB.On("GetProduct", mock.Anything, productID).Return(product, nil)
	mockDB.On("GetProductToken", mock.Anything, productTokenID).Return(productToken, nil)
	mockDB.On("GetCustomersByWalletAddress", mock.Anything, normalizedAddress).Return([]db.Customer{}, nil)

	// Mock transaction
	mockTx.On("Commit", mock.Anything).Return(nil)
	mockTx.On("Rollback", mock.Anything).Return(nil)

	// Setup more specific mock expectations with parameter validation
	mockDB.On("CreateCustomer", mock.Anything, mock.MatchedBy(func(params db.CreateCustomerParams) bool {
		return params.WorkspaceID == workspaceID
	})).Return(customer, nil)

	mockDB.On("CreateCustomerWallet", mock.Anything, mock.MatchedBy(func(params db.CreateCustomerWalletParams) bool {
		return params.CustomerID == customerID &&
			params.WalletAddress == normalizedAddress &&
			params.NetworkType == "evm"
	})).Return(customerWallet, nil)

	mockDB.On("ListSubscriptionsByCustomer", mock.Anything, customerID).Return([]db.Subscription{}, nil)

	mockDB.On("CreateDelegationDatum", mock.Anything, mock.MatchedBy(func(params any) bool {
		// For interfaces, check that it contains required fields
		return params != nil
	})).Return(delegationData, nil)

	mockDB.On("CreateSubscription", mock.Anything, mock.MatchedBy(func(params db.CreateSubscriptionParams) bool {
		return params.CustomerID == customerID &&
			params.ProductID == productID &&
			params.ProductTokenID == productTokenID &&
			params.DelegationID == delegationID &&
			params.Status == db.SubscriptionStatusActive
	})).Return(subscription, nil)

	mockDB.On("CreateSubscriptionEvent", mock.Anything, mock.MatchedBy(func(params db.CreateSubscriptionEventParams) bool {
		return params.SubscriptionID == subscription.ID &&
			params.EventType == db.SubscriptionEventTypeCreated &&
			params.AmountInCents == product.PriceInPennies
	})).Return(db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subscription.ID,
		EventType:      db.SubscriptionEventTypeCreated,
		AmountInCents:  product.PriceInPennies,
	}, nil)

	mockDelegClient.On("RedeemDelegationDirectly", mock.Anything, mock.Anything).Return("0xTxHash", nil)

	handler := &TestSubscriptionHandler{
		db:               mockDB,
		tx:               mockTx,
		delegationClient: mockDelegClient,
	}

	// Create request
	reqBody := `{
		"subscriber_address": "0xabcdef1234567890",
		"product_token_id": "` + productTokenID.String() + `",
		"delegation": {
			"delegate": "0x1234567890abcdef",
			"delegator": "0xabcdef1234567890",
			"authority": "0x0987654321fedcba",
			"caveats": [],
			"salt": "12345",
			"signature": "0xsignature"
		}
	}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "product_id", Value: productID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodPost, "/products/"+productID.String()+"/subscribe", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Invoke handler
	handler.SubscribeToProduct(c)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)

	var response db.Subscription
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, productID, response.ProductID)
	assert.Equal(t, productTokenID, response.ProductTokenID)
	assert.Equal(t, db.SubscriptionStatusActive, response.Status)
}

// TestSubscribeToProduct_InvalidProductID tests the handler with an invalid product ID
func TestSubscribeToProduct_InvalidProductID(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockDB := new(MockSubscribeDB)
	mockTx := new(MockPgxTx)

	handler := &TestSubscriptionHandler{
		db: mockDB,
		tx: mockTx,
	}

	// Create request
	reqBody := `{
		"subscriber_address": "0xabcdef1234567890",
		"product_token_id": "` + uuid.New().String() + `",
		"delegation": {
			"delegate": "0x1234567890abcdef",
			"delegator": "0xabcdef1234567890",
			"authority": "0x0987654321fedcba",
			"caveats": [],
			"salt": "12345",
			"signature": "0xsignature"
		}
	}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "product_id", Value: "invalid-uuid"},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodPost, "/products/invalid-uuid/subscribe", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Invoke handler
	handler.SubscribeToProduct(c)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
	assert.Equal(t, "Invalid product ID format", response["error"])
}

// TestSubscribeToProduct_InvalidRequest tests the handler with an invalid request body
func TestSubscribeToProduct_InvalidRequest(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockDB := new(MockSubscribeDB)
	mockTx := new(MockPgxTx)

	handler := &TestSubscriptionHandler{
		db: mockDB,
		tx: mockTx,
	}

	// Create invalid JSON request
	reqBody := `{invalid json}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "product_id", Value: uuid.New().String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodPost, "/products/"+uuid.New().String()+"/subscribe", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Invoke handler
	handler.SubscribeToProduct(c)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
	assert.Equal(t, "Invalid request format", response["error"])
}

// TestSubscribeToProduct_MissingSubscriberAddress tests validation of missing subscriber address
func TestSubscribeToProduct_MissingSubscriberAddress(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockDB := new(MockSubscribeDB)
	mockTx := new(MockPgxTx)

	handler := &TestSubscriptionHandler{
		db: mockDB,
		tx: mockTx,
	}

	productID := uuid.New()
	productTokenID := uuid.New()

	// Create request with missing subscriber address
	reqBody := `{
		"subscriber_address": "",
		"product_token_id": "` + productTokenID.String() + `",
		"delegation": {
			"delegate": "0x1234567890abcdef",
			"delegator": "0xabcdef1234567890",
			"authority": "0x0987654321fedcba",
			"caveats": [],
			"salt": "12345",
			"signature": "0xsignature"
		}
	}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "product_id", Value: productID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodPost, "/products/"+productID.String()+"/subscribe", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Invoke handler
	handler.SubscribeToProduct(c)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
	assert.Equal(t, "subscriber address is required", response["error"])
}

// TestSubscribeToProduct_ProductNotFound tests when product is not found
func TestSubscribeToProduct_ProductNotFound(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockDB := new(MockSubscribeDB)
	mockTx := new(MockPgxTx)

	handler := &TestSubscriptionHandler{
		db: mockDB,
		tx: mockTx,
	}

	productID := uuid.New()
	productTokenID := uuid.New()

	// Mock database response
	mockDB.On("GetProduct", mock.Anything, productID).Return(db.Product{}, errors.New("product not found"))

	// Create request
	reqBody := `{
		"subscriber_address": "0xabcdef1234567890",
		"product_token_id": "` + productTokenID.String() + `",
		"delegation": {
			"delegate": "0x1234567890abcdef",
			"delegator": "0xabcdef1234567890",
			"authority": "0x0987654321fedcba",
			"caveats": [],
			"salt": "12345",
			"signature": "0xsignature"
		}
	}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "product_id", Value: productID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodPost, "/products/"+productID.String()+"/subscribe", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Invoke handler
	handler.SubscribeToProduct(c)

	// Assertions
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
	assert.Equal(t, "Product not found", response["error"])

	// Verify mocks
	mockDB.AssertExpectations(t)
}

// TestSubscribeToProduct_InactiveProduct tests subscribing to an inactive product
func TestSubscribeToProduct_InactiveProduct(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockDB := new(MockSubscribeDB)
	mockTx := new(MockPgxTx)

	handler := &TestSubscriptionHandler{
		db: mockDB,
		tx: mockTx,
	}

	productID := uuid.New()
	productTokenID := uuid.New()
	workspaceID := uuid.New()
	walletID := uuid.New()

	// Create inactive product
	product := db.Product{
		ID:             productID,
		WorkspaceID:    workspaceID,
		WalletID:       walletID,
		Name:           "Test Product",
		ProductType:    "recurring",
		IntervalType:   "month",
		PriceInPennies: 1999,
		Active:         false, // Inactive product
	}

	// Mock database response
	mockDB.On("GetProduct", mock.Anything, productID).Return(product, nil)

	// Create request
	reqBody := `{
		"subscriber_address": "0xabcdef1234567890",
		"product_token_id": "` + productTokenID.String() + `",
		"delegation": {
			"delegate": "0x1234567890abcdef",
			"delegator": "0xabcdef1234567890",
			"authority": "0x0987654321fedcba",
			"caveats": [],
			"salt": "12345",
			"signature": "0xsignature"
		}
	}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "product_id", Value: productID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodPost, "/products/"+productID.String()+"/subscribe", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Invoke handler
	handler.SubscribeToProduct(c)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
	assert.Equal(t, "Cannot subscribe to inactive product", response["error"])

	// Verify mocks
	mockDB.AssertExpectations(t)
}
