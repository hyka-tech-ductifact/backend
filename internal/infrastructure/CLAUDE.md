# Infrastructure Layer - Hexagonal Architecture Rules

## Purpose
The infrastructure layer provides concrete implementations of domain interfaces and handles all external concerns like databases, messaging, file systems, and external APIs.

## Layer Rules

### ✅ ALLOWED
- Database implementations (GORM, SQL drivers)
- Message queue implementations (RabbitMQ, Kafka)
- External API clients
- File system operations
- Caching implementations (Redis, Memcached)
- Email service implementations
- Logging and monitoring setup
- Configuration management
- Repository implementations
- Domain and application layer imports

### ❌ FORBIDDEN
- Business logic implementation
- Use case orchestration
- HTTP request/response handling
- Presentation logic
- Direct interface layer imports

## Key Principles

1. **Implementation Details**: Contains all technical implementation details
2. **Adapter Pattern**: Adapts external systems to domain interfaces  
3. **Dependency Implementation**: Implements ports defined in application layer
4. **External Integration**: Handles all external system communications
5. **Technical Concerns**: Manages logging, caching, persistence, etc.

## Directory Structure
```
infrastructure/
├── adapters/     # External system adapters
├── database/     # Database implementations
├── messaging/    # Message queue implementations
└── external/     # External API clients
```

## Example Imports (Allowed)
```go
import (
    "backend-go-blueprint/internal/domain/entities"
    "backend-go-blueprint/internal/domain/repositories"
    "gorm.io/gorm"
    "gorm.io/driver/postgres"
    "github.com/google/uuid"
)
```

## Dependencies Flow
```
Infrastructure → Application → Domain
Infrastructure → Domain (for entities/interfaces)
```

The infrastructure layer can import from domain and application layers but should never import from interfaces layer.