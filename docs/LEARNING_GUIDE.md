# Backend Development Learning Guide - Event Service

## 🎯 Project Purpose

This **Event Service** is a learning project designed to understand and practice **backend development in Go** using **Hexagonal Architecture** (Ports & Adapters). The project focuses on creating a robust, maintainable, and testable backend service for managing football events.

### Learning Objectives
- Master Go programming for backend development
- Understand and implement Hexagonal Architecture principles
- Learn clean code practices and dependency injection
- Practice RESTful API design and implementation
- Gain experience with database integration using GORM
- Understand testing strategies for different architectural layers
- Learn proper error handling and validation patterns

---

## ⚠️ Hexagonal Architecture ≠ Clean Architecture

Before diving in, it's important to clarify: **this project uses Hexagonal Architecture, NOT Clean Architecture**. They are often confused because they share the same goal (isolate business logic from technical details), but they are **different architectural styles** created by different people.

| Aspect | Hexagonal (this project) | Clean Architecture |
|--------|--------------------------|-------------------|
| **Creator** | Alistair Cockburn (2005) | Robert C. Martin (2012) |
| **Core metaphor** | A hexagon with ports on the edges and adapters outside | Concentric circles/layers like an onion |
| **Key concepts** | Ports (interfaces) + Adapters (implementations) | 4 strict layers: Entities → Use Cases → Interface Adapters → Frameworks |
| **Structure** | Inside (domain + app) vs Outside (adapters) | 4 concentric rings with strict boundaries |
| **Use Cases** | Not a separate concept — lives inside application services | Explicit separate layer with its own rules |
| **DTOs** | Not required between layers | Strongly encouraged between every layer |
| **Complexity** | Simpler, fewer files and concepts | More structure, more files, stricter rules |
| **Best for** | Small-medium projects, microservices | Medium-large projects, complex monoliths |

**Why we chose Hexagonal:** It gives us all the benefits of separation of concerns with less boilerplate. If the project grows significantly, migrating to Clean Architecture is a mechanical refactor (split services into individual use cases, add DTOs between layers, promote HTTP handlers to their own layer).

---

## 🏗️ Architecture Overview

### The Hexagon Metaphor

Imagine your application as a **hexagon**. Inside the hexagon lives your business logic (the domain). On the edges of the hexagon there are **ports** (interfaces/contracts). Outside the hexagon there are **adapters** (concrete implementations that connect to the real world).

```
                    ┌──────────────────────────┐
                    │      Outside World        │
                    │  (HTTP, DB, APIs, CLI)    │
                    └──────────┬───────────────┘
                               │
                    ┌──────────▼───────────────┐
                    │    ADAPTERS               │
                    │  (implementations that    │
                    │   plug into the ports)    │
                    └──────────┬───────────────┘
                               │
                    ┌──────────▼───────────────┐
                    │    PORTS                  │
                    │  (interfaces/contracts)   │
                    └──────────┬───────────────┘
                               │
                    ┌──────────▼───────────────┐
                    │    APPLICATION CORE       │
                    │  (domain + services)      │
                    │  Pure business logic      │
                    └──────────────────────────┘
```

The rule is simple: **everything points inward**. Adapters know about ports, but ports don't know about adapters. The core doesn't know anything about the outside world.

### Two types of adapters

```
 ┌─────────────┐         ┌─────────────────────┐         ┌──────────────┐
 │   INBOUND   │────────►│   APPLICATION CORE   │────────►│   OUTBOUND   │
 │  ADAPTERS   │         │                     │         │  ADAPTERS    │
 │             │         │  ┌───────────────┐  │         │              │
 │ HTTP Handler│──uses──►│  │ Inbound Port  │  │         │              │
 │ gRPC Server │         │  │  (interface)  │  │         │              │
 │ CLI Command │         │  └───────┬───────┘  │         │              │
 │ Message     │         │          │          │         │              │
 │ Consumer    │         │  ┌───────▼───────┐  │         │              │
 │             │         │  │   Service     │  │         │              │
 │  (DRIVING)  │         │  │ (implements   │  │         │              │
 │             │         │  │  inbound port)│  │         │              │
 │             │         │  └───────┬───────┘  │         │              │
 │             │         │          │          │         │              │
 │             │         │  ┌───────▼───────┐  │         │              │
 │             │         │  │ Outbound Port │──uses──►│  PostgreSQL   │
 │             │         │  │  (interface)  │  │         │  Redis       │
 │             │         │  └───────────────┘  │         │  External API│
 │             │         │                     │         │              │
 │             │         │                     │         │  (DRIVEN)    │
 └─────────────┘         └─────────────────────┘         └──────────────┘
```

- **Inbound adapters (driving):** They *drive* your application. They receive an external stimulus (HTTP request, CLI command, message) and translate it into a call to the application core.
- **Outbound adapters (driven):** Your application *drives* them. When your business logic needs to persist data or call an external service, it uses an outbound port, and the adapter does the actual work.

---

## 📁 Project Structure

```
backend/
├── cmd/
│   └── api/
│       └── main.go                          # Entry point: wires everything together
│
├── internal/
│   ├── domain/                              # 🔵 THE CORE — pure business logic
│   │   ├── entities/
│   │   │   ├── event.go                     #   Domain entity
│   │   │   └── user.go                      #   Domain entity
│   │   ├── repositories/
│   │   │   └── event_repository.go          #   OUTBOUND PORT (interface)
│   │   └── valueobjects/
│   │       └── email.go                     #   Value Object
│   │
│   ├── application/                         # 🟢 APPLICATION — orchestration
│   │   ├── ports/
│   │   │   └── event_service.go             #   INBOUND PORT (interface)
│   │   └── services/
│   │       └── event_service.go             #   Service (implements inbound port)
│   │
│   └── infrastructure/                      # 🟠 ADAPTERS — connect to the real world
│       └── adapters/
│           ├── inbound/
│           │   └── http/
│           │       ├── event_handler.go      #   INBOUND ADAPTER (HTTP → Service)
│           │       └── router.go             #   HTTP routing configuration
│           └── outbound/
│               └── persistence/
│                   ├── connection.go          #   Database connection setup
│                   └── postgres_event_repo.go #   OUTBOUND ADAPTER (Service → PostgreSQL)
│
├── test/                                     # Tests organized by type
│   ├── unit/
│   ├── integration/
│   └── e2e/
│
└── docs/
    ├── LEARNING_GUIDE.md                     # This file
    └── LEARNING_CHECKLIST.md                 # Learning roadmap
```

---

## 🧩 Each element explained in detail

### 1. Domain Layer (`internal/domain/`)

> **What it is:** The heart of your application. Pure business logic that exists even without a web framework, database, or any technical tool.
>
> **Rule:** This layer has **ZERO external dependencies**. No Gin, no GORM, no HTTP concepts. Only Go standard library and domain-specific libraries (like `uuid`).

#### 1.1 Entities (`domain/entities/`)

An entity is a **business object with identity**. Two events are different even if they have the same title — because they have different IDs.

```go
// domain/entities/event.go
type Event struct {
    ID          uuid.UUID
    Title       string
    Description string
    Location    string
    StartTime   time.Time
    EndTime     time.Time
    OrganizerID uuid.UUID
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

func NewEvent(title, description, location string, ...) *Event {
    return &Event{
        ID:    uuid.New(),     // business rule: every event gets a unique ID
        Title: title,
        // ...
    }
}
```

**Key points:**
- Entities contain **business rules** (validation, state transitions)
- They are framework-agnostic: no `json:` tags, no `gorm:` tags here
- The constructor `NewEvent()` ensures you can never create an invalid entity

#### 1.2 Value Objects (`domain/valueobjects/`)

A value object is defined by its **value, not its identity**. Two emails with the same string are the same email. They are immutable and self-validating.

```go
// domain/valueobjects/email.go
type Email struct {
    value string    // unexported: can only be created through constructor
}

func NewEmail(email string) (*Email, error) {
    if !emailRegex.MatchString(email) {
        return nil, errors.New("invalid email format")
    }
    return &Email{value: email}, nil
}
```

**Key points:**
- Immutable: once created, the value cannot change
- Self-validating: impossible to have an invalid Email object
- No identity: compared by value, not by ID

#### 1.3 Repository Interfaces — OUTBOUND PORTS (`domain/repositories/`)

This is where the **outbound port** pattern lives. The domain declares *what it needs* (an interface), without knowing *how* it's implemented.

```go
// domain/repositories/event_repository.go
type EventRepository interface {
    Create(ctx context.Context, event *entities.Event) error
    GetByID(ctx context.Context, id uuid.UUID) (*entities.Event, error)
    Update(ctx context.Context, event *entities.Event) error
    Delete(ctx context.Context, id uuid.UUID) error
    List(ctx context.Context, limit, offset int) ([]*entities.Event, error)
}
```

**Key points:**
- This is a **port**: a contract/interface that defines what the domain needs
- The domain says "I need something that can save and retrieve events" — it doesn't care if it's PostgreSQL, MongoDB, or an in-memory map
- The interface lives in the domain because the **domain owns the contract**
- This enables **Dependency Inversion**: high-level business logic doesn't depend on low-level database details

> **Why is it called an "outbound port"?** Because data flows *outward* from the application core toward the external system (database). The application says "save this event" and the data goes *out* to PostgreSQL.

---

### 2. Application Layer (`internal/application/`)

> **What it is:** The orchestration layer. It coordinates domain entities and outbound ports to fulfill business operations. It's the "glue" between the outside world and the domain.
>
> **Rule:** Can import `domain/`. Cannot import `infrastructure/`. Does not know about HTTP, databases, or any framework.

#### 2.1 Inbound Port (`application/ports/`)

The inbound port is the **interface that defines what the application can do**. Inbound adapters (HTTP handlers, gRPC, CLI) depend on this interface.

```go
// application/ports/event_service.go
type EventService interface {
    CreateEvent(ctx context.Context, event *entities.Event) (*entities.Event, error)
    GetEventByID(ctx context.Context, id uuid.UUID) (*entities.Event, error)
}
```

**Key points:**
- Defines the **application's capabilities** as an interface
- Inbound adapters depend on this port, not on the concrete service
- This means you can swap the implementation (e.g., for testing) without changing the adapters
- Works with domain entities (`*entities.Event`), not HTTP request/response objects

> **Why is it called an "inbound port"?** Because requests flow *inward* from the outside world into the application core. An HTTP request comes *in* through this port.

#### 2.2 Application Service (`application/services/`)

The service **implements the inbound port**. It orchestrates the domain layer and outbound ports to fulfill business operations.

```go
// application/services/event_service.go
type eventService struct {
    eventRepo repositories.EventRepository  // depends on outbound PORT, not on PostgreSQL
}

func NewEventService(eventRepo repositories.EventRepository) *eventService {
    return &eventService{eventRepo: eventRepo}
}

func (s *eventService) CreateEvent(ctx context.Context, event *entities.Event) (*entities.Event, error) {
    // Business rule validation
    if event.EndTime.Before(event.StartTime) {
        return nil, ErrInvalidEventDuration
    }
    // Delegate to outbound port
    err := s.eventRepo.Create(ctx, event)
    if err != nil {
        return nil, err
    }
    return event, nil
}
```

**Key points:**
- Implements `ports.EventService` (the inbound port)
- Depends on `repositories.EventRepository` (the outbound port) — **never** on `PostgresEventRepository`
- Contains **application-level** business rules (validation, orchestration)
- The struct is **unexported** (`eventService`, not `EventService`) — the outside world interacts through the port interface, enforcing the contract
- Constructor `NewEventService()` receives interfaces, enabling dependency injection

---

### 3. Infrastructure Layer (`internal/infrastructure/adapters/`)

> **What it is:** The adapters that connect your application to the real world. They translate between external formats (HTTP JSON, SQL rows) and domain objects.
>
> **Rule:** Can import `domain/` and `application/`. This is the only layer that contains framework-specific code (Gin, GORM, etc.).

#### 3.1 Inbound Adapters (`adapters/inbound/http/`)

Inbound adapters **receive external requests and translate them** into calls to the inbound port. They are the "driving" side — they drive the application.

```go
// infrastructure/adapters/inbound/http/event_handler.go

// HTTP-specific DTOs — these belong HERE, not in domain or application
type CreateEventRequest struct {
    Title       string    `json:"title" binding:"required"`
    Description string    `json:"description"`
    Location    string    `json:"location" binding:"required"`
    StartTime   time.Time `json:"start_time" binding:"required"`
    EndTime     time.Time `json:"end_time" binding:"required"`
    OrganizerID uuid.UUID `json:"organizer_id" binding:"required"`
}

type EventResponse struct {
    ID          uuid.UUID `json:"id"`
    Title       string    `json:"title"`
    // ...
}

type EventHandler struct {
    eventService ports.EventService  // depends on INBOUND PORT, not concrete service
}

func (h *EventHandler) CreateEvent(c *gin.Context) {
    // 1. Parse HTTP request (external format)
    var req CreateEventRequest
    c.ShouldBindJSON(&req)

    // 2. Translate to domain entity
    event := entities.NewEvent(req.Title, req.Description, ...)

    // 3. Call inbound port
    created, err := h.eventService.CreateEvent(c.Request.Context(), event)

    // 4. Translate domain entity to HTTP response (external format)
    c.JSON(http.StatusCreated, toEventResponse(created))
}
```

**Key points:**
- DTOs (`CreateEventRequest`, `EventResponse`) live **here**, not in domain or application — they are HTTP-specific concerns with `json:` and `binding:` tags
- Depends on `ports.EventService` (the inbound port interface), NOT on `*services.eventService`
- Handles HTTP-specific concerns: status codes, JSON parsing, error formatting
- Translates between the HTTP world and the domain world
- Does NOT contain business logic — only translation and delegation

#### 3.2 Outbound Adapters (`adapters/outbound/persistence/`)

Outbound adapters **implement the outbound ports** defined in the domain. They translate between domain objects and external system formats (SQL rows, API calls, etc.).

```go
// infrastructure/adapters/outbound/persistence/postgres_event_repository.go

// Database-specific model — lives HERE, not in domain
type EventModel struct {
    ID          uuid.UUID `gorm:"type:uuid;primary_key"`
    Title       string    `gorm:"not null"`
    // ...
}

type PostgresEventRepository struct {
    db *gorm.DB
}

func (r *PostgresEventRepository) Create(ctx context.Context, event *entities.Event) error {
    // Translate domain entity → database model
    model := &EventModel{
        ID:    event.ID,
        Title: event.Title,
        // ...
    }
    return r.db.WithContext(ctx).Create(model).Error
}

func (r *PostgresEventRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Event, error) {
    var model EventModel
    r.db.WithContext(ctx).Where("id = ?", id).First(&model)
    // Translate database model → domain entity
    return r.toDomainEntity(&model), nil
}
```

**Key points:**
- **Implements** `repositories.EventRepository` (the outbound port defined in the domain)
- Database model (`EventModel`) with GORM tags lives **here**, not in domain — it's a persistence concern
- Translates between domain entities and database models in both directions
- Contains ALL database-specific code (SQL, GORM queries, connection handling)
- If you switch from PostgreSQL to MongoDB, you only change this adapter — nothing else changes

---

### 4. Entry Point (`cmd/api/main.go`)

> **What it is:** The composition root. This is where you wire everything together using dependency injection.

```go
func main() {
    // 1. Create outbound adapter (PostgreSQL)
    db, _ := persistence.NewPostgresConnection()
    eventRepo := persistence.NewPostgresEventRepository(db)

    // 2. Create application service, injecting the outbound port
    eventService := services.NewEventService(eventRepo)

    // 3. Create inbound adapter (HTTP), injecting the inbound port
    router := httpAdapter.SetupRoutes(eventService)

    // 4. Start
    router.Run(":8080")
}
```

**Key points:**
- This is the **only place** that knows about ALL concrete types
- It wires outbound adapters → services → inbound adapters
- If you want to swap PostgreSQL for MongoDB, you only change this file and create the new adapter

---

## 🔄 Dependency Flow — The Complete Picture

### Who depends on whom

```
┌─────────────────────────────────────────────────────────────────────────┐
│                                                                         │
│   ┌─────────────────┐         ┌─────────────────┐                      │
│   │  Inbound Adapter│         │ Outbound Adapter │                      │
│   │  (HTTP Handler) │         │ (PostgreSQL Repo)│                      │
│   └────────┬────────┘         └────────▲─────────┘                      │
│            │ depends on                │ implements                      │
│            ▼                           │                                 │
│   ┌─────────────────┐         ┌───────┴──────────┐                      │
│   │  Inbound Port   │         │  Outbound Port   │                      │
│   │  (EventService  │         │  (EventRepository│                      │
│   │   interface)    │         │   interface)      │                      │
│   └────────▲────────┘         └────────▲─────────┘                      │
│            │ implements                │ depends on                      │
│            │                           │                                 │
│   ┌────────┴─────────────────────────┬─┘                                │
│   │       Application Service        │                                  │
│   │       (eventService struct)      │                                  │
│   └──────────────┬───────────────────┘                                  │
│                  │ uses                                                  │
│                  ▼                                                       │
│   ┌──────────────────────────────────┐                                  │
│   │         Domain Entities          │                                  │
│   │     (Event, User, Email VO)      │                                  │
│   └──────────────────────────────────┘                                  │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### The dependency rule in one sentence

> **Everything depends on the domain. The domain depends on nothing.**

### Import rules per package

| Package | Can import | Cannot import |
|---------|-----------|---------------|
| `domain/entities` | Standard library only | Everything else |
| `domain/valueobjects` | Standard library only | Everything else |
| `domain/repositories` | `domain/entities` | `application/`, `infrastructure/` |
| `application/ports` | `domain/entities` | `infrastructure/` |
| `application/services` | `domain/entities`, `domain/repositories`, `application/ports` | `infrastructure/` |
| `infrastructure/adapters/inbound` | `application/ports`, `domain/entities` | `application/services`, `outbound/` |
| `infrastructure/adapters/outbound` | `domain/entities`, `domain/repositories` | `application/`, `inbound/` |
| `cmd/api/main.go` | **Everything** (composition root) | — |

### Data flow for a CREATE request

```
HTTP POST /events
    │
    ▼
[1] Inbound Adapter (EventHandler.CreateEvent)
    │   - Parses JSON → CreateEventRequest (DTO)
    │   - Translates DTO → entities.Event (domain entity)
    │
    ▼
[2] Inbound Port (ports.EventService interface)
    │   - Handler calls eventService.CreateEvent(event)
    │
    ▼
[3] Application Service (services.eventService)
    │   - Validates business rules (end > start)
    │   - Calls outbound port: eventRepo.Create(event)
    │
    ▼
[4] Outbound Port (repositories.EventRepository interface)
    │   - Service calls repo.Create(event)
    │
    ▼
[5] Outbound Adapter (PostgresEventRepository)
    │   - Translates entities.Event → EventModel (DB model)
    │   - Executes INSERT INTO events ...
    │
    ▼
[6] PostgreSQL Database
    │
    ▼ (response travels back up)
    │
[5] Outbound Adapter: returns nil (no error)
[4] Outbound Port: returns nil
[3] Service: returns *entities.Event
[2] Inbound Port: returns *entities.Event
[1] Inbound Adapter: translates entities.Event → EventResponse (DTO)
    │   - Writes JSON response with 201 Created
    │
    ▼
HTTP Response { "id": "...", "title": "..." }
```

---

## 🧪 How Hexagonal Architecture enables testing

The port/adapter pattern makes testing natural:

### Unit tests (domain + services)
Mock the outbound ports:

```go
// Create a mock that implements EventRepository (outbound port)
type mockEventRepo struct {}
func (m *mockEventRepo) Create(ctx context.Context, event *entities.Event) error {
    return nil // simulate success
}

// Test the service in isolation
func TestCreateEvent(t *testing.T) {
    service := services.NewEventService(&mockEventRepo{})
    event := entities.NewEvent("Test", ...)
    result, err := service.CreateEvent(context.Background(), event)
    assert.NoError(t, err)
}
```

### Integration tests (with real DB)
Use the real outbound adapter with a test database:

```go
func TestCreateEvent_Integration(t *testing.T) {
    db := setupTestDB(t)
    repo := persistence.NewPostgresEventRepository(db)
    service := services.NewEventService(repo)
    router := httpAdapter.SetupRoutes(service)
    // test with httptest
}
```

### E2E tests (full stack)
Hit the real HTTP server running with real database.

**The key insight:** Because services depend on **interfaces** (ports), you can inject mocks for unit tests, real implementations for integration tests, and run the full stack for E2E tests — all without changing any production code.

---

## 🛠️ Technologies & Tools

### Core Technologies
- **Go 1.23+**: Programming language
- **Gin**: HTTP web framework (used in inbound HTTP adapter)
- **GORM**: ORM for database operations (used in outbound persistence adapter)
- **PostgreSQL**: Primary database
- **UUID**: For unique identifiers

### Architecture Patterns
- **Hexagonal Architecture (Ports & Adapters)**: Core architectural style
- **Dependency Injection**: Manual wiring in `main.go`
- **Repository Pattern**: Data access abstraction via outbound ports
- **Adapter Pattern**: Inbound and outbound adapters translate between worlds

---

## 📚 Learning Progress & Summaries

> **Note**: For the complete learning checklist and roadmap, see `LEARNING_CHECKLIST.md`

### ✅ Topics Mastered

#### 1. Go Programming Fundamentals

**Structs and Methods** ✅
- **What I learned**: How to define business entities using structs and attach behavior with methods
- **Implementation**: Created `Event` struct with fields for football events and `NewEvent()` constructor
- **Key insight**: Struct constructors ensure valid object creation and encapsulate business rules

**Interfaces** ✅
- **What I learned**: Interfaces define contracts for behavior without implementation details
- **Implementation**: `EventRepository` (outbound port) and `EventService` (inbound port)
- **Key insight**: Interfaces enable the port/adapter pattern — the foundation of hexagonal architecture

**Error Handling** ✅
- **What I learned**: Go's explicit error handling and proper error propagation patterns
- **Implementation**: Custom errors like `ErrInvalidEventDuration` in the service layer
- **Key insight**: Errors should be handled at appropriate layers and provide meaningful context

**JSON Handling** ✅
- **What I learned**: Marshaling/unmarshaling with struct tags for API communication
- **Implementation**: Request/response DTOs with JSON tags in the inbound HTTP adapter
- **Key insight**: DTOs belong in the adapter layer, not in the domain

**Package Organization** ✅
- **What I learned**: How to structure Go projects with hexagonal architecture
- **Implementation**: domain → application → infrastructure with inbound/outbound adapters
- **Key insight**: Package structure reflects the dependency rule — imports only point inward

#### 2. Backend Development Concepts

**RESTful API Design** ✅
- **What I learned**: REST principles, resource modeling, and HTTP semantics
- **Implementation**: `POST /events` for creation, `GET /events/:id` for retrieval
- **Key insight**: HTTP concerns (status codes, JSON) stay in the inbound adapter

**Database Integration** ✅
- **What I learned**: ORM usage, model mapping, and the repository pattern
- **Implementation**: GORM with PostgreSQL in the outbound persistence adapter
- **Key insight**: Database models (`EventModel`) are separate from domain entities (`Event`) — they live in different layers

**Dependency Injection** ✅
- **What I learned**: Manual DI for loose coupling and testability
- **Implementation**: `main.go` wires adapters → services → ports
- **Key insight**: Only `main.go` knows about all concrete types

#### 3. Hexagonal Architecture

**Ports and Adapters** ✅
- **What I learned**: Ports are interfaces that define boundaries. Adapters are implementations that connect to the real world
- **Implementation**: `EventService` (inbound port), `EventRepository` (outbound port), `EventHandler` (inbound adapter), `PostgresEventRepository` (outbound adapter)
- **Key insight**: The direction of the port determines its type. Inbound = outside drives your app. Outbound = your app drives the outside.

**Dependency Inversion** ✅
- **What I learned**: High-level modules don't depend on low-level modules. Both depend on abstractions.
- **Implementation**: The service depends on `repositories.EventRepository` interface, not on `PostgresEventRepository`
- **Key insight**: The interface belongs to the layer that *needs* it (the domain owns the repository interface, the application owns the service interface)

**Layer Isolation** ✅
- **What I learned**: Each layer has strict import rules enforced by Go's package system
- **Implementation**: Domain imports nothing external. Infrastructure implements domain interfaces.
- **Key insight**: If you accidentally import infrastructure from domain, the hexagon is broken

---

## 📝 Key Learning Insights

### Architecture Benefits Discovered
1. **Testability**: Mock any port to test any layer in isolation
2. **Flexibility**: Swap PostgreSQL for MongoDB by creating a new outbound adapter — zero changes elsewhere
3. **Maintainability**: Clear boundaries make it obvious where new code should go
4. **Onboarding**: New developers can understand the system by reading the ports (interfaces)

### Go-Specific Patterns Learned
1. **Constructor Pattern**: `NewXxx()` functions for safe object creation
2. **Interface Segregation**: Small, focused interfaces (Go's implicit interface implementation helps)
3. **Unexported structs, exported interfaces**: Services are unexported, ports are exported — forces usage through interfaces
4. **Package-level encapsulation**: Go's package system naturally enforces hexagonal boundaries

### Mental Model

When in doubt about where code belongs, ask:

| Question | Answer |
|----------|--------|
| Is it a business rule? | → `domain/` |
| Is it orchestration of business operations? | → `application/services/` |
| Does it define what the app can do? | → `application/ports/` (inbound port) |
| Does it define what the app needs? | → `domain/repositories/` (outbound port) |
| Does it receive external input? | → `infrastructure/adapters/inbound/` |
| Does it talk to an external system? | → `infrastructure/adapters/outbound/` |
| Does it wire everything together? | → `cmd/api/main.go` |

---

## 🤝 Contributing to Learning

This document serves as both a learning guide and project documentation. As new features are added and concepts are learned, this guide will be updated to reflect new patterns and insights discovered while building this event service with Hexagonal Architecture.