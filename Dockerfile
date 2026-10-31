# ── Stage 0: Bundle OpenAPI spec ───────────────────────────────────
FROM docker.io/redocly/cli:2.20.5 AS spec

WORKDIR /spec

COPY contracts/openapi ./
COPY internal/config/contract_version.go /tmp/contract_version.go

# Resolve $ref and stamp info.version with ContractVersion (mirrors make contract-bundle)
RUN redocly bundle openapi.yaml -o bundled.yaml \
 && V=$(sed -n 's/^const ContractVersion = "\(.*\)"/\1/p' /tmp/contract_version.go) \
 && sed -i "1,/^  version: .*/s/^  version: .*/  version: $V/" bundled.yaml
# ── Stage 1: Build ──────────────────────────────────────────
FROM docker.io/library/golang:1.26-alpine AS builder

WORKDIR /app

# Copy dependency files FIRST (this layer gets cached if go.mod/go.sum don't change)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code (this layer invalidates when code changes)
COPY . .

# Build with optimizations: -s removes symbol table, -w removes DWARF debug info (~30% smaller binary)
ARG APP_COMMIT=unknown
ARG APP_BUILD_TIME=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X ductifact/internal/config.Commit=${APP_COMMIT} -X ductifact/internal/config.BuildTime=${APP_BUILD_TIME}" \
    -o app ./cmd/api

# ── Stage 2: Runtime ────────────────────────────────────────
FROM docker.io/library/alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/app .

# OpenAPI spec for /docs, bundled in the spec stage from contracts/openapi/
COPY --from=spec /spec/bundled.yaml contracts/openapi/bundled.yaml

ARG BACKEND_PORT=8080
EXPOSE ${BACKEND_PORT}

# Run as non-root user (security best practice)
RUN adduser -D -g '' appuser
USER appuser

CMD ["./app"]