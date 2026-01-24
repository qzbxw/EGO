-- Add is_chat_only column to maintenance_mode table
ALTER TABLE maintenance_mode ADD COLUMN is_chat_only BOOLEAN NOT NULL DEFAULT FALSE;
