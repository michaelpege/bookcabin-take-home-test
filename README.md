# BookCabin Flight Search & Aggregation System

A Clean Architecture Go backend that concurrently aggregates, normalizes, filters, compares, ranks, and caches flight search results across multiple Indonesian airline providers (Garuda Indonesia, Lion Air, Batik Air, AirAsia).

## Quickstart

### Option 1: Docker Compose (Recommended)
```bash
docker compose up --build
```
Check server health:
```bash
curl http://localhost:8080/health
curl http://localhost:8080/ping
```

### Option 2: Local Go
```bash
go run ./cmd/api
```

## Project Structure

```text
├── cmd/api/                  # Application entrypoint & DI wiring
├── internal/
│   ├── domain/               # Core entities, validation, port interfaces (no internal deps)
│   ├── platform/             # Config, slog logger, timeutil (WIB/WITA/WIT), currency
│   ├── provider/             # Airline adapters + ResilientProvider decorator (planned C3-C7)
│   ├── repository/           # Redis write-through cache + NoOp fallback (planned C9)
│   ├── usecase/              # Aggregator, filter, sorter, comparator, ranker (planned C8)
│   └── handler/http/         # HTTP router, middleware, DTO mapping
├── docs/                     # Setup, infrastructure, architecture, progress tracker
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

## Endpoints

| Method | Path | Description | Status |
| :--- | :--- | :--- | :--- |
| `GET` | `/ping` | Lightweight liveness probe (`{"status":"pong"}`) | Ready (C1) |
| `GET` | `/health` | Service health check with timestamp | Ready (C1) |
| `POST` | `/api/v1/flights/search` | Aggregated flight search, filter, sort & rank | Planned (C9) |

## Development & Verification

```bash
make test              # Run unit tests with -race
make lint              # Run go vet + golangci-lint
make check             # Run fmt, lint, and unit tests
make test-integration  # Run Redis integration tests (-tags=integration)
```

## Documentation

- [AGENTS.md](AGENTS.md) — Rules & workflow for AI coding agents
- [docs/setup.md](docs/setup.md) — Local setup & environment variables
- [docs/infrastructure.md](docs/infrastructure.md) — Containerization & Redis cache topology
- [docs/architecture.md](docs/architecture.md) — Clean Architecture & design choices
- [docs/PROGRESS.md](docs/PROGRESS.md) — Milestone progress tracker
