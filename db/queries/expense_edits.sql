-- name: UpdateExpense :one
UPDATE expenses
SET paid_by_user_id = $2,
    description = $3,
    amount_cents = $4,
    currency = $5,
    expense_date = $6,
    updated_at = now()
WHERE id = $1 AND group_id = $7 AND created_by_user_id = $8 AND deleted_at IS NULL
RETURNING id::text, created_at;

-- name: DeleteExpenseSplits :exec
DELETE FROM expense_splits WHERE expense_id = $1;