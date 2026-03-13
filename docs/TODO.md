# TODO — Code Review Findings

Hallazgos de la revisión de código. Ir marcando conforme se resuelvan.

## Alta prioridad

- [x] **`UpdateUser` no valida formato de email con VO** — Al cambiar email en `user_service.go`, no se llama a `valueobjects.NewEmail()` para validar formato.
- [x] **`UpdateUser` acepta nombre vacío** — `client_service.go` valida `*name == ""`, pero `user_service.go` no.
- [x] **Errores de repo ignorados con `_`** — En `auth_service.go` y `user_service.go`, `GetByEmail` ignora el error. Un fallo de DB se interpreta como "email disponible".

## Media prioridad

- [x] **Ownership check duplicado 3x** — `GetClientByID`, `UpdateClient`, `DeleteClient` repiten fetch + ownership. Extraer `getOwnedClient(ctx, id, userID)`.
- [ ] **Errores duplicados `ErrEmailTaken` / `ErrEmailAlreadyInUse`** — Mismo concepto en `auth_service.go` y `user_service.go`. Unificar en uno solo.
- [ ] **Boilerplate repetido `GetUserIDFromContext`** — 7 veces entre `client_handler.go` y `user_handler.go`. Extraer a middleware o helper con abort.
- [ ] **`errorMap` global mutable sin protección** — Map sin mutex, mutado desde `SetupRoutes` y tests. Los tests registran errores que quedan globalmente.
- [ ] **`init()` con `MustRegister` frágil en tests** — `metrics.go` usa `prometheus.MustRegister` en `init()`. Puede hacer panic si se importa desde múltiples test packages.

## Baja prioridad

- [ ] **`time.Now()` no inyectable** — Hardcodeado en entidades y services. Dificulta tests deterministas.
- [ ] **Update sin cambios persiste igualmente** — Si todos los campos son `nil`, se actualiza `UpdatedAt` y se hace write a DB.
- [ ] **Context keys inconsistentes** — `RequestIDKey` es `string`, `UserIDKey` es `contextKey` (typed). Unificar enfoque.
- [ ] **Constructores retornan tipos no exportados** — `NewAuthService`, `NewClientService`, `NewUserService` retornan `*authService` etc. Considerar retornar la interfaz.
- [ ] **Token duration hardcodeado** — `24 * time.Hour` en `jwt_provider.go`. Hacer configurable vía `config.JWT`.
- [ ] **GORM logger level hardcodeado** — `connection.go` siempre usa `logger.Info`. Respetar `config.Log.Level`.
- [ ] **`parseLevel` no es case-insensitive** — `LOG_LEVEL=DEBUG` no funcionaría. Añadir `strings.ToLower()`.
- [ ] **Setup repetitivo de mocks en tests** — Mismos bloques de mock se repiten en muchos tests. Usar helpers o table-driven tests.
- [ ] **No se valida `uuid.Nil` en `NewClient`** — Se puede crear un client con owner `uuid.Nil`.
- [ ] **DTOs omiten timestamps** — `ClientResponse` y `UserResponse` no incluyen `created_at`/`updated_at`.
- [ ] **Manejo inconsistente de `resp.Body.Close()`** — En contract tests, a veces se cierra manualmente y a veces no. Estandarizar convención.
