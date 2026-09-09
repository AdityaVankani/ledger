-- name: DeleteSession :exec
DELETE FROM auth_sessions
WHERE user_id = $1 AND token_hash = $2;