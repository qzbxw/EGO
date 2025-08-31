-- Revert: Make session_uuid NOT NULL again in file_attachments
-- First, update any NULL values to a default or remove orphaned records
DELETE FROM file_attachments WHERE session_uuid IS NULL;

-- Restore NOT NULL constraint
ALTER TABLE file_attachments ALTER COLUMN session_uuid SET NOT NULL;
