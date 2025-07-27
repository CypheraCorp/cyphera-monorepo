-- name: RecordStateChange :one
INSERT INTO subscription_state_history (
    subscription_id,
    from_status,
    to_status,
    from_amount_cents,
    to_amount_cents,
    line_items_snapshot,
    change_reason,
    schedule_change_id,
    initiated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetSubscriptionStateHistory :many
SELECT * FROM subscription_state_history
WHERE subscription_id = $1
ORDER BY occurred_at DESC
LIMIT $2;

-- name: GetStateChangesByScheduleChange :many
SELECT * FROM subscription_state_history
WHERE schedule_change_id = $1
ORDER BY occurred_at DESC;

-- name: GetLatestStateChange :one
SELECT * FROM subscription_state_history
WHERE subscription_id = $1
ORDER BY occurred_at DESC
LIMIT 1;

-- name: GetStateChangesByDateRange :many
SELECT * FROM subscription_state_history
WHERE subscription_id = $1
AND occurred_at >= $2
AND occurred_at <= $3
ORDER BY occurred_at DESC;

-- name: GetSubscriptionLifecycleEvents :many
SELECT 
    ssh.*,
    ssc.change_type as schedule_change_type,
    ssc.proration_amount_cents
FROM subscription_state_history ssh
LEFT JOIN subscription_schedule_changes ssc ON ssh.schedule_change_id = ssc.id
WHERE ssh.subscription_id = $1
ORDER BY ssh.occurred_at DESC;