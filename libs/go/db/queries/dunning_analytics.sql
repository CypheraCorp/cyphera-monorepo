-- name: CreateOrUpdateDunningAnalytics :one
INSERT INTO dunning_analytics (
    workspace_id,
    period_start,
    period_end,
    period_type,
    total_campaigns_started,
    total_campaigns_recovered,
    total_campaigns_lost,
    recovery_rate,
    total_at_risk_cents,
    total_recovered_cents,
    total_lost_cents,
    total_payment_retries,
    successful_payment_retries,
    total_emails_sent,
    email_open_rate,
    email_click_rate,
    recovery_by_attempt,
    avg_hours_to_recovery
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
)
ON CONFLICT (workspace_id, period_start, period_end, period_type)
DO UPDATE SET
    total_campaigns_started = dunning_analytics.total_campaigns_started + EXCLUDED.total_campaigns_started,
    total_campaigns_recovered = dunning_analytics.total_campaigns_recovered + EXCLUDED.total_campaigns_recovered,
    total_campaigns_lost = dunning_analytics.total_campaigns_lost + EXCLUDED.total_campaigns_lost,
    recovery_rate = CASE 
        WHEN (dunning_analytics.total_campaigns_recovered + EXCLUDED.total_campaigns_recovered + 
              dunning_analytics.total_campaigns_lost + EXCLUDED.total_campaigns_lost) > 0
        THEN (dunning_analytics.total_campaigns_recovered + EXCLUDED.total_campaigns_recovered)::DECIMAL / 
             (dunning_analytics.total_campaigns_recovered + EXCLUDED.total_campaigns_recovered + 
              dunning_analytics.total_campaigns_lost + EXCLUDED.total_campaigns_lost)
        ELSE 0
    END,
    total_at_risk_cents = dunning_analytics.total_at_risk_cents + EXCLUDED.total_at_risk_cents,
    total_recovered_cents = dunning_analytics.total_recovered_cents + EXCLUDED.total_recovered_cents,
    total_lost_cents = dunning_analytics.total_lost_cents + EXCLUDED.total_lost_cents,
    total_payment_retries = dunning_analytics.total_payment_retries + EXCLUDED.total_payment_retries,
    successful_payment_retries = dunning_analytics.successful_payment_retries + EXCLUDED.successful_payment_retries,
    total_emails_sent = dunning_analytics.total_emails_sent + EXCLUDED.total_emails_sent,
    email_open_rate = EXCLUDED.email_open_rate, -- Use latest rate
    email_click_rate = EXCLUDED.email_click_rate, -- Use latest rate
    recovery_by_attempt = EXCLUDED.recovery_by_attempt,
    avg_hours_to_recovery = EXCLUDED.avg_hours_to_recovery,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetDunningAnalytics :one
SELECT * FROM dunning_analytics
WHERE workspace_id = $1
    AND period_start = $2
    AND period_end = $3
    AND period_type = $4;

-- name: ListDunningAnalyticsByPeriod :many
SELECT * FROM dunning_analytics
WHERE workspace_id = $1
    AND period_type = $2
    AND period_start >= $3
    AND period_end <= $4
ORDER BY period_start DESC;

-- name: GetDunningAnalyticsSummary :one
SELECT 
    SUM(total_campaigns_started) as total_campaigns,
    SUM(total_campaigns_recovered) as total_recovered,
    SUM(total_campaigns_lost) as total_lost,
    CASE 
        WHEN SUM(total_campaigns_recovered + total_campaigns_lost) > 0
        THEN SUM(total_campaigns_recovered)::DECIMAL / SUM(total_campaigns_recovered + total_campaigns_lost)
        ELSE 0
    END as overall_recovery_rate,
    SUM(total_at_risk_cents) as total_at_risk_cents,
    SUM(total_recovered_cents) as total_recovered_cents,
    SUM(total_lost_cents) as total_lost_cents,
    AVG(avg_hours_to_recovery) as avg_hours_to_recovery
FROM dunning_analytics
WHERE workspace_id = $1
    AND period_start >= $2
    AND period_end <= $3;

-- name: GetDunningPerformanceByAttempt :one
SELECT 
    jsonb_build_object(
        '1', COALESCE(SUM((recovery_by_attempt->>'1')::int), 0),
        '2', COALESCE(SUM((recovery_by_attempt->>'2')::int), 0),
        '3', COALESCE(SUM((recovery_by_attempt->>'3')::int), 0),
        '4', COALESCE(SUM((recovery_by_attempt->>'4')::int), 0)
    ) as recovery_by_attempt
FROM dunning_analytics
WHERE workspace_id = $1
    AND period_start >= $2
    AND period_end <= $3;