.PHONY: build migrate-up migrate-down migrate-version sqlc-install sqlc-generate

build:
	go build -o bin/server ./cmd/server
	go build -o bin/migrate ./cmd/migrate

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
