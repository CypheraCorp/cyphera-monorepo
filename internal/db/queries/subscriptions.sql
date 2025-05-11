-- name: GetSubscription :one
SELECT * FROM subscriptions
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetSubscriptionWithDetails :one
SELECT 
    s.*,
    p.name as product_name,
    c.name as customer_name,
    c.email as customer_email,
    cw.wallet_address as subscriber_wallet_address,
    cw.network_type as subscriber_network_type,
    t.symbol as token_symbol,
    n.name as network_name,
    n.chain_id,
    pr.type AS price_type,
    pr.currency AS price_currency,
    pr.unit_amount_in_pennies AS price_unit_amount_in_pennies,
    pr.interval_type AS price_interval_type,
    pr.interval_count AS price_interval_count,
    pr.term_length AS price_term_length
FROM subscriptions s
JOIN products p ON p.id = s.product_id
JOIN prices pr ON pr.id = s.price_id
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
    price_id,
    product_token_id,
    token_amount,
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
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
RETURNING *;

-- name: UpdateSubscription :one
UPDATE subscriptions
SET
    customer_id = COALESCE($2, customer_id),
    product_id = COALESCE($3, product_id),
    price_id = COALESCE($4, price_id),
    product_token_id = COALESCE($5, product_token_id),
    token_amount = COALESCE($6, token_amount),
    delegation_id = COALESCE($7, delegation_id),
    customer_wallet_id = COALESCE($8, customer_wallet_id),
    status = COALESCE($9, status),
    current_period_start = COALESCE($10, current_period_start),
    current_period_end = COALESCE($11, current_period_end),
    next_redemption_date = COALESCE($12, next_redemption_date),
    total_redemptions = COALESCE($13, total_redemptions),
    total_amount_in_cents = COALESCE($14, total_amount_in_cents),
    metadata = COALESCE($15, metadata),
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
    s.id AS subscription_id,
    s.status AS subscription_status,
    s.current_period_start AS subscription_current_period_start,
    s.current_period_end AS subscription_current_period_end,
    s.created_at AS subscription_created_at,
    s.updated_at AS subscription_updated_at,
    s.token_amount AS subscription_token_amount,
    s.next_redemption_date AS subscription_next_redemption_date,
    s.total_redemptions AS subscription_total_redemptions,
    s.total_amount_in_cents AS subscription_total_amount_in_cents,

    -- Customer details
    c.id AS customer_id,
    c.name AS customer_name,
    c.email AS customer_email,

    -- Product details
    p.id AS product_id,
    p.name AS product_name,
    p.description AS product_description,
    p.image_url AS product_image_url,
    p.active AS product_active,
    p.metadata AS product_metadata,
    p.workspace_id AS product_workspace_id,

    -- Price details (from prices table)
    pr.id AS price_id,
    pr.product_id AS price_product_id, -- The product_id FK in the prices table
    pr.active AS price_active,
    pr.type AS price_type,
    pr.nickname AS price_nickname,
    pr.currency AS price_currency,
    pr.unit_amount_in_pennies AS price_unit_amount_in_pennies,
    pr.interval_type AS price_interval_type,
    pr.interval_count AS price_interval_count,
    pr.term_length AS price_term_length,
    pr.metadata AS price_metadata,
    pr.created_at AS price_created_at,
    pr.updated_at AS price_updated_at,

    -- Product token details
    pt.id AS product_token_id,
    pt.token_id AS product_token_token_id,
    pt.network_id AS product_token_network_id,
    pt.created_at AS product_token_created_at,
    pt.updated_at AS product_token_updated_at,
    

    -- Token details
    t.symbol AS token_symbol,
    t.contract_address AS token_address,

    -- Network details
    n.name AS network_name,
    n.type AS network_type,
    n.chain_id AS network_chain_id,

    -- Customer wallet details
    cw.wallet_address AS customer_wallet_address
FROM subscriptions s
JOIN customers c ON s.customer_id = c.id
JOIN products p ON s.product_id = p.id
JOIN prices pr ON s.price_id = pr.id
JOIN products_tokens pt ON s.product_token_id = pt.id
JOIN tokens t ON pt.token_id = t.id
JOIN networks n ON pt.network_id = n.id
LEFT JOIN customer_wallets cw ON s.customer_wallet_id = cw.id
WHERE s.deleted_at IS NULL
    AND c.deleted_at IS NULL
    AND p.deleted_at IS NULL
    AND pr.deleted_at IS NULL
    AND pt.deleted_at IS NULL
    AND t.deleted_at IS NULL
    AND n.deleted_at IS NULL
    AND (cw.id IS NULL OR cw.deleted_at IS NULL) -- Correct handling for LEFT JOIN on deletable table
    AND p.workspace_id = $3
ORDER BY s.created_at DESC
LIMIT $1 OFFSET $2; 