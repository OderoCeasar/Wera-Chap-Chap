-- name: CreateMessage :one
INSERT INTO messages (booking_id, sender_id, content)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetMessageWithSender :one
SELECT sqlc.embed(messages), sqlc.embed(users)
FROM messages
JOIN users ON users.id = messages.sender_id
WHERE messages.id = $1 LIMIT 1;

-- name: ListMessagesByBooking :many
SELECT sqlc.embed(messages), sqlc.embed(users)
FROM messages
JOIN users ON users.id = messages.sender_id
WHERE messages.booking_id = $1
ORDER BY messages.created_at ASC;

-- name: MarkMessagesRead :exec
-- Reading the thread marks everything the caller did not write as seen.
UPDATE messages
SET is_read = TRUE
WHERE booking_id = $1 AND sender_id <> $2 AND is_read = FALSE;
