-- Ensure one embedding per log and enable upsert to work
-- 1) Deduplicate existing rows by keeping the latest id per log_id
DELETE FROM message_embeddings me
USING message_embeddings me2
WHERE me.log_id = me2.log_id AND me.id < me2.id;

-- 2) Add a unique constraint on log_id (transaction-safe)
ALTER TABLE message_embeddings
    ADD CONSTRAINT message_embeddings_log_id_key UNIQUE (log_id);
