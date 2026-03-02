# Testing Guide

This directory contains comprehensive tests for the Ductifact backend application, organized in a centralized structure for better maintainability and clarity.

## Test Structure

```
test/
├── README.md                    # This file
├── api.http                     # HTTP API test suite
├── unit/                        # Unit tests (no dependencies)
├── integration/                 # Integration tests (requires DB)
├── e2e/                         # E2E tests (requires DB + running server)
│   ├── setup.go
│   └── README.md
└── helpers/                     # Shared test utilities
    ├── test_config.go
    └── test_utils.go
```

## Running Tests

Tests always run locally. Only the DB runs in Docker.

```bash
make test-unit          # unit tests — no dependencies needed
make test-integration   # integration tests — requires: make dev-start
make test-e2e           # E2E tests — requires: make dev-start + make run
make test               # all of the above in order
make test-coverage      # all tests + HTML coverage report
```

## Test Types

### 1. Unit Tests
- **Location**: `test/unit/`
- **Purpose**: Test individual components in isolation
- **Database**: None (pure logic testing)
- **Speed**: Fastest
- **Examples**: Domain entity validation, business logic

### 2. Integration Tests
- **Location**: `test/integration/`
- **Purpose**: Test components working together
- **Database**: Real PostgreSQL database
- **HTTP**: In-memory testing with `httptest.NewRecorder()`
- **Speed**: Fast
- **Examples**: API endpoint testing, database operations

### 3. End-to-End Tests
- **Location**: `test/e2e/`
- **Purpose**: Test complete system from HTTP to database
- **Database**: Real PostgreSQL database
- **HTTP**: Real HTTP requests to running server
- **Speed**: Slower (network overhead)
- **Examples**: Complete user workflows, system resilience

## Test Execution Comparison

| Test Type | Speed | HTTP Method | Database | Focus |
|-----------|-------|-------------|----------|-------|
| **Unit** | ⚡⚡⚡ | None | None | Individual logic |
| **Integration** | ⚡⚡ | `httptest.NewRecorder()` | Real | Component interaction |
| **E2E** | ⚡ | Real HTTP requests | Real | Complete system |

## Test Data

- Data is created programmatically in each test, no fixture files needed
- UUIDs are generated per test run to avoid collisions
- Integration and E2E tests use the development DB (`microservice_db`)

## Best Practices

### 1. Test Organization
- Keep tests close to the code they test
- Use descriptive test names
- Group related tests together
- Follow consistent naming conventions

### 2. Test Data
- Create test data programmatically
- Use realistic but minimal data
- Ensure test isolation
- Clean up after tests

### 3. Assertions
- Use descriptive assertion messages
- Test both positive and negative cases
- Verify edge cases and error conditions
- Test data consistency

### 4. Performance
- Keep tests fast and focused
- Use appropriate test types for the scenario
- Avoid unnecessary setup/teardown
- Use test caching when appropriate

## CI/CD Integration

### GitHub Actions Example
```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      - run: make test
```

## Troubleshooting

### Common Issues

1. **Database Connection Errors**
   - Ensure PostgreSQL is running
   - Check environment variables
   - Verify network connectivity

2. **Test Failures**
   - Check test isolation
   - Verify test data setup
   - Review assertion messages

3. **Performance Issues**
   - Use appropriate test types
   - Optimize database queries
   - Consider test parallelization

### Debug Mode
```bash
# Run tests with debug output
GIN_MODE=debug go test -v ./test/integration/...

# Run single test with verbose output
go test -v -run TestCreateUser_Success ./test/integration/...
```

## Contributing

When adding new tests:

1. **Choose the right test type** for your scenario
2. **Follow existing patterns** and conventions
3. **Write descriptive test names** and comments
4. **Ensure test isolation** and cleanup
5. **Update documentation** if needed

## Resources

- [Go Testing Package](https://golang.org/pkg/testing/)
- [Testify Assertion Library](https://github.com/stretchr/testify)
- [Gin Testing](https://gin-gonic.com/docs/testing/)
- [GORM Testing](https://gorm.io/docs/testing.html)