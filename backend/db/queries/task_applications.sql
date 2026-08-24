-- name: CreateTaskApplication :one
INSERT INTO task_applications (task_id, tasker_id, proposed_rate, cover_note, status)
VALUES ($1, $2, $3, $4, 'pending')
RETURNING *;

-- name: GetTaskApplication :one
SELECT * FROM task_applications
WHERE id = $1 AND task_id = $2 LIMIT 1;

-- name: CountApplicationsByTaskAndTasker :one
SELECT COUNT(*) FROM task_applications
WHERE task_id = $1 AND tasker_id = $2;

-- name: GetApplicationWithTasker :one
SELECT sqlc.embed(task_applications), sqlc.embed(tasker_profiles), sqlc.embed(users)
FROM task_applications
JOIN tasker_profiles ON tasker_profiles.id = task_applications.tasker_id
JOIN users ON users.id = tasker_profiles.user_id
WHERE task_applications.id = $1 LIMIT 1;

-- name: ListApplicationsByTaskIDs :many
-- Batch form so a page of tasks costs one query for all their applications
-- rather than one per task.
SELECT sqlc.embed(task_applications), sqlc.embed(tasker_profiles), sqlc.embed(users)
FROM task_applications
JOIN tasker_profiles ON tasker_profiles.id = task_applications.tasker_id
JOIN users ON users.id = tasker_profiles.user_id
WHERE task_applications.task_id = ANY(sqlc.arg(task_ids)::bigint[])
ORDER BY task_applications.created_at DESC;

-- name: ListApplicationsByTasker :many
SELECT sqlc.embed(task_applications), sqlc.embed(tasks), sqlc.embed(categories), sqlc.embed(users)
FROM task_applications
JOIN tasks ON tasks.id = task_applications.task_id
JOIN categories ON categories.id = tasks.category_id
JOIN users ON users.id = tasks.client_id
WHERE task_applications.tasker_id = $1
ORDER BY task_applications.created_at DESC;

-- name: UpdateApplicationStatus :exec
UPDATE task_applications
SET status = $2
WHERE id = $1;

-- name: RejectRivalApplications :exec
-- Accepting one application closes the rest for that task in the same
-- transaction, so a client can never book two taskers for one job.
UPDATE task_applications
SET status = 'rejected'
WHERE task_id = $1 AND id <> $2;
