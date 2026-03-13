# Testing

## Overview

The backend has 4 test levels, each with a different scope and purpose:

| Level | What it tests | Dependencies | Location |
|-------|---------------|--------------|----------|
| **Unit** | Domain logic, services, value objects | None (uses mocks) | `test/unit/` |
| **Integration** | Repositories against real PostgreSQL | DB | `test/integration/` |
| **Contract** | API responses match the OpenAPI spec | DB + running server | `test/contract/` |
| **E2E** | Full HTTP flows end-to-end | DB + running server | `test/e2e/` |

## Test pyramid

```
          /  E2E   \          Few — critical user flows
         /----------\
        / Contract   \        Moderate — every endpoint × status code
       /--------------\
      /  Integration    \     Moderate — repos, constraints, mappers
     /--------------------\
    /       Unit            \ Many — all domain logic and combinations
   /------------------------\
```

**Unit tests** are the foundation: fast (milliseconds), precise (pinpoint the bug), and independent (no infra needed). **E2E tests** are expensive but verify that all layers are wired correctly.

## What each level covers

### Unit tests

Test pure logic in isolation. Domain entities, value objects, and application services. Services depend on repository interfaces — we use **manual mocks** (function fields) instead of code generation frameworks.

- Value objects: all valid/invalid input combinations (table-driven tests)
- Entities: constructor validation, auto-generated fields (ID, timestamps)
- Services: business rules, error paths, orchestration logic

### Integration tests

Test repositories against a real PostgreSQL instance. Verify that SQL/GORM queries work, mappers preserve data, and DB constraints (UNIQUE, FK) are enforced.

Each test cleans the database before running (TRUNCATE) to ensure isolation.

### Contract tests

Validate that every API response conforms to the OpenAPI spec (`contracts/openapi/bundled.yaml`). Uses `kin-openapi` to load the spec and validate response bodies, status codes, and schemas automatically.

A coverage tracker ensures all spec-defined operations are tested. Untested operations fail the suite.

### E2E tests

Black-box HTTP tests against the running server. They don't import any internal code — only `net/http` and JSON. Test the critical flows: register, login, CRUD operations, error responses.

## Conventions

- **Test naming**: `Test<Function>_<Scenario>_<ExpectedResult>` (e.g. `TestCreateUser_WithDuplicateEmail_ReturnsError`)
- **Table-driven tests**: used extensively for testing multiple inputs
- **AAA pattern**: Arrange → Act → Assert
- **`require` vs `assert`**: `require` for preconditions (stops the test on failure), `assert` for verifications (continues to show all failures)
- **Test isolation**: every test creates its own data, no shared state between tests
- **Mocks**: manual mocks with function fields — simple, idiomatic Go, no external libraries

## Running tests

```bash
make test-unit              # unit tests (no dependencies)
make test-integration       # requires DB (make db-start)
make test-contract          # requires DB + server (make db-start && make app-start)
make test-e2e               # requires DB + server (make db-start && make app-start)
make test                   # all tests
```

Flags: `CI=1` (race detector + JUnit XML), `COVERAGE=1` (coverage report).
