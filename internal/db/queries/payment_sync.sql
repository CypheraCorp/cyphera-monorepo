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

-- name: GetSyncEvent :one
SELECT * FROM payment_sync_events 
WHERE id = $1;

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
WHERE workspace_id = $1 AND payment_provider = $2 AND deleted_at IS NULL;

-- name: GetCustomersByPaymentSyncStatus :many
SELECT * FROM customers 
WHERE workspace_id = $1 AND payment_sync_status = $2 AND deleted_at IS NULL;

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
WHERE c.workspace_id = $1 AND c.payment_provider = $2 AND c.payment_sync_status = $3 AND c.deleted_at IS NULL

UNION ALL

SELECT 
    'product' as entity_type,
    p.id as entity_id,
    NULL as external_id,
    p.payment_sync_status,
    p.payment_synced_at,
    p.payment_provider
FROM products p
WHERE p.workspace_id = $1 AND p.payment_provider = $2 AND p.payment_sync_status = $3 AND p.deleted_at IS NULL

UNION ALL

SELECT 
    'price' as entity_type,
    pr.id as entity_id,
    NULL as external_id,
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
    NULL as external_id,
    s.payment_sync_status,
    s.payment_synced_at,
    s.payment_provider
FROM subscriptions s
WHERE s.workspace_id = $1 AND s.payment_provider = $2 AND s.payment_sync_status = $3 AND s.deleted_at IS NULL

ORDER BY payment_synced_at DESC; 