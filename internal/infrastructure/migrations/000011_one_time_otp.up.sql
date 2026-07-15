-- Unified one-time OTP flow for registration and password reset.
--
-- Accounts are created only AFTER the email is verified via a one-time code,
-- so the email_verified_at flag is no longer needed — every user is verified.
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;

-- The link-based one-time token mechanism is replaced by numeric OTP codes.
DROP TABLE IF EXISTS one_time_tokens;

-- Generic, purpose-scoped OTP codes keyed by email.
-- Stores only the bcrypt hash of the code, never the plaintext.
-- There is at most one active OTP per (email, purpose); a new one replaces it.
CREATE TABLE IF NOT EXISTS one_time_otps (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT        NOT NULL,
    purpose    TEXT        NOT NULL,
    code_hash  TEXT        NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    attempts   INT         NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (email, purpose)
);

CREATE INDEX IF NOT EXISTS idx_one_time_otps_email_purpose ON one_time_otps (email, purpose);
