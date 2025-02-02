-- name: GetNetwork :one
SELECT * FROM networks
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetNetworkByChainID :one
SELECT * FROM networks
WHERE chain_id = $1 AND deleted_at IS NULL;

-- name: ListNetworks :many
SELECT * FROM networks
WHERE deleted_at IS NULL
ORDER BY chain_id ASC;

-- name: ListActiveNetworks :many
SELECT * FROM networks
WHERE active = true AND deleted_at IS NULL
ORDER BY chain_id ASC;

-- name: CreateNetwork :one
INSERT INTO networks (
    name,
    type,
    chain_id,
    active
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: UpdateNetwork :one
UPDATE networks
SET
    name = COALESCE($2, name),
    type = COALESCE($3, type),
    chain_id = COALESCE($4, chain_id),
    active = COALESCE($5, active),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteNetwork :exec
UPDATE networks
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL;

-- name: DeactivateNetwork :one
UPDATE networks
SET 
    active = false,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: ActivateNetwork :one
UPDATE networks
SET 
    active = true,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *; 