-- name: CreateTaskerAvailability :one
INSERT INTO tasker_availability (tasker_id, day_of_week, start_time, end_time)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: DeleteTaskerAvailability :exec
DELETE FROM tasker_availability
WHERE tasker_id = $1;

-- name: ListTaskerAvailability :many
SELECT * FROM tasker_availability
WHERE tasker_id = $1
ORDER BY day_of_week ASC, start_time ASC;

-- name: ListTaskerAvailabilityByTaskerIDs :many
SELECT * FROM tasker_availability
WHERE tasker_id = ANY(sqlc.arg(tasker_ids)::bigint[])
ORDER BY tasker_id, day_of_week ASC, start_time ASC;
