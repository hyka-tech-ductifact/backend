# Backend

Go REST API following **Hexagonal Architecture** (Ports & Adapters) principles, with PostgreSQL and Docker.

> **Note:** This project uses Hexagonal Architecture, not Clean Architecture. They share goals but differ in structure. See `docs/LEARNING_GUIDE.md` for a full explanation.

## Prerequisites

- Go 1.23+
- Docker + Compose plugin v2+
- [air](https://github.com/air-verse/air) for hot reload (`go install github.com/air-verse/air@latest`)

## Setup

```bash
cp .env.example .env   # create your local .env (git-ignored) from the template
make deps              # download Go dependencies and install dev tools
```

> The `.env` file holds database credentials and app settings. It is **not committed to git** to keep secrets out of the repo. The `.env.example` file serves as a reference with safe default values so every developer knows which variables are needed.

### Add Go bin to PATH

Dev tools like `air` are installed in `$(go env GOPATH)/bin`. Make sure this directory is in your `PATH`:

```bash
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
```

---

## Container Engine Setup

Choose the option that fits your environment.

### Option A — Local Docker Engine (standard)

Install Docker Engine locally. No extra configuration needed.

### Option B — Podman (WSL / Linux, no Docker Engine)

```bash
# Install Podman and Docker CLI compatibility layer
sudo apt install -y podman podman-docker docker-ce-cli docker-compose-plugin

# Silence the Podman emulation notice
sudo mkdir -p /etc/containers && sudo touch /etc/containers/nodocker

# Enable the Podman user socket
systemctl --user enable --now podman.socket

# Add to ~/.bashrc and reload
echo 'export DOCKER_HOST=unix:///run/user/$(id -u)/podman/podman.sock' >> ~/.bashrc
source ~/.bashrc
```

> Image names in `Dockerfile` must be fully qualified (e.g. `docker.io/library/golang:1.23-alpine`) because Podman does not resolve short names by default.

---

## 🔵 Development

DB runs in Docker. Your app runs locally with **hot reload** via `air`.

```bash
make db-start    # start PostgreSQL in Docker
make app-watch   # start the app with hot reload (air)
```

Every time you save a `.go` file, air recompiles and restarts the server automatically.

```bash
make db-stop     # stop PostgreSQL when done
```

> `.env` must have `DB_HOST=localhost` for this mode.

---

## 🟡 Testing

There are 3 test types. Each requires a different setup.

### Unit tests — no dependencies needed

```bash
make test-unit
make test-unit COVERAGE   # with coverage report
```

### Integration tests — requires DB

```bash
make db-start                           # start PostgreSQL
make test-integration
make test-integration COVERAGE          # with coverage report
```

These spin up the router internally using `httptest`. No running server needed.

### E2E tests — requires DB + running server

```bash
make db-start                   # terminal 1: start PostgreSQL
make app-run                    # terminal 2: start the server
make test-e2e                   # terminal 3: run E2E tests
make test-e2e COVERAGE          # with coverage report
```

These make real HTTP calls to `http://localhost:8080`.

> Run all types at once: `make test` or `make test COVERAGE`

---

## 🔴 Production

Both the app and DB run in Docker. Simulates a real deployment.

```bash
make prod-start    # build image and start app + DB
make prod-stop     # stop all services
```

> The app is available at `http://localhost:8080`.  
> No hot reload. Uses the compiled binary from the multi-stage `Dockerfile`.

---

## API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| POST | `/users` | Create user |
| GET | `/users/:id` | Get user by ID |
| PUT | `/users/:id` | Update user |

See `test/api.http` for request examples.

---

## Other commands

```bash
make app-build      # compile binary to bin/api
make prod-build     # build Docker image
make fmt            # format code
make lint           # lint code (requires golangci-lint)
make clean          # remove build artifacts and Docker cache
make help           # list all available commands
```
