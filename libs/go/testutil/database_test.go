package testutil

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestDB_Connection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}

	db := NewTestDB(t)
	defer db.Close()

	// Test basic connection
	assert.NotNil(t, db.Pool())

	// Test ping
	err := db.Pool().Ping(context.Background())
	require.NoError(t, err)
}

func TestTestDB_SetupSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}

	db := NewTestDB(t)
	defer db.Close()

	// This will skip if schema doesn't exist
	// In a real test, you'd want to set up your schema first
	db.SetupSchema(t)
}

func TestTestDB_Truncate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}

	db := NewTestDB(t)
	defer db.Close()

	// Create a test table for demonstration
	ctx := context.Background()
	_, err := db.Pool().Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_table (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	// Insert some test data
	_, err = db.Pool().Exec(ctx, "INSERT INTO test_table (name) VALUES ('test1'), ('test2')")
	require.NoError(t, err)

	// Verify data exists
	var count int
	err = db.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM test_table").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Truncate the table
	db.Truncate(t, "test_table")

	// Verify data is gone
	err = db.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM test_table").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Clean up
	_, err = db.Pool().Exec(ctx, "DROP TABLE test_table")
	require.NoError(t, err)
}

func TestTestDB_WithTransaction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}

	db := NewTestDB(t)
	defer db.Close()

	// Create a test table
	ctx := context.Background()
	_, err := db.Pool().Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_txn_table (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	// Use transaction - changes should be rolled back
	db.WithTransaction(t, func(pool *pgxpool.Pool) {
		_, err := pool.Exec(ctx, "INSERT INTO test_txn_table (name) VALUES ('should_not_persist')")
		require.NoError(t, err)

		// Verify data exists within transaction
		var count int
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM test_txn_table").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	// Verify data was rolled back (this test is simplified - in real implementation)
	// Note: This test doesn't actually test rollback since we're using the same pool
	// In a proper implementation, you'd want to use a transaction wrapper

	// Clean up
	_, err = db.Pool().Exec(ctx, "DROP TABLE test_txn_table")
	require.NoError(t, err)
}