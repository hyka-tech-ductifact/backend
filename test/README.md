# Testing Guide

This directory contains comprehensive tests for the Event Service application, organized in a centralized structure for better maintainability and clarity.

## Test Structure

```
test/
├── README.md                    # This file
├── api.http                     # Unified HTTP API test suite
├── unit/                        # Unit tests
│   └── domain/
│       └── entities/            # Domain entity unit tests
│           ├── event_test.go    # Event entity tests
│           └── user_test.go     # User entity tests
├── integration/                 # Integration tests
│   ├── event_test.go           # Event-related integration tests
│   └── user_test.go            # User-related integration tests
├── e2e/                        # End-to-end tests
│   ├── setup.go                # E2E test environment setup
│   ├── event_e2e_test.go       # Event-related E2E tests
│   └── README.md               # E2E test documentation
├── fixtures/                   # Test data and fixtures (future)
└── helpers/                    # Test utilities and configuration
    ├── test_config.go          # Test configuration
    └── test_utils.go           # Common test utilities
```

## Testing Approaches

### 🚀 **Local Development Testing** (Recommended for Development)

**When to use:**
- During active development
- Debugging issues
- Quick iteration and feedback

**Commands:**
```bash
# Run all tests locally
make test

# Run only unit tests
go test -v ./test/unit/...

# Run only integration tests
go test -v ./test/integration/...

# Run only E2E tests
go test -v ./test/e2e/...

# Run tests with coverage
make test-coverage
```

**Pros:**
- ⚡ Fastest execution
- 🔧 Easy debugging with IDE
- 💻 Direct access to logs
- 🚀 Quick iteration

### 🐳 **Docker-Based Testing** (Recommended for CI/CD)

**When to use:**
- CI/CD pipelines
- Integration testing with real database
- Ensuring environment consistency

**Commands:**
```bash
# Run all tests in Docker
make test-docker

# Start test database only
make test-start

# Run tests with PostgreSQL
make test-postgres

# Stop test database
make test-stop
```

**Pros:**
- 🔄 Consistent environment
- 🐳 Isolated testing
- 📋 Reproducible results
- 🚀 CI/CD ready

### 🔄 **Hybrid Approach** (Best Practice)

**Development Workflow:**
1. **Local testing** for development and debugging
2. **Docker testing** before commits
3. **CI/CD testing** for automated validation

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

## Running Tests

### Quick Start
```bash
# Start development database
make dev-start

# Run all tests
make test

# Run specific test types
go test -v ./test/unit/...      # Unit tests only
go test -v ./test/integration/... # Integration tests only
go test -v ./test/e2e/...       # E2E tests only
```

### Advanced Usage
```bash
# Run tests with coverage
make test-coverage

# Run tests in Docker environment
make test-docker

# Run tests with verbose output
go test -v -count=1 ./...
```

## Test Data Management

### Database Setup
- Integration and E2E tests use real PostgreSQL databases
- Test databases are isolated from development/production
- Automatic schema migration for test databases
- Clean state between test runs

### Test Fixtures
- Test data is created programmatically
- No external fixture files required
- UUIDs generated for unique test data
- Realistic data scenarios

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
go test -v -run TestCreateEvent_Success ./test/integration/...
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