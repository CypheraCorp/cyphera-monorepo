-- name: GetToken :one
SELECT * FROM tokens
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetTokenByAddress :one
SELECT * FROM tokens
WHERE network_id = $1 AND contract_address = $2 AND deleted_at IS NULL;

-- name: ListTokens :many
SELECT * FROM tokens
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListTokensByNetwork :many
SELECT * FROM tokens
WHERE network_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListActiveTokensByNetwork :many
SELECT * FROM tokens
WHERE network_id = $1 AND active = true AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetGasToken :one
SELECT * FROM tokens
WHERE network_id = $1 AND gas_token = true AND deleted_at IS NULL
LIMIT 1;

-- name: CreateToken :one
INSERT INTO tokens (
    network_id,
    gas_token,
    name,
    symbol,
    contract_address,
    decimals,
    active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: UpdateToken :one
UPDATE tokens
SET
    name = COALESCE($2, name),
    symbol = COALESCE($3, symbol),
    contract_address = COALESCE($4, contract_address),
    gas_token = COALESCE($5, gas_token),
    decimals = COALESCE($6, decimals),
    active = COALESCE($7, active),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteToken :exec
UPDATE tokens
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL;

-- name: DeactivateToken :one
UPDATE tokens
SET 
    active = false,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: ActivateToken :one
UPDATE tokens
SET 
    active = true,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *; 