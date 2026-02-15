-- Initial database setup
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create users table if it doesn't exist
-- GORM will handle table creation, but this ensures UUID extension is available