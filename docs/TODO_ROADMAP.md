# Roadmap del Backend — TODO

> Estado actual: Fases 1–4 completadas. Infraestructura HTTP con middlewares, segunda entidad (Client), autenticación JWT.
> Fecha de inicio: Marzo 2026

---

## Fase 1 — Testing de User ✅

**Objetivo**: Tener la entidad User completamente testeada antes de avanzar con más features.

### 1.1 Unit Tests
- [X] Tests del Value Object `Email` (`test/unit/domain/valueobjects/email_test.go`)
  - Emails válidos (con puntos, con +, con subdominios)
  - Emails inválidos (vacío, sin @, sin dominio, con espacios)
  - Verificar que devuelve `ErrInvalidEmail` específico
- [X] Tests de la Entity `User` (`test/unit/domain/entities/user_test.go`)
  - Creación con datos válidos → user con ID, timestamps
  - Nombre vacío → `ErrEmptyUserName`
  - Email inválido → `ErrInvalidEmail`
  - Dos users generan IDs distintos
- [X] Mock del repositorio (`test/unit/mocks/mock_user_repository.go`)
  - Mock manual con campos de tipo función (patrón idiomático en Go)
- [X] Tests del `UserService` (`test/unit/application/services/user_service_test.go`)
  - `CreateUser`: happy path, email duplicado, nombre vacío, fallo del repo
  - `GetUserByID`: user existente, user no encontrado
  - `UpdateUser`: actualizar nombre, actualizar email, email duplicado, user no encontrado

> **Guía de referencia**: `docs/GUIDE_TESTING.md` — secciones 5.2, 5.3 y 5.4.

### 1.2 Integration Tests
- [X] Helper de limpieza de DB (`CleanDB` en `test/helpers/test_utils.go`)
  - Truncar tablas entre tests para aislamiento
- [X] Tests de `PostgresUserRepository` (`test/integration/persistence/postgres_user_repository_test.go`)
  - Create + GetByID → los datos se persisten correctamente
  - GetByEmail → funciona con email existente, falla con email inexistente
  - Update → los cambios se reflejan en la DB
  - GetByID con UUID inexistente → error
  - Create con email duplicado → la DB rechaza por UNIQUE constraint
  - Verificar que los mappers (`toUserModel`/`toUserEntity`) no pierden datos

> **Requisito**: DB Postgres corriendo (`make db-start`).

### 1.3 E2E Tests
- [X] Tests HTTP completos (`test/e2e/user_e2e_test.go`)
  - POST /users → 201 + crear y luego GET para verificar persistencia
  - POST /users con email duplicado → 409
  - GET /users/{id} inexistente → 404
  - GET /users/{id} con UUID inválido → 400
  - PUT /users/{id} → 200 + verificar que cambió
  - POST /users sin body válido → 400

> **Requisito**: DB + API corriendo (`make db-start` + `make app-watch`).

### 1.4 Ejecutar y verificar
- [X] `make test-unit` pasa ✅
- [X] `make test-integration` pasa ✅
- [X] `make test-e2e` pasa ✅
- [X] `make test` (todos) pasa ✅

---

## Fase 2 — Mejoras de Infraestructura HTTP ✅

**Objetivo**: Hacer la API más robusta y profesional antes de añadir más entidades.

### 2.1 Middleware de logging
- [X] Crear middleware que loguee cada request: método, path, status code, duración
  - **¿Qué es un middleware?** Es una función que se ejecuta **antes y/o después** de cada request HTTP. Es como un filtro o interceptor. En Gin se registra con `r.Use(middleware)`.
  - Ejemplo: cada vez que alguien llama a `POST /users`, el middleware loguea: `POST /users → 201 (23ms)`
  - Gin ya tiene un logger por defecto (`gin.Default()` lo incluye), pero querrás personalizarlo para que el formato sea consistente con tus logs de aplicación
  - **Dónde**: `internal/infrastructure/adapters/inbound/http/middleware/logger.go`

### 2.2 Middleware de recovery
- [X] Crear middleware que capture panics y devuelva 500 en vez de crashear el servidor
  - Si un handler tiene un bug y hace `panic`, sin recovery el servidor muere. Con el middleware, devuelve un 500 limpio y sigue funcionando
  - Gin ya incluye uno con `gin.Default()`, pero puedes personalizarlo para loguear el stack trace
  - **Dónde**: `internal/infrastructure/adapters/inbound/http/middleware/recovery.go`

### 2.3 Middleware de CORS
- [X] Configurar CORS para que el frontend (en otro dominio/puerto) pueda llamar a la API
  - **¿Qué es CORS?** Cuando el frontend corre en `localhost:3000` y la API en `localhost:8080`, el navegador bloquea las requests por seguridad. CORS le dice al navegador "sí, deja pasar requests desde ese origen"
  - Sin esto, el frontend no puede llamar a tu API desde el navegador
  - Librería recomendada: `github.com/gin-contrib/cors`
  - **Dónde**: Configurar en `router.go`

### 2.4 Middleware de request ID
- [X] Añadir un UUID único a cada request para trazabilidad
  - Cada request que entra recibe un ID (ej: `X-Request-ID: abc-123`). Si algo falla, puedes buscar ese ID en los logs y ver toda la traza
  - El ID se propaga a través del `context.Context` a servicios y repositorios
  - Muy útil para debugging en producción
  - **Dónde**: `internal/infrastructure/adapters/inbound/http/middleware/request_id.go`

### 2.5 Manejo de errores centralizado
- [X] Crear un middleware o helper que mapee errores de dominio a status HTTP de forma consistente
  - Ahora mismo el mapping está en cada handler (`ErrEmailAlreadyInUse → 409`, `ErrUserNotFound → 404`). Cuando tengas 10 entidades, repetirás ese switch en cada handler
  - La idea es centralizar: un solo lugar que sabe que `ErrNotFound → 404`, `ErrConflict → 409`, etc.
  - **Dónde**: `internal/infrastructure/adapters/inbound/http/middleware/error_handler.go`

### 2.6 Versionado de API
- [X] Mover las rutas bajo `/api/v1/`
  - Ahora: `POST /users`
  - Después: `POST /api/v1/users`
  - Esto permite en el futuro tener `/api/v2/` sin romper clientes existentes
  - **Dónde**: Cambiar en `router.go` → `r.Group("/api/v1")`

---

## Fase 3 — Segunda Entidad ✅

**Objetivo**: Aplicar todo lo aprendido con User a una nueva entidad. Consolidar el patrón.

### 3.1 Elegir la entidad
- [X] Decidir qué entidad necesita el negocio (ej: `Project`, `Organization`, `Task`...)
  - Se eligió `Client` — un cliente con solo un nombre, perteneciente a un User (relación 1:N)
  - Dos usuarios pueden tener un cliente con el mismo nombre, pero son entidades independientes

### 3.2 Implementar siguiendo el mismo patrón
- [X] Domain: Entity + Value Objects + Repository interface
- [X] Application: Service + Port interface
- [X] Infrastructure: Handler + DTOs + PostgresRepository + Router
- [X] Wiring en `main.go`

### 3.3 Tests de la segunda entidad
- [X] Unit tests (entity, value objects, service con mocks)
- [X] Integration tests (repository)
- [X] E2E tests (flujo HTTP completo)

> **Guía de referencia**: `docs/GUIDE_CLIENT_ENTITY.md` — explicación detallada de todas las decisiones.

---

## Fase 4 — Autenticación y Autorización ✅

**Objetivo**: Proteger la API para que solo usuarios autenticados puedan usarla.

### 4.1 Elegir estrategia de auth
- [X] Decidir entre:
  - **JWT (JSON Web Tokens)**: El usuario hace login, recibe un token, y lo envía en cada request. El backend valida el token sin consultar la DB. Es stateless y el más común para APIs REST.
  - **Sessions**: El servidor guarda la sesión en DB/Redis. Más simple pero stateful.
  - **OAuth2 / OIDC con provider externo**: Delegas el login a Google, GitHub, Auth0, etc. Más complejo pero no gestionas contraseñas.
  - **Recomendación para empezar**: JWT propio, y más adelante integrar un provider externo si lo necesitas.

### 4.2 Implementar registro y login
- [X] Endpoint `POST /auth/register` → crear usuario con contraseña (hash con bcrypt)
- [X] Endpoint `POST /auth/login` → validar credenciales, devolver JWT
- [X] Añadir campo `PasswordHash` a la entidad User (o crear entidad `Credentials` separada)
- [X] Value Object para `Password` (mínimo 8 chars, etc.)

### 4.3 Middleware de autenticación
- [X] Crear middleware que:
  1. Lee el header `Authorization: Bearer <token>`
  2. Valida el JWT (firma, expiración)
  3. Extrae el `userID` del token y lo pone en el `context.Context`
  4. Si el token es inválido, devuelve 401
- [X] Aplicar el middleware a las rutas protegidas (todo menos `/auth/*` y `/health`)
- [X] **Dónde**: `internal/infrastructure/adapters/inbound/http/middleware/auth.go`

### 4.4 Autorización básica
- [X] Verificar que un usuario solo puede modificar **sus propios datos**
  - Rutas cambiadas a `/users/me` y `/users/me/clients` — el userID viene del JWT, no de la URL
  - Los clients se validan con `ErrClientNotOwned` en el servicio
- [X] Definir roles si es necesario (admin, user) — no necesario por ahora, se usa ownership

### 4.5 Tests de auth
- [X] Unit tests: validación de JWT, hash de passwords
- [X] Integration tests: registro + login contra DB
- [X] E2E tests: flujo completo (register → login → acceder ruta protegida → 401 sin token)

> **Guía de referencia**: `docs/GUIDE_AUTH_JWT.md` — explicación detallada de todas las decisiones.

---

## Fase 5 — API Contracts

**Objetivo**: Definir el contrato formal de la API cuando el frontend esté listo para consumirla.

### 5.1 Crear el OpenAPI spec
- [ ] Escribir `contracts/openapi/openapi.yaml` con todos los endpoints implementados
  - Incluir los schemas de request/response de todas las entidades
  - Incluir los endpoints de auth
  - Incluir todos los status codes posibles y sus responses
  - Seguir el enfoque **contract-first** (la guía ya está en `docs/GUIDE_CONTRACTS.md`)

### 5.2 Contract tests en el backend
- [ ] Helper de validación (`test/contract/contract_helper.go`)
- [ ] Contract tests para cada endpoint (`test/contract/user_contract_test.go`, etc.)
- [ ] Verificar: status codes, campos requeridos, tipos de datos

### 5.3 Integrar con el frontend
- [ ] Frontend genera tipos TypeScript desde el spec
- [ ] Validar el spec en CI (`redocly lint`)
- [ ] Swagger UI disponible en desarrollo

---

## Fase 6 — Observabilidad

**Objetivo**: Poder saber qué pasa en producción cuando algo falla.

### 6.1 Logging estructurado
- [ ] Reemplazar `log.Printf` por un logger estructurado (ej: `slog` de la stdlib de Go 1.21+, o `zerolog`)
  - **¿Qué es logging estructurado?** En vez de `log.Printf("user created: %s", id)` haces `logger.Info("user created", "user_id", id, "email", email)`. Los logs salen en JSON, lo que permite buscarlos y filtrarlos en herramientas como Grafana/Loki.
  - Ejemplo de output: `{"level":"info","msg":"user created","user_id":"abc-123","email":"juan@ex.com","time":"2026-03-03T10:00:00Z"}`kiy

### 6.2 Health check mejorado
- [ ] Que `/health` también verifique la conexión a la DB
  - Ahora devuelve siempre "healthy". Debería hacer un `db.Ping()` y devolver "unhealthy" si la DB no responde
  - Útil para que Docker/Kubernetes sepa si tu servicio realmente funciona

### 6.3 Métricas (opcional, más adelante)
- [ ] Exponer métricas Prometheus: requests/segundo, latencias, errores
- [ ] Esto es para cuando tengas monitoreo real en producción

---

## Fase 7 — CI/CD

**Objetivo**: Automatizar la validación y el despliegue.

### 7.1 GitHub Actions
- [ ] Workflow que en cada PR ejecute:
  1. `go vet ./...` (análisis estático)
  2. `go test ./test/unit/...` (unit tests)
  3. Levantar DB en Docker + `go test ./test/integration/...` (integration tests)
  4. Levantar DB + API + `go test ./test/e2e/...` (E2E tests)
  5. `golangci-lint run` (linter)

### 7.2 Docker optimizado
- [ ] Multi-stage Dockerfile (ya lo tienes, pero verificar que es óptimo)
- [ ] Build cache para que las dependencias no se descarguen en cada build

### 7.3 Despliegue
- [ ] Definir dónde desplegar (Railway, Fly.io, AWS, GCP...)
- [ ] Configurar deploy automático desde `main`

---

## Resumen visual del orden

```
Fase 1  ████████████████████  Testing de User        ✅
Fase 2  ████████████████████  Middleware + Infra HTTP ✅
Fase 3  ████████████████████  Segunda entidad         ✅
Fase 4  ████████████████████  Autenticación           ✅
Fase 5  ░░░░░░░░░░░░░░░░░░░░  API Contracts          ⬅️ SIGUIENTE
Fase 6  ░░░░░░░░░░░░░░░░░░░░  Observabilidad
Fase 7  ░░░░░░░░░░░░░░░░░░░░  CI/CD
```

> **Regla**: No saltes fases. Cada fase construye sobre la anterior. Si los tests de la Fase 1 no pasan, no tiene sentido avanzar a la Fase 2.
