-- name: CreateInvoiceLineItem :one
INSERT INTO invoice_line_items (
    invoice_id,
    description,
    quantity,
    unit_amount_in_cents,
    amount_in_cents,
    fiat_currency,
    subscription_id,
    product_id,
    network_id,
    token_id,
    crypto_amount,
    exchange_rate,
    tax_rate,
    tax_amount_in_cents,
    tax_crypto_amount,
    period_start,
    period_end,
    line_item_type,
    gas_fee_payment_id,
    is_gas_sponsored,
    gas_sponsor_type,
    gas_sponsor_name,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
)
RETURNING *;

-- name: GetInvoiceLineItem :one
SELECT * FROM invoice_line_items
WHERE id = $1;

-- name: GetInvoiceLineItems :many
SELECT * FROM invoice_line_items
WHERE invoice_id = $1
ORDER BY created_at ASC;

-- name: GetInvoiceLineItemsByType :many
SELECT * FROM invoice_line_items
WHERE invoice_id = $1 AND line_item_type = $2
ORDER BY created_at ASC;

-- name: UpdateInvoiceLineItem :one
UPDATE invoice_line_items
SET 
    description = $2,
    quantity = $3,
    unit_amount_in_cents = $4,
    amount_in_cents = $5,
    tax_rate = $6,
    tax_amount_in_cents = $7,
    tax_crypto_amount = $8,
    metadata = $9,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteInvoiceLineItem :exec
DELETE FROM invoice_line_items
WHERE id = $1;

-- name: DeleteInvoiceLineItems :exec
DELETE FROM invoice_line_items
WHERE invoice_id = $1;

-- name: GetInvoiceSubtotal :one
SELECT 
    SUM(amount_in_cents) FILTER (WHERE line_item_type = 'product') as product_subtotal,
    SUM(amount_in_cents) FILTER (WHERE line_item_type = 'gas_fee' AND NOT is_gas_sponsored) as customer_gas_fees,
    SUM(amount_in_cents) FILTER (WHERE line_item_type = 'gas_fee' AND is_gas_sponsored) as sponsored_gas_fees,
    SUM(tax_amount_in_cents) as total_tax,
    SUM(amount_in_cents) FILTER (WHERE line_item_type = 'discount') as total_discount,
    SUM(amount_in_cents) - COALESCE(SUM(amount_in_cents) FILTER (WHERE is_gas_sponsored), 0) as customer_total
FROM invoice_line_items
WHERE invoice_id = $1;

-- name: GetInvoiceCryptoAmounts :many
SELECT 
    token_id,
    network_id,
    SUM(crypto_amount) as total_crypto_amount,
    SUM(tax_crypto_amount) as total_tax_crypto_amount
FROM invoice_line_items
WHERE invoice_id = $1
    AND token_id IS NOT NULL
GROUP BY token_id, network_id;

-- name: CreateInvoiceLineItemBatch :copyfrom
INSERT INTO invoice_line_items (
    invoice_id,
    description,
    quantity,
    unit_amount_in_cents,
    amount_in_cents,
    fiat_currency,
    product_id,
    line_item_type
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
);

-- name: GetProductLineItemsByInvoice :many
SELECT 
    ili.*,
    p.name as product_name,
    p.interval_type,
    p.term_length
FROM invoice_line_items ili
LEFT JOIN products p ON ili.product_id = p.id
WHERE ili.invoice_id = $1
    AND ili.line_item_type = 'product'
ORDER BY ili.created_at ASC;

-- name: GetGasLineItemsByInvoice :many
SELECT 
    ili.*,
    gfp.gas_fee_wei,
    gfp.gas_price_gwei,
    gfp.gas_units_used,
    n.name as network_name
FROM invoice_line_items ili
LEFT JOIN gas_fee_payments gfp ON ili.gas_fee_payment_id = gfp.id
LEFT JOIN networks n ON ili.network_id = n.id
WHERE ili.invoice_id = $1
    AND ili.line_item_type = 'gas_fee'
ORDER BY ili.created_at ASC;

-- name: UpdateLineItemGasSponsorship :one
UPDATE invoice_line_items
SET 
    is_gas_sponsored = $2,
    gas_sponsor_type = $3,
    gas_sponsor_name = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: GetLineItemsByProduct :many
SELECT * FROM invoice_line_items
WHERE product_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetLineItemsByCurrency :many
SELECT * FROM invoice_line_items
WHERE invoice_id = $1 AND fiat_currency = $2
ORDER BY created_at ASC;

-- name: CreateInvoiceLineItemFromSubscription :one
INSERT INTO invoice_line_items (
    invoice_id,
    subscription_id,
    product_id,
    description,
    quantity,
    unit_amount_in_cents,
    amount_in_cents,
    fiat_currency,
    line_item_type,
    period_start,
    period_end,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING *;

-- name: BulkCreateInvoiceLineItemsFromSubscription :copyfrom
INSERT INTO invoice_line_items (
    invoice_id,
    subscription_id,
    product_id,
    description,
    quantity,
    unit_amount_in_cents,
    amount_in_cents,
    fiat_currency,
    line_item_type,
    period_start,
    period_end
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
);

-- name: GetInvoiceLineItemsBySubscription :many
SELECT * FROM invoice_line_items
WHERE subscription_id = $1 AND invoice_id = $2
ORDER BY 
    CASE 
        WHEN line_item_type = 'product' THEN 1
        WHEN line_item_type = 'gas_fee' THEN 2
        WHEN line_item_type = 'discount' THEN 3
        WHEN line_item_type = 'tax' THEN 4
        ELSE 5
    END,
    created_at ASC;