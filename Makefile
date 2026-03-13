# Ductifact Backend Makefile

.PHONY: help app-build app-run app-start app-watch app-test test test-unit test-integration test-e2e test-clean clean db-start db-stop prod-start prod-stop prod-build fmt lint deps validate-branch fetch-contract COVERAGE RACE

# Allow COVERAGE to be used as a flag (e.g. make test-unit COVERAGE)
COVERAGE: ;
	@:
_COVERAGE := $(filter COVERAGE,$(MAKECMDGOALS))

# Allow RACE to be used as a flag (e.g. make test-unit RACE)
# CI sets RACE=1 as env var; locally you can also use: make test-unit RACE
RACE: ;
	@:
_RACE := $(or $(filter RACE,$(MAKECMDGOALS)),$(if $(filter 1,$(RACE)),RACE))
_RACE_FLAG := $(if $(_RACE),-race,)

# Default target
help:
	@echo "Available commands:"
	@echo ""
	@echo "  Development:"
	@echo "    db-start       - Start database in Docker"
	@echo "    db-stop        - Stop database"
	@echo "    app-watch      - Run app with hot reloading (air), compiling and running"
	@echo "    app-build      - Compile binary to bin/api"
	@echo "    app-start      - Build and start API in background"
	@echo "    app-run        - Run app without hot reloading"
	@echo "    dev            - Build, start DB and run with hot reload"
	@echo ""
	@echo "  Testing:"
	@echo "    test-unit          - Run unit tests (no dependencies needed)"
	@echo "    test-integration   - Run integration tests (requires DB)"
	@echo "    test-contract      - Run contract tests (requires DB + running server)"
	@echo "    test-e2e           - Run E2E tests (requires DB + running server)"
	@echo "    test               - Run all tests"
	@echo "    test-clean         - Clear Go test cache"
	@echo ""
	@echo "    Add RACE to enable the race detector (slower, used in CI):"
	@echo "    make test-unit RACE"
	@echo ""
	@echo "    Add COVERAGE to any test target to generate a coverage report:"
	@echo "    make test-unit COVERAGE"
	@echo "    make test COVERAGE"
	@echo ""
	@echo "    Change test output format with TEST_FORMAT:"
	@echo "    make test-unit TEST_FORMAT=dots"
	@echo "    Formats: testdox (default), pkgname, testname, dots, dots-v2,"
	@echo "             standard-quiet, standard-verbose, pkgname-and-test-fails"
	@echo ""
	@echo "  Contracts:"
	@echo "    fetch-contract     - Download bundled.yaml from contracts release"
	@echo ""
	@echo "  Production:"
	@echo "    prod-build     - Build Docker images"
	@echo "    prod-start     - Start app + DB in Docker"
	@echo "    prod-stop      - Stop production services"
	@echo ""
	@echo "  Code quality:"
	@echo "    fmt            - Format code"
	@echo "    lint           - Lint code"
	@echo "    deps           - Install dependencies"
	@echo "    clean          - Remove build artifacts"

# Quick dev shortcut: build, start DB and run with hot reload
dev: app-build db-start app-watch

# Compile binary to bin/api
app-build:
	@echo "Building ductifact..."
	go build -o bin/api ./cmd/api

# Run the application
app-run:
	@echo "Running ductifact..."
	go run ./cmd/api

# Build and start API in background (used in CI and local testing)
app-start: app-build
	./bin/api &
	@sleep 3
	@echo "✅ API running in background"

# Run the application with hot reloading
app-watch:
	@echo "Running ductifact with hot reloading..."
	air

# Test format: testdox , pkgname (default), standard, dots, short, etc.
TEST_FORMAT ?= pkgname

define run-tests
	@if [ -n "$(_COVERAGE)" ]; then \
		gotestsum --format $(TEST_FORMAT) -- $(_RACE_FLAG) -count=1 -coverprofile=coverage.out -coverpkg=./internal/... $(1) && \
		go tool cover -func=coverage.out && \
		go tool cover -html=coverage.out -o coverage.html && \
		echo "" && \
		echo "Coverage report generated: coverage.html" && \
		explorer.exe "$$(wslpath -w coverage.html)" || true; \
	else \
		gotestsum --format $(TEST_FORMAT) -- $(_RACE_FLAG) -count=1 $(1); \
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

# Run contract tests — requires DB + server running (make db-start && make app-run)
test-contract:
	@echo "Running contract tests..."
	$(call run-tests,./test/contract/...)

# Run E2E tests — requires DB + server running (make db-start && make app-run)
test-e2e:
	@echo "Running E2E tests..."
	$(call run-tests,./test/e2e/...)

# Run all tests
test: test-unit test-integration test-contract test-e2e

# Clear Go test cache
test-clean:
	@echo "Clearing test cache..."
	go clean -testcache
	@echo "Test cache cleared!"

# Start database in Docker
db-start:
	@echo "Starting database..."
	docker compose -f docker-compose.dev.yml up -d --wait

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
	@echo "Cleaning test cache..."
	go clean -testcache
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
	docker compose build

# Start app + DB in Docker
prod-start:
	@echo "Starting production services..."
	docker compose up -d
	@echo "Waiting for app to start..."
	@sleep 3
	@if [ "$$(docker inspect -f '{{.State.Running}}' microservice_app 2>/dev/null)" != "true" ]; then \
		echo "\n❌ App container is not running. Logs:"; \
		docker compose logs --tail=30 app; \
		exit 1; \
	fi
	@restart_count=$$(docker inspect -f '{{.RestartCount}}' microservice_app 2>/dev/null); \
	if [ "$$restart_count" -gt 0 ] 2>/dev/null; then \
		echo "\n❌ App crashed and restarted $$restart_count time(s). Logs:"; \
		docker compose logs --tail=30 app; \
		exit 1; \
	fi
	@echo "✓ Services running. App logs:"
	@docker compose logs --tail=10 app

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
	go vet ./...
	golangci-lint run

# Validate branch name and target (used in CI)
# Requires BRANCH and BASE env vars (set by GitHub Actions).
# BRANCH = source branch name (e.g. feat/add-login)
# BASE   = target branch name (e.g. main) — only needed for target validation
validate-branch:
	@PATTERN='^(feat|fix|chore|docs|hotfix|test|refactor)/.+$$'; \
	if [ -z "$$BRANCH" ]; then \
		echo "❌ BRANCH env var is required"; exit 1; \
	fi; \
	if ! echo "$$BRANCH" | grep -qE "$$PATTERN"; then \
		echo "❌ Branch '$$BRANCH' does not match: $$PATTERN"; \
		echo "   Expected: feat/*, fix/*, chore/*, docs/*, hotfix/*, test/*, refactor/*"; \
		exit 1; \
	fi; \
	if [ -n "$$BASE" ]; then \
		if echo "$$BRANCH" | grep -qE '^hotfix/' && [ "$$BASE" != "release" ]; then \
			echo "❌ hotfix/* branches must target 'release', not '$$BASE'"; exit 1; \
		fi; \
		if [ "$$BASE" = "release" ] && ! echo "$$BRANCH" | grep -qE '^hotfix/'; then \
			echo "❌ Only hotfix/* branches can target 'release', got '$$BRANCH'"; exit 1; \
		fi; \
	fi; \
	echo "✅ Branch '$$BRANCH' → '$${BASE:-*}' is valid"

# Install dependencies and dev tools
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download
	@echo "Installing dev tools..."
	go install github.com/air-verse/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install gotest.tools/gotestsum@latest

# ─── Contracts ───────────────────────────────────────────────

# GitHub org/user that owns the contracts repo
CONTRACTS_REPO ?= hyka-tech-ductifact/contracts

# Download bundled.yaml from the contracts GitHub Release matching CONTRACT_VERSION.
# Requires CONTRACT_VERSION env var (from .env or CI).
# The file is saved to contracts/openapi/bundled.yaml (git-ignored).
fetch-contract:
	@if [ -z "$$CONTRACT_VERSION" ]; then \
		echo "❌ CONTRACT_VERSION env var is required"; \
		echo "   Set it in .env or export it: export CONTRACT_VERSION=0.1.0"; \
		exit 1; \
	fi; \
	echo "Fetching contracts v$$CONTRACT_VERSION..."; \
	mkdir -p contracts/openapi; \
	curl -fsSL \
		"https://github.com/$(CONTRACTS_REPO)/releases/download/v$$CONTRACT_VERSION/bundled.yaml" \
		-o contracts/openapi/bundled.yaml; \
	echo "✅ contracts/openapi/bundled.yaml (v$$CONTRACT_VERSION)"
