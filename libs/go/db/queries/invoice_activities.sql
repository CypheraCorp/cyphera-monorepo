-- name: CreateInvoiceActivity :one
INSERT INTO invoice_activities (
    invoice_id,
    workspace_id,
    activity_type,
    from_status,
    to_status,
    performed_by,
    description,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetInvoiceActivities :many
SELECT * FROM invoice_activities
WHERE invoice_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetInvoiceActivitiesByType :many
SELECT * FROM invoice_activities
WHERE invoice_id = $1 AND activity_type = $2
ORDER BY created_at DESC;

-- name: GetRecentInvoiceActivities :many
SELECT * FROM invoice_activities
WHERE workspace_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: RecordInvoiceStatusChange :one
INSERT INTO invoice_activities (
    invoice_id,
    workspace_id,
    activity_type,
    from_status,
    to_status,
    performed_by,
    description
) VALUES (
    $1, $2, 'status_changed', $3, $4, $5, $6
) RETURNING *;

-- name: RecordInvoiceCreation :one
INSERT INTO invoice_activities (
    invoice_id,
    workspace_id,
    activity_type,
    to_status,
    performed_by,
    description,
    metadata
) VALUES (
    $1, $2, 'created', $3, $4, $5, $6
) RETURNING *;

-- name: RecordInvoiceReminder :one
INSERT INTO invoice_activities (
    invoice_id,
    workspace_id,
    activity_type,
    description,
    metadata
) VALUES (
    $1, $2, 'reminder_sent', $3, $4
) RETURNING *;

-- name: GetInvoiceActivityCount :one
SELECT COUNT(*) FROM invoice_activities
WHERE invoice_id = $1;

-- name: GetInvoiceStatusHistory :many
SELECT 
    activity_type,
    from_status,
    to_status,
    performed_by,
    description,
    created_at
FROM invoice_activities
WHERE invoice_id = $1 
    AND activity_type IN ('created', 'status_changed')
ORDER BY created_at ASC;