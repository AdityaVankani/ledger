-- name: ListBalances :many
WITH currencies AS (
    SELECT currency FROM expenses WHERE group_id = $1 AND deleted_at IS NULL
    UNION
    SELECT currency FROM settlements WHERE group_id = $1 AND deleted_at IS NULL
), movements AS (
    SELECT paid_by_user_id AS user_id, currency, amount_cents AS delta
    FROM expenses WHERE group_id = $1 AND deleted_at IS NULL
    UNION ALL
    SELECT es.user_id, e.currency, -es.amount_cents AS delta
    FROM expense_splits es
    JOIN expenses e ON e.id = es.expense_id
    WHERE es.group_id = $1 AND e.deleted_at IS NULL
    UNION ALL
    SELECT paid_by_user_id AS user_id, currency, amount_cents AS delta
    FROM settlements WHERE group_id = $1 AND deleted_at IS NULL
    UNION ALL
    SELECT received_by_user_id AS user_id, currency, -amount_cents AS delta
    FROM settlements WHERE group_id = $1 AND deleted_at IS NULL
)
SELECT c.currency, u.id::text, u.email, u.display_name, u.upi_id,
       COALESCE(SUM(m.delta), 0)::bigint AS balance_cents
FROM currencies c
CROSS JOIN group_members gm
JOIN users u ON u.id = gm.user_id
LEFT JOIN movements m ON m.currency = c.currency AND m.user_id = gm.user_id
WHERE gm.group_id = $1
GROUP BY c.currency, u.id, u.email, u.display_name, u.upi_id
ORDER BY c.currency, u.display_name, u.id;