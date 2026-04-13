# Roadmap — Ductifact Backend

> **Stack**: Go 1.24 · Gin · GORM · PostgreSQL · JWT · Docker · Caddy · GitHub Actions · Prometheus
> **Architecture**: Hexagonal (Ports & Adapters)
> **Started**: March 2026

---

## Current Status

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
| 2.3 | `AuthService` (register, login) | ✅ |

### 3. HTTP Infrastructure

| # | Task | Status |
|---|------|--------|
| 3.1 | Gin router + versioned `/v1/` | ✅ |
| 3.2 | Handlers (User, Client, Auth, Health) | ✅ |
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
| 9.6 | Block `/metrics`, `/health`, `/docs` from internet | ✅ |

---

## Next Steps

### 10. Security

| # | Task | Status | Priority |
|---|------|--------|----------|
| 10.1 | Refresh tokens (JWT rotation) | ✅ | — |
| 10.2 | `POST /auth/refresh` endpoint | ✅ | — |
| 10.3 | Rate limiting (per IP and per user) | ✅ | — |
| 10.4 | Login brute-force protection | ✅ | — |
| 10.5 | Security headers middleware (HSTS, X-Frame, CSP) | ✅ | — |
| 10.6 | `POST /auth/logout` endpoint (token blacklist) | ✅ | — |

### 11. Database

| # | Task | Status | Priority |
|---|------|--------|----------|
| 11.1 | Versioned migrations (`golang-migrate`) | ✅ | — |
| 11.2 | Development data seeders | ✅ | — |
| 11.3 | Soft delete (`deleted_at` logical deletion) | ✅ | — |
| 11.4 | Optimized indexes for frequent queries | ⬜ | 🔵 Later |
| 11.5 | Connection pooling tuning | ⬜ | 🔵 Later |
| 11.6 | Automated database backups (`pg_dump` + offsite) | ✅ | — |

### 12. Developer Experience

| # | Task | Status | Priority |
|---|------|--------|----------|
| 12.1 | Pre-commit hooks (auto lint + format) | ✅ | — |
| 12.2 | Dependabot / Renovate (dependency updates) | ✅ | — |
| 12.3 | Automated changelog (conventional commits) | ✅ | — |
| 12.4 | Makefile targets for all operations | ✅ | — |
| 12.5 | Hot reload in development (`air`) | ✅ | — |
| 12.6 | Coverage report (HTML) | ✅ | — |

### 13. Advanced API

| # | Task | Status | Priority |
|---|------|--------|----------|
| 13.1 | Pagination on list endpoints | ✅ | — |
| 13.2 | Filtering and sorting (query params) | ⬜ | 🔵 Later |
| 13.3 | Resource versioning (ETags / `If-Modified-Since`) | ⬜ | 🔵 Later |
| 13.4 | Bulk operations (batch create/update) | ⬜ | 🔵 Later |
| 13.5 | Partial responses (field selection) | ⬜ | ⚪ Maybe never |
| 13.6 | Full-text search | ⬜ | 🔵 Later |

### 14. Caching & Performance

| # | Task | Status | Priority |
|---|------|--------|----------|
| 14.1 | Load testing with k6 or Vegeta (baselines) | ⬜ | 🔵 Later |
| 14.2 | Profiling with `pprof` | ⬜ | 🔵 Later |
| 14.3 | Redis cache (sessions, frequent data) | ⬜ | 🔵 Later |
| 14.4 | Cache-Control headers on responses | ⬜ | 🔵 Later |

### 15. Advanced Observability

| # | Task | Status | Priority |
|---|------|--------|----------|
| 15.1 | Grafana dashboards (latency, errors, throughput) | ⬜ 🔵 Later |
| 15.2 | Alerts (Alertmanager / Grafana alerts) | ⬜ 🔵 Later |
| 15.3 | Log aggregation (Loki + Grafana) | ⬜ | 🔵 Later |
| 15.4 | Distributed tracing with OpenTelemetry | ⬜ | 🔵 Later |
| 15.5 | Audit log (user action tracking) | ⬜ | 🔵 Later |

### 16. Business Features

| # | Task | Status | Priority |
|---|------|--------|----------|
| 16.1 | Email service (verification, password reset) | ⬜ | 🔵 Later |
| 16.2 | Background jobs / task queue (cron, async) | ⬜ | 🔵 Later |
| 16.3 | Roles and permissions (RBAC) | ⬜ | 🔵 Later |
| 16.4 | File upload (S3 / MinIO) | ⬜ | 🔵 Later |
| 16.5 | Notifications (WebSocket or SSE) | ⬜ | 🔵 Later |
| 16.6 | Data export (CSV, PDF) | ⬜ | 🔵 Later |
| 16.7 | Multi-tenancy | ⬜ | ⚪ Maybe never |

### 17. Resilience & Advanced Patterns

| # | Task | Status | Priority |
|---|------|--------|----------|
| 17.1 | Configurable timeouts per operation | ⬜ | 🔵 Later |
| 17.2 | Retry with exponential backoff | ⬜ | 🔵 Later |
| 17.3 | Circuit breaker (for external services) | ⬜ | 🔵 Later |
| 17.4 | Idempotency keys on write endpoints | ⬜ | 🔵 Later |
| 17.5 | Feature flags | ⬜ | ⚪ Maybe never |

---

## Summary

```
Domain & Entities          ████████████████████  6/6   ✅
Use Cases                  ████████████████████  3/3   ✅
HTTP Infrastructure        ████████████████████  10/10 ✅
Authentication             ████████████████████  5/5   ✅
Persistence                ████████████████████  4/4   ✅
API Contracts              ████████████████████  5/5   ✅
Observability              ████████████████████  5/5   ✅
Testing                    ████████████████████  6/6   ✅
CI/CD & DevOps             ████████████████████  6/6   ✅
Security                   ████████████████████  6/6   ✅
Database                   ████████████████░░░░  4/6
Developer Experience       ██████████░░░░░░░░░░  4/6
Advanced API               ░░░░░░░░░░░░░░░░░░░░  0/6
Caching & Performance      ░░░░░░░░░░░░░░░░░░░░  0/4
Advanced Observability     ░░░░░░░░░░░░░░░░░░░░  0/5
Business Features          ░░░░░░░░░░░░░░░░░░░░  0/7
Resilience                 ░░░░░░░░░░░░░░░░░░░░  0/5
```

> **Total progress**: 59/87 tasks completed (~68%)
> **Recommendation**: Sections 10 (Security) and 11 (Database) have the highest immediate impact.

### Priority Legend

| Flag | Meaning |
|------|---------|
| 🔴 Now | High impact, implement immediately |
| 🟡 Soon | Important, implement before scaling |
| 🔵 Later | Low priority, wait until needed |
| ⚪ Maybe never | Likely unnecessary for this project |
