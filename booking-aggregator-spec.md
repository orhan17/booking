# Booking Aggregator — Go / Hexagonal Showcase Project

A small but production-shaped Go service that aggregates offers from multiple
"tour operators" behind a Ports & Adapters architecture. Built to demonstrate
clean architecture, Go concurrency, and resilient integration design — a
scaled-down mirror of a real auto-booking system.

**Goal:** a repo that backs your strongest, most-scrutinized resume bullets
(Hexagonal, DDD, Go concurrency, resilient operator integrations) with real,
readable code you can defend line-by-line in an interview.

---

## What it does (scope)

- Exposes a REST endpoint: `POST /search` — takes search criteria (origin,
  destination, dates, passengers), returns aggregated offers.
- Fans out the query **concurrently** to N operator adapters (mocked: Anex,
  Coral, Sunmar).
- Each operator has its own latency/failure behavior. A slow or failing
  operator **must not** break the whole response — results degrade gracefully.
- `POST /bookings` — books a selected offer through the right operator adapter,
  with a **pre-booking reconciliation** check (price/availability re-validation)
  before confirming.
- `GET /bookings/{id}` — returns booking state (state machine:
  `pending → confirmed → failed`).

Deliberately **out of scope** (keep it focused): real payment, auth, a UI,
real operator APIs. Mocks are enough to show the architecture.

---

## Architecture (Ports & Adapters / Hexagonal)

Core domain knows nothing about HTTP, operators, or the database. Everything
external plugs in through ports.

```
booking-aggregator/
├── cmd/
│   └── api/
│       └── main.go              # wiring: build adapters, inject into app, start HTTP
├── internal/
│   ├── domain/                  # pure domain — no framework, no I/O
│   │   ├── offer.go             # Offer value object
│   │   ├── booking.go           # Booking aggregate + state machine
│   │   ├── search.go            # SearchCriteria value object + validation
│   │   └── errors.go            # domain errors
│   ├── app/                     # application layer — use cases orchestrating ports
│   │   ├── search_service.go    # fan-out to operators, aggregate, sort
│   │   └── booking_service.go   # reconcile + book + persist
│   ├── ports/                   # interfaces (the "hexagon" boundary)
│   │   ├── inbound.go           # SearchUseCase, BookingUseCase (driven by HTTP)
│   │   └── outbound.go          # OperatorPort, BookingRepository
│   └── adapters/
│       ├── inbound/
│       │   └── http/            # REST handlers → call inbound ports
│       │       ├── router.go
│       │       ├── search_handler.go
│       │       └── booking_handler.go
│       └── outbound/
│           ├── operators/       # one adapter per operator, all implement OperatorPort
│           │   ├── anex.go
│           │   ├── coral.go
│           │   ├── sunmar.go
│           │   └── mock.go      # configurable latency/error for tests + demo
│           └── persistence/
│               └── memory_repo.go   # in-memory BookingRepository (swap for Postgres later)
├── CLAUDE.md
├── README.md
├── Makefile
├── Dockerfile
├── docker-compose.yml
└── go.mod
```

**The rule that makes it Hexagonal:** dependencies point *inward*.
`domain` imports nothing internal. `app` imports `domain` + `ports`. `adapters`
import `ports` (and `domain` types). `main.go` is the only place that knows the
concrete adapters and wires them together. Adding a new operator = add one file
in `adapters/outbound/operators/` implementing `OperatorPort`, register it in
`main.go`. The core never changes.

---

## Features mapped to resume bullets

| Feature to build | Resume bullet it proves |
|---|---|
| Ports & Adapters layout, operators as pluggable adapters | *"Built the booking service on Ports & Adapters — new operators added without touching the core"* |
| `SearchService` fans out with goroutines + `errgroup` + bounded pool | *"Used Go concurrency to query multiple operators in parallel"* |
| `context.Context` per-operator timeout, one slow operator dropped | *"Handled operator timeouts with graceful degradation"* |
| Booking aggregate + state machine, invalid transitions rejected | *"Modeled the booking lifecycle as an explicit state machine"* |
| Pre-booking reconciliation (re-validate price/availability) | *"Reconciliation step before confirming each booking"* |
| Domain value objects, validation in the domain | *"DDD elements — business rules framework-independent and testable"* |
| Table-driven unit tests + adapter integration tests | *"Testing at every layer (PHPUnit / Go testify)"* — Go side |

---

## Concurrency design (the part interviewers dig into)

In `SearchService.Search`:

1. Build a `context.WithTimeout` (e.g. 3s overall budget).
2. Use `golang.org/x/sync/errgroup` with `SetLimit(n)` for a **bounded** fan-out.
3. Each goroutine calls one `OperatorPort.Search(ctx, criteria)`.
4. A failing/timing-out operator logs a warning and returns **no** results for
   that operator — it does **not** fail the whole request (graceful degradation).
5. Collect results into a mutex-guarded slice (or a channel), then sort/merge.

Be ready to explain on a call: why bounded concurrency (avoid goroutine
explosion + respect upstream limits), why `context` (cancellation/timeout
propagation), why you don't fail-fast on one operator (availability > completeness).

---

## Testing plan

- **Domain unit tests** — booking state transitions (valid + invalid), search
  criteria validation, offer comparison. Table-driven.
- **App unit tests** — `SearchService` with mock operators: one fast, one slow
  (exceeds timeout → dropped), one erroring → assert graceful degradation.
- **Adapter integration tests** — HTTP handlers via `httptest`, in-memory repo.
- Target: meaningful coverage on `domain` + `app`, not a coverage number for its
  own sake.

---

## README plan (this is what recruiters actually read)

1. One-line what + why.
2. **Architecture diagram** (a simple ASCII or Mermaid hexagon — core, ports,
   adapters). This single image does most of the selling.
3. "Design decisions & trade-offs" section — 4–5 bullets: why Hexagonal, why
   bounded fan-out, why graceful degradation, why in-memory repo (and how you'd
   swap Postgres). *This section is what signals senior.*
4. How to run (`make run` / `docker compose up`).
5. Example request/response.

---

## Kickoff prompt for Claude Code

Paste this as your first message in Claude Code (after `git init` + creating an
empty repo folder). The `CLAUDE.md` file goes in the repo root first so Claude
Code follows the architecture.

> Build a Go 1.22+ REST service called `booking-aggregator` following strict
> Hexagonal (Ports & Adapters) architecture. Read `CLAUDE.md` first and follow
> it exactly — dependencies point inward, domain imports nothing internal.
>
> Start with the domain layer only: `Offer` (value object), `SearchCriteria`
> (value object with validation), and `Booking` (aggregate with an explicit
> state machine: pending → confirmed → failed, rejecting invalid transitions).
> Add table-driven unit tests for the state machine and validation. Do not write
> HTTP, operators, or persistence yet.
>
> Show me the domain package and its tests, then stop so I can review before we
> add ports and the application layer.

Build it **incrementally** — domain → ports → app (with concurrency) →
operator adapters → HTTP → wiring → README. Review at each step. That way you
understand every line and the git history itself looks like clean, deliberate
senior work.
