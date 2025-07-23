-- workspace_provider_accounts.sql
-- SQLC queries for generic provider account mapping (Stripe, Chargebee, PayPal, etc.)

-- name: CreateWorkspaceProviderAccount :one
INSERT INTO workspace_provider_accounts (
    workspace_id, 
    provider_name,
    provider_account_id, 
    account_type, 
    is_active,
    environment,
    display_name,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetWorkspaceByProviderAccount :one
-- Critical query for webhook routing - maps provider account ID to workspace
SELECT 
    workspace_id, 
    provider_name,
    account_type,
    environment,
    display_name
FROM workspace_provider_accounts
WHERE provider_name = $1 
  AND provider_account_id = $2 
  AND environment = $3
  AND is_active = true
  AND deleted_at IS NULL;

-- name: GetProviderAccountByWorkspace :one
-- Get a specific provider account for a workspace
SELECT * FROM workspace_provider_accounts
WHERE workspace_id = $1 
  AND provider_name = $2
  AND is_active = true
  AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: ListProviderAccountsByWorkspace :many
-- List all provider accounts for a workspace
SELECT * FROM workspace_provider_accounts
WHERE workspace_id = $1 
  AND is_active = true
  AND deleted_at IS NULL
ORDER BY provider_name, created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListProviderAccountsByProvider :many
-- List all accounts for a specific provider across workspaces
SELECT * FROM workspace_provider_accounts
WHERE provider_name = $1
  AND is_active = true
  AND deleted_at IS NULL
ORDER BY workspace_id, created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateWorkspaceProviderAccount :one
UPDATE workspace_provider_accounts
SET 
    account_type = COALESCE(sqlc.narg('account_type'), account_type),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    environment = COALESCE(sqlc.narg('environment'), environment),
    display_name = COALESCE(sqlc.narg('display_name'), display_name),
    metadata = COALESCE(sqlc.narg('metadata'), metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 
  AND workspace_id = $2
  AND deleted_at IS NULL
RETURNING *;

-- name: DeactivateWorkspaceProviderAccount :one
UPDATE workspace_provider_accounts
SET 
    is_active = false,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 
  AND workspace_id = $2
  AND deleted_at IS NULL
RETURNING *;

-- name: DeleteWorkspaceProviderAccount :exec
UPDATE workspace_provider_accounts
SET 
    deleted_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 
  AND workspace_id = $2
  AND deleted_at IS NULL;

-- name: GetProviderAccountByID :one
SELECT * FROM workspace_provider_accounts
WHERE id = $1 
  AND workspace_id = $2
  AND deleted_at IS NULL;

-- name: CountProviderAccountsByWorkspace :one
SELECT COUNT(*) FROM workspace_provider_accounts
WHERE workspace_id = $1 
  AND is_active = true
  AND deleted_at IS NULL;

-- name: CountProviderAccountsByProvider :one
SELECT COUNT(*) FROM workspace_provider_accounts
WHERE provider_name = $1
  AND is_active = true
  AND deleted_at IS NULL;

-- name: ListActiveProviders :many
-- Get list of active providers for a workspace
SELECT DISTINCT provider_name FROM workspace_provider_accounts
WHERE workspace_id = $1 
  AND is_active = true
  AND deleted_at IS NULL
ORDER BY provider_name;

-- name: ValidateProviderAccountUnique :one
-- Check if provider account ID already exists (for constraint validation)
SELECT COUNT(*) FROM workspace_provider_accounts
WHERE provider_name = $1 
  AND provider_account_id = $2 
  AND environment = $3
  AND deleted_at IS NULL
  AND id != COALESCE(sqlc.narg('exclude_id'), '00000000-0000-0000-0000-000000000000'::UUID);

-- name: GetWorkspaceProviderAccountForWebhook :one
-- Optimized query for webhook processing - includes all needed data
SELECT 
    wpa.*,
    w.name as workspace_name,
    w.livemode as workspace_livemode
FROM workspace_provider_accounts wpa
JOIN workspaces w ON w.id = wpa.workspace_id
WHERE wpa.provider_name = $1 
  AND wpa.provider_account_id = $2 
  AND wpa.environment = $3
  AND wpa.is_active = true
  AND wpa.deleted_at IS NULL
  AND w.deleted_at IS NULL; 