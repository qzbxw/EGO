-- Drop foreign key constraints
ALTER TABLE request_logs DROP CONSTRAINT IF EXISTS fk_request_logs_session_uuid;
ALTER TABLE file_attachments DROP CONSTRAINT IF EXISTS fk_file_attachments_session_uuid;

-- Drop unique constraints
ALTER TABLE chat_sessions DROP CONSTRAINT IF EXISTS chat_sessions_uuid_unique;

-- Drop indexes
DROP INDEX IF EXISTS idx_chat_sessions_uuid;
DROP INDEX IF EXISTS idx_request_logs_session_uuid;
DROP INDEX IF EXISTS idx_file_attachments_session_uuid;

-- Restore old primary key
ALTER TABLE chat_sessions DROP CONSTRAINT chat_sessions_pkey;
ALTER TABLE chat_sessions ADD COLUMN id SERIAL PRIMARY KEY;

-- Restore old foreign key columns
ALTER TABLE request_logs ADD COLUMN session_id INTEGER REFERENCES chat_sessions(id) ON DELETE CASCADE;
ALTER TABLE file_attachments ADD COLUMN session_id INTEGER REFERENCES chat_sessions(id) ON DELETE CASCADE;

-- Drop UUID columns
ALTER TABLE chat_sessions DROP COLUMN IF EXISTS uuid;
ALTER TABLE request_logs DROP COLUMN IF EXISTS session_uuid;
ALTER TABLE file_attachments DROP COLUMN IF EXISTS session_uuid;