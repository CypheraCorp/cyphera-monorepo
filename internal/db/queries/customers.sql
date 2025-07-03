-- name: GetCustomer :one
SELECT * FROM customers
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetCustomerByExternalID :one
SELECT * FROM customers
WHERE external_id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: ListCustomers :many
SELECT * FROM customers
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetAllCustomers :many
SELECT * FROM customers
ORDER BY created_at DESC;

-- name: CreateCustomer :one
INSERT INTO customers (
    external_id,
    email,
    name,
    phone,
    description,
    metadata,
    payment_sync_status,
    payment_provider
) VALUES (
    $1, $2, $3, $4, $5, $6, 
    COALESCE($7, 'pending'), 
    $8
)
RETURNING *;

-- name: CreateCustomerWithSync :one
INSERT INTO customers (
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
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: CreateCustomerWithWeb3Auth :one
INSERT INTO customers (
    web3auth_id,
    email,
    name,
    phone,
    description,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: UpdateCustomer :one
UPDATE customers
SET
    email = COALESCE($2, email),
    name = COALESCE($3, name),
    phone = COALESCE($4, phone),
    description = COALESCE($5, description),
    metadata = COALESCE($6, metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateCustomerWithSync :one
UPDATE customers
SET
    email = COALESCE($2, email),
    name = COALESCE($3, name),
    phone = COALESCE($4, phone),
    description = COALESCE($5, description),
    metadata = COALESCE($6, metadata),
    payment_sync_status = COALESCE($7, payment_sync_status),
    payment_synced_at = COALESCE($8, payment_synced_at),
    payment_sync_version = COALESCE($9, payment_sync_version),
    payment_provider = COALESCE($10, payment_provider),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteCustomer :exec
UPDATE customers
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetCustomerByEmail :one
SELECT * FROM customers
WHERE email = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetCustomerByWeb3AuthID :one
SELECT * FROM customers
WHERE web3auth_id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: CountCustomers :one
SELECT COUNT(*) FROM customers
WHERE deleted_at IS NULL;

-- name: ListCustomersWithPagination :many
SELECT * FROM customers
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- Payment Sync Related Customer Queries

-- name: GetCustomersByExternalIDs :many
SELECT * FROM customers 
WHERE external_id = ANY($1::text[]) AND deleted_at IS NULL;

-- name: GetCustomersNeedingSync :many
SELECT * FROM customers 
WHERE payment_sync_status = 'pending' AND deleted_at IS NULL
ORDER BY created_at ASC;

-- name: GetCustomersSyncedByProvider :many
SELECT * FROM customers 
WHERE payment_provider = $1 AND payment_sync_status != 'pending' AND deleted_at IS NULL
ORDER BY payment_synced_at DESC;

-- name: UpdateCustomerPaymentSyncStatus :one
UPDATE customers 
SET payment_sync_status = $2, 
    payment_synced_at = CASE WHEN $2 != 'pending' THEN CURRENT_TIMESTAMP ELSE payment_synced_at END,
    payment_sync_version = CASE WHEN $2 != 'pending' THEN payment_sync_version + 1 ELSE payment_sync_version END,
    payment_provider = COALESCE($3, payment_provider),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GetCustomersWithSyncConflicts :many
SELECT * FROM customers 
WHERE payment_sync_status = 'conflict' AND deleted_at IS NULL
ORDER BY payment_synced_at DESC;
