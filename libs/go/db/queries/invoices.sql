-- name: CreateInvoice :one
INSERT INTO invoices (
    workspace_id,
    customer_id,
    subscription_id,
    external_id,
    external_customer_id,
    external_subscription_id,
    status,
    collection_method,
    amount_due,
    amount_paid,
    amount_remaining,
    currency,
    due_date,
    paid_at,
    created_date,
    invoice_pdf,
    hosted_invoice_url,
    charge_id,
    payment_intent_id,
    line_items,
    tax_amount,
    total_tax_amounts,
    billing_reason,
    paid_out_of_band,
    payment_provider,
    payment_sync_status,
    payment_synced_at,
    attempt_count,
    next_payment_attempt,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
    $21, $22, $23, $24, $25, $26, $27, $28, $29, $30
) RETURNING *;

-- name: GetInvoiceByID :one
SELECT * FROM invoices 
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL;

-- name: GetInvoiceByExternalID :one
SELECT * FROM invoices 
WHERE external_id = $1 AND workspace_id = $2 AND payment_provider = $3 AND deleted_at IS NULL;

-- name: UpdateInvoice :one
UPDATE invoices SET
    customer_id = $3,
    subscription_id = $4,
    status = $5,
    collection_method = $6,
    amount_due = $7,
    amount_paid = $8,
    amount_remaining = $9,
    currency = $10,
    due_date = $11,
    paid_at = $12,
    invoice_pdf = $13,
    hosted_invoice_url = $14,
    charge_id = $15,
    payment_intent_id = $16,
    line_items = $17,
    tax_amount = $18,
    total_tax_amounts = $19,
    billing_reason = $20,
    paid_out_of_band = $21,
    attempt_count = $22,
    next_payment_attempt = $23,
    metadata = $24,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateInvoiceSyncStatus :one
UPDATE invoices SET
    payment_sync_status = $3,
    payment_synced_at = CASE 
        WHEN $3 = 'synced' THEN CURRENT_TIMESTAMP 
        ELSE payment_synced_at 
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: UpsertInvoice :one
INSERT INTO invoices (
    workspace_id,
    customer_id,
    subscription_id,
    external_id,
    external_customer_id,
    external_subscription_id,
    status,
    collection_method,
    amount_due,
    amount_paid,
    amount_remaining,
    currency,
    due_date,
    paid_at,
    created_date,
    invoice_pdf,
    hosted_invoice_url,
    charge_id,
    payment_intent_id,
    line_items,
    tax_amount,
    total_tax_amounts,
    billing_reason,
    paid_out_of_band,
    payment_provider,
    payment_sync_status,
    payment_synced_at,
    attempt_count,
    next_payment_attempt,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
    $21, $22, $23, $24, $25, $26, $27, $28, $29, $30
)
ON CONFLICT (workspace_id, external_id, payment_provider)
DO UPDATE SET
    customer_id = EXCLUDED.customer_id,
    subscription_id = EXCLUDED.subscription_id,
    status = EXCLUDED.status,
    collection_method = EXCLUDED.collection_method,
    amount_due = EXCLUDED.amount_due,
    amount_paid = EXCLUDED.amount_paid,
    amount_remaining = EXCLUDED.amount_remaining,
    currency = EXCLUDED.currency,
    due_date = EXCLUDED.due_date,
    paid_at = EXCLUDED.paid_at,
    created_date = EXCLUDED.created_date,
    invoice_pdf = EXCLUDED.invoice_pdf,
    hosted_invoice_url = EXCLUDED.hosted_invoice_url,
    charge_id = EXCLUDED.charge_id,
    payment_intent_id = EXCLUDED.payment_intent_id,
    line_items = EXCLUDED.line_items,
    tax_amount = EXCLUDED.tax_amount,
    total_tax_amounts = EXCLUDED.total_tax_amounts,
    billing_reason = EXCLUDED.billing_reason,
    paid_out_of_band = EXCLUDED.paid_out_of_band,
    payment_sync_status = EXCLUDED.payment_sync_status,
    payment_synced_at = EXCLUDED.payment_synced_at,
    attempt_count = EXCLUDED.attempt_count,
    next_payment_attempt = EXCLUDED.next_payment_attempt,
    metadata = EXCLUDED.metadata,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: DeleteInvoice :exec
UPDATE invoices SET 
    deleted_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL;

-- name: ListInvoicesByWorkspace :many
SELECT * FROM invoices 
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY created_date DESC
LIMIT $2 OFFSET $3;

-- name: ListInvoicesByCustomer :many
SELECT * FROM invoices 
WHERE workspace_id = $1 AND customer_id = $2 AND deleted_at IS NULL
ORDER BY created_date DESC
LIMIT $3 OFFSET $4;

-- name: ListInvoicesBySubscription :many
SELECT * FROM invoices 
WHERE workspace_id = $1 AND subscription_id = $2 AND deleted_at IS NULL
ORDER BY created_date DESC
LIMIT $3 OFFSET $4;

-- name: ListInvoicesByStatus :many
SELECT * FROM invoices 
WHERE workspace_id = $1 AND status = $2 AND deleted_at IS NULL
ORDER BY created_date DESC
LIMIT $3 OFFSET $4;

-- name: ListInvoicesByProvider :many
SELECT * FROM invoices 
WHERE workspace_id = $1 AND payment_provider = $2 AND deleted_at IS NULL
ORDER BY created_date DESC
LIMIT $3 OFFSET $4;

-- name: ListInvoicesBySyncStatus :many
SELECT * FROM invoices 
WHERE workspace_id = $1 AND payment_sync_status = $2 AND deleted_at IS NULL
ORDER BY created_date DESC
LIMIT $3 OFFSET $4;

-- name: CountInvoicesByWorkspace :one
SELECT COUNT(*) FROM invoices 
WHERE workspace_id = $1 AND deleted_at IS NULL;

-- name: CountInvoicesByStatus :one
SELECT COUNT(*) FROM invoices 
WHERE workspace_id = $1 AND status = $2 AND deleted_at IS NULL;

-- name: CountInvoicesByProvider :one
SELECT COUNT(*) FROM invoices 
WHERE workspace_id = $1 AND payment_provider = $2 AND deleted_at IS NULL;

-- name: GetInvoicesByExternalCustomerID :many
SELECT * FROM invoices 
WHERE workspace_id = $1 AND external_customer_id = $2 AND payment_provider = $3 AND deleted_at IS NULL
ORDER BY created_date DESC;

-- name: GetInvoicesByExternalSubscriptionID :many
SELECT * FROM invoices 
WHERE workspace_id = $1 AND external_subscription_id = $2 AND payment_provider = $3 AND deleted_at IS NULL
ORDER BY created_date DESC;

-- name: GetOverdueInvoices :many
SELECT * FROM invoices 
WHERE workspace_id = $1 
    AND status IN ('open') 
    AND due_date < CURRENT_TIMESTAMP 
    AND deleted_at IS NULL
ORDER BY due_date ASC
LIMIT $2 OFFSET $3;

-- name: GetUnpaidInvoices :many
SELECT * FROM invoices 
WHERE workspace_id = $1 
    AND status IN ('open', 'draft') 
    AND amount_remaining > 0 
    AND deleted_at IS NULL
ORDER BY created_date DESC
LIMIT $2 OFFSET $3;

-- name: GetRecentInvoices :many
SELECT * FROM invoices 
WHERE workspace_id = $1 
    AND created_date >= $2 
    AND deleted_at IS NULL
ORDER BY created_date DESC
LIMIT $3 OFFSET $4;

-- name: BulkUpdateInvoiceSyncStatus :exec
UPDATE invoices SET
    payment_sync_status = $2,
    payment_synced_at = CASE 
        WHEN $2 = 'synced' THEN CURRENT_TIMESTAMP 
        ELSE payment_synced_at 
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE workspace_id = $1 
    AND payment_provider = $3 
    AND external_id = ANY($4::text[]) 
    AND deleted_at IS NULL;

-- name: CreateInvoiceWithDetails :one
INSERT INTO invoices (
    workspace_id,
    customer_id,
    subscription_id,
    invoice_number,
    status,
    amount_due,
    currency,
    subtotal_cents,
    discount_cents,
    tax_amount_cents,
    tax_details,
    due_date,
    payment_link_id,
    delegation_address,
    qr_code_data,
    customer_tax_id,
    customer_jurisdiction_id,
    reverse_charge_applies,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
) RETURNING *;

-- name: UpdateInvoiceDetails :one
UPDATE invoices SET
    subtotal_cents = $3,
    discount_cents = $4,
    tax_amount_cents = $5,
    tax_details = $6,
    amount_due = $7,
    customer_tax_id = $8,
    customer_jurisdiction_id = $9,
    reverse_charge_applies = $10,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: GetInvoiceWithLineItems :one
SELECT 
    i.*,
    COALESCE(
        (SELECT json_agg(ili.* ORDER BY ili.created_at)
         FROM invoice_line_items ili
         WHERE ili.invoice_id = i.id),
        '[]'::json
    ) as line_items_detail
FROM invoices i
WHERE i.id = $1 AND i.workspace_id = $2 AND i.deleted_at IS NULL;

-- name: LinkInvoiceToPaymentLink :one
UPDATE invoices SET
    payment_link_id = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateInvoiceQRCode :one
UPDATE invoices SET
    qr_code_data = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateInvoiceNumber :one
UPDATE invoices SET
    invoice_number = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: GetInvoiceByNumber :one
SELECT * FROM invoices
WHERE workspace_id = $1 AND invoice_number = $2 AND deleted_at IS NULL;

-- name: GetNextInvoiceNumber :one
SELECT 
    COALESCE(MAX(CAST(REGEXP_REPLACE(invoice_number, '[^0-9]', '', 'g') AS INTEGER)), 0) + 1 as next_number
FROM invoices
WHERE workspace_id = $1 
    AND invoice_number ~ '^[A-Z]*[0-9]+$'
    AND deleted_at IS NULL;

-- name: GetInvoicesByPaymentLink :many
SELECT * FROM invoices
WHERE workspace_id = $1 AND payment_link_id = $2 AND deleted_at IS NULL
ORDER BY created_at DESC; 