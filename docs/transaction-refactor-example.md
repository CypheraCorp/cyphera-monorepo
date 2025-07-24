# Transaction Refactoring Example

This document shows how to refactor existing transaction handling to use the new transaction helper.

## Before (Old Pattern)

```go
// Begin a transaction
tx, qtx, err := h.common.BeginTx(c.Request.Context())
if err != nil {
    sendError(c, http.StatusInternalServerError, "Failed to begin transaction", err)
    return
}
defer func() {
    if rErr := tx.Rollback(c.Request.Context()); rErr != nil && !errors.Is(rErr, pgx.ErrTxClosed) {
        logger.Error("Failed to rollback transaction", zap.Error(rErr))
    }
}()

// Do work with qtx...
wallet, err := qtx.CreateWallet(c.Request.Context(), params)
if err != nil {
    sendError(c, http.StatusInternalServerError, "Failed to create wallet", err)
    return
}

// More operations...

// Commit the transaction
if err := tx.Commit(c.Request.Context()); err != nil {
    sendError(c, http.StatusInternalServerError, "Failed to commit transaction", err)
    return
}
```

## After (New Pattern)

```go
// All transaction handling is encapsulated
err := h.common.RunInTransaction(c.Request.Context(), func(qtx *db.Queries) error {
    // Do work with qtx...
    wallet, err := qtx.CreateWallet(c.Request.Context(), params)
    if err != nil {
        return fmt.Errorf("failed to create wallet: %w", err)
    }
    
    // More operations...
    
    // Return nil to commit, or error to rollback
    return nil
})

if err != nil {
    sendError(c, http.StatusInternalServerError, "Transaction failed", err)
    return
}
```

## Benefits

1. **Automatic Rollback**: No need for defer statements or manual rollback
2. **Cleaner Code**: Transaction logic is encapsulated
3. **Consistent Error Handling**: All transaction errors are handled the same way
4. **No Leaked Transactions**: Can't forget to commit or rollback
5. **Retry Support**: Use `RunInTransactionWithRetry` for automatic retry on serialization errors

## Example with Retry

```go
// For operations that might have conflicts
err := h.common.RunInTransactionWithRetry(c.Request.Context(), 3, func(qtx *db.Queries) error {
    // Check and update operations that might conflict
    subscription, err := qtx.GetSubscriptionForUpdate(c.Request.Context(), subID)
    if err != nil {
        return err
    }
    
    // Update subscription...
    
    return nil
})
```

## Real Example - Circle Wallet Creation

### Before:
```go
func (h *CircleHandler) GetWallet(c *gin.Context) {
    // ... validation code ...
    
    // Begin a transaction
    tx, qtx, err := h.common.BeginTx(c.Request.Context())
    if err != nil {
        sendError(c, http.StatusInternalServerError, "Failed to begin transaction", err)
        return
    }
    defer func() {
        if rErr := tx.Rollback(c.Request.Context()); rErr != nil && !errors.Is(rErr, pgx.ErrTxClosed) {
            logger.Error("Failed to rollback transaction in GetWallet", zap.Error(rErr))
        }
    }()
    
    // Check if wallet exists
    dbWallet, err := h.common.db.GetWalletByAddressAndCircleNetworkType(c.Request.Context(), params)
    
    if walletExists {
        // Update existing wallet...
    } else {
        // Create new wallet
        newWallet, err := qtx.CreateWallet(c.Request.Context(), walletParams)
        if err != nil {
            sendError(c, http.StatusInternalServerError, "Failed to create wallet", err)
            return
        }
        
        // Create Circle wallet entry
        _, err = qtx.CreateCircleWalletEntry(c.Request.Context(), circleParams)
        if err != nil {
            sendError(c, http.StatusInternalServerError, "Failed to create Circle wallet entry", err)
            return
        }
    }
    
    // Commit the transaction
    if err := tx.Commit(c.Request.Context()); err != nil {
        sendError(c, http.StatusInternalServerError, "Failed to commit transaction", err)
        return
    }
    
    sendSuccess(c, http.StatusOK, walletResponse)
}
```

### After:
```go
func (h *CircleHandler) GetWallet(c *gin.Context) {
    // ... validation code ...
    
    // All transaction logic encapsulated
    err := h.common.RunInTransaction(c.Request.Context(), func(qtx *db.Queries) error {
        // Check if wallet exists
        dbWallet, err := h.common.db.GetWalletByAddressAndCircleNetworkType(c.Request.Context(), params)
        
        if walletExists {
            // Update existing wallet...
            return nil
        }
        
        // Create new wallet
        newWallet, err := qtx.CreateWallet(c.Request.Context(), walletParams)
        if err != nil {
            return fmt.Errorf("failed to create wallet: %w", err)
        }
        
        // Create Circle wallet entry
        _, err = qtx.CreateCircleWalletEntry(c.Request.Context(), circleParams)
        if err != nil {
            return fmt.Errorf("failed to create Circle wallet entry: %w", err)
        }
        
        return nil // Success - transaction will be committed
    })
    
    if err != nil {
        sendError(c, http.StatusInternalServerError, "Transaction failed", err)
        return
    }
    
    sendSuccess(c, http.StatusOK, walletResponse)
}
```

## Migration Steps

1. Identify all uses of `BeginTx` in handlers
2. Replace with `RunInTransaction` or `RunInTransactionWithRetry`
3. Remove manual rollback/commit code
4. Convert early returns to error returns within the transaction function
5. Test to ensure behavior remains the same