CREATE TABLE IF NOT EXISTS maintenance_mode (
    id SERIAL PRIMARY KEY,
    is_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    bypass_token VARCHAR(255) UNIQUE,
    enabled_at TIMESTAMP,
    disabled_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_maintenance_mode_enabled ON maintenance_mode(is_enabled);
CREATE INDEX IF NOT EXISTS idx_maintenance_mode_bypass_token ON maintenance_mode(bypass_token);

-- Insert the initial record if the table is empty
INSERT INTO maintenance_mode (is_enabled)
SELECT FALSE
WHERE NOT EXISTS (SELECT 1 FROM maintenance_mode);