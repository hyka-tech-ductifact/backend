# Guía de API Contracts

## Índice
1. [¿Qué es un Contract?](#1-qué-es-un-contract)
2. [¿Por qué un repo separado?](#2-por-qué-un-repo-separado)
3. [OpenAPI (Swagger) como formato](#3-openapi-swagger-como-formato)
4. [Estructura del repo contracts](#4-estructura-del-repo-contracts)
5. [Nuestro contract: openapi.yaml](#5-nuestro-contract-openapiyaml)
   - 5.1 [Schemas que definen la forma de los datos](#51-schemas-que-definen-la-forma-de-los-datos)
   - 5.2 [Endpoints (paths)](#52-endpoints-paths)
   - 5.3 [Seguridad (JWT Bearer)](#53-seguridad-jwt-bearer)
   - 5.4 [Respuestas de error](#54-respuestas-de-error)
6. [Cómo usar el contract](#6-cómo-usar-el-contract)
7. [Contract Tests en el backend](#7-contract-tests-en-el-backend)
   - 7.1 [¿Qué son los contract tests?](#71-qué-son-los-contract-tests)
   - 7.2 [Contract tests vs E2E tests](#72-contract-tests-vs-e2e-tests)
   - 7.3 [Implementación en Go](#73-implementación-en-go)
8. [Flujo de trabajo completo](#8-flujo-de-trabajo-completo)
9. [Buenas prácticas](#9-buenas-prácticas)

---

## 1. ¿Qué es un Contract?

Un contract (contrato) es un **documento formal que define la interfaz pública de tu API**:

- Qué endpoints existen
- Qué request body esperan
- Qué response body devuelven
- Qué status codes usan
- Qué tipos de datos tiene cada campo
- Qué endpoints requieren autenticación

Es un **acuerdo** entre el backend (productor) y el frontend (consumidor). Ambos se comprometen a respetar lo que dice el contrato.

**Analogía**: Es como un contrato legal. El backend dice "yo me comprometo a devolver esto" y el frontend dice "yo me comprometo a enviar esto". Si alguno rompe el contrato, los tests lo detectan.

---

## 2. ¿Por qué un repo separado?

Tu repo `contracts/` es compartido entre backend y frontend. La razón:

```
contracts/          ← Fuente de verdad (OpenAPI spec)
    │
    ├──── backend/  consume el contract para:
    │     - Contract tests (validar que cumple el contrato)
    │     - Generar documentación automática
    │
    └──── frontend/ consume el contract para:
          - Generar tipos TypeScript automáticamente
          - Generar clientes HTTP (fetch/axios) automáticos
          - Mockear la API en tests del frontend
```

**Ventajas**:
- **Fuente de verdad única**: no hay discrepancias entre lo que el backend hace y lo que el frontend espera.
- **Versionado independiente**: puedes versionar los contracts sin depender de releases del backend o frontend.
- **Generación de código**: ambos lados pueden auto-generar tipos y clientes desde el contrato.

---

## 3. OpenAPI (Swagger) como formato

Usamos **OpenAPI 3.0** (antes llamado Swagger). Es el estándar de la industria para definir APIs REST.

**¿Por qué OpenAPI?**
- Lo entiende todo el mundo (backend, frontend, QA, DevOps).
- Herramientas automáticas: Swagger UI, generadores de código, validadores.
- Se escribe en YAML (legible) o JSON.
- Integración con CI/CD para validación automática.

---

## 4. Estructura del repo contracts

```
contracts/
├── README.md
├── LICENSE
└── openapi/
    ├── openapi.yaml          ← Archivo raíz (info, servers, security, $ref)
    ├── paths/
    │   ├── health.yaml       ← Endpoints de /health
    │   ├── auth.yaml         ← Endpoints de /auth/*
    │   ├── users.yaml        ← Endpoints de /users/me
    │   └── clients.yaml      ← Endpoints de /users/me/clients*
    └── schemas/
        ├── auth.yaml         ← RegisterRequest, LoginRequest, AuthResponse
        ├── user.yaml         ← UserResponse, UpdateUserRequest
        ├── client.yaml       ← ClientResponse, CreateClientRequest, UpdateClientRequest
        ├── health.yaml       ← HealthResponse, HealthResponseUnhealthy
        └── error.yaml        ← ErrorResponse
```

**¿Por qué dividirlo?**
- El archivo raíz `openapi.yaml` queda como un **índice** limpio (~60 líneas) que importa todo lo demás con `$ref`.
- Cuando añades una nueva entidad, creas `paths/nueva.yaml` + `schemas/nueva.yaml` sin tocar los demás archivos.
- Cada archivo es pequeño y fácil de revisar en un PR.
- Las herramientas modernas (Swagger UI, redocly, openapi-typescript) soportan `$ref` a archivos externos sin problemas.

**Cómo funcionan las `$ref`:**
```yaml
# En openapi.yaml — referencia a un path en otro archivo
paths:
  /auth/register:
    $ref: "paths/auth.yaml#/register"

# En openapi.yaml — referencia a un schema en otro archivo
components:
  schemas:
    UserResponse:
      $ref: "schemas/user.yaml#/UserResponse"

# En schemas/auth.yaml — referencia cruzada entre schemas
AuthResponse:
  properties:
    user:
      $ref: "user.yaml#/UserResponse"    # relativo al directorio schemas/
```

---

## 5. Nuestro contract: openapi.yaml

Nuestro contract refleja todos los endpoints de la API. Vamos a ver cómo se relaciona con el código Go.

### 5.1 Schemas que definen la forma de los datos

Los schemas definen la **forma** de los datos. Son equivalentes a los DTOs de tus handlers:

**En Go (código real del backend):**
```go
// auth_handler.go
type RegisterRequest struct {
    Name     string `json:"name" binding:"required"`
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8"`
}

type AuthResponse struct {
    User  UserResponse `json:"user"`
    Token string       `json:"token"`
}

// user_handler.go
type UserResponse struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// client_handler.go
type ClientResponse struct {
    ID     string `json:"id"`
    Name   string `json:"name"`
    UserID string `json:"user_id"`
}
```

**En OpenAPI (el contrato):**
```yaml
components:
  schemas:
    RegisterRequest:
      type: object
      required: [name, email, password]
      properties:
        name:
          type: string
          minLength: 1
        email:
          type: string
          format: email
        password:
          type: string
          minLength: 8

    AuthResponse:
      type: object
      required: [user, token]
      properties:
        user:
          $ref: "#/components/schemas/UserResponse"
        token:
          type: string

    UserResponse:
      type: object
      required: [id, name, email]
      properties:
        id:
          type: string
          format: uuid
        name:
          type: string
        email:
          type: string
          format: email

    ClientResponse:
      type: object
      required: [id, name, user_id]
      properties:
        id:
          type: string
          format: uuid
        name:
          type: string
        user_id:
          type: string
          format: uuid
```

**¿Ves la relación?** Cada campo del struct de Go tiene su equivalente en el schema YAML, con el tipo y las validaciones.

### 5.2 Endpoints (paths)

Nuestras rutas actuales (desde `router.go`):

| Método | Ruta | Auth? | Descripción |
|--------|------|-------|-------------|
| GET | `/api/v1/health` | No | Health check |
| POST | `/api/v1/auth/register` | No | Registro de usuario |
| POST | `/api/v1/auth/login` | No | Login |
| GET | `/api/v1/users/me` | Sí | Obtener perfil del usuario autenticado |
| PUT | `/api/v1/users/me` | Sí | Actualizar perfil |
| POST | `/api/v1/users/me/clients` | Sí | Crear cliente |
| GET | `/api/v1/users/me/clients` | Sí | Listar clientes |
| GET | `/api/v1/users/me/clients/:client_id` | Sí | Obtener un cliente |
| PUT | `/api/v1/users/me/clients/:client_id` | Sí | Actualizar un cliente |
| DELETE | `/api/v1/users/me/clients/:client_id` | Sí | Eliminar un cliente |

En OpenAPI, cada ruta se describe con su request, responses posibles y si requiere auth:

```yaml
paths:
  /auth/register:
    post:
      operationId: register
      security: []                    # ← Sin auth
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/RegisterRequest"
      responses:
        "201":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/AuthResponse"
        "400": ...
        "409": ...

  /users/me:
    get:
      operationId: getMe
      security:
        - bearerAuth: []              # ← Requiere JWT
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/UserResponse"
        "401": ...
```

### 5.3 Seguridad (JWT Bearer)

El contract también describe el esquema de autenticación:

```yaml
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
```

Los endpoints protegidos lo referencian con `security: [{bearerAuth: []}]`. Esto permite que Swagger UI muestre el botón "Authorize" y que generadores de código incluyan automáticamente el header `Authorization: Bearer <token>`.

### 5.4 Respuestas de error

Todos los handlers devuelven errores con la misma forma:

```go
c.JSON(status, gin.H{"error": err.Error()})
```

En el contract, esto se define una sola vez y se reutiliza:

```yaml
ErrorResponse:
  type: object
  required: [error]
  properties:
    error:
      type: string
      example: "email already in use"
```

---

## 6. Cómo usar el contract

### 6.1 Visualizar con Swagger UI

Puedes ver tu API de forma interactiva:

```bash
# Opción 1: Online
# Ve a https://editor.swagger.io/ y pega tu openapi.yaml

# Opción 2: Docker local
docker run -p 8081:8080 -e SWAGGER_JSON=/api/openapi.yaml \
  -v $(pwd)/contracts/openapi:/api \
  swaggerapi/swagger-ui
# Luego abre http://localhost:8081
```

### 6.2 Generar tipos TypeScript para el frontend

El frontend puede generar tipos automáticos desde el contrato:

```bash
# Usando openapi-typescript (en el repo del frontend)
npx openapi-typescript ../contracts/openapi/openapi.yaml -o src/api/types.ts
```

Esto genera algo como:

```typescript
// Auto-generado desde el contract
export interface RegisterRequest {
  name: string;
  email: string;
  password: string;
}

export interface AuthResponse {
  user: UserResponse;
  token: string;
}

export interface UserResponse {
  id: string;
  name: string;
  email: string;
}

export interface ClientResponse {
  id: string;
  name: string;
  user_id: string;
}
```

**El frontend nunca define estos tipos a mano** — siempre se generan desde el contrato.

### 6.3 Validar el contract en CI

```bash
# Instalar el validador
npm install -g @redocly/cli

# Validar que el YAML es un OpenAPI válido
redocly lint contracts/openapi/openapi.yaml
```

---

## 7. Contract Tests en el backend

### 7.1 ¿Qué son los contract tests?

Los contract tests verifican que **tu API cumple con lo que el contract promete**. No testean lógica de negocio — solo la **forma** de las respuestas:

| Verifican | No verifican |
|-----------|-------------|
| Status code correcto (201, 400, 404...) | Que los datos sean lógicamente correctos |
| Campos requeridos presentes en la respuesta | Que la DB tenga el registro |
| Tipos de datos correctos (string, uuid...) | Flujos de múltiples pasos |
| Estructura del JSON de error | Side effects (emails, logs...) |
| Respuesta 401 sin token | Que la lógica de negocio funcione |

### 7.2 Contract tests vs E2E tests — La diferencia clave

```
┌──────────────────────────────────────────────────────────────────┐
│                         E2E TEST                                 │
│                                                                  │
│  1. POST /auth/register con credenciales                        │
│  2. POST /auth/login → obtener token                            │
│  3. POST /users/me/clients con token → crear cliente   ← FLUJO │
│  4. GET /users/me/clients → listar y verificar         ← LÓGICA│
│  5. PUT /users/me/clients/:id → actualizar             ← REGLA │
│  6. DELETE /users/me/clients/:id → eliminar            ← NEGOCIO│
│  7. GET /users/me/clients/:id → 404 (ya no existe)    ← FLUJO │
│                                                                  │
│  → Testea que el SISTEMA FUNCIONA end-to-end                     │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                      CONTRACT TEST                               │
│                                                                  │
│  Test 1: POST /auth/register con body válido                     │
│    → ¿Status es 201?                                    ✅      │
│    → ¿Response tiene campo "user" tipo object?          ✅      │
│    → ¿Response tiene campo "token" tipo string?         ✅      │
│    → ¿User tiene "id", "name", "email" tipo string?    ✅      │
│                                                                  │
│  Test 2: GET /users/me sin token                                 │
│    → ¿Status es 401?                                    ✅      │
│    → ¿Response tiene campo "error" tipo string?         ✅      │
│                                                                  │
│  Test 3: POST /users/me/clients con body válido + token          │
│    → ¿Status es 201?                                    ✅      │
│    → ¿Response tiene "id", "name", "user_id" strings?  ✅      │
│                                                                  │
│  Test 4: DELETE /users/me/clients/:id existente                  │
│    → ¿Status es 204?                                    ✅      │
│    → ¿No tiene body?                                    ✅      │
│                                                                  │
│  → Testea que la FORMA de la API cumple el contrato              │
│  → NO encadena pasos, cada test es independiente                 │
└──────────────────────────────────────────────────────────────────┘
```

**Diferencias concretas:**

| | E2E | Contract |
|---|-----|----------|
| **Encadena pasos** | ✅ Sí (register → login → crear → listar) | ❌ No, cada test es atómico |
| **Verifica datos** | ✅ "El name es Juan" | ❌ "Hay un campo name y es string" |
| **Verifica reglas** | ✅ "Email duplicado → 409" | ⚠️ Solo "409 tiene formato ErrorResponse" |
| **Verifica auth** | ✅ "Token correcto permite acceso" | ⚠️ Solo "sin token → 401 con ErrorResponse" |
| **Fuente de verdad** | El código del backend | El archivo OpenAPI |
| **¿Quién los necesita?** | Solo el backend | Backend + Frontend |
| **¿Cuándo fallan?** | Cuando la lógica se rompe | Cuando alguien cambia la forma de la API |

**¿Pueden coexistir? Sí, y deben.** Los E2E testean que "funciona", los contracts testean que "cumples lo prometido". Un backend puede funcionar perfecto (E2E pasan) pero haber roto el contrato (por ejemplo, renombrar `user_id` a `userId`). Los E2E no detectan eso, los contracts sí.

### 7.3 Implementación en Go

#### Arquitectura de los contract tests

```
test/contract/
├── contract_helper.go          ← Validator + schemas (no es _test.go)
├── main_test.go                ← TestMain: setup de DB + API + helpers
├── health_contract_test.go     ← Contract tests de /health
├── auth_contract_test.go       ← Contract tests de /auth/*
├── user_contract_test.go       ← Contract tests de /users/me
└── client_contract_test.go     ← Contract tests de /users/me/clients/*
```

#### El helper: `contract_helper.go`

Este archivo **no es un test** — es un paquete que exporta herramientas de validación. Está en `test/contract/` (no en `test/contract_test/`) para que los test files lo importen.

```go
package contract

// ContractValidator validates API responses against the OpenAPI spec.
type ContractValidator struct {
    t          *testing.T
    specPath   string
    apiBaseURL string
}

// ResponseSchema holds the expected shape of a response.
type ResponseSchema struct {
    RequiredFields []string
    FieldTypes     map[string]string // "string", "number", "boolean", "object", "array"
}

// Validate checks: status code + required fields + field types.
func (rs *ResponseSchema) Validate(cv *ContractValidator, resp *http.Response, expectedStatus int) map[string]any

// ValidateNoBody checks status code when no body is expected (e.g., 204).
func (cv *ContractValidator) ValidateNoBody(resp *http.Response, expectedStatus int)

// ValidateArrayResponse checks a JSON array where each item matches the schema.
func (cv *ContractValidator) ValidateArrayResponse(resp, status, requiredFields, fieldTypes) []map[string]any
```

**Key decisions:**
- `ValidateResponse` devuelve `map[string]any` para que los tests puedan inspeccionar valores anidados (como `AuthResponse.user`).
- `ValidateNoBody` para DELETE 204 (respuestas sin cuerpo).
- `ValidateArrayResponse` para listas como `GET /users/me/clients`.

#### Los schemas en Go (reflejan el YAML)

```go
// Mirrors openapi.yaml → components.schemas.AuthResponse
var AuthResponseSchema = ResponseSchema{
    RequiredFields: []string{"user", "token"},
    FieldTypes: map[string]string{
        "user":  "object",
        "token": "string",
    },
}

// Mirrors openapi.yaml → components.schemas.UserResponse
var UserResponseSchema = ResponseSchema{
    RequiredFields: []string{"id", "name", "email"},
    FieldTypes: map[string]string{
        "id":    "string",
        "name":  "string",
        "email": "string",
    },
}

// Mirrors openapi.yaml → components.schemas.ClientResponse
var ClientResponseSchema = ResponseSchema{
    RequiredFields: []string{"id", "name", "user_id"},
    FieldTypes: map[string]string{
        "id":      "string",
        "name":    "string",
        "user_id": "string",
    },
}

// Mirrors openapi.yaml → components.schemas.ErrorResponse
var ErrorResponseSchema = ResponseSchema{
    RequiredFields: []string{"error"},
    FieldTypes: map[string]string{
        "error": "string",
    },
}
```

#### El setup: `main_test.go`

Sigue el mismo patrón que los E2E tests — comparten `TestMain`, `clean(t)`, y helpers:

```go
func TestMain(m *testing.M) {
    helpers.LoadEnv()
    baseURL := "http://" + host + ":" + port
    waitForAPI(baseURL, 10)
    db := helpers.ConnectTestDB()
    // ...
}

func clean(t *testing.T) { helpers.CleanDB(t, env.db) }
func url(path string) string { return env.baseURL + "/api/v1" + path }
```

**Diferencia con E2E**: los contract tests incluyen un helper `registerAndLogin()` que crea un usuario y devuelve el token. Muchos tests lo necesitan para endpoints protegidos, pero no verifican la respuesta del registro — solo necesitan el token.

#### Ejemplo de contract test

```go
func TestContract_CreateClient_ValidBody_Returns201_WithClientResponse(t *testing.T) {
    clean(t)
    cv := newValidator(t)

    token := registerAndLogin(t, "Client Owner", "contract-client@example.com", "securepass123")

    resp := helpers.AuthPostJSON(t, url("/users/me/clients"), token, map[string]string{
        "name": "Acme Corp",
    })

    // Contract says: 201 + ClientResponse{id, name, user_id}
    contract.ClientResponseSchema.Validate(cv, resp, http.StatusCreated)
}
```

**¿Qué valida esto?**
1. ✅ Status code es 201 (no 200, no 500)
2. ✅ El response tiene `id`, `name`, `user_id` (campos requeridos)
3. ✅ Los tres campos son de tipo `string`
4. ❌ NO verifica que `name` sea "Acme Corp" (eso es lógica, no contrato)
5. ❌ NO verifica que `user_id` corresponda al usuario del token

#### Ejecutar los contract tests

```bash
# Requisitos: DB + API corriendo
make db-start
make app-watch   # en otra terminal

# Ejecutar
make test-contract
```

---

## 8. Flujo de trabajo completo

### Cuando añades un nuevo endpoint:

```
1. Actualizar openapi.yaml en contracts/    ← Primero el contrato
2. Actualizar los schemas en contract_helper.go (si hay nuevos schemas)
3. Implementar en backend/                   ← Después el código
4. Añadir contract test                      ← Verificar que cumples
5. Añadir unit + integration + E2E tests     ← Verificar que funciona
6. Frontend genera tipos desde el contract   ← Auto-generado
```

### Cuando cambias un campo del response:

```
1. Actualizar openapi.yaml                  ← ¿Es breaking change?
2. Actualizar ResponseSchema en contract_helper.go
3. Actualizar el handler/DTO en backend
4. Los contract tests fallan si no coincide  ← Te protegen
5. Frontend re-genera tipos                  ← Se adapta automáticamente
```

### En CI/CD:

```yaml
# .github/workflows/ci.yml (ejemplo)
jobs:
  test:
    steps:
      - name: Unit tests
        run: make test-unit

      - name: Integration tests
        run: make test-integration

      - name: Contract tests
        run: make test-contract

      - name: E2E tests
        run: make test-e2e

      - name: Validate OpenAPI spec
        run: redocly lint contracts/openapi/openapi.yaml
```

---

## 9. Buenas prácticas

### 9.1 Contract-First Development

Siempre define el contrato **antes** de implementar:

```
❌ Backend implementa → extrae el spec → lo pone en contracts/
✅ Diseña el spec → backend implementa → contract tests validan
```

### 9.2 No dupliques validaciones

- **Contract test**: ¿El status code y la forma del JSON son correctos?
- **E2E test**: ¿La lógica de negocio funciona?
- **No testees lo mismo en ambos.**

```go
// Contract test — only shape
contract.ClientResponseSchema.Validate(cv, resp, http.StatusCreated)

// E2E test — shape + data + flow
assert.Equal(t, "Acme Corp", created["name"])
listResp := AuthGetJSON(t, url("/users/me/clients"), token)
assert.Len(t, clients, 1) // verifies persistence
```

### 9.3 Versionado del contract

Cuando hagas breaking changes, versiona:

```yaml
# Opción 1: En la URL (ya tenemos /api/v1/)
/api/v1/users    → versión actual
/api/v2/users    → nueva versión

# Opción 2: En el archivo
openapi-v1.yaml
openapi-v2.yaml
```

### 9.4 Emails únicos en tests

Usa prefijos por tipo de test para evitar colisiones con la DB:

```go
// E2E tests
"e2e-create@example.com"

// Contract tests
"contract-create@example.com"

// Integration tests
"integration-create@example.com"
```

### 9.5 Resumen visual

```
openapi.yaml (en contracts/)
     │
     ├──→ Backend: Contract tests validan que las respuestas cumplen
     │              (test/contract/*_contract_test.go)
     │
     ├──→ Frontend: Genera tipos TypeScript automáticos
     │
     ├──→ Swagger UI: Documentación interactiva
     │
     └──→ CI: Valida que el spec es válido + tests pasan
```
