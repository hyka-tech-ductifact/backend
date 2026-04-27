-- Add locale column to users table.
-- Default 'en' for all existing and new users.
ALTER TABLE users ADD COLUMN IF NOT EXISTS locale TEXT NOT NULL DEFAULT 'en';
