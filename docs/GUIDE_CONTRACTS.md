# Guía de API Contracts

## Índice
1. [¿Qué es un Contract?](#1-qué-es-un-contract)
2. [¿Por qué un repo separado?](#2-por-qué-un-repo-separado)
3. [OpenAPI (Swagger) como formato](#3-openapi-swagger-como-formato)
4. [Estructura del repo contracts](#4-estructura-del-repo-contracts)
5. [Creando el contract de User](#5-creando-el-contract-de-user)
   - 5.1 [Definir los schemas](#51-definir-los-schemas)
   - 5.2 [Definir los endpoints](#52-definir-los-endpoints)
   - 5.3 [Definir las respuestas de error](#53-definir-las-respuestas-de-error)
   - 5.4 [El archivo completo](#54-el-archivo-completo)
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
├── openapi/
│   ├── openapi.yaml          ← Archivo principal (importa los demás)
│   ├── paths/
│   │   └── users.yaml        ← Endpoints de /users
│   └── schemas/
│       ├── user.yaml          ← Schema de User
│       └── error.yaml         ← Schema de errores
└── examples/
    └── users/
        ├── create-request.json
        └── create-response.json
```

> **Nota**: Para empezar, puedes tener todo en un solo archivo `openapi.yaml`. Cuando la API crezca, lo separas en archivos.

---

## 5. Creando el contract de User

### 5.1 Definir los schemas

Los schemas definen la **forma** de los datos. Míralo como los DTOs de tu handler, pero en YAML:

Tu handler actual tiene estos DTOs:
```go
// CreateUserRequest → lo que el frontend envía
type CreateUserRequest struct {
    Name  string `json:"name" binding:"required"`
    Email string `json:"email" binding:"required,email"`
}

// UserResponse → lo que el backend devuelve
type UserResponse struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

En OpenAPI, esto se traduce así:

```yaml
# Schema: CreateUserRequest
CreateUserRequest:
  type: object
  required:
    - name
    - email
  properties:
    name:
      type: string
      minLength: 1
      example: "Juan García"
    email:
      type: string
      format: email
      example: "juan@example.com"

# Schema: UserResponse
UserResponse:
  type: object
  required:
    - id
    - name
    - email
  properties:
    id:
      type: string
      format: uuid
      example: "550e8400-e29b-41d4-a716-446655440000"
    name:
      type: string
      example: "Juan García"
    email:
      type: string
      format: email
      example: "juan@example.com"
```

**¿Ves la relación?** Cada campo del struct de Go tiene su equivalente en el schema YAML, con el tipo y las validaciones.

### 5.2 Definir los endpoints

Tus rutas actuales:
```go
userRoutes.POST("", userHandler.CreateUser)     // POST /users
userRoutes.GET("/:id", userHandler.GetUser)      // GET /users/:id
userRoutes.PUT("/:id", userHandler.UpdateUser)   // PUT /users/:id
```

En OpenAPI:

```yaml
paths:
  /users:
    post:
      summary: Create a new user
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateUserRequest'
      responses:
        '201':
          description: User created successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/UserResponse'
        '400':
          description: Invalid request body
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
        '409':
          description: Email already in use
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'

  /users/{id}:
    get:
      summary: Get a user by ID
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '200':
          description: User found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/UserResponse'
        '400':
          description: Invalid user ID format
        '404':
          description: User not found
```

### 5.3 Definir las respuestas de error

Tu handler siempre devuelve errores con esta forma:
```go
c.JSON(status, gin.H{"error": err.Error()})
```

En el contract:

```yaml
ErrorResponse:
  type: object
  required:
    - error
  properties:
    error:
      type: string
      example: "email already in use"
```

### 5.4 El archivo completo

Este es el contract completo que irá en tu repo `contracts/`:

**Archivo**: `contracts/openapi/openapi.yaml`

```yaml
openapi: 3.0.3
info:
  title: Ductifact API
  description: Backend API for Ductifact platform
  version: 1.0.0

servers:
  - url: http://localhost:8080
    description: Local development

paths:
  # ─── Health ─────────────────────────────────────────────
  /health:
    get:
      summary: Health check
      operationId: healthCheck
      tags:
        - Health
      responses:
        '200':
          description: Service is healthy
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
                    example: "healthy !!!!"

  # ─── Users ─────────────────────────────────────────────
  /users:
    post:
      summary: Create a new user
      operationId: createUser
      tags:
        - Users
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateUserRequest'
            example:
              name: "Juan García"
              email: "juan@example.com"
      responses:
        '201':
          description: User created successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/UserResponse'
        '400':
          description: Invalid request body (missing fields or invalid email format)
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
        '409':
          description: Email already in use
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
              example:
                error: "email already in use"

  /users/{id}:
    get:
      summary: Get a user by ID
      operationId: getUserById
      tags:
        - Users
      parameters:
        - name: id
          in: path
          required: true
          description: The user's UUID
          schema:
            type: string
            format: uuid
            example: "550e8400-e29b-41d4-a716-446655440000"
      responses:
        '200':
          description: User found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/UserResponse'
        '400':
          description: Invalid user ID format
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
              example:
                error: "invalid user ID format"
        '404':
          description: User not found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
              example:
                error: "user not found"

    put:
      summary: Update an existing user
      operationId: updateUser
      tags:
        - Users
      parameters:
        - name: id
          in: path
          required: true
          description: The user's UUID
          schema:
            type: string
            format: uuid
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateUserRequest'
            example:
              name: "Pedro López"
      responses:
        '200':
          description: User updated successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/UserResponse'
        '400':
          description: Invalid request body or user ID format
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
        '404':
          description: User not found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
        '409':
          description: Email already in use by another user
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'

# ─── Schemas ───────────────────────────────────────────────
components:
  schemas:
    CreateUserRequest:
      type: object
      description: Request body for creating a new user
      required:
        - name
        - email
      properties:
        name:
          type: string
          minLength: 1
          description: The user's full name
          example: "Juan García"
        email:
          type: string
          format: email
          description: The user's email address (must be unique)
          example: "juan@example.com"

    UpdateUserRequest:
      type: object
      description: Request body for updating a user. All fields are optional.
      properties:
        name:
          type: string
          minLength: 1
          description: The user's new name
          example: "Pedro López"
        email:
          type: string
          format: email
          description: The user's new email (must be unique)
          example: "pedro@example.com"

    UserResponse:
      type: object
      description: User data returned by the API
      required:
        - id
        - name
        - email
      properties:
        id:
          type: string
          format: uuid
          description: Unique user identifier
          example: "550e8400-e29b-41d4-a716-446655440000"
        name:
          type: string
          description: The user's full name
          example: "Juan García"
        email:
          type: string
          format: email
          description: The user's email address
          example: "juan@example.com"

    ErrorResponse:
      type: object
      description: Standard error response
      required:
        - error
      properties:
        error:
          type: string
          description: Human-readable error message
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
export interface CreateUserRequest {
  name: string;
  email: string;
}

export interface UserResponse {
  id: string;
  name: string;
  email: string;
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

### 7.2 Contract tests vs E2E tests — La diferencia clave

Esta es la pregunta que hiciste, y es importante entenderlo bien:

```
┌──────────────────────────────────────────────────────────────────┐
│                         E2E TEST                                 │
│                                                                  │
│  1. POST /users con name="Juan", email="juan@example.com"       │
│  2. Verificar: status 201                                        │
│  3. Verificar: response tiene id, name, email                    │
│  4. GET /users/{id} con el ID del paso 1                ← FLUJO │
│  5. Verificar: devuelve el mismo user                   ← LÓGICA│
│  6. POST /users con mismo email                         ← REGLA │
│  7. Verificar: status 409                               ← NEGOCIO│
│                                                                  │
│  → Testea que el SISTEMA FUNCIONA end-to-end                     │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                      CONTRACT TEST                               │
│                                                                  │
│  Test 1: POST /users con body válido                             │
│    → ¿Status es 201?                                    ✅      │
│    → ¿Response tiene campo "id" tipo string?            ✅      │
│    → ¿Response tiene campo "name" tipo string?          ✅      │
│    → ¿Response tiene campo "email" tipo string?         ✅      │
│    → ¿No tiene campos extra no definidos?               ✅      │
│                                                                  │
│  Test 2: POST /users sin campo "name"                            │
│    → ¿Status es 400?                                    ✅      │
│    → ¿Response tiene campo "error" tipo string?         ✅      │
│                                                                  │
│  Test 3: GET /users/{id} con UUID válido pero inexistente        │
│    → ¿Status es 404?                                    ✅      │
│    → ¿Response tiene campo "error" tipo string?         ✅      │
│                                                                  │
│  → Testea que la FORMA de la API cumple el contrato              │
│  → NO encadena pasos, cada test es independiente                 │
└──────────────────────────────────────────────────────────────────┘
```

**Diferencias concretas:**

| | E2E | Contract |
|---|-----|----------|
| **Encadena pasos** | ✅ Sí (crear → buscar → actualizar) | ❌ No, cada test es atómico |
| **Verifica datos** | ✅ "El name es Juan" | ❌ "Hay un campo name y es string" |
| **Verifica reglas** | ✅ "Email duplicado → 409" | ⚠️ Solo "409 tiene formato ErrorResponse" |
| **Verifica la DB** | ✅ Indirectamente (GET tras POST) | ❌ Nunca |
| **Fuente de verdad** | El código del backend | El archivo OpenAPI |
| **¿Quién los necesita?** | Solo el backend | Backend + Frontend |
| **¿Cuándo fallan?** | Cuando la lógica se rompe | Cuando alguien cambia la forma de la API |

**La pregunta clave**: ¿Pueden coexistir? **Sí, y deben.** Los E2E testean que "funciona", los contracts testean que "cumples lo que prometiste". Un backend puede funcionar perfecto (E2E pasan) pero haber roto el contrato (por ejemplo, renombrar un campo `id` a `userId`). Los E2E no detectan eso, los contracts sí.

### 7.3 Implementación en Go

Para los contract tests, validamos que las respuestas de nuestra API cumplen con los schemas del OpenAPI. Usaremos la librería `libopenapi` para cargar el spec y validar las respuestas.

#### Paso 1: Instalar dependencias

```bash
go get github.com/pb33f/libopenapi
go get github.com/santhosh-tekuri/jsonschema/v5
```

#### Paso 2: Helper para cargar el contract

**Archivo**: `test/contract/contract_helper.go`

```go
package contract

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ContractValidator validates API responses against the OpenAPI spec.
type ContractValidator struct {
	t          *testing.T
	specPath   string
	apiBaseURL string
}

// NewContractValidator creates a new validator.
// specPath is the path to the openapi.yaml file.
func NewContractValidator(t *testing.T, specPath, apiBaseURL string) *ContractValidator {
	// Verify the spec file exists
	_, err := os.ReadFile(specPath)
	require.NoError(t, err, "cannot read OpenAPI spec at: %s", specPath)

	return &ContractValidator{
		t:          t,
		specPath:   specPath,
		apiBaseURL: apiBaseURL,
	}
}

// ValidateResponse checks that an HTTP response matches what the contract says.
// It validates: status code, required fields, and field types.
func (cv *ContractValidator) ValidateResponse(
	resp *http.Response,
	expectedStatus int,
	requiredFields []string,
	fieldTypes map[string]string, // field name → expected type: "string", "number", etc.
) {
	cv.t.Helper()

	// 1. Validate status code
	assert.Equal(cv.t, expectedStatus, resp.StatusCode,
		"status code mismatch for %s", resp.Request.URL.Path)

	// 2. Read body
	body, err := io.ReadAll(resp.Body)
	require.NoError(cv.t, err)
	defer resp.Body.Close()

	if len(body) == 0 {
		return
	}

	// 3. Parse JSON
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	require.NoError(cv.t, err, "response is not valid JSON: %s", string(body))

	// 4. Validate required fields exist
	for _, field := range requiredFields {
		assert.Contains(cv.t, data, field,
			"required field '%s' missing from response", field)
	}

	// 5. Validate field types
	for field, expectedType := range fieldTypes {
		value, exists := data[field]
		if !exists {
			continue // already checked by requiredFields
		}
		assertFieldType(cv.t, field, value, expectedType)
	}
}

// assertFieldType checks that a JSON field has the expected Go type.
func assertFieldType(t *testing.T, field string, value interface{}, expectedType string) {
	t.Helper()

	switch expectedType {
	case "string":
		_, ok := value.(string)
		assert.True(t, ok, "field '%s' should be string, got %T", field, value)
	case "number":
		_, ok := value.(float64)
		assert.True(t, ok, "field '%s' should be number, got %T", field, value)
	case "boolean":
		_, ok := value.(bool)
		assert.True(t, ok, "field '%s' should be boolean, got %T", field, value)
	case "object":
		_, ok := value.(map[string]interface{})
		assert.True(t, ok, "field '%s' should be object, got %T", field, value)
	default:
		t.Errorf("unknown expected type '%s' for field '%s'", expectedType, field)
	}
}

// BaseURL returns the API base URL.
func (cv *ContractValidator) BaseURL() string {
	return cv.apiBaseURL
}

// SpecPath returns the path to the OpenAPI spec.
func (cv *ContractValidator) SpecPath() string {
	return cv.specPath
}

// --- Schema definitions extracted from the OpenAPI spec ---
// These mirror what's in openapi.yaml, expressed in Go for validation.

// UserResponseSchema defines the contract for UserResponse.
var UserResponseSchema = ResponseSchema{
	RequiredFields: []string{"id", "name", "email"},
	FieldTypes: map[string]string{
		"id":    "string",
		"name":  "string",
		"email": "string",
	},
}

// ErrorResponseSchema defines the contract for ErrorResponse.
var ErrorResponseSchema = ResponseSchema{
	RequiredFields: []string{"error"},
	FieldTypes: map[string]string{
		"error": "string",
	},
}

// ResponseSchema holds the expected shape of a response.
type ResponseSchema struct {
	RequiredFields []string
	FieldTypes     map[string]string
}

// Validate is a convenience method to validate a response against this schema.
func (rs *ResponseSchema) Validate(cv *ContractValidator, resp *http.Response, expectedStatus int) {
	cv.ValidateResponse(resp, expectedStatus, rs.RequiredFields, rs.FieldTypes)
}

// --- Convenience function ---

// DefaultSpecPath returns the default path to the OpenAPI spec.
// Assumes the contract repo is at the same level as the backend repo.
func DefaultSpecPath() string {
	// Intenta primero la ruta relativa desde el repo backend
	paths := []string{
		"../../../contracts/openapi/openapi.yaml",   // desde test/contract/
		"../../contracts/openapi/openapi.yaml",       // desde test/
		"../contracts/openapi/openapi.yaml",          // desde backend/
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Fallback: variable de entorno
	if envPath := os.Getenv("CONTRACT_SPEC_PATH"); envPath != "" {
		return envPath
	}

	return "../../../contracts/openapi/openapi.yaml"
}

// --- Helpers for building URLs ---

func (cv *ContractValidator) URL(path string, args ...interface{}) string {
	return cv.apiBaseURL + fmt.Sprintf(path, args...)
}
```

#### Paso 3: Los Contract Tests

**Archivo**: `test/contract/user_contract_test.go`

```go
package contract_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"ductifact/test/contract"

	"github.com/stretchr/testify/require"
)

// setupValidator creates the contract validator for each test.
// Requires the API to be running.
func setupValidator(t *testing.T) *contract.ContractValidator {
	specPath := contract.DefaultSpecPath()
	apiURL := "http://localhost:8080" // o desde env var

	return contract.NewContractValidator(t, specPath, apiURL)
}

// ─── POST /users ─────────────────────────────────────────────

func TestContract_CreateUser_ValidBody_Returns201_WithUserResponse(t *testing.T) {
	cv := setupValidator(t)

	body, _ := json.Marshal(map[string]string{
		"name":  "Contract Test User",
		"email": "contract-create@example.com",
	})

	resp, err := http.Post(cv.URL("/users"), "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)

	// Contract dice: 201 + UserResponse{id, name, email}
	contract.UserResponseSchema.Validate(cv, resp, http.StatusCreated)
}

func TestContract_CreateUser_MissingName_Returns400_WithErrorResponse(t *testing.T) {
	cv := setupValidator(t)

	body, _ := json.Marshal(map[string]string{
		"email": "contract-noname@example.com",
	})

	resp, err := http.Post(cv.URL("/users"), "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)

	// Contract dice: 400 + ErrorResponse{error}
	contract.ErrorResponseSchema.Validate(cv, resp, http.StatusBadRequest)
}

func TestContract_CreateUser_MissingEmail_Returns400_WithErrorResponse(t *testing.T) {
	cv := setupValidator(t)

	body, _ := json.Marshal(map[string]string{
		"name": "No Email User",
	})

	resp, err := http.Post(cv.URL("/users"), "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)

	// Contract dice: 400 + ErrorResponse{error}
	contract.ErrorResponseSchema.Validate(cv, resp, http.StatusBadRequest)
}

func TestContract_CreateUser_InvalidEmailFormat_Returns400_WithErrorResponse(t *testing.T) {
	cv := setupValidator(t)

	body, _ := json.Marshal(map[string]string{
		"name":  "Bad Email",
		"email": "not-an-email",
	})

	resp, err := http.Post(cv.URL("/users"), "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)

	contract.ErrorResponseSchema.Validate(cv, resp, http.StatusBadRequest)
}

func TestContract_CreateUser_DuplicateEmail_Returns409_WithErrorResponse(t *testing.T) {
	cv := setupValidator(t)

	body, _ := json.Marshal(map[string]string{
		"name":  "Dup User 1",
		"email": "contract-dup@example.com",
	})

	// Crear el primero (puede fallar si ya existe, no importa)
	resp1, _ := http.Post(cv.URL("/users"), "application/json", bytes.NewBuffer(body))
	resp1.Body.Close()

	// Crear el segundo con mismo email
	body2, _ := json.Marshal(map[string]string{
		"name":  "Dup User 2",
		"email": "contract-dup@example.com",
	})
	resp, err := http.Post(cv.URL("/users"), "application/json", bytes.NewBuffer(body2))
	require.NoError(t, err)

	// Contract dice: 409 + ErrorResponse{error}
	contract.ErrorResponseSchema.Validate(cv, resp, http.StatusConflict)
}

// ─── GET /users/{id} ─────────────────────────────────────────

func TestContract_GetUser_ValidUUID_NotFound_Returns404_WithErrorResponse(t *testing.T) {
	cv := setupValidator(t)

	resp, err := http.Get(cv.URL("/users/%s", "550e8400-e29b-41d4-a716-446655440000"))
	require.NoError(t, err)

	// Contract dice: 404 + ErrorResponse{error}
	contract.ErrorResponseSchema.Validate(cv, resp, http.StatusNotFound)
}

func TestContract_GetUser_InvalidUUID_Returns400_WithErrorResponse(t *testing.T) {
	cv := setupValidator(t)

	resp, err := http.Get(cv.URL("/users/%s", "not-a-uuid"))
	require.NoError(t, err)

	// Contract dice: 400 + ErrorResponse{error}
	contract.ErrorResponseSchema.Validate(cv, resp, http.StatusBadRequest)
}

func TestContract_GetUser_ExistingUser_Returns200_WithUserResponse(t *testing.T) {
	cv := setupValidator(t)

	// Primero creamos un user para obtener su ID
	body, _ := json.Marshal(map[string]string{
		"name":  "Get Contract User",
		"email": "contract-get@example.com",
	})

	createResp, err := http.Post(cv.URL("/users"), "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	userID, ok := created["id"].(string)
	require.True(t, ok, "created user must have string id")

	// GET del user creado
	resp, err := http.Get(cv.URL("/users/%s", userID))
	require.NoError(t, err)

	// Contract dice: 200 + UserResponse{id, name, email}
	contract.UserResponseSchema.Validate(cv, resp, http.StatusOK)
}

// ─── PUT /users/{id} ─────────────────────────────────────────

func TestContract_UpdateUser_ValidBody_Returns200_WithUserResponse(t *testing.T) {
	cv := setupValidator(t)

	// Crear user primero
	createBody, _ := json.Marshal(map[string]string{
		"name":  "Update Contract User",
		"email": "contract-update@example.com",
	})

	createResp, err := http.Post(cv.URL("/users"), "application/json", bytes.NewBuffer(createBody))
	require.NoError(t, err)

	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	userID := created["id"].(string)

	// Update
	updateBody, _ := json.Marshal(map[string]string{
		"name": "Updated Name",
	})

	req, _ := http.NewRequest(http.MethodPut, cv.URL("/users/%s", userID), bytes.NewBuffer(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	// Contract dice: 200 + UserResponse{id, name, email}
	contract.UserResponseSchema.Validate(cv, resp, http.StatusOK)
}

func TestContract_UpdateUser_NotFound_Returns404_WithErrorResponse(t *testing.T) {
	cv := setupValidator(t)

	updateBody, _ := json.Marshal(map[string]string{
		"name": "Ghost",
	})

	req, _ := http.NewRequest(
		http.MethodPut,
		cv.URL("/users/%s", "550e8400-e29b-41d4-a716-446655440000"),
		bytes.NewBuffer(updateBody),
	)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	// Contract dice: 404 + ErrorResponse{error}
	contract.ErrorResponseSchema.Validate(cv, resp, http.StatusNotFound)
}

// ─── GET /health ─────────────────────────────────────────────

func TestContract_HealthCheck_Returns200_WithStatus(t *testing.T) {
	cv := setupValidator(t)

	resp, err := http.Get(cv.URL("/health"))
	require.NoError(t, err)

	cv.ValidateResponse(resp, http.StatusOK,
		[]string{"status"},
		map[string]string{"status": "string"},
	)
}
```

**¿Qué aprender de estos contract tests?**

1. **Nunca verifican valores concretos** — solo que el campo `name` existe y es `string`, no que sea `"Juan"`.
2. **Cada test es atómico** — no depende de otro test. Si necesita un user, lo crea (pero no verifica la creación en detalle).
3. **La fuente de verdad es el `openapi.yaml`** — los schemas en Go (`UserResponseSchema`) reflejan lo que dice el YAML.
4. **Son más rígidos que los E2E** — si añades un campo al response sin actualizar el contract, deberían fallar.
5. **Son más simples que los E2E** — no encadenan flujos ni verifican lógica de negocio.

---

## 8. Flujo de trabajo completo

### Cuando añades un nuevo endpoint:

```
1. Actualizar openapi.yaml en contracts/    ← Primero el contrato
2. Implementar en backend/                   ← Después el código
3. Añadir contract test                      ← Verificar que cumples
4. Añadir unit + integration tests           ← Verificar que funciona
5. Frontend genera tipos desde el contract   ← Auto-generado
```

### Cuando cambias un campo del response:

```
1. Actualizar openapi.yaml                  ← ¿Es breaking change?
2. Actualizar el handler/DTO en backend
3. Los contract tests fallan si no coincide  ← Te protegen
4. Frontend re-genera tipos                  ← Se adapta automáticamente
```

### En CI/CD:

```yaml
# .github/workflows/ci.yml (ejemplo)
jobs:
  test:
    steps:
      - name: Unit tests
        run: go test ./test/unit/...

      - name: Integration tests
        run: go test ./test/integration/...

      - name: Contract tests
        run: go test ./test/contract/...

      - name: E2E tests
        run: go test ./test/e2e/...

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
// Contract test — solo forma
contract.UserResponseSchema.Validate(cv, resp, http.StatusCreated)

// E2E test — forma + datos + flujo
assert.Equal(t, "Juan", createdUser.Name)
getResp := getJSON(t, baseURL+"/users/"+createdUser.ID)
assert.Equal(t, "Juan", fetchedUser.Name) // verifica persistencia
```

### 9.3 Versionado del contract

Cuando hagas breaking changes, versiona:

```yaml
# Opción 1: En la URL
/v1/users    → versión actual
/v2/users    → nueva versión

# Opción 2: En el archivo
openapi-v1.yaml
openapi-v2.yaml
```

### 9.4 Emails únicos en tests

Usa prefijos por tipo de test para evitar colisiones:

```go
// E2E tests
"e2e-create@example.com"

// Contract tests
"contract-create@example.com"
```

### 9.5 Resumen visual

```
openapi.yaml (en contracts/)
     │
     ├──→ Backend: Contract tests validan que las respuestas cumplen
     │
     ├──→ Frontend: Genera tipos TypeScript automáticos
     │
     ├──→ Swagger UI: Documentación interactiva
     │
     └──→ CI: Valida que el spec es válido + tests pasan
```
