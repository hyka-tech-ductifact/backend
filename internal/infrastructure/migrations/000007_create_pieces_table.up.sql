CREATE TABLE IF NOT EXISTS pieces (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    title          TEXT        NOT NULL,
    order_id       UUID        NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    definition_id  UUID        NOT NULL REFERENCES piece_definitions(id),
    dimensions     JSONB       NOT NULL,
    quantity       INT         NOT NULL DEFAULT 1 CHECK (quantity > 0),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_pieces_order_id ON pieces (order_id);
CREATE INDEX IF NOT EXISTS idx_pieces_definition_id ON pieces (definition_id);
CREATE INDEX IF NOT EXISTS idx_pieces_deleted_at ON pieces (deleted_at);
