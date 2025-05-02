-- name: GetNetwork :one
SELECT * FROM networks
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetNetworkByChainID :one
SELECT * FROM networks
WHERE chain_id = $1 AND deleted_at IS NULL;

-- name: ListNetworks :many
SELECT * FROM networks
WHERE deleted_at IS NULL
    AND CASE WHEN sqlc.narg('is_testnet')::boolean IS NOT NULL THEN is_testnet = sqlc.narg('is_testnet')::boolean ELSE TRUE END
    AND CASE WHEN sqlc.narg('is_active')::boolean IS NOT NULL THEN active = sqlc.narg('is_active')::boolean ELSE TRUE END
ORDER BY chain_id ASC;

-- name: CreateNetwork :one
INSERT INTO networks (
    name,
    type,
    network_type,
    circle_network_type,
    block_explorer_url,
    chain_id,
    is_testnet,
    active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: UpdateNetwork :one
UPDATE networks
SET
    name = COALESCE($2, name),
    type = COALESCE($3, type),
    network_type = COALESCE($4, network_type),
    circle_network_type = COALESCE($5, circle_network_type),
    block_explorer_url = COALESCE($6, block_explorer_url),
    chain_id = COALESCE($7, chain_id),
    is_testnet = COALESCE($8, is_testnet),
    active = COALESCE($9, active),
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

-- name: GetNetworkByCircleNetworkType :one
SELECT * FROM networks
WHERE circle_network_type = $1 AND deleted_at IS NULL
LIMIT 1; 