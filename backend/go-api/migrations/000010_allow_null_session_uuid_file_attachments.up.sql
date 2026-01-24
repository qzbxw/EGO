-- Allow NULL session_uuid in file_attachments for temporary uploads before session creation
ALTER TABLE file_attachments ALTER COLUMN session_uuid DROP NOT NULL;

-- Drop the foreign key constraint temporarily
ALTER TABLE file_attachments DROP CONSTRAINT IF EXISTS fk_file_attachments_session_uuid;

-- Add a new constraint that allows NULL or valid session UUIDs
ALTER TABLE file_attachments ADD CONSTRAINT fk_file_attachments_session_uuid 
    FOREIGN KEY (session_uuid) REFERENCES chat_sessions(uuid) ON DELETE CASCADE;
