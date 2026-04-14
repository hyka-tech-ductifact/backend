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

INSERT INTO clients (id, name, phone, email, description, user_id, created_at, updated_at) VALUES
  ('c0000000-0000-0000-0000-000000000001', 'Acme Corp',          '+34 600 100 001', 'acme@example.com',       'Leading products manufacturer',           'a0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '7 days', NOW() - INTERVAL '7 days'),
  ('c0000000-0000-0000-0000-000000000002', 'Globex Industries',  '+34 600 100 002', 'globex@example.com',     'Global exports and logistics',            'a0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '6 days', NOW() - INTERVAL '6 days'),
  ('c0000000-0000-0000-0000-000000000003', 'Initech Solutions',  '+34 600 100 003', 'initech@example.com',    'IT consulting and TPS reports',           'a0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days'),
  ('c0000000-0000-0000-0000-000000000004', 'Umbrella LLC',       '+34 600 100 004', 'umbrella@example.com',   'Pharmaceutical research',                 'a0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '4 days', NOW() - INTERVAL '4 days'),
  ('c0000000-0000-0000-0000-000000000005', 'Wayne Enterprises',  '+34 600 100 005', '',                       'Applied sciences and defense',            'a0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'),
  ('c0000000-0000-0000-0000-000000000006', 'Stark Industries',   '',                'stark@example.com',      '',                                        'a0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
  ('c0000000-0000-0000-0000-000000000007', 'Cyberdyne Systems',  '+34 600 100 007', 'cyberdyne@example.com',  'AI and robotics',                         'a0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '1 day',  NOW() - INTERVAL '1 day'),
  ('c0000000-0000-0000-0000-000000000008', 'Wonka Chocolate',    '',                '',                       'Premium confectionery',                   'a0000000-0000-0000-0000-000000000001', NOW(),                      NOW());

-- ─── Clients (Bob) ──────────────────────────────────────────
-- 3 clients for Bob — verifies that Alice's endpoints don't leak Bob's data.

INSERT INTO clients (id, name, phone, email, description, user_id, created_at, updated_at) VALUES
  ('c0000000-0000-0000-0000-000000000009', 'Oscorp Labs',     '+34 600 200 001', 'oscorp@example.com',  'Genetics and biotech',    'b0000000-0000-0000-0000-000000000002', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
  ('c0000000-0000-0000-0000-000000000010', 'LexCorp',         '+34 600 200 002', 'lex@example.com',     'Real estate and tech',    'b0000000-0000-0000-0000-000000000002', NOW() - INTERVAL '1 day',  NOW() - INTERVAL '1 day'),
  ('c0000000-0000-0000-0000-000000000011', 'Pied Piper Inc',  '+34 600 200 003', 'pp@example.com',      'Middle-out compression',  'b0000000-0000-0000-0000-000000000002', NOW(),                      NOW());

-- ─── Projects (Alice → Acme Corp) ──────────────────────────
-- 3 projects for Acme Corp — enough to test listing and pagination.

INSERT INTO projects (id, name, address, manager_name, phone, description, client_id, created_at, updated_at) VALUES
  ('p0000000-0000-0000-0000-000000000001', 'Residential Tower B',       'Calle Mayor 12, Madrid',              'Carlos Pérez',    '+34 699 111 001', '14-storey residential building, phase 1',     'c0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '6 days', NOW() - INTERVAL '6 days'),
  ('p0000000-0000-0000-0000-000000000002', 'Office Park Norte',         'Av. de la Constitución 45, Sevilla',  'Laura Gómez',     '+34 699 111 002', 'Mixed-use office complex with parking',        'c0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '4 days', NOW() - INTERVAL '4 days'),
  ('p0000000-0000-0000-0000-000000000003', 'Warehouse Logistics Hub',   'Polígono Industrial Sur, Valencia',   'Miguel Torres',   '+34 699 111 003', 'Cold-storage warehouse for distribution',      'c0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days');

-- ─── Projects (Alice → Globex Industries) ───────────────────
-- 2 projects for Globex — verifies isolation between clients.

INSERT INTO projects (id, name, address, manager_name, phone, description, client_id, created_at, updated_at) VALUES
  ('p0000000-0000-0000-0000-000000000004', 'Port Expansion Phase 2',    'Puerto de Barcelona, Muelle 9',       'Ana Ruiz',        '+34 699 222 001', 'Container terminal expansion',                'c0000000-0000-0000-0000-000000000002', NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days'),
  ('p0000000-0000-0000-0000-000000000005', 'Solar Farm Andalucía',      'Carretera A-92 km 34, Antequera',     '',                '+34 699 222 002', '',                                            'c0000000-0000-0000-0000-000000000002', NOW() - INTERVAL '1 day',  NOW() - INTERVAL '1 day');

-- ─── Projects (Alice → Stark Industries) ────────────────────
-- 1 project with minimal fields — tests optional fields as empty strings.

INSERT INTO projects (id, name, address, manager_name, phone, description, client_id, created_at, updated_at) VALUES
  ('p0000000-0000-0000-0000-000000000006', 'Arc Reactor Facility',      '',                                    '',                '',                '',                                            'c0000000-0000-0000-0000-000000000006', NOW(),                      NOW());

-- ─── Projects (Bob → Oscorp Labs) ───────────────────────────
-- 2 projects for Bob — verifies that Alice cannot see Bob's projects.

INSERT INTO projects (id, name, address, manager_name, phone, description, client_id, created_at, updated_at) VALUES
  ('p0000000-0000-0000-0000-000000000007', 'Genetics Lab Expansion',    'Calle Genómica 8, Barcelona',         'Peter Parker',    '+34 699 333 001', 'New wing for genome sequencing',               'c0000000-0000-0000-0000-000000000009', NOW() - INTERVAL '1 day',  NOW() - INTERVAL '1 day'),
  ('p0000000-0000-0000-0000-000000000008', 'Biotech Campus',            'Parque Científico, Bilbao',           'Gwen Stacy',      '+34 699 333 002', 'R&D campus with 3 buildings',                 'c0000000-0000-0000-0000-000000000009', NOW(),                      NOW());
