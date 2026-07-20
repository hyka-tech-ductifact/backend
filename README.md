# Ductifact — Backend

Go REST API following **Hexagonal Architecture** (Ports & Adapters) with PostgreSQL and Docker.

> See [docs/GUIDE_HEXAGONAL_ARCH.md](docs/GUIDE_HEXAGONAL_ARCH.md) for an explanation of the architecture and [docs/GUIDE_AUTHORIZATION_BOUNDARIES.md](docs/GUIDE_AUTHORIZATION_BOUNDARIES.md) for the distinction between ownership, project collaboration, and internal operations.

## Prerequisites

- Go 1.26+
- Docker + Compose plugin v2+ (starts PostgreSQL, Redis, MinIO, Mailpit)

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

Infrastructure endpoints (`/healthz`, `/readyz`, `/metrics`, `/docs`) are at the root level. All business endpoints are prefixed with `/v1`.

| Method | Endpoint | Authorization | Description |
|--------|----------|---------------|-------------|
| GET | `/healthz` | No | Liveness probe |
| GET | `/readyz` | No | Readiness probe |
| GET | `/metrics` | No | Prometheus metrics |
| GET | `/docs` | No | Swagger UI (interactive docs) |
| GET | `/docs/openapi.yaml` | No | Raw OpenAPI spec |
| POST | `/auth/register` | No | Start user registration |
| POST | `/auth/login` | No | Login |
| GET | `/users/me` | JWT (self) | Get the current user |
| PUT | `/users/me` | JWT (self) | Update the current user |
| POST | `/clients` | JWT | Create a client owned by the current user |
| GET | `/clients` | JWT + ownership | List the current user's clients |
| GET | `/clients/:client_id` | JWT + ownership | Get an owned client |
| PUT | `/clients/:client_id` | JWT + ownership | Update an owned client |
| DELETE | `/clients/:client_id` | JWT + ownership | Delete an owned client |

The versioned paths in this table are served below `/v1`; for example, the
authenticated profile endpoint is `GET /v1/users/me`.

The current authorization model is intentionally limited to authentication and
resource ownership. Product collaboration (`owner`, `editor`, `viewer`) and
internal support operations are separate future capabilities; see the
[authorization boundaries guide](docs/GUIDE_AUTHORIZATION_BOUNDARIES.md).

### List query convention

The five collection endpoints use the same optional query parameters:

- `page` is 1-based and defaults to `1`.
- `page_size` defaults to `20` and accepts values from `1` to `100`.
- `search` accepts up to 200 characters and performs a case-insensitive literal
  partial match. Surrounding whitespace is ignored; `\\`, `%`, and `_` are
  treated literally.
- `sort_by` selects an allowed field for the endpoint. When it is present,
  `sort_order` accepts `asc` or `desc` and defaults to `asc`.
- Without `sort_by`, the existing order is preserved: `created_at desc` for
  clients, projects, orders, and pieces; `predefined desc, created_at desc` for
  piece definitions. `sort_order` does not change this default by itself.

| Endpoint | `search` fields | Exact filters | Allowed `sort_by` fields |
|----------|-----------------|---------------|--------------------------|
| `GET /clients` | `name`, `email` | — | `name`, `email`, `created_at`, `updated_at` |
| `GET /clients/:client_id/projects` | `name`, `address`, `manager_name` | — | `name`, `address`, `manager_name`, `created_at`, `updated_at` |
| `GET /projects/:project_id/orders` | `title` | `status=pending\|completed` | `title`, `status`, `created_at`, `updated_at` |
| `GET /orders/:order_id/pieces` | `title` | `definition_id=<uuid>` | `title`, `quantity`, `created_at`, `updated_at` |
| `GET /piece-definitions` | `name` | `predefined=true\|false`, `include_archived=true\|false` (default `false`) | `name`, `predefined`, `created_at`, `updated_at`, `archived_at` |

Invalid pagination, filtering, search, or sorting parameters return `400 Bad Request`.

See [test/api.http](test/api.http) for request examples.

## Other commands

```bash
make help              # list all available commands
make app-build         # compile binary to bin/api
make ensure-contract   # use local OpenAPI bundle or download ContractVersion
make fmt               # format code
make lint              # lint code
make clean             # remove build artifacts
```

## Updating the API contract

The contract version is defined as a Go constant in `internal/config/contract_version.go`.
When a contract change is ready to release:

1. Commit the source changes in the contracts repository and publish its new tag.
2. Update the backend constant, for example `const ContractVersion = "0.14.0"`.
3. Run `make ensure-contract` and validate the reported version.
4. Commit the contract source and backend version bump in their respective repositories.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for branch naming, workflow, and PR guidelines.
