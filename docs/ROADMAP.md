# Roadmap — Ductifact Backend

> **Stack**: Go 1.26 · Gin · GORM · PostgreSQL · JWT · Docker · Caddy · GitHub Actions · Prometheus
> **Architecture**: Hexagonal (Ports & Adapters)
> **Started**: March 2026
> **Last updated**: April 2026

---

## Phase 1 — Foundation (Complete)

### 1. Domain & Entities

| # | Task | Status |
|---|------|--------|
| 1.1 | `User` entity (full CRUD) | ✅ |
| 1.2 | `Client` entity (CRUD, 1:N relationship with User) | ✅ |
| 1.3 | Value Object `Email` (validation) | ✅ |
| 1.4 | Value Object `Password` (bcrypt hash, validation) | ✅ |
| 1.5 | Repository interfaces (ports) | ✅ |
| 1.6 | Typed domain errors | ✅ |

### 2. Application (Use Cases)

| # | Task | Status |
|---|------|--------|
| 2.1 | `UserService` (create, get, update) | ✅ |
| 2.2 | `ClientService` (CRUD with ownership) | ✅ |
| 2.3 | `AuthService` (register, login, refresh, logout) | ✅ |

### 3. HTTP Infrastructure

| # | Task | Status |
|---|------|--------|
| 3.1 | Gin router + versioned `/v1/` | ✅ |
| 3.2 | Handlers (User, Client, Auth, Health, Docs) | ✅ |
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
| 4.4 | Ownership: `/users/me`, `/users/me/clients` | ✅ |
| 4.5 | Password hashing with bcrypt | ✅ |

### 5. Persistence

| # | Task | Status |
|---|------|--------|
| 5.1 | PostgreSQL with GORM | ✅ |
| 5.2 | `PostgresUserRepository` | ✅ |
| 5.3 | `PostgresClientRepository` | ✅ |
| 5.4 | Health checker (DB ping) | ✅ |

### 6. API Contracts

| # | Task | Status |
|---|------|--------|
| 6.1 | OpenAPI spec (`contracts/openapi/`) | ✅ |
| 6.2 | Contract tests (auth, user, client, health) | ✅ |
| 6.3 | Spec validation in CI (`redocly lint`) | ✅ |
| 6.4 | Swagger UI embedded in API (`/docs`) | ✅ |
| 6.5 | TypeScript type generation for frontend | ✅ |

### 7. Observability

| # | Task | Status |
|---|------|--------|
| 7.1 | Structured logging with `slog` (JSON) | ✅ |
| 7.2 | Health check with DB verification | ✅ |
| 7.3 | Prometheus metrics endpoint | ✅ |
| 7.4 | Prometheus server (scraping + alerts) | ✅ |
| 7.5 | Grafana dashboards | ✅ |

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
| 11.6 | Automated database backups (`pg_dump` + offsite) | ✅ |

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

> **Goal**: Complete the user account lifecycle, enrich the API with filtering/sorting,
> and establish performance baselines before scaling.

### 13. Account Management

| # | Task | Status | Priority |
|---|------|--------|----------|
| 13.1 | `PUT /auth/password` (change password, requires current) | ⬜ | 🔴 Now |
| 13.2 | `DELETE /users/me` (account self-deletion, GDPR) | ⬜ | 🔴 Now |
| 13.3 | Email verification on registration (token-based) | ⬜ | 🟡 Soon |
| 13.4 | `POST /auth/forgot-password` (reset via email token) | ⬜ | 🟡 Soon |
| 13.5 | `POST /auth/reset-password` (confirm reset with token) | ⬜ | 🟡 Soon |

### 14. Advanced API

| # | Task | Status | Priority |
|---|------|--------|----------|
| 14.1 | Filtering and sorting on list endpoints (query params) | ⬜ | 🔴 Now |
| 14.2 | Pagination on all list endpoints (users for admin) | ⬜ | 🔴 Now |
| 14.3 | Resource versioning (ETags / `If-Modified-Since`) | ⬜ | 🟡 Soon |
| 14.4 | Request body validation middleware (against OpenAPI) | ⬜ | 🟡 Soon |
| 14.5 | Bulk operations (batch create/update) | ⬜ | 🔵 Later |
| 14.6 | Full-text search (PostgreSQL `tsvector`) | ⬜ | 🔵 Later |
| 14.7 | Partial responses (field selection) | ⬜ | ⚪ Maybe never |

### 15. Performance & Baselines

| # | Task | Status | Priority |
|---|------|--------|----------|
| 15.1 | Load testing with k6 (establish baselines) | ⬜ | 🔴 Now |
| 15.2 | Profiling with `pprof` (identify bottlenecks) | ⬜ | 🟡 Soon |
| 15.3 | Optimized indexes for frequent queries | ⬜ | 🟡 Soon |
| 15.4 | Connection pooling tuning (GORM pool settings) | ⬜ | 🟡 Soon |
| 15.5 | Cache-Control headers on GET responses | ⬜ | 🔵 Later |

---

## Phase 4 — Horizontal Scaling & Observability

> **Goal**: Replace in-memory adapters with Redis, add production-grade observability,
> and prepare the system for multi-instance deployment.

### 16. Redis & Distributed State

| # | Task | Status | Priority |
|---|------|--------|----------|
| 16.1 | Redis adapter for token blacklist | ⬜ | 🔴 Now |
| 16.2 | Redis adapter for rate limiter (IP + user) | ⬜ | 🔴 Now |
| 16.3 | Redis adapter for login throttler | ⬜ | 🔴 Now |
| 16.4 | Redis health check in `/readyz` endpoint | ⬜ | 🔴 Now |
| 16.5 | Redis-backed session cache (frequent user data) | ⬜ | 🟡 Soon |
| 16.6 | Configurable adapter selection (memory vs Redis via env) | ⬜ | 🟡 Soon |

### 17. Production Observability

| # | Task | Status | Priority |
|---|------|--------|----------|
| 17.1 | Grafana dashboards (latency p50/p95/p99, error rate, throughput) | ⬜ | 🔴 Now |
| 17.2 | Alertmanager rules (error spikes, latency, downtime) | ⬜ | 🔴 Now |
| 17.3 | Log aggregation with Loki + Grafana | ⬜ | 🟡 Soon |
| 17.4 | Distributed tracing with OpenTelemetry | ⬜ | 🟡 Soon |
| 17.5 | Audit log (user action tracking, stored in DB) | ⬜ | 🟡 Soon |
| 17.6 | Health check aggregation (DB + Redis + external deps) | ⬜ | 🟡 Soon |

---

## Phase 5 — Business Features

> **Goal**: Add the features that drive real product value — roles, emails,
> background processing, and richer domain entities.

### 18. Roles & Permissions

| # | Task | Status | Priority |
|---|------|--------|----------|
| 18.1 | `Role` entity (admin, user, readonly) | ⬜ | 🔵 Later |
| 18.2 | Role assignment on registration (default: user) | ⬜ | 🔵 Later |
| 18.3 | RBAC middleware (role-based route protection) | ⬜ | 🔵 Later |
| 18.4 | Admin endpoints (`GET /admin/users`, `PUT /admin/users/:id/role`) | ⬜ | 🔵 Later |
| 18.5 | Permission-based access control (fine-grained) | ⬜ | ⚪ Maybe never |

### 19. Email & Notifications

| # | Task | Status | Priority |
|---|------|--------|----------|
| 19.1 | Email service port (adapter-agnostic) | ⬜ | 🔴 Now |
| 19.2 | SMTP adapter (SendGrid / Mailgun / SES) | ⬜ | 🔴 Now |
| 19.3 | Welcome email on registration | ⬜ | 🟡 Soon |
| 19.4 | Password reset email with secure token | ⬜ | 🟡 Soon |
| 19.5 | Email verification link | ⬜ | 🟡 Soon |
| 19.6 | Notification preferences (opt-in/out per type) | ⬜ | 🔵 Later |

### 20. Background Processing

| # | Task | Status | Priority |
|---|------|--------|----------|
| 20.1 | Task queue port (adapter-agnostic interface) | ⬜ | 🟡 Soon |
| 20.2 | In-process worker pool adapter (goroutines + channels) | ⬜ | 🟡 Soon |
| 20.3 | Async email delivery via task queue | ⬜ | 🟡 Soon |
| 20.4 | Scheduled tasks (token cleanup, expired blacklist purge) | ⬜ | 🟡 Soon |
| 20.5 | Redis-backed task queue (Asynq or similar) | ⬜ | 🔵 Later |

### 21. Domain Enrichment

| # | Task | Status | Priority |
|---|------|--------|----------|
| 21.1 | Enrich `Client` entity (address, phone, tax ID, notes) | ⬜ | 🔴 Now |
| 21.2 | `Invoice` entity (linked to Client, basic CRUD) | ⬜ | 🟡 Soon |
| 21.3 | `Project` entity (linked to Client, status workflow) | ⬜ | 🟡 Soon |
| 21.4 | Domain events (event-driven internal communication) | ⬜ | 🔵 Later |
| 21.5 | Value Object `PhoneNumber` (E.164 validation) | ⬜ | 🔵 Later |
| 21.6 | Value Object `TaxID` (country-aware validation) | ⬜ | 🔵 Later |

---

## Phase 6 — Resilience & Polish

> **Goal**: Production hardening with fault tolerance patterns, data export,
> and advanced API capabilities.

### 22. Resilience Patterns

| # | Task | Status | Priority |
|---|------|--------|----------|
| 22.1 | Configurable timeouts per operation (context propagation) | ⬜ | 🟡 Soon |
| 22.2 | Retry with exponential backoff + jitter | ⬜ | 🟡 Soon |
| 22.3 | Circuit breaker for external services (email, Redis) | ⬜ | 🔵 Later |
| 22.4 | Idempotency keys on write endpoints | ⬜ | 🔵 Later |
| 22.5 | Graceful degradation (fallback to memory if Redis down) | ⬜ | 🔵 Later |
| 22.6 | Feature flags | ⬜ | ⚪ Maybe never |

### 23. Data & Export

| # | Task | Status | Priority |
|---|------|--------|----------|
| 23.1 | Data export: clients to CSV | ⬜ | 🟡 Soon |
| 23.2 | Data export: clients to PDF | ⬜ | 🔵 Later |
| 23.3 | File upload (S3 / MinIO) | ⬜ | 🔵 Later |
| 23.4 | Bulk import from CSV | ⬜ | 🔵 Later |

### 24. API Governance

| # | Task | Status | Priority |
|---|------|--------|----------|
| 24.1 | API versioning strategy (v1 deprecation plan) | ⬜ | 🔵 Later |
| 24.2 | API changelog (public, consumer-facing) | ⬜ | 🔵 Later |
| 24.3 | SDK generation (Go client library) | ⬜ | 🔵 Later |
| 24.4 | Real-time updates (WebSocket or SSE) | ⬜ | 🔵 Later |
| 24.5 | Multi-tenancy | ⬜ | ⚪ Maybe never |

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
  Observability              ████████████████████  5/5   ✅
  Testing                    ████████████████████  6/6   ✅
  CI/CD & DevOps             ████████████████████  6/6   ✅

Phase 2 — Hardening
  Security                   ████████████████████  6/6   ✅
  Database                   ████████████████████  4/4   ✅
  Developer Experience       ████████████████████  6/6   ✅

Phase 3 — Account & API Maturity
  Account Management         ░░░░░░░░░░░░░░░░░░░░  0/5
  Advanced API               ░░░░░░░░░░░░░░░░░░░░  0/7
  Performance & Baselines    ░░░░░░░░░░░░░░░░░░░░  0/5

Phase 4 — Horizontal Scaling & Observability
  Redis & Distributed State  ░░░░░░░░░░░░░░░░░░░░  0/6
  Production Observability   ░░░░░░░░░░░░░░░░░░░░  0/6

Phase 5 — Business Features
  Roles & Permissions        ░░░░░░░░░░░░░░░░░░░░  0/5
  Email & Notifications      ░░░░░░░░░░░░░░░░░░░░  0/6
  Background Processing      ░░░░░░░░░░░░░░░░░░░░  0/5
  Domain Enrichment          ░░░░░░░░░░░░░░░░░░░░  0/6

Phase 6 — Resilience & Polish
  Resilience Patterns        ░░░░░░░░░░░░░░░░░░░░  0/6
  Data & Export              ░░░░░░░░░░░░░░░░░░░░  0/4
  API Governance             ░░░░░░░░░░░░░░░░░░░░  0/5
```

> **Total progress**: 76/76 (Phase 1+2) + 0/59 (Phase 3–6) = 76/135 tasks (~56%)
> **Phases 1 & 2 complete** — solid production foundation, security, and DX.
> **Recommendation**: Start Phase 3 (Account & API Maturity) — highest user-facing impact.

### Suggested execution order

| Order | Section | Why first |
|-------|---------|-----------|
| 1st | 13. Account Management | Users need password change + deletion (GDPR) |
| 2nd | 14. Advanced API | Filtering/sorting is the most requested API feature |
| 3rd | 15. Performance | Establish baselines before adding Redis/complexity |
| 4th | 16. Redis | Required before deploying multiple instances |
| 5th | 17. Observability | Production monitoring before adding business features |
| 6th | 19. Email & Notifications | Enables password reset, verification, and user communication |
| 7th | 20. Background Processing | Async emails, scheduled cleanup tasks |
| 8th | 21. Domain Enrichment | Richer entities drive real product value |
| 9th | 22–24 | Resilience, data export, and API governance — polish and scale |
| Later | 18. Roles & Permissions | When the app becomes multi-user or needs an admin panel |

### Priority Legend

| Flag | Meaning |
|------|---------|
| 🔴 Now | High impact, implement within this phase |
| 🟡 Soon | Important, implement before moving to the next phase |
| 🔵 Later | Low priority, wait until the feature is needed |
| ⚪ Maybe never | Likely unnecessary for this project |
