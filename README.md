# Ductifact — Backend

Go REST API following **Hexagonal Architecture** (Ports & Adapters) with PostgreSQL and Docker.

> See [docs/GUIDE_HEXAGONAL_ARCH.md](docs/GUIDE_HEXAGONAL_ARCH.md) for an explanation of the architecture.

## Prerequisites

- Go 1.24+
- Docker + Compose plugin v2+

## Setup

```bash
cp .env.example .env   # create your local .env (git-ignored)
make deps              # download Go dependencies and install dev tools
```

> Make sure `$(go env GOPATH)/bin` is in your `PATH` so tools like `air` and `gotestsum` are available.
>
> For detailed environment setup (Podman, Docker, env vars), see [docs/GUIDE_SETUP.md](docs/GUIDE_SETUP.md).

## Development

```bash
make dev   # build + DB + the app with hot reload (air)
```

> `.env` must have `DB_HOST=localhost` for this mode.

## Testing

```bash
make test-unit                  # unit tests (no dependencies)
make test-integration           # integration tests (requires DB)
make test-contract              # contract tests (requires DB + running server)
make test-e2e                   # E2E tests (requires DB + running server)
make test                       # run all tests
```

For contract/E2E tests, start the server first:

```bash
make app-start      # starts DB + fetches contract + builds + runs API
make test-e2e
```

Flags: `CI=1` (race detector + JUnit XML), `COVERAGE=1` (coverage report), `TEST_FORMAT=dots`.

## Docker (smoke test)

```bash
make docker-build    # build Docker image
make docker-start    # build + start app & DB in Docker
make docker-stop     # stop Docker services
```

## API

Infrastructure endpoints (`/health`, `/metrics`, `/docs`) are at the root level. All business endpoints are prefixed with `/v1`.

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/health` | No | Health check |
| GET | `/metrics` | No | Prometheus metrics |
| GET | `/docs` | No | Swagger UI (interactive docs) |
| GET | `/docs/openapi.yaml` | No | Raw OpenAPI spec |
| POST | `/auth/register` | No | Register user |
| POST | `/auth/login` | No | Login |
| GET | `/users/me` | Yes | Get current user |
| PUT | `/users/me` | Yes | Update current user |
| POST | `/clients` | Yes | Create client |
| GET | `/clients` | Yes | List clients |
| GET | `/clients/:client_id` | Yes | Get client |
| PUT | `/clients/:client_id` | Yes | Update client |
| DELETE | `/clients/:client_id` | Yes | Delete client |

See [test/api.http](test/api.http) for request examples.

## Other commands

```bash
make help              # list all available commands
make app-build         # compile binary to bin/api
make fetch-contract    # download OpenAPI spec matching ContractVersion
make fmt               # format code
make lint              # lint code
make clean             # remove build artifacts
```

## Updating the API contract

The contract version is defined as a Go constant in `internal/config/contract_version.go`.
When the contracts repo publishes a new release:

1. Update the constant: `const ContractVersion = "0.4.0"`
2. Run `make fetch-contract` to download the matching spec
3. Commit both changes in the same PR

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for branch naming, workflow, releases, and PR guidelines.
