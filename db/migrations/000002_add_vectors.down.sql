-- Remove index on vectors_updated_at
DROP INDEX IF EXISTS idx_entries_vectors_updated_at;

-- Remove vector similarity index
DROP INDEX IF EXISTS entries_embedding_vector_idx;

-- Remove vector columns
ALTER TABLE entries 
DROP COLUMN IF EXISTS embedding_vector,
DROP COLUMN IF EXISTS vectors_updated_at;

-- Drop pgvector extension
DROP EXTENSION IF EXISTS vector;
