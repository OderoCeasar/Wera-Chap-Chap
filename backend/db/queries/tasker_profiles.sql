-- name: CreateTaskerProfile :one
INSERT INTO tasker_profiles (
  user_id,
  bio,
  hourly_rate,
  years_experience,
  service_radius_km,
  is_available
) VALUES (
  $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetTaskerProfile :one
SELECT * FROM tasker_profiles
WHERE id = $1 LIMIT 1;

-- name: GetTaskerProfileByUserID :one
SELECT * FROM tasker_profiles
WHERE user_id = $1 LIMIT 1;

-- name: GetTaskerProfileWithUser :one
SELECT sqlc.embed(tasker_profiles), sqlc.embed(users)
FROM tasker_profiles
JOIN users ON users.id = tasker_profiles.user_id
WHERE tasker_profiles.id = $1 LIMIT 1;

-- name: GetTaskerProfileWithUserByUserID :one
SELECT sqlc.embed(tasker_profiles), sqlc.embed(users)
FROM tasker_profiles
JOIN users ON users.id = tasker_profiles.user_id
WHERE tasker_profiles.user_id = $1 LIMIT 1;

-- name: ListTaskerProfiles :many
-- The directory listing. Every filter is optional: a NULL argument drops its
-- own clause, which is how one prepared statement covers what the handler used
-- to assemble by chaining query builders.
--
-- The category filter is an EXISTS rather than a JOIN so a tasker holding
-- several skills still comes back as one row.
SELECT sqlc.embed(tasker_profiles), sqlc.embed(users)
FROM tasker_profiles
JOIN users ON users.id = tasker_profiles.user_id
WHERE
  (sqlc.narg(available_only)::boolean IS NOT TRUE OR tasker_profiles.is_available = TRUE)
  AND (
    sqlc.narg(category_id)::bigint IS NULL
    OR EXISTS (
      SELECT 1 FROM tasker_skills
      WHERE tasker_skills.tasker_id = tasker_profiles.id
        AND tasker_skills.category_id = sqlc.narg(category_id)
    )
  )
  AND (sqlc.narg(max_rate)::numeric IS NULL OR tasker_profiles.hourly_rate <= sqlc.narg(max_rate))
  AND (sqlc.narg(min_rating)::numeric IS NULL OR tasker_profiles.avg_rating >= sqlc.narg(min_rating))
  AND (
    sqlc.narg(search)::text IS NULL
    OR users.full_name ILIKE '%' || sqlc.narg(search) || '%'
    OR tasker_profiles.bio ILIKE '%' || sqlc.narg(search) || '%'
  )
ORDER BY tasker_profiles.avg_rating DESC, tasker_profiles.total_reviews DESC;

-- name: UpdateTaskerProfile :one
-- Partial update: the dashboard sends only the fields the tasker touched, and
-- a NULL here means "not supplied" rather than "set to zero".
UPDATE tasker_profiles
SET
  bio = COALESCE(sqlc.narg(bio), bio),
  hourly_rate = COALESCE(sqlc.narg(hourly_rate), hourly_rate),
  years_experience = COALESCE(sqlc.narg(years_experience), years_experience),
  service_radius_km = COALESCE(sqlc.narg(service_radius_km), service_radius_km),
  is_available = COALESCE(sqlc.narg(is_available), is_available),
  updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdateTaskerRatingByUserID :exec
-- Refreshes the denormalised rating columns after a review lands. A no-op when
-- the reviewee is a client, who has no tasker profile -- hence :exec rather
-- than :one, which would report ErrNoRows for a perfectly normal case.
UPDATE tasker_profiles
SET avg_rating = $2, total_reviews = $3, updated_at = now()
WHERE user_id = $1;
