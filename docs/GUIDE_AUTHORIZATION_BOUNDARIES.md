# Guía de autorización: negocio, colaboración y operaciones internas

Esta guía explica la decisión tomada para Ductifact y, sobre todo, el modelo
mental que hay detrás. El objetivo es evitar mezclar tres problemas que parecen
similares porque todos responden a «¿puede hacer esto?», pero pertenecen a
fronteras distintas.

## 1. La decisión actual

Ductifact no tendrá por ahora roles globales `admin`, `user` o `readonly` en la
entidad `User`.

El modelo queda dividido así:

1. **API de negocio actual**: autenticación con JWT y ownership de recursos.
2. **Colaboración futura**: roles `owner`, `editor` y `viewer` asignados por
   proyecto mediante una membresía.
3. **Operaciones internas futuras**: identidad y herramientas separadas para
   soporte, investigación de incidencias y reparación controlada de datos.

No se ha perdido una funcionalidad publicada. El RBAC global era un prototipo
local todavía no liberado; retirarlo ahora evita convertir una suposición en
una parte permanente del contrato y de la base de datos.

## 2. Tres preguntas distintas

| Capa | Pregunta | Ejemplo |
|------|----------|---------|
| Autenticación | ¿Quién eres? | El JWT identifica a `user_id=123` |
| Autorización de negocio | ¿Qué puedes hacer sobre este recurso concreto? | Eres `editor` del proyecto A, pero `viewer` del B |
| Control operacional | ¿Qué acción excepcional puede ejecutar un operador y bajo qué controles? | Soporte reintenta una operación fallida con motivo y auditoría |

La autenticación no concede acceso por sí sola. Un JWT válido identifica al
actor; después, cada caso de uso todavía debe comprobar su relación con el
recurso solicitado.

## 3. Qué protege hoy la API

Actualmente todos los usuarios del producto tienen las mismas capacidades
generales. La diferencia de acceso procede de la propiedad de los datos:

```text
User
└── Client
    └── Project
        └── Order
            └── Piece
```

El `user_id` autenticado se compara con la cadena de ownership. Un usuario no
puede consultar o modificar los clientes, proyectos, órdenes o piezas de otro.
Las definiciones de pieza personalizadas también están aisladas por usuario.

Este modelo es suficiente mientras cada proyecto tenga una sola persona con
acceso. Añadir un rol global no solucionaría correctamente la colaboración: un
usuario podría necesitar permisos diferentes en proyectos diferentes.

## 4. El futuro rol pertenece a una membresía

La relación que se quiere modelar es muchos-a-muchos:

```text
User 1 ──< ProjectMembership >── 1 Project
                    │
                    └── role: owner | editor | viewer
```

Una futura `ProjectMembership` tendría, como mínimo:

- `project_id`
- `user_id`
- `role`
- `created_at` y `updated_at`
- opcionalmente `invited_by`
- una restricción única sobre `(project_id, user_id)`

Por eso `role` no debe estar en `User`: allí solo podría expresar una propiedad
global. En la membresía expresa una relación concreta.

Un mismo usuario podría ser:

- `owner` del proyecto de su empresa;
- `editor` de un proyecto compartido por un compañero;
- `viewer` de un proyecto que solo necesita revisar.

### Matriz inicial recomendada

La matriz definitiva se fijará en el ADR del punto 18.1, pero un buen punto de
partida es:

| Acción | `owner` | `editor` | `viewer` |
|--------|:-------:|:--------:|:--------:|
| Ver proyecto, órdenes y piezas | Sí | Sí | Sí |
| Editar datos del proyecto | Sí | Sí | No |
| Crear, editar o borrar órdenes y piezas | Sí | Sí | No |
| Borrar el proyecto | Sí | No | No |
| Invitar, expulsar o cambiar roles | Sí | No | No |
| Transferir ownership | Sí | No | No |

`owner`, `editor` y `viewer` son nombres de roles, pero técnicamente representan
un conjunto de permisos **dentro de un proyecto**. No forman una pirámide global
de usuarios de toda la plataforma.

### Herencia de acceso

Una orden y una pieza no necesitan membresías independientes si siempre
pertenecen a un proyecto. Su autorización puede heredarse:

```text
piece_id → order_id → project_id → ProjectMembership(user_id, project_id)
```

Esto evita que los permisos de cada nivel se contradigan y simplifica el modelo.
Si algún día una orden pudiera compartirse de manera independiente, entonces sí
habría que revisar esa frontera.

## 5. La autorización debe estar cerca del caso de uso

Un middleware de ruta es útil para validar el JWT, porque esa regla es igual
para todas las peticiones protegidas. No basta para autorizar recursos, porque
la decisión depende del identificador solicitado y del estado de PostgreSQL.

El flujo futuro recomendado es:

```text
HTTP + JWT
   │
   ├─ middleware: autentica y obtiene user_id
   │
   └─ caso de uso: carga el recurso y su project_id
          │
          ├─ política: comprueba membresía + acción requerida
          │
          ├─ ejecuta invariantes de negocio
          │
          └─ repositorio: persiste el cambio
```

La política centraliza la matriz para no repetir `if role == ...` en cada
handler. El caso de uso sigue siendo la autoridad porque conoce la acción real:
leer una pieza, borrar una orden o administrar miembros no son equivalentes.

## 6. Límites de lo que se comparte

Compartir un proyecto debe dar acceso al proyecto, sus órdenes y sus piezas. No
debe abrir automáticamente todos los recursos relacionados del propietario.

Dos límites importantes para el modelo actual son:

- **Client**: el colaborador no obtiene por defecto acceso al registro completo
  del cliente del propietario. Si la interfaz necesita mostrar su nombre, se
  puede exponer una proyección mínima y explícita.
- **PieceDefinition**: poder ver una pieza no debe permitir navegar por toda la
  biblioteca privada de definiciones del propietario. La respuesta puede incluir
  únicamente los campos referenciados necesarios o un snapshot seguro.

Estas decisiones evitan el «acceso transitivo accidental»: tener permiso sobre
A no debe revelar automáticamente todo B que esté relacionado con A.

## 7. Invariantes importantes de membresía

No basta con almacenar el rol. Las operaciones de membresía necesitan reglas:

- todo proyecto debe conservar al menos un `owner`;
- el último owner no puede salir, rebajarse a `editor` ni ser expulsado sin
  transferir antes el ownership;
- aceptar dos veces la misma invitación debe ser idempotente;
- dos cambios concurrentes no deben dejar un proyecto sin owner;
- una invitación debe expirar, poder revocarse y estar ligada al proyecto y al
  destinatario correctos;
- eliminar un usuario requiere decidir qué sucede con los proyectos que posee.

Las reglas que dependen de concurrencia deben protegerse en una transacción y,
cuando corresponda, con restricciones o bloqueos en PostgreSQL. Comprobarlas
solo en memoria deja una ventana entre «leer» y «escribir».

## 8. `401`, `403` y `404`

- `401 Unauthorized`: no existe una autenticación válida.
- `403 Forbidden`: el actor está autenticado, el recurso es visible, pero la
  acción concreta no está permitida.
- `404 Not Found`: el recurso no existe o se oculta su existencia para evitar
  enumeración entre proyectos.

No hay una única elección universal entre `403` y `404` para recursos ajenos.
Lo importante es decidirla en el ADR, aplicarla de forma consistente y probar
el aislamiento entre proyectos.

## 9. Qué es el plano de operaciones internas

Soporte no es un usuario final «más poderoso». Es otro tipo de actor, con otro
propósito, otra autenticación y un nivel de riesgo mayor.

Ejemplos legítimos de operaciones internas:

- consultar el estado técnico de una operación fallida;
- reenviar un correo o reintentar un job de forma idempotente;
- desbloquear un flujo conocido mediante una acción explícita;
- reparar datos afectados por un bug con un comando versionado;
- exportar evidencia limitada para investigar un ticket.

No se recomienda un endpoint genérico que permita editar cualquier fila. Cada
acción operacional debería tener nombre, alcance e invariantes claros.

### Bug de software frente a incidencia operacional

Si existe un bug, primero se corrige el código que lo causa. Los datos que ya
quedaron afectados pueden repararse con:

- una migración de datos, si todos necesitan la misma transformación;
- un comando o job controlado, si hay que seleccionar casos y generar un
  informe;
- una acción interna específica, si es un flujo recurrente que soporte necesita
  ejecutar de forma segura.

El usuario final no recibe acceso SQL ni una capacidad arbitraria. Recibe, si
realmente hace falta, una herramienta limitada que ejecuta el sistema bajo sus
propias reglas.

## 10. Empezar en el monolito, separar la frontera

Separación lógica y microservicio no son sinónimos. La recomendación inicial es
mantener un monolito modular y crear una frontera operacional clara:

- un binario o router interno distinto;
- red privada, sin exposición pública;
- identidad de operador separada, idealmente SSO y MFA;
- audiencia de token distinta a la API de producto;
- casos de uso operacionales explícitos;
- auditoría obligatoria y append-only;
- reutilización de las reglas de aplicación y dominio.

Esto ofrece la mayor parte del aislamiento sin asumir todavía el coste de red,
despliegue y consistencia de un microservicio.

### Cuándo tendría sentido extraerlo

La extracción se evalúa cuando exista una razón observable:

- un equipo distinto es propietario del plano operacional;
- necesita un ciclo de despliegue o escalado independiente;
- la frontera de seguridad exige aislamiento de proceso o infraestructura;
- hay suficientes workflows estables para justificar el coste.

Si se extrae, el servicio operacional puede poseer entidades como
`SupportCase`, `OperationRun` o `PrivilegedAuditEvent`. Los proyectos, órdenes y
piezas siguen perteneciendo al servicio de negocio. El servicio operacional
debe invocar su API o comandos autorizados; no debe escribir directamente sus
tablas compartidas.

## 11. Controles mínimos para una operación privilegiada

Una acción interna debería registrar:

- quién la ejecutó;
- qué recurso afectó;
- qué acción exacta se pidió;
- motivo y, cuando exista, identificador del ticket;
- estado anterior y posterior, con secretos y PII sensible enmascarados;
- fecha, resultado e identificador de correlación.

Para acciones de alto impacto también pueden añadirse `dry-run`, aprobación por
otra persona, caducidad temporal del permiso y modo *break glass*. Este último
es acceso excepcional de emergencia: debe ser raro, corto y especialmente
auditable.

## 12. Orden de implementación futuro

El roadmap separa ahora los dos trabajos:

1. El punto 18 implementará colaboración por proyecto: ADR, entidad de
   membresía, migración/backfill, políticas, API, invitaciones y pruebas.
2. El punto 25 empezará solo cuando existan casos operacionales reales y
   repetidos: threat model, identidad de operador, auditoría, comandos y una API
   interna si aporta valor.

No es necesario diseñar hoy una jerarquía de `superadmin`, `admin` y `support`.
Primero se describen acciones reales; después se agrupan en capacidades de
mínimo privilegio si la repetición demuestra que hace falta.

## 13. Conceptos clave

- **Ownership**: relación de propiedad entre actor y recurso.
- **RBAC**: permisos agrupados por roles. Puede ser útil, pero el rol necesita
  un ámbito; aquí el ámbito futuro será un proyecto.
- **ABAC**: decisión basada en atributos del actor, recurso y contexto. Una
  política como «es editor de este proyecto» combina membresía y atributos.
- **Scope**: frontera dentro de la que un permiso es válido.
- **Policy**: componente que responde si una acción está permitida.
- **Invariant**: regla que debe ser cierta antes y después de cada operación.
- **Control plane**: interfaz administrativa u operacional que gobierna el
  sistema, separada de las operaciones normales del producto.
- **Least privilege**: conceder únicamente las capacidades necesarias durante
  el tiempo necesario.
- **Audit log**: historial inmutable de acciones relevantes; no es lo mismo que
  un log de depuración que puede rotar o contener mensajes informales.

La idea más importante es esta: un nombre como «admin» no define por sí solo un
buen modelo de autorización. Primero hay que responder **sobre qué recurso**, en
**qué contexto** y para **qué acción** se concede la capacidad.
