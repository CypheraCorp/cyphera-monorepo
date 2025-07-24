package helpers

import (
	"context"
	"errors"
	"fmt"

	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// TransactionFunc is a function that executes within a database transaction
type TransactionFunc func(tx pgx.Tx) error

// WithTransaction executes a function within a database transaction.
// It automatically handles commit/rollback based on the error returned by the function.
// If the function returns an error, the transaction is rolled back.
// If the function returns nil, the transaction is committed.
func WithTransaction(ctx context.Context, pool *pgxpool.Pool, fn TransactionFunc) error {
	// Start transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure we always attempt to finalize the transaction
	defer func() {
		// If transaction is already closed (committed), rollback will return ErrTxClosed
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			logger.Log.Error("Failed to rollback transaction", 
				zap.Error(rollbackErr),
				zap.Bool("was_committed", errors.Is(rollbackErr, pgx.ErrTxClosed)),
			)
		}
	}()

	// Execute the function
	if err := fn(tx); err != nil {
		// Function returned error, transaction will be rolled back by defer
		return fmt.Errorf("transaction failed: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		// Commit failed, transaction will be rolled back by defer
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Success - transaction is committed
	return nil
}

// WithTransactionRetry executes a function within a database transaction with retry logic.
// It will retry the transaction up to maxRetries times if it encounters a serialization error.
func WithTransactionRetry(ctx context.Context, pool *pgxpool.Pool, maxRetries int, fn TransactionFunc) error {
	var err error
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err = WithTransaction(ctx, pool, fn)
		
		// If successful, return immediately
		if err == nil {
			return nil
		}
		
		// Check if error is retryable (serialization failure)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "40001" { // serialization_failure
			if attempt < maxRetries {
				logger.Log.Warn("Transaction failed due to serialization error, retrying",
					zap.Int("attempt", attempt+1),
					zap.Int("max_retries", maxRetries),
					zap.Error(err),
				)
				continue
			}
		}
		
		// Non-retryable error or max retries exceeded
		break
	}
	
	return err
}

// TransactionOptions provides additional options for transaction execution
type TransactionOptions struct {
	IsolationLevel pgx.TxIsoLevel
	AccessMode     pgx.TxAccessMode
	DeferrableMode pgx.TxDeferrableMode
}

// WithTransactionOptions executes a function within a database transaction with custom options
func WithTransactionOptions(ctx context.Context, pool *pgxpool.Pool, opts TransactionOptions, fn TransactionFunc) error {
	// Configure transaction options
	txOpts := pgx.TxOptions{
		IsoLevel:       opts.IsolationLevel,
		AccessMode:     opts.AccessMode,
		DeferrableMode: opts.DeferrableMode,
	}

	// Start transaction with options
	tx, err := pool.BeginTx(ctx, txOpts)
	if err != nil {
		return fmt.Errorf("failed to begin transaction with options: %w", err)
	}

	// Same pattern as WithTransaction
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			logger.Log.Error("Failed to rollback transaction", 
				zap.Error(rollbackErr),
				zap.Bool("was_committed", errors.Is(rollbackErr, pgx.ErrTxClosed)),
			)
		}
	}()

	if err := fn(tx); err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}