-- name: GetOrCreateAttendee :one
INSERT INTO attendees (user_id, name, last_used, use_count)
VALUES ($1, $2, NOW(), 1)
ON CONFLICT (user_id, name) 
DO UPDATE SET 
  last_used = NOW(),
  use_count = attendees.use_count + 1
RETURNING *;

-- name: SearchAttendees :many
SELECT name, last_used, use_count
FROM attendees
WHERE user_id = $1
  AND name ILIKE sqlc.arg(query) || '%'
ORDER BY use_count DESC, last_used DESC
LIMIT $2;

-- name: GetRecentAttendees :many
SELECT name, last_used, use_count
FROM attendees
WHERE user_id = $1
ORDER BY last_used DESC
LIMIT $2;

-- name: GetTopAttendees :many
SELECT name, last_used, use_count
FROM attendees
WHERE user_id = $1
ORDER BY use_count DESC, last_used DESC
LIMIT $2;
