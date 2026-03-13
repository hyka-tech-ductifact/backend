# ── Stage 1: Build ──────────────────────────────────────────
FROM docker.io/library/golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependency files FIRST (this layer gets cached if go.mod/go.sum don't change)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code (this layer invalidates when code changes)
COPY . .

# Build with optimizations: -s removes symbol table, -w removes DWARF debug info (~30% smaller binary)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o app ./cmd/api

# ── Stage 2: Runtime ────────────────────────────────────────
FROM docker.io/library/alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/app .

ARG APP_PORT=8080
EXPOSE ${APP_PORT}

# Run as non-root user (security best practice)
RUN adduser -D -g '' appuser
USER appuser

CMD ["./app"]