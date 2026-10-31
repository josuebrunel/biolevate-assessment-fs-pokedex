BIN := bin/pokedex

DB_PORT ?= 5432
DB_URL := postgres://pokedex:pokedex@localhost:$(DB_PORT)/pokedex?sslmode=disable
export APP_DATABASE_URL ?= $(DB_URL)
export DB_PORT

.PHONY: build run test test-integration lint generate eval tidy migrate-up migrate-down import docker-up docker-down

build:
	go build -o $(BIN) ./cmd/pokedex

run:
	go run ./cmd/pokedex

test:
	go test -race ./...

# Runs the suite including tests that need Postgres, starting the compose db first.
test-integration:
	docker compose up -d --wait db
	APP_TEST_DATABASE_URL=$(DB_URL) go test -race -count=1 ./...

generate:
	templ generate ./internal/web/views

# Prints the golden-query report (needs the compose db, like test-integration).
eval:
	docker compose up -d --wait db
	APP_TEST_DATABASE_URL=$(DB_URL) go test -count=1 -run 'TestGolden$$' -v ./internal/golden | grep -v ' INFO '

lint:
	go vet ./...
	golangci-lint run ./...

tidy:
	go mod tidy

migrate-up:
	go run ./cmd/pokedex -migrate up

migrate-down:
	go run ./cmd/pokedex -migrate down

import:
	go run ./cmd/pokedex -import

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down
