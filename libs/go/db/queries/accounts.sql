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
    owner_id,
    business_name,
    business_type,
    website_url,
    support_email,
    support_phone,
    finished_onboarding,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetAccountByID :one
SELECT * FROM accounts
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateAccount :one
UPDATE accounts
SET
    name = COALESCE($2, name),
    account_type = COALESCE($3, account_type),
    owner_id = COALESCE($4, owner_id),
    business_name = COALESCE($5, business_name),
    business_type = COALESCE($6, business_type),
    website_url = COALESCE($7, website_url),
    support_email = COALESCE($8, support_email),
    support_phone = COALESCE($9, support_phone),
    finished_onboarding = COALESCE($10, finished_onboarding),
    metadata = COALESCE($11, metadata),
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
SELECT a.*
FROM accounts a
JOIN users u ON a.id = u.account_id
WHERE u.id = $1 
AND a.deleted_at IS NULL 
AND u.deleted_at IS NULL
ORDER BY a.created_at DESC;

-- name: GetAccountUsers :many
SELECT 
    u.*,
    u.role,
    u.is_account_owner,
    u.created_at as joined_at
FROM users u
WHERE u.account_id = $1 
AND u.deleted_at IS NULL
ORDER BY u.is_account_owner DESC, u.created_at DESC;

-- name: SearchAccounts :many
SELECT DISTINCT a.* 
FROM accounts a
LEFT JOIN users u ON a.id = u.account_id
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