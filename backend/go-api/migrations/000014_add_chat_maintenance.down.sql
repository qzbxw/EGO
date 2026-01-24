-- Remove is_chat_only column from maintenance_mode table
ALTER TABLE maintenance_mode DROP COLUMN is_chat_only;
