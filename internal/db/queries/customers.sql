-- name: GetCustomer :one
SELECT * FROM customers
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL LIMIT 1;

-- name: GetCustomerByExternalID :one
SELECT * FROM customers
WHERE workspace_id = $1 AND external_id = $2 AND deleted_at IS NULL LIMIT 1;

-- name: ListCustomers :many
SELECT * FROM customers
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetAllCustomers :many
SELECT * FROM customers
ORDER BY created_at DESC;

-- name: CreateCustomer :one
INSERT INTO customers (
    workspace_id,
    external_id,
    email,
    name,
    phone,
    description,
    balance_in_pennies,
    currency,
    default_source_id,
    invoice_prefix,
    next_invoice_sequence,
    tax_exempt,
    tax_ids,
    metadata,
    livemode
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
)
RETURNING *;

-- name: UpdateCustomer :one
UPDATE customers
SET
    email = COALESCE($3, email),
    name = COALESCE($4, name),
    phone = COALESCE($5, phone),
    description = COALESCE($6, description),
    balance_in_pennies = COALESCE($7, balance_in_pennies),
    currency = COALESCE($8, currency),
    default_source_id = COALESCE($9, default_source_id),
    invoice_prefix = COALESCE($10, invoice_prefix),
    next_invoice_sequence = COALESCE($11, next_invoice_sequence),
    tax_exempt = COALESCE($12, tax_exempt),
    tax_ids = COALESCE($13, tax_ids),
    metadata = COALESCE($14, metadata),
    livemode = COALESCE($15, livemode),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteCustomer :exec
UPDATE customers
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL;

-- name: GetCustomerByEmail :one
SELECT * FROM customers
WHERE workspace_id = $1 AND email = $2 AND deleted_at IS NULL LIMIT 1;

-- name: GetCustomersWithWorkspaceInfo :many
SELECT 
    c.*,
    w.name as workspace_name,
    w.business_name as workspace_business_name,
    w.support_email as workspace_support_email
FROM customers c
INNER JOIN workspaces w ON c.workspace_id = w.id
WHERE c.workspace_id = $1 AND c.deleted_at IS NULL
ORDER BY c.created_at DESC;

-- name: UpdateCustomerBalance :one
UPDATE customers
SET 
    balance_in_pennies = balance_in_pennies + $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GetCustomersByBalance :many
SELECT * FROM customers
WHERE workspace_id = $1 
AND deleted_at IS NULL 
AND balance_in_pennies > $2
ORDER BY balance_in_pennies DESC;

-- name: CountCustomers :one
SELECT COUNT(*) FROM customers
WHERE workspace_id = $1 AND deleted_at IS NULL;

-- name: ListCustomersWithPagination :many
SELECT * FROM customers
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
