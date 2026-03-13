# Guía: Mejoras de Infraestructura HTTP — De Cero a Implementación

> Esta guía explica **qué son los middlewares**, **por qué los necesitas**, y te guía paso a paso para implementar cada uno en Ductifact siguiendo la arquitectura hexagonal del proyecto.

---

## Índice

1. [¿Qué es un Middleware?](#1-qué-es-un-middleware)
2. [¿Por qué necesitamos estos middlewares?](#2-por-qué-necesitamos-estos-middlewares)
3. [¿Qué hace `gin.Default()` por ti?](#3-qué-hace-gindefault-por-ti)
4. [Paso 1 — Middleware de Request ID](#4-paso-1--middleware-de-request-id)
5. [Paso 2 — Middleware de Logging](#5-paso-2--middleware-de-logging)
6. [Paso 3 — Middleware de Recovery](#6-paso-3--middleware-de-recovery)
7. [Paso 4 — Middleware de CORS](#7-paso-4--middleware-de-cors)
8. [Paso 5 — Manejo de Errores Centralizado](#8-paso-5--manejo-de-errores-centralizado)
9. [Paso 6 — Integrar todo en el Router](#9-paso-6--integrar-todo-en-el-router)
10. [Paso 7 — Tests](#10-paso-7--tests)
11. [Resumen de archivos](#11-resumen-de-archivos)
12. [Errores comunes](#12-errores-comunes)
13. [Glosario](#13-glosario)

---

## 1. ¿Qué es un Middleware?

Un **middleware** es una función que se ejecuta **antes y/o después** de cada request HTTP. Piensa en él como un **filtro** o **interceptor** que envuelve a tus handlers.

### Analogía

Imagina que tu API es un edificio de oficinas:

- Los **handlers** son las oficinas donde se hace el trabajo real.
- Los **middlewares** son los controles de seguridad, recepcionistas y cámaras en la entrada.

Cada request (visitante) pasa por todos los controles antes de llegar a la oficina, y también al salir:

```
Request entrante
    │
    ▼
┌─────────────────┐
│  Request ID     │  ← Asigna un ID único al visitante
├─────────────────┤
│  Logger         │  ← Registra quién entra y a qué hora
├─────────────────┤
│  Recovery       │  ← Red de seguridad: si algo explota, no se cae el edificio
├─────────────────┤
│  CORS           │  ← Control de acceso: ¿desde qué origen vienes?
├─────────────────┤
│  Auth           │  ← (Ya lo tienes) Verifica tu identidad con JWT
├─────────────────┤
│  Handler        │  ← La oficina: procesa la petición real
└─────────────────┘
    │
    ▼
Response saliente
```

### ¿Cómo funciona un middleware en Gin?

En Gin, un middleware es simplemente una función con firma `gin.HandlerFunc`:

```go
func MyMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // ANTES del handler: aquí preparas cosas (medir tiempo, poner headers, etc.)

        c.Next() // ← Ejecuta el siguiente middleware o el handler final

        // DESPUÉS del handler: aquí reaccionas al resultado (loguear status, etc.)
    }
}
```

La llamada a `c.Next()` es clave. Divide el middleware en dos partes:
1. **Antes de `c.Next()`**: Se ejecuta ANTES de que el request llegue al handler.
2. **Después de `c.Next()`**: Se ejecuta DESPUÉS de que el handler ya respondió.

Esto es un patrón conocido como **cadena de responsabilidad** (chain of responsibility).

### c.Next() vs c.Abort()

| Método | Qué hace | Cuándo usarlo |
|--------|---------|---------------|
| `c.Next()` | Continúa al siguiente middleware/handler | Cuando todo está bien |
| `c.Abort()` | Detiene la cadena, no ejecuta más handlers | Cuando hay un error (ej: token inválido) |
| `c.AbortWithStatusJSON()` | Abort + manda una respuesta JSON de error | Lo que usas en `AuthMiddleware` |

---

## 2. ¿Por qué necesitamos estos middlewares?

Ahora mismo tu API funciona, pero le faltan cosas fundamentales para ser **robusta y profesional**:

| Problema actual | Middleware que lo resuelve |
|----------------|--------------------------|
| Si un handler hace `panic`, el servidor **se muere** | **Recovery** — captura panics y devuelve 500 |
| No sabes qué requests recibe tu API ni cuánto tardan | **Logger** — loguea cada request con método, path, status y duración |
| Si el frontend corre en otro puerto, el **navegador bloquea** las requests | **CORS** — permite requests cross-origin |
| No puedes rastrear un error en producción a un request concreto | **Request ID** — cada request recibe un UUID único para trazabilidad |
| Cada handler repite el **mismo switch de errores** (`ErrNotFound → 404`, etc.) | **Error Handler** — centraliza el mapeo error → status code |

### ¿Por qué importa el orden?

El orden en que registras los middlewares es el orden en que se ejecutan. El orden correcto es:

```
1. Request ID    ← Primero, para que todos los logs incluyan el ID
2. Logger        ← Segundo, para loguear la request con el ID
3. Recovery      ← Tercero, para capturar panics de cualquier cosa debajo
4. CORS          ← Cuarto, antes de cualquier lógica de negocio
5. Auth          ← Solo en rutas protegidas
```

Si pones Recovery **después** del Logger, y hay un panic en el Logger, no lo capturará. Si pones Request ID **después** del Logger, el Logger no tendrá el ID para incluirlo en el log.

---

## 3. ¿Qué hace `gin.Default()` por ti?

Si miras tu `router.go`, usas `gin.Default()`:

```go
r := gin.Default()
```

`gin.Default()` equivale a:

```go
r := gin.New()
r.Use(gin.Logger())    // ← Logger básico de Gin
r.Use(gin.Recovery())  // ← Recovery básico de Gin
```

Es decir, ya tienes un logger y un recovery. **¿Entonces por qué reemplazarlos?**

| Feature | `gin.Logger()` / `gin.Recovery()` | Nuestros middlewares custom |
|---------|----------------------------------|---------------------------|
| Formato de logs | Formato propio de Gin, no configurable | Formato consistente con tus logs de aplicación |
| Request ID en logs | ❌ No | ✅ Sí |
| Log de body/headers | ❌ No | Se puede añadir |
| Recovery con log | Imprime a stdout | Loguea con request ID para trazabilidad |
| CORS | ❌ No incluido | ✅ Configurable |

**Decisión**: Vamos a reemplazar `gin.Default()` por `gin.New()` y registrar nuestros propios middlewares. Así tenemos control total.

---

## 4. Paso 1 — Middleware de Request ID

📁 `internal/infrastructure/adapters/inbound/http/middleware/request_id.go`

### ¿Qué es y para qué sirve?

Cada request que entra a tu API recibe un **UUID único**. Este ID:

1. Se propaga en el header de respuesta (`X-Request-ID`) para que el cliente lo vea.
2. Se almacena en el `context` de Gin para que todos los middlewares y handlers lo usen.
3. Aparece en **todos los logs** de ese request.

**¿Por qué?** Imagina que en producción un usuario dice "me da error". Sin request ID, tienes que buscar en miles de logs. Con request ID, el usuario te dice `X-Request-ID: abc-123` y buscar ese ID te muestra toda la traza de ese request específico.

### Decisión de diseño: ¿Dónde vive?

```
Request ID Middleware → Infrastructure (inbound adapter HTTP)
```

Es un concepto **puramente HTTP**. No tiene nada que ver con el dominio ni con la aplicación. Vive en el mismo paquete que el `AuthMiddleware` que ya tienes.

### Implementación

```go
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDKey is the key used to store/retrieve the request ID in Gin's context.
const RequestIDKey = "requestID"

// RequestIDHeader is the HTTP header name used to propagate the request ID.
const RequestIDHeader = "X-Request-ID"

// RequestIDMiddleware assigns a unique UUID to each incoming request.
//
// The ID is:
//   - Stored in Gin's context (available to all downstream handlers/middlewares)
//   - Set as a response header (X-Request-ID) so the client can reference it
//
// If the client already sends an X-Request-ID header, we respect it.
// This is useful when requests pass through multiple services (microservices).
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the client already sent a request ID
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Store in context for downstream use (logger, handlers, etc.)
		c.Set(RequestIDKey, requestID)

		// Set as response header so the client can see it
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}

// GetRequestIDFromContext extracts the request ID from the Gin context.
// Returns an empty string if not found (middleware not applied).
func GetRequestIDFromContext(c *gin.Context) string {
	value, exists := c.Get(RequestIDKey)
	if !exists {
		return ""
	}

	requestID, ok := value.(string)
	if !ok {
		return ""
	}

	return requestID
}
```

### 💡 ¿Por qué reutilizar el `X-Request-ID` del cliente?

En una arquitectura de microservicios, un request puede pasar por varios servicios (API Gateway → Backend → Servicio de emails). Si cada servicio genera un ID nuevo, pierdes la trazabilidad. Si el primer servicio genera el ID y lo propaga en el header, todos los servicios usan el mismo ID → puedes rastrear un request a través de todo el sistema.

Para tu caso (un solo servicio), no es estrictamente necesario. Pero es una buena práctica que no cuesta nada implementar y te prepara para el futuro.

### 💡 ¿Por qué no usar `uuid.New()` directamente en el logger?

Porque el ID necesita ser **el mismo** en todos los places:
- En el log de entrada.
- En el log de salida.
- En el header de respuesta.
- Si lo pasas a otros servicios.

Al meterlo en el `context`, todos los que lo necesiten leen el mismo valor.

---

## 5. Paso 2 — Middleware de Logging

📁 `internal/infrastructure/adapters/inbound/http/middleware/logger.go`

### ¿Qué es y para qué sirve?

El logger middleware registra información de **cada request** que entra y sale de tu API:

```
[2026-03-09 10:30:00] [req:abc-123] POST /api/v1/users/me/clients → 201 (23ms)
[2026-03-09 10:30:01] [req:def-456] GET  /api/v1/users/me         → 200 (5ms)
[2026-03-09 10:30:02] [req:ghi-789] POST /api/v1/auth/login        → 401 (12ms)
```

Con esta información puedes:
- Ver qué endpoints se usan más.
- Deteccionartar requests lentos.
- Correlacionar errores con requests específicos (gracias al request ID).

### Decisión de diseño: `log.Printf` vs logger externo

| Opción | Pros | Contras |
|--------|------|---------|
| `log.Printf` (stdlib) | Zero dependencias, simple | No es JSON, no tiene niveles (info/warn/error) |
| `slog` (Go 1.21+) | Stdlib, estructurado, JSON opcional | Más verboso que log.Printf |
| `zerolog` / `zap` | Muy potentes, rápidos, JSON | Dependencia externa |

**Nuestra decisión**: Usamos `log.Printf` por ahora. Es suficiente para esta fase. En la Fase 6 (Observabilidad) lo reemplazamos por `slog` con logs JSON. No queremos añadir complejidad innecesaria en este momento.

### Implementación

```go
package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware logs each HTTP request with method, path, status code,
// duration, and client IP. It also includes the request ID if available.
//
// Example output:
//   [req:abc-123] POST   /api/v1/users  201  23.45ms  192.168.1.1
//   [req:def-456] GET    /api/v1/health 200   1.23ms  192.168.1.1
//
// This middleware must be registered AFTER RequestIDMiddleware
// so that the request ID is available in the context.
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// --- BEFORE handler ---
		start := time.Now()
		path := c.Request.URL.Path
		if c.Request.URL.RawQuery != "" {
			path = path + "?" + c.Request.URL.RawQuery
		}

		// Execute the handler (and all remaining middlewares)
		c.Next()

		// --- AFTER handler ---
		duration := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		requestID := GetRequestIDFromContext(c)

		// Color the status code for readability in terminal
		// (In Fase 6 we'll switch to JSON logs, so this visual format is temporary)
		log.Printf("[req:%s] %-6s %s %d %v %s",
			requestID,
			method,
			path,
			statusCode,
			duration,
			clientIP,
		)
	}
}
```

### 💡 ¿Por qué medir el tiempo con `time.Since(start)`?

`time.Now()` captura el instante antes de ejecutar el handler. Después de `c.Next()` (que ejecuta el handler y espera a que termine), `time.Since(start)` calcula cuánto tiempo pasó. Así sabes cuánto tardó el handler en responder.

Si ves que un endpoint tarda 500ms, sabes que algo no está bien (quizás una query lenta a la DB).

### 💡 ¿Por qué el logger va DESPUÉS del Request ID?

El logger necesita leer el request ID del context. Si lo registras antes que el Request ID middleware, el context aún no tiene el ID y el log dirá `[req:]` (vacío).

Orden correcto:
```
r.Use(RequestIDMiddleware())  // 1. Genera el ID
r.Use(LoggerMiddleware())     // 2. Lee el ID y loguea
```

### 💡 ¿Qué es `c.Writer.Status()`?

Después de que el handler ejecuta `c.JSON(201, ...)`, Gin almacena ese status code (201) en el `ResponseWriter`. El logger lo lee DESPUÉS de `c.Next()` porque antes aún no se ha escrito la respuesta.

Este es el poder del patrón antes/después de `c.Next()`:
- **Antes**: Sabes qué entra (método, path, IP).
- **Después**: Sabes qué salió (status code, duración).

---

## 6. Paso 3 — Middleware de Recovery

📁 `internal/infrastructure/adapters/inbound/http/middleware/recovery.go`

### ¿Qué es un `panic` y por qué necesitas recovery?

En Go, un `panic` es un **error catastrófico** que detiene la ejecución del programa:

```go
func (h *UserHandler) GetMe(c *gin.Context) {
    var user *entities.User // nil
    c.JSON(200, user.Name) // ← PANIC: nil pointer dereference
}
```

Sin recovery:
1. El handler ejecuta `panic`.
2. Go destruye el goroutine.
3. El servidor **se muere completamente**.
4. Todos los demás usuarios pierden conexión.

Con recovery:
1. El handler ejecuta `panic`.
2. El recovery middleware **captura** el panic (con `recover()`).
3. Devuelve un **500 Internal Server Error** limpio al cliente.
4. Loguea el error y el stack trace para debugging.
5. El servidor **sigue funcionando** normalmente.

### ¿Qué es `recover()` en Go?

`recover()` es una función built-in de Go que:
- Solo funciona dentro de un `defer`.
- Captura el valor del último `panic`.
- Si no hay panic, devuelve `nil`.

```go
defer func() {
    if r := recover(); r != nil {
        // r contiene lo que se pasó a panic()
        // Aquí puedes loguear, devolver un error, etc.
    }
}()
```

### Implementación

```go
package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// RecoveryMiddleware catches panics in handlers and returns a clean 500 error
// instead of crashing the server.
//
// It logs the panic message and stack trace for debugging,
// and includes the request ID if available for traceability.
//
// This middleware should be registered early in the chain so that it can
// catch panics from any downstream middleware or handler.
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				requestID := GetRequestIDFromContext(c)

				// Log the panic with stack trace for debugging
				log.Printf("[req:%s] PANIC recovered: %v\n%s",
					requestID,
					r,
					debug.Stack(),
				)

				// Return a clean 500 to the client (don't expose internal details)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":      "internal server error",
					"request_id": requestID,
				})
			}
		}()

		c.Next()
	}
}
```

### 💡 ¿Por qué `defer` y no un `if` normal?

`recover()` **solo funciona dentro de un `defer`**. Eso es una regla del lenguaje Go. La razón es que cuando ocurre un `panic`, Go "desenrolla" la pila de llamadas (stack unwinding) y ejecuta los `defer` pendientes. Si `recover()` no estuviera en un `defer`, el programa ya habría terminado antes de que se ejecute.

### 💡 ¿Qué es `debug.Stack()`?

`runtime/debug.Stack()` devuelve un `[]byte` con el **stack trace** completo: qué función llamó a qué, en qué línea, etc. Es invaluable para debugging:

```
goroutine 1 [running]:
runtime/debug.Stack()
    /usr/local/go/src/runtime/debug/stack.go:24
main.handler()
    /app/handler.go:42           ← ¡Aquí está el bug!
net/http.(*ServeMux).ServeHTTP()
    ...
```

### 💡 ¿Por qué no exponer el error al cliente?

El `panic` puede contener información interna: paths del servidor, queries SQL, estructuras de datos. Exponer eso es un **riesgo de seguridad**. Al cliente solo le dices "internal server error" y el `request_id` para que pueda reportarlo. Tú miras el error real en los logs.

### 💡 ¿Por qué incluir `request_id` en la respuesta 500?

Es una práctica de UX de API:
- El cliente recibe un 500 con `"request_id": "abc-123"`.
- Puede enviarte ese ID al reportar el error.
- Tú buscas `abc-123` en los logs y ves exactamente qué panificó y el stack trace.

---

## 7. Paso 4 — Middleware de CORS

### ¿Qué es CORS?

**CORS** = Cross-Origin Resource Sharing (Compartir Recursos entre Orígenes).

Cuando el frontend corre en `http://localhost:3000` y la API en `http://localhost:8080`, son **orígenes distintos** (diferente puerto). Por seguridad, los navegadores **bloquean** automáticamente las requests entre orígenes diferentes.

```
Frontend (localhost:3000)              API (localhost:8080)
    │                                       │
    │  fetch('/api/v1/users/me')            │
    │ ─────────────────────────────────────►│
    │                                       │
    │  ← BLOQUEADO por el navegador ❌      │
    │     "No Access-Control-Allow-Origin"  │
```

CORS le dice al navegador: "Sí, deja pasar requests desde ese origen".

### ¿Por qué existe esta restricción?

Imagina que estás logueado en tu banco (`banco.com`). Una página maliciosa (`hacker.com`) intenta hacer un `fetch('https://banco.com/api/transfer')` desde tu navegador. Sin CORS, esa request se enviaría con tus cookies del banco y el atacante podría transferir tu dinero.

Con CORS, `banco.com` dice "solo acepto requests desde `banco.com`" y el navegador bloquea la request de `hacker.com`.

### ¿Cómo funciona CORS técnicamente?

Hay dos tipos de requests CORS:

**1. Requests simples** (GET, POST con `Content-Type: application/x-www-form-urlencoded`):
- El navegador envía la request directamente.
- Si la respuesta no tiene el header `Access-Control-Allow-Origin`, el navegador la bloquea.

**2. Requests con preflight** (PUT, DELETE, o POST con `Content-Type: application/json`):
- El navegador envía primero una request **OPTIONS** (el "preflight") preguntando "¿puedo enviar esta request?".
- El servidor responde con los headers CORS.
- Si todo es válido, el navegador envía la request real.

```
Frontend                              API
  │                                     │
  │  OPTIONS /api/v1/users/me           │  ← Preflight
  │  Origin: http://localhost:3000      │
  │ ───────────────────────────────────►│
  │                                     │
  │  204 No Content                     │
  │  Access-Control-Allow-Origin: *     │
  │  Access-Control-Allow-Methods: GET  │
  │ ◄───────────────────────────────────│
  │                                     │
  │  GET /api/v1/users/me              │  ← Request real
  │  Authorization: Bearer <JWT>        │
  │ ───────────────────────────────────►│
  │                                     │
  │  200 { user data }                  │
  │ ◄───────────────────────────────────│
```

### Decisión de diseño: librería vs implementación manual

| Opción | Pros | Contras |
|--------|------|---------|
| Manual (headers a mano) | Zero dependencias | Fácil equivocarse, muchos edge cases |
| `github.com/gin-contrib/cors` | Bien probada, configurable, maneja preflight | Una dependencia más |

**Nuestra decisión**: Usamos `github.com/gin-contrib/cors`. CORS tiene muchos edge cases (preflight, credenciales, headers expuestos) que es fácil equivocarse al implementar a mano.

### Instalación

```bash
go get github.com/gin-contrib/cors
```

### Configuración en el Router

📁 La configuración de CORS se aplica directamente en `router.go` (no es un archivo separado, porque CORS es una configuración del router, no lógica de middleware reutilizable).

```go
import (
	"time"
	"github.com/gin-contrib/cors"
)

// Inside SetupRoutes, BEFORE defining routes:
r.Use(cors.New(cors.Config{
	// AllowOrigins: Which domains can call your API.
	// In development: allow localhost on any port.
	// In production: set this to your frontend domain(s) only.
	AllowOrigins: []string{
		"http://localhost:3000",
		"http://localhost:5173",
	},

	// AllowMethods: Which HTTP methods are allowed.
	AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},

	// AllowHeaders: Which headers the client can send.
	// "Authorization" is needed for JWT, "Content-Type" for JSON bodies.
	AllowHeaders: []string{"Authorization", "Content-Type", "X-Request-ID"},

	// ExposeHeaders: Which headers the client can READ from the response.
	// By default, browsers only expose a few "safe" headers.
	ExposeHeaders: []string{"X-Request-ID"},

	// AllowCredentials: Allow cookies/auth headers.
	// Must be true if clients send Authorization headers.
	AllowCredentials: true,

	// MaxAge: How long the browser caches preflight results.
	// 12 hours = fewer preflight requests = better performance.
	MaxAge: 12 * time.Hour,
}))
```

### 💡 ¿Por qué no poner `AllowOrigins: ["*"]`?

El wildcard `*` significa "acepta requests de cualquier origen". Es tentador, pero:

1. **No funciona con `AllowCredentials: true`**. El estándar CORS prohíbe usar `*` con credenciales. Si lo haces, el navegador ignora el header.
2. **En producción es inseguro**. Cualquier sitio web podría llamar a tu API.

La excepción es si tu API es pública (como la API de GitHub). Para una API con autenticación, siempre lista los orígenes permitidos explícitamente.

### 💡 ¿Qué es `ExposeHeaders`?

Por defecto, cuando una respuesta llega al frontend, el navegador solo deja leer ciertos headers "seguros" (`Content-Type`, `Content-Length`, etc.). Si quieres que tu frontend pueda leer `X-Request-ID` de la respuesta (para mostrarlo al usuario si hay un error), necesitas exponerlo explícitamente.

### 💡 ¿Qué pasa si no configuro CORS?

Si tu API corre en `localhost:8080` y haces fetch desde `localhost:3000`:

```javascript
// En el frontend
const res = await fetch('http://localhost:8080/api/v1/users/me', {
    headers: { Authorization: 'Bearer ...' }
});
// ❌ Error: "Access to fetch has been blocked by CORS policy"
```

El request **nunca llega** a tu handler. El navegador lo bloquea antes de enviarlo (en preflight) o después de recibirlo (descarta la respuesta).

> **Nota**: Herramientas como `curl` o Postman **no aplican CORS**. CORS es una protección del **navegador**, no del servidor. Por eso tu `test/api.http` funciona sin CORS.

---

## 8. Paso 5 — Manejo de Errores Centralizado

📁 `internal/infrastructure/adapters/inbound/http/middleware/error_handler.go`

### El problema actual

Mira tu `auth_handler.go`:

```go
func (h *AuthHandler) Register(c *gin.Context) {
	// ...
	user, token, err := h.authService.Register(...)
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
}
```

Y tu `client_handler.go`:

```go
func (h *ClientHandler) CreateClient(c *gin.Context) {
	// ...
	client, err := h.clientService.CreateClient(...)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, services.ErrUserNotFound):
			status = http.StatusNotFound
		case errors.Is(err, entities.ErrEmptyClientName):
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
}
```

¿Ves el patrón repetido? Cada handler tiene un **switch** que mapea errores de dominio a status codes HTTP. Cuando tengas 10 entidades, tendrás ese switch repetido en 20 handlers.

### La solución: centralizar el mapeo

La idea es tener **un solo lugar** que sepa:
- `ErrNotFound` → 404
- `ErrConflict` → 409
- `ErrBadRequest` → 400
- etc.

Hay dos enfoques:

| Enfoque | Cómo funciona | Complejidad |
|---------|--------------|-------------|
| **Middleware con `c.Errors`** | Los handlers añaden errores a `c.Errors`, el middleware los convierte a HTTP al final | Alta (cambia todos los handlers) |
| **Helper function** | Una función que los handlers llaman en vez del switch repetido | Baja (cambio incremental) |

**Nuestra decisión**: Usamos un **helper function**. Es más simple, no requiere cambiar la forma en que funcionan los handlers, y es más explícito (ves el `handleError()` en cada handler).

### Paso 5.1 — Definir errores de dominio tipados

Primero necesitamos una forma de que los errores de dominio "sepan" qué tipo de error HTTP son. Creamos un tipo de error custom:

📁 `internal/infrastructure/adapters/inbound/http/middleware/error_handler.go`

```go
package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// DomainError represents a business error that can be mapped to an HTTP status code.
// Domain and application errors are registered in the errorMap below.
type DomainError struct {
	StatusCode int
	Message    string
}

// errorMap is the single source of truth for mapping domain errors to HTTP status codes.
// When you add a new domain error, register it here.
var errorMap = map[error]DomainError{}

// RegisterDomainError maps a domain/application error to an HTTP status code.
// Call this during initialization (in router.go or main.go) to register all known errors.
//
// Example:
//
//	middleware.RegisterDomainError(services.ErrUserNotFound, http.StatusNotFound, "user not found")
//	middleware.RegisterDomainError(services.ErrEmailAlreadyInUse, http.StatusConflict, "email already in use")
func RegisterDomainError(err error, statusCode int, message string) {
	errorMap[err] = DomainError{
		StatusCode: statusCode,
		Message:    message,
	}
}

// HandleError maps a domain/application error to the appropriate HTTP response.
// If the error is registered in the errorMap, it returns the mapped status code and message.
// If the error is unknown, it returns 500 Internal Server Error.
//
// Usage in handlers:
//
//	user, err := h.userService.GetUserByID(ctx, id)
//	if err != nil {
//	    middleware.HandleError(c, err)
//	    return
//	}
func HandleError(c *gin.Context, err error) {
	for domainErr, httpErr := range errorMap {
		if errors.Is(err, domainErr) {
			c.JSON(httpErr.StatusCode, gin.H{"error": httpErr.Message})
			return
		}
	}

	// Unknown error → 500 (don't expose internal details)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
```

### Paso 5.2 — Registrar los errores conocidos

📁 En `router.go`, al inicio de `SetupRoutes`:

```go
func SetupRoutes(...) *gin.Engine {
	// Register domain error → HTTP status mappings
	middleware.RegisterDomainError(services.ErrUserNotFound, http.StatusNotFound, "user not found")
	middleware.RegisterDomainError(services.ErrEmailAlreadyInUse, http.StatusConflict, "email already in use")
	middleware.RegisterDomainError(services.ErrEmailTaken, http.StatusConflict, "email already registered")
	middleware.RegisterDomainError(services.ErrInvalidCredentials, http.StatusUnauthorized, "invalid email or password")
	middleware.RegisterDomainError(services.ErrClientNotFound, http.StatusNotFound, "client not found")
	middleware.RegisterDomainError(services.ErrClientNotOwned, http.StatusForbidden, "client does not belong to this user")
	middleware.RegisterDomainError(entities.ErrEmptyUserName, http.StatusBadRequest, "user name cannot be empty")
	middleware.RegisterDomainError(entities.ErrEmptyClientName, http.StatusBadRequest, "client name cannot be empty")
	middleware.RegisterDomainError(valueobjects.ErrPasswordTooShort, http.StatusBadRequest, "password must be at least 8 characters")
	middleware.RegisterDomainError(valueobjects.ErrPasswordEmpty, http.StatusBadRequest, "password cannot be empty")
	middleware.RegisterDomainError(valueobjects.ErrInvalidEmail, http.StatusBadRequest, "invalid email format")

	// ... rest of setup
}
```

### Paso 5.3 — Simplificar los handlers

**Antes** (auth_handler.go):

```go
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
```

**Después**:

```go
user, token, err := h.authService.Register(c.Request.Context(), req.Name, req.Email, req.Password)
if err != nil {
	middleware.HandleError(c, err)
	return
}
```

Mucho más limpio. El handler ya no necesita saber qué error es cada uno. Solo dice "hubo un error, manejalo".

### 💡 ¿Por qué un helper y no un middleware puro?

Un middleware puro usaría `c.Errors.Last()` para obtener el error, pero eso requiere que los handlers llamen a `c.Error(err)` en vez de `c.JSON(...)`. Es un cambio de paradigma que afecta a TODOS los handlers y tests. Es más invasivo.

Con el helper function:
1. Puedes migrarlo **handler por handler** (no todo de golpe).
2. Los handlers siguen controlando cuándo responder.
3. Los tests existentes siguen funcionando mientras migras.

### 💡 ¿Qué pasa si un error no está registrado?

Devuelve `500 Internal Server Error` con un mensaje genérico. Esto es intencional:
- **Seguridad**: No expones errores internos (queries SQL, panics, etc.).
- **Debugging**: El error real se puede loguear en el logger middleware.

### 💡 ¿Puedo seguir usando el switch en algunos handlers?

Sí. El helper es opcional. Si un handler necesita hacer algo especial con un error (ej: devolver un campo extra), puedes seguir manejándolo a mano. No es todo o nada.

---

## 9. Paso 6 — Integrar todo en el Router

📁 `internal/infrastructure/adapters/inbound/http/router.go`

Ahora viene el momento de juntar todo. El cambio principal es reemplazar `gin.Default()` por `gin.New()` y registrar nuestros middlewares en el orden correcto.

### Router actual (ANTES)

```go
func SetupRoutes(
	userService usecases.UserService,
	clientService usecases.ClientService,
	authService usecases.AuthService,
	tokenProvider ports.TokenProvider,
) *gin.Engine {
	r := gin.Default() // ← Incluye gin.Logger() y gin.Recovery()

	v1 := r.Group("/api/v1")
	// ... routes ...
}
```

### Router nuevo (DESPUÉS)

```go
package http

import (
	"net/http"
	"time"

	"ductifact/internal/application/ports"
	"ductifact/internal/application/services"
	"ductifact/internal/application/usecases"
	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/valueobjects"
	"ductifact/internal/infrastructure/adapters/inbound/http/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(
	userService usecases.UserService,
	clientService usecases.ClientService,
	authService usecases.AuthService,
	tokenProvider ports.TokenProvider,
) *gin.Engine {
	// --- Register domain error mappings ---
	middleware.RegisterDomainError(services.ErrUserNotFound, http.StatusNotFound, "user not found")
	middleware.RegisterDomainError(services.ErrEmailAlreadyInUse, http.StatusConflict, "email already in use")
	middleware.RegisterDomainError(services.ErrEmailTaken, http.StatusConflict, "email already registered")
	middleware.RegisterDomainError(services.ErrInvalidCredentials, http.StatusUnauthorized, "invalid email or password")
	middleware.RegisterDomainError(services.ErrClientNotFound, http.StatusNotFound, "client not found")
	middleware.RegisterDomainError(services.ErrClientNotOwned, http.StatusForbidden, "client does not belong to this user")
	middleware.RegisterDomainError(entities.ErrEmptyUserName, http.StatusBadRequest, "user name cannot be empty")
	middleware.RegisterDomainError(entities.ErrEmptyClientName, http.StatusBadRequest, "client name cannot be empty")
	middleware.RegisterDomainError(valueobjects.ErrPasswordTooShort, http.StatusBadRequest, "password must be at least 8 characters")
	middleware.RegisterDomainError(valueobjects.ErrPasswordEmpty, http.StatusBadRequest, "password cannot be empty")
	middleware.RegisterDomainError(valueobjects.ErrInvalidEmail, http.StatusBadRequest, "invalid email format")

	// --- Create router WITHOUT default middlewares ---
	r := gin.New()

	// --- Register middlewares in correct order ---
	// 1. Request ID: first, so all logs include the ID
	r.Use(middleware.RequestIDMiddleware())

	// 2. Logger: second, to log each request with the ID
	r.Use(middleware.LoggerMiddleware())

	// 3. Recovery: third, to catch panics from anything below
	r.Use(middleware.RecoveryMiddleware())

	// 4. CORS: fourth, before any business logic
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

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
		userRoutes.GET("/me", userHandler.GetMe)
		userRoutes.PUT("/me", userHandler.UpdateMe)
	}

	// Client routes
	clientHandler := NewClientHandler(clientService)
	clientRoutes := protected.Group("/users/me/clients")
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

### 💡 ¿Por qué `gin.New()` en vez de `gin.Default()`?

`gin.Default()` registra `gin.Logger()` y `gin.Recovery()` automáticamente. Como ahora tenemos nuestras propias versiones, usamos `gin.New()` (router vacío) y registramos los nuestros. Si usaras `gin.Default()`, tendrías los middlewares duplicados.

### 💡 Diagrama del flujo completo

```
Request: POST /api/v1/users/me/clients
         Authorization: Bearer <JWT>
         Content-Type: application/json
         { "name": "Acme Corp" }
    │
    ▼
┌──────────────────────────────────────────────────────────┐
│ 1. RequestIDMiddleware                                    │
│    → Genera UUID "abc-123"                                │
│    → Set header X-Request-ID: abc-123                     │
│    → Set context: requestID = "abc-123"                   │
├──────────────────────────────────────────────────────────┤
│ 2. LoggerMiddleware                                       │
│    → Captura: start = time.Now()                          │
│    → c.Next() ... espera ... (handler ejecuta)            │
│    → Log: [req:abc-123] POST /api/v1/users/me/clients     │
│           201 23ms 192.168.1.1                            │
├──────────────────────────────────────────────────────────┤
│ 3. RecoveryMiddleware                                     │
│    → defer recover() (red de seguridad)                   │
│    → Si hay panic: log + 500 + abort                      │
├──────────────────────────────────────────────────────────┤
│ 4. CORS Middleware                                        │
│    → Verifica Origin header                               │
│    → Si es preflight (OPTIONS): responde y abort          │
│    → Si es request real: añade headers CORS a la resp.    │
├──────────────────────────────────────────────────────────┤
│ 5. AuthMiddleware (solo en rutas protegidas)              │
│    → Lee Authorization: Bearer <JWT>                      │
│    → Valida token → extrae userID                         │
│    → Set context: userID = uuid                           │
├──────────────────────────────────────────────────────────┤
│ 6. ClientHandler.CreateClient                             │
│    → Lee userID del context                               │
│    → Parsea body JSON                                     │
│    → Llama clientService.CreateClient()                   │
│    → Responde 201 { client }                              │
└──────────────────────────────────────────────────────────┘
    │
    ▼
Response: 201 Created
          X-Request-ID: abc-123
          { "id": "...", "name": "Acme Corp", "user_id": "..." }
```

---

## 10. Paso 7 — Tests

### 10.1 Tests del Request ID Middleware

📁 `test/unit/infrastructure/http/middleware/request_id_middleware_test.go`

```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ductifact/internal/infrastructure/adapters/inbound/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestIDMiddleware_GeneratesNewID(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.GET("/test", func(c *gin.Context) {
		requestID := middleware.GetRequestIDFromContext(c)
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Response header should contain the request ID
	headerID := w.Header().Get("X-Request-ID")
	assert.NotEmpty(t, headerID)
	assert.Len(t, headerID, 36) // UUID format: 8-4-4-4-12
}

func TestRequestIDMiddleware_ReusesClientID(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.GET("/test", func(c *gin.Context) {
		requestID := middleware.GetRequestIDFromContext(c)
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	clientID := "my-custom-request-id-123"
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", clientID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Should reuse the client's ID
	headerID := w.Header().Get("X-Request-ID")
	assert.Equal(t, clientID, headerID)
}

func TestRequestIDMiddleware_EachRequestGetsDifferentID(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/test", nil))

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/test", nil))

	id1 := w1.Header().Get("X-Request-ID")
	id2 := w2.Header().Get("X-Request-ID")

	assert.NotEqual(t, id1, id2, "Each request should get a unique ID")
}
```

### 10.2 Tests del Recovery Middleware

📁 `test/unit/infrastructure/http/middleware/recovery_middleware_test.go`

```go
package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ductifact/internal/infrastructure/adapters/inbound/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecoveryMiddleware_CatchesPanic_Returns500(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.RecoveryMiddleware())
	r.GET("/panic", func(c *gin.Context) {
		panic("something went terribly wrong")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()

	// This should NOT panic — the middleware catches it
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "internal server error", body["error"])
	assert.NotEmpty(t, body["request_id"])
}

func TestRecoveryMiddleware_NoPanic_PassesThrough(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RecoveryMiddleware())
	r.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
```

### 10.3 Tests del Logger Middleware

📁 `test/unit/infrastructure/http/middleware/logger_middleware_test.go`

El logger imprime a `log.Printf`, así que no podemos capturar el output fácilmente en un test. El test se enfoca en que **el middleware no rompe la cadena** y que el handler sigue funcionando:

```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ductifact/internal/infrastructure/adapters/inbound/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestLoggerMiddleware_DoesNotBreakChain(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.LoggerMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "hello"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoggerMiddleware_PreservesStatusCode(t *testing.T) {
	r := gin.New()
	r.Use(middleware.LoggerMiddleware())
	r.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"created": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}
```

### 10.4 Tests del Error Handler

📁 `test/unit/infrastructure/http/middleware/error_handler_test.go`

```go
package middleware_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"ductifact/internal/infrastructure/adapters/inbound/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test errors (not real domain errors — just for testing)
var (
	errTestNotFound = errors.New("test: not found")
	errTestConflict = errors.New("test: conflict")
)

func TestHandleError_RegisteredError_ReturnsMappedStatus(t *testing.T) {
	// Register test errors
	middleware.RegisterDomainError(errTestNotFound, http.StatusNotFound, "resource not found")
	middleware.RegisterDomainError(errTestConflict, http.StatusConflict, "resource conflict")

	r := gin.New()
	r.GET("/test-not-found", func(c *gin.Context) {
		middleware.HandleError(c, errTestNotFound)
	})
	r.GET("/test-conflict", func(c *gin.Context) {
		middleware.HandleError(c, errTestConflict)
	})

	// Test 404
	req := httptest.NewRequest(http.MethodGet, "/test-not-found", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "resource not found", body["error"])

	// Test 409
	req = httptest.NewRequest(http.MethodGet, "/test-conflict", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandleError_UnknownError_Returns500(t *testing.T) {
	unknownErr := errors.New("some unexpected database error")

	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		middleware.HandleError(c, unknownErr)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "internal server error", body["error"])
}

func TestHandleError_WrappedError_StillMatches(t *testing.T) {
	// errors.Is works with wrapped errors thanks to Go's error wrapping
	middleware.RegisterDomainError(errTestNotFound, http.StatusNotFound, "resource not found")

	wrappedErr := fmt.Errorf("service layer: %w", errTestNotFound)

	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		middleware.HandleError(c, wrappedErr)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
```

> **Nota**: El test de `WrappedError` verifica que `errors.Is` funciona con errores envueltos. En Go, si haces `fmt.Errorf("context: %w", originalErr)`, `errors.Is(wrappedErr, originalErr)` devuelve `true`. Esto es fundamental porque los servicios pueden envolver errores al propagarlos.

### 10.5 Tests del CORS

El CORS se testea mejor con un E2E o integration test, pero aquí tienes un test unitario básico:

📁 `test/unit/infrastructure/http/middleware/cors_middleware_test.go`

```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupCORSRouter() *gin.Engine {
	r := gin.New()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	return r
}

func TestCORS_AllowedOrigin_ReturnsHeaders(t *testing.T) {
	r := setupCORSRouter()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_DisallowedOrigin_NoHeaders(t *testing.T) {
	r := setupCORSRouter()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://evil-site.com")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// The request still goes through (CORS is enforced by the browser, not the server).
	// But the response won't have Access-Control-Allow-Origin for the evil origin.
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_PreflightRequest_Returns204(t *testing.T) {
	r := setupCORSRouter()

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
}
```

---

## 11. Resumen de archivos

### Archivos nuevos

| Archivo | Qué es | Capa |
|---------|--------|------|
| `middleware/request_id.go` | Middleware: asigna UUID a cada request | Infraestructura (inbound) |
| `middleware/logger.go` | Middleware: loguea cada request | Infraestructura (inbound) |
| `middleware/recovery.go` | Middleware: captura panics → 500 | Infraestructura (inbound) |
| `middleware/error_handler.go` | Helper: centraliza error → HTTP status | Infraestructura (inbound) |

### Archivos modificados

| Archivo | Qué cambia |
|---------|-----------|
| `router.go` | `gin.New()` en vez de `gin.Default()`, registro de middlewares, registro de errores, CORS config |
| `go.mod` | Nueva dependencia: `github.com/gin-contrib/cors` |

### Archivos que puedes migrar gradualmente

| Archivo | Cambio |
|---------|--------|
| `auth_handler.go` | Reemplazar switches por `middleware.HandleError(c, err)` |
| `user_handler.go` | Reemplazar switches por `middleware.HandleError(c, err)` |
| `client_handler.go` | Reemplazar switches por `middleware.HandleError(c, err)` |

### Tests nuevos

| Archivo | Qué testea |
|---------|-----------|
| `test/unit/.../request_id_middleware_test.go` | Genera ID, reutiliza ID del cliente, IDs únicos |
| `test/unit/.../recovery_middleware_test.go` | Captura panic → 500, no panic → pasa |
| `test/unit/.../logger_middleware_test.go` | No rompe la cadena, preserva status |
| `test/unit/.../error_handler_test.go` | Error registrado → status correcto, error desconocido → 500 |
| `test/unit/.../cors_middleware_test.go` | Origen permitido, origen denegado, preflight |

### Mapa visual de la arquitectura

```
cmd/
  api/
    main.go                          ← Wiring (no changes here)

internal/
  infrastructure/
    adapters/
      inbound/
        http/
          router.go                  ← MODIFIED: gin.New(), middleware chain, CORS, error registry
          auth_handler.go            ← OPTIONAL: simplify with HandleError()
          user_handler.go            ← OPTIONAL: simplify with HandleError()
          client_handler.go          ← OPTIONAL: simplify with HandleError()
          middleware/
            auth.go                  ← EXISTING (no changes)
            context.go               ← EXISTING (no changes)
            request_id.go            ← NEW
            logger.go                ← NEW
            recovery.go              ← NEW
            error_handler.go         ← NEW

test/
  unit/
    infrastructure/
      http/
        middleware/
          auth_middleware_test.go              ← EXISTING (no changes)
          request_id_middleware_test.go        ← NEW
          recovery_middleware_test.go          ← NEW
          logger_middleware_test.go            ← NEW
          error_handler_test.go               ← NEW
          cors_middleware_test.go              ← NEW
```

---

## 12. Errores comunes

### ❌ Error: "middleware ejecutado en orden incorrecto"

**Síntoma**: El logger no muestra el request ID.

**Causa**: Registraste `LoggerMiddleware()` antes de `RequestIDMiddleware()`.

**Solución**: Siempre registra Request ID primero:
```go
r.Use(middleware.RequestIDMiddleware())  // 1º
r.Use(middleware.LoggerMiddleware())     // 2º
```

### ❌ Error: "CORS: Access-Control-Allow-Origin contains wildcard and credentials"

**Síntoma**: El navegador rechaza la request CORS con un error críptico.

**Causa**: Pusiste `AllowOrigins: []string{"*"}` con `AllowCredentials: true`.

**Solución**: Lista los orígenes explícitamente:
```go
AllowOrigins: []string{"http://localhost:3000"},
```

### ❌ Error: "middleware duplicados (logger aparece dos veces)"

**Síntoma**: Cada request se loguea dos veces.

**Causa**: Estás usando `gin.Default()` (que incluye `gin.Logger()`) Y registrando `LoggerMiddleware()`.

**Solución**: Usa `gin.New()` cuando registras tus propios middlewares.

### ❌ Error: "panic en el logger middleware no se captura"

**Síntoma**: El servidor se cae a pesar de tener RecoveryMiddleware.

**Causa**: El panic ocurre en un middleware que se ejecuta ANTES del RecoveryMiddleware.

**Solución**: Registra RecoveryMiddleware lo más alto posible en la cadena:
```go
r.Use(middleware.RequestIDMiddleware())   // Si esto panifica, nadie lo captura
r.Use(middleware.LoggerMiddleware())      // pero estos middlewares son tan simples
r.Use(middleware.RecoveryMiddleware())    // que es muy improbable que pasen un panic
```

En la práctica, RequestID y Logger son tan simples que no deberían panificar. Si aún así te preocupa, puedes poner Recovery primero, pero perderías el request ID en el log del panic.

### ❌ Error: "error_handler no reconoce un error envuelto"

**Síntoma**: Un error de dominio devuelve 500 en vez del status esperado.

**Causa**: El servicio envuelve el error con `fmt.Errorf("context: %w", originalErr)` pero no usas `errors.Is`.

**Solución**: `HandleError` ya usa `errors.Is`, que maneja errores envueltos correctamente. Verifica que el error original está registrado en el `errorMap`.

---

## 13. Glosario

| Término | Definición |
|---------|-----------|
| **Middleware** | Función que intercepta requests HTTP antes y/o después del handler. Se usa para cross-cutting concerns (logging, auth, CORS, etc.) |
| **CORS** | Cross-Origin Resource Sharing. Mecanismo que permite a un servidor indicar qué orígenes pueden acceder a sus recursos. |
| **Preflight** | Request OPTIONS que el navegador envía antes de la request real para verificar si el servidor acepta CORS. |
| **Request ID** | UUID único asignado a cada request para trazabilidad en logs. |
| **Recovery** | Middleware que captura `panic` en Go y devuelve un 500 limpio en vez de crashear el servidor. |
| **`gin.Default()`** | Crea un router Gin con Logger y Recovery incluidos. Equivale a `gin.New()` + `r.Use(gin.Logger(), gin.Recovery())`. |
| **`gin.New()`** | Crea un router Gin vacío, sin middlewares. Ideal cuando quieres control total. |
| **`c.Next()`** | Ejecuta el siguiente middleware o handler en la cadena. Lo que va después de `c.Next()` se ejecuta cuando la cadena vuelve. |
| **`c.Abort()`** | Detiene la cadena de middlewares. Los handlers restantes no se ejecutan. |
| **`c.Set()` / `c.Get()`** | Almacena y recupera valores del context de Gin. Útil para pasar datos entre middlewares y handlers. |
| **Cross-cutting concern** | Funcionalidad que afecta a toda la aplicación (logging, auth, error handling) y no pertenece a un handler específico. |
| **Chain of responsibility** | Patrón de diseño donde un request pasa por una cadena de procesadores (middlewares). Cada uno decide si lo procesa o lo pasa al siguiente. |
| **`errors.Is`** | Función de Go que verifica si un error (posiblemente envuelto) coincide con un error específico. Sigue la cadena de `%w`. |

---

> **Siguiente paso**: Una vez implementados estos middlewares, puedes marcar la Fase 2 como completada y continuar con la implementación de código. Recuerda que la migración de los handlers al `HandleError` centralizado es **opcional y gradual**: puedes hacerlo handler por handler sin romper nada.
