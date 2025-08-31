-- Create maintenance_mode table
CREATE TABLE IF NOT EXISTS maintenance_mode (
    id SERIAL PRIMARY KEY,
    is_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    bypass_token VARCHAR(255) UNIQUE,
    enabled_at TIMESTAMP,
    disabled_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default record
INSERT INTO maintenance_mode (is_enabled) VALUES (FALSE);

-- Create index for faster lookups
CREATE INDEX idx_maintenance_mode_enabled ON maintenance_mode(is_enabled);
CREATE INDEX idx_maintenance_mode_bypass_token ON maintenance_mode(bypass_token);
