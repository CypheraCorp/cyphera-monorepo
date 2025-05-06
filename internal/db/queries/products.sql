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
    product_type,
    interval_type,
    term_length,
    price_in_pennies,
    currency,
    image_url,
    url,
    merchant_paid_gas,
    active,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
RETURNING *;

-- name: UpdateProduct :one
UPDATE products
SET
    name = COALESCE($3, name),
    wallet_id = COALESCE($4, wallet_id),
    description = COALESCE($5, description),
    product_type = COALESCE($6, product_type),
    interval_type = COALESCE($7, interval_type),
    term_length = COALESCE($8, term_length),
    price_in_pennies = COALESCE($9, price_in_pennies),
    currency = COALESCE($10, currency),
    image_url = COALESCE($11, image_url),
    url = COALESCE($12, url),
    merchant_paid_gas = COALESCE($13, merchant_paid_gas),
    active = COALESCE($14, active),
    metadata = COALESCE($15, metadata),
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