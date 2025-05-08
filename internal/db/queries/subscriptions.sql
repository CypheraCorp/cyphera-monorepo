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
SELECT s.* FROM subscriptions s
JOIN products p ON s.product_id = p.id
WHERE s.product_id = $1 AND p.workspace_id = $2 AND s.deleted_at IS NULL
ORDER BY s.created_at DESC;

-- name: ListActiveSubscriptions :many
SELECT * FROM subscriptions
WHERE (status = 'active' OR status = 'overdue') AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListSubscriptionsDueForRedemption :many
SELECT * FROM subscriptions
WHERE 
    (status = 'active' OR status = 'overdue')
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
WHERE (status = 'active' OR status = 'overdue') AND deleted_at IS NULL;

-- name: CountSubscriptionsByStatus :one
SELECT COUNT(*) FROM subscriptions
WHERE status = $1 AND deleted_at IS NULL;

-- name: CreateSubscription :one
INSERT INTO subscriptions (
    customer_id,
    product_id,
    product_token_id,
    token_amount,
    product_price_in_pennies,
    currency,
    interval_type,
    term_length,
    delegation_id,
    customer_wallet_id,
    status,
    current_period_start,
    current_period_end,
    next_redemption_date,
    total_redemptions,
    total_term_length,
    total_amount_in_cents,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
)
RETURNING *;

-- name: UpdateSubscription :one
UPDATE subscriptions
SET
    customer_id = COALESCE($2, customer_id),
    product_id = COALESCE($3, product_id),
    product_token_id = COALESCE($4, product_token_id),
    token_amount = COALESCE($5, token_amount),
    product_price_in_pennies = COALESCE($6, product_price_in_pennies),
    currency = COALESCE($7, currency),
    interval_type = COALESCE($8, interval_type),
    term_length = COALESCE($9, term_length),
    delegation_id = COALESCE($10, delegation_id),
    customer_wallet_id = COALESCE($11, customer_wallet_id),
    status = COALESCE($12, status),
    current_period_start = COALESCE($13, current_period_start),
    current_period_end = COALESCE($14, current_period_end),
    next_redemption_date = COALESCE($15, next_redemption_date),
    total_redemptions = COALESCE($16, total_redemptions),
    total_term_length = COALESCE($17, total_term_length),
    total_amount_in_cents = COALESCE($18, total_amount_in_cents),
    metadata = COALESCE($19, metadata),
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

-- name: GetOverdueSubscriptions :many
SELECT * FROM subscriptions
WHERE 
    (current_period_end < CURRENT_TIMESTAMP OR status = 'overdue')
    AND deleted_at IS NULL
ORDER BY current_period_end ASC;

-- name: LockSubscriptionForProcessing :one
SELECT *
FROM subscriptions
WHERE id = $1 AND (status = 'active' OR status = 'overdue') AND deleted_at IS NULL
FOR UPDATE NOWAIT
LIMIT 1;

-- name: ListSubscriptionDetailsWithPagination :many
SELECT 
    s.id,
    s.status,
    s.current_period_start,
    s.current_period_end,
    s.next_redemption_date,
    s.total_redemptions,
    s.total_term_length,
    s.total_amount_in_cents,
    s.token_amount,
    s.product_price_in_pennies,
    s.interval_type,
    s.term_length,
    s.currency,
    -- Customer details
    c.id as customer_id,
    c.name as customer_name,
    c.email as customer_email,
    -- Product details
    p.id as product_id,
    p.name as product_name,
    p.product_type,
    p.interval_type,
    p.price_in_pennies,
    -- Token details
    t.symbol as token_symbol,
    t.contract_address as token_address,
    -- Network details
    n.name as network_name,
    n.type as network_type,
    n.chain_id,
    -- Customer wallet details
    cw.wallet_address as customer_wallet_address
FROM subscriptions s
JOIN customers c ON s.customer_id = c.id
JOIN products p ON s.product_id = p.id
JOIN products_tokens pt ON s.product_token_id = pt.id
JOIN tokens t ON pt.token_id = t.id
JOIN networks n ON pt.network_id = n.id
LEFT JOIN customer_wallets cw ON s.customer_wallet_id = cw.id
WHERE s.deleted_at IS NULL
    AND c.deleted_at IS NULL
    AND p.deleted_at IS NULL
    AND pt.deleted_at IS NULL
    AND t.deleted_at IS NULL
    AND n.deleted_at IS NULL
    AND cw.deleted_at IS NULL
    AND p.workspace_id = $3
ORDER BY s.created_at DESC
LIMIT $1 OFFSET $2; 