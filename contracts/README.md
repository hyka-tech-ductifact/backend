# API Contract

OpenAPI 3.1 specification for the Ductifact API. The contract lives in the
backend repository so every commit ships the spec together with the code that
implements it.

## Structure

```
openapi/
├── openapi.yaml        # Root spec (entry point)
├── paths/              # Path definitions split by resource
└── schemas/            # Reusable schema definitions
```

`openapi/bundled.yaml` is a generated artifact (git-ignored). It resolves all
`$ref` entries into a single file consumed by Swagger UI (`/docs`), Schemathesis
and the Docker image.

## Commands

Run from the backend root. All tooling runs in Docker.

```bash
make contract-validate   # Lint and validate the spec (redocly)
make contract-bundle     # Generate openapi/bundled.yaml
make contract-breaking   # Detect breaking changes vs main (oasdiff)
make contract-ui         # Standalone Swagger UI at http://localhost:8081
```

## Versioning

`info.version` in `openapi.yaml` is a placeholder. The real value is
`ContractVersion` in `internal/config/contract_version.go`, injected into the
bundle at build time and reported by `/readyz`. Bump it when the spec changes:

| Change | Bump |
|---|---|
| Descriptions, examples, typos | PATCH |
| New endpoint or optional field | MINOR |
| Removed/renamed field or endpoint, changed response shape | MAJOR |

## Workflow

1. Edit the source files under `openapi/` (never `bundled.yaml`).
2. `make contract-validate && make contract-breaking`.
3. Implement the change in Go and bump `ContractVersion`.
4. Open a single PR with contract and implementation.
