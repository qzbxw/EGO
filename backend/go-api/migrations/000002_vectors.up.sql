-- Embeddings schema for semantic retrieval

CREATE TABLE IF NOT EXISTS message_embeddings (
    id SERIAL PRIMARY KEY,
    log_id INTEGER NOT NULL REFERENCES request_logs(id) ON DELETE CASCADE,
    embedding JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS file_chunks (
    id SERIAL PRIMARY KEY,
    file_attachment_id INTEGER NOT NULL REFERENCES file_attachments(id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL,
    chunk_text TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(file_attachment_id, chunk_index)
);

CREATE TABLE IF NOT EXISTS file_chunk_embeddings (
    id SERIAL PRIMARY KEY,
    file_chunk_id INTEGER NOT NULL REFERENCES file_chunks(id) ON DELETE CASCADE,
    embedding JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(file_chunk_id)
);


