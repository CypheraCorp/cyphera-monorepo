-- name: CreatePayment :one
INSERT INTO payments (
    workspace_id,
    invoice_id,
    subscription_id,
    subscription_event,
    customer_id,
    amount_in_cents,
    currency,
    status,
    payment_method,
    transaction_hash,
    network_id,
    token_id,
    crypto_amount,
    exchange_rate,
    has_gas_fee,
    gas_fee_usd_cents,
    gas_sponsored,
    external_payment_id,
    payment_provider,
    product_amount_cents,
    tax_amount_cents,
    gas_amount_cents,
    discount_amount_cents,
    error_message,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25
)
RETURNING *;

-- name: GetPayment :one
SELECT * FROM payments
WHERE id = $1 AND workspace_id = $2;

-- name: GetPaymentByTransactionHash :one
SELECT * FROM payments
WHERE transaction_hash = $1;

-- name: GetPaymentsByTransactionHash :many
SELECT * FROM payments
WHERE transaction_hash = $1;

-- name: GetPaymentsByWorkspace :many
SELECT * FROM payments
WHERE workspace_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetPaymentsByCustomer :many
SELECT * FROM payments
WHERE customer_id = $1 AND workspace_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetPaymentsByInvoice :many
SELECT * FROM payments
WHERE invoice_id = $1 AND workspace_id = $2
ORDER BY created_at DESC;

-- name: GetPaymentsBySubscription :many
SELECT * FROM payments
WHERE subscription_id = $1 AND workspace_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetPaymentsByStatus :many
SELECT * FROM payments
WHERE workspace_id = $1 AND status = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: UpdatePaymentStatus :one
UPDATE payments
SET 
    status = $3,
    completed_at = CASE WHEN $3 = 'completed' THEN CURRENT_TIMESTAMP ELSE completed_at END,
    failed_at = CASE WHEN $3 = 'failed' THEN CURRENT_TIMESTAMP ELSE failed_at END,
    error_message = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2
RETURNING *;

-- name: UpdatePaymentGasDetails :one
UPDATE payments
SET 
    has_gas_fee = $3,
    gas_fee_usd_cents = $4,
    gas_sponsored = $5,
    gas_amount_cents = CASE WHEN $5 = false THEN $4 ELSE 0 END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2
RETURNING *;

-- name: GetPaymentMetrics :one
SELECT 
    COUNT(*) FILTER (WHERE status = 'completed') as completed_count,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
    COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
    SUM(amount_in_cents) FILTER (WHERE status = 'completed') as total_completed_cents,
    SUM(gas_fee_usd_cents) FILTER (WHERE status = 'completed' AND has_gas_fee = true) as total_gas_fees_cents,
    SUM(gas_fee_usd_cents) FILTER (WHERE status = 'completed' AND gas_sponsored = true) as sponsored_gas_fees_cents
FROM payments
WHERE workspace_id = $1
    AND created_at >= $2
    AND created_at < $3
    AND currency = $4;

-- name: GetPaymentsByDateRange :many
SELECT * FROM payments
WHERE workspace_id = $1
    AND created_at >= $2
    AND created_at < $3
    AND currency = $4
ORDER BY created_at DESC;

-- name: GetUnreconciledPayments :many
SELECT * FROM payments
WHERE workspace_id = $1
    AND status = 'completed'
    AND invoice_id IS NULL
    AND created_at >= $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: LinkPaymentToInvoice :one
UPDATE payments
SET 
    invoice_id = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2
RETURNING *;

-- name: RefundPayment :one
UPDATE payments
SET 
    status = 'refunded',
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND status = 'completed'
RETURNING *;

-- name: CountPaymentsByWorkspace :one
SELECT COUNT(*) FROM payments
WHERE workspace_id = $1;

-- name: GetPaymentVolume :one
SELECT 
    SUM(amount_in_cents) as total_volume_cents,
    COUNT(*) as payment_count
FROM payments
WHERE workspace_id = $1
    AND status = 'completed'
    AND completed_at >= $2
    AND completed_at < $3
    AND currency = $4;

-- name: GetPaymentsByExternalId :one
SELECT * FROM payments
WHERE workspace_id = $1
    AND external_payment_id = $2
    AND payment_provider = $3;

-- name: UpdatePaymentInvoiceID :one
UPDATE payments
SET invoice_id = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: CreatePaymentBatch :copyfrom
INSERT INTO payments (
    workspace_id,
    customer_id,
    amount_in_cents,
    currency,
    status,
    payment_method,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);

-- name: GetPaymentWithGasDetails :one
SELECT 
    p.*,
    gfp.id as gas_fee_payment_id,
    gfp.gas_fee_wei,
    gfp.gas_price_gwei,
    gfp.gas_units_used,
    gfp.max_gas_units,
    gfp.base_fee_gwei,
    gfp.priority_fee_gwei,
    gfp.payment_token_id as gas_payment_token_id,
    gfp.payment_token_amount as gas_payment_token_amount,
    gfp.payment_method as gas_payment_method,
    gfp.sponsor_type,
    gfp.sponsor_id,
    gfp.sponsor_workspace_id,
    gfp.block_number,
    gfp.block_timestamp
FROM payments p
LEFT JOIN gas_fee_payments gfp ON gfp.payment_id = p.id
WHERE p.id = $1 AND p.workspace_id = $2;

-- name: GetPaymentsByCustomerWithGas :many
SELECT 
    p.*,
    CASE 
        WHEN p.has_gas_fee THEN gfp.gas_fee_wei
        ELSE NULL
    END as gas_fee_wei,
    CASE 
        WHEN p.has_gas_fee THEN gfp.sponsor_type
        ELSE NULL
    END as gas_sponsor_type
FROM payments p
LEFT JOIN gas_fee_payments gfp ON gfp.payment_id = p.id
WHERE p.customer_id = $1 AND p.workspace_id = $2
ORDER BY p.created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetPaymentBySubscriptionEvent :one
SELECT * FROM payments
WHERE subscription_event = $1
LIMIT 1;

-- name: UpdatePaymentWithBlockchainData :one
UPDATE payments
SET 
    status = $3,
    gas_fee_usd_cents = $4,
    updated_at = COALESCE($5, CURRENT_TIMESTAMP)
WHERE id = $1 AND workspace_id = $2
RETURNING *;

-- name: UpdatePaymentGasSponsorship :one
UPDATE payments
SET 
    gas_sponsored = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2
RETURNING *;