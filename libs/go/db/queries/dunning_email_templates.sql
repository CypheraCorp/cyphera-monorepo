-- name: CreateDunningEmailTemplate :one
INSERT INTO dunning_email_templates (
    workspace_id,
    name,
    template_type,
    subject,
    body_html,
    body_text,
    available_variables,
    is_active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetDunningEmailTemplate :one
SELECT * FROM dunning_email_templates
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetDunningEmailTemplateByType :one
SELECT * FROM dunning_email_templates
WHERE workspace_id = $1
    AND template_type = $2
    AND is_active = true
    AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: ListDunningEmailTemplates :many
SELECT * FROM dunning_email_templates
WHERE workspace_id = $1
    AND deleted_at IS NULL
ORDER BY template_type, name;

-- name: ListActiveDunningEmailTemplates :many
SELECT * FROM dunning_email_templates
WHERE workspace_id = $1
    AND is_active = true
    AND deleted_at IS NULL
ORDER BY template_type, name;

-- name: UpdateDunningEmailTemplate :one
UPDATE dunning_email_templates
SET 
    name = COALESCE($2, name),
    subject = COALESCE($3, subject),
    body_html = COALESCE($4, body_html),
    body_text = COALESCE($5, body_text),
    available_variables = COALESCE($6, available_variables),
    is_active = COALESCE($7, is_active),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteDunningEmailTemplate :one
UPDATE dunning_email_templates
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeactivateTemplatesByType :exec
UPDATE dunning_email_templates
SET is_active = false
WHERE workspace_id = $1 
    AND template_type = $2 
    AND id != $3
    AND deleted_at IS NULL;