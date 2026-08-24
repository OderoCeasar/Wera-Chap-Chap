-- name: CreateReview :one
INSERT INTO reviews (booking_id, reviewer_id, reviewee_id, rating, comment)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetReviewWithReviewer :one
SELECT sqlc.embed(reviews), sqlc.embed(users)
FROM reviews
JOIN users ON users.id = reviews.reviewer_id
WHERE reviews.id = $1 LIMIT 1;

-- name: CountReviewsByBookingAndReviewer :one
SELECT COUNT(*) FROM reviews
WHERE booking_id = $1 AND reviewer_id = $2;

-- name: ListReviewsByBooking :many
SELECT sqlc.embed(reviews), sqlc.embed(users)
FROM reviews
JOIN users ON users.id = reviews.reviewer_id
WHERE reviews.booking_id = $1
ORDER BY reviews.created_at DESC;

-- name: ListReviewsByReviewee :many
SELECT sqlc.embed(reviews), sqlc.embed(users)
FROM reviews
JOIN users ON users.id = reviews.reviewer_id
WHERE reviews.reviewee_id = $1
ORDER BY reviews.created_at DESC;

-- name: GetRevieweeRatingStats :one
-- Feeds the denormalised columns on tasker_profiles. COALESCE covers the
-- reviewee with no reviews yet, where AVG is NULL.
SELECT
  COALESCE(AVG(rating), 0)::numeric AS average,
  COUNT(*)::int AS total
FROM reviews
WHERE reviewee_id = $1;
