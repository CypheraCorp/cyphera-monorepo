-- name: CreateCircleUser :one
INSERT INTO circle_users (
    id,
    workspace_id,
    circle_create_date,
    pin_status,
    status,
    security_question_status
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetCircleUserByID :one
SELECT * FROM circle_users
WHERE id = $1;

-- name: GetCircleUserByWorkspaceID :one
SELECT * FROM circle_users
WHERE workspace_id = $1;

-- name: ListCircleUsers :many
SELECT * FROM circle_users
ORDER BY created_at DESC;

-- name: UpdateCircleUser :one
UPDATE circle_users
SET 
    pin_status = COALESCE($1, pin_status),
    status = COALESCE($2, status),
    security_question_status = COALESCE($3, security_question_status),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $4
RETURNING *;

-- name: UpdateCircleUserByWorkspaceID :one
UPDATE circle_users
SET 
    pin_status = COALESCE($1, pin_status),
    status = COALESCE($2, status),
    security_question_status = COALESCE($3, security_question_status),
    updated_at = CURRENT_TIMESTAMP
WHERE workspace_id = $4
RETURNING *;

-- name: DeleteCircleUser :exec
DELETE FROM circle_users
WHERE id = $1;

-- name: DeleteCircleUserByWorkspaceID :exec
DELETE FROM circle_users
WHERE workspace_id = $1;

-- name: GetCircleUserWithWallets :one
SELECT 
    cu.*,
    COUNT(cw.id) as wallet_count
FROM 
    circle_users cu
LEFT JOIN 
    circle_wallets cw ON cu.id = cw.circle_user_id
WHERE 
    cu.id = $1
GROUP BY 
    cu.id;

-- name: GetCircleUserWithWalletsByWorkspaceID :one
SELECT 
    cu.*,
    COUNT(cw.id) as wallet_count
FROM 
    circle_users cu
LEFT JOIN 
    circle_wallets cw ON cu.id = cw.circle_user_id
WHERE 
    cu.workspace_id = $1
GROUP BY 
    cu.id; 