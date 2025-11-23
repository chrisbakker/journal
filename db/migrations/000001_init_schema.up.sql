-- Enable citext extension for case-insensitive email
CREATE EXTENSION IF NOT EXISTS citext;

-- Create users table
CREATE TABLE users (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email         CITEXT UNIQUE NOT NULL,
  display_name  TEXT,
  timezone      TEXT NOT NULL DEFAULT 'America/New_York',
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create entries table
CREATE TABLE entries (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title              TEXT NOT NULL DEFAULT '',
  body_delta         JSONB NOT NULL,
  body_html          TEXT NOT NULL,
  render_version     INT NOT NULL DEFAULT 1,
  attendees_original TEXT NOT NULL DEFAULT '',
  attendees          TEXT[] NOT NULL DEFAULT '{}',
  type               TEXT NOT NULL CHECK (type IN ('meeting','notes','other')),
  day_year           INT NOT NULL,
  day_month          INT NOT NULL,
  day_day            INT NOT NULL,
  archived           BOOLEAN NOT NULL DEFAULT FALSE,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create attachments table
CREATE TABLE attachments (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  entry_id    UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
  filename    TEXT NOT NULL,
  mime_type   TEXT NOT NULL,
  size_bytes  BIGINT NOT NULL,
  data        BYTEA NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX entries_day_idx
  ON entries (user_id, day_year, day_month, day_day, archived, created_at);

CREATE INDEX entries_created_desc_idx
  ON entries (user_id, created_at DESC);

CREATE INDEX attachments_entry_idx
  ON attachments (entry_id);
