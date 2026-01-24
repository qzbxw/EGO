-- Add nullable upload_id to file_attachments for multipart batch association
-- Requires uuid-ossp (enabled in 000006)

ALTER TABLE file_attachments
    ADD COLUMN IF NOT EXISTS upload_id UUID NULL;

-- Composite index optimized for lookups by (upload_id, user_id, session_uuid)
CREATE INDEX IF NOT EXISTS idx_file_attachments_upload_batch
    ON file_attachments (upload_id, user_id, session_uuid);
