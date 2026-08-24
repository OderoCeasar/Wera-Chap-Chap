-- name: CreateUser :one
INSERT INTO users (
  email,
  password_hash,
  full_name,
  phone,
  role,
  is_verified,
  background_check_passed
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: UpdateUser :one
-- COALESCE rather than a blanket write: the profile form submits only the
-- fields it shows, and an absent key must leave the column alone instead of
-- blanking it.
UPDATE users
SET
  full_name = COALESCE(sqlc.narg(full_name), full_name),
  phone = COALESCE(sqlc.narg(phone), phone),
  avatar_url = COALESCE(sqlc.narg(avatar_url), avatar_url),
  updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $2, updated_at = now()
WHERE id = $1;
