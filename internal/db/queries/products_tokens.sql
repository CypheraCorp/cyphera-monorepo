-- name: GetProductToken :one
SELECT 
    pt.*,
    t.name as token_name,
    t.symbol as token_symbol,
    t.contract_address,
    t.gas_token,
    n.chain_id,
    n.name as network_name,
    n.type as network_type
FROM products_tokens pt
JOIN tokens t ON t.id = pt.token_id
JOIN networks n ON n.id = pt.network_id
WHERE pt.id = $1 AND pt.deleted_at IS NULL;

-- name: GetProductTokenByIds :one
SELECT 
    pt.*,
    t.name as token_name,
    t.symbol as token_symbol,
    t.contract_address,
    t.gas_token
FROM products_tokens pt
JOIN tokens t ON t.id = pt.token_id
WHERE pt.product_id = $1 
    AND pt.network_id = $2 
    AND pt.token_id = $3 
    AND pt.deleted_at IS NULL;

-- name: GetProductTokensByProduct :many
SELECT 
    pt.*,
    t.name as token_name,
    t.symbol as token_symbol,
    t.contract_address,
    t.gas_token,
    n.chain_id,
    n.name as network_name,
    n.type as network_type
FROM products_tokens pt
JOIN tokens t ON t.id = pt.token_id
JOIN networks n ON n.id = pt.network_id
WHERE pt.product_id = $1 
    AND pt.deleted_at IS NULL
ORDER BY n.chain_id ASC, t.name ASC;

-- name: GetActiveProductTokensByProduct :many
SELECT 
    pt.*,
    t.name as token_name,
    t.symbol as token_symbol,
    t.contract_address,
    t.gas_token,
    n.chain_id,
    n.name as network_name,
    n.type as network_type
FROM products_tokens pt
JOIN tokens t ON t.id = pt.token_id
JOIN networks n ON n.id = pt.network_id
WHERE pt.product_id = $1 
    AND pt.active = true 
    AND pt.deleted_at IS NULL
    AND t.active = true
    AND t.deleted_at IS NULL
    AND n.active = true
    AND n.deleted_at IS NULL
ORDER BY n.chain_id ASC, t.name ASC;

-- name: GetProductNetworks :many
SELECT DISTINCT
    n.*,
    (
        SELECT COUNT(*) 
        FROM products_tokens pt2 
        JOIN tokens t2 ON t2.id = pt2.token_id
        WHERE pt2.product_id = $1 
        AND pt2.network_id = n.id 
        AND pt2.active = true 
        AND pt2.deleted_at IS NULL
        AND t2.active = true
        AND t2.deleted_at IS NULL
    ) as active_tokens_count
FROM networks n
JOIN products_tokens pt ON pt.network_id = n.id
WHERE pt.product_id = $1 
    AND pt.deleted_at IS NULL
    AND n.deleted_at IS NULL
ORDER BY n.chain_id ASC;

-- name: GetProductTokensByNetwork :many
SELECT 
    pt.*,
    t.name as token_name,
    t.symbol as token_symbol,
    t.contract_address,
    t.gas_token
FROM products_tokens pt
JOIN tokens t ON t.id = pt.token_id
WHERE pt.product_id = $1 
    AND pt.network_id = $2
    AND pt.deleted_at IS NULL
    AND t.active = true
    AND t.deleted_at IS NULL
ORDER BY t.name ASC;

-- name: GetActiveProductTokensByNetwork :many
SELECT 
    pt.*,
    t.name as token_name,
    t.symbol as token_symbol,
    t.contract_address,
    t.gas_token
FROM products_tokens pt
JOIN tokens t ON t.id = pt.token_id
WHERE pt.product_id = $1 
    AND pt.network_id = $2
    AND pt.active = true
    AND pt.deleted_at IS NULL
    AND t.active = true
    AND t.deleted_at IS NULL
ORDER BY t.name ASC;

-- name: CreateProductToken :one
INSERT INTO products_tokens (
    product_id,
    network_id,
    token_id,
    active
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: UpdateProductToken :one
UPDATE products_tokens
SET
    active = COALESCE($4, active),
    updated_at = CURRENT_TIMESTAMP
WHERE product_id = $1 
    AND network_id = $2 
    AND token_id = $3 
    AND deleted_at IS NULL
RETURNING *;

-- name: DeleteProductToken :exec
UPDATE products_tokens
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteProductTokenByIds :exec
UPDATE products_tokens
SET deleted_at = CURRENT_TIMESTAMP
WHERE product_id = $1 
    AND network_id = $2 
    AND token_id = $3 
    AND deleted_at IS NULL;

-- name: DeactivateProductToken :one
UPDATE products_tokens
SET 
    active = false,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: ActivateProductToken :one
UPDATE products_tokens
SET 
    active = true,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeactivateAllProductTokensForNetwork :exec
UPDATE products_tokens
SET 
    active = false,
    updated_at = CURRENT_TIMESTAMP
WHERE product_id = $1 
    AND network_id = $2 
    AND deleted_at IS NULL;

-- name: DeactivateAllProductTokens :exec
UPDATE products_tokens
SET 
    active = false,
    updated_at = CURRENT_TIMESTAMP
WHERE product_id = $1 
    AND deleted_at IS NULL; 

-- name: DeleteProductTokensByProduct :exec
DELETE FROM products_tokens
WHERE product_id = $1 
    AND deleted_at IS NULL;