-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Drop old foreign key constraints
ALTER TABLE request_logs DROP CONSTRAINT IF EXISTS request_logs_session_id_fkey;
ALTER TABLE file_attachments DROP CONSTRAINT IF EXISTS file_attachments_session_id_fkey;

-- Add UUID columns to existing tables
ALTER TABLE chat_sessions ADD COLUMN uuid UUID DEFAULT uuid_generate_v4();
ALTER TABLE request_logs ADD COLUMN session_uuid UUID;
ALTER TABLE file_attachments ADD COLUMN session_uuid UUID;

-- Create indexes for UUID columns
CREATE INDEX idx_chat_sessions_uuid ON chat_sessions(uuid);
CREATE INDEX idx_request_logs_session_uuid ON request_logs(session_uuid);
CREATE INDEX idx_file_attachments_session_uuid ON file_attachments(session_uuid);

-- Update existing records to have UUIDs
UPDATE chat_sessions SET uuid = uuid_generate_v4() WHERE uuid IS NULL;
UPDATE request_logs SET session_uuid = cs.uuid 
FROM chat_sessions cs 
WHERE request_logs.session_id = cs.id AND request_logs.session_uuid IS NULL;
UPDATE file_attachments SET session_uuid = cs.uuid 
FROM chat_sessions cs 
WHERE file_attachments.session_id = cs.id AND file_attachments.session_uuid IS NULL;

-- Make UUID columns NOT NULL
ALTER TABLE chat_sessions ALTER COLUMN uuid SET NOT NULL;
ALTER TABLE request_logs ALTER COLUMN session_uuid SET NOT NULL;
ALTER TABLE file_attachments ALTER COLUMN session_uuid SET NOT NULL;

-- Add unique constraints
ALTER TABLE chat_sessions ADD CONSTRAINT chat_sessions_uuid_unique UNIQUE (uuid);

-- Add foreign key constraints for UUID
ALTER TABLE request_logs ADD CONSTRAINT fk_request_logs_session_uuid 
    FOREIGN KEY (session_uuid) REFERENCES chat_sessions(uuid) ON DELETE CASCADE;
ALTER TABLE file_attachments ADD CONSTRAINT fk_file_attachments_session_uuid 
    FOREIGN KEY (session_uuid) REFERENCES chat_sessions(uuid) ON DELETE CASCADE;

-- Drop old ID columns and make UUID the primary key
ALTER TABLE request_logs DROP COLUMN session_id;
ALTER TABLE file_attachments DROP COLUMN session_id;

-- Drop old primary key and make UUID the new primary key
ALTER TABLE chat_sessions DROP CONSTRAINT chat_sessions_pkey;
ALTER TABLE chat_sessions DROP COLUMN id;
ALTER TABLE chat_sessions ADD CONSTRAINT chat_sessions_pkey PRIMARY KEY (uuid);