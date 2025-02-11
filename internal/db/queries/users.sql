-- name: CreateUser :one
INSERT INTO users (
    supabase_id,
    email,
    account_id,
    role,
    is_account_owner,
    first_name,
    last_name,
    display_name,
    picture_url,
    phone,
    timezone,
    locale,
    email_verified,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
) RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserBySupabaseID :one
SELECT * FROM users
WHERE supabase_id = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: UpdateUser :one
UPDATE users
SET
    email = COALESCE($1, email),
    first_name = COALESCE($2, first_name),
    last_name = COALESCE($3, last_name),
    display_name = COALESCE($4, display_name),
    picture_url = COALESCE($5, picture_url),
    phone = COALESCE($6, phone),
    timezone = COALESCE($7, timezone),
    locale = COALESCE($8, locale),
    email_verified = COALESCE($9, email_verified),
    two_factor_enabled = COALESCE($10, two_factor_enabled),
    status = COALESCE($11, status),
    metadata = COALESCE($12, metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateUserRole :one
UPDATE users
SET
    role = $2,
    is_account_owner = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteUser :exec
UPDATE users
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: ListUsersByAccount :many
SELECT * FROM users
WHERE account_id = $1 AND deleted_at IS NULL
ORDER BY is_account_owner DESC, created_at DESC;

-- name: GetAccountOwner :one
SELECT * FROM users
WHERE account_id = $1 AND is_account_owner = true AND deleted_at IS NULL;

-- name: GetUserAccount :one
SELECT 
    u.*,
    a.name as account_name
FROM users u
JOIN accounts a ON u.account_id = a.id
WHERE u.id = $1 
AND u.deleted_at IS NULL 
AND a.deleted_at IS NULL;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at; 