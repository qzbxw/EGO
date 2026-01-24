-- Drop ANN indexes and vector columns (keep original JSONB)
DROP INDEX IF EXISTS me_embedding_vec_ivfflat;
DROP INDEX IF EXISTS fce_embedding_vec_ivfflat;
ALTER TABLE message_embeddings DROP COLUMN IF EXISTS embedding_vec;
ALTER TABLE file_chunk_embeddings DROP COLUMN IF EXISTS embedding_vec;
-- Do not drop extension globally, might be used by other objects