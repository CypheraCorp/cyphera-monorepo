-- name: GetAPIKey :one
SELECT * FROM api_keys
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetAPIKeyByHash :one
SELECT * FROM api_keys
WHERE key_hash = $1 AND deleted_at IS NULL AND is_active = true LIMIT 1;

-- name: ListAPIKeys :many
SELECT * FROM api_keys
WHERE account_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetAllAPIKeys :many
SELECT * FROM api_keys
ORDER BY created_at DESC;

-- name: CreateAPIKey :one
INSERT INTO api_keys (
    account_id,
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

-- name: UpdateAPIKey :one
UPDATE api_keys
SET
    name = COALESCE($2, name),
    level = COALESCE($3, level),
    expires_at = COALESCE($4, expires_at),
    is_active = COALESCE($5, is_active),
    metadata = COALESCE($6, metadata),
    last_used_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateAPIKeyFull :one
UPDATE api_keys
SET
    name = $2,
    level = $3,
    expires_at = $4,
    is_active = $5,
    metadata = $6,
    last_used_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteAPIKey :exec
UPDATE api_keys
SET 
    deleted_at = CURRENT_TIMESTAMP,
    is_active = false
WHERE id = $1 AND deleted_at IS NULL;

-- name: HardDeleteAPIKey :exec
DELETE FROM api_keys
WHERE id = $1;

-- name: UpdateAPIKeyLastUsed :exec
UPDATE api_keys
SET last_used_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: GetActiveAPIKeysCount :one
SELECT COUNT(*) 
FROM api_keys
WHERE account_id = $1 
AND deleted_at IS NULL 
AND is_active = true
AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP);

-- name: GetExpiredAPIKeys :many
SELECT * FROM api_keys
WHERE deleted_at IS NULL 
AND is_active = true
AND expires_at IS NOT NULL 
AND expires_at <= CURRENT_TIMESTAMP
ORDER BY expires_at ASC;