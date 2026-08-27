# CLAUDE.md — Working Rules

Rules and boundaries for anyone, human or AI, working in this repository.

**How to read the levels.** **MUST** is enforced by review and by the tooling gates in §16;
breaking one blocks a change. **SHOULD** is a strong default that needs a stated reason to skip.
**CAN** is allowed without further approval, and is not an invitation to add it.

Rule IDs are stable. Cite them in review (`violates CC-2`) rather than restating the rule. Deprecate
with a note; never renumber.

**Project:** Full-stack calculator — Go backend + React/TypeScript frontend.
**Requirements:** `SPEC.md` is authoritative. Read it before writing code.
**Build record:** `claudedocs/workflow_calculator.md` — what was built, in which order.
**Prompt log:** `PROMPTS.md` — every prompt appended verbatim.

---

## 1 — Before coding

- **BP-1 (MUST)** Ask clarifying questions when a requirement is ambiguous. Do not guess and build.
- **BP-2 (MUST)** Confirm the approach — API shape, data flow, failure modes — before writing code.
  For this project that confirmation already exists in `SPEC.md` and the workflow document; extend
  them rather than re-deciding per function, and update them when behaviour changes.
- **BP-3 (SHOULD)** When more than two approaches are viable, list the trade-offs and say which one
  was chosen and why. `SPEC.md` §7 is where those land.
- **BP-4 (SHOULD)** Decide the testing strategy and the observability signals up front, not after.

---

## 2 — Test-driven development

No production code is written before a failing test that requires it.

**The cycle, every time:**

1. **Red** — write the smallest test that fails for the right reason. Run it. Confirm it fails, and
   that the failure message is the expected one. A test that passes on its first run tested nothing.
2. **Green** — write the least code that makes it pass. Not the elegant version. The passing one.
3. **Refactor** — clean up with the test as the safety net. Run the tests again.

- **T-1 (MUST)** Tests are deterministic and hermetic: no clock, no network, no shared temp state.
  Table-driven where cases differ only by data; a behaviour with a different shape — an eviction
  sequence, a two-request cache interaction — stays its own named test. Both forms are required to
  be readable, which is the point of the rule.
- **T-2 (MUST)** `go test -race ./...` runs in CI. Use `t.Cleanup` for teardown, not `defer` in the
  test body, so helpers can register their own.
- **T-3 (SHOULD)** Mark tests `t.Parallel()` when they are safe to run in parallel.
- **T-4 (MUST)** Never write a batch of code and backfill tests. That is not TDD.
- **T-5 (MUST)** One behaviour per test. The name states the behaviour, not the function:
  `divide by zero is rejected`, not `TestDivide`.
- **T-6 (MUST)** Test the public surface of a layer, not its internals. Testing private helpers
  welds the test to the implementation and makes refactoring expensive.
- **T-7 (MUST)** Compare errors with `errors.Is` and a sentinel. Compare floats with
  `require.InDelta`, never `==`.
- **T-8 (MUST)** A bug fix starts with a failing test that reproduces the bug.
- **T-9 (SHOULD)** Frontend: query by accessible role (`getByRole`), not `getByTestId`. That tests
  the accessibility tree and the behaviour at once.

---

## 3 — Design principles

Textbook definitions are not the point; these are the calls to make in this repository.

**YAGNI** — Build what `SPEC.md` requires, nothing else. No feature flags, no config for a value
that never changes, no hooks for a future that has not been specified. If a capability is genuinely
needed, update `SPEC.md` first, then build it.

**KISS** — Boring beats clever. Clever is what someone decodes at 3 a.m. Prefer the standard library
over a dependency, a native platform feature over a library, one clear function over a hierarchy. If
a reviewer needs the author to explain it, simplify it.

**DRY** — One source of truth per piece of knowledge. The operation list lives in the domain and is
served over the API; the frontend does not keep a second copy. Error-to-status mapping lives in
exactly one function. **But** DRY is about knowledge, not characters: two similar lines that change
for different reasons stay separate. Premature deduplication couples what should move apart.

**SOLID** —
- *Single responsibility:* a handler translates HTTP to domain and back. It does not calculate. A
  domain function calculates. It does not know HTTP exists.
- *Open/closed:* adding an operation means a row in `specs`, a switch case and a test row — not
  edits to the handler, the validator and the router.
- *Liskov:* an implementation behind an interface honours the whole contract, error behaviour
  included. No "this one panics instead".
- *Interface segregation:* small interfaces. A consumer needing one method declares a one-method
  interface.
- *Dependency inversion:* inner layers declare the interfaces they need; outer layers supply the
  implementations. Dependencies point inward, always.

---

## 4 — Modules and dependencies

- **MD-1 (MUST)** No dependency is added without asking first. This is stricter than the usual
  "prefer stdlib" guidance, and deliberately so: every package is a maintenance liability, and this
  project is explicitly scored on restraint. If a few lines of standard library cover it, write the
  few lines.
- **MD-2 (SHOULD)** When a dependency is justified, check its transitive weight and licence before
  adding it, not after.
- **MD-3 (CAN)** `govulncheck` for dependency updates.

---

## 5 — Code style

- **CS-1 (MUST)** `gofmt` clean, `go vet ./...` clean, `golangci-lint run` clean.
- **CS-2 (MUST)** No stutter in names: `package memory; type Cache` — not `MemoryCache` inside
  `memory`.
- **CS-3 (SHOULD)** Small interfaces declared next to their consumer. Composition over inheritance.
- **CS-4 (SHOULD)** No reflection on hot paths. Generics only where they clarify.
- **CS-5 (MUST)** A function taking more than two arguments takes an input struct instead.
  `context.Context` does not count toward the total and never goes inside the struct. `*testing.T`
  is treated the same way, and test helpers are exempt: the rule protects the readability of
  production call sites, and applying it to fixtures only adds ceremony.
- **CS-6 (SHOULD)** Declare an input struct immediately before the function that consumes it.
- **CS-7 (MUST)** Comments explain *why*, not *what*. Code needing a comment to say what it does
  should be rewritten instead.
- **CS-8 (MUST)** No commented-out code — version control remembers it. No `TODO` left in a
  delivered branch: do it, or record it in `SPEC.md` §8.

**TypeScript / React**
- **CS-9 (MUST)** `strict` mode on. An `any` needs a comment explaining why nothing else works.
- **CS-10 (MUST)** Function components and hooks only. Calculator state is a pure reducer, tested
  directly.
- **CS-11 (MUST)** `eslint` clean.

---

## 6 — Errors

- **ERR-1 (MUST)** Wrap with `%w` and context when an error crosses a boundary:
  `fmt.Errorf("open %s: %w", path, err)`. A bare sentinel returned by the package that defines it is
  already in context and needs no wrapping.
- **ERR-2 (MUST)** `errors.Is` / `errors.As` for control flow. Never string matching.
- **ERR-3 (SHOULD)** Sentinel errors declared in the package that produces them, each documented
  with the condition it reports.
- **ERR-4 (MUST)** Errors are values. `panic` is not an error-handling strategy.
- **ERR-5 (MUST)** Never swallow an error. `_ = doSomething()` is banned. An error that is
  deliberately tolerated — a cache write that failed — is logged, and the reason it is tolerated is
  in a comment.
- **ERR-6 (MUST)** Exactly one function maps domain errors to HTTP status and code. Adding an error
  means adding one case there.
- **ERR-7 (MUST)** Internal detail — stack traces, driver messages, SQL — never reaches a response
  body. It goes to the log; the response carries the request id that matches it.
- **ERR-8 (CAN)** `context.WithCancelCause` / `context.Cause` to propagate a cancellation reason.

---

## 7 — Concurrency

- **CC-1 (MUST)** The sender closes a channel. Receivers never close.
- **CC-2 (MUST)** Every goroutine's lifetime is tied to a `context.Context`. No leaks.
- **CC-3 (MUST)** Shared state is protected by `sync.Mutex` or `sync/atomic`. There is no
  "probably safe".
- **CC-4 (SHOULD)** `errgroup` for fan-out work, cancelling on the first error.
- **CC-5 (CAN)** Buffered channels only with a stated reason (throughput or back-pressure).

---

## 8 — Contexts

- **CTX-1 (MUST)** If a function takes a `context.Context` it is the first parameter. Never store a
  context in a struct.
- **CTX-2 (MUST)** Propagate a non-nil context and honour `Done`, deadlines and timeouts.
- **CTX-3 (SHOULD)** Every call that does I/O or may block carries a context with a deadline.
- **CTX-4 (CAN)** `WithX(ctx)` helpers that derive deadlines from config.

---

## 9 — Structure and boundaries

### 9.1 Layers and import direction

```
transport/rest  ──►  service  ──►  domain
   (Gin, DTOs)      (use cases)   (pure logic)
                         │
                         └──►  repository  (interface declared by service)
```

Dependencies point **inward only**:

- **ST-1 (MUST)** `domain` imports no web framework, no HTTP package, no database driver, no DTO.
  Standard library only. This is what makes it trivially testable and it is non-negotiable.
- **ST-2 (MUST)** `transport` may import `service` and `domain`. `service` may import `domain`.
  `domain` imports neither. No package imports `transport`.
- **ST-3 (MUST)** A framework type — `gin.Context`, `http.Request` — reaching `service` or `domain`
  is a defect even when it compiles.
- **ST-4 (MUST)** Directory layout follows `SPEC.md` §2.1. Adding a directory means updating it
  first. `internal/` is deliberate: the compiler enforces the boundary, so the layer rule is not
  merely a convention.

### 9.2 Repository pattern

All data access goes through a repository. Nothing above it knows where data lives.

- **RP-1 (MUST)** The interface is declared by the **consumer** (`service`), not by the package that
  implements it. This is the Go idiom and it keeps the dependency pointing inward.
- **RP-2 (MUST)** The implementation lives in its own package and returns concrete types.
- **RP-3 (MUST)** The interface speaks the domain's language —
  `Find(ctx, domain.Request) (domain.Calculation, bool, error)` — never SQL, rows, keys or driver
  types.
- **RP-4 (MUST)** Storage errors are translated at the repository boundary. A driver error never
  escapes upward.
- **RP-5 (MUST)** Every repository method takes `context.Context` first.
- **RP-6 (MUST)** A store that exists to make things faster is never on the critical path: if it
  cannot be read or written, log it and carry on. A broken cache must not fail a valid request.

Applied here to an in-process LRU cache of completed calculations (`SPEC.md` §2.2, D-13). Replacing
it with a database is a constructor change in `main.go` and nothing else — that is the point of the
pattern, and it belongs in the README.

### 9.3 Dependency injection

- **DI-1 (MUST)** No global mutable state. No package-level `var db`, no `init()` doing setup.
- **DI-2 (MUST)** Every dependency passes through a constructor: `NewCalculator(repo, logger)`.
- **DI-3 (MUST)** Wiring happens exactly once, in `cmd/server/main.go`. Nowhere else.

---

## 10 — APIs and boundaries

- **API-1 (MUST)** Document every exported item: `// Foo does …`. Keep the exported surface minimal.
- **API-2 (MUST)** Accept interfaces where variation is real; return concrete types. An interface
  with one implementation that exists only to be faked is banned by YAGNI — test the concrete type
  through its public surface instead.
- **API-3 (SHOULD)** Functions small, orthogonal, composable.
- **API-4 (MUST)** Wire shapes are decided in `SPEC.md`, not in a handler. Clients branch on the
  error `code`, never on the message text, which is free to change or be translated.
- **API-5 (CAN)** Constructor options pattern for extensibility — not used here: both constructors
  take two arguments, and options for a fixed set of fields is ceremony. Revisit if one grows.

---

## 11 — Logging and observability

- **OBS-1 (MUST)** Structured logging with `log/slog`. Constant message, variables as fields. Never
  `fmt.Sprintf` into the message: a formatted message cannot be aggregated.
- **OBS-2 (MUST)** Every request gets an id, echoed in a response header and carried in the context,
  so a user's report can be matched to a log line.
- **OBS-3 (CAN)** pprof or debug endpoints, local-only or behind auth. Out of scope here.

---

## 12 — Configuration

- **CFG-1 (MUST)** Config comes from the environment, is validated at startup, and the process
  refuses to start when it is wrong. A half-configured service is worse than one that is down.
- **CFG-2 (MUST)** Config is immutable after init and passed explicitly. Never a global.
- **CFG-3 (SHOULD)** Sane defaults, documented in `.env.example`.
- **CFG-4 (MUST)** `.env.example` is committed. `.env` is not.

---

## 13 — Security

- **SEC-1 (MUST)** Validate every input at the trust boundary. Set explicit I/O timeouts —
  `ReadHeaderTimeout` above all, which is what stops Slowloris. Bound anything a client can grow:
  request bodies, and any in-memory store keyed by client input.
- **SEC-2 (MUST)** Never log secrets. Secrets live in the environment or a secret manager, never in
  code.
- **SEC-3 (SHOULD)** Least privilege by default: the container runs as a non-root user on a
  distroless image, with no shell and no package manager.
- **SEC-4 (SHOULD)** TLS at the edge. This service speaks plain HTTP inside the compose network and
  terminates TLS at the proxy in front of it; that is a deployment boundary, not an omission, and it
  is stated in the README rather than left implied.
- **SEC-5 (CAN)** Fuzz tests for untrusted input parsing.

---

## 14 — Performance

- **PERF-1 (MUST)** Measure before optimising: `pprof`, `go test -bench`, `benchstat`. A change
  justified by "this should be faster" is not justified.
- **PERF-2 (SHOULD)** No allocations on hot paths without reason; prefer `bytes` / `strings` APIs.
- **PERF-3 (CAN)** Microbenchmarks for critical functions, tracked in CI. Not adopted here: nothing
  in this service is hot enough to defend a benchmark suite.

---

## 15 — Writing functions

Before leaving a function, walk this list. If step 1 passes, stop.

1. Can you read it and honestly follow what it does? If yes, you are finished.
2. Is the cyclomatic complexity high — many independent paths, deep `if`/`else` nesting? That is a
   signal, not a proof, but it is usually right.
3. Would a known data structure or algorithm make it simpler and more robust — a parser, a tree, a
   stack, a queue?
4. Are there hidden or untested dependencies, or values that should be parameters instead? Only
   non-trivial ones that can actually change.
5. Brainstorm three better names. Is the current one the best, and consistent with the rest of the
   codebase?

---

## 16 — CI/CD and tooling gates

- **G-1 (MUST)** `go vet ./...` passes.
- **G-2 (MUST)** `golangci-lint run` passes with the project config.
- **G-3 (MUST)** `go test -race ./...` passes.
- **CI-1 (MUST)** Lint, vet, test with `-race`, and build run on every push and pull request, with
  module and build caches.
- **CI-2 (MUST)** Reproducible builds: `-trimpath`, and the version embedded through
  `-ldflags "-X main.version=$TAG"`.
- **CI-3 (SHOULD)** A change touching a MUST rule needs review sign-off.
- **CI-4 (CAN)** SBOM publication, `govulncheck`, licence checks.
- **TL-1 (CAN)** `gofumpt`, `staticcheck` beyond the configured set.
- **TL-2 (CAN)** `govulncheck` and dependency scanners.
- **TL-3 (CAN)** `gotestsum`. **Not** `mockgen` or `counterfeiter`: generated mocks encourage the
  one-implementation interfaces API-2 rules out. Where an interface genuinely varies, a hand-written
  fake in the test file is shorter and clearer.
- **TL-4 (CAN)** `oapi-codegen` for OpenAPI, `buf` for Protobuf.

---

## 17 — Definition of done

A change is not done until all of these hold:

- [ ] A test existed before the code, and it failed first.
- [ ] `go test -race ./...` and the frontend suite pass.
- [ ] `go vet` and both linters report zero findings.
- [ ] No dependency was added without being asked and agreed (MD-1).
- [ ] Layer boundaries in §9.1 are intact.
- [ ] Every exported item is documented (API-1).
- [ ] `SPEC.md` still describes what the code does — if behaviour changed, the spec changed with it.
- [ ] `PROMPTS.md` has the prompts that produced the change, appended verbatim.

---

## 18 — Where rules collide

Eight tensions have already been resolved, each stated at the rule it belongs to: MD-1 (stdlib
preference raised to "ask first"), CS-5 (test fixtures), API-5 (options pattern), OBS-3 and PERF-3
(not adopted), SEC-4 (TLS at the edge), BP-2 (the spec is the confirmed approach), T-1 (table-driven
versus one behaviour per test) and TL-3 (no generated mocks). Resolve a new one the same way — in
place, not in a second list.

---

## 19 — When a rule blocks you

Do not silently work around a rule. Name the rule, say why it is in the way, propose the change, and
wait for a decision. A rule that is quietly bypassed is worse than no rule: it makes the whole file
untrustworthy.
