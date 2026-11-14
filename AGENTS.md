# AGENTS.md — AI Agent & Contributor Guide

This file is the **primary entry point** for AI coding agents and engineers working on `bookcabin-take-home-test`.

## 1. Repository Map & Dependency Rules

```
cmd/api/main.go               # Composition root (DI wiring, tzdata, graceful shutdown)
internal/domain/              # INNERMOST CORE: entities, Validate(), FlightProvider & FlightCache ports
                              # RULE: MUST NOT import any other internal/ package
internal/usecase/             # Application logic: aggregator, filter, sorter, comparator, ranker
                              # RULE: Imports ONLY internal/domain (+ internal/platform)
internal/provider/            # Airline adapters (garuda, lion, batik, airasia) + ResilientProvider decorator
                              # RULE: Implements domain.FlightProvider; never imported by usecase
internal/repository/          # Data & cache adapters (redis, noop/memory)
                              # RULE: Implements domain.FlightCache; never imported by usecase
internal/handler/http/        # HTTP transport, DTO mapping, request logging & recovery middleware
internal/platform/            # Shared utilities (config, logger, timeutil, currency)
docs/                         # setup.md, infrastructure.md, architecture.md, PROGRESS.md, decisions/
```

## 2. Hard Invariants ("Never Break These")

1. **Never commit `BookCabin Take Home Test - Software Engineer (Backend)/`** — it is gitignored and holds the original prompt PDF.
2. **Dependency Inversion**: `internal/usecase` only depends on `internal/domain` interfaces (`FlightProvider`, `FlightCache`). Never import `internal/provider/*` or `internal/repository/*` from `usecase`.
3. **Two-Level Sanitization**:
   - Level 1 (Provider Adapter): Validate raw airline payload against its own contract and normalize to `domain.Flight`.
   - Level 2 (Domain): `flight.Validate()` enforces universal invariants (`Arrival > Departure`, `Duration > 0`, `Price > 0`).
4. **Two-Stage Filter/Sort**: Provider adapters translate only natively supported wire params; `internal/usecase` ALWAYS runs the authoritative post-aggregation `Filter` and `Sort`.
5. **Context & Partial Failure**: Provider fan-out uses isolated per-provider contexts. One failed provider (e.g., AirAsia 10% failure) MUST NOT cancel healthy providers.
6. **Non-nil JSON slices**: Initialize empty slices (`[]string{}`) so JSON marshals as `[]`, never `null`.
7. **Timezone Embedding**: Keep `import _ "time/tzdata"` in `cmd/api/main.go` and `timeutil` so `Asia/Jakarta`, `Asia/Makassar`, and `Asia/Jayapura` work in minimal containers.

## 3. Mandatory Verification Before Every Commit

Run:
```bash
make check
```
Which executes:
- `gofmt -l .` (formatting)
- `go vet ./...` (static checks)
- `golangci-lint run` (if installed)
- `go test -race ./...` (unit tests with race detector)

For Redis integration tests (requires Docker / Redis):
```bash
make test-integration
```

## 4. How to Add a New Airline Provider (4 Steps)

1. Create `internal/provider/<airline>/` with `client.go`, `normalizer.go`, `client_test.go`, and `testdata/`.
2. Implement `domain.FlightProvider` (`Name()`, `Search(ctx, domain.SearchCriteria)`), respecting `ctx.Done()` during any simulated delay.
3. Run the shared `providertest` contract suite in `client_test.go`.
4. Register the provider + its `ProviderSLA` config in `cmd/api/main.go` wrapped with `ResilientProvider`.

## 5. Additional Docs (Load On-Demand)

- [README.md](README.md) — Overview, API contract, quickstart
- [docs/setup.md](docs/setup.md) — Local Go/Docker/Redis setup & env vars
- [docs/infrastructure.md](docs/infrastructure.md) — Container topology, cache keys, TTLs
- [docs/architecture.md](docs/architecture.md) — Layer diagrams, resilience, best-value scoring
- [docs/PROGRESS.md](docs/PROGRESS.md) — Milestone commit tracker for resuming work
