-- Create attendees table to store unique attendee names
CREATE TABLE attendees (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_used  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  use_count  INT NOT NULL DEFAULT 1,
  UNIQUE(user_id, name)
);

-- Create index for fast lookups and autocomplete
CREATE INDEX idx_attendees_user_name ON attendees(user_id, name);
CREATE INDEX idx_attendees_last_used ON attendees(user_id, last_used DESC);
