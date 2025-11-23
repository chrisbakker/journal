-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Add embedding vector column (1536 dimensions for text-embedding-ada-002 or similar)
ALTER TABLE entries 
ADD COLUMN embedding_vector vector(768),
ADD COLUMN vectors_updated_at TIMESTAMP WITH TIME ZONE;

-- Create index for vector similarity search
CREATE INDEX ON entries USING ivfflat (embedding_vector vector_cosine_ops) WITH (lists = 100);

-- Add index on vectors_updated_at for efficient background job queries
CREATE INDEX idx_entries_vectors_updated_at ON entries(vectors_updated_at) WHERE archived = false;
