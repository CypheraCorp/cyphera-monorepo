-- name: CreatePaymentLink :one
INSERT INTO payment_links (
    workspace_id,
    slug,
    status,
    product_id,
    amount_in_cents,
    currency,
    payment_type,
    collect_email,
    collect_shipping,
    collect_name,
    expires_at,
    max_uses,
    redirect_url,
    qr_code_url,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
)
RETURNING *;

-- name: GetPaymentLink :one
SELECT * FROM payment_links
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL;

-- name: GetPaymentLinkBySlug :one
SELECT * FROM payment_links
WHERE slug = $1 AND deleted_at IS NULL;

-- name: GetActivePaymentLinkBySlug :one
SELECT * FROM payment_links
WHERE slug = $1 
    AND status = 'active'
    AND deleted_at IS NULL
    AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
    AND (max_uses IS NULL OR used_count < max_uses);

-- name: GetPaymentLinksByWorkspace :many
SELECT * FROM payment_links
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetPaymentLinksByProduct :many
SELECT * FROM payment_links
WHERE workspace_id = $1 
    AND product_id = $2
    AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: UpdatePaymentLink :one
UPDATE payment_links
SET 
    status = COALESCE($3, status),
    expires_at = COALESCE($4, expires_at),
    max_uses = COALESCE($5, max_uses),
    redirect_url = COALESCE($6, redirect_url),
    metadata = COALESCE($7, metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: IncrementPaymentLinkUsage :one
UPDATE payment_links
SET 
    used_count = used_count + 1,
    status = CASE 
        WHEN max_uses IS NOT NULL AND used_count + 1 >= max_uses THEN 'inactive'
        ELSE status
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2
RETURNING *;

-- name: DeactivatePaymentLink :one
UPDATE payment_links
SET 
    status = 'inactive',
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: DeletePaymentLink :one
UPDATE payment_links
SET 
    deleted_at = CURRENT_TIMESTAMP,
    status = 'inactive'
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: ExpirePaymentLinks :exec
UPDATE payment_links
SET 
    status = 'expired',
    updated_at = CURRENT_TIMESTAMP
WHERE status = 'active' 
    AND expires_at IS NOT NULL 
    AND expires_at <= CURRENT_TIMESTAMP
    AND deleted_at IS NULL;

-- name: GetPaymentLinkStats :one
SELECT 
    COUNT(*) as total_links,
    COUNT(*) FILTER (WHERE status = 'active') as active_links,
    COUNT(*) FILTER (WHERE status = 'inactive') as inactive_links,
    COUNT(*) FILTER (WHERE status = 'expired') as expired_links,
    SUM(used_count) as total_uses
FROM payment_links
WHERE workspace_id = $1 AND deleted_at IS NULL;

-- name: GetTopPaymentLinks :many
SELECT 
    pl.*,
    p.name as product_name,
    p.unit_amount_in_pennies as price_amount
FROM payment_links pl
LEFT JOIN products p ON pl.product_id = p.id
WHERE pl.workspace_id = $1 
    AND pl.deleted_at IS NULL
ORDER BY pl.used_count DESC
LIMIT $2;

-- name: CheckSlugExists :one
SELECT EXISTS(
    SELECT 1 FROM payment_links 
    WHERE slug = $1 AND deleted_at IS NULL
) as exists;

-- name: UpdatePaymentLinkQRCode :one
UPDATE payment_links
SET 
    qr_code_url = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;