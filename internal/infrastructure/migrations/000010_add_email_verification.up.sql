-- Add email verification support.
-- email_verified_at is NULL until the user clicks the verification link.
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;

-- One-time tokens — generic, type-scoped tokens (email verification, password reset, etc.).
CREATE TABLE IF NOT EXISTS one_time_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      TEXT        NOT NULL UNIQUE,
    type       TEXT        NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_one_time_tokens_user_id_type ON one_time_tokens (user_id, type);
CREATE INDEX IF NOT EXISTS idx_one_time_tokens_token ON one_time_tokens (token);
