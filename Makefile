.PHONY: run build tidy test docker-up docker-down

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

tidy:
	go mod tidy

test:
	go test ./...

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down
