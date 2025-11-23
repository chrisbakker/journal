-- Drop indexes
DROP INDEX IF EXISTS attachments_entry_idx;
DROP INDEX IF EXISTS entries_created_desc_idx;
DROP INDEX IF EXISTS entries_day_idx;

-- Drop tables
DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS entries;
DROP TABLE IF EXISTS users;

-- Drop extension
DROP EXTENSION IF EXISTS citext;
