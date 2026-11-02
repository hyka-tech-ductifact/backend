# Environment Setup Guide

## Prerequisites

- Go 1.26+
- Docker + Compose plugin v2+ (or Podman)
- Make and a C compiler (for tests with `CI=1`, which enable the race detector)
- Redis 7+ (provided by `docker compose` in dev)

## Go bin directory

Dev tools (`air`, `gotestsum`, `golangci-lint`, `golines`, `govulncheck`) are installed via `make deps` into `$(go env GOPATH)/bin`. Add it to your `PATH`:

```bash
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
source ~/.bashrc
```

## Environment variables

```bash
cp .env.example .env
```

The `.env` file holds database credentials and app settings. It is **not committed to git**. See `.env.example` for all available variables.

Key variables:

| Variable | Dev value | Docker value | Description |
|----------|-----------|--------------|-------------|
| `DB_HOST` | `localhost` | `postgres` | Use `localhost` for local dev, `postgres` for Docker Compose |
| `DB_PORT` | `5432` | `5432` | PostgreSQL port |
| `BACKEND_PORT` | `8080` | `8080` | API server port |
| `JWT_SECRET` | (any 32+ chars) | (any 32+ chars) | Secret for JWT signing |
| `AUTO_MIGRATE` | `true` | `false` | Auto-create tables on startup |
| `REDIS_HOST` | `localhost` | `redis` | Redis host |
| `REDIS_PORT` | `6379` | `6379` | Redis port |
| `REDIS_AUTH` | `false` | `false` | Enable AUTH (set `true` in production) |
| `REDIS_PASSWORD` | `noop` | `noop` | AUTH password (only used when `REDIS_AUTH=true`) |
| `REDIS_DB` | `0` | `0` | Redis database number (0–15) |

## Install dependencies

```bash
make deps
```

This runs `go mod tidy`, `go mod download`, and installs `air`, `golangci-lint`, `golines`, `gotestsum`, and `govulncheck`.

---

## Container Engine

You need a container engine for the database (dev) and for production mode. Choose one:

### Option A — Docker Engine (recommended)

Install Docker Engine following the [official docs](https://docs.docker.com/engine/install/). No extra configuration needed.

Verify:

```bash
docker compose version
```

### Option B — Podman (Ubuntu / Linux)

#### Local development with Make (hot reload, tests, formatting, lint)

In this workflow, the API and Go tools run on the host. PostgreSQL, RustFS,
Redis, Mailpit, and contract tools run in containers managed by Podman.

Install the host dependencies and Docker CLI compatibility. These package
names are available on Ubuntu 26.04; other distributions may use different names.

```bash
sudo apt-get update
sudo apt-get install -y golang-go make build-essential curl podman podman-compose podman-docker docker-compose-v2
```

The module requires Go 1.26 and pins toolchain `go1.26.10` in `go.mod`.
Go downloads that toolchain automatically when needed (with `GOTOOLCHAIN=auto`,
the default), so the first Go command needs network access.

The Makefile calls `docker` and `docker compose`. `podman-docker` provides the
Docker-compatible CLI, while Docker Compose connects to the Podman user socket:

```bash
systemctl --user enable --now podman.socket
export DOCKER_HOST="unix://${XDG_RUNTIME_DIR}/podman/podman.sock"
```

Persist the socket setting for future Bash sessions:

```bash
echo 'export DOCKER_HOST="unix://${XDG_RUNTIME_DIR}/podman/podman.sock"' >> ~/.bashrc
```

Verify the engine and Compose provider:

```bash
go version
make --version
docker info
docker compose version
```

From `backend/`, create `.env` if it does not already exist. Set `JWT_SECRET`
to a random secret (`openssl rand -base64 32`); keep service hosts as `localhost`
for the host API. Then install the Go dependencies and tools:

```bash
make deps
export PATH="$(go env GOPATH)/bin:$PATH"
```

Add the Go bin directory to `~/.bashrc` as described above for future terminals.

Start development:

```bash
make dev
```

In another terminal, from `backend/`:

```bash
curl --fail http://localhost:8080/readyz
make test-unit
make test-integration
make test-e2e
make test-contract
make fmt
make lint
make vulncheck
```

Integration tests require the services, and E2E/contract tests require the API.
`make services-start` starts just the services; `make app-start` builds and
starts the API in the background without hot reload. Stop development with
`make dev-stop`. Avoid running the containerized API at the same time as the
host API, since both use port 8080.

`make validate-branch` requires `BRANCH` and optionally `BASE`, as in CI.
`make contract-breaking` requires a locally available `BASE_REF` (default `main`).
`make clean` deletes untracked/ignored artifacts and database volumes, preserving `.env`.

#### Switching from podman-compose

Use `docker compose` consistently with the socket configuration above. Although
both providers use Podman, `podman-compose` and Docker Compose attach different
resource labels. Mixing them on the same project can cause an error such as
`com.docker.compose.network` being empty instead of `default`.

If the stack was originally created with `podman-compose`, migrate it once:

```bash
podman-compose --profile smoke down
make services-start
```

This recreates containers and the network while preserving named data volumes.
Do not add `-v`, which would delete those volumes.

#### Fully containerized alternative

For a container build and smoke test without host Go tools:

```bash
docker compose --profile smoke up -d --build
docker compose --profile smoke logs -f app
```

Stop that stack while preserving database volumes:

```bash
docker compose --profile smoke down
```

With Docker CLI compatibility configured, `make docker-start` and
`make docker-stop` provide the Makefile's container smoke-test workflow.

API docs: http://localhost:8080/docs. Email inbox: http://localhost:8025.

Use fully qualified image names (for example,
`docker.io/library/postgres:15-alpine`) so Podman can resolve them without
configuring short-name search registries.
