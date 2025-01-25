-- name: GetCustomer :one
SELECT * FROM customers
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: ListCustomers :many
SELECT * FROM customers
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: CreateCustomer :one
INSERT INTO customers (
    email,
    name,
    description,
    metadata,
    currency,
    tax_exempt,
    tax_ids,
    livemode
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: UpdateCustomer :one
UPDATE customers
SET
    email = COALESCE($2, email),
    name = COALESCE($3, name),
    description = COALESCE($4, description),
    metadata = COALESCE($5, metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteCustomer :exec
UPDATE customers
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL; 