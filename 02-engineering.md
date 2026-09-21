# Amorae: Engineering Specification

**Version:** 2.0 | **Status:** MVP definition | **Owner:** Engineering | **Last updated:** 2026-09-21

Read [`01-foundation.md`](./01-foundation.md) first for product context. This document covers functional requirements, data model, API, backend architecture, the weekly prayer scheduler, security, the PWA layer, AI, and non-functional targets.

---

## Context in one screen

- Amorae is a private PWA for two people. Every resource belongs to a **couple** and is visible only to its two members.
- Five pillars: Connect, Plan, Grow, Faith, Remember.
- Non-negotiables that shape architecture: private by default, works fully without AI, works on weak networks, deterministic prayer alternation (never AI-decided).

**Stack:** Next.js + TypeScript + Tailwind (frontend), Go + PostgreSQL (backend), Web Push (notifications), optional AI behind an interface.

**Requirement language:** Must (required for MVP), Should (expected, can slip), Later (out of MVP).

---

## 1. Architecture overview

```text
[ PWA: Next.js + Service Worker ]
            |  HTTPS, cookie auth
            v
[ Go API: handler -> service -> repository ]
            |
     +------+-------------------+
     |                          |
[ PostgreSQL ]         [ AI provider (optional) ]
     |
[ Background worker: weekly scheduler, push sender ]
```

Principles:

- **Stateless API.** No server-side session store required beyond the DB, so the API scales horizontally.
- **Layered backend.** Handlers do transport only. Business rules live in services. Data access lives in repositories. No SQL in handlers.
- **Couple-scoped everything.** `couple_id` is derived from the authenticated user, never trusted from the client.
- **Modular domains.** Each feature is its own module: `auth`, `couples`, `prayers`, `events`, `goals`, `challenges`, `journal`, `memories`, `notifications`, `ai`.

---

## 2. Functional requirements

Each feature lists what must ship and the acceptance criteria that define "done." Vague "potential features" from the source doc have been resolved into MVP or Later.

### 2.1 Authentication

**Must:**

- Sign up with email and password, login, logout, password reset, session persistence.
- Passwords hashed with argon2id.
- Session carried in an HTTP-only, Secure, SameSite=Lax cookie.

**Acceptance criteria:**

- A new user can register, receive a session, and stay logged in across app restarts until the session expires.
- Password reset invalidates old sessions for that user.
- Logout clears the session cookie and server-side session validity.
- Invalid credentials return a generic error (no account-existence leak).

### 2.2 Couple pairing

**Must:**

- A user creates a couple, becomes its first member, and can generate an invite (code and shareable link).
- A second user joins with a valid invite. Maximum two members.
- Couple profile: partner display names, avatars, pairing status.
- One couple per user in MVP.

**Later:** onboarding questionnaire, relationship start date capture during onboarding, shared preferences.

**Acceptance criteria:**

- Creating a couple returns a couple id and a pending invite.
- A valid, unexpired, unused invite lets exactly one second user join. A third join attempt is rejected.
- An expired, revoked, or already-used invite is rejected with a clear reason.
- After pairing, both members can read the couple profile. Neither can access another couple's data.

### 2.3 Weekly prayer

The signature Faith feature. See Section 6 for the scheduler algorithm.

**Must:**

- Responsibility for setting prayer points alternates weekly, beginning Sunday, decided deterministically by the backend. AI never decides whose turn it is.
- The setter can add, edit, delete, reorder, save as draft, and publish prayer points. Limit: 10 points per week.
- Once published, the partner can view the points.
- Each partner independently marks each point as prayed. Completion is per partner, not shared.
- Incomplete prayer is never framed as failure.

**Acceptance criteria:**

- For a paired couple, exactly one member is the setter each week, alternating each week.
- Only the current setter can create or edit that week's points. The non-setter gets a read-only view after publish and no view before publish.
- A published week is visible to both. Draft weeks are visible only to the setter.
- Marking a point prayed is idempotent per user (marking twice does not create duplicates).
- The eleventh prayer point in a week is rejected.

### 2.4 Shared events

**Must:** create, read, update, delete events. Fields: title, description, start time, optional end time, location, status, optional checklist, optional recap note. Events belong to the couple, not one person.

**Acceptance criteria:** both partners see the same event list. Either partner can create or edit. An event can be marked completed. Reminders can be attached (see 2.9).

### 2.5 Couple calendar

**Must:** a simple, mobile-friendly chronological view of events and the current prayer week. Read-focused. No enterprise calendar complexity (no drag-resize, no overlapping-event grids).

**Acceptance criteria:** upcoming events and the active prayer week appear in date order. Tapping an entry opens its detail.

### 2.6 Shared goals

**Must:** create goals owned by both partners. Fields: title, description, start date, end date, target value and unit, current value, status, optional checklist. Progress can be logged.

**Should not:** turn into a heavily gamified productivity product.

**Acceptance criteria:** both partners see and edit a goal. Logging progress updates the goal's current value. A goal can be marked completed or archived.

### 2.7 Couple challenges

**Must:** short guided multi-day experiences (for example a 7-day connection challenge). Each day has a prompt. Each partner marks a day done independently. Challenges are short, optional, and low-pressure.

**Acceptance criteria:** a couple can start a challenge, see each day's prompt, and mark days complete per partner. A challenge can be completed or abandoned without penalty language.

### 2.8 Connect and Remember (journal, appreciation, memories, milestones)

**Must:**

- **Journal:** shared private entries with a type (gratitude, reflection, memory, appreciation, plan, prayer reflection), body, optional photo, optional related event, optional tag.
- **Appreciation:** one partner writes a short note to the other. The recipient gets a subtle notification. Personal, not social.
- **Memories:** a private timeline of moments with title, caption, optional photo, date, optional location, optional related event.
- **Important dates and milestones:** birthdays, anniversary, relationship start, engagement, wedding, custom dates, and personal milestones, with optional reminders.

**Acceptance criteria:** entries are couple-scoped and visible to both. Appreciation notifies only the recipient. Memories read as a quiet archive, not a feed (design detail in `03-design.md`).

### 2.9 Notifications

**Must:** web push for: new prayer week, prayer reminder, upcoming event, event reminder, partner appreciation, journal activity, goal updates, challenge reminders, important date reminders. Users can subscribe, unsubscribe, and set per-category preferences.

**Acceptance criteria:** push is requested only after explaining value. A user can disable any category. No category fires more than its defined cadence (no spam).

### 2.10 AI (optional)

See Section 8. Every AI feature is optional, disable-able, and requires user review before anything is saved or published.

---

## 3. Data model

PostgreSQL. All tables use a UUID primary key (`id`), `created_at timestamptz not null default now()`, and, where mutable, `updated_at timestamptz`. Foreign keys are `not null` unless stated. Every couple-owned table has an index on `couple_id`.

### 3.1 Enums

```text
invitation_status:   pending | accepted | revoked | expired
prayer_week_status:  draft | published
event_status:        scheduled | completed | cancelled
goal_status:         active | completed | archived
challenge_status:    active | completed | abandoned
journal_type:        gratitude | reflection | memory | appreciation | plan | prayer_reflection
important_date_type: birthday | anniversary | relationship_start | engagement | wedding | milestone | custom
notification_type:   new_prayer_week | prayer_reminder | event_reminder | appreciation
                     | journal_activity | goal_update | challenge_reminder | important_date
```

### 3.2 Identity and pairing

**users**

| Column | Type | Notes |
|--------|------|-------|
| id | uuid | PK |
| email | citext | unique, not null |
| password_hash | text | not null (argon2id) |
| display_name | text | not null |
| avatar_url | text | null |
| timezone | text | IANA, default 'UTC' |
| created_at, updated_at, last_login_at | timestamptz | last_login_at null |

**couples**

| Column | Type | Notes |
|--------|------|-------|
| id | uuid | PK |
| name | text | null |
| timezone | text | IANA, not null. Drives the prayer week boundary. |
| relationship_start_date | date | null |
| created_by | uuid | FK users |
| created_at, updated_at | timestamptz | |

**couple_members**

| Column | Type | Notes |
|--------|------|-------|
| id | uuid | PK |
| couple_id | uuid | FK couples |
| user_id | uuid | FK users |
| role | text | default 'partner' |
| joined_at | timestamptz | not null. Member order (index 0 / 1) is by joined_at ascending. |

Constraints: `unique(couple_id, user_id)`; `unique(user_id)` for MVP (one couple per user); application enforces max two members per couple.

**couple_invitations**

| Column | Type | Notes |
|--------|------|-------|
| id | uuid | PK |
| couple_id | uuid | FK couples |
| code | text | unique, not null. Also used to build the invite link. |
| status | invitation_status | default 'pending' |
| created_by | uuid | FK users |
| expires_at | timestamptz | not null |
| accepted_by | uuid | FK users, null |
| accepted_at | timestamptz | null |
| created_at | timestamptz | |

### 3.3 Faith

**prayer_weeks**

| Column | Type | Notes |
|--------|------|-------|
| id | uuid | PK |
| couple_id | uuid | FK couples |
| week_start | date | Local Sunday in couple timezone |
| setter_user_id | uuid | FK users. Stored at creation so it is stable. |
| status | prayer_week_status | default 'draft' |
| published_at | timestamptz | null |
| created_at, updated_at | timestamptz | |

Constraint: `unique(couple_id, week_start)`. This makes scheduler retries safe.

**prayer_points**

| Column | Type | Notes |
|--------|------|-------|
| id | uuid | PK |
| prayer_week_id | uuid | FK prayer_weeks |
| position | int | ordering |
| text | text | not null |
| created_at, updated_at | timestamptz | |

Application enforces max 10 points per week.

**prayer_completions**

| Column | Type | Notes |
|--------|------|-------|
| id | uuid | PK |
| prayer_point_id | uuid | FK prayer_points |
| user_id | uuid | FK users |
| completed_at | timestamptz | |

Constraint: `unique(prayer_point_id, user_id)`. Makes completion idempotent.

### 3.4 Plan

**events**

| Column | Type | Notes |
|--------|------|-------|
| id | uuid | PK |
| couple_id | uuid | FK couples |
| title | text | not null |
| description | text | null |
| starts_at | timestamptz | not null |
| ends_at | timestamptz | null |
| location | text | null |
| status | event_status | default 'scheduled' |
| recap_note | text | null |
| created_by | uuid | FK users |
| created_at, updated_at | timestamptz | |

Index: `(couple_id, starts_at)`.

**event_reminders**

| Column | Type | Notes |
|--------|------|-------|
| id | uuid | PK |
| event_id | uuid | FK events |
| remind_at | timestamptz | not null |
| sent_at | timestamptz | null |

**event_checklist_items**

| Column | Type | Notes |
|--------|------|-------|
| id | uuid | PK |
| event_id | uuid | FK events |
| text | text | not null |
| is_done | bool | default false |
| position | int | |

### 3.5 Grow

**shared_goals**

| Column | Type | Notes |
|--------|------|-------|
| id | uuid | PK |
| couple_id | uuid | FK couples |
| title | text | not null |
| description | text | null |
| start_date, end_date | date | null |
| target_value | numeric | null |
| target_unit | text | null (for example "NGN", "books", "sessions") |
| current_value | numeric | default 0 |
| status | goal_status | default 'active' |
| created_by | uuid | FK users |
| created_at, updated_at | timestamptz | |

**goal_items** (optional checklist): id, goal_id (FK), text, is_done bool, position.

**goal_progress** (log): id, goal_id (FK), user_id (FK), value numeric (increment), note text null, created_at. `current_value` is maintained by summing increments in the service layer.

**challenges**: id, couple_id (FK), template_key text null, title, description null, status challenge_status default 'active', started_at, created_at.

**challenge_days**: id, challenge_id (FK), day_number int, prompt text.

**challenge_progress**: id, challenge_day_id (FK), user_id (FK), completed_at. Constraint: `unique(challenge_day_id, user_id)`.

### 3.6 Connect and Remember

**journal_entries**: id, couple_id (FK), author_id (FK users), type journal_type, body text, photo_url null, related_event_id (FK events) null, tag text null, created_at, updated_at.

**appreciations**: id, couple_id (FK), author_id (FK users), recipient_id (FK users), body text, read_at timestamptz null, created_at.

**memories**: id, couple_id (FK), title null, caption null, photo_url null, occurred_on date, location text null, related_event_id (FK events) null, created_by (FK users), created_at, updated_at.

**important_dates**: id, couple_id (FK), title, type important_date_type, date date, is_recurring bool default false, remind bool default false, created_by (FK users), created_at. Milestones and important dates share this table via `type`.

### 3.7 Platform

**push_subscriptions**: id, user_id (FK), endpoint text unique, p256dh text, auth text, user_agent text null, created_at, last_used_at null.

**notification_preferences**: id, user_id (FK) unique, prefs jsonb (a map of notification_type to bool, defaults all true), updated_at.

### 3.8 Relationships

```text
User
 └── Couple Membership (unique per user in MVP)
       └── Couple
            ├── Invitations
            ├── Prayer Weeks ── Prayer Points ── Completions
            ├── Events ── Reminders / Checklist Items
            ├── Goals ── Items / Progress
            ├── Challenges ── Days ── Progress
            ├── Journal Entries
            ├── Appreciations
            ├── Memories
            └── Important Dates
```

Every couple-owned resource must be authorization-scoped to the couple. There is no code path where a user reads or writes another couple's data.

---

## 4. API design

### 4.1 Conventions

- **Base path:** `/api/v1`.
- **Auth:** session cookie (HTTP-only, Secure, SameSite=Lax). No token in the client body.
- **Couple scoping:** the server derives `couple_id` from the authenticated user's membership. The client never sends `couple_id`. A resource id that belongs to another couple returns 404 (not 403), to avoid confirming its existence.
- **Timestamps:** ISO 8601 UTC everywhere.
- **Errors:** consistent envelope.

  ```json
  { "error": { "code": "prayer_not_setter", "message": "Only this week's setter can edit prayer points." } }
  ```

- **Status codes:** 200 ok, 201 created, 204 no content, 400 validation, 401 unauthenticated, 403 forbidden, 404 not found or not yours, 409 conflict, 429 rate limited.
- **Pagination:** cursor-based for lists that grow over time (memories, journal, prayer history): `?limit=20&cursor=...`, response `{ "data": [...], "next_cursor": "..." | null }`.
- **Validation:** server-side always. The frontend validates with Zod for UX, but the server never trusts the client.

### 4.2 Endpoints

```http
# Auth
POST   /auth/register
POST   /auth/login
POST   /auth/logout
POST   /auth/refresh
POST   /auth/forgot-password
POST   /auth/reset-password

# Couples
POST   /couples
POST   /couples/invite
POST   /couples/join
GET    /couples/me

# Prayer
GET    /prayers/current
GET    /prayers/weeks/:id
POST   /prayers/weeks/:id/publish
POST   /prayers/points
PATCH  /prayers/points/:id
DELETE /prayers/points/:id
POST   /prayers/points/:id/complete
POST   /prayers/points/reorder
GET    /prayers/history

# Events
GET    /events
POST   /events
GET    /events/:id
PATCH  /events/:id
DELETE /events/:id
POST   /events/:id/complete

# Goals
GET    /goals
POST   /goals
GET    /goals/:id
PATCH  /goals/:id
DELETE /goals/:id
POST   /goals/:id/progress

# Challenges
GET    /challenges
POST   /challenges
POST   /challenges/days/:id/complete

# Connect and Remember
GET    /journal            POST /journal
GET    /appreciations      POST /appreciations
GET    /memories           POST /memories   PATCH /memories/:id   DELETE /memories/:id
GET    /important-dates    POST /important-dates   PATCH /important-dates/:id   DELETE /important-dates/:id

# Notifications
POST   /notifications/subscribe
DELETE /notifications/subscribe
PATCH  /notifications/preferences

# AI (optional, feature-flagged)
POST   /ai/prayer-points
POST   /ai/improve-prayer
POST   /ai/reflection
```

### 4.3 Representative payloads

The tricky endpoints, spelled out. The rest follow the same shape.

**Create couple** `POST /couples`

```json
// response 201
{ "couple": { "id": "uuid", "name": null, "timezone": "Africa/Lagos", "status": "waiting_for_partner" },
  "invite": { "code": "AB12-CD34", "link": "https://amorae.app/join/AB12-CD34", "expires_at": "..." } }
```

**Join couple** `POST /couples/join`

```json
// request
{ "code": "AB12-CD34" }
// response 200 on success, 409 if the couple already has two members, 400 if invite invalid or expired
```

**Current prayer week** `GET /prayers/current`

```json
// response 200
{ "week": { "id": "uuid", "week_start": "2026-09-27", "status": "published",
            "setter": { "user_id": "uuid", "display_name": "Femi", "is_me": true } },
  "points": [
    { "id": "uuid", "position": 1, "text": "For wisdom in the decisions ahead.",
      "completions": { "me": true, "partner": false } }
  ] }
```

**Publish week** `POST /prayers/weeks/:id/publish` is idempotent. Publishing an already-published week returns 200 with no change. Only the setter can publish.

**Complete a prayer point** `POST /prayers/points/:id/complete` is idempotent per user (backed by the unique completion constraint).

**Log goal progress** `POST /goals/:id/progress`

```json
// request
{ "value": 50000, "note": "Moved this month's savings in." }
// response 200
{ "goal": { "id": "uuid", "current_value": 250000, "target_value": 500000, "status": "active" } }
```

---

## 5. Backend architecture

```text
Handler (transport, auth, validation)
   -> Service (business rules, couple scoping, cross-entity logic)
      -> Repository (parameterized SQL)
         -> PostgreSQL
```

Modules: `auth`, `couples`, `prayers`, `events`, `goals`, `challenges`, `journal`, `memories`, `notifications`, `ai`. Domain rules live in services, not handlers. A background worker handles the weekly scheduler and push delivery.

Suggested libraries: chi or echo (router), pgx with sqlc (typed queries), golang-migrate (migrations), argon2id (hashing), web-push (VAPID push), testify (tests).

---

## 6. Weekly prayer scheduler

This is the most logic-heavy part. It must be deterministic and idempotent. AI is never involved.

### 6.1 Whose turn it is

- Member order is fixed: index 0 and index 1 by `joined_at` ascending.
- For a given `week_start`, compute `week_index` = whole weeks between the couple's first prayer week and this week.
- `setter = members[week_index % 2]`. The result is stored in `prayer_weeks.setter_user_id` at creation, so the assignment is stable even if membership data changes later.

```text
Week 1 -> member[0]
Week 2 -> member[1]
Week 3 -> member[0]
Week 4 -> member[1]
```

### 6.2 Week boundary and timezone

- The week starts Sunday 00:00 in the couple's timezone (`couples.timezone`).
- `week_start` is stored as the local Sunday date.
- Because couples span timezones, the worker ticks hourly and creates any missing current week per couple, rather than firing once at a single global midnight.

### 6.3 Idempotency

- `unique(couple_id, week_start)` guarantees one week per couple per week.
- Creation uses `INSERT ... ON CONFLICT (couple_id, week_start) DO NOTHING`, so retries and overlapping worker runs are safe.
- After a new week is created, send the `new_prayer_week` push to both partners.

### 6.4 Steps per tick

1. Find couples whose local time is now on or past Sunday 00:00 and who have no week row for the current local week.
2. Compute `week_start` and `setter_user_id`.
3. Insert with `ON CONFLICT DO NOTHING`.
4. If a row was inserted, enqueue the push notification.

---

## 7. Security

- **Passwords:** argon2id. Never logged.
- **Sessions:** HTTP-only, Secure, SameSite=Lax cookie. Rotate on login. Enforce both idle and absolute expiry. Logout invalidates server-side.
- **CSRF:** SameSite=Lax plus a CSRF token on state-changing requests where the cookie could be sent cross-site.
- **Couple-level authorization:** middleware loads the caller's membership and injects `couple_id`. Every repository query filters by it. Cross-couple ids return 404.
- **Rate limiting:** auth endpoints 10 per minute per IP, general API 100 per minute per user, AI per the daily caps in Section 8.
- **Input validation and SQL safety:** validate server-side, use parameterized queries (sqlc/pgx). No string-built SQL.
- **Sensitive data:** journal, prayer, appreciation, and memory content are treated as private. Encrypt at rest at the database or disk level. Do not log bodies. Redact content from error reports.
- **Push:** store subscriptions securely, remove dead subscriptions on 410 responses.
- **Secrets:** VAPID keys, DB credentials, and any AI keys come from environment or a secret manager, never source.
- **Audit:** log security-sensitive actions (login, invite created, join, resource deletion) with actor, action, and timestamp. No content bodies.

---

## 8. AI layer (optional)

The product must work completely with AI disabled. AI is a feature flag, not a dependency.

**Features (all opt-in, all require user review before save or publish):**

- **Prayer point assistant:** user describes a situation, AI suggests draft points.
- **Improve my prayer:** AI refines the wording of a rough prayer.
- **Weekly reflection:** AI summarizes user-provided reflections.
- **Scripture suggestions:** AI may suggest themes, but references must be verified before display.

**Hard rules. AI must never:** claim divine authority, promise prayer outcomes, manipulate emotionally, invent Bible references, auto-publish generated content, or replace user judgment.

**Interface (Go), so the provider is swappable:**

```go
type AIService interface {
    GeneratePrayerPoints(ctx context.Context, input string) ([]string, error)
    ImprovePrayer(ctx context.Context, input string) (string, error)
    GenerateReflection(ctx context.Context, input PrayerReflectionInput) (string, error)
}
```

```text
Go API -> AI Service -> Provider abstraction
                          |-- Local / open-weight model
                          |-- Optional external provider
```

**Cost strategy:** target roughly $0 AI infrastructure cost during MVP where practical.

| Control | Limit |
|---------|-------|
| Prayer generations | 5 per user per day |
| Rewrite requests | 10 per user per day |
| Weekly reflection | 1 per user per week |
| Background generation | None. Generate only on explicit request. |

Do not persist or log private prayer content in AI requests beyond what the immediate request needs.

---

## 9. PWA layer

The PWA is the primary product, not a fallback website. iPhone-first.

**Manifest:** name "Amorae", short_name "Amorae", `display: standalone`, `start_url: /?source=pwa`, `orientation: portrait`, background color warm off-white, theme color deep charcoal, icons at 192 and 512 plus a maskable icon.

**Service worker caching:**

| Content | Strategy |
|---------|----------|
| App shell (HTML, JS, CSS) | Precache, cache-first with versioned busting |
| Static assets and fonts | Cache-first |
| API GET (read models) | Stale-while-revalidate, with cached fallback offline |
| Mutations | Never cached. Queued if offline (see 9.1). |

**Push:** VAPID. On permission grant, subscribe and POST the subscription to the server. Remove subscriptions the browser reports as gone (410).

**iOS specifics:** iOS has no `beforeinstallprompt`, so installation is guided manually (Add to Home Screen steps), never an aggressive popup. Respect safe-area insets with `env(safe-area-inset-*)`. Use `100dvh` for full-height layouts to handle the dynamic viewport and keyboard.

### 9.1 Offline behavior and sync

**Read offline:** previously loaded prayer points, events, goals, and cached read models remain viewable. Show a subtle offline indicator, never a full-screen "No internet."

**Write offline (limited, safe set):** mark a prayer point prayed, add an appreciation, add a journal entry, toggle a checklist item. Each queued mutation carries a client-generated idempotency key. Queue lives in IndexedDB and replays on reconnect.

**Conflict rule for MVP:** last write wins by server timestamp. Completions and checklist toggles are idempotent, so they never conflict. Publishing a prayer week and destructive deletes are online-only.

---

## 10. Non-functional requirements

| Area | Target |
|------|--------|
| Initial JS | Under 200 KB gzipped for first load |
| LCP | Under 2.5s on a mid-tier phone over 4G |
| Time to interactive | Under 3.5s cold, near-immediate on warm cache |
| API reads | p95 under 300 ms |
| Offline | Home and last-viewed content readable with no network |
| Data | Daily backups, point-in-time recovery where the host allows |

Frontend guidelines: avoid unnecessary `useEffect`, prefer TanStack Query for server state, keep components focused, no `any`, validate forms with Zod, keep API logic out of UI components.

---

## 11. Testing

Priority order: security and correctness first, then UX edge cases.

**Unit:** services (business rules), the scheduler (alternation and idempotency), auth, goal progress math.

**Integration (API + DB):** authentication, pairing, weekly alternation, prayer create and complete, events, goals, journal, memories, notifications.

**End-to-end (critical flows):** sign up, pair, setter publishes a week, both partners complete points independently, create and complete an event.

**Security:** couple-level authorization, cross-couple access attempts return 404, session expiry, input validation.

**PWA (manual checklist):** installation, standalone mode, offline reads, cached content, push permission and delivery, iOS safe-area behavior.

**UX edge cases:** small screens, large text, poor network, keyboard open, landscape, reduced motion.

---

## 12. Environment and deployment

**Local development (Docker Compose):** `web`, `api`, `postgres`. Redis is not introduced in MVP unless a concrete need appears.

**Config (environment variables):** database URL, session secret, VAPID public and private keys, AI provider key and flag, app base URL.

**Migrations:** golang-migrate, run on deploy. Never edit a shipped migration; add a new one.

---

## 13. Scalability and extensibility

The MVP is intentionally small, but it should not paint the product into a corner. Deliberate choices that keep it open:

- **Stateless API.** Auth rides in the cookie or token, so the API scales horizontally behind a load balancer with no sticky sessions.
- **Couple scoping plus indexes.** Every couple-owned table is indexed on `couple_id`, and hot lists have compound indexes (for example events on `(couple_id, starts_at)`), so query cost stays flat as data grows.
- **Membership models N, MVP enforces 2.** `couple_members` already supports multiple members and multiple couples per user. MVP enforces the limits with a `unique(user_id)` constraint and an application check. Relaxing to future relationship shapes is a constraint change, not a rewrite. Code never hardcodes "partner A" or "partner B"; it uses member order.
- **Independent modules.** `prayers`, `events`, `goals`, and the rest do not depend on each other, so teams can build and extend in parallel and add a new pillar feature without touching existing ones.
- **AI is isolated.** Behind an interface and a flag. It can be turned off, swapped, or scaled separately, and its outage never breaks core features.
- **Notifications abstracted.** Push is the MVP channel, but the notification module is written so email or SMS channels can be added without touching feature code.
- **Forward-compatible states.** Status columns use enums, so new states (for example a `snoozed` goal) extend cleanly.

---

## 14. Open questions

- Photo storage: object store (S3-compatible) versus DB-referenced URLs. Decide before Phase 4 (memories, journal photos).
- Password reset delivery: transactional email provider choice and free-tier limits.
- Whether the couple timezone is set at pairing or inferred from the creator's device. Affects the scheduler.
- Invite expiry window (proposed default: 7 days).

---

## 15. Decisions log

- **Dropped `event_participants`.** For a two-person couple, both partners are implicit participants. The table added no value in MVP. Reintroduce only if events ever include people outside the couple.
- **Merged milestones into `important_dates` via `type`.** Fewer tables, same capability. Milestones are `type = milestone`.
- **`setter_user_id` stored at week creation.** Alternation is computed once and frozen, so the setter never shifts retroactively if member data changes.
- **Cross-couple access returns 404, not 403.** Prevents resource enumeration.

---

## Change log

| Version | Date | Change |
|---------|------|--------|
| 2.0 | 2026-09-21 | Split out of the combined spec. Resolved vague feature lists into Must / Should / Later with acceptance criteria. Added full data model (columns, types, enums, constraints), API conventions and example payloads, a deterministic scheduler algorithm, concrete security controls, offline sync rules, non-functional targets, and a scalability section. |
