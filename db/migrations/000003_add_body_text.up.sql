-- Add body_text column for plain text content from Quill
ALTER TABLE entries ADD COLUMN body_text TEXT NOT NULL DEFAULT '';
