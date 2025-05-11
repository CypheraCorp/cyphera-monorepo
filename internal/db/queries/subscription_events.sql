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
    se.id AS subscription_event_id,
    se.subscription_id,
    se.event_type,
    se.transaction_hash,
    se.amount_in_cents AS event_amount_in_cents,
    se.occurred_at AS event_occurred_at,
    se.error_message,
    se.metadata AS event_metadata,
    se.created_at AS event_created_at,
    s.customer_id,
    s.status AS subscription_status,
    p.id AS product_id,
    p.name AS product_name,
    pr.id AS price_id,
    pr.type AS price_type,
    pr.currency AS price_currency,
    pr.unit_amount_in_pennies AS price_unit_amount_in_pennies,
    pr.interval_type AS price_interval_type,
    pr.interval_count AS price_interval_count,
    pr.term_length AS price_term_length,
    pt.id AS product_token_id,
    pt.token_id AS product_token_token_id,
    pt.created_at AS product_token_created_at,
    pt.updated_at AS product_token_updated_at,
    t.symbol AS product_token_symbol,
    n.id AS network_id,
    n.name AS network_name,
    n.chain_id AS network_chain_id,
    c.email AS customer_email,
    c.name AS customer_name
FROM
    subscription_events se
JOIN
    subscriptions s ON se.subscription_id = s.id
JOIN
    products p ON s.product_id = p.id
JOIN
    prices pr ON s.price_id = pr.id
JOIN
    products_tokens pt ON s.product_token_id = pt.id
JOIN
    tokens t ON pt.token_id = t.id
JOIN
    networks n ON pt.network_id = n.id
JOIN
    customers c ON s.customer_id = c.id
WHERE
    p.workspace_id = $1
    AND s.deleted_at IS NULL
    AND p.deleted_at IS NULL
    AND pr.deleted_at IS NULL
    AND se.event_type IN ('redeemed', 'failed', 'failed_redemption')
ORDER BY
    se.occurred_at DESC
LIMIT $2 OFFSET $3;

-- name: CountSubscriptionEventDetails :one
SELECT COUNT(*) 
FROM subscription_events se
JOIN subscriptions s ON se.subscription_id = s.id
JOIN products p ON s.product_id = p.id
JOIN prices pr ON s.price_id = pr.id
WHERE s.deleted_at IS NULL
    AND p.deleted_at IS NULL
    AND pr.deleted_at IS NULL
    AND p.workspace_id = $1
    AND se.event_type IN ('redeemed', 'failed', 'failed_redemption');