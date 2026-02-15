# End-to-End (E2E) Tests

## Overview

E2E tests verify that the entire system works together as a complete unit, testing real user workflows and system integration scenarios.

## Key Differences from Integration Tests

| Aspect | Integration Tests | E2E Tests |
|--------|------------------|-----------|
| **Scope** | Individual components working together | Complete system from HTTP to database |
| **HTTP Testing** | `httptest.NewRecorder()` (in-memory) | Real HTTP requests to running server |
| **Database** | Real PostgreSQL | Real PostgreSQL |
| **Speed** | Fast (no network overhead) | Slower (network overhead) |
| **Purpose** | Verify business logic | Verify complete user workflows |
| **Test Focus** | Individual API endpoints | Complete scenarios and system resilience |

## E2E Test Categories

### 1. **Complete User Workflows**
- Test multi-step user scenarios
- Verify data flows through the entire system
- Example: Creating multiple events and verifying they're all accessible

### 2. **System Health & Recovery**
- Test system resilience
- Verify graceful handling of malformed requests
- Test health endpoints and error scenarios

### 3. **Data Consistency**
- Verify data integrity across operations
- Test that data remains consistent over time
- Example: Retrieving the same event multiple times

## When to Use E2E Tests

✅ **Use E2E tests for:**
- Complete user workflows
- System integration verification
- Production-like environment testing
- CI/CD pipeline validation
- Regression testing of complete features

❌ **Don't use E2E tests for:**
- Individual business logic testing
- Fast feedback during development
- Unit-level validation
- Performance testing (use dedicated tools)

## Running E2E Tests

### Local Development
```bash
# Start services first
make dev-start

# Run E2E tests
go test -v ./test/e2e/...
```

### Docker Environment
```bash
# Start services for testing
make test-start

# Run E2E tests
go test -v ./test/e2e/...

# Stop services when done
make test-stop
```

## Test Structure

```
test/e2e/
├── setup.go              # Test environment setup
├── event_e2e_test.go     # Event-related E2E tests
└── README.md             # This file
```

## Best Practices

1. **Focus on Complete Scenarios**: Test user workflows, not individual API calls
2. **Test System Resilience**: Include error scenarios and edge cases
3. **Verify Data Consistency**: Ensure data integrity across operations
4. **Use Realistic Data**: Use data that represents real user scenarios
5. **Keep Tests Independent**: Each test should be able to run in isolation

## Example: Integration vs E2E Test

### Integration Test (test/integration/event_test.go)
```go
func TestCreateEvent_Success(t *testing.T) {
    router := helpers.SetupTestRouter(t)  // In-memory router
    
    // Test single API call
    req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer(jsonBody))
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)  // In-memory HTTP
    
    assert.Equal(t, http.StatusCreated, w.Code)
}
```

### E2E Test (test/e2e/event_e2e_test.go)
```go
func TestEventE2E_CompleteUserWorkflow(t *testing.T) {
    env := SetupTestEnvironment(t)  // Real running services
    
    // Test complete workflow
    // 1. Create multiple events
    // 2. Verify all are accessible
    // 3. Test system resilience
    
    resp, err := http.Post(env.APIBaseURL+"/events", ...)  // Real HTTP
    // ... complete scenario testing
}
```

## Integration with CI/CD

E2E tests are perfect for CI/CD pipelines because they:
- Test the complete system
- Verify production-like behavior
- Catch integration issues early
- Provide confidence in deployments

## Performance Considerations

- E2E tests are slower than integration tests
- Use them strategically in your test suite
- Consider running them less frequently than unit/integration tests
- Use parallel execution when possible
