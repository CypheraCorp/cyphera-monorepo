-- name: GetActalinkProduct :one
-- Get a single actalink product by ID
SELECT * FROM actalink_products
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetActalinkProductsByProductID :many
-- Get all actalink products for a given product_id
SELECT * FROM actalink_products
WHERE product_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetActalinkProductByProductTokenID :one
-- Get a single actalink product by product_token_id (unique due to table constraint)
SELECT * FROM actalink_products
WHERE product_token_id = $1 AND deleted_at IS NULL;

-- name: ListActalinkProducts :many
-- List all actalink products with optional pagination
SELECT * FROM actalink_products
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateActalinkProduct :one
-- Create a new actalink product
INSERT INTO actalink_products (product_id, product_token_id, actalink_payment_link_id, actalink_subscription_id)
VALUES ($1, $2, $3, $4)
RETURNING *;
