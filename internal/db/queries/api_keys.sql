-- name: GetAPIKey :one
SELECT * FROM api_keys
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetAPIKeyByKey :one
SELECT * FROM api_keys
WHERE key_hash = $1 AND deleted_at IS NULL LIMIT 1;

-- name: ListAPIKeys :many
SELECT * FROM api_keys
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetAllAPIKeys :many
SELECT * FROM api_keys
ORDER BY created_at DESC;

-- name: CreateAPIKey :one
INSERT INTO api_keys (
    workspace_id,
    name,
    key_hash,
    access_level,
    expires_at,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: UpdateAPIKey :one
UPDATE api_keys
SET
    name = COALESCE($2, name),
    access_level = COALESCE($3, access_level),
    expires_at = COALESCE($4, expires_at),
    metadata = COALESCE($5, metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteAPIKey :exec
UPDATE api_keys
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetActiveAPIKeysCount :one
SELECT COUNT(*) FROM api_keys
WHERE workspace_id = $1 AND deleted_at IS NULL;

-- name: GetExpiredAPIKeys :many
SELECT * FROM api_keys
WHERE expires_at < CURRENT_TIMESTAMP AND deleted_at IS NULL
ORDER BY created_at DESC;