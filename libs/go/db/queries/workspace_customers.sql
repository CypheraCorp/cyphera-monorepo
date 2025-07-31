-- name: AddCustomerToWorkspace :one
INSERT INTO workspace_customers (
    workspace_id,
    customer_id
) VALUES (
    $1, $2
)
ON CONFLICT (workspace_id, customer_id) 
DO UPDATE SET 
    deleted_at = NULL,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: RemoveCustomerFromWorkspace :exec
UPDATE workspace_customers 
SET deleted_at = CURRENT_TIMESTAMP
WHERE workspace_id = $1 AND customer_id = $2 AND deleted_at IS NULL;

-- name: ListWorkspaceCustomers :many
SELECT 
    c.*
FROM customers c
INNER JOIN workspace_customers wc ON c.id = wc.customer_id
WHERE wc.workspace_id = $1 AND wc.deleted_at IS NULL AND c.deleted_at IS NULL
ORDER BY c.created_at DESC;

-- name: ListWorkspaceCustomersWithPagination :many
SELECT 
    c.*
FROM customers c
INNER JOIN workspace_customers wc ON c.id = wc.customer_id
WHERE wc.workspace_id = $1 AND wc.deleted_at IS NULL AND c.deleted_at IS NULL
ORDER BY c.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountWorkspaceCustomers :one
SELECT COUNT(*)
FROM customers c
INNER JOIN workspace_customers wc ON c.id = wc.customer_id
WHERE wc.workspace_id = $1 AND wc.deleted_at IS NULL AND c.deleted_at IS NULL;

-- name: ListCustomerWorkspaces :many
SELECT 
    w.*
FROM workspaces w
INNER JOIN workspace_customers wc ON w.id = wc.workspace_id
WHERE wc.customer_id = $1 AND wc.deleted_at IS NULL AND w.deleted_at IS NULL
ORDER BY w.created_at DESC;

-- name: GetWorkspaceCustomerAssociation :one
SELECT *
FROM workspace_customers
WHERE workspace_id = $1 AND customer_id = $2 AND deleted_at IS NULL
LIMIT 1;

-- name: IsCustomerInWorkspace :one
SELECT EXISTS(
    SELECT 1 FROM workspace_customers 
    WHERE workspace_id = $1 AND customer_id = $2 AND deleted_at IS NULL
);

-- name: ListWorkspaceCustomersWithInfo :many
SELECT 
    c.*,
    w.name as workspace_name,
    w.business_name as workspace_business_name,
    w.support_email as workspace_support_email,
    wc.created_at as association_created_at
FROM customers c
INNER JOIN workspace_customers wc ON c.id = wc.customer_id
INNER JOIN workspaces w ON wc.workspace_id = w.id
WHERE wc.workspace_id = $1 AND wc.deleted_at IS NULL AND c.deleted_at IS NULL AND w.deleted_at IS NULL
ORDER BY c.created_at DESC;

-- name: ListWorkspaceCustomersWithRevenue :many
SELECT 
    c.*,
    COALESCE(SUM(p.amount_in_cents), 0) as total_revenue
FROM customers c
INNER JOIN workspace_customers wc ON c.id = wc.customer_id
LEFT JOIN payments p ON c.id = p.customer_id 
    AND p.workspace_id = $1 
    AND p.status = 'completed'
WHERE wc.workspace_id = $1 AND wc.deleted_at IS NULL AND c.deleted_at IS NULL
GROUP BY c.id, c.external_id, c.web3auth_id, c.email, c.name, c.phone, c.description, 
         c.metadata, c.finished_onboarding, c.payment_sync_status, c.payment_synced_at, 
         c.payment_sync_version, c.payment_provider, c.tax_jurisdiction_id, c.tax_id, 
         c.tax_id_type, c.tax_id_verified, c.tax_id_verified_at, c.is_business, 
         c.business_name, c.billing_country, c.billing_state, c.billing_city, 
         c.billing_postal_code, c.created_at, c.updated_at, c.deleted_at
ORDER BY c.created_at DESC
LIMIT $2 OFFSET $3; 