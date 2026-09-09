-- name: CreateUser :one
INSERT INTO users (email, display_name, password_hash)
VALUES ($1, $2, $3)
RETURNING id::text, email, display_name;

-- name: GetUserByEmail :one
SELECT id::text, email, display_name, password_hash
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id::text, email, display_name
FROM users
WHERE id = $1;

-- name: UpdateUserUPI :one
UPDATE users
SET upi_id = $2, updated_at = now()
WHERE id = $1
RETURNING id::text, email, display_name, upi_id;

-- name: CreateSession :exec
INSERT INTO auth_sessions (user_id, token_hash, expires_at)
VALUES ($1, $2, $3);

-- name: GetUserIDBySessionTokenHash :one
SELECT user_id::text
FROM auth_sessions
WHERE token_hash = $1 AND expires_at > now();