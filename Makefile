# Ductifact Backend Makefile

# Load .env if it exists (ignored in CI where env vars are set externally)
-include .env
export

.DEFAULT_GOAL := help

# ─── Variables ───────────────────────────────────────────────

# Test output format: pkgname, testdox, testname, dots, dots-v2,
# standard-quiet, standard-verbose, pkgname-and-test-fails
TEST_FORMAT ?= pkgname

# Enable CI mode: race detector + JUnit XML report (e.g. make test-unit CI=1)
CI ?= 0
_RACE_FLAG := $(if $(filter 1,$(CI)),-race,)
_junit_flag = $(if $(filter 1,$(CI)),--junitfile $(1)-test-results.xml,)

# Generate coverage report (e.g. make test-unit COVERAGE=1)
COVERAGE ?= 0

# GitHub org/user that owns the contracts repo
CONTRACTS_REPO ?= hyka-tech-ductifact/contracts

# ─── .PHONY ─────────────────────────────────────────────────

.PHONY: help \
	dev app-build app-start app-watch \
	db-start db-stop \
	test test-unit test-integration test-contract test-e2e test-clean \
	docker-build docker-start docker-stop \
	fmt lint deps clean \
	validate-branch fetch-contract

# ═══════════════════════════════════════════════════════════════
# Help
# ═══════════════════════════════════════════════════════════════

help:
	@echo "Available commands:"
	@echo ""
	@echo "  Development:"
	@echo "    dev              - Build, start DB and run with hot reload"
	@echo "    app-build        - Compile binary to bin/api"
	@echo "    app-start        - Build and start API in background"
	@echo "    app-watch        - Run app with hot reloading (air)"
	@echo ""
	@echo "  Database:"
	@echo "    db-start         - Start database in Docker"
	@echo "    db-stop          - Stop database"
	@echo ""
	@echo "  Testing:"
	@echo "    test             - Run all tests"
	@echo "    test-unit        - Run unit tests (no dependencies needed)"
	@echo "    test-integration - Run integration tests (requires DB)"
	@echo "    test-contract    - Run contract tests (requires DB + running server)"
	@echo "    test-e2e         - Run E2E tests (requires DB + running server)"
	@echo "    test-clean       - Clear Go test cache"
	@echo ""
	@echo "    Flags:"
	@echo "      CI=1         - CI mode (race detector + JUnit XML): make test-unit CI=1"
	@echo "      COVERAGE=1   - Generate coverage:                   make test-unit COVERAGE=1"
	@echo "      TEST_FORMAT  - Change output format:                make test-unit TEST_FORMAT=dots"
	@echo "        Formats: pkgname (default), testdox, testname, dots, dots-v2,"
	@echo "                 standard-quiet, standard-verbose, pkgname-and-test-fails"
	@echo ""
	@echo "  Contracts:"
	@echo "    fetch-contract   - Download bundled.yaml from contracts release"
	@echo ""
	@echo "  Docker (smoke test):"
	@echo "    docker-build     - Build Docker image"
	@echo "    docker-start     - Start app + DB in Docker (smoke test)"
	@echo "    docker-stop      - Stop Docker services"
	@echo ""
	@echo "  Code quality:"
	@echo "    fmt              - Format code"
	@echo "    lint             - Lint code"
	@echo "    deps             - Install dependencies and dev tools"
	@echo ""
	@echo "  CI:"
	@echo "    validate-branch  - Validate branch name and target"
	@echo ""
	@echo "  Maintenance:"
	@echo "    clean            - Remove build artifacts and temp files"

# ═══════════════════════════════════════════════════════════════
# Development
# ═══════════════════════════════════════════════════════════════

# Quick dev shortcut: build, start DB and run with hot reload
dev: ensure-contract app-build db-start app-watch

# Compile binary to bin/api
app-build:
	@echo "Building ductifact..."
	go build -o bin/api ./cmd/api

# Build and start API in background (used in CI and local testing)
app-start: ensure-contract app-build
	./bin/api &
	@sleep 3
	@echo "✅ API running in background"

# Run the application with hot reloading
app-watch:
	@echo "Running ductifact with hot reloading..."
	air

# ═══════════════════════════════════════════════════════════════
# Database
# ═══════════════════════════════════════════════════════════════

# Start database in Docker
db-start:
	@echo "Starting database..."
	docker compose up -d postgres --wait

# Stop database
db-stop:
	@echo "Stopping database..."
	docker compose down

# ═══════════════════════════════════════════════════════════════
# Testing
# ═══════════════════════════════════════════════════════════════

define run-tests
	@if [ "$(COVERAGE)" = "1" ]; then \
		gotestsum --format $(TEST_FORMAT) $(call _junit_flag,$(1)) -- $(_RACE_FLAG) -count=1 -coverprofile=coverage.out -coverpkg=./internal/... $(2) && \
		go tool cover -html=coverage.out -o coverage.html && \
		echo "" && \
		echo "─── Coverage Summary ───────────────────────────────" && \
		echo "" && \
		go tool cover -func=coverage.out | grep -v '100.0%' && \
		echo "" && \
		total=$$(go tool cover -func=coverage.out | grep '^total:' | awk '{print $$3}' | tr -d '%') && \
		echo "Total coverage: $${total}%" && \
		echo "" && \
		if command -v wslpath >/dev/null 2>&1; then \
			echo "📄 Coverage report: file:///$$(wslpath -m coverage.html)"; \
		else \
			echo "📄 Coverage report: $$(pwd)/coverage.html"; \
		fi; \
	else \
		gotestsum --format $(TEST_FORMAT) $(call _junit_flag,$(1)) -- $(_RACE_FLAG) -count=1 $(2); \
	fi
endef

# Run all tests
test: test-unit test-integration test-contract test-e2e

# Run unit tests — no dependencies needed
test-unit:
	@echo "Running unit tests..."
	$(call run-tests,unit,./test/unit/...)

# Run integration tests — requires DB running (make db-start)
test-integration:
	@echo "Running integration tests..."
	$(call run-tests,integration,./test/integration/...)

# Run contract tests — requires DB + server running (make db-start && make app-start)
test-contract:
	@echo "Running contract tests..."
	$(call run-tests,contract,./test/contract/...)

# Run E2E tests — requires DB + server running (make db-start && make app-start)
test-e2e:
	@echo "Running E2E tests..."
	$(call run-tests,e2e,./test/e2e/...)

# Clear Go test cache
test-clean:
	@echo "Clearing test cache..."
	go clean -testcache
	@echo "Test cache cleared!"

# ═══════════════════════════════════════════════════════════════
# Contracts
# ═══════════════════════════════════════════════════════════════

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

# Fetch contract only if bundled.yaml doesn't exist yet.
# Used by dev/app-start to avoid failing offline.
ensure-contract:
	@if [ ! -f contracts/openapi/bundled.yaml ]; then \
		echo "OpenAPI spec not found, fetching..."; \
		$(MAKE) fetch-contract; \
	fi

# ═══════════════════════════════════════════════════════════════
# Docker (smoke test — validates Dockerfile builds and app starts)
# ═══════════════════════════════════════════════════════════════

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker compose --profile smoke build

# Start app + DB in Docker (smoke test)
docker-start:
	@echo "Starting Docker services..."
	docker compose --profile smoke up -d
	@echo "Waiting for app to start..."
	@sleep 3
	@if [ "$$(docker inspect -f '{{.State.Running}}' ductifact_dev_app 2>/dev/null)" != "true" ]; then \
		echo "\n❌ App container is not running. Logs:"; \
		docker compose --profile smoke logs --tail=30 app; \
		exit 1; \
	fi
	@restart_count=$$(docker inspect -f '{{.RestartCount}}' ductifact_dev_app 2>/dev/null); \
	if [ "$$restart_count" -gt 0 ] 2>/dev/null; then \
		echo "\n❌ App crashed and restarted $$restart_count time(s). Logs:"; \
		docker compose --profile smoke logs --tail=30 app; \
		exit 1; \
	fi
	@echo "✓ Services running. App logs:"
	@docker compose --profile smoke logs --tail=10 app

# Stop Docker services
docker-stop:
	@echo "Stopping Docker services..."
	docker compose --profile smoke down

# ═══════════════════════════════════════════════════════════════
# Code quality
# ═══════════════════════════════════════════════════════════════

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	go vet ./...
	golangci-lint run

# Install dependencies and dev tools
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download
	@echo "Installing dev tools..."
	go install github.com/air-verse/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install gotest.tools/gotestsum@latest

# ═══════════════════════════════════════════════════════════════
# CI
# ═══════════════════════════════════════════════════════════════

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

# ═══════════════════════════════════════════════════════════════
# Maintenance
# ═══════════════════════════════════════════════════════════════

# Clean build artifacts and temporary files
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f *-test-results.xml
	rm -f api ductifact
	@echo "Cleaning test cache..."
	go clean -testcache
	@echo "Cleaning temporary files and directories..."
	rm -rf tmp/ temp/
	rm -f *.log *.tmp *.temp
	rm -f .DS_Store .DS_Store? ._*
	rm -f *.swp *.swo *~
	rm -f *.test *.out
	rm -f *.db *.sqlite *.sqlite3
	@echo "Cleaning Docker temporary files..."
	docker system prune -f 2>/dev/null || true
	docker image prune -f 2>/dev/null || true
	@echo "Removing empty directories..."
	find . -type d -empty -delete 2>/dev/null || true
	@echo "Cleanup completed!"
