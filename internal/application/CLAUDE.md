# Application Layer - Hexagonal Architecture Rules

## Purpose
The application layer orchestrates domain objects to fulfill business use cases. It defines the application's behavior and coordinates between domain and infrastructure.

## Layer Rules

### ✅ ALLOWED
- Use cases that implement business workflows
- Application services for complex operations
- Port interfaces for external dependencies
- Command and Query handlers
- Application-specific DTOs
- Transaction coordination
- Security and authorization logic
- Validation of input data
- Domain layer imports

### ❌ FORBIDDEN
- Database implementations (GORM models, SQL queries)
- HTTP/REST implementations (Gin handlers, HTTP logic)
- External API client implementations
- Framework-specific infrastructure code
- UI/presentation logic
- Direct infrastructure dependencies

## Key Principles

1. **Use Case Driven**: Each use case represents a business operation
2. **Orchestration**: Coordinates domain objects to fulfill business goals
3. **Port Definition**: Defines interfaces for external dependencies
4. **Transaction Boundaries**: Manages transaction scopes
5. **Input Validation**: Validates and sanitizes external input

## Directory Structure
```
application/
├── usecases/    # Business use case implementations
├── services/    # Application services
└── ports/       # Interfaces for external dependencies
```

## Example Imports (Allowed)
```go
import (
    "context"
    "backend-go-blueprint/internal/domain/entities"
    "backend-go-blueprint/internal/domain/repositories"
    "github.com/google/uuid"
)
```

## Dependencies Flow
```
Application → Domain
Infrastructure → Application (implements ports)
Interfaces → Application
```

The application layer can import from domain but should never import from infrastructure or interfaces layers.