-- name: GetCustomer :one
SELECT * FROM customers
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetCustomerByEmail :one
SELECT * FROM customers
WHERE account_id = $1 AND email = $2 AND deleted_at IS NULL LIMIT 1;

-- name: GetCustomersByScope :many
SELECT 
    c.*,
    a.name as account_name,
    a.business_name as account_business_name
FROM customers c
LEFT JOIN accounts a ON c.account_id = a.id
WHERE 
    CASE 
        WHEN $1 = 'admin'::user_role THEN true  -- Admins can see all customers
        WHEN $1 = 'account'::user_role THEN c.account_id = $2  -- Accounts can only see their customers
        ELSE false
    END
    AND c.deleted_at IS NULL
ORDER BY c.created_at DESC;

-- name: GetCustomersByAccountID :many
SELECT 
    c.*,
    a.name as account_name,
    a.business_name as account_business_name,
    a.support_email as account_support_email
FROM customers c
INNER JOIN accounts a ON c.account_id = a.id
WHERE c.deleted_at IS NULL
ORDER BY c.created_at DESC;

-- name: CreateCustomer :one
INSERT INTO customers (
    account_id,
    email,
    name,
    description,
    metadata,
    balance,
    currency,
    default_source_id,
    invoice_prefix,
    next_invoice_sequence,
    tax_exempt,
    tax_ids,
    livemode
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
RETURNING *;

-- name: UpdateCustomer :one
UPDATE customers
SET
    email = COALESCE($2, email),
    name = COALESCE($3, name),
    description = COALESCE($4, description),
    metadata = COALESCE($5, metadata),
    balance = COALESCE($6, balance),
    currency = COALESCE($7, currency),
    default_source_id = COALESCE($8, default_source_id),
    tax_exempt = COALESCE($9, tax_exempt),
    tax_ids = COALESCE($10, tax_ids),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateCustomerFull :one
UPDATE customers
SET
    email = $2,
    name = $3,
    description = $4,
    metadata = $5,
    balance = $6,
    currency = $7,
    default_source_id = $8,
    invoice_prefix = $9,
    next_invoice_sequence = $10,
    tax_exempt = $11,
    tax_ids = $12,
    livemode = $13,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteCustomer :exec
UPDATE customers
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL;

-- name: HardDeleteCustomer :exec
DELETE FROM customers
WHERE id = $1;

-- name: UpdateCustomerBalance :one
UPDATE customers
SET 
    balance = balance + $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GetCustomersByBalance :many
SELECT * FROM customers
WHERE account_id = $1 
AND deleted_at IS NULL 
AND balance > $2
ORDER BY balance DESC;
