CREATE TABLE session_plans (
    id SERIAL PRIMARY KEY,
    session_uuid UUID NOT NULL REFERENCES chat_sessions(uuid) ON DELETE CASCADE,
    title TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE plan_steps (
    id SERIAL PRIMARY KEY,
    plan_id INTEGER NOT NULL REFERENCES session_plans(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    status TEXT DEFAULT 'pending', -- pending, in_progress, completed, failed, skipped
    step_order INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_session_plans_session_uuid ON session_plans(session_uuid);
CREATE INDEX idx_plan_steps_plan_id ON plan_steps(plan_id);
