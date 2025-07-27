-- name: CreateProrationRecord :one
INSERT INTO subscription_prorations (
    subscription_id,
    schedule_change_id,
    proration_type,
    period_start,
    period_end,
    days_total,
    days_used,
    days_remaining,
    original_amount_cents,
    used_amount_cents,
    credit_amount_cents,
    applied_to_invoice_id,
    applied_to_payment_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
) RETURNING *;

-- name: GetSubscriptionProrations :many
SELECT * FROM subscription_prorations
WHERE subscription_id = $1
ORDER BY created_at DESC;

-- name: GetProrationsByScheduleChange :many
SELECT * FROM subscription_prorations
WHERE schedule_change_id = $1;

-- name: GetUnappliedProrations :many
SELECT * FROM subscription_prorations
WHERE subscription_id = $1
AND applied_to_invoice_id IS NULL
AND applied_to_payment_id IS NULL
ORDER BY created_at ASC;

-- name: ApplyProrationToInvoice :one
UPDATE subscription_prorations
SET applied_to_invoice_id = $2
WHERE id = $1
AND applied_to_invoice_id IS NULL
AND applied_to_payment_id IS NULL
RETURNING *;

-- name: ApplyProrationToPayment :one
UPDATE subscription_prorations
SET applied_to_payment_id = $2
WHERE id = $1
AND applied_to_invoice_id IS NULL
AND applied_to_payment_id IS NULL
RETURNING *;

-- name: GetProrationSummaryBySubscription :one
SELECT 
    subscription_id,
    COUNT(*) as total_prorations,
    SUM(credit_amount_cents) as total_credits,
    SUM(CASE WHEN applied_to_invoice_id IS NOT NULL OR applied_to_payment_id IS NOT NULL THEN credit_amount_cents ELSE 0 END) as applied_credits,
    SUM(CASE WHEN applied_to_invoice_id IS NULL AND applied_to_payment_id IS NULL THEN credit_amount_cents ELSE 0 END) as unapplied_credits
FROM subscription_prorations
WHERE subscription_id = $1
GROUP BY subscription_id;