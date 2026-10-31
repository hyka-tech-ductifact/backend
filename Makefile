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

# ─── OpenAPI contract ────────────────────────────────────────
CONTRACT_DIR     := contracts/openapi
CONTRACT_SPEC    := $(CONTRACT_DIR)/openapi.yaml
CONTRACT_BUNDLE  := $(CONTRACT_DIR)/bundled.yaml
CONTRACT_SOURCES := $(shell find $(CONTRACT_DIR) -name '*.yaml' ! -name bundled.yaml) internal/config/contract_version.go
CONTRACT_VERSION  = $(shell grep '^const ContractVersion' internal/config/contract_version.go | sed 's/.*"\(.*\)"/\1/')
REDOCLY_IMAGE    ?= redocly/cli:2.20.5
OASDIFF_IMAGE    ?= tufin/oasdiff:v1.35.0
BASE_REF         ?= main

# ─── .PHONY ─────────────────────────────────────────────────

.PHONY: help \
	dev dev-stop app-build app-start ensure-seed \
	services-start services-stop \
	test test-unit test-integration test-e2e test-clean \
	test-contract \
	contract-validate contract-bundle contract-breaking \
	docker-build docker-start docker-stop \
	fmt lint deps clean \
	validate-branch \
	changelog

# ═══════════════════════════════════════════════════════════════
# Help
# ═══════════════════════════════════════════════════════════════

help:
	@echo "Available commands:"
	@echo ""
	@echo "  Development:"
	@echo "    dev              - Start DB and run with hot reload (auto build)"
	@echo "    dev-stop         - Stop hot reload process and dev services"
	@echo "    app-build        - Compile binary to bin/api"
	@echo "    app-start        - Build and start API in background"
	@echo ""
	@echo "  Services:"
	@echo "    services-start   - Start dev dependencies (postgres, etc.)"
	@echo "    services-stop    - Stop dev dependencies"
	@echo ""
	@echo "  Testing:"
	@echo "    test             - Run all tests"
	@echo "    test-unit        - Run unit tests (no dependencies needed)"
	@echo "    test-integration - Run integration tests (requires DB)"
	@echo "    test-e2e         - Run E2E tests (requires running server)"
	@echo "    test-contract    - Run contract tests with Schemathesis (config: test/schemathesis.toml)"
	@echo "                       Local: 20 examples/op. CI: 100 examples/op."
	@echo "    test-clean       - Clear Go test cache"
	@echo ""
	@echo "    Flags:"
	@echo "      CI=1              - CI mode (race detector + JUnit XML): make test-unit CI=1"
	@echo "      COVERAGE=1        - Generate coverage:                   make test-unit COVERAGE=1"
	@echo "      TEST_FORMAT       - Change output format:                make test-unit TEST_FORMAT=dots"
	@echo "        Formats: pkgname (default), testdox, testname, dots, dots-v2,"
	@echo "                 standard-quiet, standard-verbose, pkgname-and-test-fails"
	@echo "      ST_MAX_EXAMPLES   - Override examples per operation:     make test-contract ST_MAX_EXAMPLES=200"
	@echo "      ST_SEED           - Reproduce a test run with a seed:     make test-contract ST_SEED=<seed>"
	@echo ""
	@echo "  Contracts (source: contracts/openapi/, version: internal/config/contract_version.go):"
	@echo "    contract-validate - Lint the OpenAPI spec (redocly)"
	@echo "    contract-bundle   - Generate contracts/openapi/bundled.yaml (only if sources changed)"
	@echo "    contract-breaking - Detect breaking changes vs BASE_REF (default: main)"
	@echo "                        make contract-breaking BASE_REF=origin/main"
	@echo ""
	@echo "  Docker (smoke test):"
	@echo "    docker-build     - Build Docker image"
	@echo "    docker-start     - Build + start app & DB in Docker"
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
	@echo "  Release:"
	@echo "    changelog        - Generate CHANGELOG.md (auto-bumps version via SemVer)"
	@echo "                       Override: make changelog VERSION=v1.0.0"
	@echo ""
	@echo "  Maintenance:"
	@echo "    clean            - Remove build artifacts and temp files"

# ═══════════════════════════════════════════════════════════════
# Development
# ═══════════════════════════════════════════════════════════════

# Start DB and run with hot reload (main dev workflow)
dev: contract-bundle services-start ensure-seed
	@echo "Running ductifact with hot reloading..."
	air

# Stop hot reload process and dev services
dev-stop: services-stop
	@echo "Stopping air and dev services..."
	@pkill -f '[a]ir' 2>/dev/null || true
	@pkill -f 'bin/[a]pi' 2>/dev/null || true
	@echo "✅ Dev environment stopped"

# Compile binary to bin/api
app-build:
	@echo "Building ductifact..."
	go build -o bin/api ./cmd/api

# Build and start API in background (used in CI and local testing)
app-start: contract-bundle app-build services-start
	./bin/api &
	@sleep 3
	@echo "✅ API running in background"

# Seed the database with development data if the users table is empty.
# Runs automatically as part of `make dev` — no manual invocation needed.
# If you need to reset: make clean && make dev
# Uses psql inside the Postgres container — no local psql install needed.
ensure-seed:
	@count=$$(docker exec ductifact_dev_postgres psql -U $(DB_USER) -d $(DB_NAME) -tAc "SELECT COUNT(*) FROM users" 2>/dev/null); \
	if [ "$$count" = "0" ]; then \
		echo "Seeding database..."; \
		docker cp test/seed.sql ductifact_dev_postgres:/tmp/seed.sql; \
		docker exec ductifact_dev_postgres psql -U $(DB_USER) -d $(DB_NAME) -f /tmp/seed.sql -q; \
		echo "✅ Seed data loaded (alice@ductifact.dev / bob@ductifact.dev — password: password123)"; \
	fi

# ═══════════════════════════════════════════════════════════════
# Services (dev dependencies)
# ═══════════════════════════════════════════════════════════════

# Start dev dependencies (postgres, etc.)
services-start:
	@echo "Starting services..."
	docker compose up -d --wait

# Stop dev dependencies
services-stop:
	@echo "Stopping services..."
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

# Run all tests (contract tests run separately: make test-contract)
test: test-unit test-integration test-e2e

# Run unit tests — no dependencies needed
test-unit:
	@echo "Running unit tests..."
	$(call run-tests,unit,./test/unit/...)

# Run integration tests — requires services running (make services-start)
test-integration:
	@echo "Running integration tests..."
	$(call run-tests,integration,./test/integration/...)

# Run E2E tests — requires server running (make app-start)
test-e2e:
	@echo "Running E2E tests..."
	$(call run-tests,e2e,./test/e2e/...)

# ─── Schemathesis (contract testing) ─────────────────────────
# Runs via Docker — config lives in test/schemathesis.toml
ST_IMAGE         ?= schemathesis/schemathesis:v4.16.1
ST_MAX_EXAMPLES  ?= $(if $(filter 1,$(CI)),100,20)
_ST_SEED_FLAG    := $(if $(ST_SEED),--seed $(ST_SEED),)

# Run contract tests with Schemathesis against the OpenAPI spec.
# Requires: running API server (make dev) and Docker.
# Auth: logs in with the seeded dev user; seed is ensured before login.
test-contract: contract-bundle ensure-seed
	@echo "Running contract tests (Schemathesis)..."
	@mkdir -p schemathesis-report
	@docker exec ductifact_dev_postgres psql -U $(DB_USER) -d $(DB_NAME) -v ON_ERROR_STOP=1 -c "INSERT INTO users (id, name, email, password_hash, locale, created_at, updated_at, deleted_at) VALUES (gen_random_uuid(), 'Schemathesis Bot', 'st@test.ductifact.dev', CHR(36) || '2a' || CHR(36) || '10' || CHR(36) || 'rEhzOFFg9EHdgq9lnGw2hulMCZSXU4RzeYV0iuLWKoVAs8Snqs9yy', 'en', NOW(), NOW(), NULL) ON CONFLICT (email) WHERE deleted_at IS NULL DO UPDATE SET name = EXCLUDED.name, password_hash = CHR(36) || '2a' || CHR(36) || '10' || CHR(36) || 'rEhzOFFg9EHdgq9lnGw2hulMCZSXU4RzeYV0iuLWKoVAs8Snqs9yy', locale = EXCLUDED.locale, updated_at = NOW(), deleted_at = NULL;" >/dev/null
	@TOKEN=$$(curl -sf http://localhost:8080/v1/auth/login \
		-H 'Content-Type: application/json' \
		-d '{"email":"st@test.ductifact.dev","password":"password123"}' \
		| grep -o '"access_token":"[^"]*"' | cut -d'"' -f4); \
	if [ -z "$$TOKEN" ]; then \
		echo "❌ Could not obtain auth token — is the API running and seeded?"; exit 1; \
	fi; \
	echo "  Auth token obtained ✅"; \
	docker run --rm --network host \
		--user 0:0 \
		-e SCHEMATHESIS_HOOKS=schemathesis_hooks \
		-v $(CURDIR)/$(CONTRACT_BUNDLE):/spec/bundled.yaml:ro \
		-v $(CURDIR)/test/schemathesis.toml:/spec/schemathesis.toml:ro \
		-v $(CURDIR)/test/schemathesis_hooks.py:/spec/schemathesis_hooks.py:ro \
		-v $(CURDIR)/schemathesis-report:/spec/schemathesis-report \
		-w /spec \
		$(ST_IMAGE) \
		run bundled.yaml \
		-H "Authorization: Bearer $$TOKEN" \
		--max-examples $(ST_MAX_EXAMPLES) \
		$(_ST_SEED_FLAG)
	@echo "✅ Contract tests passed"

# Clear Go test cache
test-clean:
	@echo "Clearing test cache..."
	go clean -testcache
	@echo "Test cache cleared!"

# ═══════════════════════════════════════════════════════════════
# Contracts (OpenAPI)
# ═══════════════════════════════════════════════════════════════
# Source of truth: contracts/openapi/ + ContractVersion (Go const).
# bundled.yaml is a generated artifact (git-ignored), consumed by
# /docs, Schemathesis and the Docker image.

# Lint and validate the OpenAPI spec
contract-validate:
	@echo "Validating OpenAPI spec..."
	docker run --rm -v $(CURDIR)/$(CONTRACT_DIR):/spec:ro $(REDOCLY_IMAGE) lint /spec/openapi.yaml

# Resolve all $ref into bundled.yaml and stamp it with ContractVersion.
# Rebuilt only when a spec file or contract_version.go changes.
contract-bundle: $(CONTRACT_BUNDLE)

$(CONTRACT_BUNDLE): $(CONTRACT_SOURCES)
	@echo "Bundling OpenAPI spec v$(CONTRACT_VERSION)..."
	@docker run --rm -v $(CURDIR)/$(CONTRACT_DIR):/spec $(REDOCLY_IMAGE) bundle /spec/openapi.yaml -o /spec/bundled.yaml
	@sed -i '0,/^  version: .*/s//  version: $(CONTRACT_VERSION)/' $@
	@echo "✅ $@ (v$(CONTRACT_VERSION))"

# Detect breaking changes against BASE_REF (branch, tag or commit).
# Skips when BASE_REF has no contract yet (first PR after the move).
contract-breaking: contract-bundle
	@if ! git cat-file -e "$(BASE_REF):$(CONTRACT_SPEC)" 2>/dev/null; then \
		echo "⚠️  $(BASE_REF) has no $(CONTRACT_SPEC) — skipping breaking-change check"; exit 0; \
	fi; \
	BASE_DIR=$$(mktemp -d); \
	git archive "$(BASE_REF)" -- $(CONTRACT_DIR) | tar -x -C "$$BASE_DIR"; \
	docker run --rm -v "$$BASE_DIR/$(CONTRACT_DIR)":/spec $(REDOCLY_IMAGE) bundle /spec/openapi.yaml -o /spec/bundled.yaml >/dev/null 2>&1; \
	docker run --rm \
		-v "$$BASE_DIR/$(CONTRACT_DIR)":/base:ro \
		-v $(CURDIR)/$(CONTRACT_DIR):/spec:ro \
		$(OASDIFF_IMAGE) breaking --fail-on ERR /base/bundled.yaml /spec/bundled.yaml; \
	EXIT=$$?; rm -rf "$$BASE_DIR"; \
	if [ $$EXIT -eq 0 ]; then echo "✅ No breaking changes vs $(BASE_REF)"; \
	else echo "❌ Breaking changes found vs $(BASE_REF)"; fi; \
	exit $$EXIT

# ═══════════════════════════════════════════════════════════════
# Docker (validates Dockerfile builds and app starts)
# ═══════════════════════════════════════════════════════════════

# Build Docker image
docker-build: contract-bundle
	@echo "Building Docker image..."
	docker compose --profile smoke build \
		--build-arg APP_COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo unknown) \
		--build-arg APP_BUILD_TIME=$$(date -u +%Y-%m-%dT%H:%M:%SZ)

# Build + start postgres & app in Docker (smoke test)
docker-start: docker-build
	@echo "Starting smoke test..."
	docker compose --profile smoke up -d --wait
	@echo "Waiting for app to be ready..."
	@sleep 2
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
	@echo "✓ Smoke test passed. App logs:"
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
	golines -m 120 -w --no-reformat-tags .
	go fmt ./...

# Lint code and OpenAPI spec
lint: contract-validate
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
	@PATTERN='^(feat|fix|chore|hotfix|dependabot)/.+$$'; \
	if [ -z "$$BRANCH" ]; then \
		echo "❌ BRANCH env var is required"; exit 1; \
	fi; \
	if ! echo "$$BRANCH" | grep -qE "$$PATTERN"; then \
		echo "❌ Branch '$$BRANCH' does not match: $$PATTERN"; \
		echo "   Expected: feat/*, fix/*, chore/*, hotfix/*"; \
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
# Release
# ═══════════════════════════════════════════════════════════════

# Generate CHANGELOG.md from conventional commits using git-cliff.
# By default, calculates the next version automatically via SemVer:
#   fix  → patch (0.2.0 → 0.2.1)
#   feat → minor (0.2.0 → 0.3.0)
#   feat! / BREAKING CHANGE → major (0.2.0 → 1.0.0)
# Override: make changelog VERSION=v1.0.0
changelog:
	@echo "Generating CHANGELOG.md..."
	$(if $(VERSION),git-cliff --tag $(VERSION) --output CHANGELOG.md,git-cliff --bump --output CHANGELOG.md)
	@echo "✅ CHANGELOG.md updated"

# ═══════════════════════════════════════════════════════════════
# Maintenance
# ═══════════════════════════════════════════════════════════════

# Clean all untracked and ignored files, except .env (contains local secrets).
# Uses git as the source of truth — anything not in git is an artifact.
# Also removes Docker volumes (database data) for a full reset.
clean:
	@echo "Cleaning all generated files..."
	git clean -fdx --exclude=.env
	go clean -testcache
	docker compose down -v 2>/dev/null || true
	@echo "✅ Cleanup completed!"
