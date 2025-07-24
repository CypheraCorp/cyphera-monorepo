package handlers

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/testutil"
)

// Simple tests to verify database mocking functionality

func TestMockDatabase_SubscriptionOperations(t *testing.T) {
	mockDB := testutil.NewMockDatabase(t)
	workspaceID := uuid.New()
	customerID := uuid.New()
	subscriptionID := uuid.New()

	t.Run("subscription exists", func(t *testing.T) {
		// Create test subscription
		expectedSubscription := testutil.CreateTestSubscription(workspaceID, customerID)
		expectedSubscription.ID = subscriptionID
		
		// Setup mock expectations
		mockDB.ExpectSubscriptionExists(subscriptionID, &expectedSubscription)

		// Test the mock expectation
		result, err := mockDB.Querier.GetSubscription(context.Background(), subscriptionID)
		require.NoError(t, err)
		assert.Equal(t, subscriptionID, result.ID)
		assert.Equal(t, workspaceID, result.WorkspaceID)
	})

	t.Run("subscription not found", func(t *testing.T) {
		// Setup mock to return not found
		mockDB.ExpectSubscriptionExists(subscriptionID, nil)

		// Test the mock expectation
		_, err := mockDB.Querier.GetSubscription(context.Background(), subscriptionID)
		require.Error(t, err)
		assert.Equal(t, pgx.ErrNoRows, err)
	})
}

func TestMockDatabase_APIKeyOperations(t *testing.T) {
	mockDB := testutil.NewMockDatabase(t)
	workspaceID := uuid.New()
	keyID := uuid.New()

	t.Run("API key operations", func(t *testing.T) {
		// Create test API key
		expectedAPIKey := testutil.CreateTestAPIKey(workspaceID, "Test Key")
		expectedAPIKey.ID = keyID

		// Setup mock expectations using fluent API
		mockDB.ExpectAPIKeyOperations().
			ExpectGet(keyID, &expectedAPIKey).
			ExpectList(workspaceID, []db.ApiKey{expectedAPIKey})

		// Test get operation
		result, err := mockDB.Querier.GetAPIKey(context.Background(), db.GetAPIKeyParams{ID: keyID})
		require.NoError(t, err)
		assert.Equal(t, keyID, result.ID)
		assert.Equal(t, "Test Key", result.Name)

		// Test list operation
		list, err := mockDB.Querier.ListAPIKeys(context.Background(), workspaceID)
		require.NoError(t, err)
		assert.Len(t, list, 1)
		assert.Equal(t, keyID, list[0].ID)
	})
}

func TestMockDatabase_TestDataHelpers(t *testing.T) {
	workspaceID := uuid.New()
	customerID := uuid.New()

	t.Run("test data creation", func(t *testing.T) {
		// Test subscription creation
		subscription := testutil.CreateTestSubscription(workspaceID, customerID)
		assert.NotEqual(t, uuid.Nil, subscription.ID)
		assert.Equal(t, workspaceID, subscription.WorkspaceID)
		assert.Equal(t, customerID, subscription.CustomerID)
		assert.Equal(t, db.SubscriptionStatusActive, subscription.Status)
		assert.True(t, subscription.CreatedAt.Valid)

		// Test API key creation
		apiKey := testutil.CreateTestAPIKey(workspaceID, "Production Key")
		assert.NotEqual(t, uuid.Nil, apiKey.ID)
		assert.Equal(t, workspaceID, apiKey.WorkspaceID)
		assert.Equal(t, "Production Key", apiKey.Name)
		assert.Equal(t, db.ApiKeyLevelWrite, apiKey.AccessLevel)
		assert.NotEmpty(t, apiKey.KeyHash)

		// Test workspace creation
		workspace := testutil.CreateTestWorkspace("Test Workspace")
		assert.NotEqual(t, uuid.Nil, workspace.ID)
		assert.Equal(t, "Test Workspace", workspace.Name)
		assert.True(t, workspace.CreatedAt.Valid)
	})
}

// Benchmark test to show database mocking performance benefits
func BenchmarkMockDatabase_GetSubscription(b *testing.B) {
	mockDB := testutil.NewMockDatabase(&testing.T{})
	subscriptionID := uuid.New()
	workspaceID := uuid.New()
	subscription := testutil.CreateTestSubscription(workspaceID, uuid.New())
	subscription.ID = subscriptionID

	// Setup mock to allow unlimited calls
	mockDB.Querier.EXPECT().
		GetSubscription(gomock.Any(), subscriptionID).
		Return(subscription, nil).
		AnyTimes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mockDB.Querier.GetSubscription(context.Background(), subscriptionID)
		if err != nil {
			b.Fatal(err)
		}
	}
}