BIN := bin/pokedex

.PHONY: build run test lint tidy docker-up docker-down

build:
	go build -o $(BIN) ./cmd/pokedex

run:
	go run ./cmd/pokedex

test:
	go test -race ./...

lint:
	go vet ./...
	golangci-lint run ./...

tidy:
	go mod tidy

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down
