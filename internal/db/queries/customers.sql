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
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: UpdateCustomer :one
UPDATE customers
SET
    email = COALESCE($3, email),
    name = COALESCE($4, name),
    phone = COALESCE($5, phone),
    description = COALESCE($6, description),
    metadata = COALESCE($7, metadata),
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

-- name: CountCustomers :one
SELECT COUNT(*) FROM customers
WHERE workspace_id = $1 AND deleted_at IS NULL;

-- name: ListCustomersWithPagination :many
SELECT * FROM customers
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountCustomersByWorkspaceID :one
SELECT COUNT(*) FROM customers
WHERE workspace_id = $1 AND deleted_at IS NULL;
