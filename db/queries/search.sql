-- name: SearchEntries :many
SELECT * FROM entries
WHERE user_id = $1
  AND archived = false
  AND (
    title ILIKE '%' || $2 || '%'
    OR body_html ILIKE '%' || $2 || '%'
    OR $2 = ANY(attendees)
  )
ORDER BY created_at DESC
LIMIT 100;
