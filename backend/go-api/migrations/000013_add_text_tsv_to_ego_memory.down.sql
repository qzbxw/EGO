-- Remove text_tsv column and its index from ego_memory table

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = 'ego_memory'
    ) THEN
        DROP INDEX IF EXISTS ego_memory_text_fts_idx;
        ALTER TABLE ego_memory DROP COLUMN IF EXISTS text_tsv;
    END IF;
END $$;
