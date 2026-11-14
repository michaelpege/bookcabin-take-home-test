# Local Setup Guide

## Prerequisites

- **Go** 1.24+ (developed on Go 1.26)
- **Docker & Docker Compose** (for containerized Redis & API)
- **golangci-lint** (optional locally; enforced in CI)

## Environment Variables

Copy `.env.example` to `.env` if overriding defaults:

| Variable | Default | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | HTTP server listen port |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `READ_TIMEOUT` | `5s` | HTTP server read timeout |
| `WRITE_TIMEOUT` | `10s` | HTTP server write timeout |
| `SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown deadline |
| `REDIS_ADDR` | `localhost:6379` | Redis address (fail-open if unreachable) |

## Running Locally

```bash
# Run API directly
go run ./cmd/api

# Run all unit tests with race detector
make test
```

## Running with Docker Compose

```bash
docker compose up --build -d
docker compose logs -f api
```
