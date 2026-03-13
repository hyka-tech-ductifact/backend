# Environment Setup Guide

## Prerequisites

- Go 1.24+
- Docker + Compose plugin v2+ (or Podman)
- Make

## Go bin directory

Dev tools (`air`, `gotestsum`, `golangci-lint`) are installed via `make deps` into `$(go env GOPATH)/bin`. Add it to your `PATH`:

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
| `APP_PORT` | `8080` | `8080` | API server port |
| `JWT_SECRET` | (any 32+ chars) | (any 32+ chars) | Secret for JWT signing |
| `AUTO_MIGRATE` | `true` | `false` | Auto-create tables on startup |

## Install dependencies

```bash
make deps
```

This runs `go mod tidy`, `go mod download`, and installs dev tools (`air`, `golangci-lint`, `gotestsum`).

---

## Container Engine

You need a container engine for the database (dev) and for production mode. Choose one:

### Option A — Docker Engine (recommended)

Install Docker Engine following the [official docs](https://docs.docker.com/engine/install/). No extra configuration needed.

Verify:

```bash
docker compose version
```

### Option B — Podman (WSL / Linux, no Docker Engine)

If you can't or don't want to install Docker Engine, Podman works as a drop-in replacement.

#### 1. Install Podman + Docker CLI compatibility

```bash
sudo apt install -y podman podman-docker docker-ce-cli docker-compose-plugin
```

#### 2. Silence the Podman emulation notice

```bash
sudo mkdir -p /etc/containers
sudo touch /etc/containers/nodocker
```

#### 3. Enable the Podman user socket

```bash
systemctl --user enable --now podman.socket
```

#### 4. Set DOCKER_HOST

Add to `~/.bashrc`:

```bash
echo 'export DOCKER_HOST=unix:///run/user/$(id -u)/podman/podman.sock' >> ~/.bashrc
source ~/.bashrc
```

#### 5. Verify

```bash
docker compose version
docker run hello-world
```

> **Note:** Image names in `Dockerfile` and `docker-compose*.yml` must be fully qualified (e.g. `docker.io/library/golang:1.24-alpine`) because Podman does not resolve short names by default.
