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
    metadata,
    payment_sync_status,
    payment_provider
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, 
    COALESCE($8, 'pending'), 
    $9
)
RETURNING *;

-- name: CreateCustomerWithSync :one
INSERT INTO customers (
    workspace_id,
    external_id,
    email,
    name,
    phone,
    description,
    metadata,
    payment_sync_status,
    payment_synced_at,
    payment_sync_version,
    payment_provider
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
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

-- name: UpdateCustomerWithSync :one
UPDATE customers
SET
    email = COALESCE($3, email),
    name = COALESCE($4, name),
    phone = COALESCE($5, phone),
    description = COALESCE($6, description),
    metadata = COALESCE($7, metadata),
    payment_sync_status = COALESCE($8, payment_sync_status),
    payment_synced_at = COALESCE($9, payment_synced_at),
    payment_sync_version = COALESCE($10, payment_sync_version),
    payment_provider = COALESCE($11, payment_provider),
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

-- Payment Sync Related Customer Queries

-- name: GetCustomersByExternalIDs :many
SELECT * FROM customers 
WHERE workspace_id = $1 AND external_id = ANY($2::text[]) AND deleted_at IS NULL;

-- name: GetCustomersNeedingSync :many
SELECT * FROM customers 
WHERE workspace_id = $1 AND payment_sync_status = 'pending' AND deleted_at IS NULL
ORDER BY created_at ASC;

-- name: GetCustomersSyncedByProvider :many
SELECT * FROM customers 
WHERE workspace_id = $1 AND payment_provider = $2 AND payment_sync_status != 'pending' AND deleted_at IS NULL
ORDER BY payment_synced_at DESC;

-- name: UpdateCustomerPaymentSyncStatus :one
UPDATE customers 
SET payment_sync_status = $3, 
    payment_synced_at = CASE WHEN $3 != 'pending' THEN CURRENT_TIMESTAMP ELSE payment_synced_at END,
    payment_sync_version = CASE WHEN $3 != 'pending' THEN payment_sync_version + 1 ELSE payment_sync_version END,
    payment_provider = COALESCE($4, payment_provider),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: GetCustomersWithSyncConflicts :many
SELECT * FROM customers 
WHERE workspace_id = $1 AND payment_sync_status = 'conflict' AND deleted_at IS NULL
ORDER BY payment_synced_at DESC;
