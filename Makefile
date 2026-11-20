.PHONY: build check-config migrate-up migrate-down migrate-version migrate-ping sqlc-install sqlc-generate

build:
	go build -o bin/server ./cmd/server
	go build -o bin/migrate ./cmd/migrate

# Validate all production environment variables, secrets, and domain configurations.
check-config:
	go run ./cmd/server --check-config

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
