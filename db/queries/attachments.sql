-- name: GetAttachment :one
SELECT * FROM attachments
WHERE id = $1 LIMIT 1;

-- name: ListAttachmentsForEntry :many
SELECT * FROM attachments
WHERE entry_id = $1
ORDER BY created_at ASC;

-- name: CreateAttachment :one
INSERT INTO attachments (
  user_id,
  entry_id,
  filename,
  mime_type,
  size_bytes,
  data
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: DeleteAttachment :exec
DELETE FROM attachments
WHERE id = $1;
