-- name: CreateCircleUser :one
INSERT INTO circle_users (
    id,
    account_id,
    token,
    encryption_key
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetCircleUserByID :one
SELECT * FROM circle_users
WHERE id = $1;

-- name: GetCircleUserByAccountID :one
SELECT * FROM circle_users
WHERE account_id = $1;

-- name: ListCircleUsers :many
SELECT * FROM circle_users
ORDER BY created_at DESC;

-- name: UpdateCircleUser :one
UPDATE circle_users
SET 
    token = COALESCE($1, token),
    encryption_key = COALESCE($2, encryption_key),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $3
RETURNING *;

-- name: UpdateCircleUserByAccountID :one
UPDATE circle_users
SET 
    token = COALESCE($1, token),
    encryption_key = COALESCE($2, encryption_key),
    updated_at = CURRENT_TIMESTAMP
WHERE account_id = $3
RETURNING *;

-- name: DeleteCircleUser :exec
DELETE FROM circle_users
WHERE id = $1;

-- name: DeleteCircleUserByAccountID :exec
DELETE FROM circle_users
WHERE account_id = $1;

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

-- name: GetCircleUserWithWalletsByAccountID :one
SELECT 
    cu.*,
    COUNT(cw.id) as wallet_count
FROM 
    circle_users cu
LEFT JOIN 
    circle_wallets cw ON cu.id = cw.circle_user_id
WHERE 
    cu.account_id = $1
GROUP BY 
    cu.id; 