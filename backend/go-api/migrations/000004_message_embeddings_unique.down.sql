-- Drop the unique constraint on log_id
ALTER TABLE message_embeddings
    DROP CONSTRAINT IF EXISTS message_embeddings_log_id_key;
