package testutil

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
)

// MockDatabase provides utilities for database mocking in unit tests
type MockDatabase struct {
	ctrl    *gomock.Controller
	Querier *mocks.MockQuerier
	t       *testing.T
}

// NewMockDatabase creates a new mock database for unit testing
func NewMockDatabase(t *testing.T) *MockDatabase {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	return &MockDatabase{
		ctrl:    ctrl,
		Querier: mocks.NewMockQuerier(ctrl),
		t:       t,
	}
}

// ExpectWorkspaceExists sets up expectation for workspace existence check
func (m *MockDatabase) ExpectWorkspaceExists(workspaceID uuid.UUID, exists bool) {
	if exists {
		m.Querier.EXPECT().
			GetWorkspace(gomock.Any(), workspaceID).
			Return(db.Workspace{
				ID:   workspaceID,
				Name: "Test Workspace",
			}, nil).
			Times(1)
	} else {
		m.Querier.EXPECT().
			GetWorkspace(gomock.Any(), workspaceID).
			Return(db.Workspace{}, pgx.ErrNoRows).
			Times(1)
	}
}

// ExpectSubscriptionExists sets up expectation for subscription existence
func (m *MockDatabase) ExpectSubscriptionExists(subscriptionID uuid.UUID, subscription *db.Subscription) {
	if subscription != nil {
		m.Querier.EXPECT().
			GetSubscription(gomock.Any(), subscriptionID).
			Return(*subscription, nil).
			Times(1)
	} else {
		m.Querier.EXPECT().
			GetSubscription(gomock.Any(), subscriptionID).
			Return(db.Subscription{}, pgx.ErrNoRows).
			Times(1)
	}
}

// ExpectListSubscriptions sets up expectation for listing subscriptions
func (m *MockDatabase) ExpectListSubscriptions(customerID uuid.UUID, subscriptions []db.Subscription) {
	m.Querier.EXPECT().
		ListSubscriptionsByCustomer(gomock.Any(), gomock.Any()).
		Return(subscriptions, nil).
		Times(1)
}

// ExpectCreateSubscription sets up expectation for subscription creation
func (m *MockDatabase) ExpectCreateSubscription(subscription db.Subscription) {
	m.Querier.EXPECT().
		CreateSubscription(gomock.Any(), gomock.Any()).
		Return(subscription, nil).
		Times(1)
}

// ExpectUpdateSubscription sets up expectation for subscription update
func (m *MockDatabase) ExpectUpdateSubscription(subscriptionID uuid.UUID, updatedSubscription db.Subscription) {
	m.Querier.EXPECT().
		UpdateSubscription(gomock.Any(), gomock.Any()).
		Return(updatedSubscription, nil).
		Times(1)
}

// ExpectDeleteSubscription sets up expectation for subscription deletion
func (m *MockDatabase) ExpectDeleteSubscription(subscriptionID uuid.UUID) {
	m.Querier.EXPECT().
		DeleteSubscription(gomock.Any(), subscriptionID).
		Return(nil).
		Times(1)
}

// ExpectAPIKeyOperations sets up expectations for API key CRUD operations
func (m *MockDatabase) ExpectAPIKeyOperations() *APIKeyMockHelper {
	return &APIKeyMockHelper{mock: m}
}

// APIKeyMockHelper provides fluent API for API key mock expectations
type APIKeyMockHelper struct {
	mock *MockDatabase
}

// ExpectCreate sets up expectation for API key creation
func (h *APIKeyMockHelper) ExpectCreate(apiKey db.ApiKey) *APIKeyMockHelper {
	h.mock.Querier.EXPECT().
		CreateAPIKey(gomock.Any(), gomock.Any()).
		Return(apiKey, nil).
		Times(1)
	return h
}

// ExpectList sets up expectation for API key listing
func (h *APIKeyMockHelper) ExpectList(workspaceID uuid.UUID, apiKeys []db.ApiKey) *APIKeyMockHelper {
	h.mock.Querier.EXPECT().
		ListAPIKeys(gomock.Any(), workspaceID).
		Return(apiKeys, nil).
		Times(1)
	return h
}

// ExpectGet sets up expectation for API key retrieval
func (h *APIKeyMockHelper) ExpectGet(keyID uuid.UUID, apiKey *db.ApiKey) *APIKeyMockHelper {
	if apiKey != nil {
		h.mock.Querier.EXPECT().
			GetAPIKey(gomock.Any(), gomock.Any()).
			Return(*apiKey, nil).
			Times(1)
	} else {
		h.mock.Querier.EXPECT().
			GetAPIKey(gomock.Any(), gomock.Any()).
			Return(db.ApiKey{}, pgx.ErrNoRows).
			Times(1)
	}
	return h
}

// ExpectUpdate sets up expectation for API key update
func (h *APIKeyMockHelper) ExpectUpdate(keyID uuid.UUID, updatedKey db.ApiKey) *APIKeyMockHelper {
	h.mock.Querier.EXPECT().
		UpdateAPIKey(gomock.Any(), gomock.Any()).
		Return(updatedKey, nil).
		Times(1)
	return h
}

// ExpectDelete sets up expectation for API key deletion
func (h *APIKeyMockHelper) ExpectDelete(keyID uuid.UUID) *APIKeyMockHelper {
	h.mock.Querier.EXPECT().
		DeleteAPIKey(gomock.Any(), keyID).
		Return(nil).
		Times(1)
	return h
}

// CreateTestSubscription creates a test subscription with realistic data
func CreateTestSubscription(workspaceID, customerID uuid.UUID) db.Subscription {
	return db.Subscription{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		CustomerID:  customerID,
		Status:      db.SubscriptionStatusActive,
		CreatedAt:   pgtype.Timestamptz{Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Valid: true},
	}
}

// CreateTestAPIKey creates a test API key with realistic data
func CreateTestAPIKey(workspaceID uuid.UUID, name string) db.ApiKey {
	return db.ApiKey{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Name:        name,
		KeyHash:     "$2a$10$test.hash.value",
		KeyPrefix:   pgtype.Text{String: "cyphera_", Valid: true},
		AccessLevel: db.ApiKeyLevelWrite,
		CreatedAt:   pgtype.Timestamptz{Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Valid: true},
	}
}

// CreateTestWorkspace creates a test workspace with realistic data
func CreateTestWorkspace(name string) db.Workspace {
	return db.Workspace{
		ID:        uuid.New(),
		Name:      name,
		CreatedAt: pgtype.Timestamptz{Valid: true},
		UpdatedAt: pgtype.Timestamptz{Valid: true},
	}
}
