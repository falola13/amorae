# API contract (v1)

The Go API is the source of truth. `apps/web/src/lib/api/types.ts` mirrors the
shapes below. Change both in the same PR.

## Conventions

- Base URL: `http://localhost:8088` in development.
- JSON in and out. Request bodies are capped at 1 MiB, and unknown fields are rejected.
- Timestamps are RFC 3339, UTC.
- IDs are UUIDv7 strings.
- Every response carries an `X-Request-ID` header. Send your own (≤128 chars,
  `[A-Za-z0-9-_.]`) to correlate logs across services.
- Authenticated endpoints take `Authorization: Bearer <token>`.

### Success envelope

```json
{ "data": { } }
```

### Error envelope

```json
{
  "error": {
    "code": "validation_failed",
    "message": "Some fields are invalid.",
    "fields": { "email": "Enter a valid email address." },
    "request_id": "0f9c…"
  }
}
```

`code` is stable and meant for programs to branch on. `message` and `fields`
are safe to show to users. Internal details are never included. A 500 always
reads `internal_error` / "Something went wrong. Please try again.", and you look
it up in the logs by `request_id`.

| Status | When |
| --- | --- |
| 400 | `invalid_json`, `validation_failed` |
| 401 | `unauthenticated`, `invalid_credentials` |
| 403 | `forbidden` |
| 404 | `*_not_found` |
| 409 | `email_taken` (and future conflicts) |
| 429 | `rate_limited`, with a `Retry-After` header in seconds |
| 500 | `internal_error` |

### Rate limits

| Scope | Limit |
| --- | --- |
| All `/v1/auth/*` endpoints | 20 requests per client IP per minute |
| `POST /v1/auth/login` | 10 attempts per email per 15 minutes, counted whether or not the account exists |

Clients should wait `Retry-After` seconds before trying again. The message is
the same for every limit, so a 429 never reveals whether an account exists.

## Types

```ts
interface User {
  id: string;
  email: string;
  display_name: string;
  created_at: string;
  updated_at: string;
}

interface AuthResult {
  token: string;      // opaque; send as a Bearer token
  expires_at: string;
  user: User;
}
```

## Endpoints

### `POST /v1/auth/register`

```json
{ "email": "ada@example.com", "password": "correct horse", "display_name": "Ada" }
```

- `201` → `{ "data": AuthResult }`
- `400 validation_failed`: `fields` may contain `email`, `password`, `display_name`
- `409 email_taken`
- `429 rate_limited`: see *Rate limits*

Rules: the email is trimmed, lower-cased and must be a bare address. The password is
8–72 bytes. The display name is trimmed and 1–50 characters.

### `POST /v1/auth/login`

```json
{ "email": "ada@example.com", "password": "correct horse" }
```

- `200` → `{ "data": AuthResult }`
- `401 invalid_credentials`: identical for unknown email and wrong password
- `429 rate_limited`: see *Rate limits*

### `POST /v1/auth/logout` (auth)

- `204`. Idempotent: logging out an already-dead session also returns 204.

### `GET /v1/users/me` (auth)

- `200` → `{ "data": User }`
- `401 unauthenticated`

### `PATCH /v1/users/me` (auth)

```json
{ "display_name": "Ada L." }
```

- `200` → `{ "data": User }`
- `400 validation_failed`

## Operational endpoints (no envelope)

| Endpoint | Purpose |
| --- | --- |
| `GET /healthz` | Liveness. `200 {"status":"ok"}` whenever the process can serve. |
| `GET /readyz` | Readiness. `200 {"status":"ready"}`, or `503 {"status":"unavailable"}` if Postgres is unreachable. |
| `GET /metrics` | Prometheus metrics, served on a **separate listener** (`METRICS_ADDR`, default `127.0.0.1:9090`), never on the API port. |
