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
```

## Versioning

The contract has no version of its own: it is versioned by the backend commit.
`info.version` is `dev` in the source and is stamped with the build commit in
the Docker image; `/readyz` reports the same `commit`. Compatibility is enforced
by `make contract-breaking` (and the `Contract Spec` CI job), not by a number.
A breaking change that must ship goes through a new path prefix (`/v2`).

## Workflow

1. Edit the source files under `openapi/` (never `bundled.yaml`).
2. `make contract-validate && make contract-breaking`.
3. Implement the change in Go.
4. Open a single PR with contract and implementation.
