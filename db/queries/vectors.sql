-- name: GetEntriesNeedingVectors :many
SELECT id, title, body_text, created_at, updated_at
FROM entries
WHERE user_id = $1
  AND archived = false
  AND (embedding_vector IS NULL 
       OR vectors_updated_at IS NULL 
       OR updated_at > vectors_updated_at)
ORDER BY updated_at DESC
LIMIT $2;

-- name: UpdateEntryVector :exec
UPDATE entries
SET embedding_vector = $2,
    vectors_updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: SearchSimilarEntries :many
SELECT id, title, body_text, day_year, day_month, day_day, attendees, created_at, updated_at,
       (embedding_vector <-> $2::vector) AS distance
FROM entries
WHERE user_id = $1
  AND archived = false
  AND embedding_vector IS NOT NULL
ORDER BY embedding_vector <-> $2::vector
LIMIT $3;
