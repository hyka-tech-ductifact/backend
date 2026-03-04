# Guía de Testing — Backend Go

## Índice
1. [Visión general](#1-visión-general)
2. [Pirámide de testing](#2-pirámide-de-testing)
3. [Estructura de carpetas](#3-estructura-de-carpetas)
4. [Convenciones en Go](#4-convenciones-en-go)
5. [Unit Tests](#5-unit-tests)
   - 5.1 [Qué son y qué testean](#51-qué-son-y-qué-testean)
   - 5.2 [Value Object: Email](#52-value-object-email)
   - 5.3 [Entity: User](#53-entity-user)
   - 5.4 [Service: UserService (con mocks)](#54-service-userservice-con-mocks)
6. [Integration Tests](#6-integration-tests)
   - 6.1 [Qué son y qué testean](#61-qué-son-y-qué-testean)
   - 6.2 [Setup de DB para tests](#62-setup-de-db-para-tests)
   - 6.3 [Repository: PostgresUserRepository](#63-repository-postgresuserrepository)
7. [E2E Tests](#7-e2e-tests)
   - 7.1 [Qué son y qué testean](#71-qué-son-y-qué-testean)
   - 7.2 [Setup del servidor](#72-setup-del-servidor)
   - 7.3 [Test de flujo completo](#73-test-de-flujo-completo)
8. [Tabla resumen](#8-tabla-resumen)
9. [Comandos para ejecutar tests](#9-comandos-para-ejecutar-tests)
10. [Buenas prácticas](#10-buenas-prácticas)

---

## 1. Visión general

Tenemos 3 niveles de testing, cada uno con un propósito diferente:

- **Unit**: Testea lógica aislada sin dependencias externas.
- **Integration**: Testea la interacción entre componentes reales (ej: repo + DB).
- **E2E**: Testea el flujo completo HTTP → Handler → Service → Repo → DB.

Cada nivel requiere distinto setup, velocidad y granularidad.

---

## 2. Pirámide de testing

```
         /  E2E  \           ← Pocos tests (flujos críticos)
        /----------\
       / Integration \       ← Moderados (repos, adapters)
      /----------------\
     /      Unit        \    ← Muchos tests (dominio, servicios)
    /--------------------\
```

- **Base (Unit)**: Rápidos, muchos, sin dependencias. Son el cimiento.
- **Medio (Integration)**: Necesitan DB real, más lentos, pero validan que la infra funciona.
- **Cima (E2E)**: Necesitan todo levantado, los más lentos, solo happy paths y flujos críticos.

### ¿Por qué más unitarios y menos E2E?

La intuición dice: *"si el E2E prueba lo que el usuario real usa, ¿por qué no hacer más de esos?"*
Hay 4 razones prácticas por las que la pirámide pone los unitarios en la base:

**1. Velocidad**
Los unitarios se ejecutan en milisegundos sin dependencias. Los E2E necesitan DB + app corriendo y cada llamada HTTP tiene overhead de red. Con pocas entidades la diferencia es despreciable, pero a medida que crece el proyecto (500+ tests), un CI con solo E2E puede tardar 10x más.

**2. Diagnóstico preciso**
Si un E2E falla (`POST /users` devuelve 500), el error puede estar en el handler, el service, el repo, la DB o la conexión — hay que investigar toda la cadena. Si un unitario falla (`TestCreateUser_WithDuplicateEmail`), sabes que el bug está exactamente en `user_service.go`. Los unitarios reducen el tiempo de debugging.

**3. Fragilidad**
Los E2E dependen de muchas cosas: Docker corriendo, PostgreSQL arriba, app compilada y arrancada, puerto libre, `.env` correcto. Si cualquiera falla, **todos** los E2E fallan. Los unitarios no dependen de nada externo.

**4. Cobertura combinatoria**
Validar 16 variaciones de email (7 válidos + 9 inválidos) con E2E requiere 16 llamadas HTTP reales, cada una limpiando la DB. Con unitarios se cubren las mismas combinaciones en milisegundos.

### Cada capa responde una pregunta diferente

| Capa | Pregunta que responde | Ejemplo en este proyecto |
|------|----------------------|--------------------------|
| **Unit** | ¿La lógica es correcta? | `NewEmail("")` devuelve `ErrInvalidEmail` |
| **Integration** | ¿Los componentes se conectan bien? | El mapper `toUserModel` preserva todos los campos |
| **E2E** | ¿El usuario puede usarlo? | `POST /users` → `GET /users/:id` devuelve lo mismo |

### ¿Los E2E hacen redundantes a los unitarios?

**No.** Un test unitario verde **no garantiza** que el sistema funcione (el mapper podría estar roto). Pero un E2E verde **tampoco reemplaza** a los unitarios: sin ellos, cuando algo falla estás buscando la aguja en el pajar.

La clave: los unitarios dan feedback rápido y preciso al desarrollar. Los E2E dan confianza de que las piezas **encajan**. Si los unitarios pasan pero un E2E no, sabes que el bug está en la integración entre componentes — reduces la búsqueda enormemente.

> **Regla práctica**: unitarios para toda la lógica y combinaciones (muchos, baratos). E2E para los flujos principales del usuario (pocos, costosos, pero imprescindibles).

---

## 3. Estructura de carpetas

```
test/
├── unit/                          ← Tests unitarios
│   ├── domain/
│   │   ├── entities/
│   │   │   └── user_test.go       ← Tests de la entidad User
│   │   └── valueobjects/
│   │       └── email_test.go      ← Tests del VO Email
│   ├── application/
│   │   └── services/
│   │       └── user_service_test.go  ← Tests del servicio con mocks
│   └── mocks/
│       └── mock_user_repository.go   ← Mock manual del repositorio
├── integration/
│   └── persistence/
│       └── postgres_user_repository_test.go  ← Tests del repo con DB real
├── e2e/
│   ├── setup.go                   ← Health-check + CleanDB
│   └── user_test.go               ← Tests HTTP puros contra localhost
└── helpers/
    └── setup.go                   ← SetupTestDB, CleanDB, LoadEnv
```

> **Nota**: Usamos una carpeta `test/` separada (en lugar de `_test.go` junto al código de producción) para mantener los tests organizados por tipo y evitar que los tests de integración/e2e se mezclen con el código de dominio.

---

## 4. Convenciones en Go

### 4.1 Nomenclatura de archivos

Los archivos de test **siempre** terminan en `_test.go`:
```
user.go          ← código de producción
user_test.go     ← tests
```

### 4.2 Nomenclatura de funciones

Toda función de test empieza con `Test` y recibe `*testing.T`:
```go
func TestNewUser_WithValidData_ReturnsUser(t *testing.T) {
    // ...
}
```

Patrón recomendado: `Test<Función>_<Escenario>_<ResultadoEsperado>`

### 4.3 Table-driven tests

Go tiene un patrón idiomático para probar múltiples casos: **table-driven tests**.
En vez de escribir una función por caso, defines una tabla de casos y los iteras:

```go
func TestNewEmail(t *testing.T) {
    tests := []struct {
        name    string   // nombre descriptivo del caso
        email   string   // input
        wantErr bool     // ¿esperamos error?
    }{
        {"valid email", "user@example.com", false},
        {"empty email", "", true},
        {"no at sign", "userexample.com", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := valueobjects.NewEmail(tt.email)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

**¿Por qué `t.Run()`?**
- Cada caso aparece como un sub-test con su propio nombre en la salida.
- Puedes ejecutar un caso individual: `go test -run "TestNewEmail/empty_email"`.
- Si un caso falla, los demás siguen ejecutándose.

### 4.4 Testify: assert vs require

Usamos `github.com/stretchr/testify` que ya está en tu `go.mod`:

```go
import (
    "github.com/stretchr/testify/assert"   // falla pero sigue ejecutando
    "github.com/stretchr/testify/require"   // falla y detiene el test
)

// assert: usa cuando quieres ver TODOS los fallos de un test
assert.Equal(t, "Juan", user.Name)
assert.NoError(t, err)

// require: usa cuando no tiene sentido seguir si falla
require.NoError(t, err)  // si hay error, las siguientes líneas no se ejecutan
user := result           // esto solo se ejecuta si require pasó
```

**Regla**: Usa `require` para precondiciones (errores que invalidan el resto del test), `assert` para verificaciones finales.

---

## 5. Unit Tests

### 5.1 Qué son y qué testean

- **No necesitan** base de datos, servidor HTTP, ni ninguna dependencia externa.
- **Se ejecutan en milisegundos**.
- Testean la lógica pura: validaciones, reglas de negocio, transformaciones.
- Cuando el código depende de una interfaz (como `UserRepository`), usamos **mocks**.

**¿Qué testear con unit tests en tu proyecto?**
| Componente | Qué validar |
|-----------|-------------|
| `Email` (Value Object) | Formatos válidos e inválidos |
| `User` (Entity) | Creación con datos válidos, nombre vacío, email inválido |
| `UserService` (Application) | Lógica de orquestación: email duplicado, user no encontrado, updates parciales |

---

### 5.2 Value Object: Email

**Archivo**: `test/unit/domain/valueobjects/email_test.go`

```go
package valueobjects_test

import (
	"testing"

	"ductifact/internal/domain/valueobjects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmail_WithValidEmails_ReturnsEmail(t *testing.T) {
	validEmails := []struct {
		name  string
		email string
	}{
		{"simple email", "user@example.com"},
		{"with dots", "first.last@example.com"},
		{"with plus", "user+tag@example.com"},
		{"with subdomain", "user@mail.example.com"},
		{"with numbers", "user123@example.com"},
	}

	for _, tt := range validEmails {
		t.Run(tt.name, func(t *testing.T) {
			email, err := valueobjects.NewEmail(tt.email)

			require.NoError(t, err)
			assert.Equal(t, tt.email, email.String())
		})
	}
}

func TestNewEmail_WithInvalidEmails_ReturnsError(t *testing.T) {
	invalidEmails := []struct {
		name  string
		email string
	}{
		{"empty string", ""},
		{"no at sign", "userexample.com"},
		{"no domain", "user@"},
		{"no local part", "@example.com"},
		{"spaces", "user @example.com"},
		{"double at", "user@@example.com"},
		{"no TLD", "user@example"},
	}

	for _, tt := range invalidEmails {
		t.Run(tt.name, func(t *testing.T) {
			email, err := valueobjects.NewEmail(tt.email)

			assert.Nil(t, email)
			assert.ErrorIs(t, err, valueobjects.ErrInvalidEmail)
		})
	}
}
```

**¿Qué aprender de este test?**
1. El paquete es `valueobjects_test` (con `_test`), no `valueobjects`. Esto fuerza a testear solo la API pública.
2. Usamos `table-driven tests` para probar muchos inputs sin repetir código.
3. Verificamos **tanto el happy path como los errores**.
4. Usamos `assert.ErrorIs` para verificar el tipo exacto de error, no solo "que haya un error".

---

### 5.3 Entity: User

**Archivo**: `test/unit/domain/entities/user_test.go`

```go
package entities_test

import (
	"testing"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/valueobjects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser_WithValidData_ReturnsUser(t *testing.T) {
	user, err := entities.NewUser("Juan", "juan@example.com")

	require.NoError(t, err)
	assert.Equal(t, "Juan", user.Name)
	assert.Equal(t, "juan@example.com", user.Email)
	assert.NotEmpty(t, user.ID)           // UUID generado automáticamente
	assert.False(t, user.CreatedAt.IsZero()) // Timestamp generado
	assert.False(t, user.UpdatedAt.IsZero())
}

func TestNewUser_WithEmptyName_ReturnsError(t *testing.T) {
	user, err := entities.NewUser("", "juan@example.com")

	assert.Nil(t, user)
	assert.ErrorIs(t, err, entities.ErrEmptyUserName)
}

func TestNewUser_WithInvalidEmail_ReturnsError(t *testing.T) {
	user, err := entities.NewUser("Juan", "invalid-email")

	assert.Nil(t, user)
	assert.ErrorIs(t, err, valueobjects.ErrInvalidEmail)
}

func TestNewUser_GeneratesUniqueIDs(t *testing.T) {
	user1, err1 := entities.NewUser("User1", "user1@example.com")
	user2, err2 := entities.NewUser("User2", "user2@example.com")

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, user1.ID, user2.ID, "cada user debe tener un ID único")
}
```

**¿Qué aprender de este test?**
1. Testeamos el constructor `NewUser()` — es la **única forma** de crear un User válido.
2. Verificamos que las **validaciones del dominio** funcionan (nombre vacío, email inválido).
3. Verificamos que los campos autogenerados (ID, timestamps) se crean correctamente.
4. No necesitamos mock de nada — la entidad es pura lógica de dominio.

---

### 5.4 Service: UserService (con mocks)

El `UserService` depende de `UserRepository` (una interfaz). Para hacer unit tests, **creamos un mock** de esa interfaz.

#### Paso 1: Crear el mock

**Archivo**: `test/unit/mocks/mock_user_repository.go`

```go
package mocks

import (
	"context"

	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
)

// MockUserRepository implements repositories.UserRepository for testing.
// Cada método es un campo de tipo función que puedes configurar en cada test.
type MockUserRepository struct {
	CreateFn     func(ctx context.Context, user *entities.User) error
	GetByIDFn    func(ctx context.Context, id uuid.UUID) (*entities.User, error)
	GetByEmailFn func(ctx context.Context, email string) (*entities.User, error)
	UpdateFn     func(ctx context.Context, user *entities.User) error
}

func (m *MockUserRepository) Create(ctx context.Context, user *entities.User) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, user)
	}
	return nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	if m.GetByEmailFn != nil {
		return m.GetByEmailFn(ctx, email)
	}
	return nil, nil
}

func (m *MockUserRepository) Update(ctx context.Context, user *entities.User) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, user)
	}
	return nil
}
```

**¿Por qué un mock manual y no una librería?**
- En Go, los mocks manuales son idiomáticos y simples.
- No necesitas aprender una librería extra (como `mockgen` o `testify/mock`).
- Tienes control total: cada test configura solo las funciones que necesita.
- Si más adelante quieres, puedes migrar a `mockery` o `gomock`, pero para empezar esto es perfecto.

#### Paso 2: Tests del servicio

**Archivo**: `test/unit/application/services/user_service_test.go`

```go
package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ductifact/internal/application/services"
	"ductifact/internal/domain/entities"
	"ductifact/test/unit/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- CreateUser ---

func TestCreateUser_WithValidData_ReturnsUser(t *testing.T) {
	// ARRANGE: configuramos el mock
	mockRepo := &mocks.MockUserRepository{
		// GetByEmail devuelve nil (no existe el email) — comportamiento por defecto
		// Create no hace nada — comportamiento por defecto
	}

	svc := services.NewUserService(mockRepo)

	// ACT
	user, err := svc.CreateUser(context.Background(), "Juan", "juan@example.com")

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, "Juan", user.Name)
	assert.Equal(t, "juan@example.com", user.Email)
	assert.NotEmpty(t, user.ID)
}

func TestCreateUser_WithDuplicateEmail_ReturnsError(t *testing.T) {
	// ARRANGE: GetByEmail devuelve un user existente
	existingUser := &entities.User{
		ID:    uuid.New(),
		Name:  "Existing",
		Email: "juan@example.com",
	}

	mockRepo := &mocks.MockUserRepository{
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			return existingUser, nil
		},
	}

	svc := services.NewUserService(mockRepo)

	// ACT
	user, err := svc.CreateUser(context.Background(), "Juan", "juan@example.com")

	// ASSERT
	assert.Nil(t, user)
	assert.ErrorIs(t, err, services.ErrEmailAlreadyInUse)
}

func TestCreateUser_WithEmptyName_ReturnsError(t *testing.T) {
	mockRepo := &mocks.MockUserRepository{}
	svc := services.NewUserService(mockRepo)

	user, err := svc.CreateUser(context.Background(), "", "juan@example.com")

	assert.Nil(t, user)
	assert.Error(t, err) // El error viene del dominio (entities.ErrEmptyUserName)
}

func TestCreateUser_WhenRepoFails_ReturnsError(t *testing.T) {
	// ARRANGE: Create falla
	mockRepo := &mocks.MockUserRepository{
		CreateFn: func(ctx context.Context, user *entities.User) error {
			return errors.New("db connection lost")
		},
	}

	svc := services.NewUserService(mockRepo)

	// ACT
	user, err := svc.CreateUser(context.Background(), "Juan", "juan@example.com")

	// ASSERT
	assert.Nil(t, user)
	assert.EqualError(t, err, "db connection lost")
}

// --- GetUserByID ---

func TestGetUserByID_WithExistingUser_ReturnsUser(t *testing.T) {
	expectedID := uuid.New()
	expectedUser := &entities.User{
		ID:    expectedID,
		Name:  "Juan",
		Email: "juan@example.com",
	}

	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			if id == expectedID {
				return expectedUser, nil
			}
			return nil, errors.New("not found")
		},
	}

	svc := services.NewUserService(mockRepo)

	user, err := svc.GetUserByID(context.Background(), expectedID)

	require.NoError(t, err)
	assert.Equal(t, expectedUser.Name, user.Name)
}

func TestGetUserByID_WithNonExistingUser_ReturnsError(t *testing.T) {
	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			return nil, errors.New("not found")
		},
	}

	svc := services.NewUserService(mockRepo)

	user, err := svc.GetUserByID(context.Background(), uuid.New())

	assert.Nil(t, user)
	assert.ErrorIs(t, err, services.ErrUserNotFound)
}

// --- UpdateUser ---

func TestUpdateUser_WithNewName_UpdatesOnlyName(t *testing.T) {
	existingUser := &entities.User{
		ID:        uuid.New(),
		Name:      "Juan",
		Email:     "juan@example.com",
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now().Add(-time.Hour),
	}

	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			// Devolvemos una copia para que el test sea más realista
			copy := *existingUser
			return &copy, nil
		},
		UpdateFn: func(ctx context.Context, user *entities.User) error {
			return nil
		},
	}

	svc := services.NewUserService(mockRepo)
	newName := "Pedro"

	user, err := svc.UpdateUser(context.Background(), existingUser.ID, &newName, nil)

	require.NoError(t, err)
	assert.Equal(t, "Pedro", user.Name)
	assert.Equal(t, "juan@example.com", user.Email) // email sin cambiar
}

func TestUpdateUser_WithDuplicateEmail_ReturnsError(t *testing.T) {
	userID := uuid.New()
	existingUser := &entities.User{
		ID:    userID,
		Name:  "Juan",
		Email: "juan@example.com",
	}

	mockRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			copy := *existingUser
			return &copy, nil
		},
		GetByEmailFn: func(ctx context.Context, email string) (*entities.User, error) {
			// Otro user ya tiene ese email
			return &entities.User{ID: uuid.New(), Email: email}, nil
		},
	}

	svc := services.NewUserService(mockRepo)
	newEmail := "taken@example.com"

	user, err := svc.UpdateUser(context.Background(), userID, nil, &newEmail)

	assert.Nil(t, user)
	assert.ErrorIs(t, err, services.ErrEmailAlreadyInUse)
}
```

**¿Qué aprender de este test?**
1. **Patrón AAA**: Arrange (setup mock) → Act (llamar al servicio) → Assert (verificar resultado).
2. Cada test configura **solo lo que necesita** del mock. Si no configuras `CreateFn`, por defecto no hace nada (retorna nil).
3. Testeamos **todos los caminos**: happy path, errores de dominio, errores de infra, reglas de negocio.
4. El servicio se testea **sin DB real** — solo con mocks. Es instantáneo.

---

## 6. Integration Tests

### 6.1 Qué son y qué testean

- Testean que tu código **interactúa correctamente con una dependencia real** (en nuestro caso, PostgreSQL).
- **Necesitan una DB real** corriendo (Docker o local).
- Son más lentos que los unit tests, pero verifican que tu SQL/GORM funciona.

**¿Qué testear con integration tests?**
| Componente | Qué validar |
|-----------|-------------|
| `PostgresUserRepository` | Create, GetByID, GetByEmail, Update funcionan contra Postgres real |
| Mappers (`toUserModel` / `toUserEntity`) | Los datos se persisten y recuperan correctamente |
| Constraints de la DB | El `UNIQUE` del email se respeta |

### 6.2 Setup de DB para tests

Ya tienes `test/helpers/test_utils.go` con `SetupTestDB()`. Para los integration tests necesitas:

1. **Una DB Postgres corriendo** (tu `docker-compose.dev.yml` ya la tiene).
2. **Limpiar datos entre tests** para que no se contaminen entre sí.

Para limpiar datos, añade una función helper:

**Añadir a** `test/helpers/test_utils.go`:

```go
// CleanDB truncates all tables to ensure test isolation.
// Call this at the beginning of each integration test.
func CleanDB(t *testing.T, db *gorm.DB) {
	err := db.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE").Error
	require.NoError(t, err)
}
```

### 6.3 Repository: PostgresUserRepository

**Archivo**: `test/integration/persistence/postgres_user_repository_test.go`

```go
package persistence_test

import (
	"context"
	"testing"

	"ductifact/internal/domain/entities"
	"ductifact/internal/infrastructure/adapters/outbound/persistence"
	"ductifact/test/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupRepo crea el repositorio con una DB limpia para cada test.
func setupRepo(t *testing.T) *persistence.PostgresUserRepository {
	db := helpers.SetupTestDB(t)
	helpers.CleanDB(t, db)
	return persistence.NewPostgresUserRepository(db)
}

func TestPostgresUserRepository_Create_And_GetByID(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	// Creamos un user válido usando el constructor de dominio
	user, err := entities.NewUser("Juan", "juan@example.com")
	require.NoError(t, err)

	// CREATE
	err = repo.Create(ctx, user)
	require.NoError(t, err)

	// GET BY ID
	found, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)

	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, "Juan", found.Name)
	assert.Equal(t, "juan@example.com", found.Email)
	assert.False(t, found.CreatedAt.IsZero())
}

func TestPostgresUserRepository_GetByEmail(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	user, _ := entities.NewUser("Juan", "juan@example.com")
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// GET BY EMAIL
	found, err := repo.GetByEmail(ctx, "juan@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)

	// EMAIL NO EXISTENTE
	notFound, err := repo.GetByEmail(ctx, "noexiste@example.com")
	assert.Error(t, err)
	assert.Nil(t, notFound)
}

func TestPostgresUserRepository_Update(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	user, _ := entities.NewUser("Juan", "juan@example.com")
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Modificamos el user
	user.Name = "Pedro"
	user.Email = "pedro@example.com"

	err = repo.Update(ctx, user)
	require.NoError(t, err)

	// Verificamos que se actualizó
	found, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "Pedro", found.Name)
	assert.Equal(t, "pedro@example.com", found.Email)
}

func TestPostgresUserRepository_GetByID_NotFound(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	found, err := repo.GetByID(ctx, uuid.New())

	assert.Error(t, err)
	assert.Nil(t, found)
}

func TestPostgresUserRepository_Create_DuplicateEmail_Fails(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	user1, _ := entities.NewUser("Juan", "mismo@example.com")
	err := repo.Create(ctx, user1)
	require.NoError(t, err)

	user2, _ := entities.NewUser("Pedro", "mismo@example.com")
	err = repo.Create(ctx, user2)

	// La DB debe rechazar el email duplicado por el UNIQUE constraint
	assert.Error(t, err)
}
```

**¿Qué aprender de este test?**
1. **Necesita DB real** — no hay mocks aquí. Estamos testeando que GORM genera el SQL correcto.
2. `setupRepo()` limpia la DB antes de cada test — **aislamiento entre tests**.
3. Testeamos que las **constraints de la DB** funcionan (email UNIQUE).
4. Usamos `entities.NewUser()` para crear datos de test — reutilizamos la validación del dominio.
5. Son más lentos, pero **verifican cosas que los unit tests no pueden**: SQL real, tipos de PostgreSQL, índices.

---

## 7. E2E Tests

### 7.1 Qué son y qué testean

- Testean el **sistema completo**: levantan un servidor HTTP real y hacen requests reales.
- Verifican que **todas las capas están correctamente conectadas**.
- Son los más lentos y los que más infra necesitan (DB + API corriendo).

**¿Qué testear con E2E?**
Solo los **flujos críticos** del negocio:
- Crear un user y verificar que se puede recuperar.
- Intentar crear un user con email duplicado y recibir 409.
- Obtener un user inexistente y recibir 404.
- Actualizar un user y verificar los cambios.

### 7.2 Setup del servidor

Ya tienes `test/e2e/setup.go` con el setup base. Los E2E tests asumen que la API está corriendo (ya sea en Docker o localmente).

### 7.3 Test de flujo completo

**Archivo**: `test/e2e/user_e2e_test.go`

```go
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- DTOs para los tests (independientes de los del handler) ---

type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type userResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// --- Helper para hacer requests ---

func postJSON(t *testing.T, url string, body interface{}) *http.Response {
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	return resp
}

func getJSON(t *testing.T, url string) *http.Response {
	resp, err := http.Get(url)
	require.NoError(t, err)
	return resp
}

func putJSON(t *testing.T, url string, body interface{}) *http.Response {
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func decodeJSON[T any](t *testing.T, resp *http.Response) T {
	var result T
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	defer resp.Body.Close()
	return result
}

// --- Tests ---

func TestE2E_CreateAndGetUser(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Cleanup()

	// 1. Crear user
	createResp := postJSON(t, env.APIBaseURL+"/users", createUserRequest{
		Name:  "Juan E2E",
		Email: "juan.e2e@example.com",
	})
	assert.Equal(t, http.StatusCreated, createResp.StatusCode)

	createdUser := decodeJSON[userResponse](t, createResp)
	assert.Equal(t, "Juan E2E", createdUser.Name)
	assert.Equal(t, "juan.e2e@example.com", createdUser.Email)
	assert.NotEmpty(t, createdUser.ID)

	// 2. Obtener el user creado
	getResp := getJSON(t, fmt.Sprintf("%s/users/%s", env.APIBaseURL, createdUser.ID))
	assert.Equal(t, http.StatusOK, getResp.StatusCode)

	fetchedUser := decodeJSON[userResponse](t, getResp)
	assert.Equal(t, createdUser.ID, fetchedUser.ID)
	assert.Equal(t, "Juan E2E", fetchedUser.Name)
}

func TestE2E_CreateUser_DuplicateEmail_Returns409(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Cleanup()

	body := createUserRequest{
		Name:  "User1",
		Email: "duplicate.e2e@example.com",
	}

	// Primera creación — OK
	resp1 := postJSON(t, env.APIBaseURL+"/users", body)
	assert.Equal(t, http.StatusCreated, resp1.StatusCode)
	resp1.Body.Close()

	// Segunda creación con mismo email — Conflict
	body.Name = "User2"
	resp2 := postJSON(t, env.APIBaseURL+"/users", body)
	assert.Equal(t, http.StatusConflict, resp2.StatusCode)

	errResp := decodeJSON[errorResponse](t, resp2)
	assert.Contains(t, errResp.Error, "email already in use")
}

func TestE2E_GetUser_NotFound_Returns404(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Cleanup()

	resp := getJSON(t, env.APIBaseURL+"/users/550e8400-e29b-41d4-a716-446655440000")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

func TestE2E_CreateUser_InvalidBody_Returns400(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Cleanup()

	// Sin name (requerido por el binding)
	resp := postJSON(t, env.APIBaseURL+"/users", map[string]string{
		"email": "test@example.com",
	})
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}

func TestE2E_UpdateUser(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Cleanup()

	// 1. Crear user
	createResp := postJSON(t, env.APIBaseURL+"/users", createUserRequest{
		Name:  "Original",
		Email: "update.e2e@example.com",
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	created := decodeJSON[userResponse](t, createResp)

	// 2. Actualizar nombre
	updateResp := putJSON(t, fmt.Sprintf("%s/users/%s", env.APIBaseURL, created.ID), map[string]string{
		"name": "Updated",
	})
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)

	updated := decodeJSON[userResponse](t, updateResp)
	assert.Equal(t, "Updated", updated.Name)
	assert.Equal(t, "update.e2e@example.com", updated.Email) // email sin cambiar
}
```

**¿Qué aprender de este test?**
1. **No importa nada del código interno** — solo usa `net/http` y JSON. Es una caja negra.
2. Los helpers (`postJSON`, `decodeJSON`) evitan repetir código de HTTP.
3. Usamos `decodeJSON[T]` con **generics de Go** para deserializar sin boilerplate.
4. Cada test tiene su propio email único para evitar conflictos si se ejecutan en paralelo.
5. Testeamos **status codes + body** — es lo que le importa al consumidor de tu API.

---

## 8. Tabla resumen

| Aspecto | Unit | Integration | E2E |
|---------|------|-------------|-----|
| **Velocidad** | ⚡ ms | 🐢 1-5s | 🐌 5-30s |
| **Dependencias** | Ninguna | DB real | DB + API corriendo |
| **Qué testea** | Lógica pura | Repo + DB | HTTP → DB → HTTP |
| **Mocks** | Sí | No | No |
| **Cantidad** | Muchos | Moderados | Pocos |
| **Dónde falla** | Lógica de negocio | SQL / Mappers | Wiring / HTTP |
| **Carpeta** | `test/unit/` | `test/integration/` | `test/e2e/` |

---

## 9. Comandos para ejecutar tests

```bash
# ─── Unit tests (no necesitan nada externo) ───
go test ./test/unit/...

# Ejecutar un test específico
go test ./test/unit/domain/entities/ -run TestNewUser_WithValidData

# Con verbose (ver detalle de cada sub-test)
go test -v ./test/unit/...

# ─── Integration tests (necesitan DB corriendo) ───
# Primero levantar la DB:
docker compose -f docker-compose.dev.yml up -d db

# Luego ejecutar:
go test ./test/integration/...

# ─── E2E tests (necesitan DB + API corriendo) ───
# Opción 1: Local
docker compose -f docker-compose.dev.yml up -d
go test ./test/e2e/...

# Opción 2: Todo en Docker
docker compose -f docker-compose.dev.yml run --rm e2e-tests

# ─── Todos los tests ───
go test ./...

# ─── Con cobertura ───
go test -cover ./test/unit/...
go test -coverprofile=coverage.out ./test/unit/...
go tool cover -html=coverage.out    # Ver en navegador
```

---

## 10. Buenas prácticas

### 10.1 Naming

```go
// ✅ Descriptivo: Test<Función>_<Escenario>_<Resultado>
func TestCreateUser_WithDuplicateEmail_ReturnsError(t *testing.T)

// ❌ Genérico
func TestCreateUser(t *testing.T)
func TestCreateUser2(t *testing.T)
```

### 10.2 Un assert por concepto

```go
// ✅ Cada assert verifica un concepto diferente
assert.Equal(t, "Juan", user.Name)       // nombre correcto
assert.NotEmpty(t, user.ID)              // ID generado
assert.False(t, user.CreatedAt.IsZero()) // timestamp generado

// ❌ No metas lógica compleja en los tests
assert.True(t, user.Name == "Juan" && user.Email == "juan@example.com")
```

### 10.3 Tests independientes

Cada test debe poder ejecutarse **solo y en cualquier orden**:

```go
// ✅ Cada test crea sus propios datos
func TestGetUser(t *testing.T) {
    user, _ := entities.NewUser("Test", "test@example.com")
    // ... usa user
}

// ❌ No dependas de datos creados por otro test
var globalUser *entities.User // ← nunca hagas esto
```

### 10.4 No testees la implementación, testea el comportamiento

```go
// ✅ Testea QUÉ hace
user, err := entities.NewUser("", "test@example.com")
assert.ErrorIs(t, err, entities.ErrEmptyUserName)

// ❌ No testees CÓMO lo hace (detalles internos)
assert.Equal(t, 3, len(user.validationRules)) // ← frágil
```

### 10.5 Build tags para separar tests lentos (opcional)

Si quieres que `go test ./...` solo ejecute unit tests por defecto:

```go
//go:build integration
// +build integration

package persistence_test
// ...
```

Y ejecutas con:
```bash
go test ./...                                    # solo unit
go test -tags=integration ./test/integration/... # integration
go test -tags=e2e ./test/e2e/...                 # e2e
```

Esto es opcional, pero útil cuando el proyecto crece.
