-- name: GetSubscriptionEvent :one
SELECT * FROM subscription_events
WHERE id = $1 LIMIT 1;

-- name: GetSubscriptionEventByTransactionHash :one
SELECT * FROM subscription_events
WHERE transaction_hash = $1 LIMIT 1;

-- name: ListSubscriptionEvents :many
SELECT * FROM subscription_events
ORDER BY occurred_at DESC;

-- name: ListSubscriptionEventsBySubscription :many
SELECT * FROM subscription_events
WHERE subscription_id = $1
ORDER BY occurred_at DESC;

-- name: ListSubscriptionEventsByType :many
SELECT * FROM subscription_events
WHERE event_type = $1
ORDER BY occurred_at DESC;

-- name: ListFailedSubscriptionEvents :many
SELECT * FROM subscription_events
WHERE event_type = 'failed'
ORDER BY occurred_at DESC;

-- name: ListRecentSubscriptionEvents :many
SELECT * FROM subscription_events
WHERE occurred_at >= $1
ORDER BY occurred_at DESC;

-- name: ListRecentSubscriptionEventsByType :many
SELECT * FROM subscription_events
WHERE event_type = $1 AND occurred_at >= $2
ORDER BY occurred_at DESC;

-- name: ListSubscriptionEventsWithPagination :many
SELECT * FROM subscription_events
ORDER BY occurred_at DESC
LIMIT $1 OFFSET $2;

-- name: CountSubscriptionEvents :one
SELECT COUNT(*) FROM subscription_events;

-- name: CountSubscriptionEventsByType :one
SELECT COUNT(*) FROM subscription_events
WHERE event_type = $1;

-- name: CountSubscriptionEventsBySubscription :one
SELECT COUNT(*) FROM subscription_events
WHERE subscription_id = $1;

-- name: CreateSubscriptionEvent :one
INSERT INTO subscription_events (
    subscription_id,
    event_type,
    transaction_hash,
    amount_in_cents,
    occurred_at,
    error_message,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: CreateRedemptionEvent :one
INSERT INTO subscription_events (
    subscription_id,
    event_type,
    transaction_hash,
    amount_in_cents,
    occurred_at,
    metadata
) VALUES (
    $1, 'redeemed', $2, $3, CURRENT_TIMESTAMP, $4
)
RETURNING *;

-- name: CreateFailedRedemptionEvent :one
INSERT INTO subscription_events (
    subscription_id,
    event_type,
    amount_in_cents,
    occurred_at,
    error_message,
    metadata
) VALUES (
    $1, 'failed', $2, CURRENT_TIMESTAMP, $3, $4
)
RETURNING *;

-- name: UpdateSubscriptionEvent :one
UPDATE subscription_events
SET
    event_type = COALESCE($2, event_type),
    transaction_hash = COALESCE($3, transaction_hash),
    amount_in_cents = COALESCE($4, amount_in_cents),
    error_message = COALESCE($5, error_message),
    metadata = COALESCE($6, metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: GetTotalAmountBySubscription :one
SELECT COALESCE(SUM(amount_in_cents), 0) as total_amount
FROM subscription_events
WHERE subscription_id = $1 AND event_type = 'redeemed';

-- name: GetSuccessfulRedemptionCount :one
SELECT COUNT(*) 
FROM subscription_events
WHERE subscription_id = $1 AND event_type = 'redeemed';

-- name: GetLatestSubscriptionEvent :one
SELECT * FROM subscription_events
WHERE subscription_id = $1
ORDER BY occurred_at DESC
LIMIT 1;

-- name: ListSubscriptionEventDetailsWithPagination :many
SELECT 
    se.id,
    se.event_type,
    se.transaction_hash,
    se.amount_in_cents,
    se.occurred_at,
    se.error_message,
    -- Subscription details
    s.id as subscription_id,
    s.status as subscription_status,
    -- Customer details
    c.id as customer_id,
    c.name as customer_name,
    c.email as customer_email,
    -- Product details
    p.id as product_id,
    p.name as product_name,
    p.product_type,
    p.interval_type,
    -- Token details
    t.symbol as token_symbol,
    t.contract_address as token_address,
    -- Network details
    n.name as network_name,
    n.type as network_type,
    n.chain_id,
    -- Customer wallet details
    cw.wallet_address as customer_wallet_address
FROM subscription_events se
JOIN subscriptions s ON se.subscription_id = s.id
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
    AND p.workspace_id = $3
    AND se.event_type IN ('redeemed', 'failed', 'failed_redemption')
ORDER BY se.occurred_at DESC
LIMIT $1 OFFSET $2;

-- name: CountSubscriptionEventDetails :one
SELECT COUNT(*) 
FROM subscription_events se
JOIN subscriptions s ON se.subscription_id = s.id
JOIN products p ON s.product_id = p.id
WHERE s.deleted_at IS NULL
    AND p.deleted_at IS NULL
    AND p.workspace_id = $1
    AND se.event_type IN ('redeemed', 'failed', 'failed_redemption');