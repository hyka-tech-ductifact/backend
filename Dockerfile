FROM docker.io/library/golang:1.23-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o app ./cmd/api

# Final stage
FROM docker.io/library/alpine:latest

WORKDIR /app

# Install ca-certificates
RUN apk --no-cache add ca-certificates

# Copy the binary from builder stage
COPY --from=builder /app/app .

# Expose port (must match APP_PORT in .env)
ARG APP_PORT=8080
EXPOSE ${APP_PORT}

# Command to run
CMD ["./app"]