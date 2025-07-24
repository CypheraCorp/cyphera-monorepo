# Cyphera API Testing Guide

This guide provides comprehensive documentation for testing the Cyphera API, including patterns, examples, and best practices.

## Overview

The Cyphera API uses a multi-layered testing approach:

- **Unit Tests**: Test individual components in isolation
- **Integration Tests**: Test complete workflows with database
- **End-to-End Tests**: Test full API flows (planned)
- **Performance Tests**: Benchmark critical operations

## Testing Infrastructure

### Test Configuration

Configuration is managed through `test.config.json`:

```json
{
  "coverage": {
    "threshold": 60,
    "exclude_patterns": ["**/main.go", "**/gen/**", "**/*.pb.go"],
    "include_patterns": ["./apps/api/**", "./libs/go/**"]
  },
  "database": {
    "test_db_name": "cyphera_test",
    "test_db_host": "localhost",
    "test_db_port": 5432
  }
}
```

### Essential Commands

```bash
# Run all unit tests (includes mocked tests)
make test

# Run only mocked unit tests (fast)
make test-mock

# Run tests with coverage
make test-coverage

# Generate HTML coverage report
make test-coverage-html

# Run integration tests with real database
make test-integration

# Start/stop test database
make test-db-up
make test-db-down

# Generate mocks
make generate-mocks
```

## Testing Patterns

### 1. Database Testing Strategy

We use a hybrid approach for database testing:

#### Mocked Database Tests (Unit Tests)
- **Purpose**: Fast, isolated unit tests
- **Use Case**: Testing business logic without database dependencies
- **Benefits**: No I/O, deterministic, parallel-safe
- **Files**: Tests ending with `*_mock_test.go`

```go
func TestHandler_WithMocks(t *testing.T) {
    // Create mock database
    mockDB := testutil.NewMockDatabase(t)
    
    // Set expectations
    mockDB.ExpectSubscriptionExists(subscriptionID, &expectedSubscription)
    
    // Test with mocked database
    handler := NewHandler(&CommonServices{DB: mockDB.Querier})
    // ... test logic
}
```

#### Real Database Tests (Integration Tests)
- **Purpose**: End-to-end validation with actual database
- **Use Case**: Testing complex queries, transactions, constraints
- **Benefits**: Realistic behavior, schema validation
- **Files**: Tests in `tests/integration/` with `+build integration` tag

```go
// +build integration

func TestHandler_Integration(t *testing.T) {
    // Use real test database
    testDB := testutil.NewTestDB(t)
    defer testDB.Close()
    
    // Test with real database
    queries := db.New(testDB.Pool())
    // ... integration test logic
}
```

### 2. Unit Testing Handlers

#### Basic Handler Structure Tests

```go
func TestNewSubscriptionHandler_Creation(t *testing.T) {
    common := createTestCommonServices()
    delegationClient := createTestDelegationClient()
    
    handler := NewSubscriptionHandler(common, delegationClient)
    
    require.NotNil(t, handler)
    assert.Equal(t, common, handler.common)
    assert.Equal(t, delegationClient, handler.delegationClient)
}
```

#### Error Handling Tests

```go
func TestSubscriptionExistsError_Implementation(t *testing.T) {
    subscription := &db.Subscription{ID: testWorkspaceID}
    err := &SubscriptionExistsError{Subscription: subscription}
    
    expectedMsg := "subscription already exists with ID: " + testWorkspaceID.String()
    assert.Equal(t, expectedMsg, err.Error())
    
    // Test error interface compliance
    var _ error = err
}
```

#### Request Validation Tests

```go
func TestAccountHandler_RequestStructures(t *testing.T) {
    testCases := []struct {
        name        string
        requestBody map[string]interface{}
        expectValid bool
    }{
        {
            name: "valid sign in request",
            requestBody: map[string]interface{}{
                "token":    "valid.jwt.token",
                "provider": "web3auth",
            },
            expectValid: true,
        },
        {
            name: "missing token",
            requestBody: map[string]interface{}{
                "provider": "web3auth",
            },
            expectValid: false,
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            jsonData, err := json.Marshal(tc.requestBody)
            require.NoError(t, err)
            
            // Validation logic here
        })
    }
}
```

### 2. Security Testing

#### API Key Security

```go
func TestAPIKeyHandler_HashingValidation(t *testing.T) {
    testKey := "cyphera_" + hex.EncodeToString([]byte("test_key_data"))
    
    // Hash the key
    hashedKey, err := bcrypt.GenerateFromPassword([]byte(testKey), bcrypt.DefaultCost)
    require.NoError(t, err)
    
    // Verify the hash
    err = bcrypt.CompareHashAndPassword(hashedKey, []byte(testKey))
    assert.NoError(t, err, "Hash verification should succeed")
    
    // Verify wrong key fails
    wrongKey := "wrong_key"
    err = bcrypt.CompareHashAndPassword(hashedKey, []byte(wrongKey))
    assert.Error(t, err, "Hash verification should fail for wrong key")
}
```

#### Key Exposure Prevention

```go
func TestAPIKeyHandler_SecurityPatterns(t *testing.T) {
    rawKey := "cyphera_secret_key"
    hashedKey, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
    
    response := struct {
        ID        uuid.UUID `json:"id"`
        KeyPrefix string    `json:"key_prefix"`
        HashedKey string    `json:"-"` // Never exposed
    }{
        ID:        uuid.New(),
        KeyPrefix: rawKey[:12] + "...",
        HashedKey: string(hashedKey),
    }
    
    jsonData, _ := json.Marshal(response)
    jsonString := string(jsonData)
    
    assert.NotContains(t, jsonString, rawKey, "Raw key should not appear in JSON")
    assert.NotContains(t, jsonString, string(hashedKey), "Hashed key should not appear in JSON")
}
```

### 3. Integration Testing

#### Database Integration

```go
type SubscriptionIntegrationTestSuite struct {
    suite.Suite
    testDB     *testutil.TestDB
    router     *gin.Engine
    handler    *handlers.SubscriptionHandler
    workspaceID uuid.UUID
}

func (suite *SubscriptionIntegrationTestSuite) SetupSuite() {
    suite.testDB = testutil.NewTestDB(suite.T())
    suite.testDB.SetupSchema(suite.T())
    
    // Set up test dependencies
    queries := db.New(suite.testDB.Pool())
    common := &handlers.CommonServices{
        // Inject test database
    }
    suite.handler = handlers.NewSubscriptionHandler(common, nil)
    
    // Set up routes
    suite.setupRoutes()
}

func (suite *SubscriptionIntegrationTestSuite) SetupTest() {
    // Clean database before each test
    suite.testDB.Truncate(suite.T())
    suite.insertTestData()
}
```

#### HTTP Endpoint Testing

```go
func (suite *SubscriptionIntegrationTestSuite) TestGetSubscription_Integration() {
    subscriptionID := uuid.New()
    
    req := httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/"+subscriptionID.String(), nil)
    req.Header.Set("X-Workspace-ID", suite.workspaceID.String())
    
    w := httptest.NewRecorder()
    suite.router.ServeHTTP(w, req)
    
    suite.Equal(http.StatusNotFound, w.Code)
    
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    suite.Require().NoError(err)
    suite.Contains(response, "error")
}
```

### 4. Database Mock Usage

#### Using Database Mock Utilities

The `testutil.MockDatabase` provides fluent helper methods for common database operations:

```go
func TestWithDatabaseMocks(t *testing.T) {
    mockDB := testutil.NewMockDatabase(t)
    workspaceID := uuid.New()
    
    // Fluent API for setting expectations
    subscription := testutil.CreateTestSubscription(workspaceID, uuid.New())
    mockDB.ExpectSubscriptionExists(subscriptionID, &subscription)
    
    // Chain expectations for complex operations
    mockDB.ExpectAPIKeyOperations().
        ExpectCreate(apiKey).
        ExpectList(workspaceID, []db.ApiKey{apiKey})
    
    // Use in handler
    handler := NewHandler(&CommonServices{DB: mockDB.Querier})
}
```

#### Common Mock Patterns

```go
// Expect entity exists
mockDB.ExpectSubscriptionExists(id, &subscription)  // Found
mockDB.ExpectSubscriptionExists(id, nil)           // Not found

// Expect list operations
mockDB.ExpectListSubscriptions(workspaceID, subscriptions)

// Expect CRUD operations
mockDB.ExpectCreateSubscription(subscription)
mockDB.ExpectUpdateSubscription(id, updatedSubscription)
mockDB.ExpectDeleteSubscription(id)

// API Key operations (fluent interface)
mockDB.ExpectAPIKeyOperations().
    ExpectGet(keyID, &apiKey).
    ExpectDelete(keyID)
```

#### Test Data Helpers

```go
// Create realistic test data
subscription := testutil.CreateTestSubscription(workspaceID, customerID)
apiKey := testutil.CreateTestAPIKey(workspaceID, "Test Key")
workspace := testutil.CreateTestWorkspace("Test Workspace")

// All test data includes proper UUIDs, timestamps, and relationships
```

### 5. External Service Mocks

#### Using Generated Mocks

```go
func TestWithMocks(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    
    // Create mock
    mockCircleClient := mocks.NewMockCircleClientInterface(ctrl)
    
    // Set expectations
    mockCircleClient.EXPECT().
        CreateUserWithPinAuth(gomock.Any(), "external-user-123").
        Return(&circle.UserResponse{}, nil).
        Times(1)
    
    // Use mock in test
    response, err := mockCircleClient.CreateUserWithPinAuth(context.Background(), "external-user-123")
    assert.NoError(t, err)
    assert.NotNil(t, response)
}
```

#### Helper Functions

```go
// Helper to create mocks with proper cleanup
func NewMockCircleClientForTest(t *testing.T) *mocks.MockCircleClientInterface {
    ctrl := gomock.NewController(t)
    t.Cleanup(ctrl.Finish)
    return mocks.NewMockCircleClientInterface(ctrl)
}
```

## Best Practices

### 1. Test Organization

```
handlers/
├── handlers_test_base.go          # Common test utilities
├── subscription_handlers_test.go  # Subscription handler tests
├── account_handlers_test.go       # Account handler tests
└── apikey_handlers_test.go       # API key handler tests

tests/
└── integration/
    └── subscription_integration_test.go
```

### 2. Test Data Management

```go
// Use consistent test UUIDs
var (
    testWorkspaceID  = uuid.MustParse("01234567-89ab-cdef-0123-456789abcdef")
    testCustomerID   = uuid.MustParse("11234567-89ab-cdef-0123-456789abcdef")
    testProductID    = uuid.MustParse("21234567-89ab-cdef-0123-456789abcdef")
)

// Create test fixtures
func createTestSubscription() db.Subscription {
    return db.Subscription{
        ID:          uuid.New(),
        CustomerID:  testCustomerID,
        WorkspaceID: testWorkspaceID,
        Status:      db.SubscriptionStatusActive,
        // ... other required fields
    }
}
```

### 3. Error Testing

```go
func TestErrorHandling(t *testing.T) {
    tests := []struct {
        name           string
        input          interface{}
        expectedStatus int
        expectedError  string
    }{
        {"invalid UUID", "not-a-uuid", http.StatusBadRequest, "Invalid UUID format"},
        {"missing workspace", nil, http.StatusBadRequest, "Missing workspace header"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 4. Performance Testing

```go
func BenchmarkCriticalOperation(b *testing.B) {
    setup := createBenchmarkSetup()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        result := performOperation(setup)
        _ = result
    }
}
```

## Database Testing

### Using TestDB Utility

```go
func TestWithDatabase(t *testing.T) {
    // Create test database connection
    testDB := testutil.NewTestDB(t)
    defer testDB.Close()
    
    // Clean up tables
    testDB.Truncate(t, "subscriptions", "customers")
    
    // Use in transaction (auto-rollback)
    testDB.WithTransaction(t, func(pool *pgxpool.Pool) {
        // Database operations here
        // Will be automatically rolled back
    })
}
```

### Schema Management

```go
func TestDatabaseSchema(t *testing.T) {
    testDB := testutil.NewTestDB(t)
    defer testDB.Close()
    
    // Ensure schema is set up
    testDB.SetupSchema(t)
    
    // Verify required tables exist
    ctx := context.Background()
    var exists bool
    err := testDB.Pool().QueryRow(ctx, 
        "SELECT EXISTS(SELECT FROM pg_tables WHERE tablename = 'subscriptions')",
    ).Scan(&exists)
    
    require.NoError(t, err)
    assert.True(t, exists, "subscriptions table should exist")
}
```

## CI/CD Integration

### GitHub Actions Configuration

The project includes comprehensive GitHub Actions workflows:

- **Unit Tests**: Fast feedback on basic functionality
- **Integration Tests**: Database and service integration
- **Coverage Reports**: Ensure adequate test coverage
- **Security Scans**: Automated security analysis
- **Linting**: Code quality enforcement

### Local Testing with Act

```bash
# Install act CLI for local GitHub Actions testing
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Run tests locally
act -j unit-tests
act -j integration-tests --secret-file .secrets
```

## Coverage Requirements

- **Minimum Coverage**: 60% (configurable)
- **Critical Handlers**: Aim for 80%+ coverage
- **Security Components**: 90%+ coverage
- **Exclusions**: Generated code, main files, protobuf files

### Coverage Analysis

```bash
# Generate coverage report
make test-coverage

# View coverage by function
go tool cover -func=coverage.out

# Generate HTML report
make test-coverage-html
open coverage.html
```

## Troubleshooting

### Common Issues

1. **Database Connection Issues**
   ```bash
   # Ensure test database is running
   make test-db-up
   
   # Check connection
   psql "postgres://postgres:postgres@localhost:5433/cyphera_test"
   ```

2. **Mock Generation Issues**
   ```bash
   # Regenerate mocks
   make generate-mocks
   
   # Verify mock interfaces
   go build ./mocks/...
   ```

3. **Race Conditions**
   ```bash
   # Run with race detection
   go test -race ./...
   
   # Use proper synchronization in tests
   ```

## Examples Repository

For more examples, see:
- `handlers/subscription_unit_test.go` - Comprehensive unit testing
- `handlers/apikey_handlers_test.go` - Security testing patterns
- `tests/integration/` - Integration testing examples
- `libs/go/testutil/` - Testing utilities and helpers

## Contributing

When adding new tests:

1. Follow existing patterns and naming conventions
2. Include both positive and negative test cases
3. Test error conditions and edge cases
4. Add benchmark tests for performance-critical code
5. Update documentation for new testing patterns
6. Ensure tests are deterministic and can run in parallel

For questions or contributions to the testing infrastructure, please refer to the main project documentation or create an issue.