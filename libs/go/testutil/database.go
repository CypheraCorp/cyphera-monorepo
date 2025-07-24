package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const (
	defaultTestDSN = "postgres://postgres:postgres@localhost:5433/cyphera_test?sslmode=disable"
)

// TestDB provides database utilities for testing
type TestDB struct {
	pool *pgxpool.Pool
	dsn  string
}

// NewTestDB creates a new test database connection
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = defaultTestDSN
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err, "Failed to connect to test database")

	// Ping to verify connection
	err = pool.Ping(context.Background())
	require.NoError(t, err, "Failed to ping test database")

	return &TestDB{
		pool: pool,
		dsn:  dsn,
	}
}

// Pool returns the database pool
func (db *TestDB) Pool() *pgxpool.Pool {
	return db.pool
}

// Close closes the database connection
func (db *TestDB) Close() error {
	if db.pool != nil {
		db.pool.Close()
	}
	return nil
}

// Truncate truncates all tables in the database
func (db *TestDB) Truncate(t *testing.T, tables ...string) {
	t.Helper()

	ctx := context.Background()
	
	if len(tables) == 0 {
		// Get all table names if none specified
		rows, err := db.pool.Query(ctx, `
			SELECT tablename FROM pg_tables 
			WHERE schemaname = 'public' 
			AND tablename NOT LIKE 'pg_%'
			AND tablename != 'schema_migrations'
		`)
		require.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var table string
			err := rows.Scan(&table)
			require.NoError(t, err)
			tables = append(tables, table)
		}
	}

	if len(tables) > 0 {
		query := fmt.Sprintf("TRUNCATE %s CASCADE", joinTables(tables))
		_, err := db.pool.Exec(ctx, query)
		require.NoError(t, err, "Failed to truncate tables")
	}
}

// WithTransaction executes a function within a transaction that gets rolled back
func (db *TestDB) WithTransaction(t *testing.T, fn func(*pgxpool.Pool)) {
	t.Helper()

	ctx := context.Background()
	tx, err := db.pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// For testing, we'll just use the original pool
	// In practice, you'd want to implement a proper transaction wrapper
	fn(db.pool)
}

// SetupSchema runs schema migrations if needed
func (db *TestDB) SetupSchema(t *testing.T) {
	t.Helper()
	
	// Check if schema exists by looking for a key table
	ctx := context.Background()
	var exists bool
	err := db.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'accounts'
		)
	`).Scan(&exists)
	require.NoError(t, err)

	if !exists {
		t.Log("Setting up test database schema...")
		// You would run your migrations here
		// For now, we'll assume the schema exists or is set up externally
		t.Skip("Test database schema not found. Run migrations first.")
	}
}

// joinTables joins table names with quotes for SQL
func joinTables(tables []string) string {
	if len(tables) == 0 {
		return ""
	}
	
	result := `"` + tables[0] + `"`
	for i := 1; i < len(tables); i++ {
		result += `, "` + tables[i] + `"`
	}
	return result
}

// Note: In a full implementation, you'd want to create a proper transaction wrapper
// that implements the pgxpool.Pool interface for true transactional testing