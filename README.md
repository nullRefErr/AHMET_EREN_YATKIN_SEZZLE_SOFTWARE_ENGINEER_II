# Calculator

A full-stack calculator: a Go REST service and a React + TypeScript front end.

```bash
docker compose up --build
```

| | |
|---|---|
| App | <http://localhost:3000> |
| API | <http://localhost:8080/api/v1> |
| Health | <http://localhost:8080/healthz> |

That is the whole setup. Nothing else needs installing, and there is no database to provision.

---

## Contents

- [Running it](#running-it)
- [API](#api)
- [Architecture](#architecture)
- [Design decisions](#design-decisions)
- [Assumptions](#assumptions)
- [Testing and coverage](#testing-and-coverage)
- [Trade-offs and known limitations](#trade-offs-and-known-limitations)
- [What I would do next](#what-i-would-do-next)
- [Repository layout](#repository-layout)

---

## Running it

### With Docker (recommended)

`docker compose up --build`, as above. nginx serves the built app on port 3000 and proxies `/api` to the backend, so the browser only ever
talks to one origin and CORS never comes up.

### Without Docker

Requires Go 1.27 and Node 22 with pnpm.

**Backend** — listens on `:8080`:

```bash
cd backend && go run ./cmd/server
```

**Frontend** — Vite dev server on `:5173`, proxying `/api` to `localhost:8080`:

```bash
cd frontend && pnpm install && pnpm dev
```

### Configuration

Every setting has a working default, so nothing has to be set to run the project. Anything
invalid stops the process at startup rather than being silently ignored.

Under Docker, copy `.env.example` to `.env` and Compose reads it. `PORT` is the exception: it is
fixed at 8080 because the published port and the nginx proxy both name it, so changing it there
would only break the proxy.

Without Docker there is no `.env` loader — the service reads the environment directly, so pass a
setting on the command line (`PORT=9999 go run ./cmd/server`) or export it.

| Variable | Default | Meaning |
|---|---|---|
| `PORT` | `8080` | Port the API listens on |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn` or `error` |
| `CACHE_SIZE` | `1024` | Calculations kept in memory |
| `SHUTDOWN_TIMEOUT` | `10s` | Grace period for in-flight requests on SIGTERM |
| `ALLOWED_ORIGINS` | *(empty)* | Comma-separated CORS origins. Not needed behind nginx |

### Common commands

```bash
make test    # both suites with the race detector, and rewrite COVERAGE.md
make lint    # go vet, golangci-lint, eslint, tsc
make up      # docker compose up --build
make down    # stop and remove volumes
```

Anything touching the front end installs its dependencies first, so these work on a fresh clone
with nothing set up.

---

## API

### `POST /api/v1/calculations`

```bash
curl -X POST http://localhost:8080/api/v1/calculations \
  -H 'Content-Type: application/json' \
  -d '{"operation":"divide","operands":[10,4]}'
```

```json
{ "operation": "divide", "operands": [10, 4], "result": 2.5, "cached": false }
```

Send the same request again and `cached` is `true` — the result came from memory rather than being
recomputed.

Operations and how many operands each takes:

| Operation | Operands | `[a, b]` gives |
|---|---|---|
| `add` | 2 | `a + b` |
| `subtract` | 2 | `a - b` |
| `multiply` | 2 | `a × b` |
| `divide` | 2 | `a ÷ b` |
| `power` | 2 | `a` to the power of `b` |
| `sqrt` | **1** | `√a` |
| `percentage` | 2 | `a` percent of `b`, i.e. `a × b ÷ 100` |

### `GET /api/v1/operations`

```bash
curl http://localhost:8080/api/v1/operations
```

```json
{
  "operations": [
    { "name": "add", "operands": 2 },
    { "name": "subtract", "operands": 2 },
    { "name": "multiply", "operands": 2 },
    { "name": "divide", "operands": 2 },
    { "name": "power", "operands": 2 },
    { "name": "sqrt", "operands": 1 },
    { "name": "percentage", "operands": 2 }
  ]
}
```

The keypad is built from this response. The front end never keeps its own copy of the operation
list, so the two cannot drift apart.

### `GET /healthz`

Returns `200` while the process is alive. Docker uses it as the container health check.

### Errors

Every failure has the same shape:

```bash
curl -X POST http://localhost:8080/api/v1/calculations \
  -H 'Content-Type: application/json' \
  -d '{"operation":"divide","operands":[10,0]}'
```

```json
{
  "error": { "code": "DIVISION_BY_ZERO", "message": "Division by zero is not defined." },
  "request_id": "b79100b284b71d688ae94b1a"
}
```

| Situation | Status | `code` |
|---|---|---|
| Malformed JSON, unknown operation, non-finite operand | 400 | `INVALID_REQUEST` |
| Wrong number of operands for the operation | 400 | `INVALID_OPERAND_COUNT` |
| Division by zero | 422 | `DIVISION_BY_ZERO` |
| Square root of a negative number | 422 | `NEGATIVE_SQRT` |
| Result outside the range of `float64` | 422 | `NUMERIC_OVERFLOW` |
| Result is not a number, e.g. `(-8)^0.5` | 422 | `UNDEFINED_RESULT` |
| Anything unexpected | 500 | `INTERNAL_ERROR` |

`request_id` is echoed in the `X-Request-Id` header and appears in the matching log line, so a
report from a user leads straight to the request that produced it. Internal detail — stack traces,
error strings from libraries — never reaches the response body.

---

## Architecture

```
┌──────────────────┐                ┌────────────────────────────────────┐
│  React SPA       │   /api/v1/*    │  Go backend  :8080                 │
│  nginx  :3000    │ ─────────────► │  transport/rest   HTTP, DTOs       │
└──────────────────┘                │        │                           │
                                    │        ▼                           │
                                    │  service          orchestration    │
                                    │      │        │                    │
                                    │      ▼        ▼                    │
                                    │  domain     repository             │
                                    │  arithmetic  in-process LRU        │
                                    └────────────────────────────────────┘
```

Dependencies point inward only. `domain` imports nothing but the standard library — no web
framework, no DTO, no storage — which is what makes the arithmetic testable without starting a
server. A `gin.Context` reaching `service` or `domain` would be a defect even if it compiled.

The layers sit under `internal/`, so the Go compiler enforces the boundary rather than leaving it
to review.

---

## Design decisions

### One service, not three

The brief asks for "a backend microservice", singular, and says to prefer correctness and clarity
over extra features. A gateway in front of a single upstream adds a network hop and nothing else,
and a database with no data to hold is decoration. So this is one service with clear internal
seams, not three processes.

The seams are real: swapping the in-memory repository for a database is a constructor change in
`main.go`. Splitting `service` into its own process later would follow the boundary that already
exists. [What I would do next](#what-i-would-do-next) covers where those splits would go and what
would justify them.

### One resource, not seven endpoints

`POST /calculations` with an `operation` field, rather than `/add`, `/subtract` and so on. REST
resources are nouns. Seven verb endpoints would mean seven handlers, seven validation paths and
seven test suites, and adding an operation would change the shape of the API. One resource means
one validation path and one test table, and a new operation is a row in a slice.

### `percentage` had to be defined

"Percentage" means at least three different things: *a% of b*, *a is what percent of b*, and
*percent change from a to b*. `percentage(50, 200)` is `100`, `25` or `300` depending on which one
is meant.

This implementation uses **`a` percent of `b`**, so `percentage(50, 200) = 100`. It is the reading
that matches the `%` key on a physical calculator. The ambiguity is worth naming rather than
resolving silently, because a caller who assumed a different one would get plausible wrong answers
with no error.

### `float64`, and what that costs

Arithmetic uses IEEE-754 double precision. `0.1 + 0.2` is `0.30000000000000004`. That is the format
behaving as specified, not a defect, and the API does not hide it.

Money would need a decimal type. `domain.Number` is a type alias over `float64` precisely so that
such a change starts in one place instead of being spread across every signature.

### 400 versus 422

**400 means the request could not be understood** — malformed JSON, an unknown operation, an
operand that is not a finite number. **422 means it was understood and arithmetic cannot answer
it** — dividing by zero, the square root of a negative number, a result that overflows.

The distinction is genuinely arguable. Being consistent about it matters more than which side of
the argument the project lands on.

### Validation lives in one place

Go's common validation libraries treat a zero value as "not supplied". A `required` rule on the
operands would reject `{"operation":"add","operands":[0,0]}` — a perfectly valid request, and the
first thing anyone tries. This code uses no `required` rule at all. Operands decode into
`[]*float64` so that a JSON `null` can be told apart from a zero, and the operand count is checked
by the domain, which already has to know how many each operation takes. One rule, one place.

A related trap: JSON has no `Infinity` literal, but `1e400` is valid JSON and decodes to `+Inf`.
Operands are therefore checked for being finite *after* decoding, not just for being well-formed.

### Errors carry codes, not sentences

The front end branches on `code` and never on `message`. Message text is free to be reworded or
translated; a code is a contract. Parsing English out of a response is brittle and makes
translation impossible.

### The client validates for comfort, the server for correctness

The front end disables what cannot be submitted and shows failures immediately, but the server
validates everything again and is the only authority. Client-side validation is a user-experience
feature; it is not a security boundary.

### The repository owns an in-process cache

`service` declares the `Repository` interface it needs; `repository/memory` implements it. The
interface is written by the consumer, in the domain's own language — `Find(ctx, domain.Request)`,
not a cache key or a map — so nothing above it knows where results are kept.

The store is an LRU with a fixed capacity. The bound is not an optimisation: an unbounded map would
let a client that sends endlessly varying operands grow the process until it is killed, which makes
capacity a limit at a trust boundary. Failed calculations are never stored, and a store that cannot
be read or written is logged and stepped over — a broken cache must never fail a valid request.

Commutative operations share an entry: `2 + 3` and `3 + 2` are the same key, which roughly doubles
the hit rate. `subtract`, `divide`, `power` and `percentage` deliberately do not, because for them
the operand order is part of the question.

**The honest part:** adding two numbers takes nanoseconds and a map lookup takes microseconds, so
at this scale the cache is *slower* than the arithmetic it caches. It is here to give the
repository pattern something real to own rather than an empty interface, and it would begin to earn
its place with genuinely expensive operations. That trade-off is stated rather than hidden, and the
`cached` field in every response makes the behaviour observable instead of theoretical.

### The container probes itself

The backend image is distroless and runs as a non-root user: no shell, no package manager, and
about 35 MB, nearly all of it the static Go binary. That also means no `curl` for a Docker health check, so the binary answers the question
itself — `server -healthcheck` performs the request and exits with a status code. `docker compose`
waits on it before starting the front end.

### Shutdown is graceful

Docker sends `SIGTERM` and then waits. The server stops accepting connections, lets in-flight
requests finish within `SHUTDOWN_TIMEOUT`, and then exits. Without that, every deploy cuts off
whatever was in progress.

### Front-end state is a reducer

Calculator behaviour lives in one pure reducer with explicit actions. It has no React and no
network in it, so every state transition is tested by calling a function. Redux would be a large
answer to a small question here.

One decision sits outside it: pressing an operator has to finish whatever calculation is already on
screen, and finishing one means a request. A pure reducer cannot make requests, so the hook decides
when to send and the reducer only records the answer.

Three libraries the plan called for were left out after the scope narrowed: a server-state library
for a single POST, a schema validator for a four-field response, and a mock-service worker for
tests that stub `fetch` in three lines. Responses are still validated at runtime by a small type
guard, because TypeScript types are erased and a proxy returning an HTML error page must not become
a `NaN` on screen.

---

## Assumptions

- **No authentication.** Nothing in the brief calls for it. It would belong in a gateway, not here.
- **No persistence.** Calculation history is not requested, so nothing is stored beyond the cache.
- **Binary operations, chained left to right.** Pressing an operator finishes the calculation on
  screen and carries the answer forward, so `9 + 1 * 2` is 20. There is no operator precedence.
- **Interface text is English**, on the assumption the reviewer reads English.
- **Time budget.** The brief suggests 2–4 hours. This landed at roughly 7, with the extra time in
  tests and documentation, both of which the brief lists as deliverables.

---

## Testing and coverage

```bash
make test           # both suites with the race detector, and rewrite COVERAGE.md
make test-backend   # Go only, no coverage — the fast loop while writing code
make test-frontend  # front end only, no coverage
```

**[COVERAGE.md](COVERAGE.md) is the report**, per package and per file for both layers. It is
generated rather than written by hand, and it is rewritten by the same command that runs the
tests: a report you have to remember to refresh separately is a report that goes stale, which is
exactly what happened twice here before it was wired this way. A failing run leaves the file
alone rather than publishing coverage for code that does not work.

The same run writes `backend/coverage.html` to browse line by line, and CI uploads both profiles
as build artifacts on every push.

The numbers matter less than what is behind them, so here is what each layer is tested for.

| Layer | What the tests are for |
|---|---|
| `domain` | Every row of the edge-case table: zero operands, negatives, subnormals, overflow, `NaN`, wrong operand counts. This is where correctness lives, so this is where the tests concentrate |
| `service` | Recall versus recompute, failures never stored, and that a broken repository still returns a correct answer |
| `repository/memory` | Key determinism, commutativity, LRU eviction, recency refresh, and concurrent access under `-race` |
| `transport/rest` | Status codes and error codes for every failure, the zero-operand case, the `cached` flag over two requests, and that CORS does not reflect an origin it was not given |
| `config` | Defaults, parsing, and that bad values stop the process |
| `cmd/server` | Wiring and signal handling, so only the health probe is unit-tested. Exercising the rest means running a process, which is why this is the one thinly covered package and why that is deliberate |
| Front-end reducer | Every state transition, including recovering from an error and continuing from a result |
| `useCalculator` | The parts that need a request and so cannot live in the reducer: chaining `75 + 52 - 30` into two calculations, a failure stopping the chain, and retry after a network error |
| Front-end components | The real reducer and client driven against a stubbed `fetch`, queried by accessible role, including the keypad built from the API and keys disabled while a request is in flight |

Two details are deliberate throughout: errors are compared with `errors.Is` against sentinels
rather than by string, and floats are compared with a delta rather than `==`. Front-end tests query
by accessible role rather than test ids, which tests the accessibility tree and the behaviour at
the same time.

Every layer was developed test-first: each began with a failing test, run and seen to fail for the
right reason, and no production code was written before one existed that required it.

There is one exception and it is worth naming. The responsive layout fix — the calculator was
using 234px of a 375px screen — has no test. The bug is in layout, and jsdom does not lay anything
out, so the front-end suite cannot see it before or after the fix. It was found by measuring in a
real browser and verified the same way. Covering it properly would mean a browser-driving test
framework, which is a dependency worth deciding on deliberately rather than adding inside a bug
fix.

---

## Trade-offs and known limitations

- **The cache costs more than it saves at this scale.** Covered above. It is a demonstration of the
  repository boundary with a real owner, and it is measurable rather than assumed.
- **The cache is per-process.** Two instances behind a load balancer would each keep their own, and
  hit rates would fall. A shared cache is the fix, and it is a change to one constructor.
- **A commutative hit echoes the stored operand order.** Asking for `3 + 2` after `2 + 3` was
  computed returns `operands: [2, 3]`. The result is identical; the echo is not.
- **`float64`, not decimal.** Fine for a calculator, wrong for money.
- **No operator precedence.** Operators chain left to right like a pocket calculator, so
  `9 + 1 * 2` is 20 rather than 11. There is no expression to type and no precedence to apply.
- **No rate limiting.** It belongs at an edge that does not exist yet.
- **TLS terminates at the edge.** The service speaks plain HTTP inside the compose network, on the
  assumption of a proxy in front of it in a real deployment.

---

## What I would do next

In the order I would actually do them:

1. **An OpenAPI document** as the contract, with front-end types generated from it. Today the
   shapes are agreed by hand in two places.
2. **Calculation history** — a `POST` writes an event, a separate reader serves
   `GET /history` with cursor pagination, not `OFFSET`. This is the first thing that genuinely
   wants PostgreSQL, and it is where a second service starts to make sense: the audit path can then
   fail without taking the calculation path down with it.
3. **A shared cache** once more than one instance runs, with `singleflight` in front of it so that
   a thousand simultaneous identical requests become one computation rather than a thundering herd.
4. **Metrics** — `cache_hit_ratio` and a latency histogram per operation, which would turn the
   honest claim above about caching into a measured one.
5. **A gateway** if a second service ever exists, and only then: authentication, rate limiting and
   request aggregation belong at that edge, not scattered through services.
6. **A decimal numeric type** behind the existing `domain.Number` alias, if this were ever to touch
   money.

---

## Repository layout

```
.
├── backend/
│   ├── cmd/server/          # wiring, graceful shutdown, health probe
│   └── internal/
│       ├── config/          # environment to typed config, validated at startup
│       ├── domain/          # arithmetic and its errors — standard library only
│       ├── service/         # orchestration; declares the Repository interface
│       ├── repository/      # in-process LRU implementation
│       └── transport/rest/  # Gin router, DTOs, error mapping
├── frontend/
│   └── src/
│       ├── api/             # client, runtime response checks
│       ├── calculator/      # the reducer: all calculator behaviour, pure
│       ├── components/      # Display, Keypad, ErrorBanner
│       └── hooks/           # connects the reducer to the API
├── docker-compose.yml
├── SPEC.md                  # the requirements this was built against
├── CLAUDE.md                # engineering rules the code is held to
├── COVERAGE.md              # rewritten by `make test`, both layers
├── scripts/coverage.sh      # what generates it
└── PROMPTS.md               # every AI prompt used, verbatim
```
