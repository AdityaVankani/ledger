.PHONY: run test sqlc migrate-up migrate-down

run:
	go run ./cmd/api

test:
	go test ./...

# Install the sqlc CLI separately, then regenerate typed query code after changing db/queries.
sqlc:
	sqlc generate

# Requires golang-migrate: https://github.com/golang-migrate/migrate
migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

