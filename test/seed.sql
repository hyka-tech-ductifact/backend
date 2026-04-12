-- Ductifact Development Seed Data
--
-- Populates the database with realistic test data for local development.
-- Run with: make seed
--
-- This is NOT idempotent — run it on a clean database only.
-- If you need to reset: make services-stop && make services-start && make seed
--
-- All passwords: password123

-- ─── Users ──────────────────────────────────────────────────
-- Two users to test ownership isolation (user A cannot see user B's clients).

INSERT INTO users (id, name, email, password_hash, created_at, updated_at) VALUES
  ('a0000000-0000-0000-0000-000000000001', 'Alice Dev',   'alice@ductifact.dev', '$2a$10$eypcHfvBhsY2MfQov.1mxulMaHdiKiBgv7U9z.ISiCyYoibAksegq', NOW(), NOW()),
  ('b0000000-0000-0000-0000-000000000002', 'Bob Tester',  'bob@ductifact.dev',   '$2a$10$eypcHfvBhsY2MfQov.1mxulMaHdiKiBgv7U9z.ISiCyYoibAksegq', NOW(), NOW());

-- ─── Clients (Alice) ────────────────────────────────────────
-- 8 clients for Alice — enough to test pagination (default page size is usually 10).

INSERT INTO clients (id, name, user_id, created_at, updated_at) VALUES
  ('c0000000-0000-0000-0000-000000000001', 'Acme Corp',          'a0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '7 days', NOW() - INTERVAL '7 days'),
  ('c0000000-0000-0000-0000-000000000002', 'Globex Industries',  'a0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '6 days', NOW() - INTERVAL '6 days'),
  ('c0000000-0000-0000-0000-000000000003', 'Initech Solutions',  'a0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days'),
  ('c0000000-0000-0000-0000-000000000004', 'Umbrella LLC',       'a0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '4 days', NOW() - INTERVAL '4 days'),
  ('c0000000-0000-0000-0000-000000000005', 'Wayne Enterprises',  'a0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'),
  ('c0000000-0000-0000-0000-000000000006', 'Stark Industries',   'a0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
  ('c0000000-0000-0000-0000-000000000007', 'Cyberdyne Systems',  'a0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '1 day',  NOW() - INTERVAL '1 day'),
  ('c0000000-0000-0000-0000-000000000008', 'Wonka Chocolate',    'a0000000-0000-0000-0000-000000000001', NOW(),                      NOW());

-- ─── Clients (Bob) ──────────────────────────────────────────
-- 3 clients for Bob — verifies that Alice's endpoints don't leak Bob's data.

INSERT INTO clients (id, name, user_id, created_at, updated_at) VALUES
  ('c0000000-0000-0000-0000-000000000009', 'Oscorp Labs',     'b0000000-0000-0000-0000-000000000002', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
  ('c0000000-0000-0000-0000-000000000010', 'LexCorp',         'b0000000-0000-0000-0000-000000000002', NOW() - INTERVAL '1 day',  NOW() - INTERVAL '1 day'),
  ('c0000000-0000-0000-0000-000000000011', 'Pied Piper Inc',  'b0000000-0000-0000-0000-000000000002', NOW(),                      NOW());
