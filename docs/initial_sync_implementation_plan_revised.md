# Initial Sync Implementation Plan (Revised)

## Overview

This document outlines the implementation plan for the **Initial Sync Functionality** that integrates seamlessly with the existing Cyphera project structure, database schema, and handler patterns.

## Goals

1. **Primary Goal**: Implement initial data synchronization from Stripe to Cyphera using existing database tables
2. **Integration Goal**: Follow existing handler patterns and database transaction workflows
3. **Workspace Goal**: Properly handle workspace-based multi-tenancy
4. **Consistency Goal**: Maintain consistency with existing codebase patterns and naming conventions

## Database Integration Strategy

### Direct Schema Integration (Overwrite Existing Schema)
Instead of using ALTER statements, we'll update the existing `internal/db/init-scripts/01-init.sql` file to include payment sync columns directly in the table definitions. This approach ensures a clean schema and avoids migration complexities.

#### Updated Table Definitions with Payment Sync Support

**Customers Table** (with payment sync columns):
```sql
CREATE TABLE IF NOT EXISTS customers (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    external_id VARCHAR(255),
    email VARCHAR(255),
    name VARCHAR(255),
    phone VARCHAR(255),
    description TEXT,
    metadata JSONB,
    -- Payment sync tracking columns
    payment_sync_status VARCHAR(20) DEFAULT 'pending',
    payment_synced_at TIMESTAMP WITH TIME ZONE,
    payment_sync_version INTEGER DEFAULT 1,
    payment_provider VARCHAR(50), -- 'stripe', 'chargebee', etc.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(workspace_id, external_id)
);
```

**Products Table** (with payment sync columns):
```sql
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    name TEXT NOT NULL,
    description TEXT,
    image_url TEXT,
    url TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB DEFAULT '{}'::jsonb,
    -- Payment sync tracking columns
    payment_sync_status VARCHAR(20) DEFAULT 'pending', 
    payment_synced_at TIMESTAMP WITH TIME ZONE,
    payment_sync_version INTEGER DEFAULT 1,
    payment_provider VARCHAR(50), -- 'stripe', 'chargebee', etc.
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
```

**Prices Table** (with payment sync columns):
```sql
CREATE TABLE prices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id),
    active BOOLEAN NOT NULL DEFAULT true,
    type price_type NOT NULL, -- 'recurring' or 'one_off'
    nickname TEXT,
    currency currency NOT NULL, -- 'USD', 'EUR'
    unit_amount_in_pennies INTEGER NOT NULL,
    interval_type interval_type NOT NULL,
    term_length INTEGER NOT NULL, -- Nullable, for 'recurring' type, e.g., 12 for 12 months
    metadata JSONB DEFAULT '{}'::jsonb,
    -- Payment sync tracking columns
    payment_sync_status VARCHAR(20) DEFAULT 'pending',
    payment_synced_at TIMESTAMP WITH TIME ZONE, 
    payment_sync_version INTEGER DEFAULT 1,
    payment_provider VARCHAR(50), -- 'stripe', 'chargebee', etc.
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT prices_recurring_fields_check CHECK (
        (type = 'recurring' AND interval_type IS NOT NULL AND term_length IS NOT NULL AND term_length > 0) OR
        (type = 'one_off' AND interval_type IS NULL AND term_length IS NULL)
    )
);
```

**Subscriptions Table** (with payment sync columns):
```sql
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    product_id UUID NOT NULL REFERENCES products(id),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    price_id UUID NOT NULL REFERENCES prices(id),
    product_token_id UUID NOT NULL REFERENCES products_tokens(id),
    token_amount INTEGER NOT NULL,
    delegation_id UUID NOT NULL REFERENCES delegation_data(id),
    customer_wallet_id UUID REFERENCES customer_wallets(id),
    status subscription_status NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    current_period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    next_redemption_date TIMESTAMP WITH TIME ZONE,
    total_redemptions INT NOT NULL DEFAULT 0,
    total_amount_in_cents INT NOT NULL DEFAULT 0,
    metadata JSONB DEFAULT '{}'::jsonb,
    -- Payment sync tracking columns
    payment_sync_status VARCHAR(20) DEFAULT 'pending',
    payment_synced_at TIMESTAMP WITH TIME ZONE,
    payment_sync_version INTEGER DEFAULT 1,
    payment_provider VARCHAR(50), -- 'stripe', 'chargebee', etc.
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
```

#### New Payment Sync Tables

**Payment Sync Sessions Table** (for tracking sync jobs):
```sql
CREATE TABLE IF NOT EXISTS payment_sync_sessions (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    provider_name VARCHAR(50) NOT NULL, -- 'stripe', 'chargebee', 'recurly', etc.
    session_type VARCHAR(50) NOT NULL, -- 'initial_sync', 'partial_sync', 'delta_sync'
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed', 'cancelled'
    entity_types TEXT[] NOT NULL, -- ['customers', 'products', 'prices', 'subscriptions', 'invoices']
    config JSONB DEFAULT '{}',
    progress JSONB DEFAULT '{}',
    error_summary JSONB,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
```

**Payment Sync Events Table** (for detailed sync tracking):
```sql
CREATE TABLE IF NOT EXISTS payment_sync_events (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES payment_sync_sessions(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    provider_name VARCHAR(50) NOT NULL, -- 'stripe', 'chargebee', etc.
    entity_type VARCHAR(50) NOT NULL, -- 'customer', 'product', 'price', 'subscription', 'invoice'
    entity_id UUID, -- Reference to the actual entity (customer_id, product_id, etc.)
    external_id VARCHAR(255), -- Provider's ID (Stripe ID, Chargebee ID, etc.)
    event_type VARCHAR(50) NOT NULL, -- 'sync_started', 'sync_completed', 'sync_failed', 'sync_skipped'
    event_message TEXT,
    event_details JSONB,
    occurred_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

#### Required Indexes for Payment Sync

```sql
-- Payment sync sessions indexes
CREATE INDEX idx_payment_sync_sessions_workspace_id ON payment_sync_sessions(workspace_id);
CREATE INDEX idx_payment_sync_sessions_provider ON payment_sync_sessions(provider_name);
CREATE INDEX idx_payment_sync_sessions_status ON payment_sync_sessions(status) WHERE deleted_at IS NULL;

-- Payment sync events indexes
CREATE INDEX idx_payment_sync_events_session_id ON payment_sync_events(session_id);
CREATE idx_payment_sync_events_provider ON payment_sync_events(provider_name);
CREATE INDEX idx_payment_sync_events_entity_type ON payment_sync_events(entity_type);
CREATE INDEX idx_payment_sync_events_external_id ON payment_sync_events(external_id);

-- Payment sync status indexes for existing tables
CREATE INDEX idx_customers_payment_provider ON customers(payment_provider) WHERE deleted_at IS NULL;
CREATE INDEX idx_customers_payment_sync_status ON customers(payment_sync_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_payment_provider ON products(payment_provider) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_payment_sync_status ON products(payment_sync_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_prices_payment_provider ON prices(payment_provider) WHERE deleted_at IS NULL;
CREATE INDEX idx_prices_payment_sync_status ON prices(payment_sync_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_subscriptions_payment_provider ON subscriptions(payment_provider) WHERE deleted_at IS NULL;
CREATE INDEX idx_subscriptions_payment_sync_status ON subscriptions(payment_sync_status) WHERE deleted_at IS NULL;
```

#### Required Triggers

```sql
-- Add updated_at trigger for payment_sync_sessions
CREATE TRIGGER set_payment_sync_sessions_updated_at
    BEFORE UPDATE ON payment_sync_sessions
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();
```

### Benefits of Direct Schema Integration

1. **Clean Schema**: No migration history or ALTER statement dependencies
2. **Atomic Changes**: All schema changes applied in one transaction
3. **Consistent Naming**: Generic payment provider naming from the start
4. **Optimal Indexes**: Indexes optimized for payment sync queries from day one
5. **Single Source of Truth**: Complete schema definition in one file
6. **Development Reset**: Easy to reset development databases with complete schema

### Schema Update Process

1. **Backup Current Data**: Export any important test data from development database
2. **Update Schema File**: Modify `internal/db/init-scripts/01-init.sql` with payment sync columns
3. **Drop and Recreate**: Drop development database and recreate with new schema
4. **Import Test Data**: Re-import any required test data
5. **Update Application**: Ensure application code works with new schema
6. **Generate New Queries**: Run `sqlc generate` to update database query code

This approach ensures the initial sync functionality has a solid foundation with optimized database schema from the beginning, without the complexity of managing schema migrations during development.

## Implementation Structure

### 1. Stripe Initial Sync Service (`internal/client/payment_sync/stripe/initial_sync.go`)

```go
package stripe

import (
    "context"
    "time"
    ps "cyphera-api/internal/client/payment_sync"
    "github.com/google/uuid"
)

// StripeInitialSyncService handles initial bulk synchronization from Stripe
type StripeInitialSyncService struct {
    stripeService *StripeService
    logger        *zap.Logger
}

// NewStripeInitialSyncService creates a new initial sync service
func NewStripeInitialSyncService(stripeService *StripeService, logger *zap.Logger) *StripeInitialSyncService {
    return &StripeInitialSyncService{
        stripeService: stripeService,
        logger:        logger,
    }
}

// InitialSyncConfig represents configuration for initial sync
type InitialSyncConfig struct {
    WorkspaceID      uuid.UUID         `json:"workspace_id"`
    EntityTypes      []string          `json:"entity_types"` // ["customers", "products", "prices", "subscriptions", "invoices"]
    BatchSize        int               `json:"batch_size"`
    ConcurrentBatches int              `json:"concurrent_batches"`
    DateRange        *DateRange        `json:"date_range,omitempty"`
    ConflictStrategy ConflictStrategy  `json:"conflict_strategy"`
    DryRun           bool             `json:"dry_run"`
}

type DateRange struct {
    Start time.Time `json:"start"`
    End   time.Time `json:"end"`
}

type ConflictStrategy string
const (
    ConflictSkip      ConflictStrategy = "skip"      // Skip existing records
    ConflictOverwrite ConflictStrategy = "overwrite" // Overwrite with Stripe data
    ConflictFlag      ConflictStrategy = "flag"      // Flag for manual review
)

// SyncSession represents a sync session
type SyncSession struct {
    ID          uuid.UUID              `json:"id"`
    WorkspaceID uuid.UUID              `json:"workspace_id"`
    Status      SyncStatus             `json:"status"`
    Config      InitialSyncConfig      `json:"config"`
    Progress    map[string]*EntityProgress `json:"progress"`
    StartedAt   *time.Time             `json:"started_at,omitempty"`
    CompletedAt *time.Time             `json:"completed_at,omitempty"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}

type SyncStatus string
const (
    SyncStatusPending   SyncStatus = "pending"
    SyncStatusRunning   SyncStatus = "running"
    SyncStatusCompleted SyncStatus = "completed"
    SyncStatusFailed    SyncStatus = "failed"
    SyncStatusCancelled SyncStatus = "cancelled"
)

type EntityProgress struct {
    EntityType     string    `json:"entity_type"`
    TotalEstimated int64     `json:"total_estimated"`
    Processed      int64     `json:"processed"`
    Successful     int64     `json:"successful"`
    Failed         int64     `json:"failed"`
    Skipped        int64     `json:"skipped"`
    LastCursor     string    `json:"last_cursor"`
    IsComplete     bool      `json:"is_complete"`
    LastError      string    `json:"last_error,omitempty"`
}

// StartInitialSync initiates a full synchronization
func (s *StripeInitialSyncService) StartInitialSync(ctx context.Context, config InitialSyncConfig) (*SyncSession, error) {
    // Implementation here
}

// GetSyncStatus returns the current status of a sync session
func (s *StripeInitialSyncService) GetSyncStatus(ctx context.Context, sessionID uuid.UUID) (*SyncSession, error) {
    // Implementation here
}

// Private methods for entity synchronization
func (s *StripeInitialSyncService) syncCustomers(ctx context.Context, sessionID uuid.UUID, config InitialSyncConfig) error {
    // Implementation here
}

func (s *StripeInitialSyncService) syncProducts(ctx context.Context, sessionID uuid.UUID, config InitialSyncConfig) error {
    // Implementation here
}
```

### 2. Initial Sync Handler (`internal/handlers/initial_sync_handlers.go`)

```go
package handlers

import (
    "context"
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    psStripe "cyphera-api/internal/client/payment_sync/stripe"
)

// InitialSyncHandler manages initial sync operations
type InitialSyncHandler struct {
    common               *CommonServices
    stripeInitialSync    *psStripe.StripeInitialSyncService
}

// NewInitialSyncHandler creates a new initial sync handler
func NewInitialSyncHandler(common *CommonServices, stripeInitialSync *psStripe.StripeInitialSyncService) *InitialSyncHandler {
    return &InitialSyncHandler{
        common:            common,
        stripeInitialSync: stripeInitialSync,
    }
}

// StartInitialSyncRequest represents the request to start initial sync
type StartInitialSyncRequest struct {
    EntityTypes       []string                      `json:"entity_types" binding:"required"`
    BatchSize         int                           `json:"batch_size"`
    ConcurrentBatches int                           `json:"concurrent_batches"`
    ConflictStrategy  psStripe.ConflictStrategy     `json:"conflict_strategy"`
    DryRun            bool                          `json:"dry_run"`
    DateRange         *psStripe.DateRange           `json:"date_range,omitempty"`
}

// SyncSessionResponse represents the API response for sync sessions
type SyncSessionResponse struct {
    ID          string                                `json:"id"`
    Object      string                                `json:"object"`
    WorkspaceID string                                `json:"workspace_id"`
    Status      string                                `json:"status"`
    Config      psStripe.InitialSyncConfig            `json:"config"`
    Progress    map[string]*psStripe.EntityProgress   `json:"progress"`
    StartedAt   *int64                                `json:"started_at,omitempty"`
    CompletedAt *int64                                `json:"completed_at,omitempty"`
    CreatedAt   int64                                 `json:"created_at"`
    UpdatedAt   int64                                 `json:"updated_at"`
}

// StartInitialSync godoc
// @Summary Start initial synchronization from Stripe
// @Description Initiates a bulk synchronization of data from Stripe to Cyphera
// @Tags initial-sync
// @Accept json
// @Produce json
// @Param request body StartInitialSyncRequest true "Initial sync configuration"
// @Success 201 {object} SyncSessionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /sync/stripe/initial [post]
func (h *InitialSyncHandler) StartInitialSync(c *gin.Context) {
    workspaceID := c.GetHeader("X-Workspace-ID")
    parsedWorkspaceID, err := uuid.Parse(workspaceID)
    if err != nil {
        sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
        return
    }

    var request StartInitialSyncRequest
    if err := c.ShouldBindJSON(&request); err != nil {
        sendError(c, http.StatusBadRequest, "Invalid request format", err)
        return
    }

    // Set defaults
    if request.BatchSize == 0 {
        request.BatchSize = 100
    }
    if request.ConcurrentBatches == 0 {
        request.ConcurrentBatches = 3
    }
    if request.ConflictStrategy == "" {
        request.ConflictStrategy = psStripe.ConflictSkip
    }

    config := psStripe.InitialSyncConfig{
        WorkspaceID:       parsedWorkspaceID,
        EntityTypes:       request.EntityTypes,
        BatchSize:         request.BatchSize,
        ConcurrentBatches: request.ConcurrentBatches,
        ConflictStrategy:  request.ConflictStrategy,
        DryRun:            request.DryRun,
        DateRange:         request.DateRange,
    }

    session, err := h.stripeInitialSync.StartInitialSync(c.Request.Context(), config)
    if err != nil {
        sendError(c, http.StatusInternalServerError, "Failed to start initial sync", err)
        return
    }

    response := toSyncSessionResponse(session)
    sendSuccess(c, http.StatusCreated, response)
}

// GetSyncStatus godoc  
// @Summary Get synchronization status
// @Description Retrieves the current status of a sync session
// @Tags initial-sync
// @Accept json
// @Produce json
// @Param session_id path string true "Sync Session ID"
// @Success 200 {object} SyncSessionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /sync/stripe/sessions/{session_id} [get]
func (h *InitialSyncHandler) GetSyncStatus(c *gin.Context) {
    workspaceID := c.GetHeader("X-Workspace-ID")
    parsedWorkspaceID, err := uuid.Parse(workspaceID)
    if err != nil {
        sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
        return
    }

    sessionID := c.Param("session_id")
    parsedSessionID, err := uuid.Parse(sessionID)
    if err != nil {
        sendError(c, http.StatusBadRequest, "Invalid session ID format", err)
        return
    }

    session, err := h.stripeInitialSync.GetSyncStatus(c.Request.Context(), parsedSessionID)
    if err != nil {
        handleDBError(c, err, "Sync session not found")
        return
    }

    // Verify workspace access
    if session.WorkspaceID != parsedWorkspaceID {
        sendError(c, http.StatusForbidden, "Access denied to this sync session", nil)
        return
    }

    response := toSyncSessionResponse(session)
    sendSuccess(c, http.StatusOK, response)
}

// Helper function to convert session to response
func toSyncSessionResponse(session *psStripe.SyncSession) SyncSessionResponse {
    response := SyncSessionResponse{
        ID:          session.ID.String(),
        Object:      "sync_session",
        WorkspaceID: session.WorkspaceID.String(),
        Status:      string(session.Status),
        Config:      session.Config,
        Progress:    session.Progress,
        CreatedAt:   session.CreatedAt.Unix(),
        UpdatedAt:   session.UpdatedAt.Unix(),
    }

    if session.StartedAt != nil {
        startedAt := session.StartedAt.Unix()
        response.StartedAt = &startedAt
    }

    if session.CompletedAt != nil {
        completedAt := session.CompletedAt.Unix()
        response.CompletedAt = &completedAt
    }

    return response
}
```

### 3. Database Queries (`internal/db/queries/sync.sql`)

```sql
-- name: CreateSyncSession :one
INSERT INTO payment_sync_sessions (
    workspace_id, provider_name, session_type, status, entity_types, config
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetSyncSession :one
SELECT * FROM payment_sync_sessions 
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL;

-- name: GetSyncSessionByProvider :one
SELECT * FROM payment_sync_sessions 
WHERE id = $1 AND workspace_id = $2 AND provider_name = $3 AND deleted_at IS NULL;

-- name: UpdateSyncSessionStatus :one
UPDATE payment_sync_sessions 
SET status = $2, updated_at = CURRENT_TIMESTAMP,
    started_at = CASE WHEN $2 = 'running' AND started_at IS NULL THEN CURRENT_TIMESTAMP ELSE started_at END,
    completed_at = CASE WHEN $2 IN ('completed', 'failed', 'cancelled') THEN CURRENT_TIMESTAMP ELSE completed_at END
WHERE id = $1 
RETURNING *;

-- name: UpdateSyncSessionProgress :one  
UPDATE payment_sync_sessions
SET progress = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: CreateSyncEvent :one
INSERT INTO payment_sync_events (
    session_id, workspace_id, provider_name, entity_type, entity_id, external_id, 
    event_type, event_message, event_details
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: ListSyncSessions :many
SELECT * FROM payment_sync_sessions 
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListSyncSessionsByProvider :many
SELECT * FROM payment_sync_sessions 
WHERE workspace_id = $1 AND provider_name = $2 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetCustomersByExternalIDs :many
SELECT * FROM customers 
WHERE workspace_id = $1 AND external_id = ANY($2::text[]) AND deleted_at IS NULL;

-- name: GetCustomersByPaymentProvider :many
SELECT * FROM customers 
WHERE workspace_id = $1 AND payment_provider = $2 AND deleted_at IS NULL;

-- name: UpdateCustomerSyncStatus :one
UPDATE customers 
SET payment_sync_status = $2, payment_synced_at = CURRENT_TIMESTAMP, 
    payment_sync_version = payment_sync_version + 1, payment_provider = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 
RETURNING *;

-- name: UpdateProductSyncStatus :one
UPDATE products 
SET payment_sync_status = $2, payment_synced_at = CURRENT_TIMESTAMP, 
    payment_sync_version = payment_sync_version + 1, payment_provider = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 
RETURNING *;

-- name: UpdatePriceSyncStatus :one
UPDATE prices 
SET payment_sync_status = $2, payment_synced_at = CURRENT_TIMESTAMP, 
    payment_sync_version = payment_sync_version + 1, payment_provider = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 
RETURNING *;

-- name: UpdateSubscriptionSyncStatus :one
UPDATE subscriptions 
SET payment_sync_status = $2, payment_synced_at = CURRENT_TIMESTAMP, 
    payment_sync_version = payment_sync_version + 1, payment_provider = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 
RETURNING *;

-- name: GetSyncEventsBySession :many
SELECT * FROM payment_sync_events 
WHERE session_id = $1
ORDER BY occurred_at DESC
LIMIT $2 OFFSET $3;

-- name: GetSyncEventsByProvider :many
SELECT * FROM payment_sync_events 
WHERE workspace_id = $1 AND provider_name = $2
ORDER BY occurred_at DESC
LIMIT $3 OFFSET $4;
```

### 4. Server Integration (`internal/server/server.go`)

```go
// Add to existing server.go where handlers are initialized

// Initialize Stripe service for initial sync
stripeService := stripe.NewStripeService(logger.Log)
err = stripeService.Configure(ctx, map[string]string{
    "api_key":        os.Getenv("STRIPE_SECRET_KEY"),
    "webhook_secret": os.Getenv("STRIPE_WEBHOOK_SECRET"),
})
if err != nil {
    logger.Fatal("Failed to configure Stripe service", zap.Error(err))
}

// Initialize initial sync service
stripeInitialSync := stripe.NewStripeInitialSyncService(stripeService, logger.Log)

// Initialize initial sync handler
initialSyncHandler := handlers.NewInitialSyncHandler(commonServices, stripeInitialSync)

// Add to route registration
syncRoutes := r.Group("/sync")
syncRoutes.Use(validateAPIKey(dbQueries))
{
    stripeSync := syncRoutes.Group("/stripe")
    {
        stripeSync.POST("/initial", initialSyncHandler.StartInitialSync)
        stripeSync.GET("/sessions/:session_id", initialSyncHandler.GetSyncStatus)
        stripeSync.PUT("/sessions/:session_id/cancel", initialSyncHandler.CancelSync)
        stripeSync.GET("/sessions", initialSyncHandler.ListSyncSessions)
    }
}
```

## Implementation Phases

### Phase 1: Core Infrastructure (Week 1)
1. **Database Schema Updates** - Add sync tracking columns and tables
2. **Initial Sync Service** - Implement `StripeInitialSyncService` with basic session management
3. **Initial Sync Handler** - Create handler following existing patterns
4. **Database Queries** - Add required SQL queries for sync management

### Phase 2: Entity Synchronization (Week 1-2)  
1. **Customer Sync** - Implement customer synchronization with conflict resolution
2. **Product Sync** - Implement product synchronization
3. **Price Sync** - Implement price synchronization  
4. **Progress Tracking** - Add detailed progress tracking and error handling

### Phase 3: Advanced Features (Week 2)
1. **Subscription Sync** - Implement subscription synchronization
2. **Invoice Sync** - Implement invoice synchronization
3. **Conflict Resolution** - Implement advanced conflict resolution strategies
4. **Error Recovery** - Add retry logic and error recovery mechanisms

### Phase 4: Integration & Testing (Week 2-3)
1. **Webhook Integration** - Ensure sync coordinates with existing webhook handling
2. **API Documentation** - Add Swagger documentation
3. **Testing** - Comprehensive testing of sync functionality
4. **Monitoring** - Add monitoring and alerting

## Key Alignment with Existing Patterns

### ✅ Database Patterns
- Uses existing workspace multi-tenancy
- Leverages existing `external_id` fields in customers table
- Follows existing UUID primary key pattern
- Uses existing `updated_at` trigger pattern
- Maintains existing soft delete pattern with `deleted_at`

### ✅ Handler Patterns  
- Uses `CommonServices` dependency injection
- Follows existing error handling (`sendError`, `handleDBError`)
- Uses existing transaction pattern (`BeginTx`)
- Maintains existing response type naming conventions
- Includes proper godoc documentation

### ✅ Stripe Integration
- Builds on existing Stripe service implementation
- Follows existing mapping pattern between Stripe and internal types
- Maintains existing logging patterns with zap
- Uses existing payment_sync interface patterns

### ✅ API Patterns
- Follows existing URL structure and naming
- Uses existing middleware patterns (`validateAPIKey`)
- Maintains existing request/response structure
- Follows existing pagination patterns

## Benefits of This Approach

1. **Minimal Database Changes** - Leverages existing tables with minimal additions
2. **Consistent Patterns** - Follows all existing code patterns and conventions  
3. **Workspace Isolation** - Properly handles multi-tenant workspace structure
4. **Transaction Safety** - Uses existing transaction patterns for data consistency
5. **Error Handling** - Maintains existing error handling and logging patterns
6. **API Consistency** - Follows existing API design patterns
7. **Gradual Implementation** - Can be implemented incrementally without breaking changes

This revised plan ensures the initial sync functionality integrates seamlessly with your existing codebase while maintaining all established patterns and conventions. 