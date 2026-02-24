# Backend

Go REST API following Clean Architecture (Hexagonal) principles, with PostgreSQL and Docker.

## Prerequisites

- Go 1.23+
- Docker + Compose plugin v2+
- [air](https://github.com/air-verse/air) for hot reload (`go install github.com/air-verse/air@latest`)

## Setup

```bash
cp .env.example .env   # configure your environment variables once
make deps              # download Go dependencies
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
make dev-start   # start PostgreSQL in Docker
make dev         # start the app with hot reload (air)
```

Every time you save a `.go` file, air recompiles and restarts the server automatically.

```bash
make dev-stop    # stop PostgreSQL when done
```

> `.env` must have `DB_HOST=localhost` for this mode.

---

## 🟡 Testing

There are 3 test types. Each requires a different setup.

### Unit tests — no dependencies needed

```bash
go test -v ./test/unit/...
```

### Integration tests — requires DB

```bash
make dev-start                      # start PostgreSQL
go test -v ./test/integration/...
```

These spin up the router internally using `httptest`. No running server needed.

### E2E tests — requires DB + running server

```bash
make dev-start                  # terminal 1: start PostgreSQL
make run                        # terminal 2: start the server
go test -v ./test/e2e/...       # terminal 3: run E2E tests
```

These make real HTTP calls to `http://localhost:8080`.

---

## 🔴 Production

Both the app and DB run in Docker. Simulates a real deployment.

```bash
make docker-run    # build image and start app + DB
make docker-stop   # stop all services
```

> The app is available at `http://localhost:8080`.  
> No hot reload. Uses the compiled binary from the multi-stage `Dockerfile`.

---

## API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| POST | `/events` | Create event |
| GET | `/events/:id` | Get event by ID |
| GET | `/events` | List events |
| PUT | `/events/:id` | Update event |
| DELETE | `/events/:id` | Delete event |

See `test/api.http` for request examples.

---

## Other commands

```bash
make build          # compile binary to bin/api
make test-coverage  # run tests with HTML coverage report
make fmt            # format code
make lint           # lint code (requires golangci-lint)
make clean          # remove build artifacts and Docker cache
make help           # list all available commands
```
