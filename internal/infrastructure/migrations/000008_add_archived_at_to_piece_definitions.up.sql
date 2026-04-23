-- Add archived_at column for soft-archive (deactivation without deletion).
-- NULL = active, non-NULL = archived.
ALTER TABLE piece_definitions
    ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

-- Partial index: only non-archived rows are indexed, so queries that
-- filter out archived definitions are fast.
CREATE INDEX IF NOT EXISTS idx_piece_definitions_archived
    ON piece_definitions (archived_at)
    WHERE archived_at IS NOT NULL;
