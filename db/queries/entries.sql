-- name: GetEntry :one
SELECT * FROM entries
WHERE id = $1 AND archived = false LIMIT 1;

-- name: ListEntriesForDay :many
SELECT * FROM entries
WHERE user_id = $1
  AND day_year = $2
  AND day_month = $3
  AND day_day = $4
  AND archived = false
ORDER BY created_at ASC;

-- name: CreateEntry :one
INSERT INTO entries (
  user_id,
  title,
  body_delta,
  body_html,
  body_text,
  attendees_original,
  attendees,
  type,
  day_year,
  day_month,
  day_day
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING *;

-- name: UpdateEntry :one
UPDATE entries
SET title = $2,
    body_delta = $3,
    body_html = $4,
    body_text = $5,
    attendees_original = $6,
    attendees = $7,
    type = $8,
    updated_at = NOW()
WHERE id = $1 AND archived = false
RETURNING *;

-- name: SoftDeleteEntry :exec
UPDATE entries
SET archived = true,
    updated_at = NOW()
WHERE id = $1;

-- name: GetDaysWithEntries :many
SELECT DISTINCT day_year, day_month, day_day
FROM entries
WHERE user_id = $1
  AND day_year = $2
  AND day_month = $3
  AND archived = false
ORDER BY day_year, day_month, day_day;

-- name: ListAllEntries :many
SELECT * FROM entries
WHERE archived = false
ORDER BY day_year DESC, day_month DESC, day_day DESC, created_at ASC;
