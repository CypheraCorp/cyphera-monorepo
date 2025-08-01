-- name: ListRecentFailedPayments :many
-- Get recent failed payments that don't have dunning campaigns
SELECT DISTINCT ON (se.subscription_id) 
    se.*,
    s.workspace_id,
    s.customer_id,
    s.product_id,
    s.status as subscription_status
FROM subscription_events se
JOIN subscriptions s ON se.subscription_id = s.id
WHERE se.event_type IN ('failed', 'failed_redemption')
    AND se.occurred_at >= $1
    AND NOT EXISTS (
        SELECT 1 
        FROM dunning_campaigns dc 
        WHERE dc.subscription_id = se.subscription_id
            AND dc.status IN ('active', 'paused')
            AND dc.created_at > se.occurred_at
    )
ORDER BY se.subscription_id, se.occurred_at DESC;

-- name: GetFailedPaymentCount :one
-- Get count of failed payments for a subscription
SELECT COUNT(*) as failed_count
FROM subscription_events
WHERE subscription_id = $1
    AND event_type IN ('failed', 'failed_redemption')
    AND occurred_at >= $2;

-- name: CheckExistingDunningCampaign :one
-- Check if there's an active dunning campaign for a subscription
SELECT EXISTS(
    SELECT 1
    FROM dunning_campaigns
    WHERE subscription_id = $1
        AND status IN ('active', 'paused')
        AND (completed_at IS NULL OR completed_at > $2)
) as exists;

-- name: GetSubscriptionPaymentHistory :many
-- Get payment history for campaign strategy determination
SELECT 
    event_type,
    occurred_at,
    amount_in_cents,
    transaction_hash
FROM subscription_events
WHERE subscription_id = $1
    AND event_type IN ('redeemed', 'failed', 'failed_redemption')
ORDER BY occurred_at DESC
LIMIT 100;