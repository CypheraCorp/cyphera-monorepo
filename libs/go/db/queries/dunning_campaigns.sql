-- name: CreateDunningCampaign :one
INSERT INTO dunning_campaigns (
    workspace_id,
    configuration_id,
    subscription_id,
    payment_id,
    customer_id,
    status,
    original_failure_reason,
    original_amount_cents,
    currency,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: GetDunningCampaign :one
SELECT 
    dc.*,
    dconf.name as configuration_name,
    dconf.max_retry_attempts,
    dconf.retry_interval_days,
    c.email as customer_email,
    c.name as customer_name
FROM dunning_campaigns dc
JOIN dunning_configurations dconf ON dc.configuration_id = dconf.id
JOIN customers c ON dc.customer_id = c.id
WHERE dc.id = $1;

-- name: GetActiveDunningCampaignForSubscription :one
SELECT * FROM dunning_campaigns
WHERE subscription_id = $1 
    AND status = 'active';

-- name: GetActiveDunningCampaignForPayment :one
SELECT * FROM dunning_campaigns
WHERE payment_id = $1 
    AND status = 'active';

-- name: ListDunningCampaigns :many
SELECT 
    dc.*,
    c.email as customer_email,
    c.name as customer_name,
    s.product_id as subscription_product_id
FROM dunning_campaigns dc
JOIN customers c ON dc.customer_id = c.id
LEFT JOIN subscriptions s ON dc.subscription_id = s.id
WHERE dc.workspace_id = $1
    AND (@status::varchar IS NULL OR dc.status = @status)
    AND (@customer_id::uuid IS NULL OR dc.customer_id = @customer_id)
ORDER BY dc.created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListDunningCampaignsForRetry :many
SELECT * FROM dunning_campaigns
WHERE status = 'active'
    AND next_retry_at <= CURRENT_TIMESTAMP
ORDER BY next_retry_at
LIMIT $1;

-- name: UpdateDunningCampaign :one
UPDATE dunning_campaigns
SET 
    status = COALESCE($2, status),
    current_attempt = COALESCE($3, current_attempt),
    next_retry_at = $4,
    last_retry_at = $5,
    recovered = COALESCE($6, recovered),
    recovered_at = $7,
    recovered_amount_cents = COALESCE($8, recovered_amount_cents),
    final_action_taken = COALESCE($9, final_action_taken),
    final_action_at = $10,
    completed_at = $11,
    metadata = COALESCE($12, metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: RecoverDunningCampaign :one
UPDATE dunning_campaigns
SET 
    status = 'completed',
    recovered = true,
    recovered_at = CURRENT_TIMESTAMP,
    recovered_amount_cents = $2,
    completed_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: FailDunningCampaign :one
UPDATE dunning_campaigns
SET 
    status = 'completed',
    recovered = false,
    final_action_taken = $2,
    final_action_at = CURRENT_TIMESTAMP,
    completed_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: PauseDunningCampaign :one
UPDATE dunning_campaigns
SET 
    status = 'paused',
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: ResumeDunningCampaign :one
UPDATE dunning_campaigns
SET 
    status = 'active',
    next_retry_at = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: GetDunningCampaignStats :one
SELECT 
    COUNT(*) FILTER (WHERE status = 'active') as active_campaigns,
    COUNT(*) FILTER (WHERE status = 'completed' AND recovered = true) as recovered_campaigns,
    COUNT(*) FILTER (WHERE status = 'completed' AND recovered = false) as lost_campaigns,
    SUM(original_amount_cents) FILTER (WHERE status = 'active') as at_risk_amount_cents,
    SUM(recovered_amount_cents) FILTER (WHERE recovered = true) as recovered_amount_cents,
    SUM(original_amount_cents) FILTER (WHERE status = 'completed' AND recovered = false) as lost_amount_cents
FROM dunning_campaigns
WHERE workspace_id = $1
    AND created_at >= $2
    AND created_at < $3;

-- name: GetCampaignsNeedingFinalAction :many
SELECT 
    dc.*,
    dconf.final_action,
    dconf.final_action_config
FROM dunning_campaigns dc
JOIN dunning_configurations dconf ON dc.configuration_id = dconf.id
WHERE dc.status = 'active'
AND dc.current_attempt_number >= dconf.max_retry_attempts
AND dc.final_action_taken IS NULL
AND dconf.final_action IS NOT NULL
AND dconf.final_action != ''
ORDER BY dc.created_at ASC;