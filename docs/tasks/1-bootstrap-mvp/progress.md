# Milestone Progress Tracker

Track commit-by-commit progress so any fresh agent session can resume immediately.

| Commit | Milestone | Status | Verification |
| :--- | :--- | :--- | :--- |
| **C1** | Bootstrap Go Clean Architecture server (`/health`, `/ping`, `slog`, Docker, CI, Docs) | **Done** (PR #2) | `make check` |
| **C2** | `internal/domain` + `timeutil` (WIB/WITA/WIT) + `currency` (IDR) + unit tests | **Done** (PR #4) | `go test -race ./internal/domain/... ./internal/platform/...` |
| **C3** | `ResilientProvider` decorator (per-SLA timeout, rate limit, retry) + mock auth | **Done** (PR #6) | `go test -race ./internal/provider/...` |
| **C4** | Garuda Indonesia provider adapter + fixture tests | **Done** (PR #8) | `go test -race ./internal/provider/garuda/...` |
| **C4.5** | Vertical E2E Slice: `usecase.Aggregator`, `POST /api/v1/flights/search` (Garuda), `trace_id` logs | **PR Open** (Issue #9) | `go test -race ./...` + 100-curl Docker check |
| **C5** | Lion Air provider adapter + fixture tests | Pending | `go test -race ./internal/provider/lion/...` |
| **C6** | Batik Air provider adapter + fixture tests | Pending | `go test -race ./internal/provider/batik/...` |
| **C7** | AirAsia provider adapter (90% SLA simulation) + fixture tests | Pending | `go test -race ./internal/provider/airasia/...` |
| **C8** | `internal/usecase` (aggregator, filter, sorter, comparator, ranker) + `-race` tests | Pending | `go test -race ./internal/usecase/...` |
| **C9** | `internal/repository/redis` (2-tier write-through cache) + `POST /api/v1/flights/search` | Pending | `make check && make test-integration` |
| **C10** | Final `README.md` (curl examples, complexity analysis, design choices) + benchmarks | Pending | `go test -bench=. ./...` |
