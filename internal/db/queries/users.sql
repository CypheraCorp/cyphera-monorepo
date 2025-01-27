-- name: CreateUser :one
INSERT INTO users (
    auth0_id,
    email,
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
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByAuth0ID :one
SELECT * FROM users
WHERE auth0_id = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: UpdateUser :one
UPDATE users
SET
    email = COALESCE($2, email),
    first_name = COALESCE($3, first_name),
    last_name = COALESCE($4, last_name),
    display_name = COALESCE($5, display_name),
    picture_url = COALESCE($6, picture_url),
    phone = COALESCE($7, phone),
    timezone = COALESCE($8, timezone),
    locale = COALESCE($9, locale),
    email_verified = COALESCE($10, email_verified),
    two_factor_enabled = COALESCE($11, two_factor_enabled),
    status = COALESCE($12, status),
    metadata = COALESCE($13, metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteUser :exec
UPDATE users
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: ListUsersByAccount :many
SELECT u.* FROM users u
JOIN user_accounts ua ON u.id = ua.user_id
WHERE ua.account_id = $1 
AND u.deleted_at IS NULL 
AND ua.deleted_at IS NULL
ORDER BY u.created_at DESC;

-- User Account Relationship Queries

-- name: AddUserToAccount :one
INSERT INTO user_accounts (
    user_id,
    account_id,
    role,
    is_owner
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetUserAccountRole :one
SELECT * FROM user_accounts
WHERE user_id = $1 
AND account_id = $2 
AND deleted_at IS NULL;

-- name: UpdateUserAccountRole :one
UPDATE user_accounts
SET 
    role = $3,
    is_owner = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1 
AND account_id = $2 
AND deleted_at IS NULL
RETURNING *;

-- name: RemoveUserFromAccount :exec
UPDATE user_accounts
SET deleted_at = CURRENT_TIMESTAMP
WHERE user_id = $1 
AND account_id = $2 
AND is_owner = false; -- Prevent removal of account owners

-- name: GetAccountOwner :one
SELECT u.* FROM users u
JOIN user_accounts ua ON u.id = ua.user_id
WHERE ua.account_id = $1 
AND ua.is_owner = true 
AND u.deleted_at IS NULL 
AND ua.deleted_at IS NULL;

-- name: ListUserAccounts :many
SELECT 
    u.*,
    a.name as account_name,
    ua.role,
    ua.is_owner
FROM users u
JOIN user_accounts ua ON u.id = ua.user_id
JOIN accounts a ON ua.account_id = a.id
WHERE u.id = $1 
AND u.deleted_at IS NULL 
AND ua.deleted_at IS NULL
AND a.deleted_at IS NULL
ORDER BY ua.created_at DESC; 