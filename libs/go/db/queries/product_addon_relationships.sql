-- name: CreateProductAddonRelationship :one
INSERT INTO product_addon_relationships (
    base_product_id,
    addon_product_id,
    is_required,
    max_quantity,
    min_quantity,
    display_order,
    metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetProductAddonRelationship :one
SELECT * FROM product_addon_relationships
WHERE id = $1;

-- name: GetProductAddonRelationshipByProducts :one
SELECT * FROM product_addon_relationships
WHERE base_product_id = $1 AND addon_product_id = $2;

-- name: ListProductAddons :many
SELECT 
    par.*,
    p.id as addon_id,
    p.name as addon_name,
    p.description as addon_description,
    p.image_url as addon_image_url,
    p.price_type as addon_price_type,
    p.currency as addon_currency,
    p.unit_amount_in_pennies as addon_unit_amount,
    p.interval_type as addon_interval_type,
    p.term_length as addon_term_length,
    p.active as addon_active
FROM product_addon_relationships par
JOIN products p ON par.addon_product_id = p.id
WHERE par.base_product_id = $1
  AND p.deleted_at IS NULL
ORDER BY par.display_order, p.name;

-- name: ListRequiredProductAddons :many
SELECT 
    par.*,
    p.id as addon_id,
    p.name as addon_name,
    p.description as addon_description,
    p.image_url as addon_image_url,
    p.price_type as addon_price_type,
    p.currency as addon_currency,
    p.unit_amount_in_pennies as addon_unit_amount,
    p.interval_type as addon_interval_type,
    p.term_length as addon_term_length,
    p.active as addon_active
FROM product_addon_relationships par
JOIN products p ON par.addon_product_id = p.id
WHERE par.base_product_id = $1
  AND par.is_required = true
  AND p.deleted_at IS NULL
ORDER BY par.display_order, p.name;

-- name: ListBaseProductsForAddon :many
SELECT 
    par.*,
    p.id as base_id,
    p.name as base_name,
    p.description as base_description,
    p.image_url as base_image_url,
    p.price_type as base_price_type,
    p.currency as base_currency,
    p.unit_amount_in_pennies as base_unit_amount,
    p.interval_type as base_interval_type,
    p.term_length as base_term_length,
    p.active as base_active
FROM product_addon_relationships par
JOIN products p ON par.base_product_id = p.id
WHERE par.addon_product_id = $1
  AND p.deleted_at IS NULL
ORDER BY p.name;

-- name: UpdateProductAddonRelationship :one
UPDATE product_addon_relationships
SET
    is_required = COALESCE($2, is_required),
    max_quantity = COALESCE($3, max_quantity),
    min_quantity = COALESCE($4, min_quantity),
    display_order = COALESCE($5, display_order),
    metadata = COALESCE($6, metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteProductAddonRelationship :exec
DELETE FROM product_addon_relationships
WHERE id = $1;

-- name: DeleteProductAddonRelationshipByProducts :exec
DELETE FROM product_addon_relationships
WHERE base_product_id = $1 AND addon_product_id = $2;

-- name: DeleteAllAddonsForProduct :exec
DELETE FROM product_addon_relationships
WHERE base_product_id = $1;

-- name: CountProductAddons :one
SELECT COUNT(*) FROM product_addon_relationships
WHERE base_product_id = $1;

-- name: ValidateAddonForProduct :one
SELECT EXISTS(
    SELECT 1 FROM product_addon_relationships
    WHERE base_product_id = $1 AND addon_product_id = $2
) as is_valid;