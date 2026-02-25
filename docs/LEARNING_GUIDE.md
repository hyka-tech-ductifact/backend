# Backend Development Learning Guide - Event Service

## 🎯 Project Purpose

This **Event Service** is a learning project designed to understand and practice **backend development in Go** using **Hexagonal Architecture** (also known as Ports and Adapters). The project focuses on creating a robust, maintainable, and testable backend service for managing football events.

### Learning Objectives
- Master Go programming for backend development
- Understand and implement Hexagonal Architecture principles
- Learn clean code practices and dependency injection
- Practice RESTful API design and implementation
- Gain experience with database integration using GORM
- Understand testing strategies for different architectural layers
- Learn proper error handling and validation patterns

---

## 🏗️ Architecture Overview

This project follows **Hexagonal Architecture** to maintain clean separation of concerns and ensure testability.

### Architecture Layers

```
┌─────────────────────────────────────────────────────────────┐
│                    INTERFACES LAYER                         │
│  (HTTP Handlers, CLI, gRPC - External Communication)       │
├─────────────────────────────────────────────────────────────┤
│                   APPLICATION LAYER                         │
│     (Use Cases, Business Orchestration, DTOs)              │
├─────────────────────────────────────────────────────────────┤
│                     DOMAIN LAYER                            │
│  (Entities, Business Rules, Repository Interfaces)         │
├─────────────────────────────────────────────────────────────┤
│                 INFRASTRUCTURE LAYER                        │
│  (Database, External APIs, Framework Implementations)      │
└─────────────────────────────────────────────────────────────┘
```

### Dependency Flow
```
Interfaces → Application → Domain ← Infrastructure
```
- **Domain** is the core and doesn't depend on anything
- **Application** orchestrates domain logic
- **Infrastructure** implements domain interfaces
- **Interfaces** handle external communication

---

## 📁 Project Structure

```
event-service/
├── cmd/
│   └── api/                    # Application entry point
│       └── main.go            # Dependency injection & server setup
├── internal/
│   ├── domain/                 # 🔵 DOMAIN LAYER
│   │   ├── entities/          # Business entities (Event, User)
│   │   ├── repositories/      # Repository interfaces (ports)
│   │   └── valueobjects/      # Value objects
│   ├── application/           # 🟢 APPLICATION LAYER
│   │   ├── usecases/         # Business use cases
│   │   └── ports/            # Port interfaces
│   ├── infrastructure/        # 🟠 INFRASTRUCTURE LAYER
│   │   └── adapters/
│   │       └── out/database/  # Database implementations
│   └── interfaces/           # 🟡 INTERFACES LAYER
│       └── http/             # HTTP handlers and routing
├── examples.http             # API testing examples
├── docker-compose.yml        # Database setup
└── LEARNING_GUIDE.md        # This guide
```

---

## 🧩 Key Components Implemented

### 1. Domain Layer (`internal/domain/`)
**Purpose**: Contains pure business logic, entities, and business rules.

#### Entities
- **`Event`**: Core business entity representing football events
  - Fields: ID, Title, Description, Location, StartTime, EndTime, OrganizerID
  - Constructor: `NewEvent()` with business logic

#### Repository Interfaces
- **`EventRepository`**: Defines data access contracts
  - Methods: Create, GetByID, Update, Delete, List, GetByOrganizerID

### 2. Application Layer (`internal/application/`)
**Purpose**: Orchestrates business workflows and use cases.

#### Use Cases
- **`EventUseCase`**: Handles event-related business operations
  - `CreateEvent()`: Validates and creates events
  - `GetEventByID()`: Retrieves events
  - Business validation (e.g., end time after start time)

#### DTOs (Data Transfer Objects)
- **`CreateEventRequest`**: Input validation and binding
- **`EventResponse`**: Standardized output format

### 3. Infrastructure Layer (`internal/infrastructure/`)
**Purpose**: Implements technical concerns and external integrations.

#### Database Adapters
- **`PostgresEventRepository`**: GORM-based PostgreSQL implementation
- **`EventModel`**: Database mapping model
- Auto-migration support

### 4. Interfaces Layer (`internal/interfaces/`)
**Purpose**: Handles external communication protocols.

#### HTTP Handlers
- **`EventHandler`**: RESTful API endpoints
  - `POST /events`: Create football events
  - `GET /events/:id`: Retrieve events
- Input validation and error handling
- HTTP status code management

---

## 🛠️ Technologies & Tools

### Core Technologies
- **Go 1.21+**: Programming language
- **Gin**: HTTP web framework
- **GORM**: ORM for database operations
- **PostgreSQL**: Primary database
- **UUID**: For unique identifiers

### Development Tools
- **Docker & Docker Compose**: Database containerization
- **REST Client**: API testing (examples.http)
- **Go Modules**: Dependency management

### Architecture Patterns
- **Hexagonal Architecture**: Clean separation of concerns
- **Dependency Injection**: Loose coupling between layers
- **Repository Pattern**: Data access abstraction
- **DTO Pattern**: Input/output data transformation

---

## 🚀 Getting Started

### Prerequisites
- Go 1.21 or higher
- Docker and Docker Compose
- REST client (VS Code REST Client extension recommended)

### Setup Instructions

1. **Start the database**:
   ```bash
   docker compose up -d
   ```

2. **Run the application**:
   ```bash
   go run cmd/api/main.go
   ```

3. **Test the API**:
   - Use the examples in `examples.http`
   - Create football events via `POST /events`
   - Retrieve events via `GET /events/:id`

---

## 📚 Learning Progress & Summaries

> **Note**: For the complete learning checklist and roadmap, see `LEARNING_CHECKLIST.md`

### ✅ Topics Mastered

#### 1. Go Programming Fundamentals

**Structs and Methods** ✅
- **What I learned**: How to define business entities using structs and attach behavior with methods
- **Implementation**: Created `Event` struct with fields for football events and `NewEvent()` constructor
- **Key insight**: Struct constructors ensure valid object creation and encapsulate business rules
- **Example**: `func NewEvent(title, description, location string, startTime, endTime time.Time, organizerID uuid.UUID) *Event`

**Interfaces** ✅
- **What I learned**: Interfaces define contracts for behavior without implementation details
- **Implementation**: `EventRepository` interface defines data access methods
- **Key insight**: Interfaces enable dependency inversion and make testing easier
- **Example**: Repository interface allows switching between PostgreSQL, MongoDB, or mock implementations

**Error Handling** ✅
- **What I learned**: Go's explicit error handling and proper error propagation patterns
- **Implementation**: Custom errors like `ErrInvalidEventDuration` and HTTP error responses
- **Key insight**: Errors should be handled at appropriate layers and provide meaningful context
- **Example**: Validation errors return 400, business logic errors return appropriate HTTP codes

**JSON Handling** ✅
- **What I learned**: Marshaling/unmarshaling with struct tags for API communication
- **Implementation**: Request/response DTOs with JSON tags and validation
- **Key insight**: Separate internal models from API models for flexibility
- **Example**: `CreateEventRequest` with `json:"title" binding:"required"` tags

**Package Organization** ✅
- **What I learned**: How to structure Go projects with clean package boundaries
- **Implementation**: Hexagonal architecture with domain, application, infrastructure, interfaces layers
- **Key insight**: Package structure should reflect architectural decisions and dependency flow

#### 2. Backend Development Concepts

**RESTful API Design** ✅
- **What I learned**: REST principles, resource modeling, and HTTP semantics
- **Implementation**: `POST /events` for creation, `GET /events/:id` for retrieval
- **Key insight**: URLs represent resources, HTTP methods represent actions
- **Example**: Football events as resources with proper CRUD operations

**Input Validation** ✅
- **What I learned**: Multiple validation layers - structural, business rules, and constraints
- **Implementation**: Gin binding for structural validation, custom business logic validation
- **Key insight**: Validate early and provide clear error messages
- **Example**: Validating that event end time is after start time

**Database Integration** ✅
- **What I learned**: ORM usage, model mapping, and database operations
- **Implementation**: GORM with PostgreSQL, auto-migrations, and repository pattern
- **Key insight**: Separate database models from domain entities for flexibility
- **Example**: `EventModel` with GORM tags vs `Event` domain entity

**Dependency Injection** ✅
- **What I learned**: Manual DI for loose coupling and testability
- **Implementation**: Constructor injection in `main.go` wiring up all dependencies
- **Key insight**: Dependencies flow inward toward the domain layer
- **Example**: `NewEventUseCase(eventRepo)` receiving interface, not concrete type

#### 3. Hexagonal Architecture

**Layer Separation** ✅
- **What I learned**: Each layer has specific responsibilities and clean boundaries
- **Implementation**: Domain (business logic), Application (orchestration), Infrastructure (technical), Interfaces (communication)
- **Key insight**: Dependencies only flow inward, domain layer is isolated
- **Example**: Domain entities don't import any external frameworks

**Dependency Inversion** ✅
- **What I learned**: High-level modules don't depend on low-level modules
- **Implementation**: Domain defines repository interfaces, infrastructure implements them
- **Key insight**: Interfaces belong to the client (domain), not the implementation
- **Example**: `EventRepository` interface in domain, `PostgresEventRepository` in infrastructure

**Ports and Adapters** ✅
- **What I learned**: Ports are interfaces, adapters are implementations
- **Implementation**: Repository interfaces as ports, database implementations as adapters
- **Key insight**: Multiple adapters can implement the same port
- **Example**: Could have PostgreSQL, MongoDB, or in-memory adapters for same repository port

### 📝 Key Learning Insights

#### Architecture Benefits Discovered
1. **Testability**: Each layer can be tested in isolation with mocks
2. **Flexibility**: Can swap implementations without changing business logic
3. **Maintainability**: Clear separation makes code easier to understand and modify
4. **Scalability**: Architecture supports adding new features without breaking existing code

#### Go-Specific Patterns Learned
1. **Constructor Pattern**: `NewXxx()` functions for object creation
2. **Interface Segregation**: Small, focused interfaces for specific behaviors
3. **Error Wrapping**: Adding context to errors as they propagate up layers
4. **Struct Embedding**: Composing behavior rather than inheritance

#### Best Practices Adopted
1. **Package Naming**: Clear, descriptive package names that reflect purpose
2. **Interface Design**: Interfaces should be small and focused on specific behaviors
3. **Error Messages**: Descriptive errors that help debugging and user experience
4. **Validation Strategy**: Multiple validation layers for comprehensive input checking

### 🔍 Current Understanding Level

**Beginner → Intermediate Progress**
- ✅ Can structure a Go project with clean architecture
- ✅ Can implement CRUD operations with proper error handling
- ✅ Can design REST APIs following standard conventions
- ✅ Can integrate with databases using ORM patterns
- ✅ Can apply dependency injection for loose coupling

**Next Learning Focus**
- Testing strategies and test-driven development
- Middleware implementation for cross-cutting concerns
- Advanced Go concurrency patterns
- Performance optimization and profiling

---

## 📝 Notes and Learnings

### Key Insights
- **Hexagonal Architecture** provides excellent testability and maintainability
- **Dependency Injection** makes the system flexible and loosely coupled
- **Repository Pattern** abstracts data access and enables easy testing
- **Clean separation** between business logic and technical concerns

### Common Patterns Learned
1. **Entity Constructors**: `NewEvent()` ensures valid object creation
2. **Error Propagation**: Consistent error handling across layers
3. **DTO Mapping**: Clean separation between internal and external models
4. **Interface Segregation**: Small, focused interfaces

---

## 🤝 Contributing to Learning

This document serves as both a learning guide and project documentation. As new features are added and concepts are learned, this guide will be updated to reflect:

- New architectural patterns implemented
- Additional Go concepts mastered
- Best practices discovered
- Challenges overcome and solutions found

The goal is to create a comprehensive reference for Go backend development using hexagonal architecture, based on real hands-on experience building this event service.