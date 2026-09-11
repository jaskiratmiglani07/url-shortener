.PHONY: all build run test clean docker-up docker-down

all: build

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

test:
	go test -v -race ./...

clean:
	rm -rf bin/ coverage.html coverage.out

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v
