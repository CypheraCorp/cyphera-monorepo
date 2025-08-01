-- name: CreateScheduleChange :one
INSERT INTO subscription_schedule_changes (
    subscription_id,
    change_type,
    scheduled_for,
    from_line_items,
    to_line_items,
    proration_amount_cents,
    proration_calculation,
    status,
    reason,
    initiated_by,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetScheduleChange :one
SELECT * FROM subscription_schedule_changes
WHERE id = $1;

-- name: GetDueScheduledChanges :many
SELECT * FROM subscription_schedule_changes
WHERE status = 'scheduled' 
AND scheduled_for <= $1
ORDER BY scheduled_for ASC;

-- name: GetSubscriptionScheduledChanges :many
SELECT * FROM subscription_schedule_changes
WHERE subscription_id = $1
AND status IN ('scheduled', 'processing')
ORDER BY scheduled_for ASC;

-- name: UpdateScheduleChangeStatus :one
UPDATE subscription_schedule_changes
SET 
    status = $2,
    processed_at = CASE WHEN $2 IN ('completed', 'failed') THEN CURRENT_TIMESTAMP ELSE processed_at END,
    error_message = COALESCE($3, error_message),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: CancelScheduledChange :one
UPDATE subscription_schedule_changes
SET 
    status = 'cancelled',
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 
AND status = 'scheduled'
RETURNING *;

-- name: GetSubscriptionWithCustomerDetails :one
SELECT 
    s.*,
    c.name as customer_name,
    c.email as customer_email,
    p.name as product_name,
    p.unit_amount_in_pennies as price_amount
FROM subscriptions s
JOIN customers c ON s.customer_id = c.id
JOIN products p ON s.product_id = p.id
WHERE s.id = $1;

-- name: UpdateSubscriptionForUpgrade :one
UPDATE subscriptions
SET 
    product_id = COALESCE($2, product_id),
    total_amount_in_cents = COALESCE($3, total_amount_in_cents),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: ScheduleSubscriptionCancellation :one
UPDATE subscriptions
SET 
    cancel_at = $2,
    cancellation_reason = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: CancelSubscriptionImmediately :one
UPDATE subscriptions
SET 
    status = 'canceled',
    cancelled_at = CURRENT_TIMESTAMP,
    cancellation_reason = COALESCE($2, cancellation_reason),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: PauseSubscription :one
UPDATE subscriptions
SET 
    status = 'suspended',
    paused_at = CURRENT_TIMESTAMP,
    pause_ends_at = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: ResumeSubscription :one
UPDATE subscriptions
SET 
    status = 'active',
    paused_at = NULL,
    pause_ends_at = NULL,
    current_period_start = $2,
    current_period_end = $3,
    next_redemption_date = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: ReactivateScheduledCancellation :one
UPDATE subscriptions
SET 
    cancel_at = NULL,
    cancellation_reason = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
AND cancel_at IS NOT NULL
AND status = 'active'
RETURNING *;

-- name: GetSubscriptionsDueForCancellation :many
SELECT * FROM subscriptions
WHERE status = 'active'
AND cancel_at IS NOT NULL
AND cancel_at <= $1
ORDER BY cancel_at ASC;

-- name: GetSubscriptionsDueForResumption :many
SELECT * FROM subscriptions
WHERE status = 'suspended'
AND pause_ends_at IS NOT NULL
AND pause_ends_at <= $1
ORDER BY pause_ends_at ASC;