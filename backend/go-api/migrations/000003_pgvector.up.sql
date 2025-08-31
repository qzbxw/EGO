-- Enable pgvector and add vector columns with backfill and indexes

-- enable extension
CREATE EXTENSION IF NOT EXISTS vector;

-- helper to convert JSONB array to vector literal and cast
CREATE OR REPLACE FUNCTION jsonb_to_vector(arr JSONB)
RETURNS vector AS $$
  SELECT ('[' || string_agg((elem::text), ',') || ']')::vector
  FROM (
    SELECT (jsonb_array_elements(arr))::text::float
  ) AS t(elem);
$$ LANGUAGE sql IMMUTABLE;

-- add vector columns
ALTER TABLE message_embeddings ADD COLUMN IF NOT EXISTS embedding_vec vector(256);
ALTER TABLE file_chunk_embeddings ADD COLUMN IF NOT EXISTS embedding_vec vector(256);

-- backfill existing rows (best effort)
UPDATE message_embeddings SET embedding_vec = jsonb_to_vector(embedding) WHERE embedding_vec IS NULL;
UPDATE file_chunk_embeddings SET embedding_vec = jsonb_to_vector(embedding) WHERE embedding_vec IS NULL;

-- create ANN indexes
CREATE INDEX IF NOT EXISTS me_embedding_vec_ivfflat ON message_embeddings USING ivfflat (embedding_vec vector_cosine_ops) WITH (lists = 100);
CREATE INDEX IF NOT EXISTS fce_embedding_vec_ivfflat ON file_chunk_embeddings USING ivfflat (embedding_vec vector_cosine_ops) WITH (lists = 100);