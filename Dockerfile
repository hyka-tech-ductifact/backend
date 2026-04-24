# ── Stage 1: Build ──────────────────────────────────────────
FROM docker.io/library/golang:1.26-alpine AS builder

WORKDIR /app

# Copy dependency files FIRST (this layer gets cached if go.mod/go.sum don't change)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code (this layer invalidates when code changes)
COPY . .

# Build with optimizations: -s removes symbol table, -w removes DWARF debug info (~30% smaller binary)
ARG APP_VERSION=dev
ARG APP_COMMIT=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X ductifact/internal/config.Version=${APP_VERSION} -X ductifact/internal/config.Commit=${APP_COMMIT}" \
    -o app ./cmd/api

# ── Stage 2: Runtime ────────────────────────────────────────
FROM docker.io/library/alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/app .

# Copy OpenAPI spec for /docs endpoint.
# The spec must exist — run `make ensure-contract` before building.
# CI does this automatically; locally `make docker-build` handles it.
COPY contracts/openapi/bundled.yaml contracts/openapi/bundled.yaml

ARG APP_PORT=8080
EXPOSE ${APP_PORT}

# Run as non-root user (security best practice)
RUN adduser -D -g '' appuser
USER appuser

CMD ["./app"]