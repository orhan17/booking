# CLAUDE.md — booking-aggregator

Go service that aggregates offers from multiple tour operators behind a
Ports & Adapters (Hexagonal) architecture. Keep this file authoritative.

## Golden rule: dependencies point inward

- `internal/domain` — pure. Imports **nothing** internal, no third-party I/O,
  no framework. Only the standard library and its own types.
- `internal/app` — use cases. Imports `domain` and `ports` only.
- `internal/ports` — interfaces only. No implementations.
- `internal/adapters` — implement ports. May import `ports` and `domain` types.
- `cmd/api/main.go` — the ONLY place that knows concrete adapters and wires them.

If a change would make `domain` or `app` import an adapter, framework, or
external client — stop. That's an architecture violation. Introduce a port instead.

## Conventions

- Go 1.22+. Module path: `github.com/orhan17/booking-aggregator`.
- Errors: wrap with `fmt.Errorf("...: %w", err)`. Domain errors live in
  `domain/errors.go` as sentinel errors, checked with `errors.Is`.
- No global state. Dependencies are injected via constructors (`NewXxx(...)`).
- Context is the first argument on every port method and use case:
  `Search(ctx context.Context, ...)`.
- Keep functions small; keep the domain free of `net/http`, DB drivers, and
  operator SDKs.

## Concurrency (SearchService)

- Fan out to operators with `golang.org/x/sync/errgroup`, bounded via
  `g.SetLimit(n)`.
- Propagate an overall timeout with `context.WithTimeout`.
- A single operator failing or timing out must NOT fail the request — log it and
  return that operator's results as empty. Availability over completeness.
- Guard shared result slices with a mutex, or collect via channel.

## Testing

- Table-driven tests. Every domain rule (state transitions, validation) is tested.
- `SearchService` tests use mock operators: fast / slow-past-timeout / erroring —
  assert graceful degradation.
- HTTP handlers tested via `net/http/httptest`.
- Run: `go test ./...`. Keep tests fast and deterministic (no real sleeps beyond
  small, controlled ones in mocks).

## Commands

- `make run` — run the API locally.
- `make test` — `go test ./... -race`.
- `make lint` — `golangci-lint run`.
- `docker compose up` — run the service in a container.

## Workflow

- Build incrementally: domain → ports → app → adapters → http → wiring → README.
- Pause after each layer for review. Don't scaffold the whole tree at once.
- Prefer small, reviewable commits with clear messages.

## Do not

- Do not add payment, auth, or a real DB unless asked — in-memory repo is the
  default; Postgres is a later, optional swap behind `BookingRepository`.
- Do not put business logic in HTTP handlers — they only translate HTTP ↔ ports.
- Do not import adapters from `app` or `domain`.
