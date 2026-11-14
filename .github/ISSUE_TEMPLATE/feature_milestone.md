---
name: Feature Milestone
about: Scoped Clean Architecture milestone or feature task
title: "[C#] <Short Milestone Title>"
labels: enhancement
---

## Context
<!-- Why this milestone is needed and how it fits the roadmap -->

## Layer(s) Affected
- [ ] `internal/domain`
- [ ] `internal/platform`
- [ ] `internal/provider`
- [ ] `internal/repository`
- [ ] `internal/usecase`
- [ ] `internal/handler`

## Contract / Input-Output
<!-- Structs, interfaces, or JSON schema changes -->

## Acceptance Criteria
- [ ] Implementation adheres to `AGENTS.md` dependency rules
- [ ] Unit tests added (and integration tests if touching Redis/HTTP)
- [ ] `make check` passes cleanly

## Verification Command
```bash
make check
```
