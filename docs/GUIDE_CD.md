# Guía de CD — Despliegue en Debian 12 con Cloudflare Tunnel

> **Fase 7.3 + 7.4 del Roadmap**
> Desplegar automáticamente a **staging** desde `main` y a **producción** desde tags `v*`.
> Ambos entornos corren en el mismo servidor (`jcapsule.work`) con Cloudflare Tunnel.

---

## 1. Visión general

El flujo completo de CI/CD sigue el modelo definido en `CONTRIBUTING.md`:

```
─── Flujo normal (features/fixes) ─────────────────────

  PR mergeada → main
       │
       ▼
  CI (GitHub Actions) → lint + tests
       │
  Todo pasa? ✅
       │
       ▼
  CD → Build imagen → Deploy a STAGING
       │
       ▼
  QA valida en staging (staging-api.ductifact.jcapsule.work)
       │
  Aprobado? ✅
       │
       ▼
  Tag v0.4.0 → release branch → CD → Deploy a PRODUCCIÓN

─── Flujo hotfix (bug urgente en producción) ──────────

  PR mergeada → release
       │
       ▼
  CI → lint + tests
       │
  Tag v0.4.1 en release
       │
       ▼
  CD → Deploy a PRODUCCIÓN (sin código no validado de main)
       │
  Merge release → main (backport del fix)
```

### ¿Qué es CD?

**CD (Continuous Deployment)** automatiza el despliegue después de que CI pasa:

- **Staging**: se despliega automáticamente cada vez que un PR se mergea a `main`. No requiere intervención manual. Accesible en `staging-api.ductifact.jcapsule.work`.
- **Producción**: se despliega cuando se pushea un **tag** `v*`. El tag lo creas tú manualmente después de que QA valide staging (ver `CONTRIBUTING.md` §6). Accesible en `api.ductifact.jcapsule.work`.

Esto es **Continuous Delivery** (no Deployment) para producción — hay un paso manual intencional (el tag) que actúa como puerta de aprobación.

---

## 2. Arquitectura en el servidor

El servidor Debian 12 corre **dos entornos** (staging y producción) aislados con redes Docker separadas. **Cloudflare Tunnel** se encarga de TLS y de exponer los servicios a internet sin necesidad de abrir puertos en el servidor. **Caddy** (que ya tienes instalado) actúa como reverse proxy para ductifact y el resto de tus servicios.

```
Internet
    │
    ▼
  Cloudflare Edge (TLS termination, WAF, DDoS protection)
    │
    ▼
  Cloudflare Tunnel (conexión saliente desde tu servidor)
    │
    ▼
  cloudflared (daemon en el servidor)
    │
    ▼
  Caddy (reverse proxy, ya instalado en el host)
    │
    ├── api.ductifact.jcapsule.work     → app-prod (:8090)
    ├── staging-api.ductifact.jcapsule.work  → app-staging (:8091)
    ├── grafana.ductifact.jcapsule.work  → Grafana (:3000) (opcional)
    └── ductifact.jcapsule.work          → (futuro frontend)

  ┌─── Entorno PRODUCTION ────────────────────────┐
  │  app-prod     (Go, :8090) ── contenedor Docker │
  │  postgres-prod (:5432)    ── contenedor Docker │
  │  prometheus   (:9090)     ── contenedor Docker │
  │  grafana      (:3000)     ── contenedor Docker │
  └────────────────────────────────────────────────┘

  ┌─── Entorno STAGING ───────────────────────────┐
  │  app-staging     (Go, :8091) ── contenedor Docker │
  │  postgres-staging (:5433)    ── contenedor Docker │
  └────────────────────────────────────────────────┘
```

### ¿Qué es Cloudflare Tunnel?

Normalmente, exponer un servicio requiere abrir puertos (80, 443) en el firewall y gestionar certificados TLS. Cloudflare Tunnel invierte este modelo:

1. El daemon `cloudflared` en tu servidor establece una **conexión saliente** a Cloudflare.
2. Cloudflare recibe el tráfico de los usuarios, termina TLS y lo reenvía por el túnel.
3. Tu servidor **no necesita abrir ningún puerto al público** (ni 80, ni 443).

| Ventaja | Explicación |
|---------|-------------|
| Sin puertos abiertos | El firewall bloquea **todo** el tráfico entrante de internet. SSH también va por el túnel. Superficie de ataque: cero puertos públicos. |
| TLS automático | Cloudflare gestiona los certificados. No necesitas Certbot ni Let's Encrypt. |
| DDoS protection | Cloudflare filtra ataques antes de que lleguen a tu servidor. |
| WAF incluido | Web Application Firewall que bloquea ataques comunes (SQL injection, XSS). |
| IP oculta | Tu IP real nunca se expone. Los atacantes no pueden escanear puertos directamente. |

### ¿Por qué Caddy como reverse proxy?

Ya tienes Caddy instalado en el servidor para otros servicios. Reutilizarlo para ductifact tiene sentido:
- **Ya está corriendo** — no añades otro componente, solo entradas al Caddyfile existente
- Enruta peticiones a los contenedores correctos según el subdominio
- Bloquea rutas internas (`/metrics`, `/health`) con directivas simples
- Pasa la IP real del cliente desde Cloudflare a tu API
- Cuando añadas el frontend, solo es otra entrada en el Caddyfile

> **¿Por qué no un Nginx containerizado?** Porque ya tienes Caddy en el host — añadir Nginx sería doble reverse proxy (Cloudflare → Caddy → Nginx → app) sin beneficio. Menos componentes = menos cosas que romper.

---

## 3. Estructura de archivos

`backend/` e `infra/` son **repositorios independientes**. Cada uno tiene su propia carpeta `.github/` si necesita workflows.

```
backend/                             ← repositorio Git del backend
├── .github/
│   └── workflows/
│       ├── ci.yml                   ← ya lo tienes de la guía CI
│       └── cd.yml                   ← NUEVO: build + push imagen + deploy
├── Dockerfile
└── ...

infra/                               ← repositorio Git de infraestructura (SEPARADO)
├── docker-compose.prod.yml          ← entorno de producción
├── docker-compose.staging.yml       ← entorno de staging
├── .env.prod.example
├── .env.staging.example
├── scripts/
│   └── deploy.sh                    ← script de deploy (llamado desde CD)
└── prometheus/
    └── prometheus.yml
```

> **¿Por qué repos separados?** Porque cuando añadas un servicio `frontend/`, la infra orquestará ambos servicios. Si `infra/` estuviera dentro de `backend/`, el frontend no podría usarlo. Repos separados permiten que la infraestructura evolucione independientemente de cada servicio.

---

## 4. Setup inicial del servidor (una sola vez)

Esto lo haces una vez. Después, los deploys son automáticos.

### 4.1 Requisitos previos

Tu servidor Debian 12 ya está funcionando con:
- Host: `jcapsule.work`
- Cloudflare Tunnel configurado (el tráfico ya llega al servidor)
- Acceso SSH a través de Cloudflare Tunnel vía `ssh.jcapsule.work`

> **Nota**: Como tu SSH va por Cloudflare Tunnel, necesitas `cloudflared` instalado también en tu **máquina local** y esta entrada en tu `~/.ssh/config`:
>
> ```
> Host ssh.jcapsule.work
>     ProxyCommand cloudflared access ssh --hostname %h
> ```
>
> Así todos los comandos `ssh`, `scp` y `ssh-copy-id` que aparecen en esta guía funcionan transparentemente a través del túnel.

Si aún no tienes acceso SSH con key, genera una:

```bash
# En tu máquina local
ssh-keygen -t ed25519 -C "tu@email.com"
ssh-copy-id tu-usuario@ssh.jcapsule.work
```

### 4.2 Configuración inicial del servidor

Conéctate al servidor:

```bash
ssh tu-usuario@ssh.jcapsule.work
```

Ejecuta estos comandos para preparar el servidor:

```bash
# 1. Actualizar el sistema
sudo apt update && sudo apt upgrade -y

# 2. Crear usuario para deploys (no usar root ni tu usuario personal)
sudo adduser deploy --disabled-password --gecos ""
sudo usermod -aG sudo deploy

# 3. Configurar SSH para el usuario deploy
sudo mkdir -p /home/deploy/.ssh
sudo cp ~/.ssh/authorized_keys /home/deploy/.ssh/
sudo chown -R deploy:deploy /home/deploy/.ssh
sudo chmod 700 /home/deploy/.ssh
sudo chmod 600 /home/deploy/.ssh/authorized_keys

# 4. Instalar Docker
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker deploy

# 5. Instalar Docker Compose plugin
sudo apt install -y docker-compose-plugin

# 6. Configurar firewall — solo red local
#    (Cloudflare Tunnel no necesita puertos abiertos, ni siquiera SSH)
sudo apt install -y ufw
sudo ufw allow from 192.168.1.0/24   # ← subred LAN principal
sudo ufw allow from 192.168.0.0/24   # ← subred LAN secundaria
sudo ufw --force enable

# 7. Crear directorio para backups
sudo mkdir -p /home/deploy/backups
sudo chown deploy:deploy /home/deploy/backups

# 8. Verificar
docker --version
docker compose version
sudo ufw status
```

**¿Qué hace cada paso?**

| Paso | Explicación |
|------|-------------|
| `adduser deploy` | Creas un usuario sin privilegios para los despliegues. Nunca uses `root` para correr tu aplicación — si alguien compromete tu app, no tendrá acceso root al servidor. |
| `--disabled-password` | El usuario `deploy` no tiene contraseña. Solo se puede acceder por SSH key. Más seguro. |
| `usermod -aG docker deploy` | Permite al usuario `deploy` ejecutar Docker sin `sudo`. |
| `ufw allow from 192.168.1.0/24` | Permite que dispositivos de tu red local accedan a todos los puertos (Samba, Jellyfin, SSH local, etc.). Ajusta a tu subred real — compruébala con `ip -4 addr show`. Desde internet **todos los puertos están cerrados** — cero superficie de ataque. SSH llega por Cloudflare Tunnel (conexión saliente → localhost), así que no necesita puerto abierto. |

> **Importante**: después de añadir `deploy` al grupo `docker`, cierra la sesión y vuelve a conectar como `deploy` para que el cambio de grupo surta efecto.

### 4.3 Instalar cloudflared

`cloudflared` es el daemon que mantiene el túnel con Cloudflare. Si aún no lo tienes instalado:

```bash
# Como root o con sudo
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb \
  -o /tmp/cloudflared.deb
sudo dpkg -i /tmp/cloudflared.deb
rm /tmp/cloudflared.deb

# Verificar
cloudflared --version
```

### 4.4 Configurar Cloudflare Tunnel

Si ya tienes un túnel configurado, puedes saltar al paso de **añadir los subdominios**. Si no:

```bash
# 1. Autenticarse con Cloudflare
cloudflared tunnel login
# Se abrirá un navegador. Selecciona tu dominio jcapsule.work.

# 2. Crear el túnel
cloudflared tunnel create ductifact
# Anota el UUID del túnel (ej: a1b2c3d4-e5f6-...)
# Esto genera un archivo de credenciales en ~/.cloudflared/<UUID>.json

# 3. Configurar DNS en Cloudflare para los subdominios
cloudflared tunnel route dns ductifact api.ductifact.jcapsule.work
cloudflared tunnel route dns ductifact staging-api.ductifact.jcapsule.work
cloudflared tunnel route dns ductifact grafana.ductifact.jcapsule.work   # opcional
```

**¿Qué hace cada paso?**

| Paso | Explicación |
|------|-------------|
| `tunnel login` | Obtiene un certificado que autoriza a tu servidor a crear túneles bajo tu dominio. |
| `tunnel create` | Crea un túnel con nombre `ductifact`. Genera un UUID único y unas credenciales. |
| `tunnel route dns` | Crea un registro CNAME en Cloudflare que apunta cada subdominio al túnel. Así `api.ductifact.jcapsule.work` → túnel → tu servidor. |

### 4.5 Crear archivo de configuración de cloudflared

```bash
# Si cloudflared lo instalaste como servicio del sistema:
sudo mkdir -p /etc/cloudflared

# Crear el config
sudo nano /etc/cloudflared/config.yml
```

Contenido de `config.yml`:

```yaml
tunnel: <UUID-DE-TU-TUNEL>
credentials-file: /etc/cloudflared/<UUID-DE-TU-TUNEL>.json

ingress:
  # ── Producción ─────────────────────────────────────────────
  - hostname: api.ductifact.jcapsule.work
    service: http://localhost:80

  # ── Staging ────────────────────────────────────────────────
  - hostname: staging-api.ductifact.jcapsule.work
    service: http://localhost:80

  # ── Grafana (opcional) ─────────────────────────────────────
  - hostname: grafana.ductifact.jcapsule.work
    service: http://localhost:80

  # ── Futuro frontend ───────────────────────────────────────
  # - hostname: ductifact.jcapsule.work
  #   service: http://localhost:80

  # ── Catch-all (obligatorio, debe ser el último) ────────────
  - service: http_status:404
```

> **Nota**: Todas las rutas HTTP apuntan a Caddy en `localhost:80`. Caddy se encarga de distinguir por hostname y enviar cada petición al contenedor Docker correcto. La ruta SSH apunta directamente al daemon SSH del servidor (`localhost:22`). Así cloudflared solo tiene una responsabilidad: reenviar tráfico del túnel al servicio correcto.

Copia las credenciales del túnel:

```bash
# Si las credenciales están en tu home, cópialas
sudo cp ~/.cloudflared/<UUID>.json /etc/cloudflared/
```

Instala cloudflared como servicio del sistema:

```bash
sudo cloudflared service install
sudo systemctl enable cloudflared
sudo systemctl start cloudflared

# Verificar que está corriendo
sudo systemctl status cloudflared
```

### 4.6 Crear la estructura en el servidor

Como usuario `deploy`:

```bash
ssh deploy@ssh.jcapsule.work

# Crear directorios
mkdir -p ~/ductifact
mkdir -p ~/backups
```

### 4.7 Configurar SSH key para GitHub Actions

GitHub Actions necesita poder hacer SSH al servidor para desplegar. Genera una key **dedicada** para CI/CD:

```bash
# En tu máquina local (no en el servidor)
ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/ductifact_deploy
```

Esto genera dos archivos:
- `~/.ssh/ductifact_deploy` — la clave **privada** (la sube a GitHub como Secret)
- `~/.ssh/ductifact_deploy.pub` — la clave **pública** (la pones en el servidor)

```bash
# Copiar la clave pública al servidor
ssh-copy-id -i ~/.ssh/ductifact_deploy.pub deploy@ssh.jcapsule.work
```

### 4.8 Crear Service Token en Cloudflare Access

GitHub Actions necesita autenticarse ante Cloudflare para usar el túnel SSH. Para eso creas un **Service Token** (credencial máquina-a-máquina):

1. Ve a **Cloudflare Dashboard → Zero Trust → Access → Service Auth → Service Tokens**
2. Click en **Create Service Token**
3. Nombre: `github-actions-deploy`
4. Cloudflare te da dos valores:
   - **Client ID** — algo como `abc123.access`
   - **Client Secret** — solo se muestra una vez, cópialo

Después, crea una **Access Application** que permita este token:

1. Ve a **Zero Trust → Access → Applications → Add an application**
2. Tipo: **Self-hosted**
3. Application name: `SSH Deploy`
4. Subdomain: `ssh.jcapsule.work`
5. En **Policies**, crea una política:
   - Policy name: `Allow GitHub Actions`
   - Action: **Service Auth**
   - Include: **Service Token** → selecciona `github-actions-deploy`

Esto autoriza al Service Token a acceder a `ssh.jcapsule.work` a través del túnel.

> **¿Qué es un Service Token?** Es un par de credenciales (ID + Secret) diseñado para que máquinas (no personas) se autentiquen ante Cloudflare Access. `cloudflared` envía estas credenciales como headers HTTP (`CF-Access-Client-Id` y `CF-Access-Client-Secret`) al establecer la conexión. Sin ellas, Cloudflare bloquea el acceso SSH.

### 4.9 Configurar GitHub Secrets

En tu repositorio **`backend`** de GitHub, ve a **Settings → Secrets and variables → Actions** y crea estos secrets:

| Secret | Valor | Descripción |
|--------|-------|-------------|
| `VPS_HOST` | `ssh.jcapsule.work` | Hostname SSH del servidor (a través de Cloudflare Tunnel) |
| `VPS_USER` | `deploy` | Usuario SSH para deploys |
| `VPS_SSH_KEY` | Contenido de `~/.ssh/ductifact_deploy` | La clave privada completa (incluye `-----BEGIN...` y `-----END...`) |
| `CF_ACCESS_CLIENT_ID` | Client ID del Service Token | El que Cloudflare te dio al crear el Service Token (sección 4.8) |
| `CF_ACCESS_CLIENT_SECRET` | Client Secret del Service Token | Se muestra solo una vez al crearlo. Si lo pierdes, regenera el token. |

> **Nota**: los secrets de base de datos, JWT y dominio no van en el repo de `backend`. Se configuran directamente en los archivos `.env.prod` y `.env.staging` del servidor, gestionados desde el repo `infra/`. Así se separan las responsabilidades: el backend solo sabe cómo construir y subir su imagen.

---

## 5. Archivos de infraestructura (repo `infra/`)

> **Estos archivos viven en el repositorio `infra/`**, separado de `backend/`. Se documentan aquí como referencia para entender la arquitectura completa, pero se mantienen y versionan en su propio repo.

### 5.1 `docker-compose.prod.yml`

Compose de **producción**. Orquesta la API, PostgreSQL, Prometheus y Grafana.

```yaml
services:
  # ── PostgreSQL ─────────────────────────────────────────────
  postgres:
    image: postgres:15-alpine
    container_name: ductifact_prod_postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: ${DB_NAME}
    volumes:
      - prod_postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - prod_internal

  # ── Backend API ────────────────────────────────────────────
  app:
    image: ghcr.io/${GITHUB_REPO}:latest
    container_name: ductifact_prod_app
    restart: unless-stopped
    ports:
      - "127.0.0.1:8090:8090"  # Solo localhost — Caddy hace proxy aquí
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=${DB_USER}
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_NAME=${DB_NAME}
      - APP_PORT=8090
      - JWT_SECRET=${JWT_SECRET}
      - CORS_ORIGINS=${CORS_ORIGINS}
      - CONTRACT_VERSION=${CONTRACT_VERSION}
      - LOG_LEVEL=info
      - LOG_FORMAT=json
      - GIN_MODE=release
      - AUTO_MIGRATE=true
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - prod_internal

  # ── Prometheus (metrics scraping) ──────────────────────────
  prometheus:
    image: prom/prometheus:latest
    container_name: ductifact_prometheus
    restart: unless-stopped
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus_data:/prometheus
    networks:
      - prod_internal

  # ── Grafana (dashboards) ───────────────────────────────────
  grafana:
    image: grafana/grafana:latest
    container_name: ductifact_grafana
    restart: unless-stopped
    ports:
      - "127.0.0.1:3000:3000"  # Solo localhost — Caddy hace proxy aquí
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD:-admin}
    volumes:
      - grafana_data:/var/lib/grafana
    networks:
      - prod_internal

volumes:
  prod_postgres_data:
  prometheus_data:
  grafana_data:

networks:
  prod_internal:
    driver: bridge
```

**Detalles clave:**

| Concepto | Explicación |
|----------|-------------|
| `127.0.0.1:8090:8090` | Expone el puerto **solo en localhost**. Caddy (corriendo en el host) puede acceder, pero no es accesible desde internet. Si pusieras `0.0.0.0:8090:8090`, cualquiera con la IP podría acceder directamente saltando Caddy. |
| Sin Nginx | Caddy ya está en el host — no necesitamos otro reverse proxy dentro de Docker. Menos contenedores = menos complejidad y menos memoria. |
| Sin redes `external` | Cada compose es independiente. Caddy accede a los contenedores por los puertos expuestos en localhost, no por redes Docker compartidas. |
| `ghcr.io/${GITHUB_REPO}:latest` | La imagen Docker no se construye en el servidor. Se construye en GitHub Actions, se sube a ghcr.io y el servidor solo hace `docker pull`. |
| `GIN_MODE=release` | Desactiva el modo debug de Gin. En producción quieres que sea silencioso y rápido. |
| `LOG_FORMAT=json` | En producción los logs van en JSON para poder parsearlos con herramientas (Grafana/Loki). |

### 5.2 `docker-compose.staging.yml`

Compose de **staging**. Más ligero — solo la API y PostgreSQL. No necesita Prometheus ni Grafana.

```yaml
services:
  # ── PostgreSQL (staging) ───────────────────────────────────
  postgres:
    image: postgres:15-alpine
    container_name: ductifact_staging_postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: ${DB_NAME}
    volumes:
      - staging_postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - staging_internal

  # ── Backend API (staging) ──────────────────────────────────
  app:
    image: ghcr.io/${GITHUB_REPO}:staging
    container_name: ductifact_staging_app
    restart: unless-stopped
    ports:
      - "127.0.0.1:8091:8091"  # Solo localhost — Caddy hace proxy aquí
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=${DB_USER}
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_NAME=${DB_NAME}
      - APP_PORT=8091
      - JWT_SECRET=${JWT_SECRET}
      - CORS_ORIGINS=${CORS_ORIGINS}
      - CONTRACT_VERSION=${CONTRACT_VERSION}
      - LOG_LEVEL=debug
      - LOG_FORMAT=json
      - GIN_MODE=release
      - AUTO_MIGRATE=true
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - staging_internal

volumes:
  staging_postgres_data:

networks:
  staging_internal:
    driver: bridge
```

**Diferencias clave entre staging y producción:**

| Concepto | Staging | Producción |
|----------|---------|------------|
| Imagen Docker | `ghcr.io/.../...:staging` | `ghcr.io/.../...:latest` |
| Puerto de la API | `8091` | `8090` |
| Log level | `debug` (más detallado) | `info` (solo lo importante) |
| Prometheus / Grafana | No (innecesario) | Sí |
| Base de datos | Separada (puede llenarse de datos de test) | Separada (datos reales) |
| Red Docker | `staging_internal` | `prod_internal` |
| Contenedores | Prefijo `ductifact_staging_*` | Prefijo `ductifact_prod_*` |

> **¿Por qué redes separadas?** Para que staging y producción estén completamente aislados. Un bug en staging que tire la base de datos no afecta a producción. Cada entorno tiene su propio PostgreSQL con sus propios datos.

### 5.3 `.env.prod.example`

Plantilla de las variables de producción. El archivo `.env.prod` real **nunca se commitea**.

```dotenv
# ── Database ─────────────────────────────────────────────────
DB_USER=ductifact_user
DB_PASSWORD=CHANGE_ME_generate_with_openssl_rand_-base64_24
DB_NAME=ductifact_db

# ── Application ──────────────────────────────────────────────
JWT_SECRET=CHANGE_ME_generate_with_openssl_rand_-base64_32
CORS_ORIGINS=https://ductifact.jcapsule.work,https://api.ductifact.jcapsule.work
CONTRACT_VERSION=1.0.0

# ── GitHub image ─────────────────────────────────────────────
GITHUB_REPO=tu-usuario/ductifact

# ── Grafana ──────────────────────────────────────────────────
GRAFANA_PASSWORD=CHANGE_ME_secure_password
```

### 5.4 `.env.staging.example`

```dotenv
# ── Database ─────────────────────────────────────────────────
DB_USER=ductifact_staging_user
DB_PASSWORD=CHANGE_ME_different_from_prod
DB_NAME=ductifact_staging_db

# ── Application ──────────────────────────────────────────────
JWT_SECRET=CHANGE_ME_different_from_prod
CORS_ORIGINS=https://staging-api.ductifact.jcapsule.work
CONTRACT_VERSION=1.0.0

# ── GitHub image ─────────────────────────────────────────────
GITHUB_REPO=tu-usuario/ductifact
```

> **Importante**: staging y producción usan **contraseñas, JWT secrets y bases de datos diferentes**. Si compartiesen credenciales, un token generado en staging funcionaría en producción — un riesgo de seguridad.

### 5.5 `prometheus/prometheus.yml`

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: "ductifact-api-prod"
    metrics_path: "/api/v1/metrics"
    static_configs:
      - targets: ["ductifact_prod_app:8090"]
```

Prometheus solo monitoriza producción. Hace scrape del endpoint `/api/v1/metrics` cada 15 segundos. Como está en la misma red Docker (`prod_internal`), puede acceder al contenedor directamente. Desde internet está bloqueado por Caddy.

---

## 6. GitHub Actions CD: `.github/workflows/cd.yml`

Este workflow vive en el repositorio **`backend`** (`backend/.github/workflows/cd.yml`). Tiene **dos jobs**: uno para staging (desde `main`) y otro para producción (desde tags `v*`).

```yaml
name: CD

on:
  # Staging: cuando CI pasa en main
  workflow_run:
    workflows: ["CI"]
    branches: [main]
    types: [completed]

  # Producción: cuando se pushea un tag v*
  push:
    tags:
      - "v*"

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  # ══════════════════════════════════════════════════════════
  # STAGING — se ejecuta cuando CI pasa en main
  # ══════════════════════════════════════════════════════════
  deploy-staging:
    name: Deploy to Staging
    runs-on: ubuntu-latest
    if: >-
      github.event_name == 'workflow_run' &&
      github.event.workflow_run.conclusion == 'success'

    permissions:
      contents: read
      packages: write

    steps:
      - uses: actions/checkout@v4

      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and push (staging tag)
        uses: docker/build-push-action@v6
        with:
          context: .
          push: true
          tags: |
            ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:staging
            ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:staging-${{ github.sha }}

      - name: Install cloudflared
        run: |
          curl -fsSL https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb \
            -o /tmp/cloudflared.deb
          sudo dpkg -i /tmp/cloudflared.deb

      - name: Deploy to staging via Cloudflare Tunnel
        env:
          CF_ACCESS_CLIENT_ID: ${{ secrets.CF_ACCESS_CLIENT_ID }}
          CF_ACCESS_CLIENT_SECRET: ${{ secrets.CF_ACCESS_CLIENT_SECRET }}
        run: |
          mkdir -p ~/.ssh
          echo "${{ secrets.VPS_SSH_KEY }}" > ~/.ssh/deploy_key
          chmod 600 ~/.ssh/deploy_key

          ssh -o StrictHostKeyChecking=no \
              -o UserKnownHostsFile=/dev/null \
              -o "ProxyCommand=cloudflared access ssh --hostname %h" \
              -i ~/.ssh/deploy_key \
              ${{ secrets.VPS_USER }}@${{ secrets.VPS_HOST }} \
              "~/ductifact/infra/scripts/deploy.sh staging ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:staging"

  # ══════════════════════════════════════════════════════════
  # PRODUCTION — se ejecuta cuando se pushea un tag v*
  # ══════════════════════════════════════════════════════════
  deploy-production:
    name: Deploy to Production
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && startsWith(github.ref, 'refs/tags/v')

    permissions:
      contents: read
      packages: write

    steps:
      - uses: actions/checkout@v4

      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract version from tag
        id: version
        run: echo "VERSION=${GITHUB_REF#refs/tags/}" >> $GITHUB_OUTPUT

      - name: Build and push (production tags)
        uses: docker/build-push-action@v6
        with:
          context: .
          push: true
          tags: |
            ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:latest
            ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ steps.version.outputs.VERSION }}

      - name: Install cloudflared
        run: |
          curl -fsSL https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb \
            -o /tmp/cloudflared.deb
          sudo dpkg -i /tmp/cloudflared.deb

      - name: Deploy to production via Cloudflare Tunnel
        env:
          CF_ACCESS_CLIENT_ID: ${{ secrets.CF_ACCESS_CLIENT_ID }}
          CF_ACCESS_CLIENT_SECRET: ${{ secrets.CF_ACCESS_CLIENT_SECRET }}
        run: |
          mkdir -p ~/.ssh
          echo "${{ secrets.VPS_SSH_KEY }}" > ~/.ssh/deploy_key
          chmod 600 ~/.ssh/deploy_key

          ssh -o StrictHostKeyChecking=no \
              -o UserKnownHostsFile=/dev/null \
              -o "ProxyCommand=cloudflared access ssh --hostname %h" \
              -i ~/.ssh/deploy_key \
              ${{ secrets.VPS_USER }}@${{ secrets.VPS_HOST }} \
              "~/ductifact/infra/scripts/deploy.sh prod ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:latest"
```

### Explicación del flujo:

**1. Dos triggers, dos jobs**

| Trigger | Job | Cuándo |
|---------|-----|--------|
| `workflow_run` (CI pasa en main) | `deploy-staging` | Cada PR mergeada a main |
| `push` tag `v*` | `deploy-production` | Cuando creas un tag de release |

Cada job tiene su condición `if` para que no se ejecuten ambos a la vez.

**2. Tags de imagen diferentes**

| Entorno | Tags |
|---------|------|
| Staging | `:staging`, `:staging-abc123f` |
| Producción | `:latest`, `:v0.4.0` |

Staging siempre usa `:staging`. Producción usa `:latest` + el tag semántico. El tag con SHA o versión permite rollback.

**3. GitHub Container Registry (ghcr.io)**

En vez de Docker Hub (que tiene límites en el free tier), usamos **ghcr.io** que viene incluido gratis con GitHub.

**4. SSH deploy vía Cloudflare Tunnel**

Como tu servidor no expone el puerto SSH a internet (todo va por Cloudflare Tunnel), el workflow:
1. Instala `cloudflared` en el runner de GitHub Actions
2. Usa `cloudflared access ssh` como `ProxyCommand` — esto establece la conexión SSH a través del túnel
3. Las variables `CF_ACCESS_CLIENT_ID` y `CF_ACCESS_CLIENT_SECRET` autentican al runner ante Cloudflare Access
4. Una vez conectado por SSH, ejecuta `~/ductifact/infra/scripts/deploy.sh` que:
   - Hace `git pull` del repo `infra` para tener la config más reciente
   - Hace `docker pull` de la imagen
   - Reinicia el contenedor con `docker compose up -d app`
   - Verifica que el contenedor está corriendo
   - Limpia imágenes antiguas

**5. Script de deploy en `infra/scripts/deploy.sh`**

La lógica de deploy vive en el repo `infra`, no en el workflow del backend. Esto tiene varias ventajas:
- Puedes probar el script directamente en el servidor (`./scripts/deploy.sh staging ghcr.io/...`)
- Cambios en la lógica de deploy no requieren tocar el repo backend
- El workflow queda limpio — solo hace build, push y SSH

> **¿Por qué no `appleboy/ssh-action`?** Esa action asume una conexión SSH directa al host. Como tu SSH pasa por Cloudflare Tunnel, necesitas `cloudflared` como proxy. Usar SSH nativo con `ProxyCommand` es la forma estándar de hacer esto.

> **¿Por qué no reconstruir en el servidor?** Porque compilar Go consume CPU y RAM. Construir en GitHub Actions (que tiene 7GB de RAM) y solo hacer `pull` en el servidor es más eficiente y no afecta al tráfico en producción.

---

## 7. Primer despliegue manual

El CD automático funciona después del primer setup. La primera vez necesitas configurar el servidor manualmente.

### 7.1 Clonar el repo de infraestructura en el servidor

```bash
ssh deploy@ssh.jcapsule.work
cd ~

# Clonar el repo de infra
git clone https://github.com/tu-usuario/ductifact-infra.git ~/ductifact/infra
```

> **¿Por qué clonar?** Así puedes hacer `git pull` en el servidor para actualizar la configuración de infra (compose, prometheus, Caddyfile de referencia) sin tener que copiar archivos manualmente cada vez.

### 7.2 Crear los archivos `.env` en el servidor

```bash
ssh deploy@ssh.jcapsule.work
cd ~/ductifact/infra

# Producción
cp .env.prod.example .env.prod
nano .env.prod   # editar con valores reales

# Staging
cp .env.staging.example .env.staging
nano .env.staging   # editar con valores reales (DIFERENTES a prod)
```

Genera passwords seguros:

```bash
# Generar contraseñas random
openssl rand -base64 24   # para DB_PASSWORD
openssl rand -base64 32   # para JWT_SECRET
```

> **Usa contraseñas DIFERENTES para staging y producción.** Si son iguales, un token de staging podría funcionar en producción.

### 7.3 Levantar todo

```bash
cd ~/ductifact/infra

# Login en ghcr.io (necesitas un Personal Access Token de GitHub con scope `read:packages`)
echo "TU_GITHUB_TOKEN" | docker login ghcr.io -u TU_USUARIO --password-stdin

# Levantar staging
docker compose --env-file .env.staging -f docker-compose.staging.yml up -d

# Levantar producción (incluye Prometheus, Grafana)
docker compose --env-file .env.prod -f docker-compose.prod.yml up -d

# Verificar todos los contenedores
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
```

Deberías ver algo como:

```
NAMES                       STATUS                   PORTS
ductifact_prod_app          Up 2 minutes             127.0.0.1:8090->8090/tcp
ductifact_prod_postgres     Up 2 minutes (healthy)
ductifact_staging_app       Up 3 minutes             127.0.0.1:8091->8091/tcp
ductifact_staging_postgres  Up 3 minutes (healthy)
ductifact_prometheus        Up 2 minutes
ductifact_grafana           Up 2 minutes             127.0.0.1:3000->3000/tcp
```

> **Nota**: los puertos solo aparecen en `127.0.0.1` (localhost). No son accesibles desde internet — Caddy es quien los expone a través de Cloudflare Tunnel.

### 7.4 Configurar Caddy

Añade las entradas de ductifact a tu Caddyfile existente:

```bash
# Editar el Caddyfile
sudo nano /etc/caddy/Caddyfile

# Añadir los bloques de ductifact (ver sección 2 para la arquitectura de subdominios)

# Recargar Caddy (sin downtime)
sudo caddy reload --config /etc/caddy/Caddyfile

# Verificar que no hay errores
sudo systemctl status caddy
```

### 7.5 Verificar desde tu máquina

```bash
# Health check de producción
curl https://api.ductifact.jcapsule.work/api/v1/health

# Health check de staging
curl https://staging-api.ductifact.jcapsule.work/api/v1/health

# Verificar que /metrics está bloqueado
curl -I https://api.ductifact.jcapsule.work/api/v1/metrics
# Debería devolver 403 Forbidden

# Verificar Grafana (si configuraste el subdominio)
curl https://grafana.ductifact.jcapsule.work
```

### 7.6 Verificar Cloudflare Tunnel

```bash
# En el servidor
sudo systemctl status cloudflared

# Ver logs del túnel
sudo journalctl -u cloudflared -f --no-pager | tail -20

# Verificar conexiones activas
cloudflared tunnel info ductifact
```

---

## 8. Diagrama del flujo completo

```
 Developer
    │
    ├── merge PR → main
    │
    ▼
 GitHub Actions CI (.github/workflows/ci.yml)
    ├── lint & vet
    ├── unit tests
    ├── integration tests (+ Postgres)
    ├── contract tests (+ Postgres + API)
    └── e2e tests (+ Postgres + API)
    │
    ▼ (todo pasa ✅)
    │
 GitHub Actions CD (.github/workflows/cd.yml) — job: deploy-staging
    ├── docker build (multi-stage, Go → Alpine)
    ├── docker push → ghcr.io/user/ductifact:staging
    └── SSH → jcapsule.work
         ├── docker pull :staging
         └── docker compose -f docker-compose.staging.yml up -d app
                │
                ▼
         STAGING (staging-api.ductifact.jcapsule.work) ─ QA validates here
                │
           Approved? ✅
                │
                ▼
 Developer tags release (manual step):
   git tag -a v0.4.0 → git push origin v0.4.0
                │
                ▼
 GitHub Actions CD — job: deploy-production
    ├── docker build
    ├── docker push → ghcr.io/user/ductifact:v0.4.0 + :latest
    └── SSH → jcapsule.work
         ├── docker pull :latest
         └── docker compose -f docker-compose.prod.yml up -d app
                │
                ▼
         Servidor Debian 12 (jcapsule.work)
         ┌──────────────────────────────────────────────┐
         │  Cloudflare Tunnel (cloudflared)              │
         │    ├── api.ductifact.jcapsule.work            │
         │    ├── staging-api.ductifact.jcapsule.work        │
         │    └── grafana.ductifact.jcapsule.work        │
         │                                              │
         │  Caddy (reverse proxy, host-level)           │
         │    ├── api.ductifact.../api/* → :8090        │
         │    ├── staging.ductifact.../api/* → :8091    │
         │    ├── /metrics → ❌ 403                      │
         │    └── grafana.ductifact... → :3000           │
         │                                              │
         │  ┌─ Producción ─────────────────────────┐    │
         │  │  App (Go, :8090)                     │    │
         │  │  PostgreSQL (:5432)                  │    │
         │  │  Prometheus (:9090)                  │    │
         │  │  Grafana (:3000)                     │    │
         │  └──────────────────────────────────────┘    │
         │                                              │
         │  ┌─ Staging ────────────────────────────┐    │
         │  │  App (Go, :8091)                     │    │
         │  │  PostgreSQL (:5433)                  │    │
         │  └──────────────────────────────────────┘    │
         └──────────────────────────────────────────────┘

 ─── Hotfix flow ────────────────────────────────

 Bug in production!
    │
    ▼
 hotfix/* branch from release
    │
 PR → release (CI runs on release branch too)
    │
 Tag v0.4.1 on release → CD deploys to production
    │
 Merge release → main (backport)
```

---

## 9. Mantenimiento del servidor

### 9.1 Ver logs

```bash
# Logs de la API de producción (últimas 100 líneas, en tiempo real)
docker logs -f --tail=100 ductifact_prod_app

# Logs de la API de staging
docker logs -f --tail=100 ductifact_staging_app

# Logs de Caddy
sudo journalctl -u caddy -f

# Logs de Cloudflare Tunnel
sudo journalctl -u cloudflared -f

# Logs de todos los servicios de producción
docker compose -f docker-compose.prod.yml logs -f

# Logs de todos los servicios de staging
docker compose -f docker-compose.staging.yml logs -f
```

### 9.2 Backup de la base de datos

```bash
# Backup manual (producción)
docker exec ductifact_prod_postgres \
  pg_dump -U ductifact_user ductifact_db > ~/backups/prod_$(date +%Y%m%d).sql

# Backup manual (staging)
docker exec ductifact_staging_postgres \
  pg_dump -U ductifact_staging_user ductifact_staging_db > ~/backups/staging_$(date +%Y%m%d).sql

# Restaurar
cat ~/backups/prod_20260320.sql | docker exec -i ductifact_prod_postgres \
  psql -U ductifact_user ductifact_db
```

Automatiza los backups con un cron job:

```bash
# Editar crontab del usuario deploy
crontab -e

# Backup diario de producción a las 3am, mantener los últimos 7 días
0 3 * * * docker exec ductifact_prod_postgres pg_dump -U ductifact_user ductifact_db | gzip > ~/backups/prod_$(date +\%Y\%m\%d).sql.gz && find ~/backups -name "prod_*.sql.gz" -mtime +7 -delete

# Backup diario de staging a las 4am (opcional, menos crítico)
0 4 * * * docker exec ductifact_staging_postgres pg_dump -U ductifact_staging_user ductifact_staging_db | gzip > ~/backups/staging_$(date +\%Y\%m\%d).sql.gz && find ~/backups -name "staging_*.sql.gz" -mtime +3 -delete
```

### 9.3 Rollback

Si un deploy a producción rompe algo, tienes varias opciones:

**Opción 1: Rollback rápido con Docker** (segundos)

```bash
cd ~/ductifact/infra

# Ver las versiones disponibles
docker images ghcr.io/tu-usuario/ductifact

# Volver a la versión anterior
docker compose --env-file .env.prod -f docker-compose.prod.yml stop app

# Cambiar temporalmente la imagen en docker-compose.prod.yml
# image: ghcr.io/tu-usuario/ductifact:v0.3.0
docker compose --env-file .env.prod -f docker-compose.prod.yml up -d app
```

**Opción 2: Hotfix por la rama `release`** (minutos, lo correcto)

Este es el flujo completo descrito en `CONTRIBUTING.md` §7:

```bash
# 1. Crear hotfix desde release
git checkout -b hotfix/fix-urgent-bug origin/release

# 2. Arreglar, testear, PR → release
# 3. Tag nuevo en release
git checkout release && git pull
git tag -a v0.4.1 -m "Hotfix v0.4.1"
git push origin v0.4.1
# → CD despliega automáticamente a producción

# 4. Merge back to main
git checkout main && git pull
git merge release
git push origin main
```

> **Opción 1** es para emergencias (la API está caída, necesitas restaurar YA). **Opción 2** es el flujo correcto que queda registrado en git con un tag, un PR y CI validado.

### 9.4 Actualizaciones de seguridad del servidor

```bash
# Cada semana o dos semanas
ssh deploy@ssh.jcapsule.work
sudo apt update && sudo apt upgrade -y

# Si se actualiza el kernel, reiniciar
sudo reboot
```

### 9.5 Actualizar cloudflared

```bash
# Verificar versión actual
cloudflared --version

# Actualizar
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb \
  -o /tmp/cloudflared.deb
sudo dpkg -i /tmp/cloudflared.deb
sudo systemctl restart cloudflared
```

### 9.6 Reset de staging

A veces staging acumula datos de prueba y quieres empezar limpio:

```bash
cd ~/ductifact/infra

# Parar staging completamente
docker compose --env-file .env.staging -f docker-compose.staging.yml down

# Eliminar el volumen de datos (¡solo staging!)
docker volume rm ductifact_staging_postgres_data

# Levantar de nuevo (GORM AutoMigrate recrea las tablas al arrancar la app)
docker compose --env-file .env.staging -f docker-compose.staging.yml up -d
```

---

## 10. Seguridad adicional con Cloudflare

### 10.1 Verificar que solo Cloudflare accede

Aunque no hay puertos abiertos (gracias al túnel), es buena práctica verificar:

```bash
# Ver puertos escuchando en el servidor
sudo ss -tlnp

# Deberías ver servicios escuchando, pero UFW los bloquea desde internet:
#   *:22    (SSH — solo accesible desde LAN y Cloudflare Tunnel)
#   *:80    (Caddy — solo accesible desde localhost/cloudflared)
#   *:3000  (Grafana — solo accesible desde localhost/Caddy)
```

Ningún puerto está abierto al público en el firewall. `cloudflared` y Caddy se comunican por localhost. La LAN accede por las reglas `ufw allow from 192.168.x.0/24`.

### 10.2 Configuraciones recomendadas en Cloudflare Dashboard

En tu panel de Cloudflare, configura:

| Setting | Valor | Explicación |
|---------|-------|-------------|
| SSL/TLS mode | **Full (strict)** | Cloudflare verifica el certificado entre su edge y tu servidor. Con tunnel, Full es suficiente. |
| Always Use HTTPS | **On** | Redirige HTTP a HTTPS automáticamente. |
| Minimum TLS Version | **1.2** | Bloquea clientes con TLS antiguo (inseguro). |
| Browser Integrity Check | **On** | Bloquea requests con user-agents sospechosos. |
| Bot Fight Mode | **On** | Protección adicional contra bots maliciosos. |

### 10.3 Restringir acceso a staging (opcional)

Si quieres que staging solo sea accesible para ti:

1. Ve a **Cloudflare Dashboard → Zero Trust → Access → Applications**
2. Crea una aplicación para `staging-api.ductifact.jcapsule.work`
3. Configura una política que solo permita tu email
4. Cloudflare pedirá autenticación antes de permitir acceso

Esto es útil para evitar que bots o curiosos accedan a staging.

---

## 11. Orden de implementación

1. **Configurar servidor** — sección 4 (usuario deploy, Docker, cloudflared)
2. **Configurar Cloudflare Tunnel** — sección 4.4–4.5 (túnel + subdominios)
3. **Repo `infra/` + archivos de configuración** — sección 5
4. **Añadir entradas al Caddyfile** — sección 7.4 (directamente en el servidor)
5. **`backend/.github/workflows/ci.yml`** — primero que funcione CI (si no lo tienes ya)
6. **Primer despliegue manual** — sección 7
7. **`backend/.github/workflows/cd.yml`** — automatizar despliegues
8. **Verificar seguridad** — sección 10
9. **Grafana** — opcional, conectar a Prometheus para dashboards

---

## 12. Checklist final

- [ ] Servidor configurado (Docker, UFW solo LAN, usuario `deploy`)
- [ ] `cloudflared` instalado y corriendo como servicio
- [ ] Túnel creado con subdominios: `ssh.jcapsule.work`, `api.ductifact.jcapsule.work`, `staging-api.ductifact.jcapsule.work`
- [ ] SSH key de deploy configurada
- [ ] Service Token de Cloudflare Access creado (sección 4.8)
- [ ] GitHub Secrets configurados en el repo `backend` (`VPS_HOST`, `VPS_USER`, `VPS_SSH_KEY`, `CF_ACCESS_CLIENT_ID`, `CF_ACCESS_CLIENT_SECRET`)
- [ ] Repo `infra/` creado con docker-compose.prod.yml, docker-compose.staging.yml, Caddyfile.ductifact, prometheus.yml
- [ ] Repo `infra/` clonado en el servidor (`~/ductifact/infra/`)
- [ ] `.env.prod` y `.env.staging` creados en el servidor (con credenciales DIFERENTES)
- [ ] `backend/.github/workflows/ci.yml` funciona en `main` y `release`
- [ ] `backend/.github/workflows/cd.yml` despliega a staging (desde main) y producción (desde tags)
- [ ] `.dockerignore` creado en la raíz de `backend/`
- [ ] Dockerfile optimizado (ldflags, non-root user)
- [ ] `/metrics` y `/health` bloqueados desde internet (Caddy devuelve 403)
- [ ] Entradas de ductifact añadidas al Caddyfile del servidor
- [ ] Cloudflare SSL/TLS mode en Full (strict)
- [ ] Backup de DB automatizado con cron
- [ ] Rama `release` creada tras primer tag (ver `CONTRIBUTING.md` §6–7)
