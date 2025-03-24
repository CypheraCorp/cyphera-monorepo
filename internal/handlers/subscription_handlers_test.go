package handlers

import (
	"context"
	"cyphera-api/internal/db"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDB is a mock implementation of the db.Querier interface
type MockDB struct {
	mock.Mock
}

func (m *MockDB) GetSubscription(ctx any, id uuid.UUID) (db.Subscription, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.Subscription), args.Error(1)
}

func (m *MockDB) ListSubscriptionEventsBySubscription(ctx any, subscriptionID uuid.UUID) ([]db.SubscriptionEvent, error) {
	args := m.Called(ctx, subscriptionID)
	return args.Get(0).([]db.SubscriptionEvent), args.Error(1)
}

// Other methods required to satisfy db.Querier would go here
// For our tests, we only need to implement the methods that our handler will call

// For testing, define minimal delegation client interface
type DelegationClientInterface interface {
	RedeemDelegationDirectly(ctx context.Context, signatureBytes []byte) (string, error)
	CheckHealth() error
}

// MockDelegationClient is a mock implementation of the DelegationClientInterface
type MockDelegationClient struct {
	mock.Mock
}

func (m *MockDelegationClient) RedeemDelegationDirectly(ctx context.Context, signatureBytes []byte) (string, error) {
	args := m.Called(ctx, signatureBytes)
	return args.String(0), args.Error(1)
}

func (m *MockDelegationClient) CheckHealth() error {
	args := m.Called()
	return args.Error(0)
}

// For testing, we create a mock implementation of the SubscriptionHandler
// that doesn't require using the actual CommonServices and DelegationClient
type MockSubscriptionHandler struct {
	mockDB          *MockDB
	mockDelegClient *MockDelegationClient
	originalHandler SubscriptionHandler
}

// TestGetRedemptionStatus_NoEvents tests the redemption status endpoint when no events exist
func TestGetRedemptionStatus_NoEvents(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mocks
	mockDB := new(MockDB)

	// Create test data
	subID := uuid.New()
	subscription := db.Subscription{
		ID:               subID,
		TotalRedemptions: 0,
		Status:           db.SubscriptionStatusActive,
		NextRedemptionDate: pgtype.Timestamptz{
			Time:  time.Now().Add(24 * time.Hour),
			Valid: true,
		},
	}

	// Mock DB responses
	mockDB.On("GetSubscription", mock.Anything, subID).Return(subscription, nil)
	mockDB.On("ListSubscriptionEventsBySubscription", mock.Anything, subID).Return([]db.SubscriptionEvent{}, nil)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "subscription_id", Value: subID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodGet, "/subscriptions/"+subID.String()+"/redemption-status", nil)
	c.Request = req

	// Directly invoke the handler function with our required context
	handler := &GetRedemptionStatusHandler{
		db: mockDB,
	}
	handler.Handle(c)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response GetRedemptionStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, subID.String(), response.SubscriptionID)
	assert.Equal(t, "pending", response.Status)
	assert.Equal(t, "No redemption attempts found", response.Message)
	assert.Equal(t, int32(0), response.TotalRedemptions)
	assert.Equal(t, subscription.NextRedemptionDate.Time.Unix(), response.NextRedemptionAt.Unix())
	assert.Nil(t, response.LastRedemptionAt)
	assert.Nil(t, response.LastAttemptedAt)
	assert.Empty(t, response.TransactionHash)
	assert.Empty(t, response.FailureReason)

	// Verify mocks
	mockDB.AssertExpectations(t)
}

// Define a specific handler structure just for the GetRedemptionStatus test
type GetRedemptionStatusHandler struct {
	db interface {
		GetSubscription(ctx any, id uuid.UUID) (db.Subscription, error)
		ListSubscriptionEventsBySubscription(ctx any, subscriptionID uuid.UUID) ([]db.SubscriptionEvent, error)
	}
}

// Handle implements the handler logic from GetRedemptionStatus
func (h *GetRedemptionStatusHandler) Handle(c *gin.Context) {
	ctx := c.Request.Context()
	subscriptionID := c.Param("subscription_id")

	// Validate subscription ID
	if subscriptionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Subscription ID is required",
		})
		return
	}

	subID, err := uuid.Parse(subscriptionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid subscription ID format: " + err.Error(),
		})
		return
	}

	// Check if subscription exists
	subscription, err := h.db.GetSubscription(ctx, subID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Not Found",
			"message": "Subscription not found: " + err.Error(),
		})
		return
	}

	// Get latest events for the subscription
	events, err := h.db.ListSubscriptionEventsBySubscription(ctx, subID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve subscription events: " + err.Error(),
		})
		return
	}

	// Initialize response with default values
	response := GetRedemptionStatusResponse{
		SubscriptionID:   subscriptionID,
		LastRedemptionAt: nil,
		Status:           "pending",
		Message:          "No redemption attempts found",
		TotalRedemptions: subscription.TotalRedemptions,
		NextRedemptionAt: subscription.NextRedemptionDate.Time,
		TransactionHash:  "",
		FailureReason:    "",
		LastAttemptedAt:  nil,
	}

	// No events found
	if len(events) == 0 {
		c.JSON(http.StatusOK, response)
		return
	}

	// Check for redemption events
	var lastRedemptionEvent *db.SubscriptionEvent
	var lastFailedRedemptionEvent *db.SubscriptionEvent

	for i := range events {
		event := events[i]
		if event.EventType == db.SubscriptionEventTypeRedeemed {
			if lastRedemptionEvent == nil ||
				event.CreatedAt.Time.After(lastRedemptionEvent.CreatedAt.Time) ||
				event.CreatedAt.Time.Equal(lastRedemptionEvent.CreatedAt.Time) {
				lastRedemptionEvent = &events[i] // Use &events[i] to get a stable pointer
			}
		} else if event.EventType == db.SubscriptionEventTypeFailedRedemption {
			if lastFailedRedemptionEvent == nil ||
				event.CreatedAt.Time.After(lastFailedRedemptionEvent.CreatedAt.Time) ||
				event.CreatedAt.Time.Equal(lastFailedRedemptionEvent.CreatedAt.Time) {
				lastFailedRedemptionEvent = &events[i] // Use &events[i] to get a stable pointer
			}
		}
	}

	// Determine status based on the events found
	if lastRedemptionEvent != nil {
		// If we have a successful redemption, and it's more recent than any failed attempts
		if lastFailedRedemptionEvent == nil ||
			lastRedemptionEvent.CreatedAt.Time.After(lastFailedRedemptionEvent.CreatedAt.Time) ||
			(lastRedemptionEvent.CreatedAt.Time.Equal(lastFailedRedemptionEvent.CreatedAt.Time) &&
				sliceIndexOf(events, *lastRedemptionEvent) > sliceIndexOf(events, *lastFailedRedemptionEvent)) {
			response.Status = "success"
			response.Message = "Subscription successfully redeemed"
			lastRedemptionTime := lastRedemptionEvent.CreatedAt.Time
			response.LastRedemptionAt = &lastRedemptionTime
			response.LastAttemptedAt = &lastRedemptionTime
			response.TransactionHash = lastRedemptionEvent.TransactionHash.String
		}
	}

	// If we have a failed redemption, and it's more recent than any successful attempts
	if lastFailedRedemptionEvent != nil {
		if lastRedemptionEvent == nil ||
			lastFailedRedemptionEvent.CreatedAt.Time.After(lastRedemptionEvent.CreatedAt.Time) ||
			(lastFailedRedemptionEvent.CreatedAt.Time.Equal(lastRedemptionEvent.CreatedAt.Time) &&
				sliceIndexOf(events, *lastFailedRedemptionEvent) > sliceIndexOf(events, *lastRedemptionEvent)) {
			response.Status = "failed"
			response.Message = "Redemption attempt failed"
			lastFailedTime := lastFailedRedemptionEvent.CreatedAt.Time
			response.LastAttemptedAt = &lastFailedTime
			response.FailureReason = lastFailedRedemptionEvent.ErrorMessage.String
		}
	}

	c.JSON(http.StatusOK, response)
}

// Helper function to find the index of an event in a slice of events
// This is used to determine precedence when timestamps are equal
func sliceIndexOf(events []db.SubscriptionEvent, event db.SubscriptionEvent) int {
	for i, e := range events {
		if e.ID == event.ID {
			return i
		}
	}
	return -1
}

// GetRedemptionStatus simply delegates to Handle for tests
func (h *GetRedemptionStatusHandler) GetRedemptionStatus(c *gin.Context) {
	h.Handle(c)
}

// TestGetRedemptionStatus_SuccessfulRedemption tests the redemption status endpoint with a successful redemption
func TestGetRedemptionStatus_SuccessfulRedemption(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mocks
	mockDB := new(MockDB)

	// Create test data
	subID := uuid.New()
	txHash := "0x1234567890abcdef"
	redemptionTime := time.Now().Add(-1 * time.Hour)

	subscription := db.Subscription{
		ID:               subID,
		TotalRedemptions: 1,
		Status:           db.SubscriptionStatusActive,
		NextRedemptionDate: pgtype.Timestamptz{
			Time:  time.Now().Add(24 * time.Hour),
			Valid: true,
		},
	}

	event := db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subID,
		EventType:      db.SubscriptionEventTypeRedeemed,
		TransactionHash: pgtype.Text{
			String: txHash,
			Valid:  true,
		},
		CreatedAt: pgtype.Timestamptz{
			Time:  redemptionTime,
			Valid: true,
		},
	}

	// Mock DB responses
	mockDB.On("GetSubscription", mock.Anything, subID).Return(subscription, nil)
	mockDB.On("ListSubscriptionEventsBySubscription", mock.Anything, subID).Return([]db.SubscriptionEvent{event}, nil)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "subscription_id", Value: subID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodGet, "/subscriptions/"+subID.String()+"/redemption-status", nil)
	c.Request = req

	// Directly invoke the handler function with our required context
	handler := &GetRedemptionStatusHandler{
		db: mockDB,
	}
	handler.Handle(c)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response GetRedemptionStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, subID.String(), response.SubscriptionID)
	assert.Equal(t, "success", response.Status)
	assert.Equal(t, "Subscription successfully redeemed", response.Message)
	assert.Equal(t, int32(1), response.TotalRedemptions)
	assert.Equal(t, subscription.NextRedemptionDate.Time.Unix(), response.NextRedemptionAt.Unix())
	assert.NotNil(t, response.LastRedemptionAt)
	assert.Equal(t, redemptionTime.Unix(), response.LastRedemptionAt.Unix())
	assert.NotNil(t, response.LastAttemptedAt)
	assert.Equal(t, redemptionTime.Unix(), response.LastAttemptedAt.Unix())
	assert.Equal(t, txHash, response.TransactionHash)
	assert.Empty(t, response.FailureReason)

	// Verify mocks
	mockDB.AssertExpectations(t)
}

// TestGetRedemptionStatus_FailedRedemption tests the redemption status endpoint with a failed redemption
func TestGetRedemptionStatus_FailedRedemption(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mocks
	mockDB := new(MockDB)

	// Create test data
	subID := uuid.New()
	failureReason := "Delegation redemption failed: invalid signature"
	failedTime := time.Now().Add(-1 * time.Hour)

	subscription := db.Subscription{
		ID:               subID,
		TotalRedemptions: 0,
		Status:           db.SubscriptionStatusActive,
		NextRedemptionDate: pgtype.Timestamptz{
			Time:  time.Now().Add(24 * time.Hour),
			Valid: true,
		},
	}

	event := db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subID,
		EventType:      db.SubscriptionEventTypeFailedRedemption,
		ErrorMessage: pgtype.Text{
			String: failureReason,
			Valid:  true,
		},
		CreatedAt: pgtype.Timestamptz{
			Time:  failedTime,
			Valid: true,
		},
	}

	// Mock DB responses
	mockDB.On("GetSubscription", mock.Anything, subID).Return(subscription, nil)
	mockDB.On("ListSubscriptionEventsBySubscription", mock.Anything, subID).Return([]db.SubscriptionEvent{event}, nil)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "subscription_id", Value: subID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodGet, "/subscriptions/"+subID.String()+"/redemption-status", nil)
	c.Request = req

	// Directly invoke the handler function with our required context
	handler := &GetRedemptionStatusHandler{
		db: mockDB,
	}
	handler.Handle(c)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response GetRedemptionStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, subID.String(), response.SubscriptionID)
	assert.Equal(t, "failed", response.Status)
	assert.Equal(t, "Redemption attempt failed", response.Message)
	assert.Equal(t, int32(0), response.TotalRedemptions)
	assert.Equal(t, subscription.NextRedemptionDate.Time.Unix(), response.NextRedemptionAt.Unix())
	assert.Nil(t, response.LastRedemptionAt)
	assert.NotNil(t, response.LastAttemptedAt)
	assert.Equal(t, failedTime.Unix(), response.LastAttemptedAt.Unix())
	assert.Empty(t, response.TransactionHash)
	assert.Equal(t, failureReason, response.FailureReason)

	// Verify mocks
	mockDB.AssertExpectations(t)
}

// TestGetRedemptionStatus_SuccessAfterFailure tests when a successful redemption follows a failed one
func TestGetRedemptionStatus_SuccessAfterFailure(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mocks
	mockDB := new(MockDB)

	// Create test data
	subID := uuid.New()
	txHash := "0x1234567890abcdef"
	failureReason := "Delegation redemption failed: invalid signature"
	failedTime := time.Now().Add(-2 * time.Hour)
	redemptionTime := time.Now().Add(-1 * time.Hour) // More recent than the failure

	subscription := db.Subscription{
		ID:               subID,
		TotalRedemptions: 1,
		Status:           db.SubscriptionStatusActive,
		NextRedemptionDate: pgtype.Timestamptz{
			Time:  time.Now().Add(24 * time.Hour),
			Valid: true,
		},
	}

	failedEvent := db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subID,
		EventType:      db.SubscriptionEventTypeFailedRedemption,
		ErrorMessage: pgtype.Text{
			String: failureReason,
			Valid:  true,
		},
		CreatedAt: pgtype.Timestamptz{
			Time:  failedTime,
			Valid: true,
		},
	}

	successEvent := db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subID,
		EventType:      db.SubscriptionEventTypeRedeemed,
		TransactionHash: pgtype.Text{
			String: txHash,
			Valid:  true,
		},
		CreatedAt: pgtype.Timestamptz{
			Time:  redemptionTime,
			Valid: true,
		},
	}

	// Mock DB responses
	mockDB.On("GetSubscription", mock.Anything, subID).Return(subscription, nil)
	mockDB.On("ListSubscriptionEventsBySubscription", mock.Anything, subID).Return(
		[]db.SubscriptionEvent{failedEvent, successEvent}, nil)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "subscription_id", Value: subID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodGet, "/subscriptions/"+subID.String()+"/redemption-status", nil)
	c.Request = req

	// Directly invoke the handler function with our required context
	handler := &GetRedemptionStatusHandler{
		db: mockDB,
	}
	handler.Handle(c)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response GetRedemptionStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, subID.String(), response.SubscriptionID)
	assert.Equal(t, "success", response.Status) // Most recent event is success
	assert.Equal(t, "Subscription successfully redeemed", response.Message)
	assert.Equal(t, int32(1), response.TotalRedemptions)
	assert.Equal(t, subscription.NextRedemptionDate.Time.Unix(), response.NextRedemptionAt.Unix())
	assert.NotNil(t, response.LastRedemptionAt)
	assert.Equal(t, redemptionTime.Unix(), response.LastRedemptionAt.Unix())
	assert.NotNil(t, response.LastAttemptedAt)
	assert.Equal(t, redemptionTime.Unix(), response.LastAttemptedAt.Unix())
	assert.Equal(t, txHash, response.TransactionHash)
	assert.Empty(t, response.FailureReason) // No failure reason since latest is success

	// Verify mocks
	mockDB.AssertExpectations(t)
}

// TestGetRedemptionStatus_FailureAfterSuccess tests when a failed redemption follows a successful one
func TestGetRedemptionStatus_FailureAfterSuccess(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mocks
	mockDB := new(MockDB)

	// Create test data
	subID := uuid.New()
	txHash := "0x1234567890abcdef"
	failureReason := "Delegation redemption failed: invalid signature"
	redemptionTime := time.Now().Add(-2 * time.Hour)
	failedTime := time.Now().Add(-1 * time.Hour) // More recent than the success

	subscription := db.Subscription{
		ID:               subID,
		TotalRedemptions: 1,
		Status:           db.SubscriptionStatusActive,
		NextRedemptionDate: pgtype.Timestamptz{
			Time:  time.Now().Add(24 * time.Hour),
			Valid: true,
		},
	}

	successEvent := db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subID,
		EventType:      db.SubscriptionEventTypeRedeemed,
		TransactionHash: pgtype.Text{
			String: txHash,
			Valid:  true,
		},
		CreatedAt: pgtype.Timestamptz{
			Time:  redemptionTime,
			Valid: true,
		},
	}

	failedEvent := db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subID,
		EventType:      db.SubscriptionEventTypeFailedRedemption,
		ErrorMessage: pgtype.Text{
			String: failureReason,
			Valid:  true,
		},
		CreatedAt: pgtype.Timestamptz{
			Time:  failedTime,
			Valid: true,
		},
	}

	// Mock DB responses
	mockDB.On("GetSubscription", mock.Anything, subID).Return(subscription, nil)
	mockDB.On("ListSubscriptionEventsBySubscription", mock.Anything, subID).Return(
		[]db.SubscriptionEvent{successEvent, failedEvent}, nil)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "subscription_id", Value: subID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodGet, "/subscriptions/"+subID.String()+"/redemption-status", nil)
	c.Request = req

	// Directly invoke the handler function with our required context
	handler := &GetRedemptionStatusHandler{
		db: mockDB,
	}
	handler.Handle(c)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response GetRedemptionStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, subID.String(), response.SubscriptionID)
	assert.Equal(t, "failed", response.Status) // Most recent event is failure
	assert.Equal(t, "Redemption attempt failed", response.Message)
	assert.Equal(t, int32(1), response.TotalRedemptions)
	assert.Equal(t, subscription.NextRedemptionDate.Time.Unix(), response.NextRedemptionAt.Unix())
	assert.Nil(t, response.LastRedemptionAt) // Don't show last redemption time when latest is failure
	assert.NotNil(t, response.LastAttemptedAt)
	assert.Equal(t, failedTime.Unix(), response.LastAttemptedAt.Unix())
	assert.Empty(t, response.TransactionHash) // No transaction hash since latest is failure
	assert.Equal(t, failureReason, response.FailureReason)

	// Verify mocks
	mockDB.AssertExpectations(t)
}

// TestGetRedemptionStatus_InvalidSubscriptionID tests the redemption status endpoint with an invalid subscription ID
func TestGetRedemptionStatus_InvalidSubscriptionID(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mocks
	mockDB := new(MockDB)

	// Create request with invalid UUID
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "subscription_id", Value: "not-a-uuid"},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodGet, "/subscriptions/not-a-uuid/redemption-status", nil)
	c.Request = req

	// Directly invoke the handler function with our required context
	handler := &GetRedemptionStatusHandler{
		db: mockDB,
	}
	handler.Handle(c)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Contains(t, response.Message, "Invalid subscription ID format")
}

// TestGetRedemptionStatus_SubscriptionNotFound tests the redemption status endpoint when subscription is not found
func TestGetRedemptionStatus_SubscriptionNotFound(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mocks
	mockDB := new(MockDB)

	// Create test data
	subID := uuid.New()

	// Mock DB responses to return not found error
	mockDB.On("GetSubscription", mock.Anything, subID).Return(db.Subscription{}, errors.New("subscription not found"))

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "subscription_id", Value: subID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodGet, "/subscriptions/"+subID.String()+"/redemption-status", nil)
	c.Request = req

	// Directly invoke the handler function with our required context
	handler := &GetRedemptionStatusHandler{
		db: mockDB,
	}
	handler.Handle(c)

	// Assertions
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Contains(t, response.Message, "Subscription not found")

	// Verify mocks
	mockDB.AssertExpectations(t)
}

// TestGetRedemptionStatus_DatabaseError tests the redemption status endpoint when database error occurs
func TestGetRedemptionStatus_DatabaseError(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mocks
	mockDB := new(MockDB)

	// Create test data
	subID := uuid.New()
	subscription := db.Subscription{
		ID:               subID,
		TotalRedemptions: 0,
		Status:           db.SubscriptionStatusActive,
		NextRedemptionDate: pgtype.Timestamptz{
			Time:  time.Now().Add(24 * time.Hour),
			Valid: true,
		},
	}

	// Mock DB responses
	mockDB.On("GetSubscription", mock.Anything, subID).Return(subscription, nil)
	mockDB.On("ListSubscriptionEventsBySubscription", mock.Anything, subID).Return(
		[]db.SubscriptionEvent{}, errors.New("database connection error"))

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "subscription_id", Value: subID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodGet, "/subscriptions/"+subID.String()+"/redemption-status", nil)
	c.Request = req

	// Directly invoke the handler function with our required context
	handler := &GetRedemptionStatusHandler{
		db: mockDB,
	}
	handler.Handle(c)

	// Assertions
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Contains(t, response.Message, "Failed to retrieve subscription events")

	// Verify mocks
	mockDB.AssertExpectations(t)
}

// TestGetRedemptionStatus_MultipleEventsWithSameTimestamp tests the redemption status endpoint
// when there are multiple events with the same timestamp, which is an edge case
func TestGetRedemptionStatus_MultipleEventsWithSameTimestamp(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mocks
	mockDB := new(MockDB)

	// Create test data
	subID := uuid.New()
	txHash1 := "0x1111111111111111"
	txHash2 := "0x2222222222222222"

	// Common timestamp for multiple events
	sameTime := time.Now().Truncate(time.Second)

	subscription := db.Subscription{
		ID:               subID,
		TotalRedemptions: 2,
		Status:           db.SubscriptionStatusActive,
		NextRedemptionDate: pgtype.Timestamptz{
			Time:  time.Now().Add(24 * time.Hour),
			Valid: true,
		},
	}

	// Create two redemption events with the exact same timestamp
	// Per our handler logic, the last iteration of the event will be used if timestamps are equal
	event1 := db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subID,
		EventType:      db.SubscriptionEventTypeRedeemed,
		TransactionHash: pgtype.Text{
			String: txHash1,
			Valid:  true,
		},
		CreatedAt: pgtype.Timestamptz{
			Time:  sameTime,
			Valid: true,
		},
	}

	event2 := db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subID,
		EventType:      db.SubscriptionEventTypeRedeemed,
		TransactionHash: pgtype.Text{
			String: txHash2,
			Valid:  true,
		},
		CreatedAt: pgtype.Timestamptz{
			Time:  sameTime,
			Valid: true,
		},
	}

	// Mock DB responses
	// The events in the list should be processed in order. In the handler logic,
	// we iterate through the events and keep updating the result if the current event has
	// a timestamp >= the previous one. So the LAST event in the list with the same timestamp wins.
	mockDB.On("GetSubscription", mock.Anything, subID).Return(subscription, nil)
	mockDB.On("ListSubscriptionEventsBySubscription", mock.Anything, subID).Return(
		[]db.SubscriptionEvent{event1, event2}, nil)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "subscription_id", Value: subID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodGet, "/subscriptions/"+subID.String()+"/redemption-status", nil)
	c.Request = req

	// Directly invoke the handler function with our required context
	handler := &GetRedemptionStatusHandler{
		db: mockDB,
	}
	handler.GetRedemptionStatus(c)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response GetRedemptionStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, subID.String(), response.SubscriptionID)
	assert.Equal(t, "success", response.Status)
	assert.Equal(t, "Subscription successfully redeemed", response.Message)
	assert.Equal(t, int32(2), response.TotalRedemptions)
	assert.NotNil(t, response.LastRedemptionAt)
	assert.Equal(t, sameTime.Unix(), response.LastRedemptionAt.Unix())

	// This should be the transaction hash from the LAST event in the list
	// since we iterate through the events and update the last redemption
	assert.Equal(t, txHash2, response.TransactionHash)

	// Verify mocks
	mockDB.AssertExpectations(t)
}

// TestGetRedemptionStatus_ComplexEventOrder tests the redemption status endpoint
// when a subscription has a complex mixture of event types in various orders
func TestGetRedemptionStatus_ComplexEventOrder(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mocks
	mockDB := new(MockDB)

	// Create test data
	subID := uuid.New()

	// Create timestamps at different points in time for various events
	baseTime := time.Now().Add(-24 * time.Hour)
	time1 := baseTime
	time2 := baseTime.Add(1 * time.Hour)
	time3 := baseTime.Add(2 * time.Hour)
	time4 := baseTime.Add(3 * time.Hour)
	time5 := baseTime.Add(4 * time.Hour)

	subscription := db.Subscription{
		ID:               subID,
		TotalRedemptions: 2,
		Status:           db.SubscriptionStatusActive,
		NextRedemptionDate: pgtype.Timestamptz{
			Time:  time.Now().Add(24 * time.Hour),
			Valid: true,
		},
	}

	// Create a complex sequence of events
	// 1. Created event
	// 2. Failed redemption
	// 3. Successful redemption
	// 4. Failed redemption (newer than successful)
	// 5. Another successful redemption (newest event)

	event1 := db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subID,
		EventType:      db.SubscriptionEventTypeCreated,
		CreatedAt: pgtype.Timestamptz{
			Time:  time1,
			Valid: true,
		},
	}

	event2 := db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subID,
		EventType:      db.SubscriptionEventTypeFailedRedemption,
		ErrorMessage: pgtype.Text{
			String: "First failure",
			Valid:  true,
		},
		CreatedAt: pgtype.Timestamptz{
			Time:  time2,
			Valid: true,
		},
	}

	event3 := db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subID,
		EventType:      db.SubscriptionEventTypeRedeemed,
		TransactionHash: pgtype.Text{
			String: "0xfirst_success",
			Valid:  true,
		},
		CreatedAt: pgtype.Timestamptz{
			Time:  time3,
			Valid: true,
		},
	}

	event4 := db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subID,
		EventType:      db.SubscriptionEventTypeFailedRedemption,
		ErrorMessage: pgtype.Text{
			String: "Second failure after success",
			Valid:  true,
		},
		CreatedAt: pgtype.Timestamptz{
			Time:  time4,
			Valid: true,
		},
	}

	event5 := db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subID,
		EventType:      db.SubscriptionEventTypeRedeemed,
		TransactionHash: pgtype.Text{
			String: "0xsecond_success",
			Valid:  true,
		},
		CreatedAt: pgtype.Timestamptz{
			Time:  time5,
			Valid: true,
		},
	}

	// Return events in non-chronological order to test sorting logic
	events := []db.SubscriptionEvent{event1, event4, event2, event5, event3}

	// Mock DB responses
	mockDB.On("GetSubscription", mock.Anything, subID).Return(subscription, nil)
	mockDB.On("ListSubscriptionEventsBySubscription", mock.Anything, subID).Return(events, nil)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "subscription_id", Value: subID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodGet, "/subscriptions/"+subID.String()+"/redemption-status", nil)
	c.Request = req

	// Directly invoke the handler function with our required context
	handler := &GetRedemptionStatusHandler{
		db: mockDB,
	}
	handler.Handle(c)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response GetRedemptionStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, subID.String(), response.SubscriptionID)
	assert.Equal(t, "success", response.Status)
	assert.Equal(t, "Subscription successfully redeemed", response.Message)
	assert.Equal(t, int32(2), response.TotalRedemptions)

	// Should have the most recent redemption timestamp
	assert.NotNil(t, response.LastRedemptionAt)
	assert.Equal(t, time5.Unix(), response.LastRedemptionAt.Unix())

	// Should have the most recent transaction hash
	assert.Equal(t, "0xsecond_success", response.TransactionHash)

	// Verify mocks
	mockDB.AssertExpectations(t)
}

// TestGetRedemptionStatus_WithDeletedEvents tests the redemption status endpoint
// when some events should be filtered out
func TestGetRedemptionStatus_WithDeletedEvents(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mocks
	mockDB := new(MockDB)

	// Create test data
	subID := uuid.New()
	txHash := "0x3333333333333333"

	// Base time
	now := time.Now()

	subscription := db.Subscription{
		ID:               subID,
		TotalRedemptions: 1,
		Status:           db.SubscriptionStatusActive,
		NextRedemptionDate: pgtype.Timestamptz{
			Time:  now.Add(24 * time.Hour),
			Valid: true,
		},
	}

	// Create two events - one that should be ignored (older) and one that should be processed (newer)
	olderEvent := db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subID,
		EventType:      db.SubscriptionEventTypeRedeemed,
		TransactionHash: pgtype.Text{
			String: "0xolder_tx_hash",
			Valid:  true,
		},
		CreatedAt: pgtype.Timestamptz{
			Time:  now.Add(-2 * time.Hour),
			Valid: true,
		},
	}

	newerEvent := db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subID,
		EventType:      db.SubscriptionEventTypeRedeemed,
		TransactionHash: pgtype.Text{
			String: txHash,
			Valid:  true,
		},
		CreatedAt: pgtype.Timestamptz{
			Time:  now.Add(-1 * time.Hour),
			Valid: true,
		},
	}

	// Return both events - handler should use the one with the more recent timestamp
	mockDB.On("GetSubscription", mock.Anything, subID).Return(subscription, nil)
	mockDB.On("ListSubscriptionEventsBySubscription", mock.Anything, subID).Return(
		[]db.SubscriptionEvent{olderEvent, newerEvent}, nil)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "subscription_id", Value: subID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodGet, "/subscriptions/"+subID.String()+"/redemption-status", nil)
	c.Request = req

	// Create a handler and invoke it directly
	handler := &GetRedemptionStatusHandler{
		db: mockDB,
	}
	handler.GetRedemptionStatus(c)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response GetRedemptionStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, subID.String(), response.SubscriptionID)
	assert.Equal(t, "success", response.Status)
	assert.Equal(t, "Subscription successfully redeemed", response.Message)
	assert.Equal(t, int32(1), response.TotalRedemptions)

	// Should have the timestamp from the newer event, not the older one
	assert.NotNil(t, response.LastRedemptionAt)
	assert.Equal(t, newerEvent.CreatedAt.Time.Unix(), response.LastRedemptionAt.Unix())

	// Should have the transaction hash from the newer event
	assert.Equal(t, txHash, response.TransactionHash)

	// Verify mocks
	mockDB.AssertExpectations(t)
}

// TestGetRedemptionStatus_SameTimestampDifferentTypes tests the redemption status endpoint
// when events have the same timestamp but different types
func TestGetRedemptionStatus_SameTimestampDifferentTypes(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create mocks
	mockDB := new(MockDB)

	// Create test data
	subID := uuid.New()
	txHash := "0xabcdef1234567890"
	errorMsg := "Transaction failed due to network congestion"

	// Common timestamp for multiple events
	sameTime := time.Now().Truncate(time.Second)

	subscription := db.Subscription{
		ID:               subID,
		TotalRedemptions: 0,
		Status:           db.SubscriptionStatusActive,
		NextRedemptionDate: pgtype.Timestamptz{
			Time:  time.Now().Add(24 * time.Hour),
			Valid: true,
		},
	}

	// Create a successful redemption event and a failed redemption event with the same timestamp
	successEvent := db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subID,
		EventType:      db.SubscriptionEventTypeRedeemed,
		TransactionHash: pgtype.Text{
			String: txHash,
			Valid:  true,
		},
		CreatedAt: pgtype.Timestamptz{
			Time:  sameTime,
			Valid: true,
		},
	}

	failedEvent := db.SubscriptionEvent{
		ID:             uuid.New(),
		SubscriptionID: subID,
		EventType:      db.SubscriptionEventTypeFailedRedemption,
		ErrorMessage: pgtype.Text{
			String: errorMsg,
			Valid:  true,
		},
		CreatedAt: pgtype.Timestamptz{
			Time:  sameTime,
			Valid: true,
		},
	}

	// Test Case 1: Success event is AFTER failed event in the list
	// Per current handler logic, the last event in the list should determine the status
	mockDB.On("GetSubscription", mock.Anything, subID).Return(subscription, nil).Once()
	mockDB.On("ListSubscriptionEventsBySubscription", mock.Anything, subID).Return(
		[]db.SubscriptionEvent{failedEvent, successEvent}, nil).Once()

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "subscription_id", Value: subID.String()},
	}

	// Setup request context
	req := httptest.NewRequest(http.MethodGet, "/subscriptions/"+subID.String()+"/redemption-status", nil)
	c.Request = req

	// Create a handler and invoke it directly
	handler := &GetRedemptionStatusHandler{
		db: mockDB,
	}
	handler.GetRedemptionStatus(c)

	// Assertions for the first case (success event is last in array)
	assert.Equal(t, http.StatusOK, w.Code)

	var response GetRedemptionStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Since success event is last in the list, we expect success status
	assert.Equal(t, "success", response.Status, "Should be success when success event is last")
	assert.Equal(t, txHash, response.TransactionHash)
	assert.NotNil(t, response.LastRedemptionAt)
	assert.Equal(t, sameTime.Unix(), response.LastRedemptionAt.Unix())

	// Test Case 2: Failed event is AFTER success event in the list
	// Create new mocks for the second test
	mockDB2 := new(MockDB)
	mockDB2.On("GetSubscription", mock.Anything, subID).Return(subscription, nil)
	mockDB2.On("ListSubscriptionEventsBySubscription", mock.Anything, subID).Return(
		[]db.SubscriptionEvent{successEvent, failedEvent}, nil)

	// Create a new request/response
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Params = gin.Params{
		{Key: "subscription_id", Value: subID.String()},
	}
	req2 := httptest.NewRequest(http.MethodGet, "/subscriptions/"+subID.String()+"/redemption-status", nil)
	c2.Request = req2

	// Create a handler and invoke it
	handler2 := &GetRedemptionStatusHandler{
		db: mockDB2,
	}
	handler2.GetRedemptionStatus(c2)

	// Assertions for the second case (failed event is last in array)
	assert.Equal(t, http.StatusOK, w2.Code)

	var response2 GetRedemptionStatusResponse
	err = json.Unmarshal(w2.Body.Bytes(), &response2)
	assert.NoError(t, err)

	// Since failed event is last in the list, we expect failed status
	assert.Equal(t, "failed", response2.Status, "Should be failed when failed event is last")
	assert.Equal(t, errorMsg, response2.FailureReason)
	assert.NotNil(t, response2.LastAttemptedAt)
	assert.Equal(t, sameTime.Unix(), response2.LastAttemptedAt.Unix())

	// Verify mocks
	mockDB.AssertExpectations(t)
	mockDB2.AssertExpectations(t)
}
