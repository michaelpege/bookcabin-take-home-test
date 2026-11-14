# System Architecture

## Clean Architecture Layers

1. **`internal/domain`**: Pure entities (`Flight`, `SearchCriteria`), `Flight.Validate()`, and port interfaces (`FlightProvider`, `FlightCache`). Zero internal dependencies.
2. **`internal/usecase`**: Concurrent fan-out aggregator, $O(N)$ single-pass filter, $O(N \log N)$ sorter, cross-provider price comparator, and min-max Best-Value ranker.
3. **`internal/provider`**: Airline-specific adapters (`garuda`, `lion`, `batik`, `airasia`) wrapped by `ResilientProvider` (Go struct-wrapping decorator for per-SLA timeout, token-bucket rate limit, exponential backoff + jitter, and mock auth).
4. **`internal/repository`**: Redis write-through cache implementing `domain.FlightCache`, plus `NoOpCache` for unit testing.
5. **`internal/handler/http`**: HTTP transport, request validation, DTO mapping, and `slog` middleware.

## Key Design Decisions

- **2-Level Sanitization**: Provider adapters validate & normalize airline-specific quirks (e.g., Garuda `GA315` segments, Batik `+0700` & `"Y"`, Lion naive IANA TZ, AirAsia float hours); `domain.Flight.Validate()` enforces universal invariants.
- **Two-Stage Filter/Sort**: Adapters translate only natively supported wire params; `usecase` runs the authoritative global filter and sort.
- **Partial Failure Resilience**: Provider goroutines use isolated per-provider contexts so one unstable provider never aborts healthy ones.
