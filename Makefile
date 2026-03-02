# Ductifact Backend Makefile

.PHONY: help app-build app-run app-watch app-test test test-unit test-integration test-e2e clean db-start db-stop prod-start prod-stop prod-build fmt lint deps

# Default target
help:
	@echo "Available commands:"
	@echo ""
	@echo "  Development:"
	@echo "    db-start       - Start database in Docker"
	@echo "    db-stop        - Stop database"
	@echo "    app-watch      - Run app with hot reloading (air)"
	@echo "    app-run        - Run app without hot reloading"
	@echo "    app-build      - Compile binary to bin/api"
	@echo ""
	@echo "  Testing:"
	@echo "    test-unit          - Run unit tests (no dependencies needed)"
	@echo "    test-integration   - Run integration tests (requires DB)"
	@echo "    test-e2e           - Run E2E tests (requires DB + running server)"
	@echo "    test               - Run all tests"
	@echo ""
	@echo "    Add COVERAGE to any test target to generate a coverage report:"
	@echo "    make test-unit COVERAGE"
	@echo "    make test COVERAGE"
	@echo ""
	@echo "  Production:"
	@echo "    prod-start     - Start app + DB in Docker"
	@echo "    prod-stop      - Stop production services"
	@echo "    prod-build     - Build Docker image"
	@echo ""
	@echo "  Code quality:"
	@echo "    fmt            - Format code"
	@echo "    lint           - Lint code"
	@echo "    deps           - Install dependencies"
	@echo "    clean          - Remove build artifacts"

# Compile binary to bin/api
app-build:
	@echo "Building ductifact..."
	go build -o bin/api ./cmd/api

# Run the application
app-run:
	@echo "Running ductifact..."
	go run ./cmd/api

# Run the application with hot reloading
app-watch:
	@echo "Running ductifact with hot reloading..."
	air

# Coverage flag: run with COVERAGE to generate a report (e.g. make test-unit COVERAGE)
_COVERAGE := $(filter command line,$(origin COVERAGE))

define run-tests
	@if [ -n "$(_COVERAGE)" ]; then \
		go test -v -coverprofile=coverage.out $(1) && \
		go tool cover -html=coverage.out -o coverage.html && \
		echo "Coverage report generated: coverage.html"; \
	else \
		go test -v $(1); \
	fi
endef

# Run unit tests — no dependencies needed
test-unit:
	@echo "Running unit tests..."
	$(call run-tests,./test/unit/...)

# Run integration tests — requires DB running (make db-start)
test-integration:
	@echo "Running integration tests..."
	$(call run-tests,./test/integration/...)

# Run E2E tests — requires DB + server running (make db-start && make app-run)
test-e2e:
	@echo "Running E2E tests..."
	$(call run-tests,./test/e2e/...)

# Run all tests
test: test-unit test-integration test-e2e

# Start database in Docker
db-start:
	@echo "Starting database..."
	docker compose -f docker-compose.dev.yml up -d

# Stop database
db-stop:
	@echo "Stopping database..."
	docker compose -f docker-compose.dev.yml down

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f api ductifact
	@echo "Cleaning temporary files and directories..."
	rm -rf tmp/
	rm -rf temp/
	rm -f *.log
	rm -f *.tmp
	rm -f *.temp
	rm -f .DS_Store
	rm -f .DS_Store?
	rm -f ._*
	rm -f *.swp
	rm -f *.swo
	rm -f *~
	rm -f *.test
	rm -f *.out
	rm -f *.db
	rm -f *.sqlite
	rm -f *.sqlite3
	@echo "Cleaning Docker temporary files..."
	docker system prune -f 2>/dev/null || true
	docker image prune -f 2>/dev/null || true
	@echo "Removing empty directories..."
	find . -type d -empty -delete 2>/dev/null || true
	@echo "Cleanup completed!"

# Build Docker image
prod-build:
	@echo "Building Docker image..."
	docker build -t ductifact .

# Start app + DB in Docker
prod-start:
	@echo "Starting production services..."
	docker compose up -d

# Stop production services
prod-stop:
	@echo "Stopping production services..."
	docker compose down

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Install dependencies and dev tools
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download
	@echo "Installing dev tools..."
	go install github.com/air-verse/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
