GO ?= go
GO_PACKAGES ?= ./...
GO_SOURCE_DIRS ?= cmd internal
CGO_ENABLED ?= 1

COVERAGE_FILE ?= coverage.out
COVERAGE_SUMMARY ?= coverage-summary.txt
GOVULNCHECK_VERSION ?= v1.8.0
DOCKER_IMAGE ?= halisi:local
DOCKER_CONTAINER ?= halisi
DOCKER_PORT ?= 8080
DOCKER_ENV_FILE ?= .env

export CGO_ENABLED

.DEFAULT_GOAL := help

.PHONY: help fmt fmt-check mod-check vet quality build test security check ci run docker-build docker-run

help: ## Show the available commands
	@awk 'BEGIN {FS = ":.*## "; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

fmt: ## Format all Go source files
	gofmt -w $(GO_SOURCE_DIRS)

fmt-check: ## Fail if any Go source file needs formatting
	@unformatted="$$(gofmt -l $(GO_SOURCE_DIRS))"; \
	if [ -n "$$unformatted" ]; then \
		printf 'Run make fmt on these files:\n%s\n' "$$unformatted"; \
		exit 1; \
	fi

mod-check: ## Verify dependencies and confirm go.mod/go.sum are tidy
	$(GO) mod download
	$(GO) mod verify
	$(GO) mod tidy -diff

vet: ## Run Go static analysis
	$(GO) vet $(GO_PACKAGES)

quality: fmt-check mod-check vet ## Run formatting, dependency, and vet checks

build: ## Compile every Go package
	$(GO) build $(GO_PACKAGES)

test: ## Run race-enabled tests and generate coverage files
	$(GO) test -race -count=1 -timeout=5m -coverprofile=$(COVERAGE_FILE) $(GO_PACKAGES)
	$(GO) tool cover -func=$(COVERAGE_FILE) | tee $(COVERAGE_SUMMARY)

security: ## Check for reachable known vulnerabilities
	$(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) $(GO_PACKAGES)

check: quality build test security ## Run all Go checks used by CI

ci: check docker-build ## Run every CI check, including the Docker build

run: ## Run the application locally
	$(GO) run ./cmd/api

docker-build: ## Build the application container image
	docker build --tag $(DOCKER_IMAGE) .

docker-run: docker-build ## Build and run the container (uses local .env file if present)
	@if [ -f "$(DOCKER_ENV_FILE)" ]; then \
		docker run --rm --name $(DOCKER_CONTAINER) --publish $(DOCKER_PORT):8080 --env-file $(DOCKER_ENV_FILE) $(DOCKER_IMAGE); \
	else \
		docker run --rm --name $(DOCKER_CONTAINER) --publish $(DOCKER_PORT):8080 $(DOCKER_IMAGE); \
	fi
