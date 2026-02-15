# Event Service Makefile

.PHONY: help build run dev test test-coverage clean docker-build docker-run docker-stop fmt lint deps test-docker test-start test-stop dev-start dev-stop

# Default target
help:
	@echo "Available commands:"
	@echo "  build          - Build the application"
	@echo "  run            - Run the application"
	@echo "  dev            - Run with hot reloading (air)"
	@echo "  test           - Run all tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  test-docker    - Run tests in Docker with database"
	@echo "  test-start     - Start test services (database + test runner)"
	@echo "  test-stop      - Stop test services"
	@echo "  dev-start      - Start development database only"
	@echo "  dev-stop       - Stop development database"
	@echo "  clean          - Clean build artifacts"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run with Docker Compose"
	@echo "  docker-stop    - Stop Docker Compose services"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  deps           - Install dependencies"

# Build the application
build:
	@echo "Building event-service..."
	go build -o bin/api ./cmd/api

# Run the application
run:
	@echo "Running event-service..."
	go run ./cmd/api

# Run the application with hot reloading
dev:
	@echo "Running event-service with hot reloading..."
	air

# Run all tests
test:
	@echo "Running all tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests in Docker with proper database connection
test-docker:
	@echo "Running tests in Docker with database..."
	@echo "Starting test database..."
	@docker compose -f docker-compose.test.yml up -d test-db
	@echo "Waiting for database to be ready..."
	@sleep 5
	@echo "Running tests..."
	@docker run --rm --network event-service_default \
		-e DB_HOST=test-db -e DB_PORT=5432 -e DB_USER=test_user \
		-e DB_PASSWORD=test_password -e DB_NAME=event_service_test \
		event-service-test make test
	@echo "Cleaning up..."
	@docker compose -f docker-compose.test.yml down

# Start test services (database + test runner)
test-start:
	@echo "Starting test services..."
	docker compose -f docker-compose.test.yml up -d

# Stop test services
test-stop:
	@echo "Stopping test services..."
	docker compose -f docker-compose.test.yml down

# Start development database only
dev-start:
	@echo "Starting development database..."
	docker compose -f docker-compose.dev.yml up -d

# Stop development database
dev-stop:
	@echo "Stopping development database..."
	docker compose -f docker-compose.dev.yml down

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f api event-service
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
docker-build:
	@echo "Building Docker image..."
	docker build -t event-service .

# Run with Docker Compose
docker-run:
	@echo "Starting services with Docker Compose..."
	docker compose up -d

# Stop Docker Compose
docker-stop:
	@echo "Stopping Docker Compose services..."
	docker compose down

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download
