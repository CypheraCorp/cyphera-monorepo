-- name: CreateDashboardMetric :one
INSERT INTO dashboard_metrics (
    workspace_id,
    metric_date,
    metric_type,
    metric_hour,
    fiat_currency,
    mrr_cents,
    arr_cents,
    total_revenue_cents,
    new_revenue_cents,
    expansion_revenue_cents,
    contraction_revenue_cents,
    total_customers,
    new_customers,
    churned_customers,
    reactivated_customers,
    active_subscriptions,
    new_subscriptions,
    cancelled_subscriptions,
    paused_subscriptions,
    trial_subscriptions,
    churn_rate,
    growth_rate,
    ltv_avg_cents,
    successful_payments,
    failed_payments,
    pending_payments,
    total_payment_volume_cents,
    avg_payment_size_cents,
    total_gas_fees_cents,
    sponsored_gas_fees_cents,
    customer_gas_fees_cents,
    avg_gas_fee_cents,
    gas_sponsorship_rate,
    unique_wallet_addresses,
    new_wallet_addresses,
    network_metrics,
    token_metrics,
    avg_payment_confirmation_time_seconds,
    payment_success_rate
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38, $39
)
ON CONFLICT (workspace_id, metric_date, metric_type, metric_hour, fiat_currency)
DO UPDATE SET
    mrr_cents = EXCLUDED.mrr_cents,
    arr_cents = EXCLUDED.arr_cents,
    total_revenue_cents = EXCLUDED.total_revenue_cents,
    new_revenue_cents = EXCLUDED.new_revenue_cents,
    expansion_revenue_cents = EXCLUDED.expansion_revenue_cents,
    contraction_revenue_cents = EXCLUDED.contraction_revenue_cents,
    total_customers = EXCLUDED.total_customers,
    new_customers = EXCLUDED.new_customers,
    churned_customers = EXCLUDED.churned_customers,
    reactivated_customers = EXCLUDED.reactivated_customers,
    active_subscriptions = EXCLUDED.active_subscriptions,
    new_subscriptions = EXCLUDED.new_subscriptions,
    cancelled_subscriptions = EXCLUDED.cancelled_subscriptions,
    paused_subscriptions = EXCLUDED.paused_subscriptions,
    trial_subscriptions = EXCLUDED.trial_subscriptions,
    churn_rate = EXCLUDED.churn_rate,
    growth_rate = EXCLUDED.growth_rate,
    ltv_avg_cents = EXCLUDED.ltv_avg_cents,
    successful_payments = EXCLUDED.successful_payments,
    failed_payments = EXCLUDED.failed_payments,
    pending_payments = EXCLUDED.pending_payments,
    total_payment_volume_cents = EXCLUDED.total_payment_volume_cents,
    avg_payment_size_cents = EXCLUDED.avg_payment_size_cents,
    total_gas_fees_cents = EXCLUDED.total_gas_fees_cents,
    sponsored_gas_fees_cents = EXCLUDED.sponsored_gas_fees_cents,
    customer_gas_fees_cents = EXCLUDED.customer_gas_fees_cents,
    avg_gas_fee_cents = EXCLUDED.avg_gas_fee_cents,
    gas_sponsorship_rate = EXCLUDED.gas_sponsorship_rate,
    unique_wallet_addresses = EXCLUDED.unique_wallet_addresses,
    new_wallet_addresses = EXCLUDED.new_wallet_addresses,
    network_metrics = EXCLUDED.network_metrics,
    token_metrics = EXCLUDED.token_metrics,
    avg_payment_confirmation_time_seconds = EXCLUDED.avg_payment_confirmation_time_seconds,
    payment_success_rate = EXCLUDED.payment_success_rate,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetDashboardMetric :one
SELECT * FROM dashboard_metrics
WHERE workspace_id = $1
    AND metric_date = $2
    AND metric_type = $3
    AND metric_hour = $4
    AND fiat_currency = $5;

-- name: GetLatestDashboardMetrics :one
SELECT * FROM dashboard_metrics
WHERE workspace_id = $1
    AND metric_type = $2
    AND fiat_currency = $3
ORDER BY metric_date DESC, metric_hour DESC NULLS LAST
LIMIT 1;

-- name: GetDashboardMetricsByDateRange :many
SELECT * FROM dashboard_metrics
WHERE workspace_id = $1
    AND metric_date >= $2
    AND metric_date <= $3
    AND metric_type = $4
    AND fiat_currency = $5
ORDER BY metric_date DESC, metric_hour DESC NULLS LAST;

-- name: GetHourlyMetrics :many
SELECT * FROM dashboard_metrics
WHERE workspace_id = $1
    AND metric_date = $2
    AND metric_type = 'hourly'
    AND fiat_currency = $3
ORDER BY metric_hour ASC;

-- name: GetDailyMetrics :many
SELECT * FROM dashboard_metrics
WHERE workspace_id = $1
    AND metric_date >= $2
    AND metric_date <= $3
    AND metric_type = 'daily'
    AND fiat_currency = $4
ORDER BY metric_date DESC;

-- name: GetMonthlyMetrics :many
SELECT * FROM dashboard_metrics
WHERE workspace_id = $1
    AND metric_date >= DATE_TRUNC('month', $2::date)
    AND metric_date <= DATE_TRUNC('month', $3::date)
    AND metric_type = 'monthly'
    AND fiat_currency = $4
ORDER BY metric_date DESC;

-- name: GetCurrentMRR :one
SELECT 
    mrr_cents,
    arr_cents,
    active_subscriptions,
    total_customers
FROM dashboard_metrics
WHERE workspace_id = $1
    AND metric_type = 'monthly'
    AND metric_date = DATE_TRUNC('month', CURRENT_DATE)
    AND fiat_currency = $2;

-- name: GetRevenueGrowth :one
SELECT 
    dm1.metric_date as current_period,
    dm1.total_revenue_cents as current_revenue,
    dm2.total_revenue_cents as previous_revenue,
    CASE 
        WHEN dm2.total_revenue_cents > 0 
        THEN ((dm1.total_revenue_cents - dm2.total_revenue_cents)::FLOAT / dm2.total_revenue_cents) * 100
        ELSE 0
    END as growth_percentage
FROM dashboard_metrics dm1
LEFT JOIN dashboard_metrics dm2 ON 
    dm2.workspace_id = dm1.workspace_id 
    AND dm2.metric_type = dm1.metric_type
    AND dm2.fiat_currency = dm1.fiat_currency
    AND dm2.metric_date = CASE 
        WHEN dm1.metric_type = 'daily' THEN dm1.metric_date - INTERVAL '1 day'
        WHEN dm1.metric_type = 'monthly' THEN dm1.metric_date - INTERVAL '1 month'
        ELSE dm1.metric_date - INTERVAL '1 year'
    END
WHERE dm1.workspace_id = $1
    AND dm1.metric_date = $2
    AND dm1.metric_type = $3
    AND dm1.fiat_currency = $4;

-- name: GetPaymentMetricsSummary :one
SELECT 
    SUM(successful_payments) as total_successful_payments,
    SUM(failed_payments) as total_failed_payments,
    SUM(total_payment_volume_cents) as total_volume_cents,
    AVG(payment_success_rate) as avg_success_rate,
    SUM(total_gas_fees_cents) as total_gas_fees,
    SUM(sponsored_gas_fees_cents) as total_sponsored_gas
FROM dashboard_metrics
WHERE workspace_id = $1
    AND metric_date >= $2
    AND metric_date <= $3
    AND metric_type = $4
    AND fiat_currency = $5;

-- name: GetCustomerMetricsTrend :many
SELECT 
    metric_date,
    total_customers,
    new_customers,
    churned_customers,
    churn_rate,
    growth_rate
FROM dashboard_metrics
WHERE workspace_id = $1
    AND metric_date >= $2
    AND metric_date <= $3
    AND metric_type = $4
    AND fiat_currency = $5
ORDER BY metric_date ASC;

-- name: GetNetworkMetrics :one
SELECT 
    metric_date,
    network_metrics,
    token_metrics
FROM dashboard_metrics
WHERE workspace_id = $1
    AND metric_date = $2
    AND metric_type = $3
    AND fiat_currency = $4;

-- name: DeleteOldMetrics :exec
DELETE FROM dashboard_metrics
WHERE metric_date < $1
    AND metric_type = $2;

-- name: UpdateMetricNetworkData :one
UPDATE dashboard_metrics
SET 
    network_metrics = $5,
    token_metrics = $6,
    updated_at = CURRENT_TIMESTAMP
WHERE workspace_id = $1
    AND metric_date = $2
    AND metric_type = $3
    AND fiat_currency = $4
RETURNING *;