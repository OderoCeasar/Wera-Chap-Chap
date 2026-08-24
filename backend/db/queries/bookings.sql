-- name: CreateBooking :one
INSERT INTO bookings (task_id, client_id, tasker_id, agreed_rate, status)
VALUES ($1, $2, $3, $4, 'confirmed')
RETURNING *;

-- name: GetBooking :one
SELECT * FROM bookings
WHERE id = $1 LIMIT 1;

-- name: GetBookingWithRelations :one
-- users is joined twice: once as the client who posted the task, once as the
-- account behind the tasker profile. The aliases are what the assembled
-- response's "client" and "tasker.user" fields are built from.
SELECT
  sqlc.embed(bookings),
  sqlc.embed(tasks),
  sqlc.embed(categories),
  sqlc.embed(tasker_profiles),
  sqlc.embed(client),
  sqlc.embed(tasker_user)
FROM bookings
JOIN tasks ON tasks.id = bookings.task_id
JOIN categories ON categories.id = tasks.category_id
JOIN tasker_profiles ON tasker_profiles.id = bookings.tasker_id
JOIN users AS client ON client.id = bookings.client_id
JOIN users AS tasker_user ON tasker_user.id = tasker_profiles.user_id
WHERE bookings.id = $1 LIMIT 1;

-- name: ListBookingsForUser :many
-- Everything the caller takes part in, on either side. tasker_id is NULL for a
-- caller with no tasker profile, which drops that half of the predicate.
SELECT
  sqlc.embed(bookings),
  sqlc.embed(tasks),
  sqlc.embed(categories),
  sqlc.embed(tasker_profiles),
  sqlc.embed(client),
  sqlc.embed(tasker_user)
FROM bookings
JOIN tasks ON tasks.id = bookings.task_id
JOIN categories ON categories.id = tasks.category_id
JOIN tasker_profiles ON tasker_profiles.id = bookings.tasker_id
JOIN users AS client ON client.id = bookings.client_id
JOIN users AS tasker_user ON tasker_user.id = tasker_profiles.user_id
WHERE bookings.client_id = sqlc.arg(client_id)
   OR (sqlc.narg(tasker_id)::bigint IS NOT NULL AND bookings.tasker_id = sqlc.narg(tasker_id))
ORDER BY bookings.created_at DESC;

-- name: ListBookingsByTasker :many
SELECT
  sqlc.embed(bookings),
  sqlc.embed(tasks),
  sqlc.embed(categories),
  sqlc.embed(tasker_profiles),
  sqlc.embed(client),
  sqlc.embed(tasker_user)
FROM bookings
JOIN tasks ON tasks.id = bookings.task_id
JOIN categories ON categories.id = tasks.category_id
JOIN tasker_profiles ON tasker_profiles.id = bookings.tasker_id
JOIN users AS client ON client.id = bookings.client_id
JOIN users AS tasker_user ON tasker_user.id = tasker_profiles.user_id
WHERE bookings.tasker_id = $1
ORDER BY bookings.created_at DESC;

-- name: UpdateBookingStatus :exec
-- started_at/completed_at are passed as NULL for the transitions that do not
-- set them, and COALESCE keeps whatever an earlier transition already stamped.
UPDATE bookings
SET
  status = sqlc.arg(status),
  started_at = COALESCE(sqlc.narg(started_at), started_at),
  completed_at = COALESCE(sqlc.narg(completed_at), completed_at)
WHERE id = sqlc.arg(id);
