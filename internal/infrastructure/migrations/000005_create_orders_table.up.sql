CREATE TABLE IF NOT EXISTS orders (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    title      TEXT        NOT NULL,
    status     TEXT        NOT NULL DEFAULT 'pending',
    project_id UUID        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_orders_project_id ON orders (project_id);
CREATE INDEX IF NOT EXISTS idx_orders_deleted_at ON orders (deleted_at);
