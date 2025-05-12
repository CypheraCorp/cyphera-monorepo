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
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
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