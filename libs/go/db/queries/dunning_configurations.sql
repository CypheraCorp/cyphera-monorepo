-- name: CreateDunningConfiguration :one
INSERT INTO dunning_configurations (
    workspace_id,
    name,
    description,
    is_active,
    is_default,
    max_retry_attempts,
    retry_interval_days,
    attempt_actions,
    final_action,
    final_action_config,
    send_pre_dunning_reminder,
    pre_dunning_days,
    allow_customer_retry,
    grace_period_hours
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
RETURNING *;

-- name: GetDunningConfiguration :one
SELECT * FROM dunning_configurations
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetDefaultDunningConfiguration :one
SELECT * FROM dunning_configurations
WHERE workspace_id = $1 
    AND is_default = true 
    AND deleted_at IS NULL;

-- name: ListDunningConfigurations :many
SELECT * FROM dunning_configurations
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY is_default DESC, name;

-- name: UpdateDunningConfiguration :one
UPDATE dunning_configurations
SET 
    name = COALESCE($2, name),
    description = COALESCE($3, description),
    is_active = COALESCE($4, is_active),
    max_retry_attempts = COALESCE($5, max_retry_attempts),
    retry_interval_days = COALESCE($6, retry_interval_days),
    attempt_actions = COALESCE($7, attempt_actions),
    final_action = COALESCE($8, final_action),
    final_action_config = COALESCE($9, final_action_config),
    send_pre_dunning_reminder = COALESCE($10, send_pre_dunning_reminder),
    pre_dunning_days = COALESCE($11, pre_dunning_days),
    allow_customer_retry = COALESCE($12, allow_customer_retry),
    grace_period_hours = COALESCE($13, grace_period_hours),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SetDefaultDunningConfiguration :exec
UPDATE dunning_configurations
SET is_default = false
WHERE workspace_id = $1 AND id != $2 AND deleted_at IS NULL;

-- name: DeleteDunningConfiguration :one
UPDATE dunning_configurations
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;