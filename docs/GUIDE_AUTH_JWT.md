# Guía: Autenticación con JWT — De Cero a Implementación

> Esta guía explica **qué es la autenticación**, **cómo funciona JWT**, y te guía paso a paso para implementarla en Ductifact siguiendo la arquitectura hexagonal del proyecto.

---

## Índice

1. [Conceptos fundamentales](#1-conceptos-fundamentales)
2. [¿Qué es JWT?](#2-qué-es-jwt)
3. [Anatomía de un JWT](#3-anatomía-de-un-jwt)
4. [Flujo completo de autenticación](#4-flujo-completo-de-autenticación)
5. [Decisión de diseño: JWT propio](#5-decisión-de-diseño-jwt-propio)
6. [Paso 1 — Value Object: Password](#6-paso-1--value-object-password)
7. [Paso 2 — Añadir PasswordHash a User](#7-paso-2--añadir-passwordhash-a-user)
8. [Paso 3 — Servicio de Auth (dominio de aplicación)](#8-paso-3--servicio-de-auth-dominio-de-aplicación)
9. [Paso 4 — Generar y validar JWTs](#9-paso-4--generar-y-validar-jwts)
10. [Paso 5 — Endpoints de Register y Login](#10-paso-5--endpoints-de-register-y-login)
11. [Paso 6 — Middleware de autenticación](#11-paso-6--middleware-de-autenticación)
12. [Paso 7 — Proteger rutas](#12-paso-7--proteger-rutas)
13. [Paso 8 — Autorización (ownership)](#13-paso-8--autorización-ownership)
14. [Paso 9 — Base de datos](#14-paso-9--base-de-datos)
15. [Paso 10 — Wiring en main.go](#15-paso-10--wiring-en-maingo)
16. [Paso 11 — Tests](#16-paso-11--tests)
17. [Resumen de archivos](#17-resumen-de-archivos)
18. [Errores comunes y seguridad](#18-errores-comunes-y-seguridad)
19. [Glosario](#19-glosario)

---

## 1. Conceptos fundamentales

### Autenticación vs Autorización

Son dos conceptos diferentes que siempre van juntos:

| Concepto | Pregunta que responde | Ejemplo |
|----------|----------------------|---------|
| **Autenticación** (AuthN) | ¿Quién eres? | "Soy el usuario juan@example.com y mi contraseña es X" |
| **Autorización** (AuthZ) | ¿Qué puedes hacer? | "Puedes ver y editar TUS clientes, pero no los de otro usuario" |

**Orden**: Primero te autenticas (demuestras quién eres), y después el sistema te autoriza (decide qué te deja hacer).

### ¿Por qué necesitamos autenticación?

Ahora mismo, cualquier persona puede:
- Crear usuarios
- Ver los clientes de cualquier usuario
- Modificar datos de cualquier usuario

Eso es un desastre en producción. Con autenticación:
1. El usuario se registra (una vez).
2. El usuario hace login (recibe un "pase" que demuestra quién es).
3. En cada request, envía ese pase → el servidor sabe quién está pidiendo.
4. El servidor decide si le deja hacer lo que pide.

### Estrategias comunes de autenticación

| Estrategia | Cómo funciona | Stateless? | Complejidad |
|------------|--------------|-----------|-------------|
| **Sessions** | El servidor guarda un registro de quién está logueado (en memoria, Redis, o DB). El cliente recibe un `session_id` en una cookie. | ❌ Stateful | Baja |
| **JWT** | El servidor genera un token firmado con los datos del usuario. El cliente lo envía en cada request. El servidor valida la firma sin consultar la DB. | ✅ Stateless | Media |
| **OAuth2 / OIDC** | Delegas el login a un proveedor externo (Google, GitHub, Auth0). El proveedor te da un token. | ✅ Stateless | Alta |
| **API Keys** | El usuario recibe una key estática que envía en cada request. | ✅ Stateless | Baja (pero menos seguro) |

### ¿Por qué elegimos JWT?

1. **Stateless**: El servidor no guarda estado de sesión. Esto es ideal para APIs REST, que por definición son stateless.
2. **Escalable**: Si mañana tienes 3 servidores, todos pueden validar el mismo token sin compartir estado.
3. **Standard**: JWT es un estándar abierto (RFC 7519). Hay librerías en todos los lenguajes.
4. **Self-contained**: El token lleva la información del usuario dentro, así que no necesitas ir a la DB en cada request para saber quién es.
5. **Compatible con frontend**: Los SPAs (React, Vue, etc.) trabajan muy bien con JWT en el header `Authorization`.

---

## 2. ¿Qué es JWT?

**JWT = JSON Web Token**. Es un string codificado que contiene información (llamada **claims**) y está firmado digitalmente.

Ejemplo de un JWT real:
```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJhYmMtMTIzIiwiZW1haWwiOiJqdWFuQGV4LmNvbSIsImV4cCI6MTcwOTQ3MjAwMH0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
```

No es encriptado (cualquiera puede leer su contenido). Es **firmado** (nadie puede modificarlo sin invalidar la firma).

> 🔑 **Analogía**: Un JWT es como un carnet de identidad. Cualquiera puede leer tu nombre y foto, pero no puede falsificarlo porque tiene un sello oficial. Si alguien cambia los datos, el sello ya no coincide.

---

## 3. Anatomía de un JWT

Un JWT tiene tres partes separadas por puntos (`.`):

```
HEADER.PAYLOAD.SIGNATURE
```

### 3.1 Header

Dice qué algoritmo se usó para firmar:

```json
{
  "alg": "HS256",
  "typ": "JWT"
}
```

- `HS256` = HMAC con SHA-256. Es un algoritmo **simétrico**: usa la misma clave secreta para firmar y para verificar.
- Hay algoritmos asimétricos como `RS256` (clave pública/privada), pero `HS256` es más simple y suficiente para un solo servidor.

### 3.2 Payload (Claims)

Contiene los datos del usuario y metadatos del token:

```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "email": "juan@example.com",
  "iat": 1709380000,
  "exp": 1709466400
}
```

| Claim | Significado | Tipo |
|-------|------------|------|
| `sub` (subject) | ID del usuario. Es el claim más importante. | Registrado (estándar) |
| `email` | Email del usuario. Opcional pero útil. | Privado (custom) |
| `iat` (issued at) | Cuándo se generó el token (Unix timestamp). | Registrado |
| `exp` (expiration) | Cuándo expira el token (Unix timestamp). | Registrado |

**Claims registrados** son los definidos por el estándar RFC 7519. **Claims privados** son los que tú añades según tu necesidad.

> ⚠️ **NUNCA pongas información sensible en el payload** (contraseña, tarjeta de crédito, etc.). El payload está codificado en Base64, **no encriptado**. Cualquiera puede decodificarlo.

### 3.3 Signature (Firma)

Es lo que impide que alguien modifique el token:

```
HMAC-SHA256(
  base64UrlEncode(header) + "." + base64UrlEncode(payload),
  SECRET_KEY
)
```

El servidor tiene un `SECRET_KEY` que solo él conoce. Con esa clave:
1. **Firma** el token al generarlo (login).
2. **Verifica** el token cuando lo recibe (cada request).

Si alguien modifica el payload (ej: cambia el `sub` por el ID de otro usuario), la firma ya no coincide → el servidor rechaza el token.

### Diagrama visual

```
┌──────────────────────────────────────────────────────────────────────┐
│                           JWT Token                                  │
│                                                                      │
│  ┌──────────────┐  ┌───────────────────────┐  ┌──────────────────┐  │
│  │   HEADER     │  │      PAYLOAD          │  │   SIGNATURE      │  │
│  │              │  │                       │  │                  │  │
│  │ {            │  │ {                     │  │  HMAC-SHA256(    │  │
│  │  "alg":"HS256│  │  "sub":"user-uuid",  │  │    header +      │  │
│  │  "typ":"JWT" │  │  "email":"j@ex.com", │  │    payload,      │  │
│  │ }            │  │  "exp": 1709466400   │  │    SECRET_KEY    │  │
│  │              │  │ }                     │  │  )               │  │
│  └──────────────┘  └───────────────────────┘  └──────────────────┘  │
│       (Base64)            (Base64)                (Base64)           │
│                                                                      │
│  eyJhbGci...       eyJzdWIi...                SflKxwR...            │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 4. Flujo completo de autenticación

### 4.1 Registro (una vez)

```
Cliente                          Servidor
  │                                │
  │  POST /api/v1/auth/register    │
  │  { name, email, password }     │
  │ ──────────────────────────────►│
  │                                │ 1. Validar datos
  │                                │ 2. Hash de la password (bcrypt)
  │                                │ 3. Guardar user con hash en DB
  │                                │ 4. Generar JWT
  │  201 { user, token }           │
  │ ◄──────────────────────────────│
  │                                │
```

### 4.2 Login (cuando el token expira o la primera vez)

```
Cliente                          Servidor
  │                                │
  │  POST /api/v1/auth/login       │
  │  { email, password }           │
  │ ──────────────────────────────►│
  │                                │ 1. Buscar user por email
  │                                │ 2. Comparar password con hash
  │                                │ 3. Si coincide → generar JWT
  │  200 { user, token }           │
  │ ◄──────────────────────────────│
  │                                │
```

### 4.3 Request autenticado (cada llamada a la API)

```
Cliente                          Servidor
  │                                │
  │  GET /api/v1/users/me          │
  │  Authorization: Bearer <JWT>   │
  │ ──────────────────────────────►│
  │                                │ 1. Middleware extrae token
  │                                │ 2. Verifica firma + expiración
  │                                │ 3. Extrae userID del claim "sub"
  │                                │ 4. Pone userID en el context
  │                                │ 5. El handler usa userID del context
  │  200 { user data }             │
  │ ◄──────────────────────────────│
  │                                │
```

### 4.4 Request sin token o con token inválido

```
Cliente                          Servidor
  │                                │
  │  GET /api/v1/users/me          │
  │  (sin header Authorization)    │
  │ ──────────────────────────────►│
  │                                │ 1. Middleware no encuentra token
  │  401 { "error": "unauthorized" }│
  │ ◄──────────────────────────────│
  │                                │
```

---

## 5. Decisión de diseño: JWT propio

### ¿Por qué no usamos un proveedor externo (Auth0, Firebase Auth)?

| Factor | JWT propio | Proveedor externo |
|--------|-----------|-------------------|
| **Control** | Total | Limitado a lo que ofrece el proveedor |
| **Aprendizaje** | Entiendes todo el flujo | "Caja negra" que hace magia |
| **Complejidad** | Media (tú escribes el código) | Baja (SDK hace todo) |
| **Seguridad** | Tú eres responsable | Ellos son expertos (mejor en producción real) |
| **Costo** | $0 | Free tier limitado, luego pago |
| **Migración** | Fácil de reemplazar después | Lock-in con el proveedor |

Para un proyecto de aprendizaje, implementar JWT propio es lo correcto. Entenderás cada pieza. En producción real, considera usar Auth0/Firebase Auth para no reinventar la rueda.

### ¿Dónde vive el auth en la arquitectura hexagonal?

```
┌─────────────────────────────────────────────────────────────┐
│                        ADAPTERS                              │
│                                                              │
│  ┌─────────────────┐        ┌──────────────────────────┐    │
│  │  auth_handler.go │        │  middleware/auth.go       │    │
│  │  (inbound)       │        │  (inbound)                │    │
│  │                  │        │                           │    │
│  │  POST /register  │        │  Valida JWT en cada       │    │
│  │  POST /login     │        │  request protegido        │    │
│  └────────┬─────────┘        └─────────────┬─────────────┘    │
│           │                                │                  │
├───────────┼────────────────────────────────┼──────────────────┤
│           │       APPLICATION              │                  │
│           ▼                                │                  │
│  ┌─────────────────┐        ┌──────────────▼─────────────┐   │
│  │  auth_service.go │        │  jwt.go (token provider)   │   │
│  │  (port + impl)   │        │  (outbound port + adapter) │   │
│  │                  │        │                            │   │
│  │  Register()      │        │  GenerateToken()           │   │
│  │  Login()         │        │  ValidateToken()           │   │
│  └────────┬─────────┘        └────────────────────────────┘   │
│           │                                                   │
├───────────┼───────────────────────────────────────────────────┤
│           │       DOMAIN                                      │
│           ▼                                                   │
│  ┌─────────────────┐        ┌─────────────────────────┐      │
│  │  User entity     │        │  Password (Value Object) │      │
│  │  + PasswordHash  │        │  Validación + hashing    │      │
│  └─────────────────┘        └─────────────────────────┘      │
│                                                               │
└───────────────────────────────────────────────────────────────┘
```

**Decisiones clave:**
- El **hashing de passwords** vive en el dominio (Value Object), porque es una regla de negocio: "una password debe tener mínimo 8 caracteres y se almacena como hash".
- La **generación/validación de JWT** vive en la aplicación/infraestructura, porque JWT es un detalle de transporte (podríamos usar sessions mañana).
- El **middleware** vive en la infraestructura HTTP, porque es un adapter inbound.

---

## 6. Paso 1 — Value Object: Password

📁 `internal/domain/valueobjects/password.go`

La password tiene reglas de negocio: longitud mínima, no puede estar vacía, y debe almacenarse como hash (nunca en texto plano).

```go
package valueobjects

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrPasswordEmpty    = errors.New("password cannot be empty")
	ErrInvalidPassword  = errors.New("invalid password")
)

// Password is a value object that handles password validation and hashing.
// It stores the bcrypt hash, never the raw password.
type Password struct {
	hash string
}

// NewPassword validates the raw password and returns a Password with the bcrypt hash.
// The raw password is never stored.
func NewPassword(raw string) (*Password, error) {
	if raw == "" {
		return nil, ErrPasswordEmpty
	}
	if len(raw) < 8 {
		return nil, ErrPasswordTooShort
	}

	// bcrypt.GenerateFromPassword:
	// - Añade un salt aleatorio automáticamente (cada hash es diferente aunque la password sea igual)
	// - bcrypt.DefaultCost = 10 → 2^10 iteraciones. Más alto = más seguro pero más lento.
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return &Password{hash: string(hash)}, nil
}

// NewPasswordFromHash creates a Password from an already-hashed value.
// Used when loading from the database (the hash is already computed).
func NewPasswordFromHash(hash string) *Password {
	return &Password{hash: hash}
}

// Compare checks if the given raw password matches the stored hash.
// Returns nil on success, ErrInvalidPassword on failure.
func (p *Password) Compare(raw string) error {
	err := bcrypt.CompareHashAndPassword([]byte(p.hash), []byte(raw))
	if err != nil {
		return ErrInvalidPassword
	}
	return nil
}

// Hash returns the bcrypt hash string (for storage in DB).
func (p *Password) Hash() string {
	return p.hash
}
```

### 💡 ¿Qué es bcrypt y por qué no usamos SHA-256?

**bcrypt** es un algoritmo de hashing diseñado específicamente para passwords. Tiene tres propiedades que lo hacen ideal:

| Propiedad | bcrypt | SHA-256 |
|-----------|--------|---------|
| **Lento por diseño** | ✅ Sí (configurable con `cost`) | ❌ No, es rápido |
| **Salt incluido** | ✅ Automático | ❌ Debes añadirlo manualmente |
| **Resistente a ataques de fuerza bruta** | ✅ Sí (la lentitud lo impide) | ❌ Un atacante prueba millones/segundo |

¿Por qué importa que sea lento? Si alguien roba tu base de datos con los hashes:
- Con SHA-256: puede probar **10 mil millones** de passwords por segundo.
- Con bcrypt (cost=10): puede probar **~1,000** por segundo. Un ataque que con SHA-256 tarda 1 segundo, con bcrypt tarda **115 días**.

### 💡 ¿Por qué `bcrypt` en el dominio y no en infraestructura?

`golang.org/x/crypto/bcrypt` es una librería de la stdlib extendida de Go (el paquete `x/`). No es un framework, no es una dependencia de infraestructura. Es un algoritmo criptográfico, como `math/big` o `crypto/sha256`.

La validación "una password debe tener mínimo 8 caracteres" y "se almacena como hash" son **reglas de negocio**. Si cambias de bcrypt a argon2 mañana, solo cambias el Value Object, nada más.

Si prefieres ser más purista, podrías definir una interfaz `PasswordHasher` como outbound port y que bcrypt sea un adapter. Pero para este proyecto, es over-engineering.

### 💡 ¿Por qué NewPasswordFromHash?

Cuando lees un user de la DB, el password hash ya existe. No puedes pasar por `NewPassword` porque no tienes la password en texto plano (ni deberías). `NewPasswordFromHash` te permite reconstruir el Value Object desde el hash almacenado.

---

## 7. Paso 2 — Añadir PasswordHash a User

### 7.1 Modificar la entidad User

📁 `internal/domain/entities/user.go`

```go
type User struct {
	ID           uuid.UUID
	Name         string
	Email        string
	PasswordHash string    // ← NUEVO: almacena el hash bcrypt
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
```

### 7.2 Nuevo constructor para registro

No modificamos `NewUser` para no romper los tests existentes. Creamos `NewUserWithPassword`:

```go
// NewUserWithPassword creates a User with a hashed password (for registration).
func NewUserWithPassword(name, email, password string) (*User, error) {
	if name == "" {
		return nil, ErrEmptyUserName
	}

	validEmail, err := valueobjects.NewEmail(email)
	if err != nil {
		return nil, err
	}

	// Validate and hash the password via the value object
	pwd, err := valueobjects.NewPassword(password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &User{
		ID:           uuid.New(),
		Name:         name,
		Email:        validEmail.String(),
		PasswordHash: pwd.Hash(),
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}
```

### 💡 ¿Por qué no meter Password como campo tipo `Password` en la entidad?

Mismo razonamiento que con `Email`: almacenamos el valor primitivo (`string`) para simplificar mappers y serialización. La validación ya ocurrió en el constructor. A partir de ahí, el hash es un string confiable.

### 7.3 Actualizar el modelo y los mappers de DB

📁 `internal/infrastructure/adapters/outbound/persistence/postgres_user_repository.go`

Añadir `PasswordHash` al `UserModel` y a los mappers:

```go
type UserModel struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name         string    `gorm:"not null"`
	Email        string    `gorm:"uniqueIndex;not null"`
	PasswordHash string    `gorm:"column:password_hash;not null;default:''"` // ← NUEVO
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
```

Y actualizar los mappers `toUserModel` y `toUserEntity` para incluir `PasswordHash`.

### 7.4 Actualizar la tabla en la DB

📁 `init.sql`

```sql
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(255) NOT NULL,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL DEFAULT '',   -- ← NUEVO
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
```

> **Nota**: El `DEFAULT ''` permite que los usuarios existentes (creados sin password) sigan funcionando. En una migración real usarías una herramienta de migraciones, pero GORM's AutoMigrate lo manejará.

---

## 8. Paso 3 — Servicio de Auth (dominio de aplicación)

### 8.1 Puerto de entrada (interface)

📁 `internal/application/usecases/auth_service.go`

```go
package usecases

import (
	"context"

	"ductifact/internal/domain/entities"
)

// AuthService is the inbound port for authentication operations.
type AuthService interface {
	Register(ctx context.Context, name, email, password string) (*entities.User, string, error)
	Login(ctx context.Context, email, password string) (*entities.User, string, error)
}
```

- `Register` devuelve el user creado + el JWT token (para que el usuario quede logueado automáticamente tras registrarse).
- `Login` devuelve el user + el JWT token.

### 8.2 Puerto de salida para tokens (interface)

📁 `internal/application/ports/token_provider.go`

```go
package ports

import "github.com/google/uuid"

// TokenProvider is the outbound port for JWT operations.
// It is defined as an interface so the auth service doesn't depend on
// a specific JWT library — the implementation lives in infrastructure.
type TokenProvider interface {
	GenerateToken(userID uuid.UUID, email string) (string, error)
	ValidateToken(tokenString string) (*TokenClaims, error)
}

// TokenClaims holds the data extracted from a valid token.
type TokenClaims struct {
	UserID uuid.UUID
	Email  string
}
```

### 💡 ¿Por qué TokenProvider es una interface y no una función directa?

Porque así el `AuthService` y el middleware de auth no dependen de `golang-jwt/jwt`. Si mañana quieres cambiar la librería de JWT, o usar tokens opacos, o pasar a PASETO, solo cambias el adapter que implementa `TokenProvider`.

Esto sigue el patrón de la arquitectura hexagonal: **el dominio/aplicación define la interface (port), la infraestructura la implementa (adapter)**. Nota: esto aplica a los **outbound ports** (como `TokenProvider` y `UserRepository`), donde la aplicación define lo que necesita y el adaptador lo implementa. Los adaptadores inbound (como los handlers HTTP) funcionan al revés: **consumen** la interfaz del caso de uso, no la implementan.

### 8.3 Implementación del servicio

📁 `internal/application/services/auth_service.go`

```go
package services

import (
	"context"
	"errors"

	"ductifact/internal/application/ports"
	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/repositories"
	"ductifact/internal/domain/valueobjects"
)

// --- Application-level errors ---

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
)

// authService implements usecases.AuthService.
type authService struct {
	userRepo      repositories.UserRepository
	tokenProvider ports.TokenProvider
}

// NewAuthService creates a new AuthService.
func NewAuthService(userRepo repositories.UserRepository, tokenProvider ports.TokenProvider) *authService {
	return &authService{
		userRepo:      userRepo,
		tokenProvider: tokenProvider,
	}
}

// Register creates a new user with a hashed password and returns a JWT.
func (s *authService) Register(ctx context.Context, name, email, password string) (*entities.User, string, error) {
	// Step 1: Check if email is already taken
	existing, _ := s.userRepo.GetByEmail(ctx, email)
	if existing != nil {
		return nil, "", ErrEmailTaken
	}

	// Step 2: Create user entity (validates name + email + password, hashes password)
	user, err := entities.NewUserWithPassword(name, email, password)
	if err != nil {
		return nil, "", err
	}

	// Step 3: Persist
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, "", err
	}

	// Step 4: Generate JWT so the user is logged in immediately
	token, err := s.tokenProvider.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// Login verifies credentials and returns a JWT.
func (s *authService) Login(ctx context.Context, email, password string) (*entities.User, string, error) {
	// Step 1: Find user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal whether the email exists or not (security)
		return nil, "", ErrInvalidCredentials
	}

	// Step 2: Compare password with stored hash
	pwd := valueobjects.NewPasswordFromHash(user.PasswordHash)
	if err := pwd.Compare(password); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	// Step 3: Generate JWT
	token, err := s.tokenProvider.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}
```

### 💡 ¿Por qué el error de login es siempre "invalid email or password"?

**NUNCA** digas "este email no existe" o "la contraseña es incorrecta" por separado. Eso le dice a un atacante:
- "El email no existe" → el atacante sabe que no vale la pena y prueba otro.
- "La contraseña es incorrecta" → el atacante sabe que el email **sí** existe y hace fuerza bruta sobre la password.

Con un error genérico ("invalid email or password"), el atacante no sabe cuál de los dos está mal.

---

## 9. Paso 4 — Generar y validar JWTs

### 9.1 Implementación del TokenProvider

📁 `internal/infrastructure/auth/jwt_provider.go`

Esta es la implementación concreta del outbound port `TokenProvider`. Usa la librería `golang-jwt/jwt/v5`.

```go
package auth

import (
	"errors"
	"os"
	"time"

	"ductifact/internal/application/ports"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
)

// JWTProvider implements ports.TokenProvider using golang-jwt.
type JWTProvider struct {
	secretKey     []byte
	tokenDuration time.Duration
}

// NewJWTProvider creates a new JWTProvider.
// The secret key is read from the environment variable JWT_SECRET.
func NewJWTProvider() *JWTProvider {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// In production, this should be a fatal error.
		// For development, use a default (NEVER do this in production).
		secret = "dev-secret-change-me-in-production"
	}

	return &JWTProvider{
		secretKey:     []byte(secret),
		tokenDuration: 24 * time.Hour, // Token expires in 24 hours
	}
}

// jwtClaims extends jwt.RegisteredClaims with custom fields.
type jwtClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateToken creates a signed JWT for the given user.
func (p *JWTProvider) GenerateToken(userID uuid.UUID, email string) (string, error) {
	now := time.Now()

	claims := jwtClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),                          // "sub" claim = user ID
			IssuedAt:  jwt.NewNumericDate(now),                  // "iat" claim
			ExpiresAt: jwt.NewNumericDate(now.Add(p.tokenDuration)), // "exp" claim
			Issuer:    "ductifact",                              // "iss" claim
		},
	}

	// Create token with HS256 signing method
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret key
	return token.SignedString(p.secretKey)
}

// ValidateToken parses and validates a JWT, returning the claims.
func (p *JWTProvider) ValidateToken(tokenString string) (*ports.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method is what we expect (prevents algorithm switching attacks)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return p.secretKey, nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, ErrInvalidToken
	}

	return &ports.TokenClaims{
		UserID: userID,
		Email:  claims.Email,
	}, nil
}
```

### 💡 ¿Qué es el algorithm switching attack?

Si no verificas que el algoritmo sea `HS256`, un atacante podría enviar un token con `"alg": "none"` (sin firma) y el servidor lo aceptaría. La línea que verifica `*jwt.SigningMethodHMAC` previene esto.

### 💡 ¿Por qué el secreto viene de una variable de entorno?

El `JWT_SECRET` es la clave más importante de tu aplicación. Quien la tenga puede generar tokens válidos para cualquier usuario. **NUNCA** debe estar en el código fuente ni en git.

Añade a tu `.env`:
```
JWT_SECRET=una-clave-secreta-larga-y-aleatoria-de-al-menos-32-caracteres
```

### Dependencia a instalar

```bash
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto
```

---

## 10. Paso 5 — Endpoints de Register y Login

### 10.1 Auth Handler

📁 `internal/infrastructure/adapters/inbound/http/auth_handler.go`

```go
package http

import (
	"errors"
	"net/http"

	"ductifact/internal/application/services"
	"ductifact/internal/application/usecases"
	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/valueobjects"

	"github.com/gin-gonic/gin"
)

// --- DTOs ---

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	User  UserResponse `json:"user"`
	Token string       `json:"token"`
}

// --- Handler ---

type AuthHandler struct {
	authService usecases.AuthService
}

func NewAuthHandler(authService usecases.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, token, err := h.authService.Register(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, services.ErrEmailTaken):
			status = http.StatusConflict
		case errors.Is(err, entities.ErrEmptyUserName):
			status = http.StatusBadRequest
		case errors.Is(err, valueobjects.ErrPasswordTooShort),
			errors.Is(err, valueobjects.ErrPasswordEmpty),
			errors.Is(err, valueobjects.ErrInvalidEmail):
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{
		User:  *toUserResponse(user),
		Token: token,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, token, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		User:  *toUserResponse(user),
		Token: token,
	})
}
```

### 💡 ¿Por qué el Register devuelve 201 y el Login devuelve 200?

- `201 Created`: Se creó un recurso nuevo (el usuario).
- `200 OK`: No se creó nada, solo se validaron credenciales y se devolvió información.

### 💡 ¿Por qué la validación `min=8` está en el DTO Y en el Value Object?

Doble validación:
1. **DTO** (`binding:"min=8"`): Fail fast en la capa HTTP. Si el password es corto, ni siquiera llega al servicio. Devuelve un error de formato Gin limpio.
2. **Value Object** (`NewPassword`): Protección del dominio. Aunque alguien llame al servicio sin pasar por el handler (tests, CLI, gRPC), la validación sigue activa.

---

## 11. Paso 6 — Middleware de autenticación

### 11.1 ¿Qué es un middleware?

Un middleware es una función que intercepta **cada request** antes de que llegue al handler. Es como un guardia de seguridad en la puerta:

```
Request → [Middleware Auth] → [Handler] → Response
                 │
                 ├─ Token válido? → Pasa al handler con userID en el context
                 └─ Token inválido? → 401 Unauthorized (no llega al handler)
```

### 11.2 Implementación

📁 `internal/infrastructure/adapters/inbound/http/middleware/auth.go`

```go
package middleware

import (
	"net/http"
	"strings"

	"ductifact/internal/application/ports"

	"github.com/gin-gonic/gin"
)

// contextKey es un tipo privado para evitar colisiones en el context.
// Si usaras un string como key, otro paquete podría sobrescribirlo accidentalmente.
type contextKey string

const UserIDKey contextKey = "userID"

// AuthMiddleware creates a Gin middleware that validates JWT tokens.
// It extracts the token from the Authorization header, validates it,
// and puts the userID in the request context.
func AuthMiddleware(tokenProvider ports.TokenProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Step 1: Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header required",
			})
			return
		}

		// Step 2: Extract the token (format: "Bearer <token>")
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header must be: Bearer <token>",
			})
			return
		}

		tokenString := parts[1]

		// Step 3: Validate the token
		claims, err := tokenProvider.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			return
		}

		// Step 4: Put the userID in Gin's context (available to all handlers)
		c.Set(string(UserIDKey), claims.UserID)

		// Step 5: Continue to the next handler
		c.Next()
	}
}
```

### 💡 ¿Qué es `c.Set` y `c.Get`?

Gin tiene su propio mecanismo de context. `c.Set("key", value)` almacena un valor que cualquier handler posterior puede leer con `c.Get("key")` o `c.MustGet("key")`.

Esto es cómo el middleware "pasa" el `userID` al handler: lo mete en el context, y el handler lo saca.

### 11.3 Helper para extraer el userID del context

📁 `internal/infrastructure/adapters/inbound/http/middleware/context.go`

```go
package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var ErrUserIDNotInContext = errors.New("user ID not found in context")

// GetUserIDFromContext extracts the authenticated user's ID from the Gin context.
// This should only be called in handlers behind the AuthMiddleware.
func GetUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	value, exists := c.Get(string(UserIDKey))
	if !exists {
		return uuid.Nil, ErrUserIDNotInContext
	}

	userID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil, ErrUserIDNotInContext
	}

	return userID, nil
}
```

---

## 12. Paso 7 — Proteger rutas

📁 `internal/infrastructure/adapters/inbound/http/router.go`

Ahora dividimos las rutas en **públicas** (sin auth) y **protegidas** (con auth):

```go
func SetupRoutes(
	userService usecases.UserService,
	clientService usecases.ClientService,
	authService usecases.AuthService,
	tokenProvider ports.TokenProvider,
) *gin.Engine {
	r := gin.Default()

	v1 := r.Group("/api/v1")

	// --- Public routes (no auth required) ---

	v1.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// Auth routes
	authHandler := NewAuthHandler(authService)
	authRoutes := v1.Group("/auth")
	{
		authRoutes.POST("/register", authHandler.Register)
		authRoutes.POST("/login", authHandler.Login)
	}

	// --- Protected routes (auth required) ---

	protected := v1.Group("")
	protected.Use(middleware.AuthMiddleware(tokenProvider))

	// User routes
	userHandler := NewUserHandler(userService)
	userRoutes := protected.Group("/users")
	{
		userRoutes.GET("/me", userHandler.GetMe)       // ← NUEVO: obtener usuario del token
		userRoutes.PUT("/me", userHandler.UpdateMe)     // ← NUEVO: actualizar usuario del token
	}

	// Client routes (nested under users)
	clientHandler := NewClientHandler(clientService)
	clientRoutes := protected.Group("/users/me/clients") // ← Ahora siempre son "mis" clientes
	{
		clientRoutes.POST("", clientHandler.CreateClient)
		clientRoutes.GET("", clientHandler.ListClients)
		clientRoutes.GET("/:client_id", clientHandler.GetClient)
		clientRoutes.PUT("/:client_id", clientHandler.UpdateClient)
		clientRoutes.DELETE("/:client_id", clientHandler.DeleteClient)
	}

	return r
}
```

### 💡 Cambio importante: de `/users/:user_id/` a `/users/me/`

Antes, el `user_id` venía en la URL. Ahora viene del **token JWT**. Esto tiene dos ventajas:

1. **Seguridad**: Un usuario no puede poner el ID de otro usuario en la URL.
2. **Simplicidad**: El frontend no necesita saber su propio `user_id`. Simplemente usa `/users/me`.

El handler saca el `userID` del context (puesto por el middleware), no de la URL.

### 💡 ¿Qué pasa con los handlers existentes?

Los handlers de `User` y `Client` necesitan adaptarse para tomar el `userID` del context en vez de la URL. Por ejemplo:

**Antes** (en `client_handler.go`):
```go
func (h *ClientHandler) CreateClient(c *gin.Context) {
    userID, err := uuid.Parse(c.Param("user_id"))  // ← De la URL
    // ...
}
```

**Después:**
```go
func (h *ClientHandler) CreateClient(c *gin.Context) {
    userID, err := middleware.GetUserIDFromContext(c)  // ← Del token JWT
    // ...
}
```

---

## 13. Paso 8 — Autorización (ownership)

Con autenticación, ya sabemos **quién** hace la request. La autorización decide **qué** puede hacer.

### 13.1 Reglas de autorización del proyecto

| Recurso | Regla |
|---------|-------|
| User (su perfil) | Solo puede ver/editar **su propio** perfil (el del token) |
| Clients | Solo puede CRUD **sus propios** clientes |

### 13.2 ¿Cómo se implementa?

La autorización ya está en tu código, solo cambia de dónde viene el `userID`:

**Antes**: El `userID` venía de la URL (`c.Param("user_id")`) y confiabas en que el cliente enviaba su propio ID.

**Ahora**: El `userID` viene del token JWT (que está firmado, no se puede falsificar). La autorización es automática:

```go
// El userID viene del token → es SEGURO
userID, _ := middleware.GetUserIDFromContext(c)

// El clientService ya verifica ownership:
// if client.UserID != userID → ErrClientNotOwned → 403
client, err := h.clientService.GetClientByID(ctx, clientID, userID)
```

### 13.3 Ejemplo: GetMe handler

```go
func (h *UserHandler) GetMe(c *gin.Context) {
    // El userID viene del middleware (del token JWT)
    userID, err := middleware.GetUserIDFromContext(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }

    user, err := h.userService.GetUserByID(c.Request.Context(), userID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, toUserResponse(user))
}
```

No hay riesgo de que un usuario vea los datos de otro: el `userID` **solo puede venir del token válido**.

---

## 14. Paso 9 — Base de datos

### 14.1 Migración de la tabla users

📁 `init.sql` (actualizado)

```sql
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(255) NOT NULL,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
```

### 14.2 GORM AutoMigrate

GORM debería detectar la nueva columna `password_hash` y añadirla automáticamente. Si no, puedes ejecutar manualmente:

```sql
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255) NOT NULL DEFAULT '';
```

---

## 15. Paso 10 — Wiring en main.go

📁 `cmd/api/main.go`

```go
func main() {
	_ = godotenv.Load()

	// Database
	db, err := persistence.NewPostgresConnection()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// --- Token provider ---
	tokenProvider := auth.NewJWTProvider()

	// --- User wiring ---
	userRepo := persistence.NewPostgresUserRepository(db)
	userService := services.NewUserService(userRepo)

	// --- Client wiring ---
	clientRepo := persistence.NewPostgresClientRepository(db)
	clientService := services.NewClientService(clientRepo, userRepo)

	// --- Auth wiring ---
	authService := services.NewAuthService(userRepo, tokenProvider)

	// --- HTTP ---
	router := httpAdapter.SetupRoutes(userService, clientService, authService, tokenProvider)

	// ...
}
```

El `authService` recibe el `userRepo` (para buscar usuarios por email) y el `tokenProvider` (para generar JWTs). El `tokenProvider` también se pasa al router para que el middleware pueda validar tokens.

---

## 16. Paso 11 — Tests

### 16.1 Unit Tests del Value Object Password

📁 `test/unit/domain/valueobjects/password_test.go`

```go
func TestNewPassword_ValidPassword(t *testing.T) {
    pwd, err := valueobjects.NewPassword("securepass123")
    assert.NoError(t, err)
    assert.NotEmpty(t, pwd.Hash())
}

func TestNewPassword_TooShort(t *testing.T) {
    _, err := valueobjects.NewPassword("short")
    assert.ErrorIs(t, err, valueobjects.ErrPasswordTooShort)
}

func TestNewPassword_Empty(t *testing.T) {
    _, err := valueobjects.NewPassword("")
    assert.ErrorIs(t, err, valueobjects.ErrPasswordEmpty)
}

func TestPassword_Compare_Success(t *testing.T) {
    pwd, _ := valueobjects.NewPassword("securepass123")
    err := pwd.Compare("securepass123")
    assert.NoError(t, err)
}

func TestPassword_Compare_WrongPassword(t *testing.T) {
    pwd, _ := valueobjects.NewPassword("securepass123")
    err := pwd.Compare("wrongpassword")
    assert.ErrorIs(t, err, valueobjects.ErrInvalidPassword)
}

func TestNewPassword_DifferentHashesForSamePassword(t *testing.T) {
    pwd1, _ := valueobjects.NewPassword("securepass123")
    pwd2, _ := valueobjects.NewPassword("securepass123")
    // bcrypt includes a random salt, so hashes should differ
    assert.NotEqual(t, pwd1.Hash(), pwd2.Hash())
}
```

### 16.2 Unit Tests del AuthService

📁 `test/unit/application/services/auth_service_test.go`

Necesitas mocks para `UserRepository` (ya lo tienes) y para `TokenProvider` (nuevo):

📁 `test/unit/mocks/mock_token_provider.go`

```go
package mocks

import "github.com/google/uuid"

type MockTokenProvider struct {
    GenerateTokenFn func(userID uuid.UUID, email string) (string, error)
    ValidateTokenFn func(tokenString string) (*ports.TokenClaims, error)
}

func (m *MockTokenProvider) GenerateToken(userID uuid.UUID, email string) (string, error) {
    return m.GenerateTokenFn(userID, email)
}

func (m *MockTokenProvider) ValidateToken(tokenString string) (*ports.TokenClaims, error) {
    return m.ValidateTokenFn(tokenString)
}
```

Tests del servicio:

```go
func TestRegister_HappyPath(t *testing.T) {
    // Mock repo: no existing user, create succeeds
    // Mock token: returns "mock-token"
    // Assert: user created, token returned, password hash stored
}

func TestRegister_EmailAlreadyTaken(t *testing.T) {
    // Mock repo: GetByEmail returns existing user
    // Assert: ErrEmailTaken
}

func TestLogin_HappyPath(t *testing.T) {
    // Mock repo: returns user with valid hash
    // Assert: user returned, token returned
}

func TestLogin_WrongPassword(t *testing.T) {
    // Mock repo: returns user with hash
    // Compare fails
    // Assert: ErrInvalidCredentials
}

func TestLogin_EmailNotFound(t *testing.T) {
    // Mock repo: GetByEmail returns error
    // Assert: ErrInvalidCredentials (same error, no info leak)
}
```

### 16.3 Integration Tests

📁 `test/integration/auth/auth_test.go`

```go
func TestRegisterAndLogin(t *testing.T) {
    // 1. Register a user with name, email, password
    // 2. Assert: user created in DB with password_hash (not empty)
    // 3. Login with same credentials
    // 4. Assert: token returned, token is valid
}

func TestRegisterDuplicateEmail(t *testing.T) {
    // 1. Register user A
    // 2. Try to register user B with same email
    // 3. Assert: ErrEmailTaken
}
```

### 16.4 E2E Tests

📁 `test/e2e/auth_test.go`

```go
func TestE2E_Register_201(t *testing.T) {
    // POST /api/v1/auth/register { name, email, password }
    // Assert: 201, body has user + token
}

func TestE2E_Register_DuplicateEmail_409(t *testing.T) {
    // Register once → 201
    // Register again with same email → 409
}

func TestE2E_Login_200(t *testing.T) {
    // Register → Login with same credentials → 200, body has token
}

func TestE2E_Login_WrongPassword_401(t *testing.T) {
    // Register → Login with wrong password → 401
}

func TestE2E_ProtectedRoute_WithToken_200(t *testing.T) {
    // Register → use token → GET /api/v1/users/me → 200
}

func TestE2E_ProtectedRoute_NoToken_401(t *testing.T) {
    // GET /api/v1/users/me sin token → 401
}

func TestE2E_ProtectedRoute_InvalidToken_401(t *testing.T) {
    // GET /api/v1/users/me con token inventado → 401
}
```

---

## 17. Resumen de archivos

### Archivos nuevos

| Capa | Archivo | Propósito |
|------|---------|-----------|
| Domain | `internal/domain/valueobjects/password.go` | Value Object: validación + hashing |
| Application | `internal/application/usecases/auth_service.go` | Use case: interface del auth service |
| Application | `internal/application/ports/token_provider.go` | Outbound port: interface para generar/validar tokens (technical port) |
| Application | `internal/application/services/auth_service.go` | Implementación del servicio de auth |
| Infrastructure | `internal/infrastructure/auth/jwt_provider.go` | Adapter: implementación JWT del TokenProvider |
| Infrastructure | `internal/infrastructure/adapters/inbound/http/auth_handler.go` | Handler HTTP: register + login |
| Infrastructure | `internal/infrastructure/adapters/inbound/http/middleware/auth.go` | Middleware: valida JWT en requests protegidos |
| Infrastructure | `internal/infrastructure/adapters/inbound/http/middleware/context.go` | Helper: extraer userID del Gin context |
| Tests | `test/unit/domain/valueobjects/password_test.go` | Tests del Value Object |
| Tests | `test/unit/mocks/mock_token_provider.go` | Mock del TokenProvider |
| Tests | `test/unit/application/services/auth_service_test.go` | Tests del AuthService |
| Tests | `test/e2e/auth_test.go` | Tests E2E de auth |

### Archivos modificados

| Archivo | Cambio |
|---------|--------|
| `internal/domain/entities/user.go` | Añadir `PasswordHash` + `NewUserWithPassword` |
| `internal/infrastructure/adapters/outbound/persistence/postgres_user_repository.go` | Añadir `PasswordHash` al model y mappers |
| `internal/infrastructure/adapters/inbound/http/router.go` | Rutas públicas vs protegidas, nuevo parámetro `authService` |
| `internal/infrastructure/adapters/inbound/http/user_handler.go` | Nuevos handlers `GetMe`/`UpdateMe` |
| `internal/infrastructure/adapters/inbound/http/client_handler.go` | Sacar `userID` del context en vez de la URL |
| `cmd/api/main.go` | Wiring del tokenProvider y authService |
| `init.sql` | Columna `password_hash` en users |
| `go.mod` | Dependencias `golang-jwt/jwt/v5` + `golang.org/x/crypto` |
| `.env` | Variable `JWT_SECRET` |

---

## 18. Errores comunes y seguridad

### ❌ Almacenar passwords en texto plano
**Nunca**. Siempre bcrypt. Si te hackean la DB, los hashes no revelan las passwords.

### ❌ Usar un JWT_SECRET corto o predecible
Usa al menos 32 caracteres aleatorios. Genera uno con:
```bash
openssl rand -base64 32
```

### ❌ No validar la expiración del token
`golang-jwt` valida `exp` automáticamente. Pero asegúrate de que el claim `exp` existe en tus tokens.

### ❌ Poner el token en el body o en query params
El token va **siempre** en el header `Authorization: Bearer <token>`. Las URLs se loguean, los query params se guardan en el historial del navegador.

### ❌ Tokens que nunca expiran
24h es un buen punto de partida. En producción puedes implementar **refresh tokens** (un segundo token de larga duración que te permite obtener nuevos access tokens sin hacer login de nuevo).

### ❌ Revelar si un email existe en el login
El error siempre debe ser genérico: "invalid email or password". Nunca "user not found" vs "wrong password".

### ❌ No verificar el algoritmo en ValidateToken
Siempre verifica que el `alg` del header sea el esperado. Esto previene el algorithm switching attack.

---

## 19. Glosario

| Término | Definición |
|---------|-----------|
| **JWT** | JSON Web Token. Token firmado que contiene claims sobre el usuario. |
| **Claim** | Un par key-value dentro del payload del JWT. Ejemplo: `"sub": "user-123"`. |
| **Bearer Token** | Esquema de autenticación HTTP. El token se envía en el header `Authorization: Bearer <token>`. |
| **bcrypt** | Algoritmo de hashing diseñado para passwords. Lento por diseño. |
| **Salt** | Valor aleatorio añadido al password antes de hashear. Previene ataques con tablas rainbow. bcrypt lo incluye automáticamente. |
| **Hash** | Función one-way: puedes convertir "password123" en "$2a$10$...", pero no al revés. |
| **HS256** | HMAC-SHA256. Algoritmo simétrico para firmar JWTs. Una clave para firmar y verificar. |
| **Middleware** | Función que intercepta requests antes de llegar al handler. Se usa para auth, logging, CORS, etc. |
| **Stateless** | El servidor no guarda estado entre requests. Cada request es independiente. |
| **Refresh Token** | Token de larga duración para obtener nuevos access tokens sin hacer login. No implementado en esta fase. |

---

## Orden de implementación sugerido

```
1. ██░░░░░░░░  go get dependencias (jwt, bcrypt)
2. ████░░░░░░  Value Object: Password
3. ██████░░░░  Modificar User entity + model + DB
4. ████████░░  TokenProvider (interface + implementación JWT)
5. ██████████  AuthService (interface + implementación)
6. ██████████  AuthHandler (register + login)
7. ██████████  Middleware de auth
8. ██████████  Proteger rutas en router.go
9. ██████████  Adaptar handlers existentes (userID del context)
10.██████████  Tests (unit → integration → e2e)
```

> **Regla de oro**: Después de cada paso, ejecuta los tests existentes (`make test`) para asegurarte de que no rompiste nada. Si rompen, arréglalo antes de seguir.
