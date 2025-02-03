-- name: GetWorkspace :one
SELECT * FROM workspaces
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: ListWorkspacesByAccountID :many
SELECT * FROM workspaces
WHERE account_id = $1 
AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListWorkspaces :many
SELECT * FROM workspaces
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetAllWorkspaces :many
SELECT * FROM workspaces
ORDER BY created_at DESC;

-- name: CreateWorkspace :one
INSERT INTO workspaces (
    account_id,
    name,
    description,
    business_name,
    business_type,
    website_url,
    support_email,
    support_phone,
    metadata,
    livemode
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: UpdateWorkspace :one
UPDATE workspaces
SET
    name = COALESCE($2, name),
    description = COALESCE($3, description),
    business_name = COALESCE($4, business_name),
    business_type = COALESCE($5, business_type),
    website_url = COALESCE($6, website_url),
    support_email = COALESCE($7, support_email),
    support_phone = COALESCE($8, support_phone),
    metadata = COALESCE($9, metadata),
    livemode = COALESCE($10, livemode),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteWorkspace :exec
UPDATE workspaces
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL;

-- name: HardDeleteWorkspace :exec
DELETE FROM workspaces
WHERE id = $1;

-- name: ListWorkspaceCustomers :many
SELECT c.* FROM customers c
INNER JOIN workspaces w ON c.workspace_id = w.id
WHERE w.id = $1 AND c.deleted_at IS NULL
ORDER BY c.created_at DESC;

-- name: GetAccountByWorkspaceID :one
SELECT a.* FROM accounts a
JOIN workspaces w ON w.account_id = a.id
WHERE w.id = $1 
AND w.deleted_at IS NULL 
AND a.deleted_at IS NULL
LIMIT 1;

-- name: CountWorkspaceCustomers :one
SELECT COUNT(*) FROM customers
WHERE workspace_id = $1 AND deleted_at IS NULL;

-- name: ListWorkspaceCustomersWithPagination :many
SELECT 
    c.*
FROM customers c
JOIN workspaces w ON c.workspace_id = w.id
WHERE w.id = $1 AND c.deleted_at IS NULL
ORDER BY c.created_at DESC
LIMIT $2 OFFSET $3;