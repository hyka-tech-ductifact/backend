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

INSERT INTO users (id, name, email, password_hash, locale, created_at, updated_at) VALUES
  ('a0000000-0000-0000-0000-000000000001', 'Alice Dev',   'alice@ductifact.dev', '$2a$10$eypcHfvBhsY2MfQov.1mxulMaHdiKiBgv7U9z.ISiCyYoibAksegq', 'en', NOW(), NOW()),
  ('b0000000-0000-0000-0000-000000000002', 'Bob Tester',  'bob@ductifact.dev',   '$2a$10$eypcHfvBhsY2MfQov.1mxulMaHdiKiBgv7U9z.ISiCyYoibAksegq', 'es', NOW(), NOW());

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
  ('f0000000-0000-0000-0000-000000000001', 'Residential Tower B',       'Calle Mayor 12, Madrid',              'Carlos Pérez',    '+34 699 111 001', '14-storey residential building, phase 1',     'c0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '6 days', NOW() - INTERVAL '6 days'),
  ('f0000000-0000-0000-0000-000000000002', 'Office Park Norte',         'Av. de la Constitución 45, Sevilla',  'Laura Gómez',     '+34 699 111 002', 'Mixed-use office complex with parking',        'c0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '4 days', NOW() - INTERVAL '4 days'),
  ('f0000000-0000-0000-0000-000000000003', 'Warehouse Logistics Hub',   'Polígono Industrial Sur, Valencia',   'Miguel Torres',   '+34 699 111 003', 'Cold-storage warehouse for distribution',      'c0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days');

-- ─── Projects (Alice → Globex Industries) ───────────────────
-- 2 projects for Globex — verifies isolation between clients.

INSERT INTO projects (id, name, address, manager_name, phone, description, client_id, created_at, updated_at) VALUES
  ('f0000000-0000-0000-0000-000000000004', 'Port Expansion Phase 2',    'Puerto de Barcelona, Muelle 9',       'Ana Ruiz',        '+34 699 222 001', 'Container terminal expansion',                'c0000000-0000-0000-0000-000000000002', NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days'),
  ('f0000000-0000-0000-0000-000000000005', 'Solar Farm Andalucía',      'Carretera A-92 km 34, Antequera',     '',                '+34 699 222 002', '',                                            'c0000000-0000-0000-0000-000000000002', NOW() - INTERVAL '1 day',  NOW() - INTERVAL '1 day');

-- ─── Projects (Alice → Stark Industries) ────────────────────
-- 1 project with minimal fields — tests optional fields as empty strings.

INSERT INTO projects (id, name, address, manager_name, phone, description, client_id, created_at, updated_at) VALUES
  ('f0000000-0000-0000-0000-000000000006', 'Arc Reactor Facility',      '',                                    '',                '',                '',                                            'c0000000-0000-0000-0000-000000000006', NOW(),                      NOW());

-- ─── Projects (Bob → Oscorp Labs) ───────────────────────────
-- 2 projects for Bob — verifies that Alice cannot see Bob's projects.

INSERT INTO projects (id, name, address, manager_name, phone, description, client_id, created_at, updated_at) VALUES
  ('f0000000-0000-0000-0000-000000000007', 'Genetics Lab Expansion',    'Calle Genómica 8, Barcelona',         'Peter Parker',    '+34 699 333 001', 'New wing for genome sequencing',               'c0000000-0000-0000-0000-000000000009', NOW() - INTERVAL '1 day',  NOW() - INTERVAL '1 day'),
  ('f0000000-0000-0000-0000-000000000008', 'Biotech Campus',            'Parque Científico, Bilbao',           'Gwen Stacy',      '+34 699 333 002', 'R&D campus with 3 buildings',                 'c0000000-0000-0000-0000-000000000009', NOW(),                      NOW());

-- ─── Orders (Alice → Acme Corp → Residential Tower B) ──────
-- 3 orders for the first project — enough to test listing and pagination.

INSERT INTO orders (id, title, status, description, project_id, created_at, updated_at) VALUES
  ('e0000000-0000-0000-0000-000000000001', 'Steel beams – lot 3',         'pending',   'First batch of structural steel for floors 1-5',  'f0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days'),
  ('e0000000-0000-0000-0000-000000000002', 'Concrete mix – delivery 1',   'completed', 'High-strength concrete for foundation',            'f0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'),
  ('e0000000-0000-0000-0000-000000000003', 'Electrical wiring – phase A', 'pending',   '',                                                 'f0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '1 day',  NOW() - INTERVAL '1 day');

-- ─── Orders (Alice → Acme Corp → Office Park Norte) ────────
-- 2 orders for the second project — verifies isolation between projects.

INSERT INTO orders (id, title, status, description, project_id, created_at, updated_at) VALUES
  ('e0000000-0000-0000-0000-000000000004', 'HVAC units – building A',     'pending',   'Central air conditioning for office floors',       'f0000000-0000-0000-0000-000000000002', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'),
  ('e0000000-0000-0000-0000-000000000005', 'Elevator installation',       'completed', 'Two passenger elevators, capacity 1000kg',         'f0000000-0000-0000-0000-000000000002', NOW() - INTERVAL '1 day',  NOW() - INTERVAL '1 day');

-- ─── Orders (Alice → Globex → Port Expansion Phase 2) ──────
-- 1 order with minimal data — tests defaults.

INSERT INTO orders (id, title, status, description, project_id, created_at, updated_at) VALUES
  ('e0000000-0000-0000-0000-000000000006', 'Container crane rental',      'pending',   '',                                                 'f0000000-0000-0000-0000-000000000004', NOW(),                      NOW());

-- ─── Orders (Bob → Oscorp Labs → Genetics Lab Expansion) ───
-- 2 orders for Bob — verifies that Alice cannot see Bob's orders.

INSERT INTO orders (id, title, status, description, project_id, created_at, updated_at) VALUES
  ('e0000000-0000-0000-0000-000000000007', 'Gene sequencer – model X',    'pending',   'Latest generation genome sequencer',               'f0000000-0000-0000-0000-000000000007', NOW() - INTERVAL '1 day',  NOW() - INTERVAL '1 day'),
  ('e0000000-0000-0000-0000-000000000008', 'Lab bench furniture',         'completed', '',                                                 'f0000000-0000-0000-0000-000000000007', NOW(),                      NOW());

-- ─── Piece Definitions (predefined — visible to all users) ─────────────────
-- 2 predefined definitions for common pieces.

INSERT INTO piece_definitions (id, name, image_url, dimension_schema, predefined, user_id, created_at, updated_at) VALUES
  ('d0000000-0000-0000-0000-000000000001', 'Standard Rectangle', '', '["Length","Width"]',              true,  NULL,                                    NOW() - INTERVAL '30 days', NOW() - INTERVAL '30 days'),
  ('d0000000-0000-0000-0000-000000000002', 'Standard Beam',      '', '["Length","Width","Height"]',     true,  NULL,                                    NOW() - INTERVAL '30 days', NOW() - INTERVAL '30 days');

-- ─── Piece Definitions (Alice — custom) ─────────────────────────────────────
-- 3 custom definitions for Alice.

INSERT INTO piece_definitions (id, name, image_url, dimension_schema, predefined, user_id, created_at, updated_at) VALUES
  ('d0000000-0000-0000-0000-000000000003', 'L-Bracket',       'https://example.com/l-bracket.png', '["ArmLength","ArmWidth","Thickness"]', false, 'a0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days'),
  ('d0000000-0000-0000-0000-000000000004', 'Circular Plate',  '',                                   '["Diameter","Thickness"]',              false, 'a0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'),
  ('d0000000-0000-0000-0000-000000000005', 'Angle Iron',      '',                                   '["Length","Width"]',                    false, 'a0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '1 day',  NOW() - INTERVAL '1 day');

-- ─── Piece Definitions (Bob — custom) ───────────────────────────────────────
-- 1 custom definition for Bob — verifies that Alice cannot see Bob's custom defs.

INSERT INTO piece_definitions (id, name, image_url, dimension_schema, predefined, user_id, created_at, updated_at) VALUES
  ('d0000000-0000-0000-0000-000000000006', 'Lab Panel',       '',                                   '["Length","Width","Thickness"]',        false, 'b0000000-0000-0000-0000-000000000002', NOW(),                      NOW());

-- ─── Pieces (Alice → Acme Corp → Residential Tower B → Steel beams) ────────
-- 3 pieces for the first order — enough to test listing and pagination.

INSERT INTO pieces (id, title, order_id, definition_id, dimensions, quantity, created_at, updated_at) VALUES
  ('a1000000-0000-0000-0000-000000000001', 'Main beam – floor 1',   'e0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000002', '{"Length":600,"Width":30,"Height":15}',   8,  NOW() - INTERVAL '4 days', NOW() - INTERVAL '4 days'),
  ('a1000000-0000-0000-0000-000000000002', 'Cross beam – floor 1',  'e0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000002', '{"Length":400,"Width":25,"Height":12}',   12, NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'),
  ('a1000000-0000-0000-0000-000000000003', 'Support bracket',       'e0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000003', '{"ArmLength":20,"ArmWidth":10,"Thickness":2}', 24, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days');

-- ─── Pieces (Alice → Acme Corp → Residential Tower B → Concrete mix) ───────
-- 1 piece for the second order.

INSERT INTO pieces (id, title, order_id, definition_id, dimensions, quantity, created_at, updated_at) VALUES
  ('a1000000-0000-0000-0000-000000000004', 'Foundation plate',      'e0000000-0000-0000-0000-000000000002', 'd0000000-0000-0000-0000-000000000001', '{"Length":300,"Width":200}',              4,  NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days');

-- ─── Pieces (Bob → Oscorp → Gene sequencer) ────────────────────────────────
-- 1 piece for Bob — verifies that Alice cannot see Bob's pieces.

INSERT INTO pieces (id, title, order_id, definition_id, dimensions, quantity, created_at, updated_at) VALUES
  ('a1000000-0000-0000-0000-000000000005', 'Sequencer housing',     'e0000000-0000-0000-0000-000000000007', 'd0000000-0000-0000-0000-000000000006', '{"Length":120,"Width":60,"Thickness":3}', 2,  NOW() - INTERVAL '1 day',  NOW() - INTERVAL '1 day');
