-- name: CreateSubscriptionLineItem :one
INSERT INTO subscription_line_items (
    subscription_id,
    product_id,
    line_item_type,
    quantity,
    unit_amount_in_pennies,
    currency,
    price_type,
    interval_type,
    total_amount_in_pennies,
    is_active,
    metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetSubscriptionLineItem :one
SELECT * FROM subscription_line_items
WHERE id = $1;

-- name: ListSubscriptionLineItems :many
SELECT 
    sli.*,
    p.name as product_name,
    p.description as product_description,
    p.image_url as product_image_url,
    p.product_type as product_type
FROM subscription_line_items sli
JOIN products p ON sli.product_id = p.id
WHERE sli.subscription_id = $1
ORDER BY 
    CASE WHEN sli.line_item_type = 'base' THEN 0 ELSE 1 END,
    p.name;

-- name: ListActiveSubscriptionLineItems :many
SELECT 
    sli.*,
    p.name as product_name,
    p.description as product_description,
    p.image_url as product_image_url,
    p.product_type as product_type
FROM subscription_line_items sli
JOIN products p ON sli.product_id = p.id
WHERE sli.subscription_id = $1
  AND sli.is_active = true
ORDER BY 
    CASE WHEN sli.line_item_type = 'base' THEN 0 ELSE 1 END,
    p.name;

-- name: GetSubscriptionBaseLineItem :one
SELECT 
    sli.*,
    p.name as product_name,
    p.description as product_description,
    p.image_url as product_image_url
FROM subscription_line_items sli
JOIN products p ON sli.product_id = p.id
WHERE sli.subscription_id = $1
  AND sli.line_item_type = 'base'
  AND sli.is_active = true
LIMIT 1;

-- name: ListSubscriptionAddonLineItems :many
SELECT 
    sli.*,
    p.name as product_name,
    p.description as product_description,
    p.image_url as product_image_url
FROM subscription_line_items sli
JOIN products p ON sli.product_id = p.id
WHERE sli.subscription_id = $1
  AND sli.line_item_type = 'addon'
  AND sli.is_active = true
ORDER BY p.name;

-- name: UpdateSubscriptionLineItemQuantity :one
UPDATE subscription_line_items
SET
    quantity = $2,
    total_amount_in_pennies = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeactivateSubscriptionLineItem :one
UPDATE subscription_line_items
SET
    is_active = false,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: ActivateSubscriptionLineItem :one
UPDATE subscription_line_items
SET
    is_active = true,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteSubscriptionLineItem :exec
DELETE FROM subscription_line_items
WHERE id = $1;

-- name: DeleteAllSubscriptionLineItems :exec
DELETE FROM subscription_line_items
WHERE subscription_id = $1;

-- name: CalculateSubscriptionTotal :one
SELECT 
    COALESCE(SUM(total_amount_in_pennies), 0) as total_amount
FROM subscription_line_items
WHERE subscription_id = $1
  AND is_active = true;

-- name: CountSubscriptionLineItems :one
SELECT COUNT(*) FROM subscription_line_items
WHERE subscription_id = $1
  AND is_active = true;

-- name: GetSubscriptionLineItemByProduct :one
SELECT * FROM subscription_line_items
WHERE subscription_id = $1
  AND product_id = $2
  AND is_active = true
LIMIT 1;

-- name: BatchCreateSubscriptionLineItems :copyfrom
INSERT INTO subscription_line_items (
    subscription_id,
    product_id,
    line_item_type,
    quantity,
    unit_amount_in_pennies,
    currency,
    price_type,
    interval_type,
    total_amount_in_pennies,
    is_active,
    metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);