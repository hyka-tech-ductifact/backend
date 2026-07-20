# Roadmap — Ductifact Backend

> **Stack**: Go 1.26 · Gin · GORM · PostgreSQL · JWT · Docker · Caddy · GitHub Actions · Prometheus
> **Architecture**: Hexagonal (Ports & Adapters)
> **Started**: March 2026
> **Last updated**: July 2026

---

## Planning Principles

- Protect data and security boundaries before adding convenience features.
- Deliver and consume complete vertical product flows before expanding the domain.
- Measure before adding caches, indexes, queues, or distributed-system patterns.
- Keep the modular monolith until deployment, scale, security, or team ownership proves a service boundary.
- Treat `🔴 Now` as the current engineering queue; `🟡 Next` is not a commitment until the preceding safety work is complete.

---

## Phase 1 — Foundation (Complete)

### 1. Domain & Entities

| # | Task | Status |
|---|------|--------|
| 1.1 | `User` entity (full CRUD) | ✅ |
| 1.2 | Core business entities: `Client`, `Project`, `Order`, `PieceDefinition`, and `Piece` | ✅ |
| 1.3 | Value Object `Email` (validation) | ✅ |
| 1.4 | Value Object `Password` (bcrypt hash, validation) | ✅ |
| 1.5 | Repository interfaces (ports) | ✅ |
| 1.6 | Typed domain errors | ✅ |

### 2. Application (Use Cases)

| # | Task | Status |
|---|------|--------|
| 2.1 | `UserService` (create, get, update) | ✅ |
| 2.2 | Business services for Client → Project → Order → Piece, including piece definitions and ownership | ✅ |
| 2.3 | `AuthService` (register, login, refresh, logout) | ✅ |

### 3. HTTP Infrastructure

| # | Task | Status |
|---|------|--------|
| 3.1 | Gin router + versioned `/v1/` | ✅ |
| 3.2 | Handlers for Auth, User, Client, Project, Order, PieceDefinition, Piece, Health, Files, and Docs | ✅ |
| 3.3 | Middleware: Logging (structured) | ✅ |
| 3.4 | Middleware: Recovery (panic → 500) | ✅ |
| 3.5 | Middleware: CORS | ✅ |
| 3.6 | Middleware: Request ID (traceability) | ✅ |
| 3.7 | Middleware: Centralized error handler | ✅ |
| 3.8 | Middleware: Auth JWT (Bearer token) | ✅ |
| 3.9 | Middleware: Prometheus metrics | ✅ |
| 3.10 | Graceful shutdown | ✅ |

### 4. Authentication & Authorization

| # | Task | Status |
|---|------|--------|
| 4.1 | JWT (signing, expiration, validation) | ✅ |
| 4.2 | `POST /auth/register` + `POST /auth/login` | ✅ |
| 4.3 | Protected routes (JWT middleware) | ✅ |
| 4.4 | Ownership across `/users/me` and the Client → Project → Order → Piece resource chain | ✅ |
| 4.5 | Password hashing with bcrypt | ✅ |

### 5. Persistence

| # | Task | Status |
|---|------|--------|
| 5.1 | PostgreSQL with GORM | ✅ |
| 5.2 | `PostgresUserRepository` | ✅ |
| 5.3 | PostgreSQL repositories for business resources and one-time OTPs | ✅ |
| 5.4 | Health checker (DB ping) | ✅ |

### 6. API Contracts

| # | Task | Status |
|---|------|--------|
| 6.1 | OpenAPI spec (`contracts/openapi/`) | ✅ |
| 6.2 | Contract tests across infrastructure, account, and business endpoints | ✅ |
| 6.3 | Spec validation in CI (`redocly lint`) | ✅ |
| 6.4 | Swagger UI embedded in API (`/docs`) | ✅ |
| 6.5 | Versioned bundled OpenAPI release artifact consumed by the backend | ✅ |

### 7. Observability

| # | Task | Status |
|---|------|--------|
| 7.1 | Structured logging with `slog` (JSON) | ✅ |
| 7.2 | Health check with DB verification | ✅ |
| 7.3 | Prometheus metrics endpoint | ✅ |

### 8. Testing

| # | Task | Status |
|---|------|--------|
| 8.1 | Unit tests (entities, VOs, services, JWT) | ✅ |
| 8.2 | Hand-written mocks (repositories, ports) | ✅ |
| 8.3 | Integration tests (repositories against DB) | ✅ |
| 8.4 | Contract tests (OpenAPI compliance) | ✅ |
| 8.5 | E2E tests (full HTTP flow) | ✅ |
| 8.6 | CI with race detector + JUnit XML | ✅ |

### 9. CI/CD & DevOps

| # | Task | Status |
|---|------|--------|
| 9.1 | GitHub Actions (lint, vet, tests, contracts) | ✅ |
| 9.2 | Docker multi-stage + optimized cache | ✅ |
| 9.3 | `docker-compose` (dev + prod) | ✅ |
| 9.4 | Automated deploy to Hetzner VPS | ✅ |
| 9.5 | Caddy reverse proxy + HTTPS (Let's Encrypt) | ✅ |
| 9.6 | Block `/metrics`, `/healthz`, `/readyz`, `/docs` from internet | ✅ |

Current CD model: each merge into `main` publishes an immutable candidate image; production promotion is decided in `infra` via manifest updates.

---

## Phase 2 — Hardening (Complete)

### 10. Security

| # | Task | Status |
|---|------|--------|
| 10.1 | Refresh tokens (JWT rotation) | ✅ |
| 10.2 | `POST /auth/refresh` endpoint | ✅ |
| 10.3 | Rate limiting (per IP and per user) | ✅ |
| 10.4 | Login brute-force protection | ✅ |
| 10.5 | Security headers middleware (HSTS, X-Frame, CSP) | ✅ |
| 10.6 | `POST /auth/logout` endpoint (token blacklist) | ✅ |

### 11. Database

| # | Task | Status |
|---|------|--------|
| 11.1 | Versioned migrations (`golang-migrate`) | ✅ |
| 11.2 | Development data seeders | ✅ |
| 11.3 | Soft delete (`deleted_at` logical deletion) | ✅ |
| 11.4 | Backup/restore tooling with retention and an operational runbook | ✅ |

### 12. Developer Experience

| # | Task | Status |
|---|------|--------|
| 12.1 | Pre-commit hooks (auto lint + format) | ✅ |
| 12.2 | Dependabot / Renovate (dependency updates) | ✅ |
| 12.3 | Automated changelog (conventional commits) | ✅ |
| 12.4 | Makefile targets for all operations | ✅ |
| 12.5 | Hot reload in development (`air`) | ✅ |
| 12.6 | Coverage report (HTML) | ✅ |

---

## Phase 3 — Account & API Maturity

> **Goal**: Complete the account lifecycle, enforce safe test/migration boundaries,
> and add measured API/performance improvements before product expansion.

### 13. Account Management

| # | Task | Status | Priority |
|---|------|--------|----------|
| 13.1 | `PUT /auth/password` (change password, requires current) | ✅ | 
| 13.2 | `DELETE /users/me` (account self-deletion, GDPR) | ✅ |
| 13.3 | Email verification during registration (OTP-based) | ✅ |
| 13.4 | `POST /auth/password/reset` (request password-reset OTP) | ✅ |
| 13.5 | `POST /auth/password/reset/verify` (confirm reset with OTP) | ✅ |

### 14. Advanced API

| # | Task | Status | Priority |
|---|------|--------|----------|
| 14.1 | Filtering and sorting on list endpoints (query params) | ✅ |
| 14.2 | Pagination on all existing list endpoints | ✅ |
| 14.3 | Central request-body and content-type limits for JSON and multipart (`413`/`415`) | ⬜ | 🔴 Now |
| 14.4 | Conditional requests and caching (`ETag`, `If-None-Match`, `Last-Modified`) on measured read paths | ⬜ | 🔵 Later |
| 14.5 | Bulk operations (batch create/update) | ⬜ | ⚪ Need-driven |
| 14.6 | Full-text search (PostgreSQL `tsvector`) | ⬜ | ⚪ Need-driven |
| 14.7 | Partial responses (field selection) | ⬜ | ⚪ Maybe never |

### 15. Test Safety & Performance Baselines

| # | Task | Status | Priority |
|---|------|--------|----------|
| 15.1 | Dedicated test database/container plus a guard that refuses destructive cleanup outside test environments | ⬜ | 🔴 Now |
| 15.2 | Migration verification from both an empty DB and the last released schema; document roll-forward/rollback procedure | ⬜ | 🔴 Now |
| 15.3 | Load-test a representative Client → Project → Order → Piece flow with k6 and record a baseline | ⬜ | 🟡 Next |
| 15.4 | Reproducible `pprof` workflow for CPU, memory, goroutines, and blocking analysis | ⬜ | 🔵 Later |
| 15.5 | Review slow queries with `EXPLAIN (ANALYZE, BUFFERS)` and add indexes only from evidence | ⬜ | 🟡 Next |
| 15.6 | Tune PostgreSQL connection-pool limits and timeouts from the load-test baseline | ⬜ | 🟡 Next |

---

## Phase 4 — Horizontal Scaling & Observability

> **Goal**: Replace in-memory adapters with Redis, add production-grade observability,
> and prepare the system for multi-instance deployment.

### 16. Redis & Distributed State

| # | Task | Status | Priority |
|---|------|--------|----------|
| 16.1 | Redis adapter for token blacklist | ✅ 
| 16.2 | Redis adapter for rate limiter (IP + user) | ✅ |
| 16.3 | Redis adapter for login throttler | ✅ |
| 16.4 | Redis health check in `/readyz` endpoint | ✅ |
| 16.5 | Shared read cache only after profiling identifies a hot path and an invalidation strategy | ⬜ | ⚪ Need-driven |
| 16.6 | Memory fallback exists; acceptable for local development and single-instance environments | ✅ |
| 16.7 | Production policy for Redis failure: fail closed or leave readiness, with cross-instance security tests | ⬜ | 🔴 Now |

### 17. Production Observability

| # | Task | Status | Priority |
|---|------|--------|----------|
| 17.1 | Grafana dashboard for latency, error rate, throughput, and service health | ✅ |
| 17.2 | Prometheus alert rules for error spikes, latency, and downtime | ✅ |
| 17.3 | Alertmanager notification routing, tested receivers, and linked response runbooks | ⬜ | 🟡 Next |
| 17.4 | Log aggregation with Loki + Grafana | ⬜ | 🔵 Later |
| 17.5 | Distributed tracing with OpenTelemetry | ⬜ | ⚪ Need-driven |
| 17.6 | Append-only audit infrastructure for user actions and future privileged operator actions | ⬜ | 🔵 Later |
| 17.7 | Aggregated readiness for DB, Redis, MinIO, and SMTP, with critical/degraded states | ✅ |

---

## Phase 5 — Business Features

> **Goal**: Add the features that drive real product value — project
> collaboration, emails, background processing, and richer domain entities.

> **Product validation gate**: before adding more entities, consume the existing
> Client → Project → Order → Piece flow from the frontend with generated contract
> types. Use that end-to-end flow to decide which business feature comes next.

### 18. Project Collaboration & Scoped Authorization

Collaboration roles belong to the relationship between a user and one project;
they are not global roles on `User`. A person may own one project, edit another,
and only view a third. Orders and pieces inherit access from their project.

| # | Task | Status | Priority |
|---|------|--------|----------|
| 18.1 | Authorization ADR: project boundary, `owner`/`editor`/`viewer` matrix, 403/404 policy, and sharing limits | ⬜ | 🟡 Next |
| 18.2 | `ProjectRole` value object and `ProjectMembership` entity with unique `(project_id, user_id)` membership | ⬜ | 🟡 Next |
| 18.3 | Make each project creator its first `owner`; backfill existing projects from their client owner transactionally | ⬜ | 🟡 Next |
| 18.4 | Central project policy used by Project, Order, and Piece use cases; do not rely only on route middleware | ⬜ | 🟡 Next |
| 18.5 | Access-aware `GET /projects` collection and member API: list, add, change role, and remove members | ⬜ | 🔵 Later |
| 18.6 | Membership invariants: at least one owner, ownership transfer, leave-project rules, and concurrency safety | ⬜ | 🔵 Later |
| 18.7 | Invitations with expiring/revocable tokens, idempotent acceptance, and email delivery (uses §19) | ⬜ | 🔵 Later |
| 18.8 | OpenAPI contract, API documentation, generated types, and compatibility review | ⬜ | 🔵 Later |
| 18.9 | Unit, integration, E2E, contract, concurrency, and cross-project isolation tests | ⬜ | 🔵 Later |

Sharing a project exposes its project data, orders, and pieces. It must not
implicitly expose the owner's client record or private piece-definition library;
collaborators receive only the minimum referenced data needed to understand a
shared piece. See [GUIDE_AUTHORIZATION_BOUNDARIES.md](GUIDE_AUTHORIZATION_BOUNDARIES.md).

### 19. Email & Notifications

| # | Task | Status | Priority |
|---|------|--------|----------|
| 19.1 | Email service port (adapter-agnostic) | ✅ |
| 19.2 | SMTP adapter (SendGrid / Mailgun / SES) | ✅ |
| 19.3 | Localized registration OTP email in HTML and plain text | ✅ |
| 19.4 | Localized password-reset OTP email in HTML and plain text | ✅ |
| 19.5 | Rate-limited account-already-registered notice without account enumeration | ✅ |
| 19.6 | Notification preferences (opt-in/out per type) | ⬜ | 🔵 Later |

### 20. Background Processing

| # | Task | Status | Priority |
|---|------|--------|----------|
| 20.1 | Background-delivery ADR: guarantees, retries, idempotency, ordering, and dead-letter policy | ⬜ | 🟡 Next |
| 20.2 | Task queue port (adapter-agnostic interface) | ⬜ | 🟡 Next |
| 20.3 | In-process adapter for local development and explicitly non-critical tasks | ⬜ | 🟡 Next |
| 20.4 | Durable outbox/queue with retry backoff, dead-letter handling, and observable job state | ⬜ | 🔵 Later |
| 20.5 | Move transactional emails to the durable path; recover cleanly from SMTP failure | ⬜ | 🔵 Later |
| 20.6 | Scheduled cleanup tasks for expired OTPs and blacklist entries | ⬜ | 🔵 Later |
| 20.7 | Redis-backed task queue (for example Asynq) only when throughput or multi-instance workers require it | ⬜ | ⚪ Need-driven |

### 21. Domain Enrichment

| # | Task | Status | Priority |
|---|------|--------|----------|
| 21.1 | Measurement/Unit ADR and value objects: mm/cm/in, decimal precision, conversion, and rounding policy | ⬜ | 🔴 Now |
| 21.2 | Complete the existing `Client` profile with address, tax ID, and notes where product flows require them | ⬜ | 🔵 Later |
| 21.3 | Enrich the existing `Project` entity with a status workflow | ⬜ | 🔵 Later |
| 21.4 | Extend the current `Order` pending/completed workflow only from validated business requirements | ⬜ | 🔵 Later |
| 21.5 | Discover and document Quote/Invoice lifecycle, numbering, taxes, and immutability before creating CRUD | ⬜ | 🔵 Later |
| 21.6 | Strengthen the existing `Phone` value object with E.164 normalization when international data requires it | ⬜ | 🔵 Later |
| 21.7 | Country-aware `TaxID` value object | ⬜ | ⚪ Need-driven |
| 21.8 | Domain events only after a real internal consumer exists | ⬜ | ⚪ Need-driven |

---

## Phase 6 — Resilience & Polish

> **Goal**: Production hardening with fault tolerance patterns, data export,
> and advanced API capabilities.

### 22. Resilience Patterns

| # | Task | Status | Priority |
|---|------|--------|----------|
| 22.1 | Per-operation time budgets and context propagation for DB, SMTP, and object storage calls | ⬜ | 🟡 Next |
| 22.2 | Retry with exponential backoff and jitter only for classified, idempotent external operations | ⬜ | 🔵 Later |
| 22.3 | Circuit breakers only after observed external-dependency failure patterns | ⬜ | ⚪ Need-driven |
| 22.4 | Idempotency keys for externally retried write endpoints | ⬜ | 🔵 Later |
| 22.5 | Scheduled encrypted offsite backups, retention monitoring, and a periodic restore drill | ⬜ | 🔴 Now |
| 22.6 | Feature flags | ⬜ | ⚪ Maybe never |

### 23. Data & Export

| # | Task | Status | Priority |
|---|------|--------|----------|
| 23.1 | Data export: clients to CSV | ⬜ | 🔵 Later |
| 23.2 | Data export: clients to PDF | ⬜ | 🔵 Later |
| 23.3 | Piece-definition image upload, validation, thumbnails, and MinIO storage | ✅ |
| 23.4 | Bulk import from CSV | ⬜ | 🔵 Later |

### 24. API Governance

| # | Task | Status | Priority |
|---|------|--------|----------|
| 24.1 | API versioning strategy (v1 deprecation plan) | ⬜ | 🔵 Later |
| 24.2 | API changelog (public, consumer-facing) | ⬜ | 🔵 Later |
| 24.3 | Generate TypeScript API types/client and consume them from the frontend | ⬜ | 🟡 Next |
| 24.4 | Automate contract release/version drift checks and coordinated backend update PRs | ⬜ | 🟡 Next |
| 24.5 | Real-time updates (WebSocket or SSE) | ⬜ | ⚪ Need-driven |
| 24.6 | Organizations / multi-tenancy with organization-scoped memberships | ⬜ | ⚪ Need-driven |

---

## Phase 7 — Internal Platform Operations

> **Goal**: Provide narrowly scoped, auditable tools for support and data repair
> without turning product users into global administrators or bypassing business
> invariants.

### 25. Internal Platform Operations

This is a separate control plane for trusted operators. Start inside the modular
monolith with a distinct entry point and identity boundary; extract a service
only when deployment, security, scale, or team ownership creates a real need.

| # | Task | Status | Priority |
|---|------|--------|----------|
| 25.1 | Catalogue recurring operational cases and write a threat model; avoid a generic `write:any` capability | ⬜ | 🔵 Later |
| 25.2 | Separate operator identity from product users (SSO, MFA, dedicated token audience, revocation) | ⬜ | 🔵 Later |
| 25.3 | Append-only privileged audit events with actor, target, reason/ticket, and before/after data (depends on §17.6) | ⬜ | 🔵 Later |
| 25.4 | Transactional repair commands/jobs with dry-run, idempotency, explicit scope, and runbooks | ⬜ | 🔵 Later |
| 25.5 | Internal Admin/Support API on a separate router or binary and private network boundary, reusing application rules | ⬜ | 🔵 Later |
| 25.6 | Least-privilege support reads with purpose checks and PII masking | ⬜ | 🔵 Later |
| 25.7 | Explicit operational mutations with approval or break-glass controls, executed by the owning business module | ⬜ | ⚪ Maybe never |
| 25.8 | Backoffice UI only after operational workflows become recurring and stable | ⬜ | ⚪ Maybe never |
| 25.9 | Evaluate microservice extraction; never let a separate operations service write another service's tables directly | ⬜ | ⚪ Maybe never |

---

## Summary

```
Phase 1 — Foundation
  Domain & Entities          ████████████████████  6/6   ✅
  Use Cases                  ████████████████████  3/3   ✅
  HTTP Infrastructure        ████████████████████  10/10 ✅
  Authentication             ████████████████████  5/5   ✅
  Persistence                ████████████████████  4/4   ✅
  API Contracts              ████████████████████  5/5   ✅
  Observability              ████████████████████  3/3   ✅
  Testing                    ████████████████████  6/6   ✅
  CI/CD & DevOps             ████████████████████  6/6   ✅

Phase 2 — Hardening
  Security                   ████████████████████  6/6   ✅
  Database                   ████████████████████  4/4   ✅
  Developer Experience       ████████████████████  6/6   ✅

Phase 3 — Account & API Maturity
  Account Management         ████████████████████  5/5   ✅
  Advanced API               ██████░░░░░░░░░░░░░░  2/7
  Test Safety & Performance  ░░░░░░░░░░░░░░░░░░░░  0/6

Phase 4 — Horizontal Scaling & Observability
  Redis & Distributed State  ██████████████░░░░░░  5/7
  Production Observability   █████████░░░░░░░░░░░  3/7

Phase 5 — Business Features
  Project Collaboration      ░░░░░░░░░░░░░░░░░░░░  0/9
  Email & Notifications      ████████████████░░░░  5/6
  Background Processing      ░░░░░░░░░░░░░░░░░░░░  0/7
  Domain Enrichment          ░░░░░░░░░░░░░░░░░░░░  0/8

Phase 6 — Resilience & Polish
  Resilience Patterns        ░░░░░░░░░░░░░░░░░░░░  0/6
  Data & Export              █████░░░░░░░░░░░░░░░  1/4
  API Governance             ░░░░░░░░░░░░░░░░░░░░  0/6

Phase 7 — Internal Platform Operations
  Internal Operations        ░░░░░░░░░░░░░░░░░░░░  0/9
```

> **Total progress**: 64/64 (Phase 1+2) + 21/87 (Phase 3–7) = 85/151 tasks (~56%) -> **Phases 1 & 2 complete** — solid production foundation, security, and DX.
> **Phase 3**: Account Management ✅, API Maturity in progress.
> **Phase 4**: Distributed Redis adapters are in place; production failure policy and notification routing remain pending.
> **Phase 5**: Email foundations are in place; project-scoped collaboration remains planned.
> **Phase 7**: Internal operations are deliberately deferred until concrete, recurring support needs exist.

### Suggested execution order

| Order | Section | Why first |
|-------|---------|-----------|
| 1st | 15.1–15.2 Test and migration safety | Prevent destructive tests from touching development data and prove schema upgrades safely |
| 2nd | 22.5 Recovery | Automate offsite backups and prove that a restore actually works |
| 3rd | 14.3 + 16.7 Runtime boundaries | Limit request bodies and define secure Redis failure behavior in production |
| 4th | 21.1 Measurements | Decide units and decimal precision before more piece data depends on the current representation |
| Product gate | 24.3 + frontend vertical slice | Consume the existing Client → Project → Order → Piece API before expanding it |
| If sharing is validated | 18.1–18.6 Project Collaboration | Implement scoped membership and owner invariants; invitations can follow later |
| Then | 20.1–20.5 Background Processing | Make email delivery durable only after its guarantees are explicit |
| After measurement | 15.3–15.6 Performance | Load-test, profile, inspect SQL, and tune the pool from evidence |
| Only on demonstrated need | 25. Internal Platform Operations | Begin with audited commands; extract a service only for a real boundary |

### Explicitly Deferred Until a Trigger Exists

| Capability | Trigger to reconsider it |
|------------|--------------------------|
| Shared read cache | Profiling shows a stable, expensive hot read and there is a clear invalidation policy |
| Loki or OpenTelemetry | Metrics and structured logs cannot diagnose real production incidents efficiently |
| Full-text search, bulk operations, partial responses | A consumer has a concrete dataset and workflow that needs them |
| Circuit breakers | External dependency failures are recurring and retries/timeouts are insufficient |
| WebSocket/SSE | The product requires time-sensitive server-driven updates |
| Organizations / multi-tenancy | Customers need shared billing, membership, or isolation above the project level |
| Internal operations microservice | Security, deployment, scale, or team ownership requires process-level separation |

### Priority Legend

| Flag | Meaning |
|------|---------|
| 🔴 Now | Current safety, data-integrity, or security queue |
| 🟡 Next | High-value work after the current safety queue |
| 🔵 Later | Wait for product evidence or measured technical need |
| ⚪ Need-driven / Maybe never | Do not schedule without an explicit trigger |
