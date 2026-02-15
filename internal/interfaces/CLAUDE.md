# Interfaces Layer - Hexagonal Architecture Rules

## Purpose
The interfaces layer handles external communication protocols and user interactions. It translates external requests into application use cases and formats responses.

## Layer Rules

### ✅ ALLOWED
- HTTP handlers and controllers
- gRPC service implementations
- CLI command implementations
- WebSocket handlers
- Message queue consumers/producers
- Event listeners
- Request/Response DTOs
- Input validation and serialization
- Authentication and authorization middleware
- Application layer imports

### ❌ FORBIDDEN
- Business logic implementation
- Database access or queries
- Domain entity manipulation
- Infrastructure implementations
- Direct domain layer imports (use application layer)

## Key Principles

1. **Protocol Adaptation**: Adapts external protocols to application use cases
2. **Input/Output Handling**: Manages request parsing and response formatting
3. **Validation**: Validates external input before passing to application layer
4. **Error Translation**: Converts application errors to appropriate responses
5. **Authentication**: Handles authentication and authorization concerns

## Directory Structure
```
interfaces/
├── http/     # HTTP REST API handlers
├── grpc/     # gRPC service implementations  
└── cli/      # Command line interface
```

## Example Imports (Allowed)
```go
import (
    "backend-go-blueprint/internal/application/usecases"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "net/http"
)
```

## Dependencies Flow  
```
Interfaces → Application → Domain
```

The interfaces layer should only import from the application layer and never directly from domain or infrastructure layers.