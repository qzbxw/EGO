ALTER TABLE users
ADD COLUMN llm_provider TEXT NOT NULL DEFAULT 'ego',
ADD COLUMN llm_model TEXT,
ADD COLUMN encrypted_api_key TEXT;