# Guía: Observabilidad — Logging Estructurado y Health Check

> Esta guía explica **qué es la observabilidad**, **por qué importa**, y te guía paso a paso para implementar logging estructurado con `log/slog` y un health check real en Ductifact.

---

## Índice

1. [¿Qué es la observabilidad?](#1-qué-es-la-observabilidad)
2. [Los tres pilares de la observabilidad](#2-los-tres-pilares-de-la-observabilidad)
3. [Estado actual del proyecto](#3-estado-actual-del-proyecto)
4. [Pilar 1 — Logging estructurado con `slog`](#4-pilar-1--logging-estructurado-con-slog)
   - 4.1 [¿Qué es `log/slog` y por qué usarlo?](#41-qué-es-logslog-y-por-qué-usarlo)
   - 4.2 [Conceptos clave de `slog`](#42-conceptos-clave-de-slog)
   - 4.3 [Paso 1 — Crear el paquete de logging](#43-paso-1--crear-el-paquete-de-logging)
   - 4.4 [Paso 2 — Inicializar el logger en `main.go`](#44-paso-2--inicializar-el-logger-en-maingo)
   - 4.5 [Paso 3 — Migrar el middleware de logging](#45-paso-3--migrar-el-middleware-de-logging)
   - 4.6 [Paso 4 — Migrar el middleware de recovery](#46-paso-4--migrar-el-middleware-de-recovery)
   - 4.7 [Paso 5 — Migrar el error handler](#47-paso-5--migrar-el-error-handler)
   - 4.8 [Paso 6 — Migrar `main.go` y `connection.go`](#48-paso-6--migrar-maingo-y-connectiongo)
   - 4.9 [Paso 7 — Logging contextual en servicios (opcional)](#49-paso-7--logging-contextual-en-servicios-opcional)
5. [Pilar 2 — Health check mejorado](#5-pilar-2--health-check-mejorado)
   - 5.1 [¿Qué es un buen health check?](#51-qué-es-un-buen-health-check)
   - 5.2 [Liveness vs Readiness](#52-liveness-vs-readiness)
   - 5.3 [Paso 1 — Definir el port `HealthChecker`](#53-paso-1--definir-el-port-healthchecker)
   - 5.4 [Paso 2 — Implementar el adapter `PostgresHealthChecker`](#54-paso-2--implementar-el-adapter-postgreshealthchecker)
   - 5.5 [Paso 3 — Crear el health handler](#55-paso-3--crear-el-health-handler)
   - 5.6 [Paso 4 — Registrar la ruta e inyectar desde `main.go`](#56-paso-4--registrar-la-ruta-e-inyectar-desde-maingo)
6. [Bonus — Graceful shutdown](#6-bonus--graceful-shutdown)
7. [Pilar 3 — Métricas (visión futura)](#7-pilar-3--métricas-visión-futura)
8. [Resumen de archivos modificados y creados](#8-resumen-de-archivos-modificados-y-creados)
9. [Checklist de implementación](#9-checklist-de-implementación)
10. [Glosario](#10-glosario)

---

## 1. ¿Qué es la observabilidad?

Imagina que tu API lleva 3 meses en producción y de repente los usuarios empiezan a reportar que "va lenta". ¿Cómo sabes qué está pasando? Si no puedes "mirar dentro" de tu sistema, estás a ciegas.

**Observabilidad** es la capacidad de entender el estado interno de un sistema a partir de sus outputs externos (logs, métricas, trazas). No es una herramienta, es una **propiedad** de tu sistema.

La diferencia con el **monitoreo** clásico:

| Concepto | Pregunta que responde | Ejemplo |
|----------|----------------------|---------|
| **Monitoreo** | ¿Está roto algo que ya conozco? | "Alerta: el CPU está al 95%" |
| **Observabilidad** | ¿Por qué se está comportando así? | "Los requests al endpoint `/clients` tardan 2s porque la query no tiene índice" |

El monitoreo te dice QUE hay un problema. La observabilidad te ayuda a entender POR QUÉ.

---

## 2. Los tres pilares de la observabilidad

```
┌─────────────────────────────────────────────────────┐
│                  OBSERVABILIDAD                     │
│                                                     │
│  ┌─────────────┐  ┌─────────────┐  ┌────────────┐  │
│  │    LOGS     │  │  MÉTRICAS   │  │   TRAZAS   │  │
│  │             │  │             │  │            │  │
│  │ "Qué pasó"  │  │ "Cuánto"    │  │ "El viaje" │  │
│  │             │  │             │  │            │  │
│  │ Eventos     │  │ Números     │  │ Request    │  │
│  │ discretos   │  │ agregados   │  │ completo   │  │
│  │ con         │  │ en el       │  │ a través   │  │
│  │ contexto    │  │ tiempo      │  │ de         │  │
│  │             │  │             │  │ servicios  │  │
│  └─────────────┘  └─────────────┘  └────────────┘  │
│                                                     │
│  ✅ Fase 6        ⬜ Fase 6         ⬜ Futuro      │
│  (este paso)      (opcional)        (microservicios)│
└─────────────────────────────────────────────────────┘
```

### Logs (lo que implementaremos)

Un **log** es un registro de un evento que ocurrió en tu sistema. Es como la caja negra de un avión.

**Sin estructura** (lo que tenemos ahora):
```
2026/03/09 10:00:00 [req:abc-123] POST   /api/v1/users  201  23.45ms  192.168.1.1
```

**Con estructura** (lo que queremos):
```json
{
  "time": "2026-03-09T10:00:00.000Z",
  "level": "INFO",
  "msg": "request completed",
  "request_id": "abc-123",
  "method": "POST",
  "path": "/api/v1/users",
  "status": 201,
  "duration_ms": 23.45,
  "client_ip": "192.168.1.1"
}
```

¿Por qué importa la diferencia? Porque con logs en JSON puedes:
- **Buscar**: "busca todos los logs donde `status >= 500` y `path` contiene `/clients`"
- **Filtrar**: "muéstrame solo los logs de nivel ERROR del último minuto"
- **Agregar**: "¿cuál es la duración promedio de los requests a `/auth/login`?"
- **Alertar**: "avísame si hay más de 10 errores 500 en 5 minutos"

Con `log.Printf` no puedes hacer nada de esto sin parsear el texto con regex.

### Métricas (visión futura)

Números que se agregan en el tiempo: requests/segundo, latencia p99, memoria usada. Piensa en un dashboard de Grafana. Lo dejaremos preparado conceptualmente pero no lo implementamos en esta fase.

### Trazas (fuera de scope)

Seguimiento de un request a través de múltiples servicios. Útil en microservicios. No lo necesitamos ahora — Ductifact es un monolito.

---

## 3. Estado actual del proyecto

Antes de cambiar nada, entendamos qué tenemos:

### Logging actual

Todos los archivos usan `log.Printf` del paquete estándar:

| Archivo | Qué loguea |
|---------|-----------|
| `cmd/api/main.go` | `log.Fatalf` al iniciar, `log.Printf` al arrancar |
| `middleware/logger.go` | Cada request HTTP: método, path, status, duración |
| `middleware/recovery.go` | Panics con stack trace |
| `helpers/error_handler.go` | Errores no mapeados (los 500) |
| `persistence/connection.go` | Fallos de migración |

**Problemas con `log.Printf`:**

1. **Sin niveles**: No distingues INFO de ERROR de WARN. Si tu app genera 10.000 logs/hora, ¿cómo filtras solo los errores?
2. **Sin estructura**: El output es texto plano. Para encontrar algo necesitas `grep` y ojos.
3. **Sin contexto propagado**: Si un request falla en el servicio, no tienes el `request_id` ahí para correlacionar con el log del middleware.
4. **Sin configuración**: No puedes cambiar el formato (JSON vs texto) ni el nivel mínimo sin cambiar código.

### Health check actual

```go
v1.GET("/health", func(c *gin.Context) {
    c.JSON(200, gin.H{"status": "healthy !!!!"})
})
```

Siempre devuelve "healthy" aunque la base de datos esté caída. Eso es como un doctor que dice "estás sano" sin ni siquiera mirarte.

---

## 4. Pilar 1 — Logging estructurado con `slog`

### 4.1 ¿Qué es `log/slog` y por qué usarlo?

`log/slog` es el paquete de **logging estructurado** que viene en la **stdlib de Go desde la versión 1.21**. Como usamos Go 1.24, lo tenemos disponible sin instalar nada.

¿Por qué `slog` y no `zerolog` o `zap`?

| Opción | Ventaja | Desventaja |
|--------|---------|------------|
| **`log/slog`** (stdlib) | Sin dependencia externa, API estable, respaldado por el equipo de Go | Ligeramente menos rápido que zerolog |
| **`zerolog`** | El más rápido, zero-allocation | Dependencia externa, API diferente al resto de Go |
| **`zap`** (Uber) | Muy maduro, gran ecosistema | Pesado, API verbosa, dependencia externa |

**Nuestra elección: `slog`**. Es la opción idiomática en Go moderno. Es "suficientemente rápido" para el 99% de los casos, no agrega dependencias, y será la forma estándar de loguear en Go de aquí en adelante.

### 4.2 Conceptos clave de `slog`

#### Niveles de log

`slog` define 4 niveles, de menor a mayor severidad:

```
DEBUG   →   INFO   →   WARN   →   ERROR
```

| Nivel | Cuándo usarlo | Ejemplo |
|-------|--------------|---------|
| `DEBUG` | Información detallada para debugging. No se activa en producción. | "Resolving user from token: user_id=abc-123" |
| `INFO` | Eventos normales del flujo. Lo que quieres ver en producción. | "request completed", "server started on port 8080" |
| `WARN` | Algo inesperado que NO es un error, pero merece atención. | "CORS_ORIGINS not set, defaulting to *" |
| `ERROR` | Algo falló y necesita atención. | "failed to connect to database", "unhandled error in handler" |

Al configurar el nivel mínimo (por ejemplo `INFO`), todos los logs de nivel menor (`DEBUG`) se ignoran. Esto te permite tener logs de debug en el código sin que contaminen producción.

#### Handler: cómo se formatean los logs

Un **handler** en `slog` decide el formato de salida. Los dos principales:

```go
// TextHandler — para desarrollo local (legible por humanos)
// Output: time=2026-03-09T10:00:00.000Z level=INFO msg="server started" port=8080
slog.New(slog.NewTextHandler(os.Stdout, opts))

// JSONHandler — para producción (parseable por máquinas)
// Output: {"time":"2026-03-09T10:00:00.000Z","level":"INFO","msg":"server started","port":8080}
slog.New(slog.NewJSONHandler(os.Stdout, opts))
```

En desarrollo quieres texto legible. En producción quieres JSON. Lo controlaremos con una variable de entorno.

#### Attrs: contexto en cada log

La potencia del logging estructurado está en los **atributos** (key-value pairs):

```go
// ❌ Antes (log.Printf) — contexto mezclado en un string
log.Printf("[req:%s] POST /users 201 23ms", requestID)

// ✅ Después (slog) — cada dato es un campo separado
slog.Info("request completed",
    "request_id", requestID,
    "method", "POST",
    "path", "/users",
    "status", 201,
    "duration_ms", 23,
)
```

Cada campo es buscable, filtrable y agregable de forma independiente.

#### Logger con contexto (`With`)

Puedes crear un logger que **siempre incluya ciertos campos**:

```go
// Create a logger that always includes the request_id
logger := slog.Default().With("request_id", "abc-123")

// All subsequent logs from this logger include request_id automatically
logger.Info("processing request")  // {"msg":"processing request","request_id":"abc-123"}
logger.Error("something failed")   // {"msg":"something failed","request_id":"abc-123"}
```

Esto es perfecto para el middleware: creas un logger con el `request_id` y lo metes en el contexto de Gin. Cualquier código downstream que lo lea, automáticamente incluye el request ID.

---

### 4.3 Paso 1 — Crear el paquete de logging

Necesitamos un lugar centralizado para crear y configurar el logger. Siguiendo la arquitectura hexagonal, el logging es **infraestructura** — no pertenece al dominio.

**Archivo**: `internal/infrastructure/logging/logger.go`

```go
package logging

import (
	"log/slog"
	"os"
)

// NewLogger creates a configured slog.Logger based on the environment.
//
// In production (LOG_FORMAT=json), it outputs JSON lines for machine parsing.
// In development (default), it outputs human-readable text.
//
// The log level is controlled by LOG_LEVEL env var (debug, info, warn, error).
// Defaults to "info" if not set.
func NewLogger() *slog.Logger {
	level := parseLevel(os.Getenv("LOG_LEVEL"))

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if os.Getenv("LOG_FORMAT") == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

// parseLevel converts a string level to slog.Level.
// Defaults to INFO if the string is not recognized.
func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
```

**¿Por qué un paquete separado y no configurarlo directamente en `main.go`?**

1. **Reutilización**: Si mañana quieres cambiar el formato, lo haces en un solo lugar.
2. **Testabilidad**: Puedes testear que `NewLogger()` respeta las env vars.
3. **Separación**: El `main.go` solo llama `logging.NewLogger()` y lo inyecta — no sabe los detalles.

---

### 4.4 Paso 2 — Inicializar el logger en `main.go`

Ahora hay que crear el logger al arrancar y establecerlo como el logger global.

```go
package main

import (
	"log/slog"
	"os"

	"ductifact/internal/application/services"
	httpAdapter "ductifact/internal/infrastructure/adapters/inbound/http"
	"ductifact/internal/infrastructure/adapters/outbound/persistence"
	"ductifact/internal/infrastructure/auth"
	"ductifact/internal/infrastructure/logging"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	// --- Logger ---
	logger := logging.NewLogger()
	slog.SetDefault(logger)

	// Database
	db, err := persistence.NewPostgresConnection()
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	// --- User wiring ---
	userRepo := persistence.NewPostgresUserRepository(db)
	userService := services.NewUserService(userRepo)

	// --- Client wiring ---
	clientRepo := persistence.NewPostgresClientRepository(db)
	clientService := services.NewClientService(clientRepo, userRepo)

	// --- Auth wiring ---
	tokenProvider := auth.NewJWTProvider()
	authService := services.NewAuthService(userRepo, tokenProvider)

	// --- HTTP ---
	router := httpAdapter.SetupRoutes(userService, clientService, authService, tokenProvider)

	port := os.Getenv("APP_PORT")
	if port == "" {
		slog.Error("APP_PORT is not set — check your .env file")
		os.Exit(1)
	}

	slog.Info("server starting", "port", port)
	if err := router.Run(":" + port); err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}
```

**Nota clave**: `slog.SetDefault(logger)` hace que cualquier llamada a `slog.Info()`, `slog.Error()`, etc. en cualquier parte del código use nuestro logger configurado. No necesitas pasar el logger como parámetro a cada función (aunque puedes hacerlo si quieres más control).

**¿Por qué `os.Exit(1)` en vez de `log.Fatalf`?** `log.Fatalf` usa el stdlib `log`, que es lo que queremos reemplazar. Con `slog.Error()` + `os.Exit(1)` logramos lo mismo pero con logging estructurado.

---

### 4.5 Paso 3 — Migrar el middleware de logging

Este es el cambio más importante. El middleware de logging es el que más logs genera.

**Archivo**: `internal/infrastructure/adapters/inbound/http/middleware/logger.go`

**Antes:**
```go
log.Printf("[req:%s] %-6s %s %d %v %s",
    requestID, method, path, statusCode, duration, clientIP,
)
```

**Después:**
```go
package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware logs each HTTP request with structured fields.
//
// Output example (JSON format):
//
//	{"time":"2026-03-09T10:00:00Z","level":"INFO","msg":"request completed",
//	 "request_id":"abc-123","method":"POST","path":"/api/v1/users",
//	 "status":201,"duration_ms":23.45,"client_ip":"192.168.1.1"}
//
// This middleware must be registered AFTER RequestIDMiddleware.
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if c.Request.URL.RawQuery != "" {
			path = path + "?" + c.Request.URL.RawQuery
		}

		c.Next()

		status := c.Writer.Status()
		duration := time.Since(start)
		requestID := GetRequestIDFromContext(c)

		// Choose log level based on status code
		attrs := []slog.Attr{
			slog.String("request_id", requestID),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Float64("duration_ms", float64(duration.Microseconds())/1000.0),
			slog.String("client_ip", c.ClientIP()),
		}

		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}

		slog.LogAttrs(c.Request.Context(), level, "request completed", attrs...)
	}
}
```

**¿Qué cambió?**

1. **Niveles automáticos**: Un 500 se loguea como ERROR, un 404 como WARN, un 200 como INFO. Antes todo era igual.
2. **Campos separados**: Cada dato (method, path, status, duration) es un campo independiente, no texto concatenado.
3. **`slog.LogAttrs`**: Esta función es la más eficiente de `slog` — evita allocations innecesarias al pasar los atributos como `slog.Attr` en vez de `any`.
4. **`c.Request.Context()`**: Pasamos el contexto del request, lo que permite que futuras integraciones (trazas, por ejemplo) funcionen automáticamente.

---

### 4.6 Paso 4 — Migrar el middleware de recovery

**Archivo**: `internal/infrastructure/adapters/inbound/http/middleware/recovery.go`

**Después:**
```go
package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// RecoveryMiddleware catches panics and returns a clean 500 error.
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				requestID := GetRequestIDFromContext(c)

				slog.Error("panic recovered",
					"request_id", requestID,
					"panic", r,
					"stack", string(debug.Stack()),
				)

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

El stack trace ahora es un campo `"stack"` que puedes buscar en tu herramienta de logs. Antes era texto pegado al final del Printf.

---

### 4.7 Paso 5 — Migrar el error handler

**Archivo**: `internal/infrastructure/adapters/inbound/http/helpers/error_handler.go`

Cambiar el `log.Printf` del final:

**Antes:**
```go
log.Printf("[ERROR] unhandled error: %v (path: %s %s)", err, c.Request.Method, c.Request.URL.Path)
```

**Después:**
```go
slog.Error("unhandled error",
    "error", err.Error(),
    "method", c.Request.Method,
    "path", c.Request.URL.Path,
)
```

No olvides cambiar el import de `"log"` por `"log/slog"`.

---

### 4.8 Paso 6 — Migrar `main.go` y `connection.go`

#### `connection.go`

**Antes:**
```go
log.Printf("Failed to migrate database: %v", err)
```

**Después:**
```go
slog.Warn("auto-migration failed", "error", err)
```

Usamos `Warn` y no `Error` porque la migración fallida no necesariamente mata la app (las tablas pueden ya existir).

Cambia también el import de `"log"` por `"log/slog"`.

#### `main.go`

Ya se mostró completo en el Paso 2 (sección 4.4). Todos los `log.Fatalf` y `log.Printf` se reemplazan por su equivalente `slog`.

---

### 4.9 Paso 7 — Logging contextual en servicios (opcional)

Esto es un paso **opcional pero recomendado** para cuando quieras más visibilidad dentro de la lógica de negocio.

La idea es que el `request_id` que genera el middleware llegue hasta los servicios, para que si un servicio loguea algo, el log incluya el request ID.

#### Opción A: Usar `slog` directamente en servicios

```go
// In a service method:
func (s *UserService) CreateUser(ctx context.Context, name, email string) (*entities.User, error) {
    slog.InfoContext(ctx, "creating user", "email", email)

    // ... business logic ...

    slog.InfoContext(ctx, "user created", "user_id", user.ID)
    return user, nil
}
```

**Nota**: Para que `slog.InfoContext(ctx, ...)` incluya el request ID automáticamente, necesitarías un handler personalizado que extraiga el request ID del context. Esto es más avanzado y lo puedes hacer más adelante.

#### Opción B (más simple): No loguear en servicios

Los servicios del dominio son lógica pura. Si algo falla, devuelven un error que el handler loguea. Los middlewares ya cubren el 90% de la visibilidad. Los logs en servicios son útiles cuando necesitas debugging fino, pero no son obligatorios.

**Recomendación**: Empieza sin logs en servicios. Agrégalos solo cuando necesites investigar un bug específico.

---

## 5. Pilar 2 — Health check mejorado

### 5.1 ¿Qué es un buen health check?

Un health check no es solo "el proceso está corriendo". Es una **verificación de que el servicio puede hacer su trabajo**. Si tu API no puede hablar con la DB, no es realmente "healthy" aunque responda HTTP.

Un buen health check responde:

| Pregunta | Cómo se verifica |
|----------|-----------------|
| ¿El proceso está vivo? | Si responde HTTP, sí |
| ¿Puede conectar a la DB? | `db.Ping()` |
| ¿Cuánto lleva corriendo? | `time.Since(startTime)` |
| ¿Qué versión es? | Variable de build o env var |

### 5.2 Liveness vs Readiness

En entornos orquestados (Docker Compose con health check, Kubernetes), hay dos tipos de checks:

| Check | Pregunta | Qué pasa si falla |
|-------|----------|-------------------|
| **Liveness** | "¿Estás vivo?" | El orquestador reinicia el contenedor |
| **Readiness** | "¿Puedes recibir tráfico?" | El orquestador deja de enviarle requests (pero no lo mata) |

**¿Por qué son diferentes?** Imagina que tu API arranca pero la DB aún no está lista. El proceso está **vivo** (liveness: ✅) pero no está **listo** para recibir tráfico (readiness: ❌). Kubernetes dejaría de enviarle requests hasta que la DB responda, sin matar el proceso.

Para nuestro caso (un solo servicio con Docker Compose), **un solo endpoint `/health` que verifique la DB es suficiente**. Si más adelante migramos a Kubernetes, separamos en `/healthz` (liveness) y `/readyz` (readiness).

### 5.3 Paso 1 — Definir el port `HealthChecker`

Antes de crear el handler, necesitamos una **interfaz** (port) para verificar la salud de la infraestructura. ¿Por qué? Porque el handler HTTP vive en la capa `inbound`, y **no debe conocer GORM, ni `*sql.DB`, ni ningún detalle de la base de datos**. Si pasásemos `*gorm.DB` al handler, romperíamos la arquitectura hexagonal: la capa HTTP tendría un import directo a la librería de persistencia.

La solución es el mismo patrón que ya usamos con `TokenProvider` y los repositorios: definir una interfaz en `ports/` e implementarla en `persistence/`.

```
┌─────────────────────┐
│   health_handler.go  │  ← solo conoce ports.HealthChecker
│   (inbound/http)     │
└─────────┬───────────┘
          │ usa la interfaz
          ▼
┌─────────────────────┐
│  ports/              │
│  health_checker.go   │  ← interfaz: Ping(ctx) error
└─────────┬───────────┘
          │ implementada por
          ▼
┌─────────────────────┐
│  persistence/        │
│  health_checker.go   │  ← usa *gorm.DB internamente
└─────────────────────┘
```

**Archivo**: `internal/application/ports/health_checker.go`

```go
package ports

import "context"

// HealthChecker is the outbound port for verifying infrastructure health.
// It is defined as an interface so the HTTP layer doesn't depend on
// a specific database library — the implementation lives in infrastructure.
type HealthChecker interface {
	Ping(ctx context.Context) error
}
```

**¿Por qué `context.Context` como parámetro?** Porque `Ping` puede ser una operación de red (TCP al servidor de la DB). Si el request HTTP ya venció (timeout o cancel), no tiene sentido seguir esperando la respuesta del ping. Pasar el contexto permite cancelación y timeouts automáticos.

### 5.4 Paso 2 — Implementar el adapter `PostgresHealthChecker`

Ahora creamos la implementación concreta que usa GORM. Este archivo vive en `persistence/` porque conoce los detalles de la base de datos.

**Archivo**: `internal/infrastructure/adapters/outbound/persistence/health_checker.go`

```go
package persistence

import (
	"context"

	"gorm.io/gorm"
)

// PostgresHealthChecker implements ports.HealthChecker using GORM.
type PostgresHealthChecker struct {
	db *gorm.DB
}

// NewPostgresHealthChecker creates a new PostgresHealthChecker.
func NewPostgresHealthChecker(db *gorm.DB) *PostgresHealthChecker {
	return &PostgresHealthChecker{db: db}
}

// Ping verifies the database connection is alive.
func (h *PostgresHealthChecker) Ping(ctx context.Context) error {
	sqlDB, err := h.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
```

**Nota**: Usamos `PingContext(ctx)` en lugar de `Ping()` para que una cancelación del request HTTP se propague hasta la conexión TCP al servidor de la DB.

### 5.5 Paso 3 — Crear el health handler

Ahora el handler solo depende de la interfaz `ports.HealthChecker`, no de GORM ni de `*sql.DB`.

**Archivo**: `internal/infrastructure/adapters/inbound/http/health_handler.go`

```go
package http

import (
	"net/http"
	"time"

	"ductifact/internal/application/ports"

	"github.com/gin-gonic/gin"
)

// HealthHandler provides health check endpoints for the API.
type HealthHandler struct {
	healthChecker ports.HealthChecker
	startTime     time.Time
}

// NewHealthHandler creates a new HealthHandler.
// Call this at application startup and pass the time the app started.
func NewHealthHandler(healthChecker ports.HealthChecker, startTime time.Time) *HealthHandler {
	return &HealthHandler{
		healthChecker: healthChecker,
		startTime:     startTime,
	}
}

// Check verifies that the API and its dependencies are healthy.
//
// Response 200 (healthy):
//
//	{
//	  "status": "healthy",
//	  "uptime": "2h35m10s",
//	  "database": "connected"
//	}
//
// Response 503 (unhealthy):
//
//	{
//	  "status": "unhealthy",
//	  "uptime": "2h35m10s",
//	  "database": "disconnected",
//	  "error": "connection refused"
//	}
func (h *HealthHandler) Check(c *gin.Context) {
	uptime := time.Since(h.startTime).Round(time.Second).String()

	if err := h.healthChecker.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "unhealthy",
			"uptime":   uptime,
			"database": "disconnected",
			"error":    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "healthy",
		"uptime":   uptime,
		"database": "connected",
	})
}
```

**Decisiones de diseño:**

1. **`ports.HealthChecker` como dependencia**: El handler **no importa GORM**. Solo conoce una interfaz con `Ping(ctx) error`. Esto es la clave de la arquitectura hexagonal — las dependencias apuntan hacia adentro (dominio/ports), nunca hacia afuera (infraestructura concreta).
2. **`c.Request.Context()`**: Propagamos el contexto del request HTTP al ping. Si el cliente cierra la conexión, el ping se cancela automáticamente.
3. **`startTime`**: Lo pasa `main.go` al crear el handler. Es más limpio que un global.
4. **Status 503 Service Unavailable**: Es el código HTTP correcto cuando el servicio no puede funcionar. Los orquestadores (Kubernetes, load balancers) entienden que 503 = "dejá de enviarle tráfico".
5. **No exponemos info sensible**: No mostramos string de conexión, contraseñas, ni detalles internos del error de DB más allá del mensaje.

### 5.6 Paso 4 — Registrar la ruta e inyectar desde `main.go`

En `router.go`, recibimos `ports.HealthChecker` (no `*gorm.DB`). Fíjate que `router.go` no necesita importar `gorm.io/gorm` en absoluto.

**Cambiar la firma de `SetupRoutes`:**

```go
func SetupRoutes(
	healthChecker ports.HealthChecker,
	userService usecases.UserService,
	clientService usecases.ClientService,
	authService usecases.AuthService,
	tokenProvider ports.TokenProvider,
) *gin.Engine {
```

**Registrar el health handler:**

```go
// --- Public routes (no auth required) ---

healthHandler := NewHealthHandler(healthChecker, time.Now())
v1.GET("/health", healthHandler.Check)
```

**En `main.go`**, crear el adapter y pasarlo al router:

```go
// --- Health checker ---
healthChecker := persistence.NewPostgresHealthChecker(db)

// --- HTTP server ---
router := httpAdapter.SetupRoutes(healthChecker, userService, clientService, authService, tokenProvider)
```

**El flujo de inyección completo queda así:**

```
main.go
  ├── persistence.NewPostgresHealthChecker(db)   → crea el adapter concreto
  └── httpAdapter.SetupRoutes(healthChecker, ...) → pasa como ports.HealthChecker
        └── NewHealthHandler(healthChecker, ...)  → el handler solo ve la interfaz
              └── h.healthChecker.Ping(ctx)       → llama al método de la interfaz
```

**¿Por qué `time.Now()` en el router y no en `main.go`?** Porque el router se crea durante el arranque, así que `time.Now()` en ese punto captura el momento de inicio. Si quieres más precisión, puedes capturar el tiempo al principio de `main()` y pasarlo como parámetro.

---

## 6. Bonus — Graceful shutdown

Este paso no está en el roadmap original, pero es una mejora natural cuando implementas observabilidad. Un **graceful shutdown** asegura que cuando el servidor recibe una señal de parada (Ctrl+C, `docker stop`, deploy nuevo), termina los requests en curso antes de apagarse.

**¿Por qué importa?**

Sin graceful shutdown:
1. Llega un request `POST /users` (va a escribir en la DB)
2. Haces `Ctrl+C` → el proceso muere inmediatamente
3. El request queda a medias → datos corruptos, cliente recibe error de conexión

Con graceful shutdown:
1. Llega un request `POST /users`
2. Haces `Ctrl+C` → el servidor deja de aceptar requests nuevos
3. Espera a que el request en curso termine (con un timeout máximo)
4. Cierra la conexión a la DB limpiamente
5. Sale con código 0

**Implementación en `main.go`:**

```go
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ductifact/internal/application/services"
	httpAdapter "ductifact/internal/infrastructure/adapters/inbound/http"
	"ductifact/internal/infrastructure/adapters/outbound/persistence"
	"ductifact/internal/infrastructure/auth"
	"ductifact/internal/infrastructure/logging"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file (ignored if not found, e.g. in Docker/CI)
	_ = godotenv.Load()

	// --- Logger ---
	logger := logging.NewLogger()
	slog.SetDefault(logger)

	// --- Database ---
	db, err := persistence.NewPostgresConnection()
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	// --- User wiring ---
	userRepo := persistence.NewPostgresUserRepository(db)
	userService := services.NewUserService(userRepo)

	// --- Client wiring ---
	clientRepo := persistence.NewPostgresClientRepository(db)
	clientService := services.NewClientService(clientRepo, userRepo)

	// --- Auth wiring ---
	tokenProvider := auth.NewJWTProvider()
	authService := services.NewAuthService(userRepo, tokenProvider)

	// --- Health checker ---
	healthChecker := persistence.NewPostgresHealthChecker(db)

	// --- HTTP server ---
	router := httpAdapter.SetupRoutes(healthChecker, userService, clientService, authService, tokenProvider)

	port := os.Getenv("APP_PORT")
	if port == "" {
		slog.Error("APP_PORT is not set — check your .env file")
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start server in a goroutine so it doesn't block
	go func() {
		slog.Info("server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// --- Graceful shutdown ---
	// Wait for interrupt signal (Ctrl+C) or SIGTERM (docker stop)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	// Give in-flight requests 10 seconds to finish
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	// Close database connection
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}

	slog.Info("server stopped gracefully")
}
```

**Flujo de lo que pasa:**

```
1. main() arranca → crea logger, DB, servicios, healthChecker, router
2. srv.ListenAndServe() corre en una goroutine
3. main() se queda bloqueado en <-quit (esperando señal)
4. Llega SIGINT (Ctrl+C) o SIGTERM (docker stop)
5. srv.Shutdown(ctx) → deja de aceptar requests, espera los en curso
6. Cierra DB
7. Log "server stopped gracefully"
8. Fin
```

---

## 7. Pilar 3 — Métricas (visión futura)

No vamos a implementar métricas ahora, pero es bueno entender qué son y cómo encajan.

### ¿Qué métricas querrías exponer?

| Métrica | Tipo | Ejemplo |
|---------|------|---------|
| Requests totales | Counter | `http_requests_total{method="POST", path="/users", status="201"}` |
| Duración de requests | Histogram | `http_request_duration_seconds{method="GET", path="/health"}` p50=0.002, p99=0.05 |
| Conexiones activas a DB | Gauge | `db_connections_active = 5` |
| Errores | Counter | `http_errors_total{status="500"}` |

### ¿Cómo se implementaría?

1. Agregar dependencia: `github.com/prometheus/client_golang`
2. Crear un middleware que registre métricas por request
3. Exponer endpoint `GET /metrics` (formato Prometheus)
4. Configurar Prometheus para que haga scraping de ese endpoint
5. Visualizar en Grafana

Cuando llegue el momento (posiblemente con CI/CD o al tener un entorno de producción real), es solo agregar un middleware más a la cadena. La arquitectura ya está preparada.

---

## 8. Resumen de archivos modificados y creados

### Archivos nuevos

| Archivo | Propósito |
|---------|----------|
| `internal/infrastructure/logging/logger.go` | Fábrica del logger configurado |
| `internal/application/ports/health_checker.go` | Port (interfaz) para verificar salud de la infraestructura |
| `internal/infrastructure/adapters/outbound/persistence/health_checker.go` | Adapter que implementa `HealthChecker` usando GORM |
| `internal/infrastructure/adapters/inbound/http/health_handler.go` | Health check HTTP handler |

### Archivos modificados

| Archivo | Cambio |
|---------|--------|
| `cmd/api/main.go` | Inicializar slog, crear `PostgresHealthChecker`, graceful shutdown |
| `middleware/logger.go` | `log.Printf` → `slog.LogAttrs` con niveles |
| `middleware/recovery.go` | `log.Printf` → `slog.Error` |
| `helpers/error_handler.go` | `log.Printf` → `slog.Error` |
| `persistence/connection.go` | `log.Printf` → `slog.Warn` |
| `router.go` | Recibir `ports.HealthChecker`, usar `HealthHandler` |

### Variables de entorno nuevas

| Variable | Valores | Default | Propósito |
|----------|---------|---------|----------|
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error` | `info` | Nivel mínimo de logging |
| `LOG_FORMAT` | `json`, `text` | `text` | Formato de salida |

Agrégalas a tu `.env`:

```env
# Logging
LOG_LEVEL=debug
LOG_FORMAT=text
```

En producción (Docker, CI):

```env
LOG_LEVEL=info
LOG_FORMAT=json
```

---

## 9. Checklist de implementación

Sigue este orden. Después de cada paso, verifica que `make test` siga pasando.

- [ ] **Paso 1**: Crear `internal/infrastructure/logging/logger.go`
- [ ] **Paso 2**: Modificar `cmd/api/main.go` — inicializar logger, reemplazar `log.*`
- [ ] **Paso 3**: Migrar `middleware/logger.go` → `slog`
- [ ] **Paso 4**: Migrar `middleware/recovery.go` → `slog`
- [ ] **Paso 5**: Migrar `helpers/error_handler.go` → `slog`
- [ ] **Paso 6**: Migrar `persistence/connection.go` → `slog`
- [ ] **Paso 7**: Verificar que no queda ningún `log.Printf` en el proyecto (`grep -r "log\." --include="*.go" internal/ cmd/`)
- [ ] **Paso 8**: Crear `ports/health_checker.go` (interfaz `HealthChecker`)
- [ ] **Paso 9**: Crear `persistence/health_checker.go` (`PostgresHealthChecker`)
- [ ] **Paso 10**: Crear `health_handler.go` usando `ports.HealthChecker`
- [ ] **Paso 11**: Modificar `router.go` — recibir `ports.HealthChecker`, usar health handler
- [ ] **Paso 12**: Modificar `main.go` — crear `PostgresHealthChecker`, pasarlo al router
- [ ] **Paso 13**: Implementar graceful shutdown en `main.go`
- [ ] **Paso 14**: Agregar `LOG_LEVEL` y `LOG_FORMAT` al `.env`
- [ ] **Paso 15**: `make test` pasa ✅
- [ ] **Paso 16**: Probar manualmente — arrancar la app, hacer requests, verificar formato de logs
- [ ] **Paso 17**: Probar health check con DB arriba (`/health` → 200) y DB abajo (`/health` → 503)

---

## 10. Glosario

| Término | Definición |
|---------|-----------|
| **Logging estructurado** | Logs con campos clave-valor separados en vez de texto plano. Permiten búsqueda y filtrado programático. |
| **`log/slog`** | Paquete de logging estructurado de la stdlib de Go (desde 1.21). |
| **Handler (slog)** | Componente que decide el formato de salida del log (JSON, texto, etc.). No confundir con HTTP handler. |
| **Attr** | Un par clave-valor en un log estructurado. Ej: `slog.String("method", "POST")`. |
| **Level** | La severidad de un log: DEBUG < INFO < WARN < ERROR. |
| **Health check** | Endpoint que verifica si el servicio y sus dependencias funcionan correctamente. |
| **Liveness probe** | "¿Estás vivo?" — Si falla, el orquestador reinicia el proceso. |
| **Readiness probe** | "¿Puedes recibir tráfico?" — Si falla, el orquestador deja de enviarle requests. |
| **Graceful shutdown** | Proceso de parada limpia: dejar de aceptar requests nuevos, terminar los en curso, cerrar conexiones. |
| **503 Service Unavailable** | Código HTTP que indica que el servicio no puede procesar requests temporalmente. |
| **Counter (Prometheus)** | Métrica que solo sube: total de requests, total de errores. |
| **Gauge (Prometheus)** | Métrica que sube y baja: conexiones activas, memoria usada. |
| **Histogram (Prometheus)** | Distribución de valores: latencias, tamaños de respuesta. Calcula percentiles (p50, p99). |
