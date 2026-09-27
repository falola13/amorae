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
  birthday: null | { month: number; day: number; year: number | null }; // year is optional
  photo_url?: string; // signed, unguessable, versioned; absent when there is no photo
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
{ "display_name": "Ada L.", "timezone": "Africa/Lagos", "birthday": { "month": 9, "day": 30, "year": 1990 } }
```

- `200` → `{ "data": User }`
- `400 validation_failed`: `fields` may contain `display_name`, `timezone`, `birthday`
- `timezone` is optional (omit or `""` to leave it unchanged) and must be an IANA zone name.
- `birthday` follows the usual absent/null/value rule: omit the key to leave it unchanged, send
  `"birthday": null` to clear it, or send `{ "month", "day", "year"? }` to set it. `year` is
  optional — a birthday can be kept without saying which year. `month`/`day` must form a real
  calendar date (year 2000 stands in when no year is given, so Feb 29 is always allowed); a given
  `year` must be between 1900 and this year and the resulting date must not be in the future.
  Either problem is `fields.birthday`, worded `"That isn’t a date."` or `"That’s in the future."`.
- `email` is **not** accepted here (it's rejected as an unknown field): use the endpoint below.

### `POST /v1/users/me/photo/ticket` (auth)

A signature for uploading one photo **directly to Cloudinary**, so no image byte passes
through this API — same shape and reasoning as memories' photo ticket, scoped to the
caller's own profile picture instead of a memory.

- `200` → `{ "data": { "upload_url": string, "fields": { [key: string]: string } } }`.
  Post the file to `upload_url` as multipart with every pair in `fields` plus `file`,
  changing nothing, since the signature covers exactly those values. The server chose the
  asset's name, so the client cannot decide where a file lands.
- `400 photos_unavailable` when Cloudinary is not configured on this server
- `401 unauthenticated`

### `PUT /v1/users/me/photo` (auth)

Tell the profile that the upload finished. No body — the server already knows the only
name the file can have.

- `200` → `{ "data": User }`, now with `photo_url` set
- `400 photos_unavailable`
- `401 unauthenticated`

### `DELETE /v1/users/me/photo` (auth)

Take the photo off the profile and delete the file at Cloudinary. The deletion happens
first: if Cloudinary refuses, nothing here changes and the error says so, rather than
reporting a picture gone while it is still stored. Idempotent.

- `200` → `{ "data": User }`, `photo_url` now absent
- `400 photos_unavailable`
- `401 unauthenticated`

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

- `204`. The user, their sessions and their couple membership are deleted, their profile
  photo destroyed at Cloudinary with them (best-effort — a Cloudinary hiccup never blocks
  deleting the account). A partner keeps their account and the couple.
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
joined; `{ id, display_name, role, photo_url? }` — never their email), `invite_code` (only
while a usable code exists and the couple has one member), `started_on`, `timezone`,
`onboarding` flags. `me.photo_url` and `partner.photo_url` are generated the same way as
`User.photo_url`, so each of you sees the other's picture without either seeing anything
else about their account. A `Couple` is always a live one; a couple that has ended is a
different shape, `EndedCouple` (below).

An authenticated write may carry an **`Idempotency-Key`** header. The API stores the reply
against it, so the same key sent twice replays the first answer rather than doing the work again —
which is what lets the offline queue retry a create without risking a second row (FR-PWA-009).
Keys are per person, valid for seven days, and refused if reused on a different endpoint. A
replayed response carries `Idempotent-Replay: true`. Reads and requests without the header are
untouched.

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
| `PUT /v1/prayers/current/points` `{ points: PrayerPoint[] }` | Either partner (DEC-33) may edit or delete any point at any time, whether or not the other partner has already prayed it; the app confirms before sending a delete. ≤10 points; the order of the array is the order, and each point's `position` is ignored on the way in. Points are matched by `id` and kept, so reordering does not discard what has been prayed on them; a point left out is deleted, and its completions with it. An `id` the week does not already own is treated as a new point — the server chooses primary keys. `weekdays` is optional per point — see below |
| `POST /v1/prayers/current/publish` | Either partner (DEC-33). Publishing an already-published week is a no-op, and does not move `published_at` or `published_by`. Either partner may still edit after publishing — fixing a typo is not a betrayal — until the other partner prays any of it |
| `POST` / `DELETE /v1/prayers/points/:id/complete` | Marks, or unmarks, the caller's own prayer **for today** — a couple prays the week's points every day, not once and done (DEC-33). Today is the couple-local date; only a point actually scheduled for today, in the current week, can be marked, otherwise 409 `not_for_today`. 409 `prayer_not_shared` on a draft week. The other partner's progress is untouched |
| `PATCH /v1/prayers/weeks/:id/reflection` `{ reflection }` | the caller's reflection |
| `PUT` / `DELETE /v1/prayers/points/:id/answered` `{ note }` | Marks a prayer answered, or takes the mark back. Either partner may — a prayer belongs to them both, and the one who notices is not always the one who wrote it down. Deliberately **not** limited to the current week: prayers are answered months later, and an endpoint that only worked for seven days would miss most of what it exists to catch. `note` is optional; `PUT` twice is an edit of the note, not a second answer, and does not move `answered_at`. 409 `prayer_not_shared` on a draft |
| `GET /v1/prayers/answered` | Every answered prayer, newest answer first, each with the week it came from. Note what is absent: there is no way to ask for the prayers that were *not* answered, and no count of them — see FR-PRAY-012 |

`PrayerWeek.status` is `draft` (still being written) or `published`. A draft
is visible to both partners now, not just the setter — either may step in
and write or publish it (DEC-33); `setter_id` still names whose turn it is
(it decides `KindNewWeek` and reads as a label), it just no longer gates who
may act. `week_end` is `week_start` plus six days and is computed, never
stored. A point's text is `text` in JSON and `body` in the database; the DTO
is where those two names meet.

A point's `weekdays` is `number[]`, `0`=Sunday .. `6`=Saturday, sorted and
deduplicated; empty or absent means every day, so a point that runs all week
never has to say so. Stored as a bitmask (`prayer_points.weekdays`); which
weekday a date is is the couple-local weekday, and the week itself still
runs Sunday→Saturday (`StartOfWeek`).

`PrayerWeek.my_completed`/`partner_completed` mean different things
depending which week this is: for the **current** week, they are today's
completions only (couple-local), and reset the next day; for any **other**
week (history), they are every point completed on **any** day of that week.
`today` is `YYYY-MM-DD`, couple-local, and present only on the current week
— absent everywhere else, since a history week has no "today" of its own.
`days` is always seven entries, Sunday through Saturday, each
`{ date, points, mine, partner }`: `points` is which point ids were
scheduled that day, `mine`/`partner` which of those the caller/the other
partner actually prayed that day.

### Together

| Endpoint | Notes |
| --- | --- |
| `GET`, `POST /v1/events` · `GET`, `PATCH /v1/events/:id` | `Event` belongs to the couple; both partners always **see** every event (DEC-16). `kind` is `together` (the default — either partner may edit it, as before) or `mine` (its creator only — everyone else's write is a 404, same as another couple's event entirely). Only the creator may change `kind` on an existing event; anyone else sending one is 400 `validation_failed` with `fields.kind`, except on an event from before `kind` existed, which has no creator on record and so is open to either. `created_by` is the id of whoever made it, or `null` on anything that predates it. A PATCH field that is absent is left alone and `""` clears it; `checklist` is sent whole, so the order is the order and an item left out is removed. Another couple's event is 404 (DEC-19). `reminder` is free text on the wire, but the client offers a fixed set of phrases ("1 hour before", "the morning of", "the day before"…) and the worker reads that set into a send time in the couple's timezone (FR-NOTF-007). A phrase outside it is kept as written and simply never fires |
| `DELETE /v1/events/:id` | 204. A `together` event: either partner, removed for both, checklist included. A `mine` event: its creator only; anyone else's attempt is 404 |
| `POST` / `DELETE /v1/events/:id/complete` | Same ownership rule as `DELETE`: open to either partner on `together`, to the creator only on `mine` |
| `PATCH /v1/events/:id/checklist/:item` `{ done }` | Same ownership rule as `DELETE` |
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
| `PATCH /v1/journal/:id` `{ tag, text }` | 200 the updated entry, same shape as `POST`. Only the entry's author may edit it — the couple's partner gets 404, same as another couple's entry entirely (DEC-19). The date it was filed under never moves |
| `DELETE /v1/journal/:id` | 204. Only the entry's author may remove it; anyone else's attempt, including the partner's, is 404 |
| `GET`, `POST /v1/appreciations` · `DELETE /v1/appreciations/:id` | delete is the sender's undo: 204 within 30 seconds of sending; 403 `forbidden` for the partner's note; 409 `undo_window_closed` after the window. The web app offers Undo for 5 seconds and never queues it offline |
| `GET`, `POST /v1/memories` | A couple's kept moments, newest first, the same for both of them. No author, no reactions, no comments — an archive, not a feed. `location` and `note` are optional. `has_photo` is accepted on POST and ignored: the server decides whether a photo exists, and it becomes true only when a file has actually landed. `photo_url` appears on a memory that has one, carrying a version so that a replaced photo is served from a new address rather than out of a CDN's copy of the old one |
| `PUT /v1/memories/:id` `{ title, date, location, note }` | 200 the updated memory, same shape as `POST`. Either partner may edit it, since it belongs to them both — no author to check (DEC-16). The photo is untouched; use the photo endpoints for that |
| `POST /v1/memories/{id}/photo/ticket` | A signature for uploading one photo **directly to Cloudinary**, so no image byte passes through this API. Answers `{upload_url, fields}` — post the file to `upload_url` as multipart with every pair in `fields` plus `file`, changing nothing, since the signature covers exactly those values. The server chose the asset's name, so the client cannot decide where a file lands. `422 photos_unavailable` when Cloudinary is not configured |
| `PUT /v1/memories/{id}/photo` | Tell the memory the upload finished. No body — the server already knows the only name the file can have. This is the step that makes `has_photo` true, which is why a failed upload leaves a memory honest rather than showing a broken picture |
| `DELETE /v1/memories/{id}/photo` | Take the photo off a moment and delete the file at Cloudinary. The deletion happens first: if Cloudinary refuses, nothing here changes and the error says so, rather than reporting a picture gone while it is still stored. The moment itself stays. Idempotent |
| `DELETE /v1/memories/{id}` | Forget a moment entirely, its photo included and deleted with it. Idempotent — one already gone leaves the same nothing behind |
| _(no endpoint)_ `goal_crossing` | Halfway, and done. Both partners, once per crossing, keyed so that passing halfway, slipping back and passing it again says nothing the second time. Only the furthest point just passed — one contribution taking a goal from nothing to finished is one piece of news, not two. The amount is never in it (FR-NOTF-005.AC2). Gated by `goal_milestones`, which is on by default, unlike `goals` — that announces every contribution and is off for that reason |
| _(no endpoint)_ `event_over` | Once an event has finished, both partners (its creator only, on a `mine` event) are asked whether it is worth keeping, leading to the event where keeping it is already the first thing offered. An event with an end time is over then; with only a start it is taken to run two hours; with neither it is a whole day, and the question waits until the next morning rather than arriving at midnight. Asked once per event, within a day of it ending, and gated by `event_followups` — a switch of its own now, split off from `event_reminders`, since not wanting to be told beforehand says nothing about afterwards |
| _(no endpoint)_ `event_added` | When one partner creates a `together` event, the other partner — never the creator, and never on a `mine` event — is told once, named: "`<name>` added something for you both", with the event's own title as the body. Keyed on the event, so it cannot arrive twice. Gated by `partner_events` |
| `POST /v1/nudge` | One partner telling the other they are thinking of them: no body, nothing to reply to, and no record kept beyond the send itself. Sent immediately rather than on the next tick, because five minutes late is a different thought. Three a day; a sent one answers `{ "left": n }`, how many more today, so the limit is seen before it is met. Refused with `their_quiet_hours` or `their_day_is_full` rather than queued, so the sender is told they are asleep instead of the thought being silently dropped |
| `GET`, `POST /v1/milestones` | The dates a couple keeps: birthdays, anniversaries, the day they met — one entity, not one per kind (BR-DATE-01). `date` is the day it happened, never the next time it comes round; which year's occurrence is being looked at is worked out by whoever asks (the list screen, the reminder worker). Either partner may add one and it belongs to them both, so there is no author. `reminder` means "remind us every year" and defaults to true; a date with it off is kept but never announced. Listed oldest first — what counts as "coming up" depends on today, so the client decides it |
| `DELETE /v1/milestones/{id}` | Remove a date, and any yearly reminder with it. Either partner may, because it belongs to them both (DEC-16). Idempotent |

### Timeline

| Endpoint | Notes |
| --- | --- |
| `GET /v1/timeline` | The couple's shared story: one read-only feed of what happened, newest first, drawn from seven other modules' own tables rather than stored anywhere itself — there is nothing to create, edit, or delete here. Query params: `before` (RFC3339 instant, optional — pages strictly earlier than it; absent means the most recent page), `filter` (`all` · `prayer` · `moments` · `plans`; unknown or absent means `all`), `limit` (`1`–`50`, default `30`, clamped rather than rejected). Answers `{ items: TimelineItem[], next }`, `next` an RFC3339 instant to pass back as the next page's `before`, or `null` once there is nothing further back |

`prayer` is every **past** published prayer week (a week still open, or not yet
shared, does not appear) plus every answered prayer, across every week the
couple has ever had. `moments` is memories, journal entries, and
appreciations. `plans` is finished goals and events that are done or whose
day has passed — an event still ahead of the couple belongs on their
calendar, not their history.

A `TimelineItem` is `{ id, type, at, date, title, sub, path, photo_url?, actor_id? }`.
`type` is one of `prayer_week`, `prayer_answered`, `memory`, `event`, `goal`,
`journal`, `appreciation`. `at` is the instant the item sorts and pages by;
`date` is that instant's **couple-local** calendar day (`YYYY-MM-DD`), for
grouping by month on the client — computed here so a client never has to
reason about the couple's timezone itself (DEC-27). `path` is where tapping
the item goes. `photo_url` appears only on a `memory` that has a photo and
only when Cloudinary is configured, versioned the same way `GET /v1/memories`
already versions one. `actor_id` appears only when the item has a single
actor (who answered a prayer, wrote a journal entry, sent an appreciation,
created an event) — absent for a whole prayer week, a memory, and a finished
goal, none of which belong to one partner more than the other.

A finished goal's `at` is the last time it was touched (`updated_at`) — goals
carries no separate "completed at" column, so the moment it was marked done
stands in for when it happened. Paging is cursor-based on `at`, not an
offset, so couples who don't come back for a week don't see items shift
around when they do; ties (two items at the same instant) break the same way
on every page, by `type` then `id`.

### Notifications

| Endpoint | Notes |
| --- | --- |
| `GET`, `PATCH /v1/notifications/preferences` | `NotificationPrefs`. Every PATCH field is optional and a missing one is left alone, so the client can send one switch. `reminder_time` is `HH:MM` read in the **user's own** timezone, not the couple's (FR-NOTF-002); anything else is 400 `validation_failed` with `fields.reminder_time`. Reading does not create a row — somebody who never opens the screen gets the defaults and leaves no trace of having been asked. `prayer_answered` defaults **on** and controls FR-NOTF-008. `event_followups` gates `event_over` (split off from `event_reminders`, defaults **on**); `partner_events` gates `event_added` (defaults **on**). `default_event_reminder` is what the create-event screen offers as already chosen — one of `""` (no default), `"at the time"`, `"10 minutes before"`, `"30 minutes before"`, `"1 hour before"`, `"2 hours before"`, `"the morning of"`, `"1 day before"`; anything else is 400 `validation_failed` with `fields.default_event_reminder`. Defaults to `"1 hour before"`, matching what the screen always offered before this was a preference |
| `POST /v1/notifications/subscribe` | The browser's `PushSubscription.toJSON()`; 204. Keyed on `endpoint`, so re-subscribing the same browser replaces its keys rather than collecting a second row that would send everything twice. The endpoint must be `https://`. 400 `validation_failed` naming `endpoint`, `keys.p256dh` or `keys.auth` |
| `GET /v1/notifications/inbox` | `{ items: NotificationItem[], unread }`, newest first, at most 30 — the server keeps only that many per person (older rows are pruned on the next send). Each item is `{ id, kind, title, body, path, created_at, read }`. Written the moment a notification is decided to go out (claimed), whether or not a device was subscribed or the push itself failed; one still waiting on quiet hours or the daily cap is written only once it actually goes |
| `POST /v1/notifications/inbox/read` | Marks every one of the caller's rows read at once; 204. There is no per-row "mark this one read" — opening the list is what reading it means |

Preferences are per person, never per couple: partners choose their own, and one
of them turning something off says nothing about the other. Defaults are on,
except goals and challenges — following one of those is opted into rather than
something that starts buzzing on its own (FR-NOTF-006).

### Notification kinds

Every kind the worker (or a direct send, for `nudge`) can produce. "Immediate"
means the write that causes it also pokes the worker to run a pass right
away, rather than waiting for the next cron tick (`POST /internal/tick`,
every five minutes); a kind still waits out its own undo window or budget
rule regardless of when the worker next runs. "Scheduled" kinds are tied to a
moment in time (a reminder, a date) rather than to a write, so nothing pokes
them — the cron tick is the only thing that ever finds them due.

| Kind | Fires when | Recipient | Path | Preference | Timing |
| --- | --- | --- | --- | --- | --- |
| `new_week` | A new week is waiting to be set (draft, empty) | The setter, about their own week | `/prayers/set` | `new_week` | Scheduled |
| `week_published` | A week is published and has points in it | Whoever didn't publish it | `/prayers` | `new_week` (shared with `new_week` — publishing and setting are the same switch) | Immediate — poked by `prayers.Publish` |
| `prayer_reminder` | Something scheduled today is still unprayed, past the person's own reminder time | Self | `/prayers` | `prayer_reminder` | Scheduled (exempt from quiet hours/daily cap — `AskedFor`, it's a time they chose) |
| `event_reminder` | An event's reminder lead has arrived | Self (creator only on a `mine` event) | `/together/events/{id}` | `event_reminders` | Scheduled (exempt from quiet hours/daily cap, same reason) |
| `important_date` | A birthday, anniversary, or kept date is a week out or arrives today | The other partner(s) (never the birthday's own subject) | `/together/milestones` | `important_dates` | Scheduled |
| `appreciation` | A note settles past its 30s undo window | The other partner | `/together/appreciation` | `appreciation` | Immediate — poked by `appreciation.Send` (still waits out the undo window) |
| `journal` | A journal entry is written | The other partner | `/together/journal` | `journal` | Immediate — poked by `journal.Add` |
| `goal` | Progress is logged on a goal that isn't finished | The other partner | `/together/goals` | `goals` (**off** by default — opt-in) | Immediate — poked by `goals.LogProgress` |
| `goal_crossing` | A contribution just crossed 50% or 100% of a goal's target | Both partners | `/together/goals` | `goal_milestones` | Immediate — poked by `goals.LogProgress` (same write as `goal`) |
| `challenge` | A live challenge's day isn't marked yet, past the morning | Self | `/together/challenges` | `challenges` (**off** by default — opt-in) | Scheduled (perishable — a held one is simply dropped, not queued) |
| `prayer_answered` | A prayer is marked answered, past its 1-minute undo window | The other partner | `/prayers/answered` | `prayer_answered` | Immediate — poked by `prayers.SetAnswered` (only on marking answered, not on taking it back) |
| `both_prayed` | Both partners have prayed everything scheduled for today | Both partners | `/prayers` | `together` | Immediate — poked by `prayers.SetCompletion` (only on marking done, not on unmarking) |
| `both_marked` | Both partners have marked a challenge day | Both partners | `/together/challenges` | `together` | Immediate — poked by `challenges.Mark` (only on marking, not on clearing) |
| `memory_on_this_day` | A memory's date recurs today | Both partners (one notification per day, oldest memory named) | `/together/memories` | `memories` | Scheduled |
| `event_over` | An event has just finished | Both partners on a `together` event, creator only on `mine` | `/together/events/{id}` | `event_followups` | Scheduled |
| `event_added` | A `together` event is created | The other partner (never the creator, never for a `mine` event) | `/together/events/{id}` | `partner_events` | Immediate — poked by `events.Create` |
| `nudge` | A partner taps "thinking of you" | The other partner | `/` | *(none — see below)* | Immediate by nature; not part of a worker tick at all |

Two things worth flagging from this audit rather than fixing silently:

- **`nudge` has no preference toggle of its own.** It still respects the
  recipient's quiet hours and daily cap (checked directly in `Service.Nudge`,
  the same `Budget.Allows` rules the worker uses), and it has its own
  three-a-day ceiling on top — but there is no `notifications` field a
  person can switch off to stop nudges specifically, the way every other
  kind has one.
- **`goal` and `goal_crossing` both point at the goals list (`/together/goals`),
  not at the goal itself (`/together/goals/{id}`)**, unlike every event kind,
  which links straight to the event. Both routes exist; this is a narrower
  landing than it could be, not a broken one.
