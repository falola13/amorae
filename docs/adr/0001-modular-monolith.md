# ADR 0001: Modular monolith for the API

- Status: accepted
- Date: 2026-09-21

## Context

Amorae is new. The domain boundaries are guesses, the team is small, and
nothing is under load yet. The request was for an architecture that is "SOLID
and scalable".

## Decision

Build the API as **one Go binary with internal feature modules**
(`internal/modules/*`). Modules talk to each other only through small
interfaces declared by the consumer, and one composition root (`internal/app`)
wires them together.

## Consequences

- One deploy, one database, one log stream, and in-process calls. Transactions
  across features are a normal `InTx`, not a saga.
- Scaling is horizontal: run more replicas of the same stateless binary.
- A module can later be extracted into a service by swapping its in-process
  adapter for a network client. The interface is already there. What *won't*
  come for free is splitting the database, so modules must not join across each
  other's tables. Go through the owning module's repository instead.
- The risk is the classic one: boundaries erode into a big ball of mud. The
  dependency table in `ARCHITECTURE.md` and code review are the defence.

## Alternatives considered

- **Microservices from day one.** Rejected. It pays the cost of distributed
  systems (network failure, eventual consistency, N deploy pipelines, distributed
  tracing) before there's any load or team structure that needs it, and it
  locks in boundaries before they're understood.
- **Layered monolith (`handlers/`, `services/`, `repositories/` at the top level).**
  Rejected. Every feature change touches every layer directory, and there's no
  unit you could ever extract.
