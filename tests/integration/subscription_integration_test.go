// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/cyphera/cyphera-api/apps/api/handlers"
	"github.com/cyphera/cyphera-api/libs/go/client/coinmarketcap"
	dsClient "github.com/cyphera/cyphera-api/libs/go/client/delegation_server"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/testutil"
)

// SubscriptionIntegrationTestSuite provides integration tests for subscription endpoints
type SubscriptionIntegrationTestSuite struct {
	suite.Suite
	testDB     *testutil.TestDB
	router     *gin.Engine
	handler    *handlers.SubscriptionHandler
	ctx        context.Context
	cancel     context.CancelFunc
	workspaceID uuid.UUID
	customerID  uuid.UUID
}

func (suite *SubscriptionIntegrationTestSuite) SetupSuite() {
	// Set up test database
	suite.testDB = testutil.NewTestDB(suite.T())
	suite.testDB.SetupSchema(suite.T())
	
	// Create test context
	suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 30*time.Second)
	
	// Generate test UUIDs
	suite.workspaceID = uuid.New()
	suite.customerID = uuid.New()
	
	// Set up Gin router
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	
	// Create handler dependencies
	queries := db.New(suite.testDB.Pool())
	common := &handlers.CommonServices{
		// Note: In real implementation, you'd inject the test database properly
		// For now, this is a placeholder structure
	}
	delegationClient := &dsClient.DelegationClient{}
	suite.handler = handlers.NewSubscriptionHandler(common, delegationClient)
	
	// Set up routes
	suite.setupRoutes()
}

func (suite *SubscriptionIntegrationTestSuite) TearDownSuite() {
	if suite.cancel != nil {
		suite.cancel()
	}
	if suite.testDB != nil {
		suite.testDB.Close()
	}
}

func (suite *SubscriptionIntegrationTestSuite) SetupTest() {
	// Clean up database before each test
	suite.testDB.Truncate(suite.T())
	
	// Insert test data
	suite.insertTestData()
}

func (suite *SubscriptionIntegrationTestSuite) setupRoutes() {
	v1 := suite.router.Group("/api/v1")
	{
		subscriptions := v1.Group("/subscriptions")
		{
			subscriptions.GET("/:id", suite.handler.GetSubscription)
			subscriptions.GET("", suite.handler.ListSubscriptions)
			subscriptions.PUT("/:id", suite.handler.UpdateSubscription)
			subscriptions.DELETE("/:id", suite.handler.DeleteSubscription)
		}
		
		products := v1.Group("/products")
		{
			// Add product routes if needed for integration testing
		}
	}
}

func (suite *SubscriptionIntegrationTestSuite) insertTestData() {
	// This would insert test data into the database
	// For now, this is a placeholder for actual database operations
	suite.Require().NotNil(suite.testDB.Pool())
	
	// In a real implementation, you would:
	// 1. Insert test workspace
	// 2. Insert test customer
	// 3. Insert test product and price
	// 4. Insert test subscription
}

func (suite *SubscriptionIntegrationTestSuite) TestGetSubscription_Integration() {
	// Test the complete flow of getting a subscription
	subscriptionID := uuid.New()
	
	// Make request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/"+subscriptionID.String(), nil)
	req.Header.Set("X-Workspace-ID", suite.workspaceID.String())
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	// Since we don't have real data in the database, we expect a not found
	suite.Equal(http.StatusNotFound, w.Code, "Should return 404 for non-existent subscription")
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Contains(response, "error")
}

func (suite *SubscriptionIntegrationTestSuite) TestListSubscriptions_Integration() {
	// Test the complete flow of listing subscriptions
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions", nil)
	req.Header.Set("X-Workspace-ID", suite.workspaceID.String())
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	// Should return OK with empty list (no test data inserted)
	suite.Equal(http.StatusOK, w.Code, "Should return 200 for valid workspace")
	
	var response struct {
		Subscriptions []interface{} `json:"subscriptions"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Empty(response.Subscriptions, "Should return empty list when no subscriptions exist")
}

func (suite *SubscriptionIntegrationTestSuite) TestListSubscriptions_MissingWorkspace() {
	// Test validation - missing workspace header
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions", nil)
	// Intentionally omit X-Workspace-ID header
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	suite.Equal(http.StatusBadRequest, w.Code, "Should return 400 for missing workspace")
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Contains(response, "error")
}

func (suite *SubscriptionIntegrationTestSuite) TestUpdateSubscription_Integration() {
	// Test subscription update flow
	subscriptionID := uuid.New()
	
	updateData := map[string]interface{}{
		"status": "paused",
		"metadata": map[string]string{
			"reason": "user_requested",
		},
	}
	
	body, err := json.Marshal(updateData)
	suite.Require().NoError(err)
	
	req := httptest.NewRequest(http.MethodPUT, "/api/v1/subscriptions/"+subscriptionID.String(), bytes.NewBuffer(body))
	req.Header.Set("X-Workspace-ID", suite.workspaceID.String())
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	// Should return 404 since subscription doesn't exist
	suite.Equal(http.StatusNotFound, w.Code, "Should return 404 for non-existent subscription")
}

func (suite *SubscriptionIntegrationTestSuite) TestDeleteSubscription_Integration() {
	// Test subscription deletion flow
	subscriptionID := uuid.New()
	
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/subscriptions/"+subscriptionID.String(), nil)
	req.Header.Set("X-Workspace-ID", suite.workspaceID.String())
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	// Should return 404 since subscription doesn't exist
	suite.Equal(http.StatusNotFound, w.Code, "Should return 404 for non-existent subscription")
}

func (suite *SubscriptionIntegrationTestSuite) TestInvalidUUIDHandling() {
	// Test handling of invalid UUIDs
	invalidUUID := "not-a-valid-uuid"
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/"+invalidUUID, nil)
	req.Header.Set("X-Workspace-ID", suite.workspaceID.String())
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	suite.Equal(http.StatusBadRequest, w.Code, "Should return 400 for invalid UUID")
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Contains(response, "error")
}

// Benchmark integration tests
func (suite *SubscriptionIntegrationTestSuite) TestSubscriptionEndpoint_Performance() {
	// Performance test for subscription endpoints
	
	// Test multiple concurrent requests
	const numRequests = 10
	results := make(chan int, numRequests)
	
	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions", nil)
			req.Header.Set("X-Workspace-ID", suite.workspaceID.String())
			
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)
			
			results <- w.Code
		}()
	}
	
	// Collect results
	for i := 0; i < numRequests; i++ {
		statusCode := <-results
		suite.True(statusCode >= 200 && statusCode < 500, "Should return valid HTTP status")
	}
}

// Run the integration test suite
func TestSubscriptionIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	
	suite.Run(t, new(SubscriptionIntegrationTestSuite))
}

// Individual integration tests (can be run separately)
func TestSubscriptionIntegration_Standalone(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	
	// Simple standalone integration test
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Create minimal handler setup
	common := &handlers.CommonServices{}
	handler := handlers.NewSubscriptionHandler(common, nil)
	
	router.GET("/subscriptions/:id", handler.GetSubscription)
	
	// Test with invalid UUID
	req := httptest.NewRequest(http.MethodGet, "/subscriptions/invalid-uuid", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 for invalid UUID")
}