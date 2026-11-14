# Infrastructure & Containerization

## Container Topology

`docker-compose.yml` provisions two services:
1. **`api`**: Multi-stage Go build (`golang:alpine` builder -> `alpine` non-root runtime).
   - Embeds Go timezone database via `import _ "time/tzdata"` so `Asia/Jakarta` (WIB), `Asia/Makassar` (WITA), and `Asia/Jayapura` (WIT) resolve without OS tzdata packages.
   - Exposes `8080` with `/health` healthcheck.
2. **`redis`**: `redis:8-alpine` with `redis-cli ping` healthcheck.

## Cache Taxonomy (Implemented in C9)

- **L1 Base Aggregation Cache** (`flight_search:{origin}:{dest}:{date}:{pax}:{cabin}`): Stores normalized, validated flights before filtering/sorting so filter/sort tweaks execute in `<5ms`.
- **Per-Provider Cache** (`provider:{name}:{origin}:{dest}:{date}:{pax}:{cabin}`): Isolates airline TTLs and prevents re-querying slow providers when another provider had a transient failure.
- **Write-Through + `singleflight` + Fail-Open**: Cache misses deduplicate via `singleflight.Group`, write through to Redis (skipping long TTLs on partial failures), and fail open to live queries if Redis is down.
