-- name: CreateTask :one
INSERT INTO tasks (
  client_id,
  category_id,
  title,
  description,
  location_address,
  location_lat,
  location_lng,
  budget_type,
  budget_amount,
  scheduled_at,
  status
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'open'
) RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks
WHERE id = $1 LIMIT 1;

-- name: GetTaskWithRelations :one
SELECT sqlc.embed(tasks), sqlc.embed(categories), sqlc.embed(users)
FROM tasks
JOIN categories ON categories.id = tasks.category_id
JOIN users ON users.id = tasks.client_id
WHERE tasks.id = $1 LIMIT 1;

-- name: ListOpenTasks :many
-- The marketplace feed taskers browse. As with ListTaskerProfiles, each filter
-- is skipped when its argument is NULL.
SELECT sqlc.embed(tasks), sqlc.embed(categories), sqlc.embed(users)
FROM tasks
JOIN categories ON categories.id = tasks.category_id
JOIN users ON users.id = tasks.client_id
WHERE tasks.status = 'open'
  AND (sqlc.narg(category_id)::bigint IS NULL OR tasks.category_id = sqlc.narg(category_id))
  AND (sqlc.narg(min_budget)::numeric IS NULL OR tasks.budget_amount >= sqlc.narg(min_budget))
  AND (
    sqlc.narg(search)::text IS NULL
    OR tasks.title ILIKE '%' || sqlc.narg(search) || '%'
    OR tasks.description ILIKE '%' || sqlc.narg(search) || '%'
  )
ORDER BY tasks.created_at DESC;

-- name: ListTasksByClient :many
SELECT sqlc.embed(tasks), sqlc.embed(categories), sqlc.embed(users)
FROM tasks
JOIN categories ON categories.id = tasks.category_id
JOIN users ON users.id = tasks.client_id
WHERE tasks.client_id = $1
ORDER BY tasks.created_at DESC;

-- name: UpdateTask :one
UPDATE tasks
SET
  category_id = $2,
  title = $3,
  description = $4,
  location_address = $5,
  location_lat = $6,
  location_lng = $7,
  budget_type = $8,
  budget_amount = $9,
  scheduled_at = $10,
  updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateTaskStatus :exec
UPDATE tasks
SET status = $2, updated_at = now()
WHERE id = $1;
