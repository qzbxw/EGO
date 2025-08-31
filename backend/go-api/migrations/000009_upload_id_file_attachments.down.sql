-- Revert upload_id on file_attachments
DROP INDEX IF EXISTS idx_file_attachments_upload_batch;
ALTER TABLE file_attachments DROP COLUMN IF EXISTS upload_id;
