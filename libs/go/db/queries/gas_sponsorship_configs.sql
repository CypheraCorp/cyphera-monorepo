-- name: CreateGasSponsorshipConfig :one
INSERT INTO gas_sponsorship_configs (
    workspace_id,
    sponsorship_enabled,
    sponsor_customer_gas,
    sponsor_threshold_usd_cents,
    monthly_budget_usd_cents,
    sponsor_for_products,
    sponsor_for_customers,
    sponsor_for_tiers
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
ON CONFLICT (workspace_id)
DO UPDATE SET
    sponsorship_enabled = EXCLUDED.sponsorship_enabled,
    sponsor_customer_gas = EXCLUDED.sponsor_customer_gas,
    sponsor_threshold_usd_cents = EXCLUDED.sponsor_threshold_usd_cents,
    monthly_budget_usd_cents = EXCLUDED.monthly_budget_usd_cents,
    sponsor_for_products = EXCLUDED.sponsor_for_products,
    sponsor_for_customers = EXCLUDED.sponsor_for_customers,
    sponsor_for_tiers = EXCLUDED.sponsor_for_tiers,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetGasSponsorshipConfig :one
SELECT * FROM gas_sponsorship_configs
WHERE workspace_id = $1;

-- name: UpdateGasSponsorshipConfig :one
UPDATE gas_sponsorship_configs
SET 
    sponsorship_enabled = COALESCE($2, sponsorship_enabled),
    sponsor_customer_gas = COALESCE($3, sponsor_customer_gas),
    sponsor_threshold_usd_cents = COALESCE($4, sponsor_threshold_usd_cents),
    monthly_budget_usd_cents = COALESCE($5, monthly_budget_usd_cents),
    sponsor_for_products = COALESCE($6, sponsor_for_products),
    sponsor_for_customers = COALESCE($7, sponsor_for_customers),
    sponsor_for_tiers = COALESCE($8, sponsor_for_tiers),
    updated_at = CURRENT_TIMESTAMP
WHERE workspace_id = $1
RETURNING *;

-- name: UpdateGasSponsorshipSpending :exec
UPDATE gas_sponsorship_configs
SET 
    current_month_spent_cents = current_month_spent_cents + $2,
    updated_at = COALESCE($3, CURRENT_TIMESTAMP)
WHERE workspace_id = $1;

-- name: ResetGasSponsorshipMonthlySpending :exec
UPDATE gas_sponsorship_configs
SET 
    current_month_spent_cents = 0,
    last_reset_date = $2,
    updated_at = COALESCE($3, CURRENT_TIMESTAMP)
WHERE workspace_id = $1;

-- name: GetActiveGasSponsorships :many
SELECT * FROM gas_sponsorship_configs
WHERE sponsorship_enabled = true
    AND sponsor_customer_gas = true
    AND (monthly_budget_usd_cents IS NULL OR current_month_spent_cents < monthly_budget_usd_cents);

-- name: CheckGasSponsorshipEligibility :one
SELECT 
    sponsorship_enabled,
    sponsor_customer_gas,
    sponsor_threshold_usd_cents,
    monthly_budget_usd_cents,
    current_month_spent_cents,
    CASE 
        WHEN NOT sponsorship_enabled THEN false
        WHEN NOT sponsor_customer_gas THEN false
        WHEN monthly_budget_usd_cents IS NOT NULL AND current_month_spent_cents >= monthly_budget_usd_cents THEN false
        ELSE true
    END as is_eligible,
    CASE 
        WHEN monthly_budget_usd_cents IS NOT NULL 
        THEN monthly_budget_usd_cents - current_month_spent_cents
        ELSE NULL
    END as remaining_budget_cents
FROM gas_sponsorship_configs
WHERE workspace_id = $1;

-- name: GetGasSponsorshipsByProduct :one
SELECT 
    workspace_id,
    sponsorship_enabled,
    sponsor_for_products
FROM gas_sponsorship_configs
WHERE workspace_id = $1
    AND sponsorship_enabled = true
    AND sponsor_for_products ? $2;

-- name: GetGasSponsorshipsByCustomer :one
SELECT 
    workspace_id,
    sponsorship_enabled,
    sponsor_for_customers
FROM gas_sponsorship_configs
WHERE workspace_id = $1
    AND sponsorship_enabled = true
    AND sponsor_for_customers ? $2;

-- name: GetWorkspacesNeedingReset :many
SELECT workspace_id 
FROM gas_sponsorship_configs
WHERE (last_reset_date IS NULL OR last_reset_date < DATE_TRUNC('month', CURRENT_DATE))
    AND current_month_spent_cents > 0;

-- name: GetSponsorshipConfigsNeedingReset :many
SELECT * FROM gas_sponsorship_configs
WHERE sponsorship_enabled = true
    AND (last_reset_date IS NULL 
        OR last_reset_date < date_trunc('month', $1::date))
ORDER BY workspace_id;