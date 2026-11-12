.PHONY: migrate-up migrate-down migrate-version

migrate-up:
	go run ./cmd/migrate up

# Revert one version, including its data.
migrate-down:
	go run ./cmd/migrate down

migrate-version:
	go run ./cmd/migrate version
