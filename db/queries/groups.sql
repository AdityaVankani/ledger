-- name: CreateGroup :one
INSERT INTO groups (name, created_by_user_id)
VALUES ($1, $2)
RETURNING id::text, name, created_at;

-- name: AddGroupMember :exec
INSERT INTO group_members (group_id, user_id)
VALUES ($1, $2);

-- name: ListGroupsForUser :many
SELECT g.id::text, g.name, g.created_at
FROM groups g
JOIN group_members gm ON gm.group_id = g.id
WHERE gm.user_id = $1
ORDER BY g.created_at DESC;

-- name: DeleteGroup :one
DELETE FROM groups
WHERE id = $1
	AND EXISTS (SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)
RETURNING id::text;