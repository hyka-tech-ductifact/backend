# Guía: Entidad Client — Explicación de Decisiones

> Esta guía explica las decisiones de diseño tomadas al implementar la entidad `Client` y su relación con `User`.

---

## 1. Relación User ↔ Client (One-to-Many)

### ¿Qué tipo de relación es?

Un **User** puede tener muchos **Clients**, pero cada **Client** pertenece a exactamente un **User**. Esto es una relación **1:N** (uno a muchos).

```
User (1) ─────── (*) Client
 │                    │
 │ id (PK)            │ id (PK)
 │ name               │ name
 │ email              │ user_id (FK → users.id)
 │ created_at         │ created_at
 │ updated_at         │ updated_at
```

### ¿Cómo se implementa en la base de datos?

En `init.sql`, la tabla `clients` tiene una columna `user_id` que es una **foreign key** apuntando a `users.id`:

```sql
CREATE TABLE IF NOT EXISTS clients (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
```

**Puntos clave:**

- `NOT NULL`: Un client siempre debe tener un dueño. No pueden existir clientes "huérfanos".
- `REFERENCES users(id)`: La DB garantiza que el `user_id` apunta a un usuario real. Si intentas crear un client con un `user_id` que no existe, PostgreSQL rechaza la operación.
- `ON DELETE CASCADE`: Si eliminas un usuario, **todos sus clientes se eliminan automáticamente**. Esto evita datos huérfanos.
- El índice `idx_clients_user_id` acelera las queries `WHERE user_id = ?` (como el listado de clientes de un usuario).

### ¿Por qué NO hay unique constraint en el nombre?

Dos usuarios **diferentes** pueden tener un cliente con el mismo nombre (ej: ambos pueden tener un cliente "Acme Corp"). Son entidades independientes que simplemente comparten un nombre.

Un mismo usuario también podría tener dos clientes con el mismo nombre (no se requirió restricción `UNIQUE(user_id, name)`). Si en el futuro quieres impedirlo, puedes añadir esa restricción.

---

## 2. La Entidad en el Dominio (`internal/domain/entities/client.go`)

```go
type Client struct {
    ID        uuid.UUID
    Name      string
    UserID    uuid.UUID  // ← La FK vive también en la entidad de dominio
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### ¿Por qué `UserID` está en la entidad de dominio y no solo en la DB?

Porque la **propiedad** de un client por un user es una **regla de negocio**, no solo un detalle de persistencia. El servicio necesita saber quién es el dueño para:

1. **Verificar que el usuario existe** antes de crear un client.
2. **Verificar ownership** (propiedad): que un usuario solo pueda ver/editar/borrar **sus propios** clientes.

Si `UserID` solo existiera en el modelo de DB, el servicio no podría hacer estas validaciones sin depender de infraestructura.

### Constructor `NewClient`

```go
func NewClient(name string, userID uuid.UUID) (*Client, error) {
    if name == "" {
        return nil, ErrEmptyClientName
    }
    // ...
}
```

El constructor recibe `userID` como parámetro obligatorio. No puedes crear un Client sin decir a quién pertenece. Esto garantiza la integridad desde el momento de la creación.

---

## 3. URLs Anidadas (Nested Routes)

### ¿Por qué `/users/:user_id/clients` y no `/clients`?

Las rutas reflejan la relación de propiedad:

```
POST   /api/v1/users/:user_id/clients          → Crear un client para ese user
GET    /api/v1/users/:user_id/clients           → Listar los clients de ese user
GET    /api/v1/users/:user_id/clients/:id       → Obtener un client específico
PUT    /api/v1/users/:user_id/clients/:id       → Actualizar un client
DELETE /api/v1/users/:user_id/clients/:id       → Eliminar un client
```

**Ventajas:**

- La URL deja claro que un client **pertenece a** un user.
- El `user_id` viene en la URL, no en el body. Esto es más RESTful.
- Cuando tengas autenticación (Fase 4), el `user_id` vendrá del token JWT en vez de la URL.

**Alternativa que NO elegimos:**

```
POST /api/v1/clients  { "name": "Acme", "user_id": "..." }
```

Esto funciona, pero es menos expresivo. La URL anidada comunica mejor la jerarquía.

---

## 4. Verificación de Ownership en el Servicio

### ¿Por qué `ErrClientNotOwned`?

En `client_service.go`, cuando haces `GetClientByID`, `UpdateClient` o `DeleteClient`, se verifica que el `userID` del request coincida con el `userID` del client:

```go
func (s *clientService) GetClientByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.Client, error) {
    client, err := s.clientRepo.GetByID(ctx, id)
    if err != nil {
        return nil, ErrClientNotFound
    }

    if client.UserID != userID {
        return nil, ErrClientNotOwned  // ← 403 Forbidden
    }

    return client, nil
}
```

**¿Por qué no filtrar por `user_id` directamente en la query SQL?**

Podríamos hacer `WHERE id = ? AND user_id = ?` en el repository. Pero separarlo tiene ventajas:

1. **El repository es simple**: solo busca por ID. Una responsabilidad.
2. **El servicio contiene la lógica de negocio**: decide qué hacer si no es el dueño.
3. **Puedes distinguir entre "no existe" (404) y "no es tuyo" (403)**. Si filtraras en SQL, ambos casos devuelven `nil` y no sabes cuál fue.

---

## 5. El Servicio de Client Depende del Repositorio de User

### ¿Por qué?

```go
type clientService struct {
    clientRepo repositories.ClientRepository
    userRepo   repositories.UserRepository  // ← necesita acceso a users
}
```

En `CreateClient`, antes de crear el client, verificamos que el usuario exista:

```go
func (s *clientService) CreateClient(ctx context.Context, name string, userID uuid.UUID) (*entities.Client, error) {
    // Step 1: Verify the user exists
    _, err := s.userRepo.GetByID(ctx, userID)
    if err != nil {
        return nil, ErrUserNotFound
    }
    // ...
}
```

**¿No basta con que la DB lance un FK error?**

La DB **sí** rechazaría la inserción por la FK constraint. Pero:

1. El error que devuelve la DB es un error de infraestructura (error SQL), no un error de negocio claro.
2. Al verificar en el servicio, devolvemos `ErrUserNotFound` que el handler mapea a **404**. Sin esta verificación, el handler recibiría un error genérico de DB y devolvería **500**.
3. **Fail fast**: es mejor detectar el problema temprano con un mensaje claro.

---

## 6. Wiring en `main.go`

```go
// --- User wiring ---
userRepo := persistence.NewPostgresUserRepository(db)
userService := services.NewUserService(userRepo)

// --- Client wiring ---
clientRepo := persistence.NewPostgresClientRepository(db)
clientService := services.NewClientService(clientRepo, userRepo)  // ← recibe AMBOS repos

// --- HTTP ---
router := httpAdapter.SetupRoutes(userService, clientService)
```

El `clientService` recibe el `userRepo` porque necesita verificar que los usuarios existen. No usa el `userService` sino el `userRepo` directamente, porque solo necesita la operación `GetByID` y no quiere depender de toda la interfaz `UserService`.

---

## 7. El TRUNCATE en `CleanDB`

```go
func CleanDB(t *testing.T, db *gorm.DB) {
    err := db.Exec("TRUNCATE TABLE clients, users RESTART IDENTITY CASCADE").Error
    require.NoError(t, err)
}
```

`CASCADE` en el TRUNCATE significa que al limpiar `users`, también se limpian las filas de `clients` que referencian a esos users. Lo listamos explícitamente (`clients, users`) para ser claros sobre qué tablas se limpian.

---

## 8. Resumen de Archivos Creados/Modificados

### Archivos nuevos (Client)

| Capa | Archivo | Propósito |
|------|---------|-----------|
| Domain | `internal/domain/entities/client.go` | Entidad + constructor + validación |
| Domain | `internal/domain/repositories/client_repository.go` | Interfaz del repositorio (outbound port) |
| Application | `internal/application/usecases/client_service.go` | Interfaz del servicio (use case) |
| Application | `internal/application/services/client_service.go` | Implementación del servicio |
| Infrastructure | `internal/infrastructure/adapters/outbound/persistence/postgres_client_repository.go` | Repositorio PostgreSQL + model + mappers |
| Infrastructure | `internal/infrastructure/adapters/inbound/http/client_handler.go` | Handler HTTP + DTOs + mappers |
| Tests | `test/unit/domain/entities/client_test.go` | Tests de la entidad |
| Tests | `test/unit/mocks/mock_client_repository.go` | Mock manual del repositorio |
| Tests | `test/unit/application/services/client_service_test.go` | Tests del servicio con mocks |
| Tests | `test/integration/persistence/postgres_client_repository_test.go` | Tests contra DB real |
| Tests | `test/e2e/client_test.go` | Tests HTTP completos |

### Archivos modificados

| Archivo | Cambio |
|---------|--------|
| `internal/infrastructure/adapters/inbound/http/router.go` | Añadidas rutas de Client anidadas bajo `/users/:user_id/clients` |
| `internal/infrastructure/adapters/outbound/persistence/connection.go` | Añadido `ClientModel` al AutoMigrate |
| `cmd/api/main.go` | Wiring del clientRepo, clientService, y paso al router |
| `init.sql` | Añadida tabla `clients` con FK a `users` |
| `test/helpers/setup.go` | AutoMigrate de `ClientModel` + TRUNCATE actualizado |
| `test/helpers/http.go` | Añadidos `DeleteJSON` y `ParseBodyArray` |

---

## 9. Patrón General — Lo que se Repite por Entidad

Ahora que tienes dos entidades, el patrón es claro. Para cada nueva entidad necesitas:

1. **Domain**: `entities/X.go` + `repositories/X_repository.go`
2. **Application**: `ports/X_service.go` + `services/X_service.go`
3. **Infrastructure**: `persistence/postgres_X_repository.go` + `http/X_handler.go`
4. **Wiring**: líneas en `main.go` + rutas en `router.go`
5. **DB**: tabla en `init.sql` + model en el repositorio
6. **Tests**: unit + integration + e2e

Si la siguiente entidad tarda lo mismo que esta, hay algo que se puede abstraer (ej: un error handler centralizado, que ya está planificado en la Fase 2).
