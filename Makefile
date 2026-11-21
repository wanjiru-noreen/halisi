.PHONY: build check-config readiness-check launch-check migrate-up migrate-down migrate-version migrate-ping sqlc-install sqlc-generate

build:
	mkdir -p bin
	go build -o bin/server ./cmd/server
	go build -o bin/migrate ./cmd/migrate
	go build -o bin/launchcheck ./cmd/launchcheck

# Validate all production environment variables, secrets, and domain configurations.
check-config:
	go run ./cmd/server --check-config

# Execute production readiness test suite.
readiness-check:
	go test -v ./tests/integration -run TestReadiness

# Run automated launch validation probes against a target instance (default: localhost:8080).
launch-check:
	go run ./cmd/launchcheck -url $(or $(TARGET_URL),http://localhost:8080)

sqlc-install:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1

sqlc-generate:
	sqlc generate

migrate-up:
	go run ./cmd/migrate up

# Revert one version, including its data.
migrate-down:
	go run ./cmd/migrate down

migrate-version:
	go run ./cmd/migrate version

# Verify database connectivity without running migrations.
migrate-ping:
	go run ./cmd/migrate ping
