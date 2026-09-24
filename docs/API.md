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
| `POST /v1/auth/password/forgot` | 10 requests per email per 15 minutes |
| Password and account changes that ask for the current password | 10 attempts per account per 15 minutes, shared between email change, password change and deletion |
| All `/v1/couples/*` endpoints | 60 requests per client IP per minute |
| `POST /v1/couples/join` | 10 attempts per account per 15 minutes |

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
{
  "email": "ada@example.com",
  "password": "correct horse",
  "display_name": "Ada",
  "age_confirmed": true,
  "accepted_terms": true,
  "faith_consent": false
}
```

- `201` → `{ "data": AuthResult }`
- `400 validation_failed`: `fields` may contain `email`, `password`, `display_name`,
  `age_confirmed`, `accepted_terms`

`age_confirmed` and `accepted_terms` must be `true`. Each sign-up records one consent
row per kind (`age_18`, `terms`, `privacy`, and `faith_content` only when
`faith_consent` is true), with the policy version taken from server config, never
from the request.
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

### `GET /v1/sessions` (auth)

Your own live sessions, newest first — for "where you're signed in". It carries
no token hash, no raw user agent and no IP: `device` is a coarse label the API
builds from the user agent ("Safari on iPhone"), and nothing here says anything
about your partner's devices.

```ts
interface SessionInfo {
  current: boolean;      // the session making this request
  device: string;        // "Chrome on Windows", or "Unknown device"
  created_at: string;
  last_used_at?: string; // absent until the session is used again after sign-in
  expires_at: string;
}
```

- `200` → `{ "data": SessionInfo[] }`. Expired sessions are left out.
- `401 unauthenticated`

`last_used_at` is written at most once an hour per session, so it is a
recognition aid, not an audit trail.

### `DELETE /v1/sessions/others` (auth)

Ends every other session for this account. The session making the request
survives: someone securing their account from the phone in their hand should
not be signed out by it.

- `200` → `{ "data": { "signed_out": 2 } }`
- `401 unauthenticated`

### `PUT /v1/users/me/password` (auth)

```json
{ "current_password": "correct horse", "new_password": "correct horse battery" }
```

- `204`. Every other session is ended; the one making the request survives.
- `400 validation_failed`: `fields.new_password` (same rule as sign-up), or
  `fields.current_password` when it is missing or wrong
- `429 rate_limited`

### `DELETE /v1/users/me` (auth)

```json
{ "confirm": "delete", "current_password": "correct horse" }
```

- `204`. The user, their sessions and their couple membership are deleted. A partner
  keeps their account and the couple.
- `400 invalid_json` when there is no body
- `400 validation_failed`: `fields.confirm` unless it is the word `delete` (any case,
  surrounding spaces ignored), or `fields.current_password` when it is missing or wrong
- `429 rate_limited`

### `POST /v1/auth/password/forgot`

```json
{ "email": "ada@example.com" }
```

- `204` whether or not the email has an account, so the answer never reveals which
  emails exist. For a known email, a link to `APP_URL/reset?token=…` is sent. It
  works once, for one hour.
- `429 rate_limited`

### `POST /v1/auth/password/reset`

```json
{ "token": "…", "new_password": "correct horse battery" }
```

- `204`. Every session for the account is ended, including any on this device.
- `400 validation_failed`: `fields.token` when the link is unknown, expired or used;
  `fields.new_password` for the password rule

### `GET /v1/users/me/export` (auth)

A JSON file (`Content-Disposition: attachment; filename="amorae-export.json"`) inside the
usual envelope:

```ts
{
  exported_at: string;
  user: User;
  consents: { kind: string; policy_version: string; created_at: string }[];
  couple: null | {
    id: string; name: string; started_on?: string; created_at: string;
    members: { display_name: string; role: string; you: boolean }[]; // never a partner's email
    invite_code: string;
  };
}
```

### Couples (auth)

`Couple` is flat: `id`, `name`, `me` (a `User` plus your own `role`), `partner` (null until
joined), `invite_code` (only while a usable code exists and the couple has one member),
`started_on`, `timezone`, `onboarding` flags. A `Couple` is always a live one; a couple that has ended is a
different shape, `EndedCouple` (below).

| Endpoint | Notes |
| --- | --- |
| `GET /v1/couples/me` | 200 `Couple`; 404 when you are not in a couple |
| `POST /v1/couples` | 201 `Couple`; creates the couple with the caller as first member and a fresh invite code. 409 `already_paired` when the caller is already in a couple |
| `POST /v1/couples/invite` | 200 `Couple` with a new code. The old pending code is revoked at once. 409 `couple_full` once both have joined |
| `POST /v1/couples/join` `{ code }` | 200 `Couple`, now with `partner` filled in. `code` is case- and dash-insensitive (`abc-123` = `ABC123`). 400 `validation_failed` with `fields.code` when it is empty; 400 `invite_invalid` (unknown, or your own code), `invite_expired`, `invite_used` or `invite_revoked`, each with the message also in `fields.code`; 409 `couple_full` when the couple already has two members; 409 `already_paired` when the caller is already in a couple; 429 after 10 attempts in 15 minutes |
| `PATCH /v1/couples/me` `{ name?, relationship_start_date?, timezone? }` | 200 `Couple`; dates are `YYYY-MM-DD`. `timezone` is the **couple's** — an IANA name deciding when the prayer week turns over, changeable by either partner and shared by both (DEC-27). Not the same as `me.timezone`, which is when that person's reminders fire. 400 `validation_failed` with `fields.timezone` for an unknown zone |
| `PATCH /v1/couples/role` `{ role }` | 200 `Couple`; sets the caller's own label, 1–32 characters |
| `PATCH /v1/couples/me/onboarding` | `{ couple?, install?, notifications? }`; 200 `Couple`. `install` and `notifications` are stored per person; `couple` is accepted but derived from membership, so it is always `true` in the response |
| `DELETE /v1/couples/me` | 200 `EndedCouple[]`. Leaving **ends the couple for both partners** — it does not remove one of them and leave the other holding the shared history (FR-PAIR-008). Afterwards neither is in a couple. 404 `couple_not_found` when you are in none |
| `GET /v1/couples/archived` | 200 `EndedCouple[]` — the couples you used to be in whose 30-day window is still open. Usually empty |

#### After a couple has ended

`EndedCouple` is `id`, `name`, `people` (both members, display name and role — no emails),
`started_on`, `dissolved_at` and `read_only_until`. There is nothing to act on: no invite code,
no onboarding, no writes.

Leaving ends both memberships and marks the couple. For 30 days it stays readable through
`GET /v1/couples/archived` and is included in `GET /v1/users/me/export` under `ended_couples`;
then the couple row is deleted, taking memberships, invitations and every couple-owned table
with it. Each person's own account and profile are untouched.

| After leaving | |
| --- | --- |
| `GET /v1/couples/me` | 404 `couple_not_found` — you are in no couple |
| `POST /v1/couples`, `POST /v1/couples/join` | Work normally. Ending a space does not cost you the app (Q-24): you can start a new one immediately, and the old one stays readable alongside it |
| `GET /v1/couples/archived` | The ended couple, until `read_only_until` |
| `GET /v1/users/me/export` | Includes it under `ended_couples`, with `dissolved_at` and `read_only_until` |
| Any invite code for it | Revoked at the moment it ends, and refused even if a row somehow stayed pending |
| After `read_only_until` | Gone from both the archive and the export |

Read access is bounded by the window in the query itself, not by the sweep having run, so a
sweeper that stops cannot turn 30 days into forever. The sweep is hourly, so rows can outlive
`read_only_until` by up to an hour before they are deleted.

## Operational endpoints (no envelope)

| Endpoint | Purpose |
| --- | --- |
| `GET /healthz` | Liveness. `200 {"status":"ok"}` whenever the process can serve. |
| `GET /readyz` | Readiness. `200 {"status":"ready"}`, or `503 {"status":"unavailable"}` if Postgres is unreachable. |
| `GET /metrics` | Prometheus metrics, served on a **separate listener** (`METRICS_ADDR`, default `127.0.0.1:9090`), never on the API port. |

## Planned endpoints (not built yet)

Nothing serves these yet. They are the contract the screens were built
against, and `apps/web/src/lib/api/types.ts` has the shapes. Until a Go module
lands, the router answers with a bare 404 and the screen shows "Not available
yet" rather than an error. Treat the shapes as a starting point, not a
commitment: when a module is built differently, change it here and in
`types.ts` in the same pull request. Every resource is scoped to the caller's
couple; a request for another couple's resource is a 404, never a 403, so ids
do not leak.

### Prayers

| Endpoint | Notes |
| --- | --- |
| `GET /v1/prayers/current` | `PrayerWeek` for this couple's current week, which runs Sunday to Saturday **in the couple's timezone** (DEC-27). The week is created on first read if it does not exist — safe because `UNIQUE (couple_id, week_start)` makes a second creator lose harmlessly, which is the same property that will let the scheduler pre-warm it later. 409 `waiting_for_partner` while a couple has only one member: a week needs two people to have a setter |
| `GET /v1/prayers/history` | past weeks, newest first |
| `GET /v1/prayers/weeks/:id` | |
| `PUT /v1/prayers/current/points` `{ points: PrayerPoint[] }` | Setter only; ≤10 points; the order of the array is the order, and each point's `position` is ignored on the way in. Points are matched by `id` and kept, so reordering does not discard what has been prayed on them; a point left out is deleted, and its completions with it. An `id` the week does not already own is treated as a new point — the server chooses primary keys. 403 `not_this_weeks_setter`; 409 `prayer_in_use` when the submitted list rewords or drops a point the partner has already prayed — adding, editing a point nobody has prayed, and reordering stay open for the whole week (DEC-31) |
| `POST /v1/prayers/current/publish` | Setter only. Publishing an already-published week is a no-op, and does not move `published_at`. The setter may still edit after publishing — fixing a typo is not a betrayal — until the partner prays any of it |
| `POST` / `DELETE /v1/prayers/points/:id/complete` | the caller's own completion; the other partner's is untouched |
| `PATCH /v1/prayers/weeks/:id/reflection` `{ reflection }` | the caller's reflection |

`PrayerWeek.status` is `draft` (setter still writing), `published`, or
`waiting` (the other partner sees this while the setter writes). A `waiting`
week carries no points and no progress at all — not the setter's either, since
reporting what they had prayed would describe a week you are not allowed to
read. `week_end` is `week_start` plus six days and is computed, never stored.
A point's text is `text` in JSON and `body` in the database; the DTO is where
those two names meet.

### Together

| Endpoint | Notes |
| --- | --- |
| `GET`, `POST /v1/events` · `GET`, `PATCH /v1/events/:id` | `Event` belongs to the couple; both partners are implicit participants (DEC-16) and either may edit. A PATCH field that is absent is left alone and `""` clears it; `checklist` is sent whole, so the order is the order and an item left out is removed. Another couple's event is 404 (DEC-19). `reminder` is free text on the wire, but the client offers a fixed set of phrases ("1 hour before", "the morning of", "the day before"…) and the worker reads that set into a send time in the couple's timezone (FR-NOTF-007). A phrase outside it is kept as written and simply never fires |
| `DELETE /v1/events/:id` | 204. Removes it for both partners, checklist included |
| `POST` / `DELETE /v1/events/:id/complete` | |
| `PATCH /v1/events/:id/checklist/:item` `{ done }` | |
| `GET`, `POST /v1/goals` · `GET /v1/goals/:id` | `unit` is `naira` or `count`; amounts are whole units, never a float. A goal has **no** running total field: it is the sum of `progress[].amount`, so a client that adds it up can never disagree with one that was told (BR-GOAL-01). Another couple's goal is 404 (DEC-19) |
| `PATCH /v1/goals/:id` | 200 `Goal`. Either partner may edit any field or set `done`; an absent field is left alone |
| `POST /v1/goals/:id/progress` `{ amount }` | Appends an entry carrying the caller's `user_id` and today's date. A negative amount is allowed — a correction is how a wrong number is fixed, and deleting the entry would lose that it happened. Zero is 400 |
| `GET /v1/challenges/templates` | The curated catalogue: `key`, `title`, `blurb`, `days` |
| `POST /v1/challenges` `{ template }` | 201 `Challenge`. One at a time: 409 `challenge_already_running`, 400 `challenge_unknown` |
| `GET /v1/challenges/current` | `Challenge`, or 404 `challenge_not_found` — which is the answer for a couple who have not started one, not a failure |
| `DELETE /v1/challenges/current` | 204. Leaves it, freeing them to start another |
| `PATCH /v1/challenges/current/days/:n` `{ done?, skipped? }` | 200 `Challenge`. Marks the day **for the caller only** (DEC-30); `done` and `skipped` in the response are theirs, `partner_done` and `partner_skipped` are the other's. `false` un-marks it, which is not the same as skipping; sending neither is 400 |
| `POST /internal/tick` | Not part of the product API: one pass of the notification worker, for deployments with nowhere to run a long-lived process (`docs/DEPLOYMENT.md`). Outside `/v1`, outside the envelope, and absent entirely unless `TICK_SECRET` is set. Takes the secret as a bearer token and answers 404 to anything else, so a caller without it learns nothing. Answers `{"status","sent"}`; a pass already running answers `{"status":"already running"}` rather than starting a second. Safe to expose: every send is claimed by a row first, so a thousand calls still send each notification once |
| `GET`, `POST /v1/journal` | `{ tag, text }`. The couple's shared notebook, newest first, the same for both. `tag` is one of `Gratitude`, `Reflection`, `Memory`, `Appreciation`, `Plans`, matched exactly. The author is the session and the date is the couple's local day — neither is taken from the request |
| `GET`, `POST /v1/appreciations` · `DELETE /v1/appreciations/:id` | delete is the sender's undo: 204 within 30 seconds of sending; 403 `forbidden` for the partner's note; 409 `undo_window_closed` after the window. The web app offers Undo for 5 seconds and never queues it offline |
| `GET`, `POST /v1/memories` | A couple's kept moments, newest first, the same for both of them. No author, no reactions, no comments — an archive, not a feed. `location` and `note` are optional. `has_photo` is accepted on POST and ignored: the server decides whether a photo exists, and it becomes true only when a file has actually landed. `photo_url` appears on a memory that has one |
| `POST /v1/memories/{id}/photo/ticket` | A signature for uploading one photo **directly to Cloudinary**, so no image byte passes through this API. Answers `{upload_url, fields}` — post the file to `upload_url` as multipart with every pair in `fields` plus `file`, changing nothing, since the signature covers exactly those values. The server chose the asset's name, so the client cannot decide where a file lands. `422 photos_unavailable` when Cloudinary is not configured |
| `PUT /v1/memories/{id}/photo` | Tell the memory the upload finished. No body — the server already knows the only name the file can have. This is the step that makes `has_photo` true, which is why a failed upload leaves a memory honest rather than showing a broken picture |
| `DELETE /v1/memories/{id}/photo` | Forget the photo. Idempotent |
| `GET`, `POST /v1/milestones` | The dates a couple keeps: birthdays, anniversaries, the day they met — one entity, not one per kind (BR-DATE-01). `date` is the day it happened, never the next time it comes round; which year's occurrence is being looked at is worked out by whoever asks (the list screen, the reminder worker). Either partner may add one and it belongs to them both, so there is no author. `reminder` means "remind us every year" and defaults to true; a date with it off is kept but never announced. Listed oldest first — what counts as "coming up" depends on today, so the client decides it |

### Notifications

| Endpoint | Notes |
| --- | --- |
| `GET`, `PATCH /v1/notifications/preferences` | `NotificationPrefs`. Every PATCH field is optional and a missing one is left alone, so the client can send one switch. `reminder_time` is `HH:MM` read in the **user's own** timezone, not the couple's (FR-NOTF-002); anything else is 400 `validation_failed` with `fields.reminder_time`. Reading does not create a row — somebody who never opens the screen gets the defaults and leaves no trace of having been asked |
| `POST /v1/notifications/subscribe` | The browser's `PushSubscription.toJSON()`; 204. Keyed on `endpoint`, so re-subscribing the same browser replaces its keys rather than collecting a second row that would send everything twice. The endpoint must be `https://`. 400 `validation_failed` naming `endpoint`, `keys.p256dh` or `keys.auth` |

Preferences are per person, never per couple: partners choose their own, and one
of them turning something off says nothing about the other. Defaults are on,
except goals and challenges — following one of those is opted into rather than
something that starts buzzing on its own (FR-NOTF-006).
