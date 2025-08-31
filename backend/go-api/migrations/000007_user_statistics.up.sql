-- Create user_statistics table for detailed user metrics
CREATE TABLE IF NOT EXISTS user_statistics (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Request metrics
    total_requests INTEGER NOT NULL DEFAULT 0,
    total_sessions INTEGER NOT NULL DEFAULT 0,
    total_files_uploaded INTEGER NOT NULL DEFAULT 0,
    total_files_size_bytes BIGINT NOT NULL DEFAULT 0,
    
    -- Token usage metrics
    total_prompt_tokens BIGINT NOT NULL DEFAULT 0,
    total_completion_tokens BIGINT NOT NULL DEFAULT 0,
    total_tokens BIGINT NOT NULL DEFAULT 0,
    
    -- Timing metrics (in milliseconds)
    avg_response_time_ms INTEGER NOT NULL DEFAULT 0,
    min_response_time_ms INTEGER NOT NULL DEFAULT 0,
    max_response_time_ms INTEGER NOT NULL DEFAULT 0,
    total_response_time_ms BIGINT NOT NULL DEFAULT 0,
    
    -- Thinking iterations metrics
    total_thinking_iterations INTEGER NOT NULL DEFAULT 0,
    avg_thinking_iterations REAL NOT NULL DEFAULT 0,
    max_thinking_iterations INTEGER NOT NULL DEFAULT 0,
    
    -- Advanced metrics
    total_regenerations INTEGER NOT NULL DEFAULT 0,
    memory_enabled_requests INTEGER NOT NULL DEFAULT 0,
    provider_usage JSONB NOT NULL DEFAULT '{}',
    model_usage JSONB NOT NULL DEFAULT '{}',
    
    -- Time tracking
    first_request_at TIMESTAMPTZ,
    last_request_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    UNIQUE(user_id)
);

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_user_statistics_user_id ON user_statistics(user_id);
CREATE INDEX IF NOT EXISTS idx_user_statistics_last_request ON user_statistics(last_request_at);

-- Create detailed request metrics table for storing per-request data
CREATE TABLE IF NOT EXISTS request_metrics (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_uuid TEXT NOT NULL,
    request_log_id BIGINT NOT NULL REFERENCES request_logs(id) ON DELETE CASCADE,
    
    -- Response timing
    response_time_ms INTEGER NOT NULL,
    thinking_time_ms INTEGER NOT NULL DEFAULT 0,
    synthesis_time_ms INTEGER NOT NULL DEFAULT 0,
    
    -- Thinking details
    thinking_iterations INTEGER NOT NULL DEFAULT 0,
    
    -- Request details
    is_regeneration BOOLEAN NOT NULL DEFAULT FALSE,
    memory_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    files_count INTEGER NOT NULL DEFAULT 0,
    files_size_bytes BIGINT NOT NULL DEFAULT 0,
    
    -- Provider info
    llm_provider TEXT NOT NULL DEFAULT 'ego',
    llm_model TEXT,
    
    -- Tokens
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    UNIQUE(request_log_id)
);

-- Create indexes for request_metrics
CREATE INDEX IF NOT EXISTS idx_request_metrics_user_id ON request_metrics(user_id);
CREATE INDEX IF NOT EXISTS idx_request_metrics_session ON request_metrics(session_uuid);
CREATE INDEX IF NOT EXISTS idx_request_metrics_created_at ON request_metrics(created_at);
CREATE INDEX IF NOT EXISTS idx_request_metrics_provider ON request_metrics(llm_provider);