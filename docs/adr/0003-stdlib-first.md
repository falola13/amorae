# ADR 0003: Standard library first, and every dependency earns its place

- Status: accepted
- Date: 2026-09-21

## Context

A template's dependencies become every future feature's dependencies. Each one
is something to learn, upgrade and patch, and something the team has to be able
to explain.

## Decision

Backend:

| Need | Choice | Why not something bigger |
| --- | --- | --- |
| Routing | `net/http.ServeMux` (method + path patterns, Go ≥ 1.22) | chi/gin/echo add little now that the stdlib matches methods and path values. |
| Logging | `log/slog` | Structured, levelled, in the stdlib. |
| Postgres | `pgx/v5` + hand-written SQL | An ORM hides the queries you'll need to tune. SQL in one file per module is easy to review. |
| Migrations | `goose/v3`, embedded | Versioned up/down SQL, advisory-locked, and it ships inside the binary. |
| Passwords | `golang.org/x/crypto/bcrypt` | Maintained by the Go team. |
| Metrics | `prometheus/client_golang` | The de facto standard. |
| DI | Plain constructors in `internal/app` | A container (wire, fx, dig) hides the object graph that the composition root is meant to show. |

Frontend: Next.js, React, Tailwind, and `server-only`. No validation, data
fetching, state or UI-kit libraries until a concrete need shows up.

## Consequences

- Anyone who knows Go and React can read the whole template without looking
  anything up.
- Some things are hand-rolled (the JSON envelope, form state). They're small,
  and they're tested where it matters.
- To add a dependency, write down in the PR what it replaces and why the
  stdlib or an existing dependency doesn't cover it.
