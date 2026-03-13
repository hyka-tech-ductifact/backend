# Checklist CI — Fase 7.1 + 7.2

## Workflow (`ci.yml`)

- [x] **Triggers**: push + PR en `main` y `release` + `workflow_call` para CD
- [x] **Job: Lint & Vet** — análisis estático + golangci-lint
- [x] **Job: Unit Tests** — sin dependencias externas, con `-race`
- [x] **Job: Integration Tests** — con Postgres real via `docker-compose.dev.yml`
- [x] **Job: Contract Tests** — checkout `contracts/` pinned a `CONTRACT_VERSION` + `redocly bundle` + API corriendo
- [x] **Job: E2E Tests** — flujo HTTP completo contra API + Postgres
- [x] **Job: Docker Smoke** — build imagen + start + health check `/api/v1/health`
- [x] **6 jobs corren en paralelo**
- [x] **Makefile como single source of truth** — CI usa los mismos targets que local

## Protección del repositorio (GitHub Settings)

- [ ] Branch ruleset en `main` + `release` → require PR + approvals + status checks
- [ ] Tag ruleset en `v*` → restrict creations + deletions
- [ ] **CODEOWNERS** creado → `.github/CODEOWNERS`
- [ ] Require Code Owner review → activado en branch ruleset
- [ ] Status checks marcados como **required** (los 6 jobs)
- [ ] Force push bloqueado → en `main`, `release` y tags `v*`
- [ ] Environment `production` → required reviewers activado
- [ ] Secretos de CD configurados → en Settings → Secrets

## Archivos creados/actualizados

- [x] `.github/workflows/ci.yml`
- [x] `.env.example` con `CONTRACT_VERSION=0.1.0`
- [x] Tag `v0.1.0` en repo `contracts/`
- [x] `docs/GUIDE_CI.md` actualizada
