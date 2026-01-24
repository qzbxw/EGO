-- Add text_tsv column to ego_memory table for full-text search support
-- This column is a generated column that automatically creates a tsvector from the text column
-- Note: ego_memory table is created by Python API, so we check for its existence first

DO $$ 
BEGIN
    -- Check if ego_memory table exists (created by Python API)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = 'ego_memory'
    ) THEN
        RAISE NOTICE 'ego_memory table does not exist yet (created by Python API), skipping migration';
        RETURN;
    END IF;

    -- Check if text_tsv column doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'ego_memory' AND column_name = 'text_tsv'
    ) THEN
        -- Add the generated column
        ALTER TABLE ego_memory 
        ADD COLUMN text_tsv tsvector GENERATED ALWAYS AS (to_tsvector('english', text)) STORED;
        
        -- Create GIN index for fast full-text search
        CREATE INDEX IF NOT EXISTS ego_memory_text_fts_idx ON ego_memory USING GIN (text_tsv);
        
        RAISE NOTICE 'Successfully added text_tsv column and index to ego_memory';
    ELSE
        RAISE NOTICE 'text_tsv column already exists, skipping';
    END IF;
END $$;
