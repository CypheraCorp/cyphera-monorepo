# Workspace-Based Payment Sync Implementation Plan

**Version:** 1.0  
**Date:** December 2024  
**Status:** Implementation Ready

## Overview

This document outlines the implementation plan for supporting workspace-based payment synchronization, where each workspace has its own isolated Stripe account, API keys, and data. The system must support multiple concurrent sync operations across different workspaces while maintaining strict data isolation.

## Key Requirements

### Multi-Tenancy Requirements
- ✅ **Workspace Isolation**: Each workspace has completely separate payment data
- ✅ **Concurrent Operations**: Multiple workspaces can sync simultaneously
- ✅ **Independent Configuration**: Each workspace configures its own Stripe integration
- ✅ **Scalable Architecture**: Support hundreds of workspaces syncing concurrently
- ✅ **Security**: No cross-workspace data leakage
- ✅ **Performance**: Efficient resource utilization across tenants

### Business Context
- **Workspace = Merchant**: Each workspace represents a different merchant/business
- **Independent Stripe Accounts**: Each merchant has their own Stripe account
- **Isolated Data**: Products, customers, subscriptions are workspace-specific
- **Self-Service**: Workspace owners configure their own integrations

## Architecture Design

### 1. Multi-Tenant Data Model

#### 1.1 Workspace Payment Configuration
```sql
-- Stores workspace-specific payment provider configurations
CREATE TABLE workspace_payment_configs (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    provider_name VARCHAR(50) NOT NULL, -- 'stripe', 'chargebee', etc.
    
    -- Encrypted credentials
    api_key_encrypted BYTEA NOT NULL,
    webhook_secret_encrypted BYTEA,
    
    -- Provider-specific configuration
    config_data JSONB DEFAULT '{}', -- Additional settings per provider
    
    -- Webhook management
    webhook_endpoint_id TEXT, -- Provider's webhook endpoint ID
    webhook_url TEXT, -- Our webhook URL for this workspace
    
    -- Status and metadata
    is_active BOOLEAN DEFAULT true,
    last_sync_at TIMESTAMP WITH TIME ZONE,
    sync_status VARCHAR(20) DEFAULT 'never_synced', -- 'never_synced', 'active', 'failed', 'disabled'
    
    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id),
    
    UNIQUE(workspace_id, provider_name)
);

-- Index for efficient workspace lookups
CREATE INDEX idx_workspace_payment_configs_workspace_provider 
ON workspace_payment_configs(workspace_id, provider_name) 
WHERE is_active = true;

-- Index for monitoring active configurations
CREATE INDEX idx_workspace_payment_configs_active 
ON workspace_payment_configs(provider_name, is_active, last_sync_at);
```

#### 1.2 Enhanced Payment Sync Sessions
```sql
-- Updated to include workspace isolation and improved tracking
CREATE TABLE payment_sync_sessions (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    
    -- Provider information
    provider_name VARCHAR(50) NOT NULL,
    provider_config_id UUID NOT NULL REFERENCES workspace_payment_configs(id),
    
    -- Session details
    session_type VARCHAR(50) NOT NULL, -- 'initial_sync', 'incremental_sync', 'webhook_sync'
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed', 'cancelled'
    
    -- Sync configuration
    entity_types TEXT[] NOT NULL, -- ['customers', 'products', 'prices', 'subscriptions']
    sync_mode VARCHAR(20) DEFAULT 'full', -- 'full', 'incremental', 'delta'
    batch_size INTEGER DEFAULT 100,
    max_concurrent_workers INTEGER DEFAULT 3,
    
    -- Progress and results
    config JSONB DEFAULT '{}',
    progress JSONB DEFAULT '{}',
    error_summary JSONB,
    performance_metrics JSONB DEFAULT '{}', -- Timing, throughput metrics
    
    -- Timestamps
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT valid_status CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    CONSTRAINT valid_session_type CHECK (session_type IN ('initial_sync', 'incremental_sync', 'webhook_sync'))
);

-- Indexes for efficient querying
CREATE INDEX idx_payment_sync_sessions_workspace_provider 
ON payment_sync_sessions(workspace_id, provider_name, status);

CREATE INDEX idx_payment_sync_sessions_status_created 
ON payment_sync_sessions(status, created_at) 
WHERE status IN ('pending', 'running');

CREATE INDEX idx_payment_sync_sessions_workspace_active 
ON payment_sync_sessions(workspace_id, status, started_at) 
WHERE status IN ('running', 'pending');
```

#### 1.3 Sync Coordination Table
```sql
-- Prevents concurrent syncs per workspace and manages sync queuing
CREATE TABLE workspace_sync_locks (
    workspace_id UUID PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    provider_name VARCHAR(50) NOT NULL,
    current_session_id UUID REFERENCES payment_sync_sessions(id),
    locked_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    locked_by VARCHAR(100), -- Process/instance identifier
    lock_expires_at TIMESTAMP WITH TIME ZONE,
    
    UNIQUE(workspace_id, provider_name)
);

-- Index for lock cleanup
CREATE INDEX idx_workspace_sync_locks_expires 
ON workspace_sync_locks(lock_expires_at) 
WHERE lock_expires_at IS NOT NULL;
```

### 2. Multi-Tenant Service Architecture

#### 2.1 Workspace-Aware Service Factory
```go
// PaymentSyncServiceFactory manages workspace-specific service instances
type PaymentSyncServiceFactory struct {
    logger         *zap.Logger
    db             *db.Queries
    encryptionKey  []byte
    
    // Cache management
    serviceCache   map[string]ps.PaymentSyncService // workspace_id:provider -> service
    cacheMutex     sync.RWMutex
    cacheExpiry    time.Duration
    lastCleanup    time.Time
}

// GetServiceForWorkspace returns a configured service for the workspace
func (f *PaymentSyncServiceFactory) GetServiceForWorkspace(
    ctx context.Context, 
    workspaceID, provider string,
) (ps.PaymentSyncService, error) {
    cacheKey := fmt.Sprintf("%s:%s", workspaceID, provider)
    
    // Check cache first
    f.cacheMutex.RLock()
    if service, exists := f.serviceCache[cacheKey]; exists {
        f.cacheMutex.RUnlock()
        return service, nil
    }
    f.cacheMutex.RUnlock()
    
    // Load configuration from database
    config, err := f.loadWorkspaceConfig(ctx, workspaceID, provider)
    if err != nil {
        return nil, fmt.Errorf("failed to load workspace config: %w", err)
    }
    
    // Create provider-specific service
    service, err := f.createProviderService(provider, config)
    if err != nil {
        return nil, fmt.Errorf("failed to create service: %w", err)
    }
    
    // Cache the service
    f.cacheMutex.Lock()
    f.serviceCache[cacheKey] = service
    f.cacheMutex.Unlock()
    
    return service, nil
}

// WorkspacePaymentConfig represents decrypted configuration
type WorkspacePaymentConfig struct {
    WorkspaceID      string
    ProviderName     string
    APIKey           string
    WebhookSecret    string
    ConfigData       map[string]interface{}
    WebhookEndpointID string
    IsActive         bool
}

func (f *PaymentSyncServiceFactory) loadWorkspaceConfig(
    ctx context.Context, 
    workspaceID, provider string,
) (*WorkspacePaymentConfig, error) {
    // Get encrypted config from database
    dbConfig, err := f.db.GetWorkspacePaymentConfig(ctx, db.GetWorkspacePaymentConfigParams{
        WorkspaceID:  uuid.MustParse(workspaceID),
        ProviderName: provider,
    })
    if err != nil {
        return nil, fmt.Errorf("workspace payment config not found: %w", err)
    }
    
    if !dbConfig.IsActive {
        return nil, fmt.Errorf("payment provider %s is disabled for workspace %s", provider, workspaceID)
    }
    
    // Decrypt sensitive data
    apiKey, err := f.decrypt(dbConfig.ApiKeyEncrypted)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt API key: %w", err)
    }
    
    webhookSecret, err := f.decrypt(dbConfig.WebhookSecretEncrypted)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt webhook secret: %w", err)
    }
    
    var configData map[string]interface{}
    if len(dbConfig.ConfigData) > 0 {
        if err := json.Unmarshal(dbConfig.ConfigData, &configData); err != nil {
            return nil, fmt.Errorf("failed to parse config data: %w", err)
        }
    }
    
    return &WorkspacePaymentConfig{
        WorkspaceID:       workspaceID,
        ProviderName:      provider,
        APIKey:            apiKey,
        WebhookSecret:     webhookSecret,
        ConfigData:        configData,
        WebhookEndpointID: dbConfig.WebhookEndpointID.String,
        IsActive:          dbConfig.IsActive,
    }, nil
}
```

#### 2.2 Workspace-Specific Stripe Service
```go
// WorkspaceStripeService implements PaymentSyncService for a specific workspace
type WorkspaceStripeService struct {
    workspaceID   string
    client        *stripe.Client
    webhookSecret string
    logger        *zap.Logger
    db            *db.Queries
    config        *WorkspacePaymentConfig
}

func NewWorkspaceStripeService(
    workspaceID string,
    config *WorkspacePaymentConfig,
    logger *zap.Logger,
    db *db.Queries,
) *WorkspaceStripeService {
    client := stripe.NewClient(config.APIKey, nil)
    
    return &WorkspaceStripeService{
        workspaceID:   workspaceID,
        client:        client,
        webhookSecret: config.WebhookSecret,
        logger:        logger.With(zap.String("workspace_id", workspaceID)),
        db:            db,
        config:        config,
    }
}

// GetServiceName returns the provider name with workspace context
func (s *WorkspaceStripeService) GetServiceName() string {
    return fmt.Sprintf("stripe:%s", s.workspaceID)
}

// StartInitialSync with workspace isolation
func (s *WorkspaceStripeService) StartInitialSync(
    ctx context.Context, 
    workspaceID string, 
    config ps.InitialSyncConfig,
) (ps.SyncSession, error) {
    // Validate workspace ID matches service
    if workspaceID != s.workspaceID {
        return ps.SyncSession{}, fmt.Errorf("workspace ID mismatch: expected %s, got %s", s.workspaceID, workspaceID)
    }
    
    // Acquire workspace sync lock
    lock, err := s.acquireWorkspaceLock(ctx)
    if err != nil {
        return ps.SyncSession{}, fmt.Errorf("failed to acquire sync lock: %w", err)
    }
    defer lock.Release()
    
    // Continue with sync implementation...
    return s.executeInitialSync(ctx, config)
}
```

### 3. Concurrency and Resource Management

#### 3.1 Workspace Sync Coordination
```go
// WorkspaceSyncLock manages concurrent sync operations per workspace
type WorkspaceSyncLock struct {
    workspaceID string
    provider    string
    sessionID   string
    db          *db.Queries
    logger      *zap.Logger
    acquired    bool
}

func (s *WorkspaceStripeService) acquireWorkspaceLock(ctx context.Context) (*WorkspaceSyncLock, error) {
    lock := &WorkspaceSyncLock{
        workspaceID: s.workspaceID,
        provider:    "stripe",
        db:          s.db,
        logger:      s.logger,
    }
    
    // Try to acquire lock with timeout
    lockCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    for {
        acquired, err := lock.tryAcquire(lockCtx)
        if err != nil {
            return nil, err
        }
        if acquired {
            return lock, nil
        }
        
        // Wait before retrying
        select {
        case <-lockCtx.Done():
            return nil, fmt.Errorf("timeout acquiring workspace sync lock")
        case <-time.After(5 * time.Second):
            continue
        }
    }
}

func (l *WorkspaceSyncLock) tryAcquire(ctx context.Context) (bool, error) {
    instanceID := os.Getenv("INSTANCE_ID") // Container/pod identifier
    expiresAt := time.Now().Add(2 * time.Hour) // Max sync duration
    
    // Attempt to insert lock record
    err := l.db.CreateWorkspaceSyncLock(ctx, db.CreateWorkspaceSyncLockParams{
        WorkspaceID:     uuid.MustParse(l.workspaceID),
        ProviderName:    l.provider,
        LockedBy:        instanceID,
        LockExpiresAt:   pgtype.Timestamptz{Time: expiresAt, Valid: true},
    })
    
    if err != nil {
        // Check if it's a conflict (lock already exists)
        if isUniqueConstraintError(err) {
            return false, nil // Lock not acquired, but not an error
        }
        return false, fmt.Errorf("failed to create lock: %w", err)
    }
    
    l.acquired = true
    return true, nil
}

func (l *WorkspaceSyncLock) Release() error {
    if !l.acquired {
        return nil
    }
    
    return l.db.DeleteWorkspaceSyncLock(context.Background(), db.DeleteWorkspaceSyncLockParams{
        WorkspaceID:  uuid.MustParse(l.workspaceID),
        ProviderName: l.provider,
    })
}
```

#### 3.2 Resource Pool Management
```go
// SyncResourceManager manages system resources across workspaces
type SyncResourceManager struct {
    maxConcurrentSyncs   int
    maxWorkersPerSync    int
    activeSyncs          map[string]*SyncContext
    resourceSemaphore    chan struct{}
    mutex                sync.RWMutex
    logger               *zap.Logger
}

type SyncContext struct {
    WorkspaceID string
    SessionID   string
    StartedAt   time.Time
    Workers     int
}

func NewSyncResourceManager(maxConcurrentSyncs, maxWorkersPerSync int, logger *zap.Logger) *SyncResourceManager {
    return &SyncResourceManager{
        maxConcurrentSyncs: maxConcurrentSyncs,
        maxWorkersPerSync:  maxWorkersPerSync,
        activeSyncs:        make(map[string]*SyncContext),
        resourceSemaphore:  make(chan struct{}, maxConcurrentSyncs),
        logger:             logger,
    }
}

func (rm *SyncResourceManager) AcquireResources(ctx context.Context, workspaceID, sessionID string, requestedWorkers int) error {
    // Limit workers per sync
    workers := min(requestedWorkers, rm.maxWorkersPerSync)
    
    // Acquire global sync slot
    select {
    case rm.resourceSemaphore <- struct{}{}:
        // Acquired slot
    case <-ctx.Done():
        return fmt.Errorf("timeout acquiring sync resources")
    }
    
    // Register sync context
    rm.mutex.Lock()
    rm.activeSyncs[sessionID] = &SyncContext{
        WorkspaceID: workspaceID,
        SessionID:   sessionID,
        StartedAt:   time.Now(),
        Workers:     workers,
    }
    rm.mutex.Unlock()
    
    rm.logger.Info("Acquired sync resources",
        zap.String("workspace_id", workspaceID),
        zap.String("session_id", sessionID),
        zap.Int("workers", workers),
        zap.Int("active_syncs", len(rm.activeSyncs)))
    
    return nil
}

func (rm *SyncResourceManager) ReleaseResources(sessionID string) {
    rm.mutex.Lock()
    delete(rm.activeSyncs, sessionID)
    rm.mutex.Unlock()
    
    // Release global sync slot
    <-rm.resourceSemaphore
    
    rm.logger.Info("Released sync resources",
        zap.String("session_id", sessionID),
        zap.Int("active_syncs", len(rm.activeSyncs)))
}
```

### 4. API Design for Multi-Tenant Operations

#### 4.1 Workspace Configuration Endpoints
```go
// Workspace payment provider configuration endpoints
type WorkspacePaymentHandlers struct {
    db               *db.Queries
    logger           *zap.Logger
    encryptionKey    []byte
    serviceFactory   *PaymentSyncServiceFactory
}

// POST /api/v1/workspaces/{workspace_id}/payment-providers/{provider}
// Configure payment provider for workspace
func (h *WorkspacePaymentHandlers) ConfigurePaymentProvider(c *gin.Context) {
    workspaceID := c.Param("workspace_id")
    provider := c.Param("provider")
    
    type ConfigRequest struct {
        APIKey        string                 `json:"api_key" binding:"required"`
        WebhookSecret string                 `json:"webhook_secret,omitempty"`
        ConfigData    map[string]interface{} `json:"config_data,omitempty"`
    }
    
    var req ConfigRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
        return
    }
    
    // Validate API key by testing connection
    if err := h.validateProviderCredentials(provider, req.APIKey); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{
            Error: fmt.Sprintf("invalid %s credentials: %v", provider, err),
        })
        return
    }
    
    // Encrypt sensitive data
    encryptedAPIKey, err := h.encrypt(req.APIKey)
    if err != nil {
        c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "encryption failed"})
        return
    }
    
    encryptedWebhookSecret, err := h.encrypt(req.WebhookSecret)
    if err != nil {
        c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "encryption failed"})
        return
    }
    
    configJSON, _ := json.Marshal(req.ConfigData)
    
    // Create webhook endpoint for this workspace
    webhookURL := fmt.Sprintf("%s/webhooks/%s/%s", 
        os.Getenv("BASE_URL"), provider, workspaceID)
    
    // Store configuration
    config, err := h.db.UpsertWorkspacePaymentConfig(c.Request.Context(), db.UpsertWorkspacePaymentConfigParams{
        WorkspaceID:              uuid.MustParse(workspaceID),
        ProviderName:             provider,
        ApiKeyEncrypted:          encryptedAPIKey,
        WebhookSecretEncrypted:   encryptedWebhookSecret,
        ConfigData:               configJSON,
        WebhookUrl:               pgtype.Text{String: webhookURL, Valid: true},
        IsActive:                 true,
    })
    
    if err != nil {
        h.logger.Error("Failed to save payment config", zap.Error(err))
        c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to save configuration"})
        return
    }
    
    // Set up webhook endpoint with the provider
    if err := h.setupProviderWebhook(provider, req.APIKey, webhookURL); err != nil {
        h.logger.Warn("Failed to setup webhook endpoint", zap.Error(err))
        // Continue anyway - webhook can be set up manually
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message":     "Payment provider configured successfully",
        "provider":    provider,
        "webhook_url": webhookURL,
        "config_id":   config.ID.String(),
    })
}

// GET /api/v1/workspaces/{workspace_id}/payment-providers
// List configured payment providers for workspace
func (h *WorkspacePaymentHandlers) ListPaymentProviders(c *gin.Context) {
    workspaceID := c.Param("workspace_id")
    
    configs, err := h.db.ListWorkspacePaymentConfigs(c.Request.Context(), uuid.MustParse(workspaceID))
    if err != nil {
        c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to list providers"})
        return
    }
    
    type ProviderInfo struct {
        Provider          string    `json:"provider"`
        IsActive          bool      `json:"is_active"`
        LastSyncAt        *string   `json:"last_sync_at,omitempty"`
        SyncStatus        string    `json:"sync_status"`
        WebhookURL        string    `json:"webhook_url,omitempty"`
        ConfiguredAt      string    `json:"configured_at"`
    }
    
    providers := make([]ProviderInfo, len(configs))
    for i, config := range configs {
        providers[i] = ProviderInfo{
            Provider:     config.ProviderName,
            IsActive:     config.IsActive,
            SyncStatus:   config.SyncStatus,
            WebhookURL:   config.WebhookUrl.String,
            ConfiguredAt: config.CreatedAt.Time.Format(time.RFC3339),
        }
        
        if config.LastSyncAt.Valid {
            syncTime := config.LastSyncAt.Time.Format(time.RFC3339)
            providers[i].LastSyncAt = &syncTime
        }
    }
    
    c.JSON(http.StatusOK, gin.H{
        "providers": providers,
        "total":     len(providers),
    })
}
```

#### 4.2 Enhanced Sync Management Endpoints
```go
// POST /api/v1/workspaces/{workspace_id}/sync/{provider}/initial
// Start initial sync for specific workspace
func (h *PaymentSyncHandlers) StartWorkspaceInitialSync(c *gin.Context) {
    workspaceID := c.Param("workspace_id")
    provider := c.Param("provider")
    
    // Validate workspace exists and user has access
    if err := h.validateWorkspaceAccess(c, workspaceID); err != nil {
        c.JSON(http.StatusForbidden, ErrorResponse{Error: "access denied"})
        return
    }
    
    // Get workspace-specific service
    syncService, err := h.serviceFactory.GetServiceForWorkspace(
        c.Request.Context(), workspaceID, provider,
    )
    if err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{
            Error: fmt.Sprintf("provider %s not configured for workspace", provider),
        })
        return
    }
    
    // Parse request
    var req InitialSyncRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
        return
    }
    
    // Build configuration
    config := ps.InitialSyncConfig{
        BatchSize:     req.BatchSize,
        EntityTypes:   req.EntityTypes,
        FullSync:      req.FullSync,
        MaxRetries:    3,
        RetryDelay:    2,
    }
    
    // Apply defaults
    if config.BatchSize == 0 {
        config.BatchSize = 100
    }
    if len(config.EntityTypes) == 0 {
        config.EntityTypes = []string{"customers", "products", "prices", "subscriptions"}
    }
    
    h.logger.Info("Starting workspace initial sync",
        zap.String("workspace_id", workspaceID),
        zap.String("provider", provider),
        zap.Any("config", config))
    
    // Start sync
    session, err := syncService.StartInitialSync(c.Request.Context(), workspaceID, config)
    if err != nil {
        h.logger.Error("Failed to start initial sync", zap.Error(err))
        c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to start sync"})
        return
    }
    
    c.JSON(http.StatusCreated, InitialSyncResponse{
        SessionID:   session.ID,
        Status:      session.Status,
        Provider:    session.Provider,
        EntityTypes: session.EntityTypes,
        Config:      session.Config,
        CreatedAt:   time.Unix(session.CreatedAt, 0).Format(time.RFC3339),
    })
}

// GET /api/v1/workspaces/{workspace_id}/sync/sessions
// List sync sessions for workspace
func (h *PaymentSyncHandlers) ListWorkspaceSyncSessions(c *gin.Context) {
    workspaceID := c.Param("workspace_id")
    
    // Parse query parameters
    provider := c.Query("provider")
    status := c.Query("status")
    limitStr := c.DefaultQuery("limit", "50")
    offsetStr := c.DefaultQuery("offset", "0")
    
    limit, _ := strconv.Atoi(limitStr)
    offset, _ := strconv.Atoi(offsetStr)
    
    if limit <= 0 || limit > 100 {
        limit = 50
    }
    if offset < 0 {
        offset = 0
    }
    
    wsID := uuid.MustParse(workspaceID)
    ctx := c.Request.Context()
    
    var sessions []db.PaymentSyncSession
    var total int64
    var err error
    
    // Query based on filters
    if provider != "" && status != "" {
        sessions, err = h.db.ListSyncSessionsByWorkspaceProviderStatus(ctx, 
            db.ListSyncSessionsByWorkspaceProviderStatusParams{
                WorkspaceID:  wsID,
                ProviderName: provider,
                Status:       status,
                Limit:        int32(limit),
                Offset:       int32(offset),
            })
    } else if provider != "" {
        sessions, err = h.db.ListSyncSessionsByWorkspaceProvider(ctx,
            db.ListSyncSessionsByWorkspaceProviderParams{
                WorkspaceID:  wsID,
                ProviderName: provider,
                Limit:        int32(limit),
                Offset:       int32(offset),
            })
    } else {
        sessions, err = h.db.ListSyncSessionsByWorkspace(ctx,
            db.ListSyncSessionsByWorkspaceParams{
                WorkspaceID: wsID,
                Limit:       int32(limit),
                Offset:      int32(offset),
            })
    }
    
    if err != nil {
        h.logger.Error("Failed to list sync sessions", zap.Error(err))
        c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to list sessions"})
        return
    }
    
    // Get total count
    total, _ = h.db.CountSyncSessionsByWorkspace(ctx, wsID)
    
    // Convert to response format
    responses := make([]SyncSessionResponse, len(sessions))
    for i, session := range sessions {
        responses[i] = h.mapSyncSessionToResponse(session)
    }
    
    c.JSON(http.StatusOK, gin.H{
        "sessions":     responses,
        "total":        total,
        "limit":        limit,
        "offset":       offset,
        "workspace_id": workspaceID,
    })
}
```

### 5. Webhook Management for Multi-Tenancy

#### 5.1 Workspace-Specific Webhook Routing
```go
// Webhook handler that routes to correct workspace
func (h *WebhookHandlers) HandleWorkspaceWebhook(c *gin.Context) {
    provider := c.Param("provider")
    workspaceID := c.Param("workspace_id")
    
    h.logger.Info("Received workspace webhook",
        zap.String("provider", provider),
        zap.String("workspace_id", workspaceID))
    
    // Get workspace-specific service
    syncService, err := h.serviceFactory.GetServiceForWorkspace(
        c.Request.Context(), workspaceID, provider,
    )
    if err != nil {
        h.logger.Error("Failed to get service for webhook", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{"error": "workspace not configured"})
        return
    }
    
    // Read request body
    body, err := io.ReadAll(c.Request.Body)
    if err != nil {
        h.logger.Error("Failed to read webhook body", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
        return
    }
    
    // Get signature header
    signatureHeader := c.GetHeader("Stripe-Signature")
    if signatureHeader == "" {
        h.logger.Error("Missing webhook signature")
        c.JSON(http.StatusBadRequest, gin.H{"error": "missing signature"})
        return
    }
    
    // Handle webhook with workspace-specific service
    event, err := syncService.HandleWebhook(c.Request.Context(), body, signatureHeader)
    if err != nil {
        h.logger.Error("Webhook validation failed", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook"})
        return
    }
    
    // Process webhook asynchronously for this workspace
    go h.processWebhookEvent(workspaceID, provider, event)
    
    c.JSON(http.StatusOK, gin.H{"received": true})
}

// URL pattern: /webhooks/{provider}/{workspace_id}
// Example: /webhooks/stripe/01234567-89ab-cdef-0123-456789abcdef
```

### 6. Performance and Monitoring

#### 6.1 Workspace-Specific Metrics
```go
// Metrics collection for multi-tenant operations
type WorkspaceSyncMetrics struct {
    prometheus.Collector
    
    activeSyncs     *prometheus.GaugeVec
    syncDuration    *prometheus.HistogramVec
    syncThroughput  *prometheus.CounterVec
    syncErrors      *prometheus.CounterVec
    webhookEvents   *prometheus.CounterVec
}

func NewWorkspaceSyncMetrics() *WorkspaceSyncMetrics {
    return &WorkspaceSyncMetrics{
        activeSyncs: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "workspace_sync_active_sessions_total",
                Help: "Number of active sync sessions per workspace",
            },
            []string{"workspace_id", "provider", "session_type"},
        ),
        
        syncDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "workspace_sync_duration_seconds",
                Help:    "Duration of sync operations by workspace",
                Buckets: prometheus.ExponentialBuckets(1, 2, 10),
            },
            []string{"workspace_id", "provider", "entity_type", "status"},
        ),
        
        syncThroughput: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "workspace_sync_entities_processed_total",
                Help: "Total entities processed by workspace sync",
            },
            []string{"workspace_id", "provider", "entity_type"},
        ),
        
        syncErrors: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "workspace_sync_errors_total",
                Help: "Total sync errors by workspace",
            },
            []string{"workspace_id", "provider", "error_type"},
        ),
        
        webhookEvents: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "workspace_webhook_events_total",
                Help: "Total webhook events received by workspace",
            },
            []string{"workspace_id", "provider", "event_type", "status"},
        ),
    }
}
```

#### 6.2 Performance Monitoring Dashboard
```yaml
# Grafana Dashboard Configuration for Workspace Sync Monitoring
apiVersion: 1
datasources:
  - name: prometheus
    type: prometheus
    url: http://prometheus:9090

dashboard:
  title: "Workspace Payment Sync Monitoring"
  panels:
    - title: "Active Syncs by Workspace"
      type: "stat"
      targets:
        - expr: 'sum by (workspace_id) (workspace_sync_active_sessions_total)'
    
    - title: "Sync Success Rate"
      type: "stat"
      targets:
        - expr: 'rate(workspace_sync_duration_seconds_count{status="completed"}[5m]) / rate(workspace_sync_duration_seconds_count[5m])'
    
    - title: "Sync Duration by Workspace"
      type: "graph"
      targets:
        - expr: 'histogram_quantile(0.95, workspace_sync_duration_seconds_bucket)'
    
    - title: "Webhook Processing Rate"
      type: "graph"  
      targets:
        - expr: 'rate(workspace_webhook_events_total[5m])'
    
    - title: "Entity Processing Throughput"
      type: "graph"
      targets:
        - expr: 'rate(workspace_sync_entities_processed_total[5m])'
```

## Implementation Phases

### Phase 1: Foundation (Week 1-2)
- ✅ **Database Schema**: Create workspace configuration tables
- ✅ **Encryption**: Implement credential encryption/decryption
- ✅ **Service Factory**: Build workspace-aware service factory
- ✅ **Basic API**: Workspace configuration endpoints

### Phase 2: Core Sync Engine (Week 2-3)
- ✅ **Workspace Service**: Implement workspace-specific Stripe service
- ✅ **Sync Coordination**: Add workspace locking and resource management
- ✅ **Enhanced API**: Full workspace sync management endpoints
- ✅ **Testing**: Unit tests for multi-tenant operations

### Phase 3: Webhook Integration (Week 3-4)
- ✅ **Webhook Routing**: Workspace-specific webhook endpoints
- ✅ **Event Processing**: Multi-tenant webhook event handling
- ✅ **Provider Setup**: Automated webhook endpoint creation
- ✅ **Integration Testing**: End-to-end webhook testing

### Phase 4: Production Hardening (Week 4-5)
- ✅ **Monitoring**: Workspace-specific metrics and dashboards
- ✅ **Performance**: Resource optimization and scaling
- ✅ **Security**: Security audit and penetration testing
- ✅ **Documentation**: Operational runbooks and API docs

### Phase 5: Advanced Features (Week 5-6)
- ✅ **Auto-scaling**: Dynamic resource allocation
- ✅ **Sync Scheduling**: Configurable sync schedules per workspace
- ✅ **Conflict Resolution**: Advanced conflict handling
- ✅ **Analytics**: Sync performance analytics per workspace

## Security Considerations

### Data Isolation
- **Database Level**: All queries include workspace_id filters
- **Service Level**: Services bound to specific workspaces
- **API Level**: Authorization checks for workspace access
- **Webhook Level**: Workspace-specific webhook validation

### Credential Management
- **Encryption at Rest**: AES-256 encryption for API keys
- **Key Rotation**: Support for credential rotation
- **Access Logs**: Audit all configuration changes
- **Principle of Least Privilege**: Minimal required permissions

### Resource Protection
- **Rate Limiting**: Per-workspace API rate limits
- **Resource Quotas**: Configurable sync limits per workspace
- **DDoS Protection**: Webhook endpoint protection
- **Error Handling**: No information leakage in error messages

## Success Criteria

- ✅ **Isolation**: Zero cross-workspace data contamination
- ✅ **Performance**: Support 100+ concurrent workspace syncs
- ✅ **Reliability**: 99.9% sync success rate per workspace
- ✅ **Scalability**: Linear performance scaling with workspace count
- ✅ **Security**: Pass security audit with zero critical findings
- ✅ **Usability**: Self-service configuration with clear documentation

This implementation plan provides a comprehensive foundation for supporting workspace-based payment synchronization while maintaining strict isolation, security, and performance requirements. 