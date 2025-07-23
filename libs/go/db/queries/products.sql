-- name: GetProduct :one
SELECT * FROM products
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL LIMIT 1;

-- name: GetProductWithoutWorkspaceId :one
SELECT * FROM products
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: ListProducts :many
SELECT * FROM products
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListActiveProducts :many
SELECT * FROM products
WHERE workspace_id = $1 AND active = true AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListProductsWithPagination :many
SELECT * FROM products
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountProducts :one
SELECT COUNT(*) FROM products
WHERE workspace_id = $1 AND deleted_at IS NULL;

-- name: CreateProduct :one
INSERT INTO products (
    workspace_id,
    wallet_id,
    name,
    description,
    image_url,
    url,
    active,
    metadata,
    payment_sync_status,
    payment_provider
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8,
    COALESCE($9, 'pending'),
    $10
)
RETURNING *;

-- name: CreateProductWithSync :one
INSERT INTO products (
    workspace_id,
    wallet_id,
    external_id,
    name,
    description,
    image_url,
    url,
    active,
    metadata,
    payment_sync_status,
    payment_synced_at,
    payment_sync_version,
    payment_provider
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
RETURNING *;

-- name: UpdateProduct :one
UPDATE products
SET
    name = COALESCE($3, name),
    wallet_id = COALESCE($4, wallet_id),
    description = COALESCE($5, description),
    image_url = COALESCE($6, image_url),
    url = COALESCE($7, url),
    active = COALESCE($8, active),
    metadata = COALESCE($9, metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateProductWithSync :one
UPDATE products
SET
    name = COALESCE($3, name),
    wallet_id = COALESCE($4, wallet_id),
    description = COALESCE($5, description),
    image_url = COALESCE($6, image_url),
    url = COALESCE($7, url),
    active = COALESCE($8, active),
    metadata = COALESCE($9, metadata),
    payment_sync_status = COALESCE($10, payment_sync_status),
    payment_synced_at = COALESCE($11, payment_synced_at),
    payment_sync_version = COALESCE($12, payment_sync_version),
    payment_provider = COALESCE($13, payment_provider),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteProduct :exec
UPDATE products
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL;

-- name: DeactivateProduct :one
UPDATE products
SET 
    active = false,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: ActivateProduct :one
UPDATE products
SET 
    active = true,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *; 

-- name: GetActiveProductsByWalletID :many
SELECT * FROM products
WHERE wallet_id = $1 AND deleted_at IS NULL;

-- Payment Sync Related Product Queries

-- name: GetProductsNeedingSync :many
SELECT * FROM products 
WHERE workspace_id = $1 AND payment_sync_status = 'pending' AND deleted_at IS NULL
ORDER BY created_at ASC;

-- name: GetProductsSyncedByProvider :many
SELECT * FROM products 
WHERE workspace_id = $1 AND payment_provider = $2 AND payment_sync_status != 'pending' AND deleted_at IS NULL
ORDER BY payment_synced_at DESC;

-- name: UpdateProductPaymentSyncStatus :one
UPDATE products 
SET payment_sync_status = $3, 
    payment_synced_at = CASE WHEN $3 != 'pending' THEN CURRENT_TIMESTAMP ELSE payment_synced_at END,
    payment_sync_version = CASE WHEN $3 != 'pending' THEN payment_sync_version + 1 ELSE payment_sync_version END,
    payment_provider = COALESCE($4, payment_provider),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: GetProductsWithSyncConflicts :many
SELECT * FROM products 
WHERE workspace_id = $1 AND payment_sync_status = 'conflict' AND deleted_at IS NULL
ORDER BY payment_synced_at DESC;

-- name: GetProductByExternalID :one
SELECT p.* FROM products p
WHERE p.workspace_id = $1 
  AND p.external_id = $2
  AND p.payment_provider = $3
  AND p.deleted_at IS NULL;