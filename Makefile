.PHONY: run build test test-integration lint fmt check up down

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test -race -v ./...

test-integration:
	go test -tags=integration -race -v ./...

fmt:
	gofmt -w .

lint:
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run; else echo "golangci-lint not installed locally (enforced in CI)"; fi

check:
	@unformatted=$$(gofmt -l .); if [ -n "$$unformatted" ]; then echo "Unformatted: $$unformatted"; exit 1; fi
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run; fi
	go test -race ./...

up:
	docker compose up --build -d

down:
	docker compose down
