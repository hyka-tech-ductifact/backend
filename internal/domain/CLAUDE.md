# Domain Layer - Hexagonal Architecture Rules

## Purpose
The domain layer contains the core business logic and rules. It's the heart of the hexagonal architecture and should be independent of any external concerns.

## Layer Rules

### ✅ ALLOWED
- Business entities with their core logic
- Value objects that encapsulate domain concepts
- Domain events for important business occurrences
- Repository interfaces (ports) for data persistence
- Domain services for complex business operations
- Aggregates that maintain consistency boundaries
- Domain exceptions for business rule violations

### ❌ FORBIDDEN
- Database dependencies (GORM, SQL drivers, etc.)
- HTTP/REST dependencies (Gin, HTTP handlers, etc.)
- External API clients
- Framework-specific code
- Infrastructure concerns (logging, caching, etc.)
- Application layer dependencies
- Interface layer dependencies

## Key Principles

1. **Dependency Inversion**: Domain defines interfaces, infrastructure implements them
2. **Business Logic Focus**: Only contains pure business rules and logic
3. **Framework Independence**: No external framework dependencies
4. **Testability**: Easy to unit test without external dependencies
5. **Stability**: Changes here should be rare and driven by business needs

## Directory Structure
```
domain/
├── entities/        # Core business objects
├── repositories/    # Data access interfaces
├── valueobjects/   # Immutable domain concepts
└── events/         # Domain events
```

## Example Imports (Allowed)
```go
import (
    "context"
    "time"
    "errors"
    "github.com/google/uuid"
)
```

## Dependencies Flow
```
Domain ← Application ← Infrastructure
Domain ← Application ← Interfaces
```

The domain should never import from application, infrastructure, or interfaces layers.