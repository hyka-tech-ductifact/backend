# Backend

A clean architecture Go backend following hexagonal architecture principles. Provides a solid foundation for building scalable REST APIs with PostgreSQL integration.

## Architecture

This project follows Clean Architecture (Hexagonal Architecture) with the following layers:

- **Domain**: Core business logic and entities
- **Application**: Use cases and business rules
- **Infrastructure**: External adapters (database, messaging)
- **Interfaces**: API handlers and external interfaces

## Features

- ✅ REST API with Gin framework
- ✅ PostgreSQL database integration with GORM
- ✅ Clean architecture structure (Hexagonal Architecture)
- ✅ Docker support with Docker Compose v2
- ✅ Environment-based configuration
- ✅ User management CRUD operations
- ✅ Health check endpoint
- ✅ Multi-stage Docker build for optimized images
- ✅ Database initialization with UUID extension
- ✅ Health checks for PostgreSQL container
- ✅ Makefile for simplified development workflow

## Prerequisites

- Go 1.23+
- Docker CLI + Compose plugin v2+ (Docker Engine can be local or remote)
- PostgreSQL (only if you run it outside Compose)

## Container Engine Setup

This project works with three different container engine setups. Choose the one that fits your environment.

---

### Option A — Local Docker Engine (standard)

The simplest setup: install Docker Engine locally and everything works out of the box with no extra configuration.

---

### Option B — Remote Docker Engine via `DOCKER_HOST`

If Docker Engine (`dockerd`) runs on another machine (e.g. a server or VM), you only need the **CLI** and the **Compose plugin** installed locally — no local daemon required.

```bash
# Ubuntu/Debian — install only the client tools (no dockerd)
sudo apt-get install docker-ce-cli docker-compose-plugin
```

Point the CLI to the remote engine using `DOCKER_HOST`:

```bash
# Via SSH (recommended — no open TCP ports required on the daemon)
export DOCKER_HOST=ssh://user@192.168.1.10

```

> **Note:** Published ports (e.g. `192.168.1.10:8080`) are exposed on the **remote host**, not on your local machine.

---

### Option C — Podman via `DOCKER_HOST` (WSL / Linux, no Docker Engine)

If you use **Podman** instead of Docker Engine (e.g. on Ubuntu WSL with Podman Desktop on Windows), you can point the Docker CLI and Compose plugin directly at the Podman socket. No `dockerd` is required.

```bash
# Ubuntu/Debian — install Podman and Docker CLI compatibility layer
sudo apt install -y podman podman-docker docker-ce-cli docker-compose-plugin

# Silence the Podman emulation notice
sudo mkdir -p /etc/containers && sudo touch /etc/containers/nodocker

# Enable the Podman user socket
systemctl --user enable --now podman.socket
```

Set `DOCKER_HOST` to the Podman socket and persist it in your shell:

```bash
# Add to ~/.bashrc (replace 1000 with your user id if different)
echo 'export DOCKER_HOST=unix:///run/user/$(id -u)/podman/podman.sock' >> ~/.bashrc
source ~/.bashrc
```

> **Note:** Podman runs **daemonless** — each CLI call spawns a short-lived process. The socket is created by `podman.socket` (systemd user unit) and lives at `/run/user/<uid>/podman/podman.sock`.
>
> Image names in `Dockerfile` must be **fully qualified** (e.g. `docker.io/library/golang:1.23-alpine`) because Podman does not resolve short names by default.

## Quick Start

### Using Makefile (Recommended)

The project includes a comprehensive Makefile that simplifies all common development tasks. Use `make help` to see all available commands.

#### 1. Initial Setup
```bash
# Clone the repository
git clone <repository-url>
cd event-service

# Install dependencies
make deps

# Configure database connection (edit .env file)
# Set DB_HOST to your Docker IP or 'postgres' for Docker Compose
```

#### 2. Start the Application

**Option A: Run with Docker Compose (Recommended)**
```bash
# Configure .env for Docker Compose
echo "DB_HOST=postgres" > .env
echo "DB_PORT=5432" >> .env
echo "DB_USER=postgres" >> .env
echo "DB_PASSWORD=postgres123" >> .env
echo "DB_NAME=microservice_db" >> .env
echo "PORT=8080" >> .env

# Start all services (API + Database)
make docker-run

# The API will be available at http://localhost:8080
```

**Option B: Run locally with Docker database**
```bash
# Configure .env for local development
echo "DB_HOST=192.168.1.10" > .env
echo "DB_PORT=5432" >> .env
echo "DB_USER=postgres" >> .env
echo "DB_PASSWORD=postgres123" >> .env
echo "DB_NAME=microservice_db" >> .env
echo "PORT=8080" >> .env

# Start only the database
make docker-run

# In another terminal, run the Go app locally
make run
```

**Option C: Run with hot reloading (Development)**
```bash
# Configure .env for local development (same as Option B)
echo "DB_HOST=192.168.1.10" > .env
echo "DB_PORT=5432" >> .env
echo "DB_USER=postgres" >> .env
echo "DB_PASSWORD=postgres123" >> .env
echo "DB_NAME=microservice_db" >> .env
echo "PORT=8080" >> .env

# Start database
make docker-run

# In another terminal, run with hot reloading
make dev
```

#### 3. Verify Installation
```bash
# Check service status
docker compose ps

# Test the health endpoint (API runs on localhost:8080 regardless of DB_HOST)
curl http://192.168.1.10:8080/health
curl http://192.168.1.10:8080/health
```

### Alternative: Manual Commands

If you prefer to run commands directly without the Makefile:

#### Using Docker Compose
```bash
# Start services
docker compose up -d

# Stop services
docker compose down
```

#### Local Development
```bash
# Install dependencies
go mod download

# Configure .env file first (see Environment Configuration section)
# Then run the application
go run cmd/api/main.go
```

## Development Workflow

### Essential Makefile Commands

```bash
# View all available commands
make help

# Build the application
make build

# Run the application
make run

# Run with hot reloading (requires air)
make dev

# Run all tests
make test

# Run tests with coverage report
make test-coverage

# Run tests in Docker with database
make test-docker

# Format code
make fmt

# Lint code
make lint

# Clean build artifacts
make clean
```

### Docker Commands

```bash
# Build Docker image
make docker-build

# Start services with Docker Compose
make docker-run

# Stop Docker Compose services
make docker-stop
```

## API Endpoints

### Health Check
- `GET /health` - Service health status

See `test/api.http` for detailed API usage examples and testing scenarios.

## Environment Configuration

The project uses a simple `.env` file for configuration. The `DB_HOST` value determines where your application connects to the database.

### Quick IP Changes

```bash
# Edit the .env file to change the database IP
nano .env

# Or use any text editor
code .env
```

### Common DB_HOST Values

- `postgres` - When running with Docker Compose (services communicate via service names)
- `192.168.1.10` - Your Docker IP address (current setup)
- `localhost` - When running everything locally
- `host.docker.internal` - WSL2 Docker Desktop

### Quick Setup Commands

```bash
# For Docker Compose (all services in containers)
echo "DB_HOST=postgres" > .env && echo "DB_PORT=5432" >> .env && echo "DB_USER=postgres" >> .env && echo "DB_PASSWORD=postgres123" >> .env && echo "DB_NAME=microservice_db" >> .env && echo "PORT=8080" >> .env

# For local development (app outside Docker, DB in Docker)
echo "DB_HOST=192.168.1.10" > .env && echo "DB_PORT=5432" >> .env && echo "DB_USER=postgres" >> .env && echo "DB_PASSWORD=postgres123" >> .env && echo "DB_NAME=microservice_db" >> .env && echo "PORT=8080" >> .env
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| PORT | Server port | 8080 |
| DB_HOST | Database host | 192.168.1.10 (configure as needed) |
| DB_PORT | Database port | 5432 |
| DB_USER | Database user | postgres |
| DB_PASSWORD | Database password | postgres123 |
| DB_NAME | Database name | microservice_db |

## Troubleshooting

### Database Connection Issues

If you're having trouble connecting to the database, check:

1. **Verify your DB_HOST value:**
   ```bash
   cat .env | grep DB_HOST
   ```

2. **Test database connectivity:**
   ```bash
   # If using Docker IP
   telnet 192.168.1.10 5432
   
   # If using localhost
   telnet localhost 5432
   ```

3. **Check if database is running:**
   ```bash
   docker compose ps
   ```

4. **View database logs:**
   ```bash
   docker logs microservice_postgres
   ```

### Common Issues

- **Connection refused**: Check if database is running and DB_HOST is correct
- **Authentication failed**: Verify DB_USER and DB_PASSWORD in .env
- **Database doesn't exist**: Check DB_NAME value

## Project Structure

```