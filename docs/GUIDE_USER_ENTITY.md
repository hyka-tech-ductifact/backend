# Guía: Crear la entidad Usuario desde cero

Guía completa para implementar un caso de uso nuevo en arquitectura hexagonal, con **mejores prácticas de Go idiomático**. Se construye de dentro hacia fuera: dominio → aplicación → infraestructura.

---

## Índice

1. [Filosofía y principios](#filosofía-y-principios)
2. [Paso 1 — Entidad de dominio](#paso-1--entidad-de-dominio)
3. [Paso 2 — Value Objects](#paso-2--value-objects)
4. [Paso 3 — Puerto de salida (Repository)](#paso-3--puerto-de-salida-repository)
5. [Paso 4 — Puerto de entrada (Service interface)](#paso-4--puerto-de-entrada-service-interface)
6. [Paso 5 — Servicio de aplicación](#paso-5--servicio-de-aplicación)
7. [Paso 6 — Adaptador de salida (PostgreSQL)](#paso-6--adaptador-de-salida-postgresql)
8. [Paso 7 — Adaptador de entrada (HTTP handler)](#paso-7--adaptador-de-entrada-http-handler)
9. [Paso 8 — Router (registrar rutas)](#paso-8--router-registrar-rutas)
10. [Paso 9 — Wiring en main.go](#paso-9--wiring-en-maingo)
11. [Paso 10 — Base de datos (SQL)](#paso-10--base-de-datos-sql)
12. [Resumen de archivos](#resumen-de-archivos)
13. [Errores comunes a evitar](#errores-comunes-a-evitar)

---

## Filosofía y principios

Antes de escribir código, las reglas fundamentales:

1. **Siempre de dentro hacia fuera.** Primero el dominio, al final la infraestructura. Nunca diseñes tu entidad pensando en la base de datos.
2. **El dominio NO depende de nada.** Nada de tags `json:`, nada de `gorm:`, nada de frameworks. Solo Go puro.
3. **La entidad se protege a sí misma.** Si un `User` no puede existir sin email válido, la propia entidad debe impedirlo. No confíes en que "el handler ya lo valida".
4. **Constructor obligatorio.** Nunca crear structs con `User{}` directamente. Siempre usar `NewUser(...)` que valida y devuelve error.
5. **Errores con sentido.** Definir errores de dominio como variables (`var ErrInvalidEmail = ...`), no strings sueltos. Así se pueden comparar con `errors.Is()`.
6. **Interfaces donde se consumen, no donde se implementan.** En Go, las interfaces pertenecen al consumidor.

---

## Paso 1 — Entidad de dominio

📁 `internal/domain/entities/user.go`

La entidad es el corazón. Representa *qué es un usuario* con sus reglas de negocio.

```go
package entities

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// --- Domain errors ---
// Se definen como variables para poder comparar con errors.Is().

var (
	ErrEmptyUserName = errors.New("user name cannot be empty")
)

// User is a domain entity. No framework tags (json, gorm, etc.).
// The struct fields are exported for mapper access, but construction
// MUST go through NewUser to guarantee invariants.
type User struct {
	ID        uuid.UUID
	Name      string
	Email     string // Stored as string, but validated via value object on creation
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewUser is the only way to create a valid User.
// It validates all business rules and returns an error if any fail.
func NewUser(name, email string) (*User, error) {
	if name == "" {
		return nil, ErrEmptyUserName
	}
	// Email validation is delegated to the Email value object (Step 2).
	// For now we accept any string — we'll plug in validation next.

	now := time.Now()
	return &User{
		ID:        uuid.New(),
		Name:      name,
		Email:     email,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
```

### 💡 Buenas prácticas aplicadas

| Aspecto | Patrón aplicado | Por qué |
|---------|----------------|--------|
| Handler no crea entidades de dominio | ✅ Pasa primitivos al servicio | El handler no debería conocer la estructura interna del dominio. |
| Manejo de errores | Usa `errors.Is()` para elegir status | 404 para no encontrado, 409 para duplicado, 500 para errores internos. |
| DTOs usan `string` para IDs | ✅ | En el JSON de respuesta, un UUID siempre es string. Mantener consistencia. |
| Update pasa punteros | ✅ Pasa `*string` opcionales | El handler solo transporta datos. La lógica de "qué cambió" es del servicio. |

### 💡 Consejo: ¿campos exportados o no?

En Go idiomático, los campos de una entidad suelen ser exportados (mayúscula) para permitir el acceso desde los mappers de infraestructura. **La protección no viene de ocultar campos, sino de forzar la construcción a través de `NewUser`**. Si quisieras campos privados, necesitarías getters, lo cual es anti-idiomático en Go salvo que tengas una razón fuerte.

---

## Paso 2 — Value Objects

📁 `internal/domain/valueobjects/email.go` (ya existe, lo usaremos)

Un Value Object encapsula una regla de validación. Es inmutable y se define por su valor, no por su identidad.

El archivo `email.go` ya tiene:

```go
package valueobjects

import (
	"errors"
	"regexp"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

type Email struct {
	value string
}

func NewEmail(email string) (*Email, error) {
	if !emailRegex.MatchString(email) {
		return nil, errors.New("invalid email format")
	}
	return &Email{value: email}, nil
}

func (e *Email) String() string {
	return e.value
}
```

### Ahora conectamos el Value Object a la entidad

Volvemos a `user.go` y añadimos la validación de email al constructor:

```go
package entities

import (
	"errors"
	"time"

	"ductifact/internal/domain/valueobjects"

	"github.com/google/uuid"
)

var (
	ErrEmptyUserName = errors.New("user name cannot be empty")
)

type User struct {
	ID        uuid.UUID
	Name      string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewUser(name, email string) (*User, error) {
	if name == "" {
		return nil, ErrEmptyUserName
	}

	// Validate email through the value object.
	// The VO is used for validation, but we store the raw string.
	// This avoids coupling the entity struct to the VO type.
	validEmail, err := valueobjects.NewEmail(email)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &User{
		ID:        uuid.New(),
		Name:      name,
		Email:     validEmail.String(),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
```

### 💡 Consejo: ¿por qué guardar `string` en vez del tipo `Email`?

Guardar `Email` como tipo en la entidad obligaría a todos los mappers a saber desempaquetar el VO, complicando la serialización. El patrón pragmático es: **validar con el VO, almacenar el valor primitivo**. La validación ocurre una vez en la construcción, y a partir de ahí el valor es confiable.

### 💡 Consejo: mejorar el error del Value Object

El Value Object actual devuelve `errors.New("invalid email format")`, que es un string anónimo. Sería mejor definir un error como variable:

```go
var ErrInvalidEmail = errors.New("invalid email format")
```

Así cualquier capa puede hacer `errors.Is(err, valueobjects.ErrInvalidEmail)`.

---

## Paso 3 — Puerto de salida (Repository)

📁 `internal/domain/repositories/user_repository.go`

El repositorio es una **interfaz** que define las operaciones de persistencia que el dominio necesita. No sabe de PostgreSQL, GORM, ni nada externo.

```go
package repositories

import (
	"context"

	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
)

// UserRepository is the outbound port for user persistence.
// It is defined in the domain but implemented in infrastructure.
type UserRepository interface {
	Create(ctx context.Context, user *entities.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
	Update(ctx context.Context, user *entities.User) error
}
```

### ¿Qué es `context.Context` y por qué aparece aquí?

`context.Context` es un tipo de la **biblioteca estándar de Go** (paquete `context`). No es de Gin, no es de GORM, no es de ningún framework. Es tan "de Go" como `string` o `error`.

**¿Pero no dijimos que el dominio no puede conocer infraestructura?** Correcto, y `context` NO es infraestructura:

| Import | ¿Es infraestructura? | ¿Permitido en dominio? |
|--------|----------------------|----------------------|
| `context` | ❌ No, es stdlib | ✅ Sí |
| `time` | ❌ No, es stdlib | ✅ Sí |
| `errors` | ❌ No, es stdlib | ✅ Sí |
| `github.com/google/uuid` | ⚠️ Externo, pero genérico | ✅ Aceptable |
| `github.com/gin-gonic/gin` | ✅ Sí, es HTTP | ❌ No |
| `gorm.io/gorm` | ✅ Sí, es BD | ❌ No |

**¿Qué representa a nivel conceptual?** Piénsalo como una pregunta de negocio, no técnica:

> *"¿Debería seguir ejecutando esta operación o alguien la canceló?"*

Analogía real: estás en la cola de una tienda. Llevas 30 minutos esperando. Decides irte → eso es un **context cancelado**. La tienda cierra a las 21:00 y son las 20:59 → eso es un **context con deadline**. La cancelación y los timeouts no son conceptos de HTTP ni de bases de datos — son conceptos de **cualquier operación que tarda tiempo**.

**¿Cómo funciona en la práctica?** El contexto se crea arriba del todo (en el handler HTTP) y se propaga hacia abajo:

```
HTTP Request llega
    → Gin crea un context con timeout de 30s
        → Se pasa al servicio: CreateUser(ctx, ...)
            → Se pasa al repositorio: Create(ctx, ...)
                → Se pasa a GORM: db.WithContext(ctx).Create(...)
                    → Si el usuario cierra el navegador,
                      ctx se cancela y la query de BD se aborta.
```

Sin context, si un usuario cierra la pestaña, el servidor seguiría ejecutando la query en la BD para nada — desperdiciando recursos.

**Reglas de uso de `context.Context` en Go:**
1. Siempre va como **primer parámetro** de la función.
2. **Nunca** se almacena en un struct.
3. **Nunca** se pasa `nil` — si no tienes uno, usa `context.TODO()` o `context.Background()`.

**La prueba de que no rompe la pureza del dominio:** si cambias de PostgreSQL a MongoDB, ¿sigue necesitando `context.Context`? Sí. ¿Y si cambias a una API externa? Sí. El contexto no depende de ninguna implementación concreta — es transversal.

### ¿Por qué `GetByEmail`?

Porque al crear un usuario querrás verificar que el email no esté ya en uso. Es una operación de negocio legítima. Además, el email funciona como un *identificador natural* del usuario.

### 💡 Consejo: interfaces pequeñas

En Go, las interfaces deben ser **lo más pequeñas posible**. Si tu caso de uso solo necesita `Create` y `GetByID`, no añadas `Delete` y `List` "por si acaso". Añádelos cuando los necesites. Es mucho más fácil añadir métodos a una interfaz que quitarlos.

---

## Paso 4 — Puerto de entrada (Service interface)

📁 `internal/application/usecases/user_service.go`

El caso de uso define **qué operaciones expone tu aplicación al mundo exterior**. Los adaptadores de entrada (HTTP handlers, CLI, gRPC) dependen de esta interfaz.

```go
package usecases

import (
	"context"

	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
)

// UserService is the inbound port for user operations.
// Inbound adapters (HTTP handlers, CLI, etc.) depend on this interface.
type UserService interface {
	CreateUser(ctx context.Context, name, email string) (*entities.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, name, email *string) (*entities.User, error)
}
```

### 💡 Buenas prácticas aplicadas

| Aspecto | Patrón aplicado | Por qué |
|---------|----------------|--------|
| `CreateUser` recibe | `name, email string` (primitivos) | El handler no debería construir entidades de dominio. Eso es responsabilidad del servicio. Recibir primitivos mantiene al handler ignorante del dominio. |
| `UpdateUser` recibe | `id` + punteros opcionales `*string` | Los punteros `*string` permiten distinguir "no enviado" (`nil`) de "enviado vacío" (`""`). Es el patrón estándar para *partial updates* en Go. |

### 💡 Consejo: punteros para campos opcionales en updates

```go
// *string = nil  → el campo NO se actualiza
// *string = ""   → el campo se actualiza a vacío (si las reglas lo permiten)
// *string = "x"  → el campo se actualiza a "x"
```

Esto te ahorra el problema de "¿el usuario envió vacío a propósito o simplemente no envió el campo?".

### 💡 Consejo: cuando los parámetros crecen, usa un Command

Con 2 parámetros (`name`, `email`), primitivos directos es lo ideal. Pero si mañana el usuario tiene 6+ campos (phone, address, country, bio...), la firma se vuelve inmanejable:

```go
// ❌ Demasiados parámetros — difícil de leer, fácil de confundir el orden
CreateUser(ctx context.Context, name, email, phone, address, country, bio string) (...)
```

La solución es crear un **Command** (un struct de aplicación, sin tags de framework):

```go
// internal/application/ports/user_service.go

type CreateUserCommand struct {
	Name    string
	Email   string
	Phone   string
	Address string
	Country string
	Bio     string
}

type UpdateUserCommand struct {
	Name    *string
	Email   *string
	Phone   *string
	Address *string
}

type UserService interface {
	CreateUser(ctx context.Context, cmd CreateUserCommand) (*entities.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, cmd UpdateUserCommand) (*entities.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
}
```

**Regla práctica:**

| Parámetros | Recomendación |
|-----------|---------------|
| 2-3 | ✅ Primitivos directos. Claro y simple. |
| 4-5 | ⚠️ Empieza a ser incómodo. Considera agrupar. |
| 6+ | ❌ Definitivamente usa un Command struct. |

**Detalles importantes del Command:**
- **Sin tags `json:`** — no es un DTO de HTTP, no conoce cómo llegan los datos.
- **Sin `ID`, sin `CreatedAt`** — solo contiene lo que el *cliente* envía, no lo que el sistema genera.
- **`*string` en Update** — mantiene el patrón de partial updates.
- **`GetUserByID` sigue con primitivo** — un solo `uuid.UUID` no necesita struct.

**¿Por qué NO pasar directamente el DTO de HTTP al servicio?**

```go
// ❌ MAL — el servicio depende de una estructura con tags json:
func CreateUser(ctx context.Context, req CreateUserRequest) ...

// ✅ BIEN — el servicio tiene su propio Command sin conocer HTTP
func CreateUser(ctx context.Context, cmd CreateUserCommand) ...
```

Si mañana creas un CLI o un consumidor de mensajes, pueden construir el mismo `CreateUserCommand` sin saber nada de JSON ni de HTTP.

**El handler simplemente traduce:**

```go
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest  // DTO HTTP (con tags json:)
	if err := c.ShouldBindJSON(&req); err != nil { ... }

	// HTTP DTO → Application Command
	cmd := ports.CreateUserCommand{
		Name:    req.Name,
		Email:   req.Email,
		Phone:   req.Phone,
		Address: req.Address,
	}

	user, err := h.userService.CreateUser(c.Request.Context(), cmd)
	// ...
}
```

**Para este proyecto ahora:** con solo `name` y `email`, 2 primitivos es perfecto. Cuando añadas más campos, refactoriza a Command. El cambio es mecánico y no afecta al dominio ni a la infraestructura.

---

## Paso 5 — Servicio de aplicación

📁 `internal/application/services/user_service.go`

El servicio **orquesta**: recibe datos primitivos del handler, construye la entidad de dominio, aplica reglas de negocio, y delega la persistencia al repositorio.

```go
package services

import (
	"context"
	"errors"
	"time"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/repositories"

	"github.com/google/uuid"
)

// --- Application-level errors ---

var (
	ErrEmailAlreadyInUse = errors.New("email already in use")
	ErrUserNotFound      = errors.New("user not found")
)

// userService implements usecases.UserService.
// Unexported struct: can only be created via NewUserService.
type userService struct {
	userRepo repositories.UserRepository
}

// NewUserService creates a new UserService.
// It receives the outbound port (repository interface), not a concrete implementation.
func NewUserService(userRepo repositories.UserRepository) *userService {
	return &userService{userRepo: userRepo}
}

// CreateUser orchestrates user creation:
// 1. Build the domain entity (which validates name + email).
// 2. Check email uniqueness (business rule).
// 3. Persist via repository.
func (s *userService) CreateUser(ctx context.Context, name, email string) (*entities.User, error) {
	// Step 1: Domain entity validates its own invariants
	user, err := entities.NewUser(name, email)
	if err != nil {
		return nil, err
	}

	// Step 2: Business rule — email must be unique
	existing, _ := s.userRepo.GetByEmail(ctx, email)
	if existing != nil {
		return nil, ErrEmailAlreadyInUse
	}

	// Step 3: Persist
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByID retrieves a user by ID.
func (s *userService) GetUserByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// UpdateUser applies a partial update to an existing user.
// Only non-nil fields are updated.
func (s *userService) UpdateUser(ctx context.Context, id uuid.UUID, name, email *string) (*entities.User, error) {
	// Step 1: Fetch existing
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Step 2: Apply changes
	if name != nil {
		user.Name = *name
	}
	if email != nil {
		// If email changes, check uniqueness
		if *email != user.Email {
			existing, _ := s.userRepo.GetByEmail(ctx, *email)
			if existing != nil {
				return nil, ErrEmailAlreadyInUse
			}
		}
		user.Email = *email
	}

	// Step 3: Update timestamp and persist
	user.UpdatedAt = time.Now()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
```

### ¿Dónde va cada tipo de validación?

La entidad (`NewUser`) y el servicio (`CreateUser`) validan cosas distintas. La diferencia es **qué información necesitan**:

```go
// 🔵 DOMINIO — entities.NewUser()
// Solo necesita los datos que le pasan. No consulta nada externo.
func NewUser(name, email string) (*User, error) {
    if name == ""     { ... } // ✅ autocontenida: solo miro el string
    NewEmail(email)           // ✅ autocontenida: solo valido el formato
}

// 🟢 APLICACIÓN — userService.CreateUser()
// Necesita consultar la BD para verificar estado externo.
func (s *userService) CreateUser(ctx context.Context, ...) {
    entities.NewUser(name, email)            // delega validaciones internas al dominio
    s.userRepo.GetByEmail(ctx, email)        // ✅ requiere estado externo: ¿ya existe?
}
```

| Tipo de validación | ¿Dónde? | Ejemplo | Motivo |
|-------------------|---------|---------|--------|
| **Autocontenida** | 🔵 Entidad | Nombre vacío, formato de email | Solo necesita los datos que recibe |
| **Requiere estado externo** | 🟢 Servicio | Email único, límite de usuarios | Necesita consultar la BD u otros servicios |

La entidad **no puede** verificar unicidad porque no conoce el repositorio — y no debe conocerlo. Un `User` sabe qué *es* un usuario, no qué *otros usuarios existen*.

### 💡 Buenas prácticas aplicadas

| Aspecto | Patrón aplicado | Por qué |
|---------|----------------|--------|
| ¿Quién construye la entidad? | El servicio | La creación de entidades es lógica de dominio. El handler solo conoce primitivos. |
| ¿Hay chequeo de duplicados? | ✅ Email único | Es una regla de negocio real. Si no lo haces en el servicio, dependes del error críptico de la BD. |
| ¿Cómo se hace el update? | El servicio recibe el ID y solo los campos que cambian | El handler no debería orquestar lógica. Eso es trabajo del servicio. |
| Errores de aplicación | `ErrEmailAlreadyInUse`, `ErrUserNotFound` | Permiten que el handler decida el HTTP status code apropiado. |

### 💡 Consejo: `NewUserService` retorna `*userService` (concreto), no la interfaz

En Go idiomático, los constructores retornan el **tipo concreto**, no la interfaz. ¿Por qué? Porque las interfaces pertenecen al consumidor. El router/main asignará el concreto a la interfaz `usecases.UserService` — ahí es donde el compilador verifica que implementa todos los métodos.

---

## Paso 6 — Adaptador de salida (PostgreSQL)

📁 `internal/infrastructure/adapters/outbound/persistence/postgres_user_repository.go`

Este adaptador **implementa** la interfaz `UserRepository` con PostgreSQL + GORM.

```go
package persistence

import (
	"context"
	"time"

	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- Database Model (infrastructure concern) ---

// UserModel is the GORM-specific database representation.
// It is NOT a domain entity. It lives here because only this adapter cares about it.
type UserModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string    `gorm:"not null"`
	Email     string    `gorm:"uniqueIndex;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (UserModel) TableName() string {
	return "users"
}

// --- Repository implementation ---

// PostgresUserRepository implements domain's UserRepository interface.
type PostgresUserRepository struct {
	db *gorm.DB
}

func NewPostgresUserRepository(db *gorm.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *entities.User) error {
	model := toUserModel(user)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	var model UserModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, err
	}
	return toUserEntity(&model), nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	var model UserModel
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error; err != nil {
		return nil, err
	}
	return toUserEntity(&model), nil
}

func (r *PostgresUserRepository) Update(ctx context.Context, user *entities.User) error {
	model := toUserModel(user)
	return r.db.WithContext(ctx).Save(model).Error
}

// --- Mappers (package-level functions, not methods) ---

func toUserModel(user *entities.User) *UserModel {
	return &UserModel{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func toUserEntity(model *UserModel) *entities.User {
	return &entities.User{
		ID:        model.ID,
		Name:      model.Name,
		Email:     model.Email,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}
```

### ¿Por qué funciones de paquete en vez de métodos del repositorio?

Los mappers (`toUserModel`, `toUserEntity`) son funciones de paquete (no `func (r *PostgresUserRepository)`). Esto es más limpio porque:
- No necesitan el receptor `r` — no acceden a la BD.
- Son reutilizables por otros adaptadores del mismo paquete si fuera necesario.
- Es más claro: el mapper es una transformación pura, no una operación del repositorio.

### 💡 Consejo: `UserModel` separado de `entities.User`

Esto es **fundamental**. La entidad de dominio y el modelo de base de datos son cosas distintas:

```
entities.User     → qué es un usuario para el negocio (sin tags de framework)
UserModel         → cómo se guarda en PostgreSQL (con tags de GORM)
```

El mapper traduce entre los dos mundos. Si mañana cambias de GORM a sqlx, solo tocas este archivo.

### 💡 Consejo: ¿por qué `toUserEntity` no usa `NewUser()` para validar?

Fíjate que el mapper construye la entidad directamente (`&entities.User{...}`) en vez de usar el constructor `NewUser()`. ¿No es peligroso saltarse la validación?

**No**, porque esos datos **ya fueron validados por `NewUser()` cuando se guardaron.** El flujo completo es:

```
Crear:  Handler → Service → NewUser() ← VALIDA → repo.Create() → BD
Leer:   BD → repo.GetByID() → toUserEntity() → Service → Handler
                                ↑
                         Los datos ya son válidos
```

Además, `NewUser()` genera un nuevo `uuid` y nuevos timestamps — no queremos eso al leer de la BD, queremos los valores originales.

**La protección real contra datos corruptos va en la BD, no en el mapper:**

```sql
CREATE TABLE users (
    name  VARCHAR(255) NOT NULL CHECK (length(name) > 0),
    email VARCHAR(255) NOT NULL UNIQUE
);
```

Con constraints en la BD, es **físicamente imposible** que existan datos inválidos, sin importar quién escriba (la app, un script, un DBA).

**Si algún día descubres datos corruptos en la BD** (por ejemplo, un bug en producción que se saltó validaciones), la solución es un script de migración o una herramienta en `cmd/tools/`, no un mapper defensivo permanente.

---

## Paso 7 — Adaptador de entrada (HTTP handler)

📁 `internal/infrastructure/adapters/inbound/http/user_handler.go`

El handler traduce HTTP → servicio de aplicación. No conoce la base de datos ni las entidades de dominio (solo los DTOs).

```go
package http

import (
	"errors"
	"net/http"

	"ductifact/internal/application/services"
	"ductifact/internal/application/usecases"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// --- DTOs (HTTP-specific, not domain objects) ---

type CreateUserRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

type UpdateUserRequest struct {
	Name  *string `json:"name" binding:"omitempty"`
	Email *string `json:"email" binding:"omitempty,email"`
}

type UserResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// --- Handler ---

type UserHandler struct {
	userService usecases.UserService
}

func NewUserHandler(userService usecases.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Pass primitives to the service — the handler does NOT create domain entities.
	user, err := h.userService.CreateUser(c.Request.Context(), req.Name, req.Email)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrEmailAlreadyInUse) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, toUserResponse(user))
}

func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Pass optional fields as pointers — the service handles the logic.
	user, err := h.userService.UpdateUser(c.Request.Context(), id, req.Name, req.Email)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, services.ErrUserNotFound):
			status = http.StatusNotFound
		case errors.Is(err, services.ErrEmailAlreadyInUse):
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// --- Mapper: Domain → HTTP Response ---

func toUserResponse(user *entities.User) *UserResponse {
	return &UserResponse{
		ID:    user.ID.String(),
		Name:  user.Name,
		Email: user.Email,
	}
}
```

> **Nota:** Se necesita importar `"ductifact/internal/domain/entities"` para el mapper `toUserResponse`.

### 💡 Buenas prácticas aplicadas

| Aspecto | Patrón aplicado | Por qué |
|---------|----------------|--------|
| Handler no crea entidades de dominio | ✅ Pasa primitivos al servicio | El handler no debería conocer la estructura interna del dominio. |
| Manejo de errores | Usa `errors.Is()` para elegir status | 404 para no encontrado, 409 para duplicado, 500 para errores internos. |
| DTOs usan `string` para IDs | ✅ | En el JSON de respuesta, un UUID siempre es string. Mantener consistencia. |
| Update pasa punteros | ✅ Pasa `*string` opcionales | El handler solo transporta datos. La lógica de "qué cambió" es del servicio. |

### 💡 Consejo: `errors.Is()` para mapear errores a HTTP status

Este es un patrón muy limpio. Los errores de dominio/aplicación se definen como variables (`var ErrUserNotFound = ...`). El handler usa `errors.Is()` para decidir qué código HTTP devolver. Así el servicio no sabe nada de HTTP, y el handler no sabe nada de lógica de negocio.

---

## Paso 8 — Router (registrar rutas)

📁 `internal/infrastructure/adapters/inbound/http/router.go` (modificar)

Registrar el `UserService` como parámetro y registrar las rutas:

```go
func SetupRoutes(userService usecases.UserService) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy !!!!"})
	})

	// User routes
	userHandler := NewUserHandler(userService)
	userRoutes := r.Group("/users")
	{
		userRoutes.POST("", userHandler.CreateUser)
		userRoutes.GET("/:id", userHandler.GetUser)
		userRoutes.PUT("/:id", userHandler.UpdateUser)
	}

	return r
}
```

### 💡 Consejo: ¿pasarle muchos servicios al router?

Cuando tengas 5-10 servicios, pasar todos como parámetros se vuelve incómodo. En ese punto, considera crear un struct `Dependencies` o usar un patrón de *options*. Pero para 2-3 servicios, parámetros directos es lo más simple.

---

## Paso 9 — Wiring en main.go

📁 `cmd/api/main.go` (modificar)

El `main.go` es el **composition root**: aquí se conectan todas las piezas. Es el único lugar que conoce *todas* las implementaciones concretas.

```go
func main() {
	// ...

	// Database
	db, err := persistence.NewPostgresConnection()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// --- User wiring ---
	userRepo := persistence.NewPostgresUserRepository(db)
	userService := services.NewUserService(userRepo)

	// --- HTTP ---
	router := httpAdapter.SetupRoutes(userService)

	// ...
}
```

### 💡 Consejo: el flujo de dependencias

Fíjate en el orden: **siempre de fuera hacia dentro** en el wiring.

```
DB connection → Repository (outbound adapter) → Service (application) → Router (inbound adapter)
```

Cada pieza recibe sus dependencias por **inyección de constructor** (parámetro en `New...()`). No hay variables globales, no hay singletons, no hay magia.

---

## Paso 10 — Base de datos (SQL)

📁 `init.sql` (modificar)

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    email      VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
```

### 💡 Consejo: `TIMESTAMPTZ` vs `TIMESTAMP`

Siempre usar `TIMESTAMPTZ` (con timezone). `TIMESTAMP` sin timezone puede causar bugs cuando la app corre en una timezone diferente a la BD. Con `TIMESTAMPTZ`, PostgreSQL normaliza todo a UTC internamente.

---

## Resumen de archivos

| #  | Capa | Archivo | Acción |
|----|------|---------|--------|
| 1  | 🔵 Dominio | `internal/domain/entities/user.go` | **Modificar** (añadir validación, quitar json tags) |
| 2  | 🔵 Dominio | `internal/domain/valueobjects/email.go` | Ya existe ✅ (mejorar error opcionalmente) |
| 3  | 🔵 Dominio | `internal/domain/repositories/user_repository.go` | **Crear** |
| 4  | 🟢 Aplicación | `internal/application/usecases/user_service.go` | **Crear** |
| 5  | 🟢 Aplicación | `internal/application/services/user_service.go` | **Crear** |
| 6  | 🟠 Infraestructura | `internal/infrastructure/adapters/outbound/persistence/postgres_user_repository.go` | **Crear** |
| 7  | 🟠 Infraestructura | `internal/infrastructure/adapters/inbound/http/user_handler.go` | **Crear** |
| 8  | 🟠 Infraestructura | `internal/infrastructure/adapters/inbound/http/router.go` | **Modificar** |
| 9  | ⚙️ Wiring | `cmd/api/main.go` | **Modificar** |
| 10 | 🗄️ SQL | `init.sql` | **Modificar** |

---

## Errores comunes a evitar

### ❌ Poner tags `json:` o `gorm:` en las entidades de dominio

```go
// MAL — la entidad conoce detalles de infraestructura
type User struct {
	ID    uuid.UUID `json:"id" gorm:"primaryKey"`
	Email string    `json:"email" gorm:"uniqueIndex"`
}

// BIEN — la entidad es Go puro
type User struct {
	ID    uuid.UUID
	Email string
}
```

### ❌ Constructor que no devuelve error

```go
// MAL — ¿y si el email es inválido? No hay forma de saberlo.
func NewUser(name, email string) *User { ... }

// BIEN — Go idiomático: si puede fallar, devuelve error.
func NewUser(name, email string) (*User, error) { ... }
```

### ❌ Handler creando entidades de dominio

```go
// MAL — el handler conoce la estructura interna del dominio
user := entities.NewUser(req.Name, req.Email)
created, err := h.userService.CreateUser(ctx, user)

// BIEN — el handler solo pasa datos primitivos
created, err := h.userService.CreateUser(ctx, req.Name, req.Email)
```

### ❌ Devolver siempre HTTP 400

```go
// MAL — todo es "bad request"
c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

// BIEN — errores específicos
switch {
case errors.Is(err, services.ErrUserNotFound):
	c.JSON(http.StatusNotFound, ...)
case errors.Is(err, services.ErrEmailAlreadyInUse):
	c.JSON(http.StatusConflict, ...)
default:
	c.JSON(http.StatusInternalServerError, ...)
}
```

### ❌ Interfaces demasiado grandes "por si acaso"

```go
// MAL — definir métodos que no necesitas todavía
type UserRepository interface {
	Create(...) error
	GetByID(...) (...)
	GetByEmail(...) (...)
	Update(...) error
	Delete(...) error        // ← ¿lo necesitas ahora? No.
	List(...) (...)          // ← ¿lo necesitas ahora? No.
	Search(...) (...)        // ← ¿lo necesitas ahora? No.
	CountByStatus(...) (int) // ← ¿lo necesitas ahora? No.
}

// BIEN — solo lo que necesitas hoy
type UserRepository interface {
	Create(...) error
	GetByID(...) (...)
	GetByEmail(...) (...)
	Update(...) error
}
```

---

> **Siguiente paso:** Implementar en el orden de los pasos (1→10). Cada paso compila independientemente.
> Se recomienda usar el workflow de tareas (`ai-devkit/tasks/`) para la implementación.
