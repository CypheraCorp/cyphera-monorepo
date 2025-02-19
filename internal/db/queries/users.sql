-- name: CreateUser :one
INSERT INTO users (
    supabase_id,
    email,
    account_id,
    role,
    is_account_owner,
    first_name,
    last_name,
    address_line_1,
    address_line_2,
    city,
    state_region,
    postal_code,
    country,
    display_name,
    picture_url,
    phone,
    timezone,
    locale,
    email_verified,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
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
    address_line_1 = COALESCE($4, address_line_1),
    address_line_2 = COALESCE($5, address_line_2),
    city = COALESCE($6, city),
    state_region = COALESCE($7, state_region),
    postal_code = COALESCE($8, postal_code),
    country = COALESCE($9, country),
    display_name = COALESCE($10, display_name),
    picture_url = COALESCE($11, picture_url),
    phone = COALESCE($12, phone),
    timezone = COALESCE($13, timezone),
    locale = COALESCE($14, locale),
    email_verified = COALESCE($15, email_verified),
    two_factor_enabled = COALESCE($16, two_factor_enabled),
    status = COALESCE($17, status),
    metadata = COALESCE($18, metadata),
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