-- name: ListExpenses :many
WITH expense_page AS (
    SELECT e.id, e.description, e.amount_cents, e.currency, e.expense_date,
           e.paid_by_user_id, e.created_at
    FROM expenses e
    WHERE e.group_id = $1 AND e.deleted_at IS NULL
    ORDER BY e.expense_date DESC, e.created_at DESC
    LIMIT $2
)
SELECT ep.id::text, ep.description, ep.amount_cents, ep.currency,
       ep.expense_date::text, ep.created_at,
       payer.id::text, payer.email, payer.display_name,
       split_user.id::text, split_user.email, split_user.display_name,
       es.amount_cents
FROM expense_page ep
JOIN users payer ON payer.id = ep.paid_by_user_id
JOIN expense_splits es ON es.expense_id = ep.id
JOIN users split_user ON split_user.id = es.user_id
ORDER BY ep.expense_date DESC, ep.created_at DESC, es.user_id;

-- name: CreateExpense :one
INSERT INTO expenses (
    group_id, paid_by_user_id, created_by_user_id, description,
    amount_cents, currency, expense_date
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id::text, created_at;

-- name: CreateExpenseSplit :exec
INSERT INTO expense_splits (expense_id, group_id, user_id, amount_cents)
VALUES ($1, $2, $3, $4);

-- name: DeleteExpense :one
UPDATE expenses
SET deleted_at = now(), deleted_by_user_id = $1
WHERE id = $2 AND group_id = $3 AND created_by_user_id = $1 AND deleted_at IS NULL
RETURNING id::text;