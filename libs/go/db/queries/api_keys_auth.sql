-- name: GetAllActiveAPIKeys :many
-- Used for authentication - retrieves all active API keys for bcrypt comparison
SELECT * FROM api_keys
WHERE deleted_at IS NULL 
  AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
ORDER BY created_at DESC;

-- name: UpdateAPIKeyLastUsed :exec
-- Updates the last_used_at timestamp after successful authentication
UPDATE api_keys
SET last_used_at = CURRENT_TIMESTAMP
WHERE id = $1;