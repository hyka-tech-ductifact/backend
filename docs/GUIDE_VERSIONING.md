# Guía de Versionado y Git Workflow

## Índice
1. [Las 3 "versiones" que existen](#1-las-3-versiones-que-existen)
2. [¿Cómo se relacionan?](#2-cómo-se-relacionan)
3. [Breaking changes vs non-breaking changes](#3-breaking-changes-vs-non-breaking-changes)
4. [Git: ramas vs tags](#4-git-ramas-vs-tags)
5. [Sincronización entre contracts y backend](#5-sincronización-entre-contracts-y-backend)
6. [Workflow completo: ejemplo real](#6-workflow-completo-ejemplo-real)
7. [¿Necesito ramas por versión de API?](#7-necesito-ramas-por-versión-de-api)
8. [Contract version en el health endpoint](#8-contract-version-en-el-health-endpoint)
9. [Resumen práctico](#9-resumen-práctico)

---

## 1. Las 3 "versiones" que existen

Son 3 cosas distintas que se parecen pero **no están acopladas**:

```
┌─────────────────────────────────────────────────────────────────┐
│  1. API URL version        /api/v1/users/me                    │
│     → Es la versión PÚBLICA de la interfaz                     │
│     → Cambia MUY rara vez (solo con breaking changes)          │
│     → Puede haber v1 y v2 corriendo AL MISMO TIEMPO           │
│                                                                 │
│  2. Spec version           version: 1.2.0 (en openapi.yaml)   │
│     → Es la versión del DOCUMENTO del contrato                 │
│     → Sigue semver (major.minor.patch)                         │
│     → Cambia con cada release (puede ir por la 1.15.0)         │
│                                                                 │
│  3. Git tags/releases      git tag v1.2.0                      │
│     → Marca un PUNTO EN EL TIEMPO del código                   │
│     → Sirve para saber qué código corresponde a qué versión   │
│     → Las ramas son para DESARROLLO, los tags para RELEASES    │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. ¿Cómo se relacionan?

**La URL `/api/v1/` NO es lo mismo que `version: 1.0.0` del spec.**

| | URL version (`/api/v1/`) | Spec version (`1.x.y`) |
|---|---|---|
| **Qué representa** | La interfaz pública | El documento del contrato |
| **Cuándo cambia** | Solo con breaking changes | Con cada cambio a la API |
| **Con qué frecuencia** | Años (o nunca) | Semanas/meses |
| **Ejemplo** | La API puede estar en `/api/v1/` durante 3 años | Y en ese tiempo la spec pasa de `1.0.0` → `1.15.0` |

```
Línea de tiempo:
─────────────────────────────────────────────────────────►

Spec:    1.0.0   1.1.0   1.2.0   1.3.0   ...   1.15.0   2.0.0
URL:     /api/v1/────────────────────────────────────────/api/v2/
                                                          ↑
                                              Solo aquí cambia la URL
                                              (breaking change)
```

---

## 3. Breaking changes vs non-breaking changes

### Non-breaking (spec minor/patch, URL NO cambia)

```
✅ Añadir un campo NUEVO a un response (ej: añadir "created_at" a UserResponse)
✅ Añadir un endpoint NUEVO (ej: DELETE /users/me)
✅ Añadir un campo OPCIONAL a un request
✅ Hacer un campo required → optional
✅ Añadir un nuevo status code a un endpoint
✅ Corregir un bug sin cambiar la forma del JSON
```

### Breaking (spec major, URL CAMBIA a `/api/v2/`)

```
❌ Renombrar un campo (user_id → userId)
❌ Eliminar un campo de un response
❌ Cambiar un tipo (id: string → id: number)
❌ Hacer un campo optional → required en un request
❌ Eliminar un endpoint
❌ Cambiar la estructura de un response (flat → nested)
```

### Regla semver para la spec version

```
MAJOR (1.x.x → 2.0.0) = breaking change     → URL cambia también
MINOR (1.0.x → 1.1.0) = nueva funcionalidad  → URL NO cambia
PATCH (1.0.0 → 1.0.1) = fix/docs             → URL NO cambia
```

---

## 4. Git: ramas vs tags

Las **ramas** son para desarrollo. Los **tags** son para marcar versiones.

```
main ─────●─────●─────●─────●─────●─────●─────── (siempre lo último estable)
          │           │                 │
          │     tag v1.1.0        tag v1.2.0
          │
    tag v1.0.0

feature/add-projects ──●──●──●──┐
                                │ (merge a main)
                                ▼
main ─────────────────────────●──── → tag v1.3.0
```

**Reglas simples:**
- **`main`** = la versión más reciente y estable. Siempre deployable.
- **Tags** marcan releases: `v1.0.0`, `v1.1.0`, `v1.2.0`... Son permanentes.
- **Ramas** son temporales para desarrollar features. Se borran después del merge.
- No necesitas una rama `v1` y otra `v2` (eso es solo para proyectos muy grandes).

---

## 5. Sincronización entre contracts y backend

`contracts/` y `backend/` son **repos separados**, así que van en commits diferentes. La sincronización se hace por **tags con el mismo número de versión**.

```
contracts/  (repo separado)          backend/  (repo separado)
────────────────────────             ────────────────────────
main                                 main
  │                                    │
  ● tag v1.0.0  ◄── mismo número ──►  ● tag v1.0.0
  │                                    │
  ● tag v1.1.0  ◄── mismo número ──►  ● tag v1.1.0
  │   (añadió campo                    │   (implementó el campo
  │    created_at al spec)             │    created_at en Go)
```

### ¿Quién va primero?

**Siempre el contract primero.** Es "contract-first development":

```
1.  contracts/  → PR que actualiza el spec      → merge → tag v1.1.0
                  (ej: añade campo created_at)

2.  backend/    → PR que implementa el cambio   → merge → tag v1.1.0
                  (ej: añade created_at al DTO)
                  Los contract tests validan que coincide con el spec.
```

### ¿Qué pasa si el backend va por delante?

Imagina que añades un campo en el backend pero NO actualizas el contract:

```
contracts/ v1.0.0  → spec dice: UserResponse{id, name, email}
backend/   ???     → código devuelve: UserResponse{id, name, email, created_at}

→ Los contract tests PASAN (tienen los campos requeridos del spec)
→ Pero el frontend NO sabe que created_at existe
→ "Funciona" pero el contract está desactualizado = mal
```

Por eso: **siempre actualiza el contract PRIMERO**.

### ¿Tienen que tener EXACTAMENTE el mismo tag?

Es una convención, no una regla técnica. Lo importante es que puedas saber:

> "¿Qué versión del contract implementa este backend?"

**Opción A: Mismos tags** (simple, recomendado para empezar)
```
contracts/  v1.1.0  ←→  backend/  v1.1.0
```

**Opción B: Tags independientes** (más flexible, para equipos grandes)
```
contracts/  v1.1.0  ←→  backend/  v3.5.0
                         (el backend tiene su propio ritmo de versiones,
                          pero en su README dice: "implements contracts v1.1.0")
```

**Recomendación**: empieza con la Opción A (mismos tags). Es más simple y para un proyecto con un solo developer o equipo pequeño es suficiente.

---

## 6. Workflow completo: ejemplo real

### Escenario: "Añadir campo `created_at` a UserResponse"

Este es un cambio **non-breaking** (añadir campo nuevo a un response).

```
Paso 1 — Contract (primero)
──────────────────────────────────────────────────
repo: contracts/

$ git checkout -b feature/user-created-at
# Editar schemas/user.yaml → añadir created_at
# Editar openapi.yaml → version: 1.0.0 → 1.1.0
$ git add . && git commit -m "feat: add created_at to UserResponse"
$ git push → abrir PR → review → merge a main
$ git tag v1.1.0 && git push --tags


Paso 2 — Backend (después)
──────────────────────────────────────────────────
repo: backend/

$ git checkout -b feature/user-created-at
# Editar UserResponse DTO → añadir CreatedAt
# Editar toUserResponse() mapper
# Actualizar contract_helper.go → UserResponseSchema
# Actualizar contract tests si es necesario
# Ejecutar: make test-contract → PASAN ✅
$ git add . && git commit -m "feat: add created_at to UserResponse (contracts v1.1.0)"
$ git push → abrir PR → review → merge a main
$ git tag v1.1.0 && git push --tags
```

### Escenario: "Renombrar `user_id` → `userId` en ClientResponse"

Este es un **breaking change**.

```
Paso 1 — Decisión
──────────────────────────────────────────────────
¿Realmente necesitamos este breaking change?
Si sí: spec version 1.x.x → 2.0.0, URL /api/v1/ → /api/v2/

Paso 2 — Contract
──────────────────────────────────────────────────
$ git checkout -b breaking/client-userid-rename
# Editar schemas/client.yaml → user_id → userId
# Editar openapi.yaml → version: 2.0.0
# Editar openapi.yaml → server url: /api/v2
$ git commit -m "feat!: rename user_id to userId in ClientResponse (BREAKING)"
  (nota: el "!" en el commit indica breaking change en Conventional Commits)
$ merge → tag v2.0.0

Paso 3 — Backend
──────────────────────────────────────────────────
$ git checkout -b breaking/client-userid-rename
# Añadir grupo /api/v2 en router.go (v1 puede seguir existiendo)
# Actualizar DTOs para v2
# Actualizar contract tests
$ merge → tag v2.0.0
```

---

## 7. ¿Necesito ramas por versión de API?

**No.** Este es un error común. Las versiones de API no son ramas.

```
❌  INCORRECTO:
    rama "v1" → código de /api/v1/
    rama "v2" → código de /api/v2/

✅  CORRECTO:
    rama "main" → tiene AMBAS versiones
    El router.go sirve /api/v1/ y /api/v2/ al mismo tiempo
```

```go
// router.go — ambas versiones conviven en el mismo código
v1 := r.Group("/api/v1")
v1.GET("/users/me", userHandlerV1.GetMe)   // user_id en response

v2 := r.Group("/api/v2")
v2.GET("/users/me", userHandlerV2.GetMe)   // userId en response
```

Las ramas son para **desarrollo** (feature branches), no para versiones de la API.

---

## 8. Contract version en el health endpoint

### ¿Por qué?

Cuando hay **apps móviles** (iOS/Android), no puedes obligar a todos los usuarios a actualizar al mismo tiempo. Hay versiones antiguas "en la naturaleza" que siguen haciendo requests a tu API.

El health endpoint incluye `contract_version` para que los clientes puedan saber **qué versión del contrato está implementando el backend** sin necesidad de autenticarse.

### Response

```json
GET /api/v1/health

{
  "status": "healthy",
  "uptime": "2h35m10s",
  "database": "connected",
  "contract_version": "1.0.0"
}
```

### Configuración

La versión del contrato se lee de la variable de entorno `CONTRACT_VERSION`:

```bash
# En .env o en el entorno de deploy
CONTRACT_VERSION=1.0.0
```

Si no está definida, el valor por defecto es `"1.0.0"`.

### ¿Cómo lo usa una app móvil?

```
App v2.0 (necesita contract >= 1.3.0)
  │
  ├─ GET /api/v1/health
  │    → contract_version: "1.5.0"   ✅ compatible, continuar normalmente
  │    → contract_version: "1.1.0"   ⚠️ puede que falten campos, mostrar aviso
  │    → contract_version: "2.0.0"   ❌ breaking change, pedir actualización
  │
  └─ La lógica de compatibilidad la decide el cliente (la app)
```

### Reglas de compatibilidad

```
Mismo MAJOR → compatible (campos nuevos se ignoran si no se conocen)
Mayor MINOR → el client puede tener campos que no existen aún en el server
Mayor MAJOR → INCOMPATIBLE, la app debería mostrar "actualiza la app"
```

### ¿Cuándo actualizar CONTRACT_VERSION?

Cada vez que hagas deploy de un cambio que modifica la API:

```
1. Actualizar el spec en contracts/ (openapi.yaml info.version)
2. Implementar en backend/
3. Actualizar CONTRACT_VERSION en el .env de producción al deployar
4. El valor debe coincidir con el spec version de openapi.yaml
```

---

## 9. Resumen práctico

### Tabla de decisiones

```
¿Qué hice?                               Spec version        URL        Git
─────────────────────────────────────── ─────────────────── ────────── ──────────
Añadí endpoint GET /users/me/stats       1.0.0 → 1.1.0      /api/v1/   tag v1.1.0
Añadí campo created_at a UserResponse    1.1.0 → 1.2.0      /api/v1/   tag v1.2.0
Corregí un typo en la descripción        1.2.0 → 1.2.1      /api/v1/   tag v1.2.1
Renombré user_id → userId (BREAKING)     1.2.1 → 2.0.0      /api/v2/   tag v2.0.0
```

### Checklist para cada cambio

```
□ ¿Es breaking change?
  → Sí: incrementar MAJOR de la spec, nueva URL /api/vN/
  → No: incrementar MINOR o PATCH

□ ¿Actualicé el contract PRIMERO?
  → Sí: continuar con el backend
  → No: parar y actualizar el contract

□ ¿Los tags coinciden entre repos?
  → contracts/ tag vX.Y.Z
  → backend/   tag vX.Y.Z (o referencia al contract version en el commit)

□ ¿Los contract tests pasan?
  → make test-contract → ✅
```

### Convención de commits (Conventional Commits)

```
feat: add created_at to UserResponse           → MINOR (1.1.0 → 1.2.0)
fix: correct email validation                  → PATCH (1.2.0 → 1.2.1)
feat!: rename user_id to userId (BREAKING)     → MAJOR (1.2.1 → 2.0.0)
docs: update API description                   → PATCH (1.2.1 → 1.2.2)
```

### Resumen visual

```
contracts/ repo                           backend/ repo
──────────────                            ──────────────
main ──●──●──●──                          main ──●──●──●──
       │     │                                   │     │
  v1.0.0  v1.1.0                            v1.0.0  v1.1.0
                                                      │
   spec version: 1.1.0                    commit msg: "feat: ... (contracts v1.1.0)"
   URL: /api/v1/ (no cambió)              contract tests: PASS ✅

Los tags se llaman igual.
El contract siempre va primero.
La URL /api/v1/ no cambia hasta un breaking change.
```
