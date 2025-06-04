-- Workspace Payment Configuration Queries

-- name: CreateWorkspacePaymentConfiguration :one
INSERT INTO workspace_payment_configurations (
    workspace_id,
    provider_name,
    is_active,
    is_test_mode,
    configuration,
    webhook_endpoint_url,
    webhook_secret_key,
    connected_account_id,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetWorkspacePaymentConfiguration :one
SELECT * FROM workspace_payment_configurations 
WHERE workspace_id = $1 AND provider_name = $2 AND is_active = true AND deleted_at IS NULL;

-- name: GetWorkspacePaymentConfigurationByID :one
SELECT * FROM workspace_payment_configurations 
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL;

-- name: ListWorkspacePaymentConfigurations :many
SELECT * FROM workspace_payment_configurations 
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY provider_name, created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListActiveWorkspacePaymentConfigurations :many
SELECT * FROM workspace_payment_configurations 
WHERE workspace_id = $1 AND is_active = true AND deleted_at IS NULL
ORDER BY provider_name;

-- name: UpdateWorkspacePaymentConfiguration :one
UPDATE workspace_payment_configurations 
SET 
    is_active = $3,
    is_test_mode = $4,
    configuration = $5,
    webhook_endpoint_url = $6,
    webhook_secret_key = $7,
    connected_account_id = $8,
    metadata = $9,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateWorkspacePaymentConfigurationLastSync :one
UPDATE workspace_payment_configurations 
SET 
    last_sync_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateWorkspacePaymentConfigurationLastWebhook :one
UPDATE workspace_payment_configurations 
SET 
    last_webhook_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: DeactivateWorkspacePaymentConfiguration :one
UPDATE workspace_payment_configurations 
SET 
    is_active = false,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteWorkspacePaymentConfiguration :one
UPDATE workspace_payment_configurations 
SET 
    deleted_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: GetWorkspacePaymentConfigurationByWebhookURL :one
SELECT * FROM workspace_payment_configurations 
WHERE webhook_endpoint_url = $1 AND is_active = true AND deleted_at IS NULL;

-- name: ListWorkspacePaymentConfigurationsByProvider :many
SELECT * FROM workspace_payment_configurations 
WHERE provider_name = $1 AND is_active = true AND deleted_at IS NULL
ORDER BY workspace_id;

-- name: CountWorkspacePaymentConfigurations :one
SELECT COUNT(*) FROM workspace_payment_configurations 
WHERE workspace_id = $1 AND deleted_at IS NULL;

-- name: GetWorkspacePaymentConfigurationByConnectedAccount :one
SELECT * FROM workspace_payment_configurations 
WHERE connected_account_id = $1 AND provider_name = $2 AND is_active = true AND deleted_at IS NULL;

-- Validation and utility queries

-- name: CheckWorkspaceHasPaymentProvider :one
SELECT EXISTS(
    SELECT 1 FROM workspace_payment_configurations 
    WHERE workspace_id = $1 AND provider_name = $2 AND is_active = true AND deleted_at IS NULL
);

-- name: GetWorkspaceActiveProviders :many
SELECT DISTINCT provider_name FROM workspace_payment_configurations 
WHERE workspace_id = $1 AND is_active = true AND deleted_at IS NULL;

-- name: UpdateWorkspacePaymentConfigurationConfig :one
UPDATE workspace_payment_configurations 
SET 
    configuration = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *; 