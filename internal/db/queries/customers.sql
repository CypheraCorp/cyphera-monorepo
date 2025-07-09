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
    finished_onboarding,
    payment_sync_status,
    payment_provider
) VALUES (
    @external_id,
    @email,
    @name,
    @phone,
    @description,
    @metadata,
    COALESCE(@finished_onboarding, false),
    COALESCE(@payment_sync_status, 'pending'),
    @payment_provider
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
    finished_onboarding,
    payment_sync_status,
    payment_synced_at,
    payment_sync_version,
    payment_provider
) VALUES (
    @external_id,
    @email,
    @name,
    @phone,
    @description,
    @metadata,
    COALESCE(@finished_onboarding, false),
    @payment_sync_status,
    @payment_synced_at,
    @payment_sync_version,
    @payment_provider
)
RETURNING *;

-- name: CreateCustomerWithWeb3Auth :one
INSERT INTO customers (
    web3auth_id,
    email,
    name,
    phone,
    description,
    metadata,
    finished_onboarding
) VALUES (
    @web3auth_id,
    @email,
    @name,
    @phone,
    @description,
    @metadata,
    COALESCE(@finished_onboarding, false)
)
RETURNING *;

-- name: UpdateCustomer :one
UPDATE customers
SET
    email = COALESCE(@email, email),
    name = COALESCE(@name, name),
    phone = COALESCE(@phone, phone),
    description = COALESCE(@description, description),
    metadata = COALESCE(@metadata, metadata),
    finished_onboarding = COALESCE(@finished_onboarding, finished_onboarding),
    updated_at = CURRENT_TIMESTAMP
WHERE id = @id AND deleted_at IS NULL
RETURNING *;

-- name: UpdateCustomerWithSync :one
UPDATE customers
SET
    email = COALESCE(@email, email),
    name = COALESCE(@name, name),
    phone = COALESCE(@phone, phone),
    description = COALESCE(@description, description),
    metadata = COALESCE(@metadata, metadata),
    finished_onboarding = COALESCE(@finished_onboarding, finished_onboarding),
    payment_sync_status = COALESCE(@payment_sync_status, payment_sync_status),
    payment_synced_at = COALESCE(@payment_synced_at, payment_synced_at),
    payment_sync_version = COALESCE(@payment_sync_version, payment_sync_version),
    payment_provider = COALESCE(@payment_provider, payment_provider),
    updated_at = CURRENT_TIMESTAMP
WHERE id = @id AND deleted_at IS NULL
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

-- name: UpdateCustomerOnboardingStatus :one
UPDATE customers
SET
    finished_onboarding = @finished_onboarding,
    updated_at = CURRENT_TIMESTAMP
WHERE id = @id AND deleted_at IS NULL
RETURNING *;
