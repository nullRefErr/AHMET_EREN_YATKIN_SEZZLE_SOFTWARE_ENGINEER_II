# Build Record — Calculator

An internal note. Requirements are in `SPEC.md`, engineering rules in `CLAUDE.md`, and the delivered
documentation in `README.md`.

## Phases

The order followed the layer direction in `SPEC.md` §2 — inward out, each layer green before the
next one started. Every phase began with a failing test, per `CLAUDE.md` §2.

| Phase | Scope | Status |
|---|---|---|
| F0 | Decisions, scaffolding, linter configuration | ✅ |
| F1 | Domain engine | ✅ |
| F1.5 | Service, repository and in-process cache | ✅ |
| F2 | HTTP transport layer | ✅ |
| F3 | Config, wiring, graceful shutdown | ✅ |
| F4 | Front-end reducer | ✅ |
| F5 | Front-end interface | ✅ |
| F6 | Docker and compose | ✅ |
| F7 | README and coverage | ✅ |

What each test proves is in its own name, and the coverage figures are in `README.md`. Neither is
repeated here.

## Gates

`go vet` · `golangci-lint run` · `go test -race` · `eslint` · `tsc --noEmit` — all clean.
`docker compose up --build` verified from a clean clone.

## Outstanding

- Push to a remote — **the remaining half of deliverable 1**. `git init` and the first commit are
  done at `5a1125d`; no remote is configured yet.
