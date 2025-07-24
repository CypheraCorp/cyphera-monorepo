package testutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
)

// MockCircleClient provides a mock for Circle API client
type MockCircleClient struct {
	mock.Mock
}

func (m *MockCircleClient) CreateWallet(ctx context.Context, userID string) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *MockCircleClient) GetWallet(ctx context.Context, walletID string) (interface{}, error) {
	args := m.Called(ctx, walletID)
	return args.Get(0), args.Error(1)
}

func (m *MockCircleClient) CreateTransaction(ctx context.Context, req interface{}) (interface{}, error) {
	args := m.Called(ctx, req)
	return args.Get(0), args.Error(1)
}

// MockStripeClient provides a mock for Stripe client
type MockStripeClient struct {
	mock.Mock
}

func (m *MockStripeClient) CreateCustomer(ctx context.Context, email string) (string, error) {
	args := m.Called(ctx, email)
	return args.String(0), args.Error(1)
}

func (m *MockStripeClient) CreateSubscription(ctx context.Context, customerID, priceID string) (interface{}, error) {
	args := m.Called(ctx, customerID, priceID)
	return args.Get(0), args.Error(1)
}

func (m *MockStripeClient) CancelSubscription(ctx context.Context, subscriptionID string) error {
	args := m.Called(ctx, subscriptionID)
	return args.Error(0)
}

// MockDelegationClient provides a mock for delegation server client
type MockDelegationClient struct {
	mock.Mock
}

func (m *MockDelegationClient) RedeemDelegation(ctx context.Context, req interface{}) (interface{}, error) {
	args := m.Called(ctx, req)
	return args.Get(0), args.Error(1)
}

// TestServer creates a test HTTP server with Gin
func TestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	
	return server
}

// TestContext creates a test Gin context
func TestContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	
	return ctx, recorder
}

// SetupTestEnvironment sets up common test environment variables
func SetupTestEnvironment(t *testing.T) {
	t.Helper()
	
	// Set test environment variables
	t.Setenv("APP_ENV", "test")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/cyphera_test?sslmode=disable")
}

// AssertStatusCode checks HTTP status code
func AssertStatusCode(t *testing.T, recorder *httptest.ResponseRecorder, expected int) {
	t.Helper()
	
	if recorder.Code != expected {
		t.Errorf("Expected status code %d, got %d. Response body: %s", 
			expected, recorder.Code, recorder.Body.String())
	}
}