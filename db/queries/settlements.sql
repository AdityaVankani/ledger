-- name: CreateSettlement :one
INSERT INTO settlements (
    group_id, paid_by_user_id, received_by_user_id, amount_cents,
    currency, settled_at, note
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id::text, created_at;

-- name: DeleteSettlement :one
UPDATE settlements
SET deleted_at = now(), deleted_by_user_id = $1
WHERE id = $2 AND group_id = $3 AND paid_by_user_id = $1 AND deleted_at IS NULL
RETURNING id::text;