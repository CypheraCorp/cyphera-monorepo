-- name: GetPrice :one
SELECT * FROM prices
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetPriceWithProduct :one
SELECT pr.*, p.name as product_name, p.workspace_id
FROM prices pr
JOIN products p ON pr.product_id = p.id
WHERE pr.id = $1 AND pr.deleted_at IS NULL AND p.deleted_at IS NULL;

-- name: ListPricesByProduct :many
SELECT * FROM prices
WHERE product_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListActivePricesByProduct :many
SELECT * FROM prices
WHERE product_id = $1 AND active = true AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: CreatePrice :one
INSERT INTO prices (
    product_id,
    active,
    type, -- price_type enum
    nickname,
    currency, -- currency enum
    unit_amount_in_pennies,
    interval_type, -- interval_type enum (nullable)
    term_length, -- (nullable)
    metadata,
    payment_sync_status,
    payment_provider
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9,
    COALESCE($10, 'pending'),
    $11
)
RETURNING *;

-- name: CreatePriceWithSync :one
INSERT INTO prices (
    product_id,
    external_id,
    active,
    type, -- price_type enum
    nickname,
    currency, -- currency enum
    unit_amount_in_pennies,
    interval_type, -- interval_type enum (nullable)
    term_length, -- (nullable)
    metadata,
    payment_sync_status,
    payment_synced_at,
    payment_sync_version,
    payment_provider
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
RETURNING *;

-- name: UpdatePrice :one
UPDATE prices
SET
    active = COALESCE($2, active),
    nickname = COALESCE($3, nickname),
    metadata = COALESCE($4, metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdatePriceWithSync :one
UPDATE prices
SET
    active = COALESCE($2, active),
    nickname = COALESCE($3, nickname),
    metadata = COALESCE($4, metadata),
    payment_sync_status = COALESCE($5, payment_sync_status),
    payment_synced_at = COALESCE($6, payment_synced_at),
    payment_sync_version = COALESCE($7, payment_sync_version),
    payment_provider = COALESCE($8, payment_provider),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeletePrice :exec
UPDATE prices
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL;

-- name: DeactivatePrice :one
UPDATE prices
SET 
    active = false,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: ActivatePrice :one
UPDATE prices
SET 
    active = true,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: ListPricesByWorkspace :many
SELECT pr.*
FROM prices pr
JOIN products p ON pr.product_id = p.id
WHERE p.workspace_id = $1 AND pr.deleted_at IS NULL AND p.deleted_at IS NULL
ORDER BY pr.created_at DESC;

-- name: ListActivePricesByWorkspace :many
SELECT pr.*
FROM prices pr
JOIN products p ON pr.product_id = p.id
WHERE p.workspace_id = $1 AND pr.active = true AND pr.deleted_at IS NULL AND p.deleted_at IS NULL
ORDER BY pr.created_at DESC;

-- name: CountPricesByProduct :one
SELECT COUNT(*) FROM prices
WHERE product_id = $1 AND deleted_at IS NULL;

-- name: CountPricesByWorkspace :one
SELECT COUNT(pr.id)
FROM prices pr
JOIN products p ON pr.product_id = p.id
WHERE p.workspace_id = $1 AND pr.deleted_at IS NULL AND p.deleted_at IS NULL;

-- Payment Sync Related Price Queries

-- name: GetPricesNeedingSync :many
SELECT pr.*
FROM prices pr
JOIN products p ON pr.product_id = p.id
WHERE p.workspace_id = $1 AND pr.payment_sync_status = 'pending' AND pr.deleted_at IS NULL AND p.deleted_at IS NULL
ORDER BY pr.created_at ASC;

-- name: GetPricesSyncedByProvider :many
SELECT pr.*
FROM prices pr
JOIN products p ON pr.product_id = p.id
WHERE p.workspace_id = $1 AND pr.payment_provider = $2 AND pr.payment_sync_status != 'pending' AND pr.deleted_at IS NULL AND p.deleted_at IS NULL
ORDER BY pr.payment_synced_at DESC;

-- name: UpdatePricePaymentSyncStatus :one
UPDATE prices 
SET payment_sync_status = $2, 
    payment_synced_at = CASE WHEN $2 != 'pending' THEN CURRENT_TIMESTAMP ELSE payment_synced_at END,
    payment_sync_version = CASE WHEN $2 != 'pending' THEN payment_sync_version + 1 ELSE payment_sync_version END,
    payment_provider = COALESCE($3, payment_provider),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GetPricesWithSyncConflicts :many
SELECT pr.*
FROM prices pr
JOIN products p ON pr.product_id = p.id
WHERE p.workspace_id = $1 AND pr.payment_sync_status = 'conflict' AND pr.deleted_at IS NULL AND p.deleted_at IS NULL
ORDER BY pr.payment_synced_at DESC;

-- name: GetPriceByExternalID :one
SELECT p.* FROM prices p
WHERE p.external_id = $1
  AND p.payment_provider = $2
  AND p.deleted_at IS NULL; 