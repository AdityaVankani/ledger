SELECT u.id::text, u.email, u.display_name, u.upi_id, gm.joined_at
-- name: ListGroupMembers :many
SELECT u.id::text, u.email, u.display_name, gm.joined_at
FROM group_members gm
JOIN users u ON u.id = gm.user_id
WHERE gm.group_id = $1
ORDER BY gm.joined_at ASC;

-- name: AddGroupMemberReturning :one
INSERT INTO group_members (group_id, user_id)
VALUES ($1, $2)
RETURNING joined_at;

-- name: IsGroupMember :one
SELECT EXISTS (SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2);