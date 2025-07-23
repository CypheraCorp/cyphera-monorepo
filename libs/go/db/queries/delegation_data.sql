-- name: GetDelegationData :one
SELECT * FROM delegation_data
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetDelegationDataBySignature :one
SELECT * FROM delegation_data
WHERE signature = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetDelegationsByDelegator :many
SELECT * FROM delegation_data
WHERE delegator = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetDelegationsByDelegate :many
SELECT * FROM delegation_data
WHERE delegate = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: CreateDelegationData :one
INSERT INTO delegation_data (
    delegate,
    delegator,
    authority,
    caveats,
    salt,
    signature
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: UpdateDelegationData :one
UPDATE delegation_data
SET
    delegate = COALESCE($2, delegate),
    delegator = COALESCE($3, delegator),
    authority = COALESCE($4, authority),
    caveats = COALESCE($5, caveats),
    salt = COALESCE($6, salt),
    signature = COALESCE($7, signature),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteDelegationData :exec
UPDATE delegation_data
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL;

-- name: CountDelegations :one
SELECT COUNT(*) FROM delegation_data
WHERE deleted_at IS NULL;

-- name: CountDelegationsByDelegator :one
SELECT COUNT(*) FROM delegation_data
WHERE delegator = $1 AND deleted_at IS NULL;

-- name: ListDelegationsWithPagination :many
SELECT * FROM delegation_data
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2; 