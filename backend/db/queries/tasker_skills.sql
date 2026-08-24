-- name: CreateTaskerSkill :one
INSERT INTO tasker_skills (tasker_id, category_id)
VALUES ($1, $2)
ON CONFLICT (tasker_id, category_id) DO NOTHING
RETURNING *;

-- name: DeleteTaskerSkills :exec
DELETE FROM tasker_skills
WHERE tasker_id = $1;

-- name: ListTaskerSkills :many
SELECT sqlc.embed(tasker_skills), sqlc.embed(categories)
FROM tasker_skills
JOIN categories ON categories.id = tasker_skills.category_id
WHERE tasker_skills.tasker_id = $1
ORDER BY categories.name ASC;

-- name: ListTaskerSkillsByTaskerIDs :many
-- Batch form for list endpoints: one query for the whole page of taskers
-- instead of one per row, which is what Preload was doing.
SELECT sqlc.embed(tasker_skills), sqlc.embed(categories)
FROM tasker_skills
JOIN categories ON categories.id = tasker_skills.category_id
WHERE tasker_skills.tasker_id = ANY(sqlc.arg(tasker_ids)::bigint[])
ORDER BY tasker_skills.tasker_id, categories.name ASC;
