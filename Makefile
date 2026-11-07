# ==============================================================================
# Halisi Makefile
# Developer-friendly automation for local development, testing, and builds.
# ==============================================================================

GO ?= go
TEMPL ?= templ
SQLC ?= sqlc
MIGRATE ?= migrate
DOCKER ?= docker
DOCKER_COMPOSE ?= docker compose

MIGRATE_VERSION ?= v4.19.1
BINARY_NAME ?= halisi-server
BUILD_DIR ?= ./bin
GO_PACKAGES ?= ./...
GO_SOURCE_DIRS ?= cmd internal web

COVERAGE_FILE ?= coverage.out
COVERAGE_SUMMARY ?= coverage-summary.txt
GOVULNCHECK_VERSION ?= v1.8.0

# Automatically detect race detector flag based on CGO support
RACE_FLAG ?= $(if $(filter 1,$(shell go env CGO_ENABLED)),-race,)

DOCKER_IMAGE ?= halisi:latest
DOCKER_CONTAINER ?= halisi
DOCKER_PORT ?= 8080
DOCKER_ENV_FILE ?= .env

.DEFAULT_GOAL := help

.PHONY: help install-tools up down logs ps db-shell generate templ templ-watch sqlc migrate-up migrate-down run dev tidy build clean fmt fmt-check mod-check vet quality test security check docker-build docker-run ci

# ------------------------------------------------------------------------------
# 📖 Help
# ------------------------------------------------------------------------------
help: ## Show this help message
	@awk 'BEGIN {FS = ":.*## "; printf "\nUsage: make \033[36m<target>\033[0m\n\n"} /^[a-zA-Z0-9_-]+:.*## / {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ------------------------------------------------------------------------------
# 🛠️ Tools Installation
# ------------------------------------------------------------------------------
install-tools: ## Install development CLI tools (templ, sqlc, and migrate)
	@echo "Installing templ CLI..."
	@$(GO) install github.com/a-h/templ/cmd/templ@latest
	@echo "Installing sqlc CLI..."
	@$(GO) install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@echo "Installing golang-migrate CLI..."
	@$(GO) install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION)
	@echo "Tools installed successfully."

# ------------------------------------------------------------------------------
# 🐳 Local Infrastructure (PostgreSQL & Mailpit)
# ------------------------------------------------------------------------------
up: ## Start local PostgreSQL and Mailpit in background
	$(DOCKER_COMPOSE) up -d postgres mailpit

down: ## Stop local containers
	$(DOCKER_COMPOSE) down

logs: ## Follow logs of local containers
	$(DOCKER_COMPOSE) logs -f

ps: ## Check status of local containers
	$(DOCKER_COMPOSE) ps

db-shell: ## Open interactive psql shell in local PostgreSQL container
	$(DOCKER_COMPOSE) exec postgres psql -U postgres -d halisi

# ------------------------------------------------------------------------------
# ⚡ Code Generation
# ------------------------------------------------------------------------------
generate: templ sqlc ## Run all code generators (templ & sqlc)

templ: ## Compile .templ templates into type-safe Go code
	$(TEMPL) generate

templ-watch: ## Watch .templ files and automatically recompile on save
	$(TEMPL) generate --watch

sqlc: ## Compile SQL queries into type-safe Go repository code
	$(SQLC) generate

migrate-up: ## Apply all pending database migrations (requires DATABASE_URL)
	$(MIGRATE) -path internal/database/migrations -database "$(DATABASE_URL)" up

migrate-down: ## Roll back the latest database migration (requires DATABASE_URL)
	$(MIGRATE) -path internal/database/migrations -database "$(DATABASE_URL)" down 1

# ------------------------------------------------------------------------------
# 🚀 Development & Build
# ------------------------------------------------------------------------------
dev: generate run ## Generate code and run local development server
 
run: ## Run the HTTP server locally (requires .env)
	$(GO) run ./cmd/api

tidy: ## Download dependencies and tidy go.mod / go.sum
	$(GO) mod download
	$(GO) mod tidy

build: ## Build the standalone server binary into bin/
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/api

clean: ## Remove build artifacts, coverage files, and temporary test files
	@rm -rf $(BUILD_DIR) $(COVERAGE_FILE) $(COVERAGE_SUMMARY)

# ------------------------------------------------------------------------------
# 🧪 Quality, Linting & Testing
# ------------------------------------------------------------------------------
fmt: ## Format Go code and templ templates
	gofmt -w $(GO_SOURCE_DIRS)
	@if command -v $(TEMPL) >/dev/null 2>&1; then $(TEMPL) fmt .; fi

fmt-check: ## Verify all Go source files are formatted
	@unformatted="$$(gofmt -l $(GO_SOURCE_DIRS))"; \
	if [ -n "$$unformatted" ]; then \
		printf '\033[31mError: The following files require formatting (run "make fmt"):\033[0m\n%s\n' "$$unformatted"; \
		exit 1; \
	fi

mod-check: ## Verify module dependencies integrity
	$(GO) mod download
	$(GO) mod verify
	$(GO) mod tidy -diff

vet: ## Run Go static code analysis (vet)
	$(GO) vet $(GO_PACKAGES)

quality: fmt-check mod-check vet ## Run formatting, dependency, and vet checks

test: ## Run unit tests with race detection (when supported) and coverage
	$(GO) test $(RACE_FLAG) -count=1 -timeout=5m -coverprofile=$(COVERAGE_FILE) $(GO_PACKAGES)
	@$(GO) tool cover -func=$(COVERAGE_FILE) | tee $(COVERAGE_SUMMARY)

security: ## Check for reachable known vulnerabilities using govulncheck
	$(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) $(GO_PACKAGES)

check: quality test security ## Run code quality, tests, and security scans

# ------------------------------------------------------------------------------
# 📦 Docker Container Packaging
# ------------------------------------------------------------------------------
docker-build: ## Build application Docker container image
	docker build --tag $(DOCKER_IMAGE) .

docker-run: docker-build ## Build and run container locally with .env
	@if [ -f "$(DOCKER_ENV_FILE)" ]; then \
		docker run --rm --name $(DOCKER_CONTAINER) --publish $(DOCKER_PORT):8080 --env-file $(DOCKER_ENV_FILE) $(DOCKER_IMAGE); \
	else \
		docker run --rm --name $(DOCKER_CONTAINER) --publish $(DOCKER_PORT):8080 $(DOCKER_IMAGE); \
	fi

# ------------------------------------------------------------------------------
# 🚀 Continuous Integration (CI)
# ------------------------------------------------------------------------------
ci: quality test security docker-build ## Mirror all GitHub Actions CI checks locally
