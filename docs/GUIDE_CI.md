# Guía de CI — GitHub Actions

> **Fase 7.1 + 7.2 del Roadmap**
> Automatizar la validación del código en cada Pull Request.

---

## 1. ¿Qué es CI y por qué lo necesitas?

**CI (Continuous Integration)** significa que cada vez que alguien abre un Pull Request (o hace push a `main` o `release`), un servidor ejecuta automáticamente:

1. **Análisis estático** — detecta errores sin ejecutar el código
2. **Linter** — verifica estilo y buenas prácticas
3. **Unit tests** — rápidos, sin dependencias externas
4. **Integration tests** — con una DB Postgres real
5. **Contract tests** — valida que la API cumple el OpenAPI spec
6. **E2E tests** — flujo HTTP completo contra la API corriendo

**¿El resultado?** Si algún paso falla, GitHub bloquea el merge. Nunca más rompes `main` (ni `release`) por accidente.

> **Contexto**: según el `CONTRIBUTING.md`, `main` despliega a **staging** y los **tags** despliegan a **producción**. CI corre en ambas ramas porque los hotfixes van como PRs a `release` y también necesitan validación. Ve `CONTRIBUTING.md` sección 7 para el flujo completo.

### ¿Cómo funciona GitHub Actions?

GitHub Actions lee archivos YAML en `.github/workflows/`. Cada archivo define un **workflow** (flujo de trabajo) que se ejecuta cuando ocurre un evento (push, PR, etc.).

Conceptos clave:
- **Workflow**: el archivo YAML completo (ej: `ci.yml`)
- **Job**: un grupo de pasos que corren en una máquina virtual. Los jobs pueden correr en paralelo entre sí.
- **Step**: una acción individual dentro de un job (ej: "instalar Go", "ejecutar tests")
- **Service container**: un contenedor Docker que corre junto a tu job (ej: Postgres para los integration tests)

```
Workflow (ci.yml)
  ├── Job: lint        (solo Go, sin DB)     ─┐
  ├── Job: unit-tests  (solo Go, sin DB)      ├── corren en paralelo
  ├── Job: integration (Go + Postgres)        │
  ├── Job: contract    (Go + Postgres + API)  │
  └── Job: e2e         (Go + Postgres + API) ─┘
```

---

## 2. Estructura de archivos

`backend/` es un **repositorio independiente** (no un monorepo). `.github/` va en la raíz del repo:

```
backend/                             ← raíz del repositorio Git
├── .github/
│   └── workflows/
│       ├── ci.yml                   ← el workflow de CI
│       └── cd.yml                   ← workflow de despliegue (ver GUIDE_CD.md)
├── .dockerignore                    ← NUEVO: excluir archivos del build Docker
├── Dockerfile
├── go.mod
├── cmd/
├── internal/
├── test/
└── ...
```

> **Nota**: la infraestructura de producción (docker-compose.prod.yml, nginx.conf, prometheus.yml) vive en un repositorio separado `infra/`. Ver `GUIDE_CD.md` para más detalles.

---

## 3. El workflow: `.github/workflows/ci.yml`

```yaml
name: CI

# ─── Cuándo se ejecuta ──────────────────────────────────────
# - PRs to main: new features and fixes (deploys to staging after merge)
# - PRs to release: hotfixes for production (see CONTRIBUTING.md §7)
# - Push to main/release: final validation after merge
on:
  push:
    branches: [main, release]
  pull_request:
    branches: [main, release]

# ─── Variables compartidas ───────────────────────────────────
env:
  GO_VERSION: "1.24"
  DB_USER: ci_user
  DB_PASSWORD: ci_password
  DB_NAME: ci_db
  DB_HOST: localhost
  DB_PORT: "5432"
  APP_PORT: "8080"
  JWT_SECRET: ci-test-secret-at-least-32-characters-long
  CORS_ORIGINS: "*"
  CONTRACT_VERSION: "1.0.0"
  LOG_LEVEL: error
  LOG_FORMAT: text

# ─── Jobs ────────────────────────────────────────────────────
jobs:

  # ── 1. Lint + análisis estático ────────────────────────────
  lint:
    name: Lint & Vet
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: Go vet (static analysis)
        run: go vet ./...

      - name: Install golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest

  # ── 2. Unit tests ─────────────────────────────────────────
  unit-tests:
    name: Unit Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: Run unit tests
        run: go test -v -race -count=1 ./test/unit/...

  # ── 3. Integration tests (necesitan Postgres) ─────────────
  integration-tests:
    name: Integration Tests
    runs-on: ubuntu-latest

    # Service container: Postgres corre junto al job
    services:
      postgres:
        image: postgres:15-alpine
        env:
          POSTGRES_USER: ${{ env.DB_USER }}
          POSTGRES_PASSWORD: ${{ env.DB_PASSWORD }}
          POSTGRES_DB: ${{ env.DB_NAME }}
        ports:
          - 5432:5432
        options: >-
          --health-cmd "pg_isready -U ci_user"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: Initialize database schema
        run: PGPASSWORD=${{ env.DB_PASSWORD }} psql -h localhost -U ${{ env.DB_USER }} -d ${{ env.DB_NAME }} -f init.sql

      - name: Run integration tests
        run: go test -v -race -count=1 ./test/integration/...

  # ── 4. Contract tests (necesitan Postgres + API corriendo) ─
  contract-tests:
    name: Contract Tests
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15-alpine
        env:
          POSTGRES_USER: ${{ env.DB_USER }}
          POSTGRES_PASSWORD: ${{ env.DB_PASSWORD }}
          POSTGRES_DB: ${{ env.DB_NAME }}
        ports:
          - 5432:5432
        options: >-
          --health-cmd "pg_isready -U ci_user"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: Initialize database schema
        run: PGPASSWORD=${{ env.DB_PASSWORD }} psql -h localhost -U ${{ env.DB_USER }} -d ${{ env.DB_NAME }} -f init.sql

      - name: Build and start API server
        run: |
          go build -o bin/api ./cmd/api
          ./bin/api &
          sleep 3

      - name: Run contract tests
        run: go test -v -race -count=1 ./test/contract/...

  # ── 5. E2E tests (necesitan Postgres + API corriendo) ─────
  e2e-tests:
    name: E2E Tests
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15-alpine
        env:
          POSTGRES_USER: ${{ env.DB_USER }}
          POSTGRES_PASSWORD: ${{ env.DB_PASSWORD }}
          POSTGRES_DB: ${{ env.DB_NAME }}
        ports:
          - 5432:5432
        options: >-
          --health-cmd "pg_isready -U ci_user"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: Initialize database schema
        run: PGPASSWORD=${{ env.DB_PASSWORD }} psql -h localhost -U ${{ env.DB_USER }} -d ${{ env.DB_NAME }} -f init.sql

      - name: Build and start API server
        run: |
          go build -o bin/api ./cmd/api
          ./bin/api &
          sleep 3

      - name: Run E2E tests
        run: go test -v -race -count=1 ./test/e2e/...
```

---

## 4. Explicación detallada de cada sección

### 4.1 `on:` — Cuándo se ejecuta

```yaml
on:
  push:
    branches: [main, release]
  pull_request:
    branches: [main, release]
```

- **`push` a `main`**: cuando alguien hace merge de un PR, CI corre como última validación. Después CD despliega a **staging**.
- **`push` a `release`**: cuando se mergea un hotfix, CI corre. Después se tagea manualmente y CD despliega a **producción**.
- **`pull_request` a `main` o `release`**: cuando abres o actualizas un PR, CI corre y muestra el resultado en la UI de GitHub (✅ o ❌).

> Esto es coherente con el flujo de `CONTRIBUTING.md`: features van a `main` (staging), hotfixes van a `release` (producción). CI valida ambos.

### 4.2 `env:` — Variables de entorno

Las variables definidas a nivel de workflow están disponibles en **todos los jobs**. Son las mismas que tu `.env` pero con valores de CI (contraseñas simples, log level bajo para no ensuciar el output).

`LOG_LEVEL: error` — en CI no quieres ver los logs de info/debug. Solo errores reales.

### 4.3 `services:` — Service containers

```yaml
services:
  postgres:
    image: postgres:15-alpine
    env:
      POSTGRES_USER: ${{ env.DB_USER }}
    ports:
      - 5432:5432
    options: >-
      --health-cmd "pg_isready -U ci_user"
      --health-interval 10s
      --health-timeout 5s
      --health-retries 5
```

GitHub Actions levanta un contenedor Postgres **dentro del runner**. El `options` con `--health-*` hace que el job espere a que Postgres esté listo antes de ejecutar los steps. Sin esto, los tests podrían fallar porque Postgres aún no aceptaba conexiones.

`ports: - 5432:5432` hace que el contenedor sea accesible en `localhost:5432` del runner, exactamente como en tu máquina local.

### 4.4 Database schema initialization

```yaml
- name: Initialize database schema
  run: PGPASSWORD=${{ env.DB_PASSWORD }} psql -h localhost -U ${{ env.DB_USER }} -d ${{ env.DB_NAME }} -f init.sql
```

En local, tu `docker-compose.dev.yml` monta `init.sql` automáticamente con el volumen `./init.sql:/docker-entrypoint-initdb.d/init.sql`. Pero los service containers de GitHub Actions **no soportan volúmenes**. Así que ejecutamos `psql` manualmente para crear las tablas.

> **Nota**: `psql` ya viene instalado en los runners de Ubuntu de GitHub Actions. No necesitas instalar nada extra.

### 4.5 API server para contract y E2E tests

```yaml
- name: Build and start API server
  run: |
    go build -o bin/api ./cmd/api
    ./bin/api &
    sleep 3
```

Compilamos el binario, lo lanzamos en background (`&`) y esperamos 3 segundos para que esté listo. Los contract y E2E tests necesitan hacer requests HTTP reales contra la API corriendo.

### 4.6 Test flags

```
go test -v -race -count=1 ./test/unit/...
```

| Flag | Qué hace |
|------|----------|
| `-v` | Verbose — muestra cada test que corre (útil para diagnosticar fallos en CI) |
| `-race` | Activa el **race detector** de Go — detecta accesos concurrentes no protegidos |
| `-count=1` | Desactiva el cache de tests — en CI quieres que siempre corran fresh |

---

## 5. Docker optimizado (Fase 7.2)

### 5.1 `.dockerignore`

Crea `.dockerignore` en la raíz del repo para que Docker no copie archivos innecesarios al contexto de build. Esto hace que `docker build` sea más rápido.

```dockerignore
# Test files — not needed in production image
test/
*_test.go

# Documentation
docs/
README.md
CONTRIBUTING.md

# Development files
.env
.env.example
.air.toml
tmp/
bin/

# IDE and OS files
.vscode/
.idea/
.DS_Store
*.swp
*.swo
*~

# CI/CD
.github/

# Build artifacts
coverage.out
coverage.html
*.log
```

### 5.2 Dockerfile optimizado

Tu Dockerfile actual ya usa **multi-stage build**, que es lo correcto. Un par de mejoras:

```dockerfile
# ── Stage 1: Build ──────────────────────────────────────────
FROM docker.io/library/golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependency files FIRST (this layer gets cached if go.mod/go.sum don't change)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code (this layer invalidates when code changes)
COPY . .

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o app ./cmd/api

# ── Stage 2: Runtime ────────────────────────────────────────
FROM docker.io/library/alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/app .

ARG APP_PORT=8080
EXPOSE ${APP_PORT}

# Run as non-root user (security best practice)
RUN adduser -D -g '' appuser
USER appuser

CMD ["./app"]
```

**¿Qué cambió?**

| Cambio | Por qué |
|--------|---------|
| `-ldflags="-s -w"` | Elimina los símbolos de debug del binario. Reduce el tamaño ~30% (ej: 15MB → 10MB). No afecta al funcionamiento. |
| `RUN adduser ... / USER appuser` | El contenedor corre como usuario sin privilegios. Si alguien explota una vulnerabilidad, no tiene acceso root. Es una buena práctica de seguridad estándar. |
| Sin `git` en builder | Tu código no necesita git en la fase de build. `go mod download` funciona sin él porque los módulos se descargan por HTTPS. Ahorra ~10MB en la capa del builder. |

### 5.3 Build cache de Docker

El truco clave ya lo tienes: copiar `go.mod` y `go.sum` **antes** que el código fuente. Docker cachea cada capa, así que si solo cambias código Go (no dependencias), la capa de `go mod download` se reutiliza y el build es más rápido.

```
COPY go.mod go.sum ./     ← si no cambian, Docker usa cache de aquí
RUN go mod download       ← esta capa se cachea

COPY . .                  ← siempre se invalida cuando cambias código
RUN ... go build ...      ← se re-ejecuta
```

En CI, GitHub Actions cachea las capas de Docker automáticamente si usas `docker/build-push-action`. Lo configuraremos en la guía de CD.

---

## 6. Diagrama visual del flujo

```
Developer abre PR (a main o release)
         │
         ▼
   GitHub Actions CI
         │
    ┌────┼────┬──────────┬──────────────┐
    ▼    ▼    ▼          ▼              ▼
  lint  unit  integration  contract     e2e
   │     │    │ (+ Postgres) │(+Postgres+API) │(+Postgres+API)
   │     │    │          │              │
   └─────┴────┴──────────┴──────────────┘
         │
    Todos pasan?
    ┌────┴────┐
    ▼         ▼
   ✅ PR     ❌ PR
  mergeable  blocked
    │
    ▼
  Merge → CD (ver GUIDE_CD.md)
    ├── main    → staging
    └── release → producción (tras tag manual)
```

---

## 7. Verificar que funciona

Una vez que hagas push del archivo `ci.yml`:

1. Crea una rama: `git checkout -b ci/add-github-actions`
2. Haz commit y push: `git push origin ci/add-github-actions`
3. Abre un PR en GitHub hacia `main`
4. Ve a la pestaña **Actions** — verás los 5 jobs corriendo en paralelo
5. Si alguno falla, haz clic para ver el log detallado

### Errores comunes

| Error | Causa | Solución |
|-------|-------|----------|
| `PGPASSWORD: command not found` | psql no está en PATH | Los runners de Ubuntu ya lo incluyen, verifica con `which psql` |
| `connection refused` en integration tests | Postgres no está listo | Verifica que `options` tiene `--health-*` y que el job espera al healthcheck |
| `./bin/api &` no arranca | Variable de entorno faltante | Revisa que todas las env vars requeridas están en la sección `env:` del workflow |
| Tests pasan localmente, fallan en CI | Race condition | El flag `-race` detecta bugs que en local no se manifiestan. El error te dice la línea exacta. |

---

## 8. Próximos pasos

Con el CI funcionando, ve a `docs/GUIDE_CD.md` para configurar:
- Despliegue automático a **staging** desde `main`
- Despliegue a **producción** cuando se pushea un tag `v*` desde la rama `release`

> Para entender el flujo completo de branches, releases y hotfixes, ve `CONTRIBUTING.md` §6 y §7.
