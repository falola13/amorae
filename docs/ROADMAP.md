# Amorae roadmap

**Written 2026-09-23.** Where the product actually is, what is missing or wrong,
and the order to build the rest in. Written for someone newer to backend work:
each step says what to build, in what order inside the step, what usually goes
wrong, and how you know you're done.

It is a working document. When a step lands, move it to *Done* with the date.
The requirement detail behind every line lives in
[`requirements/`](requirements/README.md); this file is only the order of work.

---

## 1. Where the product is today

**Real** — a Go endpoint, a screen, and tests:

| Area | What works |
|---|---|
| Accounts | Register, log in, log out, 30-day sessions |
| Passwords | Reset by emailed link (single use, revokes every session), change with the current password |
| Email | Change the sign-in address, re-checking the password first |
| Sessions | "Where you're signed in", sign out other devices |
| Account data | Export, delete account |
| Pairing | Create a couple, invite code, regenerate it, join by code, couple profile, per-person onboarding flags |
| Mail | `Mailer` interface: logs in development, Resend in production |
| Platform | Structured logs, Prometheus metrics, health and readiness, rate limiting, graceful shutdown, migrations in the binary |
| Web | Every screen built, installable PWA, offline queue for writes, adaptive phone/tablet/desktop layout, inline error states |

**Screens with no API behind them yet.** Each shows "Not available yet" where
its content would be, which is the honest state, not a bug:

Prayers (the weekly cycle, history, prayer mode) · Events and the calendar ·
Goals · Challenges · Journal · Appreciations · Memories · Milestones ·
Notification preferences and push.

**Not started at all:** the background worker, push delivery, photo storage.

So: the account-and-security half of the product is finished. The half that
makes it *Amorae* is not.

---

## 2. Lapses and gaps, as of today

Each one is either already tracked (the id in brackets) or new here.

### Correctness and safety

- **Idempotency keys are missing** for queued offline writes [FR-PWA-009], so
  the writes that create rows are `onlineOnly` until their endpoints take a key
  [DEC-28]. Not a Step 1 blocker: every prayer write is idempotent in the
  schema. It becomes live when Together's endpoints are built.
- **No audit log** of security-sensitive actions [NFR-SEC-020].
- **No common-password check** at registration [NFR-SEC-004].
- **No Content-Security-Policy on pages** — only on the service worker
  [NFR-SEC-015].
- **Registration still reveals whether an email exists** (`409 email_taken`).
  Closing it needs the verify-by-email flow [Q-05].
- **One API instance only.** The rate limiter is in memory, so a second
  instance would silently multiply every limit [DEC-11, Q-13].

### Product gaps

- **The faith consent gates nothing** [FR-AUTH-012, FR-PAIR-009, DEC-29]. Sign-up asks "Show me
  faith content (prayers and scripture). Optional." and records the answer with its policy
  version — and then no part of the app reads it. Untick it and you still get prayer weeks,
  scripture fields and "Two hearts, one faith". It is the one shipped promise the product
  currently breaks, and it is also what a secular mode would be built from: a space uses faith
  vocabulary only while both partners hold consent.

- **Challenge days are shared, not per-partner** [Q-23]: one `done` flag per
  day means either partner can tick it for both. Decide before the endpoint is
  built, because it changes the schema.
- **The prayer-week reflection is per-partner in the requirements
  ([FR-PRAY-006]) but a single field in `types.ts`.** Settle it when the module
  lands, and change the contract and the screen in the same commit.
- **No email verification** [Q-05], **no photo storage** [Q-06].

### Process gaps

- **No web tests at all** [NFR-MAINT-003, Q-15]. Go has ~120; the web app has
  lint, typecheck and build only.
- **CI does not check formatting or vulnerabilities**: no `prettier --check`,
  no `govulncheck`, no `npm audit`, no Dependabot [NFR-SEC-019].
- **No error reporting or product metrics** [NFR-OBS-006, Q-14], so a private
  beta would teach you less than it could.

### Design gaps

- No dark mode, and no dark palette to build one from [Q-21].
- Cold-start offline reading is not supported, by decision [DEC-04, Q-01].
- Prayer Mode has no screen wake lock, so the phone sleeps mid-prayer.

---

## 3. The plan

### Step 1 — Prayers, end to end ✅ done

The signature feature, and the best one to learn on: it has reads, writes,
permissions ("only this week's setter"), and rules that depend on time.

**Build it in this order.** It is the same order every module you already have
was built in, and it is deliberate: each layer is testable before the next
exists.

1. **Migration** (`migrations/0000N_prayers.sql`) — tables and constraints.
2. **Entity and rules** (`prayers.go`) — what a week is, who sets it, what a
   valid set of points looks like. Plain Go: no SQL, no HTTP. Test it with no
   database.
3. **Repository** (`repository_postgres.go`) — SQL only. Translate Postgres
   errors into domain errors so nothing above it imports pgx.
4. **Service** (`service.go`) — the use cases, and every permission rule.
5. **Handler and DTOs** (`handler.go`) — decode, call, map, respond.
6. **Wire it** in `internal/app`, then stop: the screens light up on their own.

**What usually goes wrong here:**

- *Creating the week on read.* `GET /prayers/current` must never insert. Two
  requests racing on a Sunday morning would create two weeks. Creation belongs
  to the worker (Step 2) with `ON CONFLICT DO NOTHING` behind a unique
  constraint [BR-PRAY-03].
- *Recomputing whose turn it is.* Compute it once, when the week is created,
  and store `setter_user_id` [DEC-18]. Recomputing on read means the setter can
  change retroactively.
- *Timezones.* A week starts Sunday 00:00 in the **couple's** timezone, not the
  server's and not each partner's [BR-PRAY-02, Q-07]. Store `week_start` as a
  plain date.
- *Permissions in the handler.* They belong in the service, where they can be
  tested without HTTP.

**Done when:** a couple can see the current week, the setter can write and
publish up to 10 points, each partner marks their own completions, history
lists past weeks, and a request for another couple's week returns 404 [DEC-19].
Plus tests: rules in `prayers_test.go`, SQL in `repository_postgres_test.go`,
use cases and permissions in `service_test.go`.

### Step 2 — The worker ✅ done (merged into Step 3)

A second entry point, `cmd/worker`, in the same module. It wakes hourly, finds
couples whose local Sunday has begun and who have no week yet, and inserts one.
Postgres is the job store — `FOR UPDATE SKIP LOCKED`, no queue [DEC-20, Q-16].

**Done when:** a couple in Lagos and a couple in London each get their week at
their own midnight, running the worker twice in a row creates nothing extra,
and killing it mid-tick loses nothing.

### Step 3 — Notifications and push ← you are here

Now the worker has something to say. VAPID keys, a `push_subscriptions` table,
delivery from the worker, subscriptions removed when the browser reports them
gone. Notification preferences already have a screen waiting.

**Watch out for:** notification text must never carry private content — a lock
screen is a public surface [FR-NOTF-005].

### Step 4 — Together: events, calendar, goals, challenges

Repetition of Step 1, which is the point: by the third module the shape should
feel boring. Settle Q-23 (per-partner challenge days) before writing that
schema.

### Step 5 — Connect and Remember: journal, appreciations, memories, dates

Same shape again. Two things here need decisions first: the appreciation undo
window is already specified [DEC-21], and photos need storage [Q-06].

### Step 6 — Before real couples use it

- Idempotency keys [FR-PWA-009] — **before** the first queued write endpoint
  goes live, which realistically means during Step 1.
- Leave couple [Q-09], audit log, common-password check, CSP.
- Answer: retention [Q-11], hosting and sub-processors [Q-12], consent for
  faith content [Q-20], minimum age [Q-10].
- A restore drill: prove a backup actually restores [NFR-AVAIL-004].

### Step 7 — Public launch

Penetration test, WCAG 2.2 AA audit, alerting with runbooks, incident
severities, argon2id [Q-02], legal review [Q-17].

---

## 4. Habits that make the rest cheaper

- **Tests in the same commit as the code.** You already have the patterns:
  in-memory fakes for services, `dbtest.New(t)` for repositories. This is why
  auth could be changed today without fear.
- **Small commits.** One logical change each. A 170-file commit cannot be
  reviewed, and cannot be reverted when one part of it is wrong.
- **`docs/API.md` and `apps/web/src/lib/api/types.ts` change together**, in the
  same commit, every time.
- **When the API and a screen disagree, change both** rather than bending the
  API to fit a screen that was built against a guess.
- **Run the checks before pushing:**

```bash
cd apps/api && gofmt -l . && go vet ./... && go test ./... -race
cd apps/web && npm run typecheck && npm run lint && npm run format:check && npm run build
```

---

## 5. Deliberately not doing

| Not doing | Why | When to revisit |
|---|---|---|
| Redis | Nothing needs it. Sessions are one indexed row; there is no measured hot query; the rate limiter is correct for one instance. It would add a service to run, secure and reason about, for no gain today. | A second API instance (shared rate limits), a job queue Postgres can't carry, or a measured hot query |
| Dark mode | No dark palette exists; every colour token is single-valued [Q-21] | After MVP, defining both values per token at once |
| Cold-start offline reading | The service worker deliberately caches nothing personal [DEC-04, Q-01] | If the offline page turns out to be a common landing |
| Native apps | Web-only is the constraint [C-01] | Only if a PWA limit blocks a goal |
| Streaks, scores, leaderboards | Against the product's own principle: intentional, not addictive | Never |

---

## 6. Decisions still owed

Nineteen open questions remain in
[`requirements/05-decisions-and-open-questions.md`](requirements/05-decisions-and-open-questions.md),
each with a recommendation ready to accept or reject. The ones that block
building, rather than launching:

| Question | Blocks |
|---|---|
| Q-16 worker runtime | Step 2 |
| Q-23 challenge completion | Step 4's schema |
| Q-06 photo storage | Step 5 |

---

## Done

| Date | What |
|---|---|
| 2026-09-24 | **The worker and push** [FR-NOTF-001..004, 007]: `cmd/worker` in the same module, hourly, holding no state — what has been sent is a row, so a restart or a second worker sends each notification once. Preferences and subscriptions endpoints; a push seam that logs in development and encrypts in production |
| 2026-09-23 | **Prayers, end to end** [FR-PRAY-001..007]: repository, service, DTO, handler and wiring. The week is created on first read rather than waiting for a worker; the setter rotation is anchored to the couple's first week; points are matched by id so reordering keeps what has been prayed |
| 2026-09-23 | The couple's timezone is its own setting, editable by either partner [DEC-27, Q-07 resolved]: the week turns over in the couple's zone, reminders in each person's |
| 2026-09-23 | Every offline write now declares whether replaying it is safe, enforced by the type [DEC-28]; the seven that create rows are online-only until their endpoints take a key [FR-PWA-009] |
| 2026-09-23 | Leaving a couple, end to end [FR-PAIR-008, DEC-25, DEC-26]: ends the space for both, 30 days to read and download, hourly purge, and an archive that does not lock either person out of starting again [Q-09, Q-24 both resolved] |
| 2026-09-23 | Account screens no longer sit behind the couple gate: settings, profile, devices and the past space stand on the account |
| 2026-09-23 | Password reset and change, account export and deletion, session list and sign-out-others, invite regeneration, Resend mailer [DEC-24] |
| 2026-09-23 | Mock API removed: the web app talks only to the real API [DEC-22] |
| 2026-09-23 | Adaptive tablet and desktop layout, animated splash, inline error states |
