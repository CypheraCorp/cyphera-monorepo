-- name: GetAccount :one
SELECT * FROM accounts
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: ListAccounts :many
SELECT * FROM accounts
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetAllAccounts :many
SELECT * FROM accounts
ORDER BY created_at DESC;

-- name: CreateAccount :one
INSERT INTO accounts (
    name,
    account_type,
    business_name,
    business_type,
    website_url,
    support_email,
    support_phone,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetAccountByID :one
SELECT * FROM accounts
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateAccount :one
UPDATE accounts
SET
    name = COALESCE($2, name),
    account_type = COALESCE($3, account_type),
    business_name = COALESCE($4, business_name),
    business_type = COALESCE($5, business_type),
    website_url = COALESCE($6, website_url),
    support_email = COALESCE($7, support_email),
    support_phone = COALESCE($8, support_phone),
    metadata = COALESCE($9, metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteAccount :exec
UPDATE accounts
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: HardDeleteAccount :exec
DELETE FROM accounts
WHERE id = $1;

-- name: ListAccountsByUser :many
SELECT 
    a.*,
    ua.role as user_role,
    ua.is_owner
FROM accounts a
JOIN user_accounts ua ON a.id = ua.account_id
WHERE ua.user_id = $1 
AND a.deleted_at IS NULL 
AND ua.deleted_at IS NULL
ORDER BY a.created_at DESC;

-- name: GetAccountUsers :many
SELECT 
    u.*,
    ua.role,
    ua.is_owner,
    ua.created_at as joined_at
FROM users u
JOIN user_accounts ua ON u.id = ua.user_id
WHERE ua.account_id = $1 
AND u.deleted_at IS NULL 
AND ua.deleted_at IS NULL
ORDER BY ua.is_owner DESC, ua.created_at DESC;

-- name: SearchAccounts :many
SELECT DISTINCT a.* 
FROM accounts a
LEFT JOIN user_accounts ua ON a.id = ua.account_id
LEFT JOIN users u ON ua.user_id = u.id
WHERE 
    (
        a.name ILIKE $1 OR
        a.business_name ILIKE $1 OR
        u.email ILIKE $1 OR
        u.display_name ILIKE $1
    )
    AND a.deleted_at IS NULL
ORDER BY a.created_at DESC
LIMIT $2
OFFSET $3;

-- name: ListAccountsByType :many
SELECT * FROM accounts
WHERE account_type = $1 AND deleted_at IS NULL
ORDER BY created_at DESC; 