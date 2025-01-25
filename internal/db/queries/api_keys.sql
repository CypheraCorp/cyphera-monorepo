-- name: GetAPIKey :one
SELECT * FROM api_keys
WHERE id = $1 AND is_active = true LIMIT 1;

-- name: GetAPIKeyByHash :one
SELECT * FROM api_keys
WHERE key_hash = $1 AND is_active = true LIMIT 1;

-- name: ListAPIKeys :many
SELECT * FROM api_keys
WHERE customer_id = $1 AND is_active = true
ORDER BY created_at DESC;

-- name: CreateAPIKey :one
INSERT INTO api_keys (
    customer_id,
    name,
    key_hash,
    level,
    expires_at,
    metadata,
    livemode
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: DeactivateAPIKey :exec
UPDATE api_keys
SET is_active = false
WHERE id = $1;

-- name: UpdateAPIKeyLastUsed :exec
UPDATE api_keys
SET last_used_at = CURRENT_TIMESTAMP
WHERE id = $1; 