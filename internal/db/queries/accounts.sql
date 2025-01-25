-- name: GetAccount :one
SELECT * FROM accounts
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetAccountByUserID :one
SELECT * FROM accounts
WHERE user_id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: ListAccounts :many
SELECT * FROM accounts
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetAllAccounts :many
SELECT * FROM accounts
ORDER BY created_at DESC;

-- name: CreateAccount :one
INSERT INTO accounts (
    user_id,
    name,
    description,
    business_name,
    business_type,
    website_url,
    support_email,
    support_phone,
    metadata,
    livemode
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: UpdateAccount :one
UPDATE accounts
SET
    name = COALESCE($2, name),
    description = COALESCE($3, description),
    business_name = COALESCE($4, business_name),
    business_type = COALESCE($5, business_type),
    website_url = COALESCE($6, website_url),
    support_email = COALESCE($7, support_email),
    support_phone = COALESCE($8, support_phone),
    metadata = COALESCE($9, metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateAccountFull :one
UPDATE accounts
SET
    name = $2,
    description = $3,
    business_name = $4,
    business_type = $5,
    website_url = $6,
    support_email = $7,
    support_phone = $8,
    metadata = $9,
    livemode = $10,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteAccount :exec
UPDATE accounts
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL;

-- name: HardDeleteAccount :exec
DELETE FROM accounts
WHERE id = $1;

-- name: ListAccountCustomers :many
SELECT c.* FROM customers c
INNER JOIN accounts a ON c.account_id = a.id
WHERE a.id = $1 AND c.deleted_at IS NULL
ORDER BY c.created_at DESC;