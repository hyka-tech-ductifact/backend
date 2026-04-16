CREATE TABLE IF NOT EXISTS piece_definitions (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name              TEXT        NOT NULL,
    image_url         TEXT        NOT NULL DEFAULT '',
    dimension_schema  JSONB       NOT NULL,
    predefined        BOOLEAN     NOT NULL DEFAULT false,
    user_id           UUID        REFERENCES users(id) ON DELETE CASCADE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_piece_definitions_user_id ON piece_definitions (user_id);
CREATE INDEX IF NOT EXISTS idx_piece_definitions_predefined ON piece_definitions (predefined);
CREATE INDEX IF NOT EXISTS idx_piece_definitions_deleted_at ON piece_definitions (deleted_at);
