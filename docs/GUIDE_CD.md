# Guía de CD — Despliegue en Hetzner VPS

> **Fase 7.3 + 7.4 del Roadmap**
> Desplegar automáticamente a **staging** desde `main` y a **producción** desde tags `v*`.

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
  QA valida en staging
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

- **Staging**: se despliega automáticamente cada vez que un PR se mergea a `main`. No requiere intervención manual.
- **Producción**: se despliega cuando se pushea un **tag** `v*`. El tag lo creas tú manualmente después de que QA valide staging (ver `CONTRIBUTING.md` §6).

Esto es **Continuous Delivery** (no Deployment) para producción — hay un paso manual intencional (el tag) que actúa como puerta de aprobación.

---

## 2. Arquitectura en el VPS

El VPS tiene **dos entornos**: staging y producción. En esta guía empezamos con producción (staging se puede añadir en un segundo VPS o en el mismo con puertos/dominios separados).

```
Internet (puerto 80/443)
    │
    ▼
  Nginx (reverse proxy + HTTPS)
    │
    ├── /api/v1/*        → backend  (Go, :8080)    ── contenedor Docker
    ├── /api/v1/metrics  → ❌ bloqueado desde internet
    ├── /health          → ❌ bloqueado desde internet
    └── /*               → (futuro frontend)
    
  PostgreSQL (:5432)   ── contenedor Docker (solo red interna)
  Prometheus  (:9090)  ── contenedor Docker (solo red interna)
  Grafana     (:3000)  ── contenedor Docker (accesible por Nginx si quieres)
```

**¿Por qué Nginx?**

Tu API Go escucha en el puerto 8080. Pero los usuarios acceden por el puerto 443 (HTTPS). Nginx actúa como intermediario:
- Recibe tráfico en 80/443 (HTTP/HTTPS)
- Termina TLS (gestiona el certificado SSL)
- Enruta las peticiones al contenedor correcto
- Bloquea rutas internas (`/metrics`, `/health`) para que no sean accesibles desde internet

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
├── docker-compose.prod.yml
├── .env.prod.example
├── init.sql                         ← copia del schema (para inicializar la DB)
├── nginx/
│   └── nginx.conf
└── prometheus/
    └── prometheus.yml
```

> **¿Por qué repos separados?** Porque cuando añadas un servicio `frontend/`, la infra orquestará ambos servicios. Si `infra/` estuviera dentro de `backend/`, el frontend no podría usarlo. Repos separados permiten que la infraestructura evolucione independientemente de cada servicio.
```

---

## 4. Setup inicial del VPS (una sola vez)

Esto lo haces una vez cuando contratas el VPS. Después, los deploys son automáticos.

### 4.1 Contratar el VPS

1. Crea una cuenta en [hetzner.com](https://www.hetzner.com/cloud)
2. Crea un servidor **CX22** (2 vCPU, 4GB RAM, €4.35/mes)
3. Elige **Ubuntu 24.04** como sistema operativo
4. Elige el datacenter más cercano (Falkenstein o Nuremberg para Europa)
5. Añade tu **SSH key pública** (si no tienes una, genera con `ssh-keygen -t ed25519`)
6. Anota la **IP pública** del servidor (ej: `65.108.xxx.xxx`)

### 4.2 Configuración inicial del servidor

Conéctate al VPS:

```bash
ssh root@TU_IP_DEL_VPS
```

Ejecuta estos comandos para preparar el servidor:

```bash
# 1. Actualizar el sistema
apt update && apt upgrade -y

# 2. Crear usuario para deploys (no usar root)
adduser deploy --disabled-password --gecos ""
usermod -aG sudo deploy

# 3. Configurar SSH para el usuario deploy
mkdir -p /home/deploy/.ssh
cp ~/.ssh/authorized_keys /home/deploy/.ssh/
chown -R deploy:deploy /home/deploy/.ssh
chmod 700 /home/deploy/.ssh
chmod 600 /home/deploy/.ssh/authorized_keys

# 4. Instalar Docker
curl -fsSL https://get.docker.com | sh
usermod -aG docker deploy

# 5. Instalar Docker Compose plugin
apt install -y docker-compose-plugin

# 6. Configurar firewall (UFW)
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable

# 7. Verificar
docker --version
docker compose version
ufw status
```

**¿Qué hace cada paso?**

| Paso | Explicación |
|------|-------------|
| `adduser deploy` | Creas un usuario sin privilegios para los despliegues. Nunca uses `root` para correr tu aplicación — si alguien compromete tu app, no tendrá acceso root al servidor. |
| `--disabled-password` | El usuario `deploy` no tiene contraseña. Solo se puede acceder por SSH key. Más seguro. |
| `usermod -aG docker deploy` | Permite al usuario `deploy` ejecutar Docker sin `sudo`. |
| `ufw allow 80/443` | Abre solo los puertos necesarios (HTTP y HTTPS). Todo lo demás está bloqueado por el firewall. |

### 4.3 Crear la estructura en el servidor

Como usuario `deploy`:

```bash
ssh deploy@TU_IP_DEL_VPS

# Crear directorios
mkdir -p ~/ductifact
```

### 4.4 Configurar SSH key para GitHub Actions

GitHub Actions necesita poder hacer SSH al VPS para desplegar. Genera una key **dedicada** para CI/CD:

```bash
# En tu máquina local (no en el VPS)
ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/ductifact_deploy
```

Esto genera dos archivos:
- `~/.ssh/ductifact_deploy` — la clave **privada** (la sube a GitHub como Secret)
- `~/.ssh/ductifact_deploy.pub` — la clave **pública** (la pones en el VPS)

```bash
# Copiar la clave pública al VPS
ssh-copy-id -i ~/.ssh/ductifact_deploy.pub deploy@TU_IP_DEL_VPS
```

### 4.5 Configurar GitHub Secrets

En tu repositorio **`backend`** de GitHub, ve a **Settings → Secrets and variables → Actions** y crea estos secrets:

| Secret | Valor | Descripción |
|--------|-------|-------------|
| `VPS_HOST` | `65.108.xxx.xxx` | IP pública del VPS |
| `VPS_USER` | `deploy` | Usuario SSH para deploys |
| `VPS_SSH_KEY` | Contenido de `~/.ssh/ductifact_deploy` | La clave privada completa (incluye `-----BEGIN...` y `-----END...`) |

> **Nota**: los secrets de base de datos, JWT y dominio (`PROD_DB_*`, `PROD_JWT_SECRET`, `PROD_DOMAIN`) no van en el repo de `backend`. Se configuran directamente en el archivo `.env.prod` del VPS, gestionado desde el repo `infra/`. Así se separan las responsabilidades: el backend solo sabe cómo construir y subir su imagen.

---

## 5. Archivos de infraestructura (repo `infra/`)

> **Estos archivos viven en el repositorio `infra/`**, separado de `backend/`. Se documentan aquí como referencia para entender la arquitectura completa, pero se mantienen y versionan en su propio repo.

### 5.1 `docker-compose.prod.yml`

Este es el compose de producción que corre en el VPS. Orquesta todos los servicios.

```yaml
services:
  # ── PostgreSQL ─────────────────────────────────────────────
  postgres:
    image: postgres:15-alpine
    container_name: ductifact_postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: ${DB_NAME}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - internal

  # ── Backend API ────────────────────────────────────────────
  app:
    image: ghcr.io/${GITHUB_REPO}:latest
    container_name: ductifact_app
    restart: unless-stopped
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=${DB_USER}
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_NAME=${DB_NAME}
      - APP_PORT=8080
      - JWT_SECRET=${JWT_SECRET}
      - CORS_ORIGINS=${CORS_ORIGINS}
      - CONTRACT_VERSION=${CONTRACT_VERSION}
      - LOG_LEVEL=info
      - LOG_FORMAT=json
      - GIN_MODE=release
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - internal

  # ── Nginx (reverse proxy) ─────────────────────────────────
  nginx:
    image: nginx:alpine
    container_name: ductifact_nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - certbot_data:/etc/letsencrypt:ro
      - certbot_webroot:/var/www/certbot:ro
    depends_on:
      - app
    networks:
      - internal
      - external

  # ── Certbot (HTTPS certificates) ──────────────────────────
  certbot:
    image: certbot/certbot
    container_name: ductifact_certbot
    volumes:
      - certbot_data:/etc/letsencrypt
      - certbot_webroot:/var/www/certbot
    entrypoint: "/bin/sh -c 'trap exit TERM; while :; do certbot renew; sleep 12h; done'"
    networks:
      - external

  # ── Prometheus (metrics scraping) ──────────────────────────
  prometheus:
    image: prom/prometheus:latest
    container_name: ductifact_prometheus
    restart: unless-stopped
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus_data:/prometheus
    networks:
      - internal

  # ── Grafana (dashboards) ───────────────────────────────────
  grafana:
    image: grafana/grafana:latest
    container_name: ductifact_grafana
    restart: unless-stopped
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD:-admin}
    volumes:
      - grafana_data:/var/lib/grafana
    networks:
      - internal

volumes:
  postgres_data:
  certbot_data:
  certbot_webroot:
  prometheus_data:
  grafana_data:

networks:
  internal:
    driver: bridge
  external:
    driver: bridge
```

**Detalles clave:**

| Concepto | Explicación |
|----------|-------------|
| `networks: internal / external` | `internal` es la red privada donde viven todos los servicios. `external` es la red que tiene acceso a internet. Solo Nginx y Certbot están en `external`. PostgreSQL, Prometheus y la API **no son accesibles directamente desde internet**. |
| `ghcr.io/${GITHUB_REPO}:latest` | La imagen Docker de tu API no se construye en el VPS. Se construye en GitHub Actions, se sube a GitHub Container Registry (ghcr.io) y el VPS solo hace `docker pull`. Así el VPS no necesita compilar Go ni tener el código fuente. |
| `GIN_MODE=release` | Desactiva el modo debug de Gin. En desarrollo ves warnings y stack traces bonitos. En producción quieres que sea silencioso y rápido. |
| `LOG_FORMAT=json` | En producción los logs van en JSON para poder parsearlos con herramientas (Grafana/Loki). En desarrollo usas `text` para legibilidad. |
| `certbot` | Contenedor que renueva automáticamente los certificados HTTPS cada 12 horas. Let's Encrypt los emite gratis pero expiran cada 90 días. |

### 5.2 `.env.prod.example`

Plantilla de las variables de producción. El archivo `.env.prod` real **nunca se commitea**.

```dotenv
# ── Database ─────────────────────────────────────────────────
DB_USER=ductifact_user
DB_PASSWORD=CHANGE_ME_generate_with_openssl_rand_-base64_24
DB_NAME=ductifact_db

# ── Application ──────────────────────────────────────────────
JWT_SECRET=CHANGE_ME_generate_with_openssl_rand_-base64_32
CORS_ORIGINS=https://tudominio.com,https://www.tudominio.com
CONTRACT_VERSION=1.0.0

# ── GitHub image ─────────────────────────────────────────────
GITHUB_REPO=tu-usuario/ductifact

# ── Grafana ──────────────────────────────────────────────────
GRAFANA_PASSWORD=CHANGE_ME_secure_password

# ── Domain (used by Certbot/Nginx) ───────────────────────────
DOMAIN=api.tudominio.com
```

### 5.3 `nginx/nginx.conf`

```nginx
events {
    worker_connections 1024;
}

http {
    # ── Logging ──────────────────────────────────────────────
    log_format main '$remote_addr - $remote_user [$time_local] '
                    '"$request" $status $body_bytes_sent '
                    '"$http_referer" "$http_user_agent"';

    access_log /var/log/nginx/access.log main;
    error_log  /var/log/nginx/error.log warn;

    # ── Security headers ─────────────────────────────────────
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # ── Upstream: backend API ────────────────────────────────
    upstream backend {
        server app:8080;
    }

    # ── HTTP → HTTPS redirect ────────────────────────────────
    server {
        listen 80;
        server_name _;

        # Certbot challenge (needed for certificate renewal)
        location /.well-known/acme-challenge/ {
            root /var/www/certbot;
        }

        # Redirect everything else to HTTPS
        location / {
            return 301 https://$host$request_uri;
        }
    }

    # ── HTTPS server ─────────────────────────────────────────
    server {
        listen 443 ssl;
        server_name _;

        # SSL certificates (managed by Certbot)
        ssl_certificate     /etc/letsencrypt/live/${DOMAIN}/fullchain.pem;
        ssl_certificate_key /etc/letsencrypt/live/${DOMAIN}/privkey.pem;

        # Modern TLS config
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers HIGH:!aNULL:!MD5;

        # ── Block internal endpoints from internet ───────────
        location = /api/v1/metrics {
            return 403;
        }

        location = /health {
            return 403;
        }

        # ── Proxy to backend API ─────────────────────────────
        location /api/ {
            proxy_pass http://backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        # ── Future: frontend ─────────────────────────────────
        # location / {
        #     proxy_pass http://frontend:3000;
        #     proxy_set_header Host $host;
        # }
    }
}
```

**¿Qué hace cada bloque?**

| Bloque | Explicación |
|--------|-------------|
| `upstream backend` | Define el grupo de servidores backend. `app:8080` es el nombre del contenedor Docker en la red interna. Nginx resuelve `app` a la IP del contenedor automáticamente gracias a la red Docker. |
| `server listen 80` | Recibe tráfico HTTP. Su único trabajo es redirigir a HTTPS (`return 301 https://...`). La excepción es `/.well-known/acme-challenge/` que Certbot necesita para validar el dominio. |
| `server listen 443 ssl` | El servidor real. Recibe HTTPS, desencripta y pasa las peticiones al backend. |
| `location = /api/v1/metrics` | `= ` es un match exacto. Devuelve 403 (Forbidden) para que nadie desde internet pueda ver las métricas de Prometheus. Prometheus las lee desde la red interna Docker. |
| `proxy_set_header X-Real-IP` | Tu API recibe la IP del cliente real, no la IP de Nginx. Sin esto, todos los requests vendrían de `172.18.0.x` (la IP interna de Nginx). |
| `X-Forwarded-Proto` | Le dice a tu API si la petición original fue HTTP o HTTPS. Útil para generar URLs correctas en las respuestas. |

### 5.4 `prometheus/prometheus.yml`

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: "ductifact-api"
    metrics_path: "/api/v1/metrics"
    static_configs:
      - targets: ["app:8080"]
```

Prometheus hace scrape (lee las métricas) del endpoint `/api/v1/metrics` de tu API cada 15 segundos. Como está en la misma red Docker (`internal`), puede acceder al contenedor `app` directamente. Desde internet está bloqueado por Nginx.

---

## 6. GitHub Actions CD: `.github/workflows/cd.yml`

Este workflow vive en el repositorio **`backend`** (`backend/.github/workflows/cd.yml`). Su responsabilidad es construir la imagen Docker del backend, subirla a ghcr.io y desplegar en el VPS.

```yaml
name: CD

# Se ejecuta solo cuando CI pasa en main
on:
  workflow_run:
    workflows: ["CI"]
    branches: [main]
    types: [completed]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  deploy:
    name: Build & Deploy
    runs-on: ubuntu-latest
    # Solo se ejecuta si CI pasó correctamente
    if: ${{ github.event.workflow_run.conclusion == 'success' }}

    permissions:
      contents: read
      packages: write

    steps:
      # ── 1. Checkout code ───────────────────────────────────
      - uses: actions/checkout@v4

      # ── 2. Login to GitHub Container Registry ──────────────
      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      # ── 3. Build and push Docker image ─────────────────────
      - name: Build and push
        uses: docker/build-push-action@v6
        with:
          context: .
          push: true
          tags: |
            ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:latest
            ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ github.sha }}

      # ── 4. Deploy to VPS via SSH ───────────────────────────
      - name: Deploy to Hetzner VPS
        uses: appleboy/ssh-action@v1
        with:
          host: ${{ secrets.VPS_HOST }}
          username: ${{ secrets.VPS_USER }}
          key: ${{ secrets.VPS_SSH_KEY }}
          script: |
            cd ~/ductifact/infra

            # Pull the latest backend image
            docker pull ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:latest

            # Restart only the app service (zero-downtime for DB)
            docker compose -f docker-compose.prod.yml up -d app

            # Verify the app is healthy
            sleep 5
            if ! docker inspect --format='{{.State.Running}}' ductifact_app | grep -q true; then
              echo "ERROR: app container is not running"
              docker logs --tail=30 ductifact_app
              exit 1
            fi

            # Clean up old images
            docker image prune -f

            echo "Deploy successful!"
```

### Explicación del flujo:

**1. ¿Cuándo se ejecuta?**

```yaml
on:
  workflow_run:
    workflows: ["CI"]
    branches: [main]
    types: [completed]
```

`workflow_run` es un trigger especial: "ejecuta este workflow cuando el workflow `CI` termine en la rama `main`". Con el `if: conclusion == 'success'`, solo despliega **si CI pasó**. Esto encadena CI → CD sin duplicar los tests.

**2. GitHub Container Registry (ghcr.io)**

En vez de Docker Hub (que tiene límites en el free tier), usamos **ghcr.io** que viene incluido gratis con GitHub. La imagen se publica como:

```
ghcr.io/tu-usuario/ductifact-backend:latest
ghcr.io/tu-usuario/ductifact-backend:abc123f   ← commit SHA (para rollback)
```

El tag `latest` siempre apunta a la última versión. El tag con `github.sha` permite volver a una versión anterior si algo falla.

**3. SSH deploy**

`appleboy/ssh-action` se conecta al VPS por SSH y ejecuta los comandos. El workflow:
1. Descarga la nueva imagen (`docker pull`)
2. Reinicia solo el contenedor `app` (`docker compose up -d app`)
3. Verifica que arrancó correctamente
4. Limpia imágenes viejas para no llenar el disco

> **¿Por qué no reconstruir en el VPS?** Porque compilar Go consume CPU y RAM. Tu VPS de 4GB podría quedarse corto. Construir en GitHub Actions (que tiene 7GB de RAM) y solo hacer `pull` en el VPS es más eficiente.

---

## 7. Primer despliegue manual

El CD automático funciona después el primer setup. La primera vez necesitas configurar el VPS manualmente.

### 7.1 Clonar el repo de infraestructura en el VPS

```bash
ssh deploy@TU_IP_VPS
cd ~

# Clonar el repo de infra
git clone https://github.com/tu-usuario/ductifact-infra.git ~/ductifact/infra

# Copiar init.sql desde tu máquina local (o añádelo al repo infra)
# scp backend/init.sql deploy@TU_IP_VPS:~/ductifact/infra/
```

> **¿Por qué clonar?** Así puedes hacer `git pull` en el VPS para actualizar la configuración de infra (nginx, compose, prometheus) sin tener que copiar archivos manualmente cada vez.

### 7.2 Crear el `.env.prod` en el VPS

```bash
ssh deploy@TU_IP_VPS
cd ~/ductifact/infra

# Copiar el example y editar con valores reales
cp .env.prod.example .env.prod
nano .env.prod   # o vim, el editor que prefieras
```

### 7.3 Obtener certificado HTTPS (primera vez)

Si tienes un dominio apuntando al VPS:

```bash
cd ~/ductifact/infra

# 1. Primero, comenta las líneas de SSL en nginx.conf y usa solo HTTP
#    (Certbot necesita que Nginx esté corriendo para validar el dominio)

# 2. Levantar Nginx en modo HTTP-only
docker compose -f docker-compose.prod.yml up -d nginx

# 3. Obtener el certificado
docker compose -f docker-compose.prod.yml run --rm certbot \
  certbot certonly --webroot -w /var/www/certbot \
  -d tu-dominio.com --email tu@email.com --agree-tos --no-eff-email

# 4. Descomentar las líneas de SSL en nginx.conf

# 5. Reiniciar todo
docker compose -f docker-compose.prod.yml up -d
```

Si **no tienes dominio aún**, puedes usar la IP directamente pero sin HTTPS. Comenta el bloque `server listen 443` y configura el bloque `listen 80` para hacer proxy directamente (sin redirect a HTTPS).

### 7.4 Levantar todo

```bash
cd ~/ductifact/infra

# Login en ghcr.io (necesitas un Personal Access Token de GitHub con scope `read:packages`)
echo "TU_GITHUB_TOKEN" | docker login ghcr.io -u TU_USUARIO --password-stdin

# Levantar todos los servicios
docker compose --env-file .env.prod -f docker-compose.prod.yml up -d

# Verificar
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs --tail=20 app
```

### 7.5 Verificar desde tu máquina

```bash
# Health check
curl https://tu-dominio.com/api/v1/health

# O si no tienes dominio:
curl http://TU_IP_VPS/api/v1/health

# Verificar que /metrics está bloqueado
curl -I https://tu-dominio.com/api/v1/metrics
# Debería devolver 403 Forbidden
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
 GitHub Actions CD (.github/workflows/cd.yml)
    ├── docker build (multi-stage, Go → Alpine)
    ├── docker push → ghcr.io/user/ductifact-backend:staging
    └── SSH → VPS (staging)
         ├── docker pull :staging
         └── docker compose up -d app  (en ~/ductifact/infra)
                │
                ▼
         STAGING ─ QA validates here
                │
           Approved? ✅
                │
                ▼
 Developer tags release (manual step):
   git tag -a v0.4.0 → git push origin v0.4.0
                │
                ▼
 GitHub Actions CD (triggered by tag v*)
    ├── docker build
    ├── docker push → ghcr.io/user/ductifact-backend:0.4.0 + :latest
    └── SSH → VPS (production)
         ├── docker pull :latest
         └── docker compose up -d app  (en ~/ductifact/infra)
                │
                ▼
         Hetzner VPS (Production)
         ┌──────────────────────────────────┐
         │  Nginx (:80/:443)                │
         │    ├── /api/* → app:8080         │
         │    └── /metrics → ❌ 403          │
         │                                  │
         │  App (Go, :8080)                 │
         │    ├── API endpoints             │
         │    └── Prometheus metrics        │
         │                                  │
         │  PostgreSQL (:5432, internal)    │
         │  Prometheus (:9090, internal)    │
         │  Grafana (:3000, internal)       │
         └──────────────────────────────────┘

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

## 9. Mantenimiento del VPS

### 9.1 Ver logs

```bash
# Logs de la API (últimos 100 líneas, en tiempo real)
docker logs -f --tail=100 ductifact_app

# Logs de Nginx
docker logs -f --tail=100 ductifact_nginx

# Logs de todos los servicios
docker compose -f docker-compose.prod.yml logs -f
```

### 9.2 Backup de la base de datos

```bash
# Backup manual
docker exec ductifact_postgres pg_dump -U ductifact_user ductifact_db > backup_$(date +%Y%m%d).sql

# Restaurar
cat backup_20260311.sql | docker exec -i ductifact_postgres psql -U ductifact_user ductifact_db
```

Es buena idea automatizar los backups con un cron job:

```bash
# Editar crontab del usuario deploy
crontab -e

# Añadir: backup diario a las 3am, mantener los últimos 7 días
0 3 * * * docker exec ductifact_postgres pg_dump -U ductifact_user ductifact_db | gzip > ~/backups/db_$(date +\%Y\%m\%d).sql.gz && find ~/backups -name "db_*.sql.gz" -mtime +7 -delete
```

### 9.3 Rollback

Si un deploy a producción rompe algo, tienes varias opciones:

**Opción 1: Rollback rápido con Docker** (segundos)

```bash
# Ver las versiones disponibles
docker images ghcr.io/tu-usuario/ductifact-backend

# Volver a la versión anterior
docker compose -f docker-compose.prod.yml stop app

# Editar docker-compose.prod.yml temporalmente para usar el tag anterior
# image: ghcr.io/tu-usuario/ductifact-backend:0.3.0
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
ssh deploy@TU_IP_VPS
sudo apt update && sudo apt upgrade -y

# Si se actualiza el kernel, reiniciar
sudo reboot
```

---

## 10. Orden de implementación

1. **Repo `infra/` + archivos de configuración** — crea el repo con docker-compose.prod.yml, nginx.conf, prometheus.yml
2. **`backend/.github/workflows/ci.yml`** — primero que funcione CI
3. **Contratar VPS + setup inicial** — sección 4 de esta guía
4. **Primer despliegue manual** — sección 7 (clona `infra/` en el VPS)
5. **`backend/.github/workflows/cd.yml`** — automatizar despliegues
6. **Certificado HTTPS** — sección 7.3
7. **Grafana** — opcional, conectar a Prometheus para dashboards

---

## 11. Checklist final

- [ ] VPS contratado y configurado (Docker, firewall, usuario `deploy`)
- [ ] SSH key de deploy configurada
- [ ] GitHub Secrets configurados en el repo `backend` (`VPS_HOST`, `VPS_USER`, `VPS_SSH_KEY`)
- [ ] Repo `infra/` creado con docker-compose.prod.yml, nginx.conf, prometheus.yml
- [ ] Repo `infra/` clonado en el VPS (`~/ductifact/infra/`)
- [ ] `backend/.github/workflows/ci.yml` funciona en `main` y `release` (todos los checks pasan)
- [ ] `backend/.github/workflows/cd.yml` despliega a staging (desde main) y producción (desde tags)
- [ ] `.dockerignore` creado en la raíz de `backend/`
- [ ] Dockerfile optimizado (ldflags, non-root user)
- [ ] HTTPS configurado con Let's Encrypt
- [ ] `/metrics` y `/health` bloqueados desde internet
- [ ] Backup de DB automatizado con cron
- [ ] Variables de entorno de producción configuradas en `.env.prod` del VPS
- [ ] Rama `release` creada tras primer tag (ver `CONTRIBUTING.md` §6–7)
