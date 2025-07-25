-- name: CreateGasFeePayment :one
INSERT INTO gas_fee_payments (
    payment_id,
    gas_fee_wei,
    gas_price_gwei,
    gas_units_used,
    max_gas_units,
    base_fee_gwei,
    priority_fee_gwei,
    payment_token_id,
    payment_token_amount,
    payment_method,
    sponsor_type,
    sponsor_id,
    sponsor_workspace_id,
    network_id,
    block_number,
    block_timestamp,
    eth_usd_price,
    token_usd_price,
    gas_fee_usd_cents
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
)
RETURNING *;

-- name: GetGasFeePayment :one
SELECT * FROM gas_fee_payments
WHERE id = $1;

-- name: GetGasFeePaymentByPaymentId :one
SELECT * FROM gas_fee_payments
WHERE payment_id = $1;

-- name: GetGasFeePaymentsBySponsor :many
SELECT * FROM gas_fee_payments
WHERE sponsor_type = $1 AND sponsor_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetGasFeePaymentsByWorkspace :many
SELECT gfp.*, p.customer_id, p.amount_in_cents as payment_amount
FROM gas_fee_payments gfp
JOIN payments p ON gfp.payment_id = p.id
WHERE p.workspace_id = $1
ORDER BY gfp.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetGasFeeMetrics :one
SELECT 
    COUNT(*) as total_transactions,
    SUM(gas_fee_usd_cents) as total_gas_fees_cents,
    AVG(gas_fee_usd_cents) as avg_gas_fee_cents,
    SUM(gas_fee_usd_cents) FILTER (WHERE sponsor_type = 'merchant') as merchant_sponsored_cents,
    SUM(gas_fee_usd_cents) FILTER (WHERE sponsor_type = 'customer') as customer_paid_cents,
    SUM(gas_fee_usd_cents) FILTER (WHERE sponsor_type = 'platform') as platform_sponsored_cents,
    AVG(gas_units_used) as avg_gas_units,
    MAX(gas_fee_usd_cents) as max_gas_fee_cents,
    MIN(gas_fee_usd_cents) as min_gas_fee_cents
FROM gas_fee_payments gfp
JOIN payments p ON gfp.payment_id = p.id
WHERE p.workspace_id = $1
    AND gfp.created_at >= $2
    AND gfp.created_at < $3;

-- name: GetGasFeesByNetwork :many
SELECT 
    n.name as network_name,
    COUNT(*) as transaction_count,
    SUM(gfp.gas_fee_usd_cents) as total_gas_fees_cents,
    AVG(gfp.gas_fee_usd_cents) as avg_gas_fee_cents,
    AVG(gfp.gas_units_used) as avg_gas_units
FROM gas_fee_payments gfp
JOIN payments p ON gfp.payment_id = p.id
JOIN networks n ON gfp.network_id = n.id
WHERE p.workspace_id = $1
    AND gfp.created_at >= $2
    AND gfp.created_at < $3
GROUP BY n.id, n.name
ORDER BY total_gas_fees_cents DESC;

-- name: GetGasSponsorshipStats :one
SELECT 
    sponsor_workspace_id,
    COUNT(*) as sponsored_count,
    SUM(gas_fee_usd_cents) as total_sponsored_cents,
    AVG(gas_fee_usd_cents) as avg_sponsored_cents
FROM gas_fee_payments
WHERE sponsor_type = 'merchant'
    AND sponsor_workspace_id = $1
    AND created_at >= $2
    AND created_at < $3
GROUP BY sponsor_workspace_id;

-- name: GetCustomerGasPayments :many
SELECT 
    gfp.*,
    p.customer_id,
    c.email as customer_email,
    n.name as network_name
FROM gas_fee_payments gfp
JOIN payments p ON gfp.payment_id = p.id
JOIN customers c ON p.customer_id = c.id
JOIN networks n ON gfp.network_id = n.id
WHERE p.customer_id = $1
    AND gfp.sponsor_type = 'customer'
ORDER BY gfp.created_at DESC
LIMIT $2 OFFSET $3;