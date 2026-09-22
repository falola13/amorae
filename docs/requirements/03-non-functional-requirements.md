# Amorae: Non-Functional Requirements

| | |
|---|---|
| **Document ID** | AMR-REQ-03 |
| **Version** | 3.0 |
| **Status** | Draft for review |
| **Owner** | Engineering |
| **Last updated** | 2026-09-22 |

---

## 1. Introduction

### 1.1 Purpose

[`02-functional-requirements.md`](./02-functional-requirements.md) says what Amorae shall do.
This document says how well: the performance, availability, scale, security, privacy,
accessibility, compatibility, offline, observability, maintainability and operations targets
the system must meet, and how each target is checked. It collects and supersedes the v2.0
engineering spec's security (§7), PWA (§9), non-functional (§10), testing (§11), deployment
(§12) and scalability (§13) sections, reconciled against the code as it stands on 2026-09-22.

### 1.2 Scope

Every non-functional requirement (`NFR-<CATEGORY>-NNN`) that applies to `apps/api` (Go),
`apps/web` (Next.js BFF), PostgreSQL, and the browser client, across dev, staging and
production. It does not restate functional behaviour (see 02) or visual design (see
[`04-design-specification.md`](./04-design-specification.md)); it references the architecture
docs for "how" rather than duplicating them.

### 1.3 How to read a target

- **Numbers are minimums or maximums**, stated with a unit and, where the metric varies by
  request or session, a percentile (`p75`, `p95`) and a measurement window (for example
  "monthly", "per release").
- **Where a target is measured** is stated explicitly, because the same metric can mean
  different things in different places:
  - **Field** — real user measurement, from actual devices on real networks. Amorae has no
    first-party field collection yet (Q-14), so field targets are aspirational until that
    exists.
  - **Lab** — a synthetic, repeatable run (for example Lighthouse CI on a fixed device and
    network profile) used as a proxy for field data.
  - **Server** — measured inside `apps/api` or `apps/web`, independent of the client device or
    network (for example the Prometheus latency histogram).
- Each requirement's **Measured by** line names the concrete method: a CI job, a dashboard
  query, a manual check, or "not yet instrumented" where that is the honest answer.

### 1.4 Environments

| Environment | Purpose | Data | Notes |
|---|---|---|---|
| **dev** | Local development (`docker compose`), `npm run dev` | Disposable, seeded or empty | No service worker (registration is production-only); mock API mode may be on |
| **staging** | Pre-production verification, QA, demos | Synthetic or anonymised, never real couples' data | Mirrors production configuration; the only place load or failure tests run |
| **production** | Real users | Real, private couple data | Targets in this document are production targets unless stated otherwise |

### 1.5 References

- [`./01-product-requirements.md`](./01-product-requirements.md) — vision, principles, roadmap phases
- [`./02-functional-requirements.md`](./02-functional-requirements.md) — functional requirements this document's targets apply to
- [`./04-design-specification.md`](./04-design-specification.md) — tokens, components, screens (accessibility and compatibility implementation detail)
- [`./05-decisions-and-open-questions.md`](./05-decisions-and-open-questions.md) — `DEC` and `Q` register cited throughout
- [`../ARCHITECTURE.md`](../ARCHITECTURE.md) — module layout, request lifecycle, rate limiting, observability, PWA, scaling path
- [`../API.md`](../API.md) — HTTP contract, conventions, error envelope
- [`../adr/`](../adr/) — architecture decision records (0001 modular monolith, 0002 BFF and opaque sessions, 0003 stdlib-first, 0004 PWA service worker)

---

## 2. Quality targets at a glance

| Category | Headline target | Status |
|---|---|---|
| Performance | Core Web Vitals p75 mobile: LCP ≤ 2.5 s, INP ≤ 200 ms, CLS ≤ 0.1 (lab today) | Not started |
| Performance | API server-side latency p95 ≤ 300 ms reads, ≤ 500 ms writes | Partial (instrumented; not yet verified under load) |
| Availability | 99.5% monthly availability, error budget ≈ 3 h 39 min/month | Not started (no external probe yet) |
| Scale | Single API instance; every couple-owned table indexed on `couple_id` | Partial |
| Security | OWASP ASVS 5.0 Level 2 target; login throttling and enumeration resistance in place | Partial |
| Privacy | NDPA + GDPR compliance; explicit consent for faith/prayer data; no third-party trackers | Partial |
| Accessibility | WCAG 2.2 Level AA across all screens | Not started |
| Compatibility | iOS Safari 16.4+, last two versions of major evergreen browsers, 320 px minimum width | Partial |
| Offline | Offline page on cold navigation; queued writes survive 7 days | Implemented |
| Observability | Structured logs, request metrics, health/readiness endpoints; no alerting yet | Partial |
| Maintainability | Go tests with `-race` against real Postgres in CI; no web tests yet | Partial |
| Operations | Environment separation, fail-fast config; no documented rollback/incident process yet | Partial |

---

## 3. Performance (PERF)

### NFR-PERF-001 Core Web Vitals at p75 on mobile

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Test |

The web app shall meet, at the 75th percentile on a mid-tier mobile profile, Largest
Contentful Paint (LCP) ≤ 2.5 s, Interaction to Next Paint (INP) ≤ 200 ms, and Cumulative
Layout Shift (CLS) ≤ 0.1.

Until first-party field data exists (Blocked by Q-14), these are lab targets: Lighthouse CI
run against a mid-tier mobile device profile over a simulated 4G connection. Once a real user
monitoring source exists, the field measurement (p75 over a 28-day rolling window) becomes the
governing number and the lab run becomes a regression gate in CI.

**Measured by:** Lighthouse CI (not yet wired into `.github/workflows/ci.yml`) against key
routes (Home, current prayer week, events list) on the mobile profile. Not currently run.

### NFR-PERF-002 Initial route JavaScript budget

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Not started | Test |

The web app shall keep the JavaScript shipped for the initial route at or below 200 KB
gzipped.

**Measured by:** a bundle-size budget check in CI (for example `next build` output compared
against a threshold, or `@next/bundle-analyzer` in a check mode). No such check exists in
`.github/workflows/ci.yml` today.

### NFR-PERF-003 API read latency

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Partial | Analysis |

The API shall serve read (GET) requests with server-side latency p95 ≤ 300 ms, measured over
a rolling 5-minute window per route, excluding endpoints that perform password hashing.

**Measured by:** the existing Prometheus request-duration histogram in
`apps/api/internal/platform/metrics`, labelled by route pattern (`docs/ARCHITECTURE.md`
§Observability). The histogram exists and is scraped on `GET /metrics`; no dashboard or
alert on the p95 value exists yet (see NFR-OBS-004).

### NFR-PERF-004 API write latency

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Partial | Analysis |

The API shall serve write (POST/PATCH/DELETE) requests with server-side latency p95 ≤ 500 ms,
measured over a rolling 5-minute window per route, excluding `/v1/auth/register` and
`/v1/auth/login` (which perform bcrypt hashing and are expected to run slower by design).

**Measured by:** the same Prometheus latency histogram as NFR-PERF-003, filtered to
non-GET methods.

### NFR-PERF-005 Warm-start app shell render

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Implemented | Demo |

When the service worker has already cached the app shell (`/_next/static/*`, icons, manifest),
a repeat visit shall render the app shell without a network round trip for those assets, so the
shell paints immediately even on a slow connection, before any personal data (which is never
cached, per ADR 0004) has loaded.

**Measured by:** manual check with DevTools → Application → Service Workers → Offline, or
throttled network, confirming the shell paints from the `precache` cache while API/page
requests still go to the network. No automated check exists.

---

## 4. Availability (AVAIL)

### NFR-AVAIL-001 Monthly availability SLO

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Analysis |

The API and web app shall together meet 99.5% monthly availability (an error budget of about
3 h 39 min per month), where a minute counts as unavailable if an external probe of `GET
/readyz` fails or the 5xx response ratio across all routes exceeds 5% for that minute.

**Measured by:** an external uptime probe (not yet set up) hitting `/readyz` on a fixed
interval, combined with the 5xx-to-total ratio from the Prometheus request-count metric. No
probe or SLO dashboard exists today.

### NFR-AVAIL-002 Recovery point objective

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Analysis |

The system shall lose no more than 24 hours of data on a full restore (RPO ≤ 24 h), via daily
backups. Where the hosting provider offers point-in-time recovery, RPO shall be ≤ 5 minutes.

**Measured by:** inspection of the backup schedule and, when point-in-time recovery is
available, its retention window. Hosting provider and region are not yet chosen (Blocked by
Q-12), so no backup schedule exists yet.

### NFR-AVAIL-003 Recovery time objective

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Demo |

The system shall be restorable to a working state within 4 hours of a decision to restore
(RTO ≤ 4 h).

**Measured by:** a timed restore drill (see NFR-AVAIL-004). No drill has been run; there is no
production deployment yet.

### NFR-AVAIL-004 Restore drill cadence

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Demo |

The team shall run a full restore drill (restore a backup to a scratch environment and verify
row counts and a sample of rows) at least quarterly once in production, and at least once
before public launch.

**Measured by:** a dated drill record showing the restore completed within RTO and the
verification checks passed. None exists yet.

### NFR-AVAIL-005 Graceful degradation of non-core channels

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Partial | Test |

An outage of email, push notifications, or the AI provider shall never block a couple from
reading or writing their core data (prayer, events, goals, challenges, journal, appreciation,
memories, important dates).

Architecturally this holds today: AI sits behind an interface and a feature flag (DEC-15), and
push/email are not yet wired into any write path, so neither can currently block a read or
write. This is "Partial" because email and push sending are not built yet (Not started per the
as-built facts), so the isolation is untested under real failure — it is a property of the
design, not yet a verified behaviour under a simulated provider outage.

**Measured by:** once email and push exist, an integration test that fails the provider and
asserts core writes still succeed. Not yet testable.

---

## 5. Scalability (SCALE)

### NFR-SCALE-001 Stateless API

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Inspection |

The API shall hold no in-process state that a request depends on across requests, other than
the in-memory rate limiter (NFR-SCALE-002). Session state lives in PostgreSQL (`sessions`
table, keyed by token hash), not in process memory.

**Measured by:** code inspection of `apps/api/internal/modules/auth` and
`internal/platform/database` (`docs/ARCHITECTURE.md` §Scaling path, row "More traffic").

### NFR-SCALE-002 Single API instance until a shared rate limiter exists

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Partial | Inspection |

The system shall run exactly one API instance in production until a shared (cross-instance)
rate limiter replaces the in-memory one, because the login-attempt and per-IP limits (DEC-11)
are correct only when a single process holds all counters. Blocked by Q-13.

**Measured by:** inspection of the deployment configuration (instance count) alongside
`apps/api/internal/platform/middleware/ratelimit.go`, which implements the one-method
`Limiter.Allow(key)` interface so a Redis- or Postgres-backed limiter can replace it without
changing call sites (`docs/ARCHITECTURE.md` §Scaling path, row "Rate limits across replicas").
No production deployment exists yet to violate or satisfy this.

### NFR-SCALE-003 Couple-scoped indexing

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Partial | Inspection |

Every table that stores couple-owned data shall carry an index on `couple_id` (or on a foreign
key that resolves to one query away), so query cost does not grow linearly with total platform
data as couples are added.

As built: `sessions` is indexed on its token hash (not couple-scoped by design — it is
user-scoped). The `couples`, `couple_members` (unique `couple_id, user_id`) and
`couple_invitations` tables exist with their documented constraints. Feature tables that will
carry `couple_id` (prayers, events, goals, challenges, journal, appreciations, memories,
important dates) are specified in 02 §4 but are mostly not yet built (mock only), so their
indexes cannot yet be verified in migrations. Status is "Partial": the pattern is established
for what exists, not yet complete for what doesn't.

**Measured by:** inspection of `apps/api/migrations/*.sql` for a `couple_id` index on every
couple-owned table, re-checked whenever a migration adds one.

### NFR-SCALE-004 Initial sizing assumption

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Not started | Analysis |

The system shall be sized, as a starting assumption to validate rather than a measured result,
for 10,000 couples with a peak load of 50 requests/second, and the scaling path for exceeding
that (documented in `docs/ARCHITECTURE.md` §Scaling path) shall be kept current as real load
data arrives.

**Measured by:** a load test against staging at the target rate, comparing p95 latency against
NFR-PERF-003/004. No load test has been run; there is no production traffic to calibrate the
assumption against yet.

### NFR-SCALE-005 Database connection pool limits

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Inspection |

The API shall bound its PostgreSQL connections with a configured pool size
(`internal/platform/database`), rather than opening connections without limit, so a traffic
spike cannot exhaust the database's connection budget.

**Measured by:** inspection of `internal/config` for the pool-size setting and of
`internal/platform/database` for where it is applied to the pgx pool.

---

## 6. Security (SEC)

### NFR-SEC-001 Application security standard

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Partial | Inspection |

The system shall target OWASP ASVS 5.0 Level 2 as its baseline security control set. Individual
controls in this section (session management, input validation, authentication, access
control) implement pieces of that baseline; there is no single pass/fail check for the standard
as a whole.

**Measured by:** periodic self-assessment against the ASVS 5.0 Level 2 checklist (manual,
Engineering-owned), and the penetration test in NFR-SEC-021 before public launch.

### NFR-SEC-002 STRIDE threat model

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Inspection |

The team shall produce a STRIDE threat model of the system before private beta, and update it
whenever a major feature (a new module, a new trust boundary, a new data flow to a third
party) ships.

**Measured by:** a dated threat-model document reviewed by Engineering, re-reviewed on major
feature changes. None exists yet.

### NFR-SEC-003 Password policy

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall require passwords of at least 10 characters, counted as Unicode code points,
with no composition rules, and at most 72 bytes while bcrypt is the password hash (DEC-06).
The web client and the API shall apply the identical rule.

**Measured by:** Go unit tests on the validation rule and the equivalent Zod schema on the web
client (`apps/web/src/lib/api/schemas.ts`); both are exercised in CI.

### NFR-SEC-004 Common and breached password check

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Test |

The system shall reject passwords found on a list of common or previously breached passwords,
in addition to the length rule in NFR-SEC-003, checked against an embedded list so that
registration and password changes do not depend on a third-party call.

**Measured by:** a unit test asserting known common passwords are rejected. No such check
exists in `apps/api/internal/modules/auth` today.

### NFR-SEC-005 Password storage

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Inspection |

Passwords shall be stored only as a salted hash, never in plain text or reversibly encrypted.
As built, the hash is bcrypt at cost factor 12. Whether to move to argon2id before launch is
open (Blocked by Q-02).

**Measured by:** code inspection of the password hashing call in
`apps/api/internal/modules/auth`, and confirmation that `password_hash` is the only
persisted form of the password.

### NFR-SEC-006 Login throttling and enumeration resistance

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The login endpoint shall respond with a generic `invalid_credentials` error for both a wrong
password and an unknown email, running a dummy bcrypt comparison for unknown emails so response
timing does not reveal which emails have accounts, and shall throttle attempts at 10 per email
per 15 minutes plus 20 requests per client IP per minute across `/v1/auth/*`, both returning
429 with the same message and a `Retry-After` header (DEC-11).

**Measured by:** Go tests covering the generic error, the dummy comparison for unknown emails,
the per-account limit being checked before bcrypt, and the 429 response
(`apps/api/internal/modules/auth/service_test.go`,
`apps/api/internal/platform/middleware/security_test.go`).

**Known gap:** registration answers `409 email_taken`, which tells a caller that an account
exists. The per-IP limit slows this down but can't hide it; closing it needs a verify-by-email
sign-up flow that answers identically either way (Q-05, and `docs/ARCHITECTURE.md` §Deliberately
not included).

### NFR-SEC-007 Session management: absolute lifetime

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

Sessions shall be opaque, random 32-byte tokens, stored only as a SHA-256 hash, valid for 30
days from creation with no sliding renewal, and revoked immediately (row deleted) on logout
(DEC-02).

**Measured by:** Go integration tests on session creation, expiry and logout in
`apps/api/internal/modules/auth`.

### NFR-SEC-008 Session management: idle timeout

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Not started | Test |

Sessions shall additionally expire after a period of inactivity, in addition to the 30-day
absolute limit. Blocked by Q-03, which proposes a 14-day idle expiry with last-use updated at
most hourly.

**Measured by:** a Go integration test asserting an idle session past the threshold is
rejected. Not built; no idle tracking exists on the `sessions` table today.

### NFR-SEC-009 Session revocation on credential change

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Not started | Test |

Changing the account password, or completing a password reset, shall revoke every other active
session for that user, so a stolen session cannot survive the user securing their account.

**Measured by:** a Go integration test asserting other sessions are deleted after a password
change. Not built; password change/reset do not exist yet (see NFR-SEC-005, Q-02, and password
reset under Not started in the as-built facts).

### NFR-SEC-010 Re-authentication for sensitive actions

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Partial | Test |

Sensitive account actions shall require the current password, not just an active session.
Implemented today for email change (`PUT /v1/users/me/email`, DEC-07): a wrong password
returns a 400 field error, never a 401 (which the client would read as session expiry), and the
endpoint shares its attempt limit with login. Password change and account deletion shall carry
the same requirement once built (DEC-07, Blocked by Q-03).

**Measured by:** Go integration test for email change (exists); equivalent tests for password
change and account deletion once those endpoints exist (not built).

### NFR-SEC-011 Couple-scoped authorisation

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Partial | Test |

Every request for a couple-owned resource shall be scoped to the caller's own couple, derived
from the authenticated session and never trusted from the client, and a resource id belonging
to another couple shall return 404, never 403 (DEC-19), so a caller cannot distinguish "not
yours" from "does not exist."

Applies today to the couples module (in progress). Modules for prayers, events, goals,
challenges, journal, appreciations, memories and important dates are not built yet, so the rule
is stated for the whole surface but only partially verified.

**Measured by:** a Go integration test per couple-owned module asserting a second couple's
member gets 404 on another couple's resource ids.

### NFR-SEC-012 Input validation

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The API shall reject request bodies containing unknown JSON fields, cap request bodies at 1
MiB, and validate every field server-side using the same rules the web client's Zod schemas
express for user experience, so the server never trusts client-side validation alone.

**Measured by:** Go unit/integration tests on the JSON decoder in `internal/platform/httpx`
(unknown-field rejection, body cap) and per-module validation tests.

### NFR-SEC-013 CSRF defence

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Inspection |

The system shall defend state-changing requests against cross-site request forgery using the
`SameSite=Lax` session cookie, the origin check on Next.js Server Actions, and a BFF proxy that
rejects state-changing requests whose `Sec-Fetch-Site` or `Origin` header indicates a
cross-site request, with no CSRF token issued or checked (DEC-09).

**Measured by:** inspection of the cross-site guard in the BFF proxy
(`apps/web/src/app/api/v1/[...path]/route.ts`) and of the session cookie attributes. An
automated test asserting a cross-site state-changing request is rejected arrives with the web
test suite (NFR-MAINT-003, Q-15).

### NFR-SEC-014 Security response headers

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Inspection |

The web app shall send `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`,
`Referrer-Policy: strict-origin-when-cross-origin`, and a `Permissions-Policy` denying camera,
microphone and geolocation, on every page.

**Measured by:** inspection of `apps/web/next.config.ts` (`securityHeaders`, applied to
`/:path*`), confirmed by this review of the live configuration.

### NFR-SEC-015 Content-Security-Policy for pages

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Not started | Inspection |

The web app shall send a `Content-Security-Policy` header on page responses (not only on the
service worker, where one already exists) restricting script, style and connection sources to
trusted origins. Priority is **Should** for private beta and escalates to **Must** before
public launch (see the release gate table in §14).

Verified in `apps/web/next.config.ts`: a `Content-Security-Policy` header is set only for the
`/sw.js` response (`default-src 'self'; script-src 'self'`); `securityHeaders`, applied to
`/:path*`, has no CSP entry. Pages are not covered.

**Measured by:** inspection of the response headers on a rendered page (not just `/sw.js`).

### NFR-SEC-016 Transport security

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Inspection |

Production traffic shall be served over TLS 1.2 or later only, with HTTP Strict Transport
Security (HSTS) enabled, enforced at the network edge (load balancer, CDN or reverse proxy)
rather than in application code.

**Measured by:** inspection of the edge/proxy configuration once a hosting provider is chosen
(Blocked by Q-12); a TLS scan (for example an SSL Labs-style check) against the production
hostname.

### NFR-SEC-017 Encryption at rest

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Inspection |

Data at rest (the PostgreSQL database and any object storage) shall be encrypted at the disk or
database level, provided by the hosting platform rather than implemented in application code.

**Measured by:** inspection of the hosting provider's encryption-at-rest setting once a
provider is chosen (Blocked by Q-12).

### NFR-SEC-018 Secrets management

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Inspection |

Secrets (database credentials, `BFF_SECRET`, session and hashing configuration, any future
VAPID or AI provider keys) shall be supplied only through environment variables or a secret
manager, never committed to the repository. `BFF_SECRET` shall be at least 32 characters, and
the API shall fail fast at startup on a missing or invalid secret.

**Measured by:** inspection of `internal/config` (fails fast on invalid values, per the
as-built facts) and of `.env.example` / `.gitignore` to confirm no real secret is committed;
`git`-history scanning for committed secrets is not automated yet.

### NFR-SEC-019 Dependency and vulnerability scanning

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Not started | Inspection |

The project shall run automated dependency and vulnerability scanning (for example
`govulncheck` for the Go module, `npm audit` for the web app, and Dependabot or an equivalent
for version updates) on a scheduled or per-PR basis.

Checked: `.github/workflows/ci.yml` has no `govulncheck` or `npm audit` step, and no
`.github/dependabot.yml` exists.

**Measured by:** inspection of `.github/workflows/*.yml` and `.github/dependabot.yml`.

### NFR-SEC-020 Security audit log

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Test |

The system shall keep an audit log of security-sensitive actions — login success, login
failure, logout, email change, password change, invite created, invite joined, leave couple,
account deletion — recording actor, action, timestamp and `request_id`, and never the content
of the action.

**Measured by:** an integration test asserting an audit row is written for each listed action.
No audit table or writer exists; only the general access log (method, path, client IP, status,
duration — see NFR-OBS-001) exists, which is not a durable per-action security record.

### NFR-SEC-021 General per-user API rate limit

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Not started | Test |

The API shall apply a general rate limit per authenticated user across non-auth endpoints, in
addition to the auth-specific limits in NFR-SEC-006, proposed at 100 requests per minute per
user (v2.0 §10).

**Measured by:** an integration test asserting the limit trips at the configured threshold. Not
built; today only `/v1/auth/*` is rate-limited (DEC-11).

### NFR-SEC-022 Join-code attempt limits

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Test |

Attempts to join a couple by invite code shall be limited per user and per IP, so that guessing
across the roughly 17.6 million possible codes (DEC-08) is impractical within the code's 7-day
lifetime. Blocked by Q-08 for the exact limits.

**Measured by:** an integration test asserting repeated wrong-code join attempts are throttled.
The couples module is in progress; this control is not yet verifiable.

### NFR-SEC-023 Penetration test before public launch

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Demo |

The system shall undergo an external penetration test before public launch, and material
findings shall be remediated or explicitly risk-accepted before that launch.

**Measured by:** a dated penetration test report and a remediation log. Not started; no
production deployment exists to test yet.

### NFR-SEC-024 Mock authentication refused in production

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Inspection |

A production build of the web app shall refuse mock sign-in (which otherwise accepts any email
and password under `NEXT_PUBLIC_API_MOCK`) unless `ALLOW_MOCK_AUTH=true` is explicitly set,
marking a deliberate demo deployment rather than a default-on backdoor.

**Measured by:** inspection of `apps/web/src/lib/config.ts` and the mock adapter's guard; a
build-time or startup check.

### NFR-SEC-025 Metrics endpoint not publicly exposed

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Inspection |

The Prometheus `/metrics` endpoint shall be served on a separate listener (`METRICS_ADDR`),
bound to loopback by default, so it cannot be reached through the public API port.

**Measured by:** inspection of `apps/api/internal/platform/metrics` and `internal/app`'s server
wiring, and a network check confirming `/metrics` is unreachable from outside the host in a
default deployment.

---

## 7. Privacy (PRIV)

### NFR-PRIV-001 Applicable law

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Inspection |

The system shall be designed and operated to comply with the Nigeria Data Protection Act 2023
(NDPA) and, for any user in scope, the GDPR.

**Measured by:** legal review of the Terms and Privacy pages and of data handling practice
(Blocked by Q-17). Draft `/terms` and `/privacy` pages exist, marked draft, pending legal
review; the compliance posture itself has not been assessed by counsel yet.

### NFR-PRIV-002 Explicit consent for faith/prayer content

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Demo |

Because prayer and faith content can reveal religious belief — sensitive personal data under
the NDPA and special-category data under GDPR Art. 9 — the system shall capture explicit,
specific consent to process this category of data, separate from general acceptance of the
Terms, recorded with a timestamp and the policy version. Where and how it is asked is Q-20.

**Measured by:** a Go/web integration test asserting registration or first prayer-feature use
is blocked until consent is recorded. Not built; no consent capture exists today.

### NFR-PRIV-003 Data minimisation

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Partial | Inspection |

The system shall collect only the data each feature needs to function, and shall not collect
data "for later" without a stated purpose. As built, the `users` and `couples` tables hold only
fields the shipped features use; the data model in 02 §4 defines the same discipline for
unbuilt modules.

**Measured by:** review of each new migration against the feature it supports, as part of code
review.

### NFR-PRIV-004 No ads, sale of data, or third-party trackers

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Inspection |

The system shall carry no advertising, shall not sell user data, and shall make no third-party
tracking requests from the web app.

Confirmed: fonts are self-hosted (Manrope, variable woff2), not loaded from a third-party font
service, consistent with the "no third-party requests" claim. A full network-request audit
across every screen has not been re-run for this document.

**Measured by:** a network-request capture across all screens showing no third-party requests
that transmit identifying data.

### NFR-PRIV-005 Data subject access and export

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Test |

A user shall be able to request and receive a machine-readable (JSON) export of their personal
data, fulfilled within 30 days of the request.

**Measured by:** an integration test exercising an export endpoint and checking the response
covers every table that references the requesting user. No export endpoint exists in Go or in
the web mock.

### NFR-PRIV-006 Right to deletion

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Mock only | Test |

A user shall be able to request deletion of their account and personal data, fulfilled within
30 days, subject to the couple-dissolution rule in Q-09.

`DELETE /v1/users/me` exists only in the web mock (localStorage); no Go endpoint exists.

**Measured by:** an integration test against the real endpoint once built, confirming data is
gone (or scheduled for deletion within the retention window, Q-11) after the call.

### NFR-PRIV-007 Rectification

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Implemented | Test |

A user shall be able to correct their own profile data (display name, timezone) directly.

**Measured by:** Go integration test on `PATCH /v1/users/me`.

### NFR-PRIV-008 Retention schedule

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Inspection |

The system shall follow a defined retention schedule: expired sessions purged 7 days after
expiry; deleted accounts hard-deleted within 30 days, with backups rolling off within 35 days;
application logs kept 30 days and never containing content. Recorded as the recommendation
under Q-11; not yet implemented as a scheduled job.

**Measured by:** inspection of the (future) retention/cleanup job and of log-storage
configuration for a 30-day expiry.

### NFR-PRIV-009 Couple dissolution

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Demo |

When a couple dissolves (a partner leaves, or one deletes their account), the system shall
follow the rule recorded at Q-09: both partners get 30 days of read-only access and export, then
shared couple content is deleted; each partner's individually authored personal data follows
their own account.

**Measured by:** an integration test simulating a leave/delete and checking read-only access
and the 30-day timer. No "leave couple" action exists yet (mock or real), per the as-built
facts.

### NFR-PRIV-010 Sub-processors and hosting region

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Inspection |

The Privacy page shall list every sub-processor (hosting, email provider, any future AI
provider) and the hosting region, and the basis for any cross-border data transfer. Blocked by
Q-12.

**Measured by:** inspection of the published Privacy page against the actual list of vendors in
use.

### NFR-PRIV-011 No content in logs, errors or metrics

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Inspection |

Application logs, error responses and metrics shall never contain the content of a couple's
data (prayer text, journal bodies, appreciation notes, event descriptions, etc.), only
identifiers, actions and technical fields.

Verified by reading `logging.go` in full: the access log records `method`, `path`, `client_ip`,
`status`, `duration_ms`, `bytes` — no body. Errors go through the single `apperr` →
`httpx.Error` mapping, which sends only `code` and a safe `message`, never the internal cause.
No content-bearing feature (prayer, journal, etc.) is built yet in Go, so this holds for the
auth/user surface that exists, not yet exercised for the richer content types.

**Measured by:** ongoing code review of every new module's logging and error handling against
this rule; a log-scrubbing test is not currently automated.

### NFR-PRIV-012 Breach notification

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Inspection |

On confirming a personal-data breach, the system operator shall notify the applicable
regulator within 72 hours and affected users without undue delay.

**Measured by:** a documented incident-response runbook covering breach notification (see
NFR-OBS-005 for the general alerting/runbook requirement). None exists yet.

### NFR-PRIV-013 Minimum age

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Demo |

The system shall state a minimum age of 18 in the Terms and shall have the user confirm it at
sign-up. Recorded as the recommendation under Q-10.

**Measured by:** a web integration test asserting registration requires the age confirmation.
No age gate exists in the sign-up flow today.

### NFR-PRIV-014 Visibility disclosure between partners

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Partial | Inspection |

Both partners shall be shown clearly, in-product, that everything either of them adds to
couple-scoped features (prayer, events, goals, journal, appreciation, memories, important
dates) is visible to both. This is a product-copy and onboarding requirement, not only a legal
one.

The pairing flow exists (couples module in progress); explicit copy stating shared visibility
has not been reviewed here.

**Measured by:** a design/content review of pairing and first-use screens (paired with
`04-design-specification.md`).

---

## 8. Accessibility (A11Y)

### NFR-A11Y-001 WCAG 2.2 Level AA conformance

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Test |

Every screen shall conform to WCAG 2.2 Level AA.

**Measured by:** the automated and manual checks in NFR-A11Y-002 through NFR-A11Y-009 together;
there is no single check for the standard as a whole.

### NFR-A11Y-002 Text contrast

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Test |

Text shall have a contrast ratio of at least 4.5:1 against its background, and at least 3:1 for
large text (≥ 24px, or ≥ 19px bold) and for the visual boundaries of UI components (SC 1.4.3,
1.4.11).

**Measured by:** automated axe checks (NFR-A11Y-008) plus a manual check of the design tokens
in `apps/web/src/app/globals.css` against `04-design-specification.md`.

### NFR-A11Y-003 Target size

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Test |

Interactive targets shall be at least 24×24 CSS px (WCAG 2.2 SC 2.5.8), with 44×44 CSS px as
the design standard for primary controls.

**Measured by:** automated axe checks where supported, plus manual review of tappable
components against `04-design-specification.md`.

### NFR-A11Y-004 Focus visibility

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Test |

Every interactive element shall show a visible focus indicator when focused via keyboard, and
that indicator shall not be obscured by other content (SC 2.4.7, 2.4.11).

**Measured by:** automated axe checks plus a manual keyboard-only pass over each screen.

### NFR-A11Y-005 Reflow and text resize

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Test |

Screens shall reflow without loss of content or function at a 320 CSS px viewport width, and
shall support text resize up to 200% without loss of content or function.

**Measured by:** manual check at 320px width and 200% browser zoom on key screens.

### NFR-A11Y-006 Reduced motion

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Not started | Test |

Animations and transitions shall respect the `prefers-reduced-motion` media query, reducing or
removing non-essential motion when set.

**Measured by:** manual check with the OS/browser reduced-motion setting enabled.

### NFR-A11Y-007 Accessible names and form errors

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Test |

Every control shall have an accessible name (visible label, `aria-label`, or equivalent). Form
validation errors shall be announced to assistive technology and programmatically associated
with their field, and no information shall be conveyed by colour alone.

**Measured by:** automated axe checks plus a manual screen-reader pass over each form.

### NFR-A11Y-008 Automated accessibility checks in end-to-end tests

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Test |

Automated axe accessibility checks shall run against key screens as part of the end-to-end
test suite. Blocked by Q-15 (Vitest/Testing Library and Playwright are the chosen tooling, but
no web tests exist yet).

**Measured by:** CI failing when an axe check reports a violation above the configured severity
threshold. No web end-to-end tests exist today (see NFR-MAINT-002).

### NFR-A11Y-009 Manual assistive-technology pass

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Demo |

A manual pass with VoiceOver (iOS) and TalkBack (Android) shall be completed over the critical
flows before public launch.

**Measured by:** a dated checklist of screens tested with each screen reader and the issues
found and resolved.

---

## 9. Compatibility (COMPAT)

### NFR-COMPAT-001 Supported browsers

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Test |

The system shall support iOS Safari 16.4 or later, and the last two versions of Chrome, Edge,
Firefox and Samsung Internet, and Android Chrome. Web push requires the PWA to be installed to
the home screen on iOS (Safari does not support web push in a normal browser tab); all other
functionality works in a normal browser tab without installing.

**Measured by:** manual verification on each listed browser/OS combination before release; no
automated cross-browser test matrix exists yet.

### NFR-COMPAT-002 Minimum viewport width and orientation

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Partial | Test |

The layout shall support a minimum viewport width of 320 CSS px and shall be designed
portrait-first.

**Measured by:** manual check at 320px width on key screens (shared with NFR-A11Y-005). Not
independently re-verified across every screen as part of this document.

### NFR-COMPAT-003 Functions without installation

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Demo |

The app shall be fully usable in a normal browser tab without being installed to the home
screen; installation adds convenience (icon, standalone display) and, on iOS, is required for
push, but is never a gate on core functionality.

**Measured by:** manual check that every core flow works in an uninstalled browser tab.

### NFR-COMPAT-004 Safari storage eviction awareness

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Not started | Analysis |

The system's offline-write design shall account for Safari's eviction of script-writable
storage (including `localStorage`, which holds the paused-mutation queue per DEC-05) for sites
that are not added to the home screen, after about 7 days of no use. Because DEC-05's queue is
also capped at 7 days, a couple who both keeps the tab un-installed and stays away for a week
risks losing both the queue's own expiry and the browser's independent eviction acting first.

Mitigation: while changes are waiting to sync, the pending-changes notice shall ask the user to
reconnect soon, and iOS users shall be encouraged to install the app (installed home-screen apps
are not subject to this eviction).

**Measured by:** analysis, plus a copy review of the pending-changes notice. Not yet mitigated in
product copy.

---

## 10. Offline (OFFL)

### NFR-OFFL-001 Offline page on cold navigation

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Demo |

When the app is opened with no network and no cached page is available, the service worker
shall serve a static, precached `/offline` page at the URL the user requested, rather than a
browser network-error page (DEC-04, ADR 0004).

**Measured by:** manual DevTools "Offline" check on a cold navigation; automated coverage is
part of the unbuilt web end-to-end suite (Blocked by Q-15).

### NFR-OFFL-002 Non-blocking offline indicator

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Inspection |

While the app is open and the network drops, the system shall show a subtle, non-blocking
status indicator, never a full-screen error, and the content already loaded shall remain
usable.

Verified in `apps/web/src/components/layout/network-banner.tsx`: it renders a quiet `Banner`
line ("Offline · Showing your saved content") when offline, and a transient "synced" banner on
reconnect, never a blocking overlay. `networkMode: "offlineFirst"` keeps last-loaded data
visible underneath it.

**Measured by:** manual check confirming the banner never blocks interaction.

### NFR-OFFL-003 Offline writes survive restart for 7 days

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Demo |

A write made while offline (a registered, paused mutation) shall be preserved across an app
restart, per signed-in user, for up to 7 days, then discarded; the writes shall be cleared
immediately on logout, session expiry, and account deletion (DEC-05).

**Measured by:** the persistence mechanism is `apps/web/src/lib/query/persist.ts`
(`@tanstack/react-query-persist-client`, per-user `localStorage` key, 7-day max age). No
automated test exists yet (Blocked by Q-15); verified by manual check per the as-built facts.

### NFR-OFFL-004 Replay order and failure handling

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Test |

Queued offline writes shall replay in the order they were made on reconnect. A replayed write
that fails server-side validation shall surface that failure to the user (for example as a
toast naming what to fix), never be silently discarded.

**Measured by:** an integration test that queues writes offline, reconnects, and asserts both
ordering and that a deliberately invalid queued write produces a visible error, not silent
loss. `apps/web/src/lib/query/offline.ts` / `client.ts` have no explicit handling for a queued
mutation that fails validation on replay beyond the general mutation-error toast, which has not
been confirmed to fire correctly for a *replayed* paused mutation specifically.

### NFR-OFFL-005 Client idempotency keys for queued writes

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Test |

Every non-idempotent write that can be queued offline shall carry a client-generated
idempotency key, so a replay after a dropped response does not create a duplicate (for example
a duplicate journal entry or appreciation note). This shall be in place before any
non-idempotent write is allowed into the offline queue.

**Measured by:** an integration test sending the same queued write twice with the same key and
asserting only one record results. No idempotency key mechanism exists in the client or the
API today; naturally-idempotent writes (marking a prayer point prayed, toggling a checklist
item) do not need one because their own uniqueness constraints already make repeats safe.

### NFR-OFFL-006 Conflict rule: last-write-wins

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Inspection |

Where two writes to the same resource conflict, the server shall resolve the conflict by
server-received timestamp: the later write wins. This applies to the eventual non-idempotent
queued writes; idempotent writes (completions, checklist toggles) never conflict by
construction.

**Measured by:** inspection of the relevant service-layer write logic once non-idempotent
queued writes exist. Not yet applicable: no such writes are built.

### NFR-OFFL-007 Online-only actions

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Inspection |

Publishing a prayer week and destructive deletes shall be online-only: the client shall not
queue them while offline, and shall instead disable or explain why the action is unavailable.

**Measured by:** manual check that the publish and delete actions are disabled (not silently
queued) while offline, once those features exist beyond the mock.

### NFR-OFFL-008 Cold offline reading

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Not started | Demo |

Blocked by Q-01. Previously loaded prayer weeks and events should remain readable on a cold
(app-closed) offline launch, per the v2.0 design intent, but ADR 0004 deliberately does not
cache personal data or pages, so this needs a new, explicit mechanism (for example encrypted
per-user IndexedDB storage cleared on logout) and a new ADR before it can be built. Not blocking
MVP: option (a) in Q-01 (no cold offline reading, ship the `/offline` page) is the recommended
position for MVP.

**Measured by:** not applicable until Q-01 is resolved.

---

## 11. Observability (OBS)

### NFR-OBS-001 Structured request logs

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Inspection |

Every API request shall produce one structured JSON log line (text in development) carrying
`request_id` and `client_ip`, plus method, path, status and duration, and no request or
response body.

**Measured by:** this review read `apps/api/internal/platform/middleware/logging.go` in full
and confirmed the fields logged (`method`, `path`, `client_ip`, `status`, `duration_ms`,
`bytes`) and the absence of any body logging.

### NFR-OBS-002 Per-route request and latency metrics

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Inspection |

The API shall expose Prometheus counters and latency histograms per route, labelled by route
pattern rather than raw path, to keep label cardinality bounded.

**Measured by:** inspection of `apps/api/internal/platform/metrics` and the `GET /metrics`
endpoint.

### NFR-OBS-003 Liveness and readiness endpoints

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The API shall expose `GET /healthz` (process is up) and `GET /readyz` (database reachable) as
separate endpoints, so an orchestrator can distinguish "not ready yet" from "dead."

**Measured by:** Go tests in `apps/api/internal/platform/httpx/router_test.go`.

### NFR-OBS-004 Log retention

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Inspection |

Application logs shall be retained for 30 days and never contain content, per the retention
schedule recorded at Q-11 (see NFR-PRIV-008).

**Measured by:** inspection of the log storage/retention configuration once a hosting provider
and log sink are chosen. Not configured yet.

### NFR-OBS-005 Alerting on SLO burn and health signals

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Demo |

The system shall alert on SLO error-budget burn rate, the 5xx response ratio, readiness-check
failures, background-worker lag, and push delivery failure rate, before public launch, each
alert paired with a runbook describing how to respond.

**Measured by:** a dated list of configured alerts and their runbooks. None exist yet; no
alerting system is wired up, and the background worker and push sending are not built (Not
started).

### NFR-OBS-006 First-party, content-free client error reporting

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Not started | Test |

The web client should report unhandled errors to a first-party collector (not a third-party
SDK, consistent with NFR-PRIV-004), sending only technical details (stack, route, build id),
never user content.

**Measured by:** an integration test asserting a thrown error reaches the collector without
including form or page content. Not built.

### NFR-OBS-007 Distributed tracing

| Priority | Release | Status | Verification |
|---|---|---|---|
| Could | Post-MVP | Not started | Inspection |

The system could adopt distributed tracing (for example OpenTelemetry) once there is more than
one service to correlate across; `request_id` covers correlation within the current modular
monolith (`docs/ARCHITECTURE.md` §Deliberately not included).

**Measured by:** not applicable until adopted.

### NFR-OBS-008 Business counters for product metrics

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Analysis |

The system shall expose first-party aggregate counters (Prometheus business counters and/or
SQL aggregates) for the PRD's success metrics, consistent with the "no third-party trackers"
privacy commitment. Blocked by Q-14.

**Measured by:** inspection of the metrics registry for named business counters once the PRD
metrics are finalised. None exist yet beyond the generic HTTP metrics.

---

## 12. Maintainability (MAINT)

### NFR-MAINT-001 CI gates

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Inspection |

Every push to `main` and every pull request shall run, and must pass, the CI pipeline defined
in `.github/workflows/ci.yml`: for the API, `gofmt -l` (fails on unformatted files), `go vet
./...`, `go test -race -count=1 ./...` against a real PostgreSQL 18 service container, and a
build of `cmd/api` and `cmd/migrate`; for the web app, `npm run lint`, `npm run typecheck`, and
`npm run build`; and, depending on both, a Docker image build for each app.

**Measured by:** inspection of `.github/workflows/ci.yml` (read in full for this document) and
the branch-protection configuration requiring these checks.

### NFR-MAINT-002 Go test coverage with race detection

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

Go services and repositories shall be covered by unit tests (in-memory fakes) and integration
tests against a real PostgreSQL instance, and the full suite shall run with the `-race` flag in
CI.

**Measured by:** `go test -race -count=1 ./...` in CI; approximately 106 unit and integration
tests exist today (per the as-built facts), run against `AMORAE_TEST_DATABASE_URL`.

### NFR-MAINT-003 Web unit and end-to-end tests

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Test |

The web app shall have unit tests (Vitest and Testing Library) and end-to-end tests
(Playwright) covering the critical flows: sign up, pair, setter publishes a week, both partners
complete points independently, create and complete an event, offline write replay, and
cross-couple access returning 404 (Q-15).

**Measured by:** CI running these suites and reporting coverage of the listed flows. Today the
web CI job runs only lint, typecheck and build; no unit or end-to-end tests exist.

### NFR-MAINT-004 API contract kept in sync

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Partial | Inspection |

`docs/API.md` and `apps/web/src/lib/api/types.ts` shall describe the same contract, and any
change to one shall be made in the same pull request as the other.

**Measured by:** code review checklist item; no automated contract-diff check exists. Holds by
convention today for the built surface (auth, users); not automatically enforced.

### NFR-MAINT-005 Forward-only migrations

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Inspection |

Database migrations shall be forward-only goose migrations; a shipped migration shall never be
edited, only superseded by a new one. Schema changes that would break a running previous
version shall use an expand/contract pattern for zero-downtime deploys.

**Measured by:** inspection of `apps/api/migrations/` history and code review rejecting edits
to already-merged migration files.

### NFR-MAINT-006 Architecture decision records

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Inspection |

A change with lasting architectural weight shall be recorded as an ADR in `docs/adr/`. Four
exist today (0001 modular monolith, 0002 BFF and opaque sessions, 0003 stdlib-first, 0004 PWA
service worker).

**Measured by:** inspection of `docs/adr/` alongside the change's pull request.

### NFR-MAINT-007 Frontend code conventions

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Partial | Inspection |

Web code should avoid `any`, validate all forms with Zod, keep server state in TanStack Query
rather than ad hoc state, and keep API call logic out of UI components (v2.0 §10).

**Measured by:** `npm run typecheck` (catches some `any` use if `noImplicitAny` is set) and
code review for the rest; no dedicated lint rule enforces every clause. Broadly followed in the
built modules (`docs/ARCHITECTURE.md` §Frontend), per convention rather than automated gate.

---

## 13. Operations (OPS)

### NFR-OPS-001 Environment separation

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Inspection |

Dev, staging and production shall use separate data stores and separate secrets; no
environment shall share a database or a secret value with another.

**Measured by:** inspection of each environment's configuration and secret store. Only local
development exists today; staging and production have not been set up (Q-12).

### NFR-OPS-002 Fail-fast configuration

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The API shall validate its configuration at startup and refuse to start on a missing or
invalid value, rather than starting in a partially-configured state.

**Measured by:** `internal/config` unit tests (`config_test.go`) covering invalid-value cases.

### NFR-OPS-003 Zero-downtime deploys

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Demo |

Deploys shall not cause downtime: migrations shall run before the new application version
starts and shall remain compatible with the previous version until it is fully retired
(expand/contract, NFR-MAINT-005), and the server shall drain in-flight requests on shutdown.

**Measured by:** a staged deploy rehearsal confirming no failed requests during rollout. Not
yet exercised; no staging/production deployment pipeline exists yet. Note: graceful shutdown
itself (`internal/platform/server`, per `docs/ARCHITECTURE.md`) is implemented; the end-to-end
zero-downtime deploy process around it is not yet demonstrated.

### NFR-OPS-004 Rollback within 15 minutes

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Demo |

The team shall be able to roll back to the previous deployed image within 15 minutes of
deciding to do so.

**Measured by:** a timed rollback rehearsal. Not yet exercised; no deployment pipeline exists
yet.

### NFR-OPS-005 Mock mode never enabled in production

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Inspection |

`NEXT_PUBLIC_API_MOCK` shall never be enabled in a production deployment; where a demo
deployment intentionally uses it, `ALLOW_MOCK_AUTH=true` shall be required in addition (see
NFR-SEC-024).

**Measured by:** the same guard covered by NFR-SEC-024, listed here as an operational control
on deployment configuration as well as a security control.

### NFR-OPS-006 Incident severity levels

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Inspection |

The team shall define incident severity levels (SEV1–SEV3) with a response-time target for
each, before public launch.

**Measured by:** a documented incident-response policy naming each severity level's response
target. None exists yet.

### NFR-OPS-007 Backups and restore drill

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Demo |

Backups and restore drills shall meet the RPO/RTO targets and cadence defined in NFR-AVAIL-002
through NFR-AVAIL-004; this entry cross-references those requirements rather than restating
them.

**Measured by:** see NFR-AVAIL-002, NFR-AVAIL-003, NFR-AVAIL-004.

### NFR-OPS-008 Dependency update cadence

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Not started | Inspection |

Dependencies shall be updated at least monthly, and a critical security patch shall be applied
within 7 days of it becoming known.

**Measured by:** inspection of dependency-update history (commit log, or a Dependabot log once
NFR-SEC-019 is in place, which this cadence depends on).

---

## 14. Release gates

The release plan in [01 §8](./01-product-requirements.md) has two gates that depend on this
document. Every requirement stays in force after its gate; the gate only says when it must first
be true.

- **Private beta:** the first time real couples' data is stored in production. Everything that
  protects that data, and everything needed to recover it, must be in place.
- **Public launch:** everything needed to operate at scale for strangers, with a verified
  compliance and accessibility posture.

**Should** and **Could** requirements do not block either gate (MoSCoW), except NFR-SEC-015,
which becomes Must at public launch.

| Category | Private beta gate | Public launch gate (additional) |
|---|---|---|
| PERF | 003, 004 | 001 |
| AVAIL | 002, 003, 005 | 001, 004 |
| SCALE | 001, 002, 003, 005 | — |
| SEC | 002, 003, 004, 005, 006, 007, 010, 011, 012, 013, 014, 016, 017, 018, 020, 022, 024, 025 | 001, 015, 023 |
| PRIV | 002, 003, 004, 006, 009, 011, 012, 014 | 001, 005, 007, 008, 010, 013 |
| A11Y | 002, 003, 004, 007 | 001, 005, 008, 009 |
| COMPAT | 001, 002, 003 | — |
| OFFL | 001, 002, 003, 004, 005, 006, 007 | — |
| OBS | 001, 002, 003, 004 | 005, 008 |
| MAINT | 001, 002, 003, 004, 005, 006 | — |
| OPS | 001, 002, 004, 005, 007 | 003, 006 |

Read a cell as "`NFR-<CATEGORY>-<NNN>` for each number listed". Some reasoning behind the split:

- Idempotency keys (NFR-OFFL-005) gate private beta because that is when queued writes first
  reach the real API; without them a dropped response can duplicate a journal entry.
- Backups and a restore procedure (NFR-AVAIL-002, -003) gate private beta; the recurring drill
  cadence (NFR-AVAIL-004) and the availability SLO (NFR-AVAIL-001) gate public launch, when
  there is enough traffic to measure.
- Account deletion (NFR-PRIV-006) gates private beta because beta users must be able to leave;
  export, retention and the sub-processor list gate public launch, matching Q-11 and Q-12.
- The full WCAG 2.2 AA audit, penetration test, alerting and incident levels gate public launch,
  as the PRD requires.

As of 2026-09-22 most of these are Not started or Partial, so this table is a checklist to close,
not a report of compliance. §2 gives the current state.

---

## 15. Traceability

Source column key: a `DEC-NN`/`Q-NN` citation points to [05](./05-decisions-and-open-questions.md);
a section reference like "v2.0 §7" points to the security/PWA/NFR/testing/deployment/scalability
sections of the v2.0 engineering spec, which this document supersedes (it remains in git
history); "01 principle"
points to the core principles in [01](./01-product-requirements.md) §6; a named standard
(OWASP ASVS 5.0, WCAG 2.2, NDPA 2023, GDPR) is the regulation or standard itself; "new" means
this document introduces the requirement with no v2.0 precedent.

| NFR ID | Source | Verification |
|---|---|---|
| NFR-PERF-001 | v2.0 §10 (LCP target, extended to INP/CLS); Q-14 | Test |
| NFR-PERF-002 | v2.0 §10 (initial JS budget) | Test |
| NFR-PERF-003 | v2.0 §10 (API reads p95) | Analysis |
| NFR-PERF-004 | v2.0 §10 (extended to writes) | Analysis |
| NFR-PERF-005 | ADR 0004 | Demo |
| NFR-AVAIL-001 | new | Analysis |
| NFR-AVAIL-002 | v2.0 §10 (backups) | Analysis |
| NFR-AVAIL-003 | v2.0 §10 (recovery) | Demo |
| NFR-AVAIL-004 | new | Demo |
| NFR-AVAIL-005 | DEC-15; ARCHITECTURE (module independence) | Test |
| NFR-SCALE-001 | v2.0 §13; ARCHITECTURE (BFF) | Inspection |
| NFR-SCALE-002 | DEC-11; Q-13 | Inspection |
| NFR-SCALE-003 | v2.0 §13 (couple scoping and indexes) | Inspection |
| NFR-SCALE-004 | v2.0 §13 | Analysis |
| NFR-SCALE-005 | ARCHITECTURE (scaling path) | Inspection |
| NFR-SEC-001 | OWASP ASVS 5.0 Level 2 | Inspection |
| NFR-SEC-002 | new | Inspection |
| NFR-SEC-003 | DEC-06 | Test |
| NFR-SEC-004 | v2.0 §7 (password hygiene); new | Test |
| NFR-SEC-005 | DEC-06; Q-02 | Inspection |
| NFR-SEC-006 | DEC-11 | Test |
| NFR-SEC-007 | DEC-02 | Test |
| NFR-SEC-008 | Q-03 | Test |
| NFR-SEC-009 | Q-03 | Test |
| NFR-SEC-010 | DEC-07; Q-03 | Test |
| NFR-SEC-011 | DEC-19; v2.0 §7 | Test |
| NFR-SEC-012 | v2.0 §7 (input validation) | Test |
| NFR-SEC-013 | DEC-09 | Inspection |
| NFR-SEC-014 | v2.0 §7 (security controls) | Inspection |
| NFR-SEC-015 | v2.0 §9 (PWA/pages) | Inspection |
| NFR-SEC-016 | v2.0 §7 (transport security) | Inspection |
| NFR-SEC-017 | v2.0 §7 (encryption at rest) | Inspection |
| NFR-SEC-018 | v2.0 §7 (secrets) | Inspection |
| NFR-SEC-019 | new | Inspection |
| NFR-SEC-020 | v2.0 §7 (audit log) | Test |
| NFR-SEC-021 | v2.0 §7 (rate limiting) | Test |
| NFR-SEC-022 | DEC-08; Q-08 | Test |
| NFR-SEC-023 | new | Demo |
| NFR-SEC-024 | ARCHITECTURE (mock mode guard) | Inspection |
| NFR-SEC-025 | ARCHITECTURE (metrics listener) | Inspection |
| NFR-PRIV-001 | NDPA 2023; GDPR; Q-17 | Inspection |
| NFR-PRIV-002 | NDPA 2023; GDPR Art. 9; Q-17 | Demo |
| NFR-PRIV-003 | 01 principle "Private by default" | Inspection |
| NFR-PRIV-004 | 01 principle; Privacy page | Inspection |
| NFR-PRIV-005 | NDPA 2023; GDPR (data-subject rights) | Test |
| NFR-PRIV-006 | NDPA 2023; GDPR; Q-09 | Test |
| NFR-PRIV-007 | NDPA 2023; GDPR (rectification) | Test |
| NFR-PRIV-008 | Q-11 | Inspection |
| NFR-PRIV-009 | Q-09 | Demo |
| NFR-PRIV-010 | Q-12 | Inspection |
| NFR-PRIV-011 | 01 principle "Private by default" | Inspection |
| NFR-PRIV-012 | NDPA 2023; GDPR (breach notification) | Inspection |
| NFR-PRIV-013 | Q-10 | Demo |
| NFR-PRIV-014 | 01 principle "A two-person experience" | Inspection |
| NFR-A11Y-001 | WCAG 2.2 Level AA | Test |
| NFR-A11Y-002 | WCAG 2.2 SC 1.4.3 / 1.4.11 | Test |
| NFR-A11Y-003 | WCAG 2.2 SC 2.5.8 | Test |
| NFR-A11Y-004 | WCAG 2.2 SC 2.4.7 / 2.4.11 | Test |
| NFR-A11Y-005 | WCAG 2.2 (reflow, text resize) | Test |
| NFR-A11Y-006 | WCAG 2.2 (reduced motion) | Test |
| NFR-A11Y-007 | WCAG 2.2 (accessible name, error identification) | Test |
| NFR-A11Y-008 | Q-15 | Test |
| NFR-A11Y-009 | WCAG 2.2 Level AA; new (launch practice) | Demo |
| NFR-COMPAT-001 | v2.0 §9 (browser targets) | Test |
| NFR-COMPAT-002 | v2.0 §9; 04 design specification | Test |
| NFR-COMPAT-003 | 01; ARCHITECTURE | Demo |
| NFR-COMPAT-004 | ADR 0004; DEC-05 | Analysis |
| NFR-OFFL-001 | DEC-04; ADR 0004 | Demo |
| NFR-OFFL-002 | v2.0 §9.1 | Inspection |
| NFR-OFFL-003 | DEC-05 | Demo |
| NFR-OFFL-004 | v2.0 §9.1 (sharpened) | Test |
| NFR-OFFL-005 | v2.0 §9.1 (idempotency key) | Test |
| NFR-OFFL-006 | v2.0 §9.1 (conflict rule) | Inspection |
| NFR-OFFL-007 | v2.0 §9.1 (online-only actions) | Inspection |
| NFR-OFFL-008 | Q-01; ADR 0004 | Demo |
| NFR-OBS-001 | ARCHITECTURE (observability) | Inspection |
| NFR-OBS-002 | ARCHITECTURE (observability) | Inspection |
| NFR-OBS-003 | ARCHITECTURE (observability) | Test |
| NFR-OBS-004 | Q-11 | Inspection |
| NFR-OBS-005 | new | Demo |
| NFR-OBS-006 | new; 01 principle (privacy) | Test |
| NFR-OBS-007 | ARCHITECTURE (deliberately not included) | Inspection |
| NFR-OBS-008 | Q-14; 01 PRD success metrics | Analysis |
| NFR-MAINT-001 | .github/workflows/ci.yml | Inspection |
| NFR-MAINT-002 | v2.0 §11 (testing) | Test |
| NFR-MAINT-003 | Q-15; v2.0 §11 | Test |
| NFR-MAINT-004 | README §4 (definition of done) | Inspection |
| NFR-MAINT-005 | v2.0 §12 (migrations) | Inspection |
| NFR-MAINT-006 | README §5 (change control) | Inspection |
| NFR-MAINT-007 | v2.0 §10 (frontend guidelines) | Inspection |
| NFR-OPS-001 | v2.0 §12 (environment and config) | Inspection |
| NFR-OPS-002 | ARCHITECTURE (config fails fast) | Test |
| NFR-OPS-003 | v2.0 §12 (deployment); NFR-MAINT-005 | Demo |
| NFR-OPS-004 | new | Demo |
| NFR-OPS-005 | ARCHITECTURE (mock mode guard) | Inspection |
| NFR-OPS-006 | new | Inspection |
| NFR-OPS-007 | NFR-AVAIL-002 / 003 / 004 (cross-reference) | Demo |
| NFR-OPS-008 | new | Inspection |

---

## Change log

| Version | Date | Change |
|---------|------|--------|
| 3.0 | 2026-09-22 | New document. Collects the v2.0 security, PWA, NFR, testing, deployment and scalability sections, adds measurable targets, compliance, accessibility, observability and operations requirements, and records current implementation status. |
