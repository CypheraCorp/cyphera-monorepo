-- name: GetSubscription :one
SELECT * FROM subscriptions
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetSubscriptionWithDetails :one
SELECT 
    s.*,
    p.name as product_name,
    p.product_type,
    p.interval_type,
    c.name as customer_name,
    c.email as customer_email,
    cw.wallet_address as subscriber_wallet_address,
    cw.network_type as subscriber_network_type,
    t.symbol as token_symbol,
    n.name as network_name,
    n.chain_id
FROM subscriptions s
JOIN products p ON p.id = s.product_id
JOIN customers c ON c.id = s.customer_id
LEFT JOIN customer_wallets cw ON cw.id = s.customer_wallet_id
JOIN products_tokens pt ON pt.id = s.product_token_id
JOIN tokens t ON t.id = pt.token_id
JOIN networks n ON n.id = pt.network_id
WHERE s.id = $1 AND s.deleted_at IS NULL;

-- name: ListSubscriptions :many
SELECT * FROM subscriptions
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListSubscriptionsByCustomer :many
SELECT * FROM subscriptions
WHERE customer_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListSubscriptionsByProduct :many
SELECT * FROM subscriptions
WHERE product_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListActiveSubscriptions :many
SELECT * FROM subscriptions
WHERE status = 'active' AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListSubscriptionsDueForRenewal :many
SELECT * FROM subscriptions
WHERE 
    status = 'active' 
    AND next_redemption_date <= $1
    AND deleted_at IS NULL
ORDER BY next_redemption_date ASC;

-- name: ListSubscriptionsWithPagination :many
SELECT * FROM subscriptions
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountSubscriptions :one
SELECT COUNT(*) FROM subscriptions
WHERE deleted_at IS NULL;

-- name: CountActiveSubscriptions :one
SELECT COUNT(*) FROM subscriptions
WHERE status = 'active' AND deleted_at IS NULL;

-- name: CountSubscriptionsByStatus :one
SELECT COUNT(*) FROM subscriptions
WHERE status = $1 AND deleted_at IS NULL;

-- name: CreateSubscription :one
INSERT INTO subscriptions (
    customer_id,
    product_id,
    product_token_id,
    delegation_id,
    customer_wallet_id,
    status,
    current_period_start,
    current_period_end,
    next_redemption_date,
    total_redemptions,
    total_amount_in_cents,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: UpdateSubscription :one
UPDATE subscriptions
SET
    customer_id = COALESCE($2, customer_id),
    product_id = COALESCE($3, product_id),
    product_token_id = COALESCE($4, product_token_id),
    delegation_id = COALESCE($5, delegation_id),
    customer_wallet_id = COALESCE($6, customer_wallet_id),
    status = COALESCE($7, status),
    current_period_start = COALESCE($8, current_period_start),
    current_period_end = COALESCE($9, current_period_end),
    next_redemption_date = COALESCE($10, next_redemption_date),
    total_redemptions = COALESCE($11, total_redemptions),
    total_amount_in_cents = COALESCE($12, total_amount_in_cents),
    metadata = COALESCE($13, metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteSubscription :exec
UPDATE subscriptions
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL;

-- name: CancelSubscription :one
UPDATE subscriptions
SET 
    status = 'canceled',
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: CompleteSubscription :one
UPDATE subscriptions
SET 
    status = 'completed',
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateSubscriptionStatus :one
UPDATE subscriptions
SET 
    status = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: IncrementSubscriptionRedemption :one
UPDATE subscriptions
SET 
    total_redemptions = total_redemptions + 1,
    total_amount_in_cents = total_amount_in_cents + $2,
    next_redemption_date = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GetSubscriptionsByDelegation :many
SELECT * FROM subscriptions
WHERE delegation_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetExpiredSubscriptions :many
SELECT * FROM subscriptions
WHERE 
    current_period_end < CURRENT_TIMESTAMP
    AND status = 'active'
    AND deleted_at IS NULL
ORDER BY current_period_end ASC;

-- name: LockSubscriptionForProcessing :one
SELECT *
FROM subscriptions
WHERE id = $1 AND status = 'active' AND deleted_at IS NULL
FOR UPDATE NOWAIT
LIMIT 1; 