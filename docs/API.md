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
  timezone: string;   // IANA zone, e.g. "Africa/Lagos"; "UTC" for new accounts
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
at least 10 characters (Unicode characters, so an emoji counts as one) and at most
72 bytes (bcrypt's limit; accents and emoji take 2–4 bytes). The web app applies the
identical rule. The display name is trimmed and 1–50 characters.

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
{ "display_name": "Ada L.", "timezone": "Africa/Lagos" }
```

- `200` → `{ "data": User }`
- `400 validation_failed`: `fields` may contain `display_name`, `timezone`
- `timezone` is optional (omit or `""` to leave it unchanged) and must be an IANA zone name.
- `email` is **not** accepted here (it's rejected as an unknown field): use the endpoint below.

### `PUT /v1/users/me/email` (auth)

Changing the email changes what the account logs in with, so it needs the current
password: a session left open on someone else's device isn't enough to take the
account over.

```json
{ "email": "ada.new@example.com", "current_password": "correct horse battery" }
```

- `200` → `{ "data": User }`
- `400 validation_failed`: `fields.email`, or `fields.current_password` when it's missing
  or wrong. A wrong password is a 400 field error, never a 401, because clients treat
  401 as "session expired".
- `409 email_taken`
- `429 rate_limited`: shares the per-account limit with login, so the password can't be
  guessed here either. Also subject to the per-IP limit on auth routes.

## Operational endpoints (no envelope)

| Endpoint | Purpose |
| --- | --- |
| `GET /healthz` | Liveness. `200 {"status":"ok"}` whenever the process can serve. |
| `GET /readyz` | Readiness. `200 {"status":"ready"}`, or `503 {"status":"unavailable"}` if Postgres is unreachable. |
| `GET /metrics` | Prometheus metrics, served on a **separate listener** (`METRICS_ADDR`, default `127.0.0.1:9090`), never on the API port. |

## Proposed endpoints (implemented by the web mock, not yet by Go)

`apps/web/src/lib/api/mock/adapter.ts` answers these with the same envelopes
and status codes, and `apps/web/src/lib/api/types.ts` has the shapes. Every
resource is scoped to the caller's couple; a request for another couple's
resource is a 404, never a 403, so ids do not leak.

### Users and couples

| Endpoint | Notes |
| --- | --- |
| `DELETE /v1/users/me` | 204; removes the user from the couple and deletes their content |
| `GET /v1/couples/me` | `Couple`: flat `id`, `name`, `me`, `partner` (null until joined), `invite_code`, `started_on`, `onboarding` flags |
| `POST /v1/couples` | 201 `Couple`; creates the couple with the caller as first member and a fresh invite code |
| `POST /v1/couples/join` `{ code }` | 200 `Couple`, now with `partner` filled in; 400 `validation_failed` with `fields.code` when it does not match; 409 when the couple is full |
| `PATCH /v1/couples/me` `{ name?, relationship_start_date? }` | 200 `Couple`; dates are `YYYY-MM-DD` |
| `PATCH /v1/couples/role` `{ role }` | 200 `Couple`; sets the caller's own label, 1–32 characters |
| `PATCH /v1/couples/me/onboarding` | `{ couple?, install?, notifications? }`; 200 `Couple`. `install` and `notifications` are stored per person; `couple` is accepted but derived from membership, so it is always `true` in the response |

### Prayers

| Endpoint | Notes |
| --- | --- |
| `GET /v1/prayers/current` | `PrayerWeek` for this couple's current Sunday-to-Saturday week; the scheduler creates it, the API never does on read |
| `GET /v1/prayers/history` | past weeks, newest first |
| `GET /v1/prayers/weeks/:id` | |
| `PUT /v1/prayers/current/points` `{ points: PrayerPoint[] }` | setter only, draft only; ≤10 points, order = array order |
| `POST /v1/prayers/current/publish` | setter only; after this, points are read-only once the partner has completed any |
| `POST` / `DELETE /v1/prayers/points/:id/complete` | the caller's own completion; the other partner's is untouched |
| `PATCH /v1/prayers/weeks/:id/reflection` `{ reflection }` | the caller's reflection |

`PrayerWeek.status` is `draft` (setter still writing), `published`, or
`waiting` (the other partner sees this while the setter writes).

### Together

| Endpoint | Notes |
| --- | --- |
| `GET`, `POST /v1/events` · `GET`, `PATCH /v1/events/:id` | `Event` belongs to the couple |
| `POST` / `DELETE /v1/events/:id/complete` | |
| `PATCH /v1/events/:id/checklist/:item` `{ done }` | |
| `GET`, `POST /v1/goals` · `GET /v1/goals/:id` | `unit` is `naira` or `count` |
| `POST /v1/goals/:id/progress` `{ amount }` | one shared total; progress rows carry `user_id` for the log only |
| `GET /v1/challenges/current` · `PATCH /v1/challenges/current/days/:n` `{ done?, skipped? }` | |
| `GET`, `POST /v1/journal` | `{ tag, text }` |
| `GET`, `POST /v1/appreciations` · `DELETE /v1/appreciations/:id` | delete is the sender's undo, within a short window |
| `GET`, `POST /v1/memories` | photo upload is a later addition |
| `GET`, `POST /v1/milestones` | |

### Notifications

| Endpoint | Notes |
| --- | --- |
| `GET`, `PATCH /v1/notifications/preferences` | `NotificationPrefs`; `reminder_time` is `HH:MM` in the user's timezone |
| `POST /v1/notifications/subscribe` | the browser's `PushSubscription.toJSON()`; 204 |
