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
│   ├── CODEOWNERS                   ← protege archivos sensibles (requiere aprobación)
│   └── workflows/
│       ├── ci.yml                   ← el workflow de CI
│       └── cd.yml                   ← workflow de despliegue (ver GUIDE_CD.md)
├── .dockerignore                    ← excluir archivos del build Docker
├── Dockerfile
├── go.mod
├── cmd/
├── internal/
├── test/
└── ...
```

> **Nota**: la infraestructura de producción (docker-compose.prod.yml, nginx.conf, prometheus.yml) vive en un repositorio separado `infra/`. Ver `GUIDE_CD.md` para más detalles.

### 2.1 ¿Por qué los workflows viven aquí y no en `infra/`?

Es una pregunta legítima: si alguien puede editar `ci.yml` en el mismo repo que el código, ¿podría saltarse las validaciones? Hay tres estrategias posibles:

| Estrategia | Dónde viven los `.yml` | Seguridad | Facilidad | Mantenibilidad |
|---|---|---|---|---|
| **Todo en el repo** | `backend/.github/workflows/` | Media | Alta | Media |
| **Repo separado** | `infra/.github/workflows/` | Alta | Baja | Baja |
| **Híbrido** (recomendado a escala) | Lógica en repo central, caller en cada repo | Alta | Alta | Alta |

**Estrategia 1 — Todo en el mismo repo** (la que usamos):
- El pipeline viaja con el código: si cambias el código, puedes ajustar el CI en el mismo PR
- El riesgo de manipulación se mitiga con **CODEOWNERS** + **branch protection** (ver sección 8)
- Es lo más práctico para equipos pequeños y proyectos en fase temprana

**Estrategia 2 — Repo separado (`infra/`)**:
- Los devs no pueden tocar los pipelines
- Pero los workflows de GitHub Actions **deben vivir en el repo donde se disparan** (`.github/workflows/`), así que necesitarías `repository_dispatch` o `workflow_dispatch` para triggerear desde otro repo — añade complejidad innecesaria
- Cambiar código y pipeline requiere 2 PRs en 2 repos, lo que ralentiza el desarrollo

**Estrategia 3 — Híbrida** (cuando escales):
- La lógica crítica (build, test, deploy) se centraliza en un repo `platform-workflows/` como **reusable workflows**
- Cada repo solo tiene un "caller" mínimo que llama al workflow centralizado
- Los devs solo pueden parametrizar (versión de Go, flags), **no cambiar el comportamiento**

```yaml
# Ejemplo: backend/.github/workflows/ci.yml (caller mínimo)
name: CI
on:
  pull_request:
    branches: [main]

jobs:
  ci:
    uses: org/platform-workflows/.github/workflows/reusable-go-ci.yml@v1
    with:
      go-version: "1.24"
    secrets: inherit
```

```yaml
# Ejemplo: platform-workflows/.github/workflows/reusable-go-ci.yml (source of truth)
name: Reusable Go CI
on:
  workflow_call:
    inputs:
      go-version:
        type: string
        default: "1.24"

jobs:
  lint-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ inputs.go-version }}
      - run: make lint
      - run: make test
```

> **Para este proyecto**: usamos la estrategia 1 (workflows en el mismo repo) porque es pragmática y suficiente. La protección la ponemos con CODEOWNERS y branch protection rules (sección 8). Si el día de mañana tienes 5+ microservicios, migras a la estrategia híbrida.

### 2.2 Estructura del repo `infra/` (separado)

El repo `infra/` contiene **solo infraestructura de despliegue**, no pipelines de CI:

```
infra/                               ← repositorio independiente
├── docker-compose.staging.yml       ← orquestación para staging
├── docker-compose.prod.yml          ← orquestación para producción
├── nginx/
│   └── nginx.conf                   ← reverse proxy / TLS termination
├── monitoring/
│   ├── prometheus.yml               ← métricas
│   └── grafana/
│       └── dashboards/
├── scripts/
│   ├── deploy.sh                    ← script de despliegue
│   └── rollback.sh                  ← rollback manual
└── README.md
```

**¿Qué va en cada repo?**

| Qué | Dónde | Por qué |
|-----|-------|---------|
| Código fuente Go | `backend/` | Es el producto, lo cambian los devs |
| CI workflows (`.github/workflows/ci.yml`) | `backend/` | Validan el código, viajan con él |
| CD workflows (`.github/workflows/cd.yml`) | `backend/` | Se triggerean por push/tag en este repo |
| Dockerfile | `backend/` | Define cómo se construye la imagen, depende del código |
| docker-compose para **dev local** | `backend/` | Los devs lo usan día a día |
| docker-compose para **staging/prod** | `infra/` | Configuración de infra, no de código |
| nginx, prometheus, grafana | `infra/` | Infraestructura transversal, no acoplada al backend |
| Scripts de deploy/rollback | `infra/` | Operaciones, no desarrollo |
| Contratos OpenAPI | `contracts/` | Compartidos entre frontend y backend |

El principio es simple: **si lo cambia un dev para añadir una feature, va en `backend/`**. **Si lo cambia un ops/SRE para configurar infraestructura, va en `infra/`**.

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
  workflow_call:                     # allows CD to re-run CI before deploying

# ─── Variables compartidas ───────────────────────────────────
# Postgres runs via docker-compose.dev.yml (single source of truth
# for image version, healthcheck, init.sql, and ports).
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

      - name: Install gotestsum
        run: go install gotest.tools/gotestsum@latest

      - name: Run unit tests
        run: make test-unit

  # ── 3. Integration tests (necesitan Postgres) ─────────────
  integration-tests:
    name: Integration Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Start Postgres
        run: docker compose -f docker-compose.dev.yml up -d --wait postgres

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: Install gotestsum
        run: go install gotest.tools/gotestsum@latest

      - name: Run integration tests
        run: make test-integration

  # ── 4. Contract tests (necesitan Postgres + API corriendo) ─
  contract-tests:
    name: Contract Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Checkout contracts repo
        uses: actions/checkout@v4
        with:
          repository: ${{ github.repository_owner }}/contracts
          ref: v${{ env.CONTRACT_VERSION }}
          path: contracts

      - name: Start Postgres
        run: docker compose -f docker-compose.dev.yml up -d --wait postgres

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: Install gotestsum
        run: go install gotest.tools/gotestsum@latest

      - name: Build and start API server
        run: |
          make app-build
          ./bin/api &
          sleep 3

      - name: Run contract tests
        run: make test-contract

  # ── 5. E2E tests (necesitan Postgres + API corriendo) ─────
  e2e-tests:
    name: E2E Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Start Postgres
        run: docker compose -f docker-compose.dev.yml up -d --wait postgres

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: Install gotestsum
        run: go install gotest.tools/gotestsum@latest

      - name: Build and start API server
        run: |
          make app-build
          ./bin/api &
          sleep 3

      - name: Run E2E tests
        run: make test-e2e
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

### 4.3 `docker compose` — Postgres como single source of truth

```yaml
- name: Start Postgres
  run: docker compose -f docker-compose.dev.yml up -d --wait postgres
```

En vez de usar `services:` de GitHub Actions (que obliga a duplicar imagen, healthcheck, puertos y env vars), usamos directamente `docker-compose.dev.yml`. Ventajas:

- **Una sola fuente de verdad**: versión de Postgres, healthcheck, puertos e `init.sql` se definen en un solo lugar.
- **Cero divergencia**: cualquier cambio en `docker-compose.dev.yml` aplica automáticamente a CI.
- **`init.sql` se monta automáticamente** vía el volumen `./init.sql:/docker-entrypoint-initdb.d/init.sql`. No necesitas un step `psql` separado.

El flag `--wait` espera a que los healthchecks definidos en el compose estén healthy antes de continuar. `docker compose` (v2) viene preinstalado en los runners `ubuntu-latest` de GitHub Actions.

Las variables de entorno del workflow (`DB_USER`, `DB_PASSWORD`, etc.) se heredan automáticamente al shell donde corre `docker compose`, así que las interpolaciones `${DB_USER}` del compose funcionan sin configuración extra.

> **¿Por qué no `services:` de Actions?** Los service containers no soportan bind mounts (volúmenes), lo que obliga a inicializar el schema manualmente con `psql`. Además, cualquier cambio en la configuración de Postgres requiere actualizar dos archivos: `docker-compose.dev.yml` y `ci.yml`. Con `docker compose` en CI, el compose es la única fuente de verdad.

### 4.4 API server para contract y E2E tests

```yaml
- name: Build and start API server
  run: |
    make app-build
    ./bin/api &
    sleep 3
```

Compilamos el binario con `make app-build` (el mismo target que usas en local), lo lanzamos en background (`&`) y esperamos 3 segundos para que esté listo. Los contract y E2E tests necesitan hacer requests HTTP reales contra la API corriendo.

### 4.5 Makefile como single source of truth

```yaml
# En ci.yml
- name: Run unit tests
  run: make test-unit
```

```makefile
# En Makefile
test-unit:
	gotestsum --format pkgname -- -race -count=1 ./test/unit/...
```

CI usa los mismos targets de `make` que el desarrollador en local. Esto garantiza que **los flags, rutas y herramientas sean siempre idénticos**. Si cambias cómo se ejecutan los tests (ej: añades `-timeout 30s`), lo cambias en el Makefile y CI se actualiza automáticamente.

| Flag | Qué hace |
|------|----------|
| `-race` | Activa el **race detector** de Go — detecta accesos concurrentes no protegidos |
| `-count=1` | Desactiva el cache de tests — cada ejecución corre fresh |

> **Nota**: CI instala `gotestsum` con `go install gotest.tools/gotestsum@latest` porque los runners de GitHub Actions no lo incluyen por defecto. En local ya lo tienes por `make deps`.

### 4.6 Checkout del repo `contracts/` para contract tests

Los contract tests validan las respuestas de la API contra el spec OpenAPI (`bundled.yaml`), que vive en el repo separado `contracts/`. En local funciona porque ambos repos son carpetas hermanas en tu workspace:

```
workspace/
├── backend/          ← repo Git
├── contracts/        ← repo Git (contiene openapi/bundled.yaml)
└── infra/            ← repo Git
```

El código usa la ruta relativa `../../../contracts/openapi/bundled.yaml` desde `test/contract/`. En CI, solo se hace checkout de `backend/`, así que esa ruta no existe.

**Solución**: hacer un segundo `actions/checkout` del repo `contracts/` en la misma ubicación relativa:

```yaml
- uses: actions/checkout@v4                    # checkout backend/ (default)

- name: Checkout contracts repo
  uses: actions/checkout@v4
  with:
    repository: ${{ github.repository_owner }}/contracts
    ref: v${{ env.CONTRACT_VERSION }}          # pin to the version backend implements
    path: contracts                            # inside workspace root
```

**¿Por qué `ref: v${{ env.CONTRACT_VERSION }}`?** El backend declara qué versión del contrato implementa vía `CONTRACT_VERSION` (ej: `1.0.0`). CI hace checkout del tag `v1.0.0` del repo `contracts/`. Así los tests validan que el backend cumple **exactamente la versión que dice implementar**, no la última versión del spec (que podría tener campos nuevos que el backend aún no soporta).

El flujo queda así:
```
contracts/ repo:  v1.0.0 ──── v1.1.0 ──── v2.0.0
                    │
backend/ repo:    CONTRACT_VERSION="1.0.0"
                    │
CI:               checkout contracts@v1.0.0 → run contract tests
```

Cuando se actualiza el contrato:
1. El equipo publica una nueva versión en `contracts/` (ej: tag `v1.1.0`)
2. Un dev actualiza `CONTRACT_VERSION` en el backend a `"1.1.0"`
3. Ajusta el código para cumplir los nuevos campos/endpoints
4. CI descarga automáticamente `contracts@v1.1.0` y valida

> **Prerequisito**: el repo `contracts/` debe tener tags semver con prefijo `v` (ej: `v1.0.0`). Si usas otro formato de tags, ajusta el `ref:` del checkout.

`actions/checkout` permite hacer checkout de **múltiples repos** en el mismo job. El segundo checkout no sobrescribe el primero porque usa `path: contracts` — lo coloca dentro del workspace.

> **Importante**: GitHub Actions **no permite** hacer checkout fuera del workspace (`path: ../contracts` falla con un error de seguridad). Por eso se coloca dentro del repo. El código de `DefaultSpecPath()` ya tiene un fallback que busca `../../contracts/openapi/bundled.yaml` desde `test/contract/`, que resuelve a la raíz del repo — exactamente donde queda `contracts/`.

`${{ github.repository_owner }}` se resuelve al owner del repo actual (tu usuario u organización), así que no necesitas hardcodear el nombre.

**¿Y si el repo `contracts/` es privado?** El `GITHUB_TOKEN` por defecto solo tiene acceso al repo actual. Para repos privados del mismo owner, necesitas un **Personal Access Token (PAT)** o un **GitHub App token** con acceso a ambos repos:

```yaml
- name: Checkout contracts repo
  uses: actions/checkout@v4
  with:
    repository: ${{ github.repository_owner }}/contracts
    ref: v${{ env.CONTRACT_VERSION }}
    path: contracts
    token: ${{ secrets.CONTRACTS_PAT }}        # only needed if contracts is private
```

Crea el secreto `CONTRACTS_PAT` en **Settings → Secrets → Actions** con un PAT que tenga scope `repo` (o `contents:read` si usas fine-grained tokens).

> **Fallback alternativo**: el código también soporta la variable de entorno `CONTRACT_SPEC_PATH`. Si prefieres no hacer checkout del repo, podrías descargar solo el archivo con `curl` y apuntar la variable. Pero el checkout es más robusto porque garantiza que siempre usas la versión correcta del spec.

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

## 8. Proteger el repositorio en GitHub

De nada sirve un CI perfecto si alguien puede saltárselo. Estas son las reglas que debes configurar en **GitHub → Settings** para que el pipeline sea una barrera real.

### 8.1 Branch Protection Rules

Ve a **Settings → Branches → Add rule** para `main` y `release`:

| Regla | Qué activas | Por qué |
|-------|------------|---------|
| **Require a pull request before merging** | ✅ | Nadie puede hacer push directo a `main`/`release`. Todo pasa por PR. |
| **Require approvals** (mínimo 1) | ✅ | Al menos un compañero revisa el código antes del merge. |
| **Dismiss stale pull request approvals** | ✅ | Si alguien aprueba y luego cambias el código, la aprobación se invalida. Evita el truco de "apruébame y luego subo otra cosa". |
| **Require status checks to pass** | ✅ | El merge se bloquea hasta que CI pase. Marca como **required** los 6 jobs: `Lint & Vet`, `Unit Tests`, `Integration Tests`, `Contract Tests`, `E2E Tests`, `Docker Build`. |
| **Require branches to be up to date** | ✅ | El PR debe estar actualizado con la rama base. Evita conflictos silenciosos. |
| **Require conversation resolution** | ✅ | Todos los comentarios del code review deben resolverse antes de mergear. |
| **Do not allow bypassing** | ✅ | Ni siquiera los admins pueden saltarse las reglas. (Desactiva esto solo si estás seguro.) |

> **Importante**: cuando actives "Require status checks to pass", los nombres de los jobs deben coincidir exactamente con los `name:` del workflow. Por ejemplo: `Lint & Vet`, `Unit Tests`, etc.

### 8.2 CODEOWNERS

Crea el archivo `.github/CODEOWNERS` en la raíz del repo. Esto **exige** la aprobación de personas concretas cuando se modifican archivos sensibles:

```
# ── CODEOWNERS ───────────────────────────────────────────────
# Syntax: <pattern>  <owners>
# Docs: https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners

# CI/CD workflows — only platform/devops team can approve changes
/.github/                @org/platform-team

# Docker config — changes affect production builds
/Dockerfile              @org/platform-team
/docker-compose*.yml     @org/platform-team
/.dockerignore           @org/platform-team

# Database schema — changes affect all environments
/init.sql                @org/platform-team @org/backend-leads

# Default: any team member can approve other changes
*                        @org/backend-team
```

Para que CODEOWNERS funcione, activa en branch protection:
- **Require review from Code Owners** ✅

Ahora, si alguien toca `ci.yml` o el `Dockerfile`, GitHub exigirá aprobación del equipo de plataforma, no solo de cualquier reviewer.

> **Si eres un equipo pequeño (1-3 personas)**: sustituye `@org/platform-team` por tu usuario (`@jsuarez`). El concepto es el mismo — ciertos archivos requieren aprobación explícita de alguien con criterio.

### 8.3 Rulesets (alternativa moderna)

GitHub Rulesets (**Settings → Rules → Rulesets**) son la evolución de branch protection. Ofrecen lo mismo pero con más granularidad:

- Se pueden aplicar a **múltiples ramas** con un solo ruleset (regex patterns)
- Permiten **reglas por tag** (útil para proteger tags de release)
- Se pueden configurar a nivel de **organización** (aplican a todos los repos — los devs no pueden desactivarlos)

> **¿Branch protection o Rulesets?** Si usas GitHub Free, usa branch protection (2 reglas: una para `main`, otra para `release`). Si tienes GitHub Team ($4/user/mes) o Enterprise, usa Rulesets porque son más potentes y centralizados.

En GitHub aparecen **3 tipos de ruleset**: Branch, Tag y Push. Necesitas configurar **2** de ellos:

#### 8.3.1 Branch Ruleset — `Protect main and release`

**Settings → Rules → Rulesets → New branch ruleset**

| Campo | Valor |
|---|---|
| **Ruleset name** | `Protect main and release` |
| **Enforcement status** | `Active` |
| **Bypass list** | Vacío (nadie se lo salta). Añade tu usuario solo para emergencias. |

**Target branches → Add target → Include by pattern**: añade `main` y `release` como dos patterns separados.

**Rules a activar:**

| Regla | Configuración |
|---|---|
| **Restrict deletions** | ✅ Nadie puede borrar `main` ni `release` |
| **Require a pull request before merging** | ✅ |
| ↳ Required approvals | `1` |
| ↳ Dismiss stale pull request approvals | ✅ |
| ↳ Require review from Code Owners | ✅ |
| ↳ Require conversation resolution before merging | ✅ |
| **Require status checks to pass** | ✅ |
| ↳ Require branches to be up to date before merging | ✅ |
| ↓ Status checks (busca y añade los 6): | `Lint & Vet`, `Unit Tests`, `Integration Tests`, `Contract Tests`, `E2E Tests`, `Docker Build` |
| **Block force pushes** | ✅ |

> **Nota**: los nombres de los status checks aparecen en el buscador **solo después** de que el workflow haya corrido al menos una vez. Haz push del `ci.yml` primero, deja que corra, y luego configura el ruleset.

Las demás reglas (`Require signed commits`, `Require linear history`, etc.) déjalas desactivadas por ahora.

#### 8.3.2 Tag Ruleset — `Protect release tags`

**Settings → Rules → Rulesets → New tag ruleset**

| Campo | Valor |
|---|---|
| **Ruleset name** | `Protect release tags` |
| **Enforcement status** | `Active` |
| **Bypass list** | Tu usuario o el equipo de maintainers (alguien tiene que poder crear tags para release) |

**Target tags → Add target → Include by pattern**: `v*`

**Rules a activar:**

| Regla | Por qué |
|---|---|
| **Restrict creations** | ✅ Solo los usuarios en la bypass list pueden crear tags `v*`. Evita que cualquiera dispare un deploy a producción. |
| **Restrict deletions** | ✅ Nadie puede borrar un tag de release. |
| **Block force pushes** | ✅ |

#### 8.3.3 Push Ruleset — NO lo necesitas

El push ruleset restringe **quién** puede hacer push a qué rutas de archivos. Es redundante si ya tienes:
- Branch ruleset que exige PRs (nadie hace push directo)
- CODEOWNERS que exige aprobación para archivos sensibles

Solo es útil en organizaciones grandes donde quieres impedir que alguien ni siquiera pueda *incluir* ciertos archivos en un commit.

#### 8.3.4 Rulesets a nivel de organización

Si tienes GitHub Team o Enterprise, puedes crear rulesets desde **Organization Settings → Rules → Rulesets** que aplican a **todos los repos** (o a un subconjunto por nombre/pattern). La ventaja clave: los admins de cada repo **no pueden desactivarlos**.

| Característica | Branch Protection | Rulesets (repo) | Rulesets (org) |
|---|---|---|---|
| Aplica a múltiples repos | No | No | **Sí** |
| Los admins del repo pueden desactivarlo | Sí | Sí | **No** |
| Disponible en GitHub Free | Sí (limitado) | Sí | **No** (Team/Enterprise) |
| Aplica por pattern de branches | No (una regla por rama) | Sí | Sí |
| Aplica por pattern de repos | N/A | N/A | Sí |

### 8.4 Seguridad en el deploy con tags (CI antes de CD)

Una preocupación legítima: si crear un tag `v*` dispara automáticamente un deploy a producción, ¿qué pasa si tageas el commit equivocado? ¿Se despliega sin validación?

**No**, si configuras CD correctamente. Hay **3 capas de seguridad** que se combinan:

**Capa 1 — El código ya pasó CI (implícita)**:
Todo lo que llega a `release` ya pasó CI porque branch protection exige que el PR tenga los checks verdes. Pero esto no es suficiente — podrías tagear el commit equivocado.

**Capa 2 — CI se re-ejecuta en el tag (workflow prerequisito)**:
El workflow de CD llama a CI como primer paso. Si falla, el deploy no se ejecuta:

```yaml
# .github/workflows/cd.yml (estructura simplificada)
name: CD
on:
  push:
    tags: ["v*"]

jobs:
  # Step 1: re-run full CI suite on the tagged commit
  ci:
    uses: ./.github/workflows/ci.yml

  # Step 2: deploy only if CI passed
  deploy:
    needs: [ci]                    # ← blocked until CI passes
    runs-on: ubuntu-latest
    environment: production        # ← requires manual approval
    steps:
      - uses: actions/checkout@v4
      # ... build image, push, deploy
```

Para que esto funcione, `ci.yml` incluye `workflow_call` en su `on:` — esto ya está configurado. Permite que otros workflows (como `cd.yml`) lo invoquen como un sub-workflow reutilizable.

**Capa 3 — Aprobación manual con GitHub Environments (la más importante)**:
El job de deploy usa `environment: production`. Cuando CD llega a este punto, GitHub **pausa la ejecución** y muestra un panel donde puedes:
- Ver qué commit se va a desplegar
- Verificar que CI pasó (los 5 jobs en verde)
- Verificar el tag y el mensaje del commit
- **Aprobar** o **rechazar** el deploy

```
┌────────────────────────────────────────────────┐
│  Deploy to production                          │
│  Waiting for review                            │
│                                                │
│  CI: ✅ Lint & Vet                             │
│      ✅ Unit Tests                             │
│      ✅ Integration Tests                      │
│      ✅ Contract Tests                         │
│      ✅ E2E Tests                              │
│                                                │
│  Tag: v1.0.1                                   │
│  Commit: abc1234 - "fix: payment timeout"      │
│                                                │
│     [ Reject ]        [ Approve and deploy ]   │
└────────────────────────────────────────────────┘
```

**El flujo completo seguro:**

```
1. Código en release (ya pasó CI vía PR)
         │
2. git tag v1.0.1 && git push origin v1.0.1
         │
3. CD se dispara
         │
4. Re-ejecuta CI completo (5 jobs)
         │
    ┌────┴────┐
    ▼         ▼
  ✅ CI     ❌ CI falla → deploy cancelado
  pasa
    │
5. GitHub pausa: aprobación manual requerida
         │
    ┌────┴────┐
    ▼         ▼
  Aprobado   Rechazado → deploy cancelado
    │
6. Deploy a producción
```

> **Resultado**: 3 puntos de fallo seguro antes de que algo llegue a producción. No existe riesgo de deploy accidental.

### 8.5 Proteger los secretos y environments

En **Settings → Secrets and variables → Actions**:

- Crea los secretos necesarios para CD (`DOCKER_USERNAME`, `DOCKER_PASSWORD`, etc.)
- Los secretos **nunca aparecen en los logs** — GitHub los enmascara automáticamente con `***`

En **Settings → Environments**:

| Environment | Configuración |
|---|---|
| **staging** | Sin protección especial (se despliega automáticamente tras merge a `main`) |
| **production** | **Required reviewers** ✅ — añade tu usuario o el equipo de leads. Esto es lo que activa el panel de aprobación manual del punto 8.4. |

### 8.6 Checklist rápido

```
□ Branch ruleset en main + release → require PR + approvals + status checks
□ Tag ruleset en v*                → restrict creations + deletions
□ CODEOWNERS creado                → .github/CODEOWNERS
□ Require Code Owner review        → activado en branch ruleset
□ Status checks marcados required  → los 6 jobs de CI
□ Force push bloqueado             → en main, release y tags v*
□ Environment production           → required reviewers activado
□ Secretos de CD configurados      → en Settings → Secrets
```

---

## 9. Próximos pasos

Con el CI funcionando, ve a `docs/GUIDE_CD.md` para configurar:
- Despliegue automático a **staging** desde `main`
- Despliegue a **producción** cuando se pushea un tag `v*` desde la rama `release`

> Para entender el flujo completo de branches, releases y hotfixes, ve `CONTRIBUTING.md` §6 y §7.
