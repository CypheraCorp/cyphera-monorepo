-- Payment Sync Sessions Queries

-- name: CreateSyncSession :one
INSERT INTO payment_sync_sessions (
    workspace_id, 
    provider_name, 
    session_type, 
    status, 
    entity_types, 
    config
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

-- name: UpdateSyncSessionError :one
UPDATE payment_sync_sessions
SET error_summary = $2, status = 'failed', updated_at = CURRENT_TIMESTAMP,
    completed_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

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

-- name: ListSyncSessionsByStatus :many
SELECT * FROM payment_sync_sessions 
WHERE workspace_id = $1 AND status = $2 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetActiveSyncSessionsByProvider :many
SELECT * FROM payment_sync_sessions 
WHERE workspace_id = $1 AND provider_name = $2 AND status IN ('pending', 'running') AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: CountSyncSessions :one
SELECT COUNT(*) FROM payment_sync_sessions 
WHERE workspace_id = $1 AND deleted_at IS NULL;

-- name: CountSyncSessionsByProvider :one
SELECT COUNT(*) FROM payment_sync_sessions 
WHERE workspace_id = $1 AND provider_name = $2 AND deleted_at IS NULL;

-- name: GetLatestSyncSessionByProvider :one
SELECT * FROM payment_sync_sessions 
WHERE workspace_id = $1 AND provider_name = $2 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: DeleteSyncSession :exec
UPDATE payment_sync_sessions
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL;

-- Payment Sync Events Queries

-- name: CreateSyncEvent :one
INSERT INTO payment_sync_events (
    session_id, 
    workspace_id, 
    provider_name, 
    entity_type, 
    entity_id, 
    external_id, 
    event_type, 
    event_message, 
    event_details
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: CreateWebhookEvent :one
-- NEW: Create webhook-specific sync event with all webhook fields
INSERT INTO payment_sync_events (
    workspace_id, 
    provider_name, 
    entity_type, 
    entity_id, 
    external_id, 
    event_type, 
    event_message, 
    event_details,
    webhook_event_id,
    provider_account_id,
    idempotency_key,
    processing_attempts,
    signature_valid
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
) RETURNING *;

-- name: GetSyncEvent :one
SELECT * FROM payment_sync_events 
WHERE id = $1;

-- name: GetWebhookEventByIdempotencyKey :one
-- NEW: Check for duplicate webhook processing using idempotency key
SELECT * FROM payment_sync_events 
WHERE workspace_id = $1 
  AND provider_name = $2 
  AND idempotency_key = $3
  AND idempotency_key IS NOT NULL
ORDER BY occurred_at DESC
LIMIT 1;

-- name: GetWebhookEventByProviderEventID :one
-- NEW: Get webhook event by provider's event ID
SELECT * FROM payment_sync_events 
WHERE workspace_id = $1 
  AND provider_name = $2 
  AND webhook_event_id = $3
  AND webhook_event_id IS NOT NULL
ORDER BY occurred_at DESC
LIMIT 1;

-- name: UpdateWebhookEventProcessingAttempts :one
-- NEW: Update processing attempts for retry logic
UPDATE payment_sync_events
SET 
    processing_attempts = processing_attempts + 1,
    event_details = COALESCE(event_details, '{}'::jsonb) || jsonb_build_object(
        'last_attempt_at', EXTRACT(epoch FROM CURRENT_TIMESTAMP),
        'retry_reason', $2
    )
WHERE id = $1
RETURNING *;

-- name: ListSyncEventsBySession :many
SELECT * FROM payment_sync_events 
WHERE session_id = $1
ORDER BY occurred_at DESC
LIMIT $2 OFFSET $3;

-- name: ListSyncEventsByProvider :many
SELECT * FROM payment_sync_events 
WHERE workspace_id = $1 AND provider_name = $2
ORDER BY occurred_at DESC
LIMIT $3 OFFSET $4;

-- name: ListWebhookEventsByProvider :many
-- NEW: List webhook events specifically (those with webhook_event_id)
SELECT * FROM payment_sync_events 
WHERE workspace_id = $1 
  AND provider_name = $2
  AND webhook_event_id IS NOT NULL
ORDER BY occurred_at DESC
LIMIT $3 OFFSET $4;

-- name: ListFailedWebhookEvents :many
-- NEW: List webhook events that failed processing
SELECT * FROM payment_sync_events 
WHERE workspace_id = $1 
  AND provider_name = $2
  AND webhook_event_id IS NOT NULL
  AND event_type = 'webhook_processing_failed'
  AND processing_attempts >= $3
ORDER BY occurred_at DESC
LIMIT $4 OFFSET $5;

-- name: ListSyncEventsByEntityType :many
SELECT * FROM payment_sync_events 
WHERE session_id = $1 AND entity_type = $2
ORDER BY occurred_at DESC
LIMIT $3 OFFSET $4;

-- name: ListSyncEventsByEventType :many
SELECT * FROM payment_sync_events 
WHERE session_id = $1 AND event_type = $2
ORDER BY occurred_at DESC
LIMIT $3 OFFSET $4;

-- name: GetSyncEventsByExternalID :many
SELECT * FROM payment_sync_events 
WHERE workspace_id = $1 AND provider_name = $2 AND external_id = $3
ORDER BY occurred_at DESC;

-- name: CountSyncEventsBySession :one
SELECT COUNT(*) FROM payment_sync_events 
WHERE session_id = $1;

-- name: CountSyncEventsBySessionAndType :one
SELECT COUNT(*) FROM payment_sync_events 
WHERE session_id = $1 AND event_type = $2;

-- name: CountSyncEventsByEntityType :one
SELECT COUNT(*) FROM payment_sync_events 
WHERE session_id = $1 AND entity_type = $2;

-- name: CountWebhookEventsByProvider :one
-- NEW: Count webhook events for a provider
SELECT COUNT(*) FROM payment_sync_events 
WHERE workspace_id = $1 
  AND provider_name = $2 
  AND webhook_event_id IS NOT NULL;

-- name: GetSyncEventsSummaryBySession :one
SELECT 
    session_id,
    COUNT(*) as total_events,
    COUNT(CASE WHEN event_type = 'sync_completed' THEN 1 END) as completed_count,
    COUNT(CASE WHEN event_type = 'sync_failed' THEN 1 END) as failed_count,
    COUNT(CASE WHEN event_type = 'sync_skipped' THEN 1 END) as skipped_count
FROM payment_sync_events 
WHERE session_id = $1
GROUP BY session_id;

-- name: GetWebhookEventsSummaryByProvider :one
-- NEW: Get webhook processing summary for a provider
SELECT 
    provider_name,
    COUNT(*) as total_webhook_events,
    COUNT(CASE WHEN event_type = 'webhook_processed_successfully' THEN 1 END) as successful_count,
    COUNT(CASE WHEN event_type = 'webhook_processing_failed' THEN 1 END) as failed_count,
    COUNT(CASE WHEN processing_attempts > 1 THEN 1 END) as retry_count,
    COUNT(CASE WHEN signature_valid = false THEN 1 END) as invalid_signature_count,
    MAX(occurred_at) as last_webhook_at
FROM payment_sync_events 
WHERE workspace_id = $1 
  AND provider_name = $2 
  AND webhook_event_id IS NOT NULL
GROUP BY provider_name;

-- name: GetLatestSyncEventsByEntityType :many
SELECT DISTINCT ON (entity_type, external_id) 
    *
FROM payment_sync_events 
WHERE session_id = $1
ORDER BY entity_type, external_id, occurred_at DESC;

-- name: DeleteSyncEventsBySession :exec
DELETE FROM payment_sync_events
WHERE session_id = $1;

-- Payment Sync Status Queries for Entities

-- name: GetCustomersByPaymentProvider :many
SELECT * FROM customers 
WHERE payment_provider = $1 AND deleted_at IS NULL;

-- name: GetCustomersByPaymentSyncStatus :many
SELECT * FROM customers 
WHERE payment_sync_status = $1 AND deleted_at IS NULL;

-- name: GetWorkspaceCustomersByPaymentProvider :many
SELECT c.* FROM customers c
INNER JOIN workspace_customers wc ON c.id = wc.customer_id
WHERE wc.workspace_id = $1 AND c.payment_provider = $2 AND wc.deleted_at IS NULL AND c.deleted_at IS NULL;

-- name: GetWorkspaceCustomersByPaymentSyncStatus :many
SELECT c.* FROM customers c
INNER JOIN workspace_customers wc ON c.id = wc.customer_id
WHERE wc.workspace_id = $1 AND c.payment_sync_status = $2 AND wc.deleted_at IS NULL AND c.deleted_at IS NULL;

-- name: GetProductsByPaymentProvider :many
SELECT * FROM products 
WHERE workspace_id = $1 AND payment_provider = $2 AND deleted_at IS NULL;

-- name: GetProductsByPaymentSyncStatus :many
SELECT * FROM products 
WHERE workspace_id = $1 AND payment_sync_status = $2 AND deleted_at IS NULL;

-- name: GetPricesByPaymentProvider :many
SELECT p.* FROM prices p
JOIN products pr ON p.product_id = pr.id
WHERE pr.workspace_id = $1 AND p.payment_provider = $2 AND p.deleted_at IS NULL AND pr.deleted_at IS NULL;

-- name: GetPricesByPaymentSyncStatus :many
SELECT p.* FROM prices p
JOIN products pr ON p.product_id = pr.id
WHERE pr.workspace_id = $1 AND p.payment_sync_status = $2 AND p.deleted_at IS NULL AND pr.deleted_at IS NULL;

-- name: GetSubscriptionsByPaymentProvider :many
SELECT * FROM subscriptions 
WHERE workspace_id = $1 AND payment_provider = $2 AND deleted_at IS NULL;

-- name: GetSubscriptionsByPaymentSyncStatus :many
SELECT * FROM subscriptions 
WHERE workspace_id = $1 AND payment_sync_status = $2 AND deleted_at IS NULL;

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

-- name: GetEntitiesBySyncStatusAndProvider :many
SELECT 
    'customer' as entity_type,
    c.id as entity_id,
    c.external_id,
    c.payment_sync_status,
    c.payment_synced_at,
    c.payment_provider
FROM customers c
INNER JOIN workspace_customers wc ON c.id = wc.customer_id
WHERE wc.workspace_id = $1 AND c.payment_provider = $2 AND c.payment_sync_status = $3 AND wc.deleted_at IS NULL AND c.deleted_at IS NULL

UNION ALL

SELECT 
    'product' as entity_type,
    p.id as entity_id,
    p.external_id,
    p.payment_sync_status,
    p.payment_synced_at,
    p.payment_provider
FROM products p
WHERE p.workspace_id = $1 AND p.payment_provider = $2 AND p.payment_sync_status = $3 AND p.deleted_at IS NULL

UNION ALL

SELECT 
    'price' as entity_type,
    pr.id as entity_id,
    pr.external_id,
    pr.payment_sync_status,
    pr.payment_synced_at,
    pr.payment_provider
FROM prices pr
JOIN products prod ON pr.product_id = prod.id
WHERE prod.workspace_id = $1 AND pr.payment_provider = $2 AND pr.payment_sync_status = $3 AND pr.deleted_at IS NULL AND prod.deleted_at IS NULL

UNION ALL

SELECT 
    'subscription' as entity_type,
    s.id as entity_id,
    s.external_id,
    s.payment_sync_status,
    s.payment_synced_at,
    s.payment_provider
FROM subscriptions s
WHERE s.workspace_id = $1 AND s.payment_provider = $2 AND s.payment_sync_status = $3 AND s.deleted_at IS NULL

ORDER BY payment_synced_at DESC;

-- Workspace Provider Configuration Queries (using workspace metadata for now)

-- name: GetWorkspaceProviderConfig :one
SELECT 
    w.id,
    w.metadata,
    COALESCE(w.metadata ->> 'payment_providers', '{}') as provider_configs
FROM workspaces w
WHERE w.id = $1 AND w.deleted_at IS NULL;

-- name: UpdateWorkspaceProviderConfig :one
UPDATE workspaces
SET 
    metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object('payment_providers', $2),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GetWorkspacesByProvider :many
SELECT w.*
FROM workspaces w
WHERE w.metadata -> 'payment_providers' ? $1 
  AND w.deleted_at IS NULL
ORDER BY w.created_at DESC;

-- Bulk Operations for Initial Sync

-- name: BulkUpdateCustomerSyncStatus :exec
UPDATE customers 
SET payment_sync_status = $1, 
    payment_synced_at = CURRENT_TIMESTAMP, 
    payment_sync_version = payment_sync_version + 1, 
    payment_provider = $2, 
    updated_at = CURRENT_TIMESTAMP
WHERE external_id = ANY($3::text[]) 
  AND deleted_at IS NULL;

-- name: BulkUpdateProductSyncStatus :exec
UPDATE products 
SET payment_sync_status = $2, 
    payment_synced_at = CURRENT_TIMESTAMP, 
    payment_sync_version = payment_sync_version + 1, 
    payment_provider = $3, 
    updated_at = CURRENT_TIMESTAMP
WHERE workspace_id = $1 
  AND external_id = ANY($4::text[]) 
  AND deleted_at IS NULL;

-- name: BulkUpdatePriceSyncStatus :exec
UPDATE prices 
SET payment_sync_status = $2, 
    payment_synced_at = CURRENT_TIMESTAMP, 
    payment_sync_version = payment_sync_version + 1, 
    payment_provider = $3, 
    updated_at = CURRENT_TIMESTAMP
FROM products p
WHERE prices.product_id = p.id
  AND p.workspace_id = $1 
  AND prices.external_id = ANY($4::text[])
  AND prices.deleted_at IS NULL;

-- name: BulkUpdateSubscriptionSyncStatus :exec
UPDATE subscriptions 
SET payment_sync_status = $2, 
    payment_synced_at = CURRENT_TIMESTAMP, 
    payment_sync_version = payment_sync_version + 1, 
    payment_provider = $3, 
    updated_at = CURRENT_TIMESTAMP
WHERE workspace_id = $1 
  AND external_id = ANY($4::text[]) 
  AND deleted_at IS NULL;

-- Cross-Entity Lookup Queries for External IDs

-- name: GetEntityByExternalIDWithWorkspace :one
SELECT 
    CASE 
        WHEN p.id IS NOT NULL THEN 'product'
        WHEN pr.id IS NOT NULL THEN 'price'
        WHEN s.id IS NOT NULL THEN 'subscription'
        ELSE 'unknown'
    END as entity_type,
    COALESCE(p.id, pr.id, s.id) as entity_id,
    COALESCE(p.external_id, pr.external_id, s.external_id) as external_id,
    COALESCE(p.payment_provider, pr.payment_provider, s.payment_provider) as payment_provider
FROM (SELECT $1 as workspace_id, $2 as external_id, $3 as payment_provider) params
LEFT JOIN products p ON p.workspace_id = params.workspace_id 
    AND p.external_id = params.external_id 
    AND p.payment_provider = params.payment_provider 
    AND p.deleted_at IS NULL
LEFT JOIN prices pr ON pr.external_id = params.external_id 
    AND pr.payment_provider = params.payment_provider 
    AND pr.deleted_at IS NULL
    AND EXISTS (SELECT 1 FROM products prod WHERE prod.id = pr.product_id AND prod.workspace_id = params.workspace_id AND prod.deleted_at IS NULL)
LEFT JOIN subscriptions s ON s.workspace_id = params.workspace_id 
    AND s.external_id = params.external_id 
    AND s.payment_provider = params.payment_provider 
    AND s.deleted_at IS NULL
WHERE COALESCE(p.id, pr.id, s.id) IS NOT NULL
LIMIT 1;

-- name: GetCustomerByExternalIDAndProvider :one
SELECT 
    'customer' as entity_type,
    c.id as entity_id,
    c.external_id,
    c.payment_provider
FROM customers c
WHERE c.external_id = $1 
    AND c.payment_provider = $2 
    AND c.deleted_at IS NULL
LIMIT 1;

-- Additional Workspace-specific Queries

-- name: GetWorkspaceSyncSummary :one
SELECT 
    w.id as workspace_id,
    w.name as workspace_name,
    COUNT(DISTINCT pss.id) as total_sync_sessions,
    COUNT(DISTINCT CASE WHEN pss.status = 'completed' THEN pss.id END) as completed_sessions,
    COUNT(DISTINCT CASE WHEN pss.status = 'failed' THEN pss.id END) as failed_sessions,
    COUNT(DISTINCT CASE WHEN pss.status IN ('pending', 'running') THEN pss.id END) as active_sessions,
    COUNT(DISTINCT CASE WHEN c.payment_sync_status = 'synced' THEN c.id END) as synced_customers,
    COUNT(DISTINCT CASE WHEN p.payment_sync_status = 'synced' THEN p.id END) as synced_products,
    COUNT(DISTINCT CASE WHEN pr.payment_sync_status = 'synced' THEN pr.id END) as synced_prices,
    COUNT(DISTINCT CASE WHEN s.payment_sync_status = 'synced' THEN s.id END) as synced_subscriptions,
    MAX(pss.completed_at) as last_successful_sync
FROM workspaces w
LEFT JOIN payment_sync_sessions pss ON w.id = pss.workspace_id AND pss.deleted_at IS NULL
LEFT JOIN workspace_customers wc ON w.id = wc.workspace_id AND wc.deleted_at IS NULL
LEFT JOIN customers c ON wc.customer_id = c.id AND c.deleted_at IS NULL
LEFT JOIN products p ON w.id = p.workspace_id AND p.deleted_at IS NULL
LEFT JOIN prices pr ON p.id = pr.product_id AND pr.deleted_at IS NULL
LEFT JOIN subscriptions s ON w.id = s.workspace_id AND s.deleted_at IS NULL
WHERE w.id = $1 AND w.deleted_at IS NULL
GROUP BY w.id, w.name;

-- name: GetProviderSyncStatusByWorkspace :many
SELECT 
    provider_name,
    COUNT(*) as total_sessions,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_sessions,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_sessions,
    COUNT(CASE WHEN status IN ('pending', 'running') THEN 1 END) as active_sessions,
    MAX(completed_at) as last_successful_sync,
    MIN(created_at) as first_sync_session
FROM payment_sync_sessions
WHERE workspace_id = $1 AND deleted_at IS NULL
GROUP BY provider_name
ORDER BY last_successful_sync DESC; 