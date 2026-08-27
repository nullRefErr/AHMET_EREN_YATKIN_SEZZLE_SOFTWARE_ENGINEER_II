# Calculator — Requirements Specification

**Project:** Sezzle take-home assignment — full-stack calculator
**Date:** 2026-08-27
**Status:** Agreed — single-service scope, all questions closed

## Source documents

| Document | Role | Authority |
|---|---|---|
| The Sezzle brief (e-mail, 2026-08-27) | The **only** source of requirements | **Binding** |
| `CLAUDE.md` | Engineering rules the code is reviewed against | Binding |
| `PROMPTS.md` | Record of the prompts used (brief, instruction 5) | Living record |

Where two disagree, **the brief wins**.

The "plan document" referred to in §1 is a technical note I wrote before starting. It is not in this
repository, and it is mentioned only to record which of its proposals were taken out of scope.

---

## 1. Scope decision — one backend service

Two sentences in the brief settle the scope:

> "Build a full-stack calculator application with a React frontend and **a backend microservice**." (singular)

> "Spend ~2–4 hours on this assignment. **Prioritize correctness, clarity, and maintainability over extra features.**"

**Decision: one Go backend service and one React front end.**

What was taken out of scope, and why:

| Proposal from the plan document | Status | Reason |
|---|---|---|
| API gateway service | **Dropped** | The brief asks for one service. A gateway with a single upstream adds a network hop and nothing else. |
| History service + PostgreSQL | **Dropped** | The brief never mentions persistence. A database with no real work to do is decoration. |
| L2 Redis cache | **Dropped** | Redis is an external dependency; it weighs down both the single-service scope and the one-command start-up. |
| L1 in-process cache | **Kept** (D-13) | Real state for the repository pattern to own. See §2.2. |
| Asynchronous events over Redis Streams | **Dropped** | There are no events to write. |
| Prometheus metrics | **Dropped** | Nothing here is worth measuring yet. |
| Rate limiting | **Dropped** | No authentication and no multi-tenancy. |
| Five separate ADR files | **Moved into the README** | The brief's "design rationale" is met in the README; separate files are overhead. |
| pnpm workspace + Turborepo | **Dropped** | There is one front-end package. A workspace for one package is cost without benefit. |

None of this is discarded — the README's **"What I Would Do Next"** explains each one and the scale
at which it would start to pay. Being able to explain an architecture with its reasons signals the
same competence as building it unnecessarily, at a tenth of the cost.

---

## 2. Architecture

```
┌──────────────────┐   HTTP/JSON    ┌────────────────────────────────────┐
│  React SPA       │  /api/v1/*     │  Go backend  :8080                 │
│  (Vite, nginx)   │ ─────────────► │  transport/rest  (gin, DTOs, errs) │
│  :3000           │                │         │                          │
└──────────────────┘                │         ▼                          │
                                    │  service         (orchestration)   │
                                    │      │        │                    │
                                    │      ▼        ▼                    │
                                    │  domain    repository              │
                                    │  (pure)    (in-process LRU)        │
                                    └────────────────────────────────────┘
```

No database and no external dependency. The only state is the in-process cache; it dies with the
process, and that is an accepted limit, stated in the README.

### 2.1 Directory layout

```
.
├── README.md                  # setup, API examples, design decisions
├── SPEC.md                    # this file
├── CLAUDE.md                  # engineering rules
├── PROMPTS.md                 # brief, instruction 5
├── Makefile  docker-compose.yml  .env.example  .gitignore
├── .github/workflows/ci.yml
│
├── backend/
│   ├── go.mod  .golangci.yml  Dockerfile
│   ├── cmd/server/            # wiring, graceful shutdown, health probe
│   └── internal/
│       ├── config/            # environment to typed config, validated at startup
│       ├── domain/            # ⭐ pure business logic — NO framework imports
│       ├── service/           # orchestration; declares the Repository interface
│       ├── repository/memory/ # in-process LRU
│       └── transport/rest/    # Gin router, DTOs, error mapping
│
└── frontend/
    ├── package.json  vite.config.ts  tsconfig.json  eslint.config.js
    ├── Dockerfile  nginx.conf  index.html
    └── src/
        ├── api/               # client + runtime response checks
        ├── calculator/        # the reducer — all calculator behaviour, pure
        ├── components/        # Display, Keypad, ErrorBanner
        └── hooks/             # useCalculator
```

Tests sit next to the code they test (`calculator_test.go`, `reducer.test.ts`), not in a separate
tree.

`internal/` is deliberate: the Go compiler closes those packages to anything outside the module, so
the layer boundary is enforced by the language rather than by review.

### 2.2 The repository pattern and the in-process cache

`CLAUDE.md` RP-1 requires the repository pattern; the YAGNI rule forbids an interface with a single
empty implementation. The tension is resolved by giving the repository **real work**: holding
computed results in memory.

| Rule | How it is applied |
|---|---|
| The consumer declares the interface | `service` declares `Repository`; `repository/memory` returns a concrete type |
| The interface speaks the domain's language | `Find(ctx, domain.Request) (domain.Calculation, bool, error)` / `Save(ctx, domain.Calculation) error` — no cache key, map or mutex escapes |
| Dependencies point inward | `service` → `domain`; `repository/memory` → `domain`. `domain` knows neither |

**Cache key:** `{operation}:{operand0}:{operand1}`, with floats written by
`strconv.FormatFloat(f, 'g', -1, 64)` so the form is deterministic — otherwise `2` and `2.0` would
produce two different keys.

**Commutativity:** for `add` and `multiply` the operands are sorted before the key is built, so
`2+3` and `3+2` find the same entry. For `subtract`, `divide`, `power` and `percentage` they are
**not** — there the order is part of the question.

**The bound is required.** An unbounded map would let a client that sends endlessly varying operands
exhaust the process memory. That is a trust boundary and it cannot be simplified away. LRU over
`container/list` plus a map — standard library, no new dependency — with the capacity from config.

**Concurrency:** HTTP handlers run in parallel, so access is guarded by `sync.Mutex` and proven by
`go test -race`.

**Failures are not stored.** Only successful calculations are kept.

**The honest note (to be repeated in the README):** adding two numbers takes nanoseconds and a map
lookup takes microseconds, so at this scale the cache is **slower** than the arithmetic it caches.
The layer exists to give the repository pattern a real responsibility and to be ready for expensive
operations. Writing that cost down rather than hiding it is what makes the layer defensible.

---

## 3. Functional requirements

Priorities come straight from the brief: **Required** = the brief demands it, **Optional** = the
brief marks it "Optional", **Proposed** = not in the brief, but cheap and it raises quality.

### 3.1 Operations

| ID | Requirement | Priority |
|---|---|---|
| FR-01 | Addition (`add`), two operands | Required |
| FR-02 | Subtraction (`subtract`), two operands | Required |
| FR-03 | Multiplication (`multiply`), two operands | Required |
| FR-04 | Division (`divide`), two operands; division by zero is an error | Required |
| FR-05 | Exponentiation (`power`), two operands | Optional |
| FR-06 | Square root (`sqrt`), **one** operand | Optional |
| FR-07 | Percentage (`percentage`), two operands | Optional |

**FR-07 decision:** `percentage(a, b) = a * b / 100` — "a percent of b". `percentage(50, 200) = 100`.

"Percentage" means at least three things: (1) a% of b, (2) a is what percent of b, (3) percent
change. The first is the one that matches the `%` key on a physical calculator. **The ambiguity and
the reading chosen are both stated in the README** — noticing an ambiguous requirement and settling
it is exactly the judgement the brief is measuring.

### 3.2 Backend API

| ID | Requirement | Priority |
|---|---|---|
| FR-10 | `POST /api/v1/calculations` — performs the operation, returns JSON | Required |
| FR-11 | `GET /api/v1/operations` — supported operations and their operand counts | Proposed |
| FR-12 | Every response is `application/json` | Required |
| FR-13 | Input validation: unknown operation, wrong operand count, malformed JSON, non-numeric operand | Required |
| FR-14 | Edge cases: division by zero, square root of a negative, overflow (±Inf), undefined result (NaN) | Required |
| FR-15 | One error envelope, mapped in one place | Required |
| FR-16 | `GET /healthz` — liveness | Proposed |
| FR-17 | Results are kept in the in-process cache through the repository; a repeated request is not recomputed | Proposed |
| FR-18 | The response says whether the result came from the cache (`cached: true\|false`) | Proposed |

**FR-10 decision — why one endpoint rather than `/add`, `/subtract`?**
REST resources are nouns, not verbs. One endpoint per operation would mean seven endpoints, seven
handlers and seven test suites, and adding an operation would change the shape of the API. One
resource plus an `operation` field gives one validation path and one table-driven test. Justified in
the README.

**FR-11 decision:** the front end learns the keypad from the backend instead of hardcoding it. One
source of truth, at a cost of about ten lines.

**FR-15 error envelope:**

```json
{
  "error": {
    "code": "DIVISION_BY_ZERO",
    "message": "Division by zero is not defined."
  },
  "request_id": "01J9X2K7P4T8QW"
}
```

Rule: internal detail — stack traces, library error strings — **never** reaches the response. It
goes to the log; the response carries the `request_id` that matches it.

### 3.3 Front end

| ID | Requirement | Priority |
|---|---|---|
| FR-20 | An intuitive calculator interface for entering input and reading the result (keypad + display) | Required |
| FR-21 | Client-side input validation with immediate feedback | Required |
| FR-22 | Backend failures shown in terms the user can act on | Required |
| FR-23 | Responsive design — basic mobile support | Required |
| FR-24 | Loading and failure states: keys disabled while a request is in flight, retry on a network failure | Proposed |
| FR-25 | Keyboard support: digits, `+ - * /`, `Enter` for equals, `Escape` to clear | Proposed |

**FR-21 / FR-22 decision:** client-side validation is **a user-experience feature, not a security
boundary**. The backend is the authority; the front end repeats the rule, it does not replace it.
That sentence goes in the README — it is a distinction that is often skipped.

Failures are shown according to the backend's `code`, never its `message`. Parsing message text is
brittle and makes translation impossible.

**FR-20 decision:** calculator state is a `useReducer` — a pure reducer over
`{ display, entering, pendingOperand, operation, operandCount, errorCode, pending }` with explicit
actions (`digit`, `decimal`, `negate`, `operator`, `submit`, `result`, `failure`, `clear`). Redux
would be a large answer to a small question here, and a pure reducer is tested by calling it.

Chaining is the one piece that is not in the reducer. Finishing a pending calculation before the
next operator requires a request, and the reducer has no way to make one, so the hook decides when
to send and the reducer only records what came back.

---

## 4. Non-functional requirements

| ID | Requirement | Acceptance | In the brief |
|---|---|---|---|
| NFR-01 | Clean, readable, idiomatic code | `golangci-lint run` and `eslint` report nothing; `gofmt` clean | "Clean, readable, and idiomatic code" |
| NFR-02 | Business logic independent of any framework | `domain` imports no web framework | "testable architecture" |
| NFR-03 | No global state | No `init()`, no package-level mutable `var`; dependencies through constructors | "maintainable code" |
| NFR-04 | Unit tests over key functionality on both layers | Every row of the §5 table is a test case | "Unit tests covering key functionality for both layers" |
| NFR-05 | **Coverage report** | `COVERAGE.md` is committed and regenerated by `make cover` from both suites; the README explains the strategy; `domain` ≥ 90% | Deliverable 3 |
| NFR-06 | No data races | `go test -race ./...` clean | — |
| NFR-07 | Documentation: setup, how to run, API examples, design decisions | §6.1 | Deliverables 2 and 6 |
| NFR-08 | Accessibility basics | `aria-label` on every key; `role="status"` + `aria-live="polite"` on the display; touch targets ≥ 44×44 px | "Intuitive UI" and a quality signal |
| NFR-09 | No secrets in the repository | `.env.example` committed, `.env` ignored | — |
| NFR-10 | One command to start | `docker compose up --build` gives a working system | Deliverable 4 (optional) |
| NFR-11 | Graceful shutdown | On SIGTERM in-flight requests finish, then the process exits | Container maturity signal |
| NFR-12 | Server timeouts set | `ReadHeaderTimeout` (Slowloris), `ReadTimeout`, `WriteTimeout`, `IdleTimeout` | — |

**On NFR-05:** the brief counts the coverage report as a **deliverable**, so an artifact has to
ship, not just a command that would produce one. `COVERAGE.md` is generated rather than written,
because a percentage typed into prose is stale as soon as the next test lands — which happened
twice here before the report was generated. The strategy stays in the README, since that is the
part a number cannot carry.

**On NFR-10:** the brief marks a Dockerfile "Optional", but deliverable 4 says it should run
frontend and backend **together**, which in practice means `docker-compose.yml`. A reviewer will try
`docker compose up --build`; if that fails, nothing else matters. Verified from a clean clone before
submission.

---

## 5. Acceptance criteria — the edge-case table

This table is also the skeleton of the `domain` test file. Every row is a case in a table-driven
test.

| Input | Expected | HTTP | Code |
|---|---|---|---|
| `add [2, 3]` | `5` | 200 | — |
| `add [0, 0]` | `0` | 200 | — |
| `add [-2, 3]` | `1` | 200 | — |
| `subtract [1e-320, 1e-320]` | `0` (subnormal, not an error) | 200 | — |
| `divide [1, 3]` | `0.3333333333333333` | 200 | — |
| `divide [10, 0]` | error | 422 | `DIVISION_BY_ZERO` |
| `divide [0, 0]` | error | 422 | `DIVISION_BY_ZERO` |
| `multiply [1e308, 10]` | error (+Inf) | 422 | `NUMERIC_OVERFLOW` |
| `sqrt [0]` | `0` | 200 | — |
| `sqrt [-4]` | error | 422 | `NEGATIVE_SQRT` |
| `sqrt [2, 3]` | error (sqrt takes one operand) | 400 | `INVALID_OPERAND_COUNT` |
| `power [0, 0]` | `1` (Go's `math.Pow`; documented) | 200 | — |
| `power [0, -1]` | error (+Inf) | 422 | `NUMERIC_OVERFLOW` |
| `power [2, 10000]` | error (+Inf) | 422 | `NUMERIC_OVERFLOW` |
| `power [-8, 0.5]` | error (NaN) | 422 | `UNDEFINED_RESULT` |
| `percentage [50, 200]` | `100` | 200 | — |
| `operation: "modulo"` | error | 400 | `INVALID_REQUEST` |
| `operands: []` | error | 400 | `INVALID_OPERAND_COUNT` |
| `operands: [1,2,3]` | error | 400 | `INVALID_OPERAND_COUNT` |
| `operands: ["a", 2]` | error | 400 | `INVALID_REQUEST` |
| Malformed JSON | error | 400 | `INVALID_REQUEST` |
| `operands: [1e400, 2]` | error (+Inf after decoding) | 400 | `INVALID_REQUEST` |

### 5.1 Two traps, settled explicitly

**1. The zero-value trap.** The `required` rule in Go's common validation libraries treats a zero
value as "not supplied". A naive `binding:"required"` on the operands rejects `add [0, 0]` with a
400 — yet `0` is a perfectly valid operand, and this is the first edge case a reviewer tries.

The fix: decode operands into a pointer slice (`[]*float64`) so "not supplied" and "zero supplied"
stay distinguishable, and validate the operand count against the operation, which the domain already
has to know.

**2. The `1e400` trap.** JSON has no `NaN` or `Infinity` literal, but `1e400` **is valid JSON** and
decodes to `+Inf` in a `float64`. Operands must therefore be checked with `math.IsInf` /
`math.IsNaN` **after** decoding; schema validation alone does not catch it.

### 5.2 400 versus 422

- **400** — the request could not be understood (syntax or shape).
- **422** — the request was understood, but arithmetic cannot answer it.

The distinction is genuinely arguable; **consistency matters more than which side is "right"**. The
choice is justified in one sentence in the README.

Two cases that are easy to conflate are separated here: `±Inf → NUMERIC_OVERFLOW` and
`NaN → UNDEFINED_RESULT`. Both are 422.

### 5.3 Testing strategy

```
    ┌ Handler tests: httptest + gin.TestMode over the real service and cache
    │  → status codes, error envelope, validation behaviour, the cached flag
    │
    └ Unit tests: the domain engine (table-driven), the front-end reducer
       → every row of the §5 table
```

Handlers are tested against the real service rather than a mock, because the service is a concrete
type: an interface with one implementation, existing only to be faked, is what `CLAUDE.md` API-2
rules out. Where an interface genuinely varies — `service.Repository` — the test declares a small
hand-written fake.

Two details are deliberate:

- **`errors.Is` against a sentinel** — comparing error message strings is brittle.
- **`InDelta` for floats** — `==` on IEEE-754 values is unreliable.

The front end queries by `getByRole`, not `getByTestId`: testing through the accessibility tree is
the correct use of React Testing Library and it exercises NFR-08 at the same time.

---

## 6. Deliverables and instructions

The brief lists four deliverables, then adds two instructions that also ask for something
concrete. Both lists are mapped here.

### Deliverables

| # | The brief asks for | Delivered as | Status |
|---|---|---|---|
| 1 | A Git repository with frontend and backend code | One repository, `backend/` + `frontend/` | Required |
| 2 | A README with setup, API examples, design decisions | `README.md` | Required |
| 3 | Unit tests and a **coverage report** | `COVERAGE.md`, generated by `make cover` from both suites and committed; the strategy behind it in the README | Required |
| 4 | Optional: a Dockerfile running frontend and backend together | Two Dockerfiles + `docker-compose.yml` | Optional |

### Instructions that ask for an artifact

| # | The brief asks for | Delivered as |
|---|---|---|
| 3, 4 | Push to a Git host and share the link | Pushed; the link goes in the reply to Sezzle |
| 5 | **The prompts used** | `PROMPTS.md` |
| 6 | Four specific README sections | §6.1 |

> **On instruction 5:** it sits under Instructions rather than Deliverables, which makes it easy to
> read as a courtesy. It is not: "Share any prompts that you used in your work" asks for something
> that has to exist. Prompts are collected in `PROMPTS.md` as the work happens; reconstructing them
> afterwards does not work.

### 6.1 Required README sections

Listed explicitly in instruction 6 of the brief:

1. Setup instructions
2. How to run the frontend and the backend
3. API call examples (`curl`)
4. Design decisions and assumptions

Two more that carry disproportionate value:

5. **Trade-offs & Known Limitations** — knowing the limits of your own work is a stronger signal
   than claiming none.
6. **What I Would Do Next** — where the layers taken out of scope in §1 (gateway, cache, history,
   metrics) are explained along with the scale that would justify them.

No empty sections and no "TODO". A section that will not be written is not opened.

---

## 7. Decisions and assumptions

Each is justified in a paragraph in the README's "Design decisions".

| # | Decision | Reason |
|---|---|---|
| D-01 | Go backend, React + TypeScript front end | The brief: "Go is preferred", "TypeScript preferred" |
| D-02 | **One backend service** — no gateway, history or cache service | The brief asks for one service and prefers correctness over extra features. See §1 |
| D-03 | One `POST /calculations`, not an endpoint per operation | Resources are nouns; one validation path, one test table |
| D-04 | `percentage` = `a * b / 100` | An ambiguous requirement; the reading that matches the `%` key was chosen |
| D-05 | `float64`, not a decimal type | IEEE-754 limits documented in the README; financial precision is out of scope |
| D-06 | 400 = syntax, 422 = semantics | Consistency comes first |
| D-07 | `domain` imports no framework | Testability is an explicit criterion in the brief |
| D-08 | Failures shown by `code`, never by `message` | Parsing text is brittle and blocks translation |
| D-09 | Client validation is UX; the backend is the authority | The security boundary is on the server |
| D-10 | HTTP framework: Gin | `net/http`-compatible, so `httptest` and standard middleware work directly, with binding and validation included. *The leaner alternative is Go 1.22+ `net/http` routing — zero dependencies for two endpoints; if that is preferred, D-10 changes.* |
| D-11 | Front-end state is `useReducer`, not Redux | Redux at this size signals one hammer for every problem; a pure reducer is tested directly |
| D-12 | ADRs as a README section, not separate files | Five ADR files for one service is documentation bloat |
| D-13 | The repository pattern over an in-process LRU cache | Demonstrate the pattern with a real responsibility rather than an empty interface. See §2.2 |
| D-14 | The cache is in-process, not Redis | Redis means an external dependency and a second container, for no gain at this scale |
| D-15 | `domain.Calculate` is a plain function, not an interface | A single-implementation interface breaks the YAGNI rule (API-2); a pure function needs no fake |

### Assumptions

- **A-01:** Authentication is not asked for. The README says when a gateway and JWT would be needed.
- **A-02:** Persistence and calculation history are not asked for — neither appears in the brief.
- **A-03:** Operations are binary and chain left to right, the way a pocket calculator works:
  pressing an operator finishes whatever is on screen first, so `9 + 1 * 2` is 20, not 11. Operator
  precedence is not implemented and is not asked for.
- **A-04:** The interface and its messages are in English, on the assumption the reviewer reads
  English.
- **A-05:** "~2–4 hours" is a suggestion and the submission window is five business days. This took
  roughly seven, with the extra time in tests and documentation — both explicit deliverables, so
  neither is where time gets cut.

---

## 8. Open questions

All closed on 2026-08-27.

| # | Question | Decision |
|---|---|---|
| Q-01 | Build the optional operations? | **Yes** — `power`, `sqrt` and `percentage` are in scope |
| Q-02 | Gin or the standard library's `net/http`? | **Gin** (D-10) |
| Q-03 | Go module path? | `calculator`, local; to be updated when a remote is added |
| Q-04 | How does the repository pattern apply to a stateless service? | Over the in-process LRU cache (D-13, §2.2) |
