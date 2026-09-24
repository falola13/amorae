# Amorae: Functional Requirements (Software Requirements Specification)

| | |
|---|---|
| **Document ID** | AMR-REQ-02 |
| **Version** | 3.0 |
| **Status** | Draft for review |
| **Owner** | Engineering |
| **Last updated** | 2026-09-22 |

---

## 1. Introduction

### 1.1 Purpose

This document specifies what the Amorae system shall do: every functional requirement (`FR`),
the business rules several requirements share (`BR`), the data the system holds, the external
interfaces it exposes and consumes, and the traceability between requirements, goals and the
API. It is written to the structure and quality bar of ISO/IEC/IEEE 29148 (a Software
Requirements Specification): each requirement is necessary, singular, unambiguous, verifiable,
feasible, and traceable to a product goal.

This is a **rewrite**, not an update, of the v2.0 "Engineering Specification" that previously
lived at this path. v2.0 was written before most of the code existed. This version keeps every
substantive functional requirement, business rule, data model detail, scheduler rule and AI rule
that v2.0 defined, and reconciles all of it against the system as actually built, as verified on
2026-09-22. Where the code and v2.0 disagree, the disagreement is recorded, not silently resolved
in either direction (per the definition of done in [`README.md`](./README.md) §4).

### 1.2 Scope

In scope: functional behaviour of the Amorae PWA (Next.js BFF) and its Go API, module by module,
for MVP and the Post-MVP AI phase. Out of scope, and covered elsewhere: *why* the product exists
and its release plan ([01](./01-product-requirements.md)); performance, security, privacy,
accessibility, compatibility, offline and operational targets
([03](./03-non-functional-requirements.md)); visual design, tokens and the screen inventory
([04](./04-design-specification.md)); the register of decisions and open questions
([05](./05-decisions-and-open-questions.md)); *how* the system is built, module boundaries and
request lifecycle ([`../ARCHITECTURE.md`](../ARCHITECTURE.md)); the HTTP contract itself
([`../API.md`](../API.md)); and architecture decision records ([`../adr/`](../adr/)).

### 1.3 Audience

Engineering (implementation and test authoring), Product (acceptance and prioritisation), and
Design (cross-checking behaviour against [04](./04-design-specification.md)). Anyone citing a
requirement elsewhere should cite its permanent ID, not a section number.

### 1.4 Conventions

This document follows the identifier scheme, requirement-writing rules (keywords, EARS patterns,
the quality bar), attribute set (Priority, Release, Implementation status, Verification) and
definition of done set out in [`README.md`](./README.md) §2–§4. In short:

- **Shall** is mandatory, **should** is recommended, **may** is optional.
- Every requirement carries Priority (Must/Should/Could/Won't), Release (MVP/Post-MVP),
  Implementation status (Implemented/Partial/UI only/Not started) and Verification
  (Test/Demo/Inspection/Analysis).
- Acceptance criteria are Given/When/Then, numbered `.AC1`, `.AC2`, … under their requirement.
- A requirement that cannot proceed until a `Q-NN` is answered says so: *Blocked by Q-NN*.
- IDs (`FR-<MODULE>-NNN`, `BR-<MODULE>-NN`) are permanent; a withdrawn requirement keeps its ID
  with status **Withdrawn** and a reason, never reused or renumbered.

### 1.5 References

- [`./01-product-requirements.md`](./01-product-requirements.md) — vision, pillars, principles, MVP scope, roadmap
- [`./03-non-functional-requirements.md`](./03-non-functional-requirements.md) — performance, security, privacy, accessibility, offline and operational targets
- [`./04-design-specification.md`](./04-design-specification.md) — tokens, components, screen inventory
- [`./05-decisions-and-open-questions.md`](./05-decisions-and-open-questions.md) — the single `DEC`/`Q` register
- [`../ARCHITECTURE.md`](../ARCHITECTURE.md) — module layout, request lifecycle, rate limiting, scaling path
- [`../API.md`](../API.md) — the HTTP contract: conventions, envelopes, error codes, built and proposed endpoints
- [`../adr/`](../adr/) — architecture decision records

---

## 2. Overall description

### 2.1 Product perspective

Amorae is a new, standalone product; it does not integrate with or replace another system. It is
a mobile-first Progressive Web App used by exactly two people — a couple — who share one private
space. There is no public surface: nothing is visible to anyone outside the couple.

System context:

```text
   Browser (PWA, or installed to the home screen)
        |  HTTPS; httpOnly session cookie "amorae_session" (DEC-02)
        v
   Next.js BFF  —  same-origin /api/v1/*                    (DEC-02, DEC-10)
        |  Authorization: Bearer <opaque token>, added server-side
        v
   Go API  —  /v1/*                                          (DEC-01, DEC-03)
        |
        v
   PostgreSQL  (the only stateful dependency)                (DEC-20)
        ^
        |  background worker (cmd/worker): hourly tick, reads/writes the
        |  same Postgres, no separate queue or cache                (Q-16)
        |
        +--> Web Push services (VAPID)        — reminders, alerts    (Q-16, FR-NOTF)
        +--> Email provider                   — reset, verification  (Q-04)
        +--> Object storage (S3-compatible)    — memory photos        (Q-06)
        +--> AI provider (Post-MVP, optional)  — behind an interface  (Q-18, DEC-15)
```

The browser never calls the Go API directly (DEC-02): every request the client makes goes to the
Next.js BFF at `/api/v1/*`, which proxies it to the Go API's `/v1/*` and refuses any path outside
`/v1` (DEC-10). *How* each of these pieces is built — the module layout, the request lifecycle
through the BFF and the API, and the scaling path — is [`../ARCHITECTURE.md`](../ARCHITECTURE.md),
not repeated here.

### 2.2 User classes

| Class | Description |
|---|---|
| **Visitor** | Not signed in. Can read the marketing/welcome screens, Terms and Privacy pages, and register or log in. |
| **Unpaired user** | Signed in, no couple yet. Can manage their own profile and create or join a couple. |
| **Partner** | Signed in and a member of a paired couple. The two partners are symmetric: there is no "partner A"/"partner B" distinction, no owner role, and no elevated permissions between them. |
| **This week's setter** | A temporal role, not an account type: whichever partner is due to set prayer points this week, computed by BR-PRAY-01. Both partners hold this role in alternating weeks. |

### 2.3 Design constraints

These decisions from [05](./05-decisions-and-open-questions.md) bound every requirement below:

- **DEC-01** — the API is a single Go binary, a modular monolith; feature modules depend only on
  interfaces they declare, wired together in one composition root. No requirement here should
  assume a separately deployable service per feature.
- **DEC-02** — the browser never calls Go directly; sessions are opaque server-side tokens behind
  the Next.js BFF, not JWTs, with no refresh endpoint.
- **DEC-03** — the API is built stdlib-first (router, hand-written SQL over pgx, goose migrations,
  stdlib testing); requirements should not presuppose a specific ORM or query-generation tool.
- **DEC-10** — the Go API is served under `/v1`; the browser only ever calls same-origin
  `/api/v1/*`.
- **DEC-20** — there is no Redis, cache or queue in MVP; Postgres is the only stateful dependency,
  including for the background worker (Q-16) and rate limiting (Q-13).

### 2.4 Assumptions and dependencies

Assumptions, constraints and dependencies are stated once, in
[01-product-requirements.md](./01-product-requirements.md) §9, alongside the product principles
(§6) and MVP scope (§7), and are not repeated here. Every functional requirement in this
document assumes those principles hold, in particular: private by default, a two-person
experience, and a product that works fully with AI disabled (01 §6).

---

## 3. Functional requirements

One subsection per module, in the order `AUTH`, `ACCT`, `PAIR`, `PRAY`, `EVT`, `CAL`, `GOAL`,
`CHAL`, `JRNL`, `APPR`, `MEM`, `DATE`, `NOTF`, `PWA`, `AI`. Each module opens with a short
description, the pillar(s) it belongs to, the product goal(s) it serves (from
[01](./01-product-requirements.md)'s five: **G-01** Activation/pairing, **G-02** Faith rhythm,
**G-03** Shared life planning/growth, **G-04** Connection and memory, **G-05**
Trust/privacy/reliability), and the `DEC`/`Q` entries that shape it. Requirements are numbered
consecutively within the module, starting at 001.

### 3.1 AUTH — Authentication

Sign-up, sign-in, session management and account recovery. Cross-cutting: every other module
depends on it, but it is not one of the five product pillars itself.

**Goals:** G-01 (Activation/pairing), G-05 (Trust/privacy/reliability)
**Related:** DEC-02, DEC-06, DEC-11, DEC-19; Q-02, Q-03, Q-04, Q-05, Q-10, Q-20

**Business rules**

- **BR-AUTH-01** — Password policy: at least 10 characters, counted as Unicode code points, no
  composition rules, maximum 72 bytes while bcrypt is the hash (DEC-06). Applied identically by
  the web client and the API. Q-02 tracks a possible move to argon2id, which would remove the
  72-byte cap.
- **BR-AUTH-02** — Login throttling: 10 attempts per email per 15 minutes, checked before any
  password hashing and applied whether or not the account exists, plus 20 requests per client IP
  per minute across `/v1/auth/*` (DEC-11). Both return 429 with `Retry-After`.
- **BR-AUTH-03** — Authentication failures never reveal whether an account exists: a generic
  `invalid_credentials` error and equalised response timing (a dummy bcrypt comparison runs for
  an unknown email) make a wrong password and an unknown email indistinguishable.

#### FR-AUTH-001 Register

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let a visitor register a new account with an email address, a password and a
display name.

**Acceptance criteria**

- **FR-AUTH-001.AC1** Given a visitor submits a unique, valid email, a password meeting
  BR-AUTH-01, and a display name of 1–50 trimmed characters, when they submit registration, then
  the API shall create the account, start a session, and respond 201 with the new user and an
  opaque session token.
- **FR-AUTH-001.AC2** Given the email is already registered, when the visitor submits, then the
  API shall respond 409 `email_taken` and shall not create a second account.
- **FR-AUTH-001.AC3** Given the email, password or display name fails validation, when the
  visitor submits, then the API shall respond 400 `validation_failed` with a per-field message
  and shall not create an account.

#### FR-AUTH-002 Login

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let a registered user start a session with their email and password.

**Acceptance criteria**

- **FR-AUTH-002.AC1** Given correct credentials for an existing account, when the user submits
  them, then the API shall respond 200 with a new opaque session token and the user's profile.
- **FR-AUTH-002.AC2** Given an unknown email or an incorrect password, when the user submits
  credentials, then the API shall respond 401 `invalid_credentials` per BR-AUTH-03, so a caller
  cannot tell which was wrong or whether the account exists.

#### FR-AUTH-003 Logout

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let a signed-in user end their session on request.

**Acceptance criteria**

- **FR-AUTH-003.AC1** Given a valid session, when the user logs out, then the API shall delete
  the session row and respond 204.
- **FR-AUTH-003.AC2** Given a session that no longer exists (already logged out, expired, or
  never valid), when logout is called with its token, then the API shall still respond 204
  (idempotent) rather than an error.

#### FR-AUTH-004 Absolute session lifetime

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall expire every session no later than 30 days after it was created (DEC-02), with
no refresh endpoint.

**Acceptance criteria**

- **FR-AUTH-004.AC1** Given a session older than 30 days, when it is used on any authenticated
  endpoint, then the API shall respond 401 `unauthenticated` and the client shall route the user
  to sign in again.

#### FR-AUTH-005 Idle session expiry

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Not started — Blocked by Q-03 | Test |

The system shall also expire a session after 14 days of inactivity, independent of the 30-day
absolute limit (Q-03 recommendation).

**Acceptance criteria**

- **FR-AUTH-005.AC1** Given a session with no use for 14 days, when it is next used, then the API
  shall respond 401 `unauthenticated` even though the absolute lifetime has not elapsed.

#### FR-AUTH-006 Password policy enforcement

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall reject a password that does not meet BR-AUTH-01 wherever a password is set.

**Acceptance criteria**

- **FR-AUTH-006.AC1** Given a password under 10 Unicode code points, when it is submitted at
  registration, then the API shall respond 400 `validation_failed` with `fields.password`.
- **FR-AUTH-006.AC2** Given a password whose UTF-8 encoding exceeds 72 bytes, when it is
  submitted, then the API shall respond 400 `validation_failed` with `fields.password` (Q-02
  tracks removing this cap).

#### FR-AUTH-007 Login throttling

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall rate-limit authentication attempts per BR-AUTH-02.

**Acceptance criteria**

- **FR-AUTH-007.AC1** Given 10 failed login attempts for one email within 15 minutes, when an
  11th attempt is made, then the API shall respond 429 `rate_limited` with `Retry-After`, whether
  or not the account exists.
- **FR-AUTH-007.AC2** Given 20 requests from one client IP to any `/v1/auth/*` endpoint within one
  minute, when a further request is made, then the API shall respond 429 `rate_limited`.

#### FR-AUTH-008 Password reset

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let a user reset a forgotten password through a time-limited emailed link.

**Acceptance criteria**

- **FR-AUTH-008.AC1** Given a user requests a reset for a registered email, when the request is
  submitted, then the system shall send a single-use, time-limited reset link and shall not
  reveal whether the email is registered.
- **FR-AUTH-008.AC2** Given a valid, unused, unexpired reset link, when the user sets a new
  password meeting BR-AUTH-01, then the API shall update the password hash and invalidate every
  existing session for that user.

#### FR-AUTH-009 Email verification

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Not started — Blocked by Q-05 | Test |

The system shall send a verification email on registration and shall require a verified address
before it can be used for password reset (Q-05).

**Acceptance criteria**

- **FR-AUTH-009.AC1** Given a new account, when registration completes, then the system shall
  send a verification email without blocking use of the app.
- **FR-AUTH-009.AC2** Given an unverified email, when the user requests a password reset, then
  the system shall decline and prompt the user to verify the address first.

#### FR-AUTH-010 Minimum age confirmation at sign-up

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started — Blocked by Q-10 | Test |

The system shall require a visitor to confirm they are at least 18 years old before an account is
created (Q-10).

**Acceptance criteria**

- **FR-AUTH-010.AC1** Given the age-confirmation control is not checked, when the visitor submits
  registration, then the client shall block submission, and the API shall reject the request if
  it is sent anyway.

#### FR-AUTH-011 Acceptance of Terms and Privacy Policy at sign-up

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Partial | Demo |

The system shall record a visitor's acceptance of the Terms of Service and Privacy Policy as part
of registration.

**Acceptance criteria**

- **FR-AUTH-011.AC1** Given the registration screen, when it is displayed, then it shall show a
  "by continuing you agree" statement linking to `/terms` and `/privacy`, both opening in a new
  tab so filled-in fields are not lost.
- **FR-AUTH-011.AC2** Given a visitor completes registration, when the account is created, then
  the system shall record that acceptance (who, when, which document version) as a distinct,
  queryable event, not only inferred from having pressed Continue.

*Status note:* only AC1 exists today — the linked copy on the registration screen. AC2 (a
recorded acceptance event) is not built, hence **Partial** rather than **Implemented**.

#### FR-AUTH-012 Explicit consent for faith content

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Partial — captured, not yet acted on | Test |

The system shall obtain a user's explicit consent to process faith content, separate from
acceptance of the Terms, before that content can be stored (NFR-PRIV; where it is asked was
Q-20, what it controls is DEC-29).

*Status note:* the consent is asked for at sign-up as its own unticked box, and recorded with
its policy version — AC1 and AC2 are met. Nothing reads it: a user who leaves the box unticked
still sees prayer weeks, scripture fields and "Two hearts, one faith". Until FR-PAIR-009 is
built, this consent is a promise the product does not keep.

**Acceptance criteria**

- **FR-AUTH-012.AC1** Given the consent control is presented unticked, when the user does not
  tick it, then the system shall not treat Terms acceptance as consent to process faith content.
- **FR-AUTH-012.AC2** Given the user consents, when consent is recorded, then the system shall
  store who consented, when, and which version of the privacy policy they saw.
- **FR-AUTH-012.AC3** Given a user has consented, when they withdraw that consent from their own
  settings, then the system shall record the withdrawal and shall stop storing faith content for
  the space at once (FR-PAIR-009.AC4).
- **FR-AUTH-012.AC4** Given consent is asked for anywhere other than sign-up, when it is asked,
  then it shall be the person's own affirmative act; no partner, and no acceptance of the Terms,
  shall ever grant it on their behalf.

### 3.2 ACCT — Account

Profile, credential changes, deletion and export. The user's relationship with Amorae as a
service, separate from their relationship with their partner.

**Goals:** G-05 (Trust/privacy/reliability), G-01 (Activation/pairing)
**Related:** DEC-05, DEC-07; Q-03, Q-05, Q-09, Q-11, Q-19

#### FR-ACCT-001 View and edit profile

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let a signed-in user view and edit their display name and timezone.

**Acceptance criteria**

- **FR-ACCT-001.AC1** Given a signed-in user, when they request their profile, then the API shall
  return `display_name`, `timezone`, and the read-only fields (`id`, `email`, `created_at`,
  `updated_at`).
- **FR-ACCT-001.AC2** Given a display name of 1–50 trimmed characters, when the user saves it,
  then the API shall update it and return the updated profile.
- **FR-ACCT-001.AC3** Given a timezone that is not a valid IANA zone name, when the user attempts
  to save it, then the API shall respond 400 `validation_failed` with `fields.timezone` and leave
  the stored value unchanged.

#### FR-ACCT-002 Change sign-in email with current password

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let a signed-in user change the account's email address by re-entering their
current password (DEC-07).

**Acceptance criteria**

- **FR-ACCT-002.AC1** Given the correct current password and a new email not already in use, when
  the user submits the change, then the API shall update the email and return the updated
  profile.
- **FR-ACCT-002.AC2** Given an incorrect current password, when the change is submitted, then the
  API shall respond 400 `validation_failed` with `fields.current_password`, never 401 (DEC-07),
  so the client does not mistake it for session expiry.
- **FR-ACCT-002.AC3** Given a new email already registered to another account, when the change is
  submitted, then the API shall respond 409 `email_taken`.

#### FR-ACCT-003 Two-step email change confirmation and old-address notice

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Not started — Blocked by Q-05 | Test |

The system shall require confirmation of a new email address before it takes effect, and shall
notify the old address when a change is made (Q-05).

**Acceptance criteria**

- **FR-ACCT-003.AC1** Given a user submits an email change, when the request succeeds, then the
  system shall email a confirmation link to the new address and shall not switch the sign-in
  email until that link is used.
- **FR-ACCT-003.AC2** Given an email change is confirmed, when it takes effect, then the system
  shall also send a notice to the previous address.

#### FR-ACCT-004 Change password with current password

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Test |

The system shall let a signed-in user change their password by re-entering their current
password (Q-03 recommendation, matching DEC-07's email-change pattern).

**Acceptance criteria**

- **FR-ACCT-004.AC1** Given the correct current password and a new password meeting BR-AUTH-01,
  when the user submits the change, then the API shall update the password hash and invalidate
  every other active session for that user.
- **FR-ACCT-004.AC2** Given an incorrect current password, when the change is submitted, then the
  API shall respond 400 `validation_failed` with `fields.current_password`.

#### FR-ACCT-005 Account deletion with re-authentication

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | UI only | Test |

The system shall let a signed-in user permanently delete their account after re-authenticating
(Q-03 recommendation).

**Acceptance criteria**

- **FR-ACCT-005.AC1** Given a signed-in user requests deletion and confirms the in-app warning,
  when the request is submitted, then the system shall remove the user from their couple, delete
  their authored content per the retention rule in §4(c), and respond 204.
- **FR-ACCT-005.AC2** Given account deletion is requested, when the confirmation is shown, then
  the client shall require the current password before submitting, in addition to the
  confirmation dialog.

*Status note:* the screen and its confirmation dialog exist; no Go endpoint does. AC2's password
re-entry is not built either, and is recorded here rather than as a separate requirement.

#### FR-ACCT-006 Data export

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Test |

The system shall let a signed-in user export their personal and couple-shared content in a
machine-readable format.

**Acceptance criteria**

- **FR-ACCT-006.AC1** Given a user requests an export, when it is generated, then the system
  shall produce a JSON file containing every entity the user owns or co-owns, using the same
  field names as the API contract.
- **FR-ACCT-006.AC2** Given an export is requested, when it is delivered, then it shall exclude
  the other partner's account credentials and include only content the requesting user is
  entitled to see.

#### FR-ACCT-007 Sign-out clears local signed-in state

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Demo |

The client shall remove all locally held signed-in state, including any queued offline changes,
when a user signs out.

**Acceptance criteria**

- **FR-ACCT-007.AC1** Given a user signs out, when sign-out completes, then the client shall
  clear the in-memory query cache and delete any saved offline changes for that user from local
  storage (DEC-05).
- **FR-ACCT-007.AC2** Given a user signs out on a shared device, when they or someone else next
  opens the app, then no content or queued write from the previous session shall be visible or
  resumable.

#### FR-ACCT-008 Active sessions and sign out other devices

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Implemented | Test |

The system shall let a signed-in user see their own active sessions and end every session other
than the current one (DEC-23).

**Acceptance criteria**

- **FR-ACCT-008.AC1** Given a user with sessions on several devices, when they open their account
  security settings, then the system shall list each of their own active sessions with its
  creation time and last use, and never show the partner's sessions.
- **FR-ACCT-008.AC2** Given the user chooses "sign out other devices", when it completes, then
  every session except the current one shall be deleted and shall respond 401 on next use.
- **FR-ACCT-008.AC3** Given a session list, when it is returned, then it shall carry no token
  hash, no raw user agent and no IP address — only a coarse device label — and it shall never
  include the partner's sessions.
- **FR-ACCT-008.AC4** Given an active session, when it is used repeatedly, then its last-used
  time shall be written at most once an hour.

---

### 3.3 PAIR — Couple pairing

Creating a couple, inviting a partner, joining, and the couple's own shared profile. The gateway
to every other module: nothing couple-scoped exists until two people are paired.

**Goals:** G-01 (Activation/pairing), G-05 (Trust/privacy/reliability)
**Related:** DEC-08, DEC-12, DEC-19; Q-07, Q-08, Q-09

**Business rules**

- **BR-PAIR-01** — A couple has exactly two members, and a user belongs to at most one couple in
  MVP (DEC-12); enforced by `unique(couple_id, user_id)` and `unique(user_id)`.
- **BR-PAIR-02** — An invite code is 3 letters and 3 digits, stored as `ABC123` and shown as
  `ABC-123` (DEC-08).
- **BR-PAIR-03** — An invite code expires 7 days after issue; a couple has one active code at a
  time, and regenerating a code invalidates the previous one (Q-08 recommendation).
- **BR-PAIR-04** — A request for a resource belonging to another couple returns 404, never 403
  (DEC-19). This rule is shared by every couple-scoped module (§3.5–§3.12).

#### FR-PAIR-001 Create couple

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Partial | Test |

The system shall let an unpaired user create a couple, becoming its first member, and issue a
fresh invite code.

**Acceptance criteria**

- **FR-PAIR-001.AC1** Given an unpaired user, when they create a couple, then the API shall
  respond 201 with the new couple (the caller as its only member so far) and an invite code valid
  per BR-PAIR-02/BR-PAIR-03.
- **FR-PAIR-001.AC2** Given a user who already belongs to a couple, when they attempt to create
  another, then the API shall reject the request (BR-PAIR-01).

#### FR-PAIR-002 Share invite code and link

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Partial | Demo |

The system shall present the invite code and a shareable join link on the couple's waiting screen
until a partner joins.

**Acceptance criteria**

- **FR-PAIR-002.AC1** Given a couple with one member, when that member views their couple screen,
  then the client shall display the invite code formatted `ABC-123` and a link that pre-fills the
  code on the join screen.

#### FR-PAIR-003 Join couple by code

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Partial | Test |

The system shall let a second, unpaired user join an existing couple by submitting its invite
code.

**Acceptance criteria**

- **FR-PAIR-003.AC1** Given a valid, unexpired code for a couple with one member, when a
  different unpaired user submits it, then the API shall add them as the second member and
  respond 200 with the couple, now showing both partners.
- **FR-PAIR-003.AC2** Given a code that does not match any active invite, when it is submitted,
  then the API shall respond 400 `validation_failed` with `fields.code`.
- **FR-PAIR-003.AC3** Given a code for a couple that already has two members, when it is
  submitted, then the API shall respond 409 without adding a third member.

#### FR-PAIR-004 Invite expiry and attempt limits

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started — Blocked by Q-08 | Test |

The system shall expire an invite code and limit join attempts per BR-PAIR-03.

**Acceptance criteria**

- **FR-PAIR-004.AC1** Given a code more than 7 days old, when it is submitted, then the API shall
  reject it as expired even if it was never used.
- **FR-PAIR-004.AC2** Given repeated incorrect join attempts from one user or one IP, when a
  threshold is crossed, then the API shall rate-limit further attempts.

#### FR-PAIR-005 Edit couple profile

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Partial | Test |

The system shall let either partner set the couple's display name and relationship start date.

**Acceptance criteria**

- **FR-PAIR-005.AC1** Given a paired couple, when either partner submits a name and/or a start
  date in `YYYY-MM-DD` form, then the API shall update the couple record and return it to both
  partners.

*Contract note:* the in-progress Go couples module registers `PATCH /v1/couples/me`, hence
**Partial**. No screen calls it yet.

#### FR-PAIR-006 Set own role label

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Partial | Test |

The system shall let a partner set their own 1–32 character role label (for example "Husband"),
shown to their partner but carrying no permission.

**Acceptance criteria**

- **FR-PAIR-006.AC1** Given a signed-in partner, when they submit a label of 1–32 characters,
  then the API shall store it against their membership and it shall appear on their partner's
  couple screen.

*Contract note:* as with FR-PAIR-005, `PATCH /v1/couples/role` is registered in the in-progress
Go couples module, with no screen calling it yet.

#### FR-PAIR-007 Onboarding flags

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Partial | Test |

The system shall track, per person, whether they have completed the install and
notification-permission onboarding steps, and derive the couple step from membership itself.

**Acceptance criteria**

- **FR-PAIR-007.AC1** Given a signed-in partner, when they complete the install or notifications
  step, then the API shall persist that flag against their own membership row, independent of
  their partner's progress.
- **FR-PAIR-007.AC2** Given a request sets the couple flag directly, when it is processed, then
  the API shall ignore the submitted value and always report `couple: true` once the caller
  belongs to a couple.

#### FR-PAIR-008 Leave couple

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let a partner leave their couple, dissolving it for both members per DEC-25.

`DELETE /v1/couples/me` ends both memberships, revokes any pending invite, and answers with the
ended couple. Neither partner is in a couple afterwards, so both are free to start a new one at
once (DEC-26); the ended one stays readable through `GET /v1/couples/archived` and in the export
for 30 days, after which an hourly sweep deletes the couple and everything that cascades from it.
In the web app: "End this space" on the couple screen, and "Your past space" in Settings and on
the pairing screen for as long as the window is open.

**Acceptance criteria**

- **FR-PAIR-008.AC1** Given a partner requests to leave, when they confirm, then the system shall
  end the couple for both members, grant each 30 days of read-only access and export, and then
  delete the couple's shared content.
- **FR-PAIR-008.AC2** Given a couple has dissolved this way, when either former partner signs in
  during the 30-day window, then they shall be able to read and export shared content but not add
  to it. Reading it is `GET /v1/couples/archived`; there is no write path to an ended couple at
  all, because no write resolves one.
- **FR-PAIR-008.AC3** Given a partner has left, when they create or join a new couple, then the
  system shall allow it immediately and keep the ended couple readable alongside it (DEC-26).

#### FR-PAIR-009 Faith vocabulary, and when a space uses it

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Test |

A space shall use faith vocabulary only while **both** partners currently hold faith consent
(DEC-29). The weekly mechanic is the same either way — one partner sets the week, the other
responds, the turn alternates — and only the words change. The scripture and verse fields exist
in faith vocabulary only.

This is derived from the two consents each time it is needed. There is no couple-level setting:
a stored mode could outlive a withdrawal, and one partner must never be able to consent on the
other's behalf.

**The rule, for every combination**

| Partner A | Partner B | The space | What happens |
|---|---|---|---|
| Consented | Consented | **Faith** | Prayers, scripture, "This week's prayers" |
| Consented | Not | **Secular** | Intentions, no scripture field, no religious framing |
| Not | Consented | **Secular** | Identical — neither partner's consent outranks the other's |
| Not | Not | **Secular** | Identical, and neither is prompted about it |

A space with one member is always secular in effect: a prayer week needs two members to have a
setter at all (`prayers.SetterFor`), so nothing faith-shaped exists before pairing completes.

**Acceptance criteria**

- **FR-PAIR-009.AC1** Given both partners hold faith consent, when either opens the weekly
  screen, then the system shall use faith vocabulary and shall offer the scripture and verse
  fields.
- **FR-PAIR-009.AC2** Given either partner does not hold faith consent, when either opens the
  weekly screen, then the system shall use secular vocabulary, shall not offer a scripture or
  verse field, and shall not store one.
- **FR-PAIR-009.AC3** Given a space is secular because one partner has not consented, when that
  partner consents in their own settings, then the space shall use faith vocabulary from then on
  without either partner confirming anything further.
- **FR-PAIR-009.AC4** Given a space uses faith vocabulary, when either partner withdraws their
  consent, then the space shall use secular vocabulary from that moment, and no further faith
  content shall be stored. What becomes of faith content already written is Q-25.
- **FR-PAIR-009.AC5** Given a partner is invited to a space, when they join, then the system
  shall not disclose the other partner's consent state to them as a fact about that person; it
  shall state the rule ("a space uses prayer when both of you have turned it on") and leave the
  inference to them.
- **FR-PAIR-009.AC6** Given secular vocabulary is in use, when a partner types religious words
  into an intention or a journal entry of their own accord, then the system shall store it as
  ordinary content; this requirement governs what Amorae asks for, not what people write.

**Vocabulary**

One mechanic, two vocabularies. The faith wording must never appear in a secular space, including
in notifications, empty states and the installed app's shortcuts.

| Faith | Secular |
|---|---|
| This week's prayers | This week's intentions |
| Prayer | Intention |
| Scripture, optional | *(field absent)* |
| Prayed | Done |
| Prayer Mode | Quiet Mode |
| New prayer week | New week |
| Two hearts, one faith | Two people, one life |

Internal names do not change: the route stays `/prayers`, the tables stay `prayer_*`, the Go
module stays `prayers`. This is a display layer, not a migration.

### 3.4 PRAY — Weekly prayer

The signature Faith feature: an alternating weekly rhythm of shared prayer points, published by
whichever partner is due, completed independently by each. The most logic-heavy module; it must
be deterministic and idempotent, and AI is never involved in whose turn it is.

**Goals:** G-02 (Faith rhythm), G-05 (Trust/privacy/reliability)
**Related:** DEC-18, DEC-19; Q-07, Q-16

**Business rules**

- **BR-PRAY-01** — The setter for a week is `members[week_index % 2]`, where members are ordered
  by `joined_at` and `week_index` counts whole weeks since the couple's first prayer week. The
  result is computed once, at week creation, and stored (`setter_user_id`), so it never shifts
  retroactively (DEC-18).
- **BR-PRAY-02** — A prayer week starts Sunday 00:00 in the *couple's* timezone (`couples.timezone`),
  not each partner's individual timezone (Q-07).
- **BR-PRAY-03** — Week creation is idempotent: `unique(couple_id, week_start)` with
  `INSERT ... ON CONFLICT DO NOTHING`, so scheduler retries and overlapping runs never create a
  duplicate week.
- **BR-PRAY-04** — A week holds at most 10 prayer points, each with a title/text and an optional
  scripture reference and verse text; order is the array order (`position`).
- **BR-PRAY-05** — Once the setter publishes a week, the partner can view it. The week itself stays
  open to the setter for the whole week — adding a prayer on Wednesday is the point of a living
  week — but a *point the partner has already completed* is read-only: it cannot be reworded or
  removed, only moved. Additions and points nobody has reached stay editable (DEC-31). Enforced by
  `CanEditPoints` in the prayers module.
- **BR-PRAY-06** — Completion is per partner and per point
  (`unique(prayer_point_id, user_id)`), so marking a point twice never creates a duplicate, and
  one partner's completion is never visible to the other as something they can edit.

#### FR-PRAY-001 View current week

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | UI only | Test |

The system shall show a signed-in partner their couple's current prayer week (Sunday-to-Saturday,
in the couple's timezone).

**Acceptance criteria**

- **FR-PRAY-001.AC1** Given a couple with a current week already created by the scheduler, when
  either partner requests it, then the API shall return its status, setter, points (subject to
  BR-PRAY-05), and each partner's completion state.
- **FR-PRAY-001.AC2** Given no current week exists yet, when it is requested, then the API shall
  not create one on this read; week creation is the scheduler's job alone (BR-PRAY-03,
  FR-PRAY-009).

#### FR-PRAY-002 Setter drafts prayer points

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | UI only | Test |

The system shall let the current week's setter write, reorder and save up to 10 draft prayer
points, each with an optional scripture reference and verse text.

**Acceptance criteria**

- **FR-PRAY-002.AC1** Given the caller is this week's setter and the week is still a draft, when
  they save up to 10 points, then the API shall store them in the given order and respond with
  the updated week.
- **FR-PRAY-002.AC2** Given the caller submits an 11th point, when they save, then the API shall
  respond 400 `validation_failed` with `fields.points` and shall not save any of the points
  (BR-PRAY-04).
- **FR-PRAY-002.AC3** Given the caller is not this week's setter, when they attempt to save
  points, then the API shall reject the request without changing the week.
- **FR-PRAY-002.AC4** Given the partner has completed a point (BR-PRAY-05), when the setter saves a
  list that rewords or drops that point, then the API shall reject it 409 `prayer_in_use` and change
  nothing.
- **FR-PRAY-002.AC5** Given the partner has completed a point, when the setter saves a list that
  adds a new point, edits one nobody has completed, or moves the completed one, then the API shall
  accept it and the partner's completion shall survive.

#### FR-PRAY-003 Publish week

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | UI only | Test |

The system shall let the setter publish their drafted week so their partner can see it.

**Acceptance criteria**

- **FR-PRAY-003.AC1** Given the caller is this week's setter, when they publish, then the API
  shall set the week's status to `published` and it shall become visible to the partner.
- **FR-PRAY-003.AC2** Given the week is already published, when publish is called again, then the
  API shall respond 200 with no change (idempotent).
- **FR-PRAY-003.AC3** Given the caller is not the setter, when they attempt to publish, then the
  API shall reject the request.

#### FR-PRAY-004 Partner sees "waiting" while the setter drafts

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | UI only | Demo |

The system shall show the non-setter partner a "waiting" status, with no points, while the
current week is still a draft.

**Acceptance criteria**

- **FR-PRAY-004.AC1** Given a week whose setter has not yet published, when the other partner
  requests it, then the API shall return status `waiting` and shall not include the draft points.

#### FR-PRAY-005 Independent completion per partner

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | UI only | Test |

The system shall let each partner mark and unmark their own completion of a published point,
independent of their partner's.

**Acceptance criteria**

- **FR-PRAY-005.AC1** Given a published point, when a partner marks it complete, then the API
  shall record that partner's completion only, and marking it again shall not create a duplicate
  (BR-PRAY-06).
- **FR-PRAY-005.AC2** Given a partner un-marks a point they completed, when they do, then the API
  shall remove only their own completion and leave their partner's untouched.

#### FR-PRAY-006 Weekly reflection

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | UI only | Test |

The system shall let each partner record their own short reflection on a prayer week.

**Acceptance criteria**

- **FR-PRAY-006.AC1** Given a signed-in partner, when they save a reflection on a week, then the
  API shall store it against that week and it shall be visible to both partners.

#### FR-PRAY-007 Prayer history

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | UI only | Test |

The system shall list a couple's past prayer weeks, newest first.

**Acceptance criteria**

- **FR-PRAY-007.AC1** Given a couple with more than one past week, when either partner requests
  history, then the API shall return the weeks ordered by `week_start` descending, excluding the
  current week.

#### FR-PRAY-008 Week detail

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | UI only | Test |

The system shall show the full detail of one specific prayer week by id.

**Acceptance criteria**

- **FR-PRAY-008.AC1** Given a week id belonging to the caller's couple, when it is requested,
  then the API shall return its points, completions and reflections.
- **FR-PRAY-008.AC2** Given a week id belonging to another couple, when it is requested, then the
  API shall respond 404 (BR-PAIR-04 / DEC-19).

#### FR-PRAY-009 Scheduler creates the week

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started — Blocked by Q-16 | Test |

The background worker shall create each couple's current prayer week automatically, once its
local Sunday begins, per BR-PRAY-01 through BR-PRAY-03.

**Acceptance criteria**

- **FR-PRAY-009.AC1** Given a couple whose local time has passed Sunday 00:00 and who has no row
  for that week, when the worker ticks, then it shall insert one row with the computed
  `setter_user_id` and shall not create a second row on a later tick for the same week
  (BR-PRAY-03).
- **FR-PRAY-009.AC2** Given couples in different timezones, when the worker ticks hourly, then
  each couple's week shall be created at its own local Sunday, not a single global time.

#### FR-PRAY-010 Push notification on new week

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started — Blocked by Q-16 | Test |

The system shall notify both partners by push when a new prayer week is created.

**Acceptance criteria**

- **FR-PRAY-010.AC1** Given the scheduler inserts a new week row, when the insert succeeds, then
  the worker shall enqueue a `new_prayer_week` push to both partners.
- **FR-PRAY-010.AC2** Given the scheduler's insert is a no-op (the week already existed), when
  that happens, then no duplicate notification shall be sent (BR-PRAY-03).

---

#### FR-PRAY-011 Mark a prayer answered

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Implemented | Test |

The system shall let either partner mark a shared prayer point as answered, with an optional note
about what happened, and take that mark back.

**Acceptance criteria**

- **FR-PRAY-011.AC1** Given a point in a published week, when either partner marks it answered,
  then the API shall record the moment, who marked it, and the note, and return the week.
- **FR-PRAY-011.AC2** Given an already-answered point, when it is marked again with a different
  note, then the API shall replace the note and shall not move the original answered time.
- **FR-PRAY-011.AC3** Given a point in a draft week, when a partner tries to mark it answered,
  then the API shall respond 409 `prayer_not_shared`.
- **FR-PRAY-011.AC4** Given an answered point, when a partner unmarks it, then the API shall clear
  the time, the marker and the note together.

*Why either partner, and why forever:* a prayer belongs to the two of them, and the one who
notices it was answered is not always the one who wrote it down. Nor is it limited to the current
week — prayers are answered on their own schedule, often months later, and a feature that only
worked for seven days would miss most of what it exists to catch. The couple-scoped lookup of the
point is the whole permission check: a point belonging to anyone else is simply not found
(BR-PAIR-04 / DEC-19).

*The note is optional on purpose.* Sometimes the answer is the whole story and there is nothing to
add. Requiring a sentence before you may mark a prayer answered would turn the gladdest action in
the app into a piece of homework.

#### FR-PRAY-012 Answered prayers

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Implemented | Test |

The system shall show a couple every prayer they have marked answered, newest first, grouped by
the month it was answered in.

**Acceptance criteria**

- **FR-PRAY-012.AC1** Given a couple with answered prayers, when either partner requests them,
  then the API shall return them ordered by answered time descending, each carrying the week it
  was prayed in.
- **FR-PRAY-012.AC2** Given a couple with none, when either partner requests them, then the API
  shall return an empty array rather than an error.

*What this screen deliberately does not have:* any list of prayers that were **not** answered, any
count of them, and any way to sort or filter by that. A tally of things you asked for and did not
receive is a wound, not a feature, and the moment this screen implies one it stops being worth
opening. Only answered things accumulate here.

*Dates:* `answered_at` is the instant, used for ordering. `answered_on` is the same moment as a
date in the couple's own timezone, computed in SQL beside the couple row the way every other
couple-local date in this system is (FR-JRNL, FR-APPR) — a prayer answered at half past midnight
in Lagos happened today, and must not read as yesterday because the server keeps UTC, nor read as
a different day to each partner.

---

### 3.5 EVT — Shared events

Events the couple plans together — dinners, dates, appointments — visible and editable by both
partners. There is no single-owner event and no separate participants list: both partners are
implicit participants in everything (DEC-16). All events are couple-scoped; a request for another
couple's event returns 404 (DEC-19).

**Goals:** G-03 (Shared life planning/growth)
**Related:** DEC-16, DEC-19

#### FR-EVT-001 Create event

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let either partner create a shared event with a title, date, optional start/end
time, location, reminder, notes and checklist.

**Acceptance criteria**

- **FR-EVT-001.AC1** Given a title and date, when either partner creates an event, then the API
  shall store it against the couple and return it with a generated id, visible to both partners
  immediately.

#### FR-EVT-002 View events

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall list a couple's events to both partners identically.

**Acceptance criteria**

- **FR-EVT-002.AC1** Given a couple with events, when either partner requests the list, then the
  API shall return the same set and fields to both.

#### FR-EVT-003 Edit event

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let either partner edit any field of a shared event.

**Acceptance criteria**

- **FR-EVT-003.AC1** Given an existing event belonging to the caller's couple, when either partner
  edits it, then the API shall save the change and it shall be visible to both partners.
- **FR-EVT-003.AC2** Given an event id belonging to another couple, when an edit is attempted,
  then the API shall respond 404 (DEC-19).

#### FR-EVT-004 Mark event complete

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let either partner mark a shared event complete, or reopen it.

**Acceptance criteria**

- **FR-EVT-004.AC1** Given an event, when either partner marks it complete, then the API shall
  set `done` to true; when either partner reopens it, the API shall set `done` back to false.

#### FR-EVT-005 Event checklist

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Implemented | Test |

The system shall let either partner toggle an individual checklist item on an event.

**Acceptance criteria**

- **FR-EVT-005.AC1** Given an event with a checklist item, when either partner toggles it, then
  the API shall update only that item's `done` flag and leave the rest of the checklist
  unchanged.

#### FR-EVT-006 Delete event

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Implemented | Test |

The system shall let either partner permanently delete a shared event.

**Acceptance criteria**

- **FR-EVT-006.AC1** Given an event belonging to the caller's couple, when either partner deletes
  it, then the API shall remove it and it shall no longer appear in the events list for either
  partner.

*Contract note:* closed. `DELETE /v1/events/:id` exists, and the event screen asks before using
it — an event disappears for both partners, so it is not a one-tap action.

---

### 3.6 CAL — Couple calendar

A simple, read-focused, mobile-friendly chronological view merging upcoming events, the dates the
couple keep, and the active prayer week. No enterprise calendar complexity: no drag-resize, no
overlapping-event grid.

**Goals:** G-03 (Shared life planning/growth), G-04 (Connection and memory)
**Related:** DEC-19, FR-DATE

#### FR-CAL-001 Chronological view

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Demo |

The system shall show a couple's upcoming events, kept dates and current prayer week together, in
date order.

**Acceptance criteria**

- **FR-CAL-001.AC1** Given a couple with events and a current prayer week, when either partner
  opens the calendar, then the client shall list them merged and sorted by date, nearest first.
- **FR-CAL-001.AC2** Given a kept date set to come round every year (FR-DATE-001), when the
  calendar shows the day it falls on in any year, then the client shall list it there with how
  many years it has been, and shall not list a date kept without that flag.

*Implementation note:* the calendar has no dedicated backend of its own; it composes the FR-EVT,
FR-DATE and FR-PRAY reads on the client, so its status follows theirs and it is verified by demo
rather than a distinct API test.

*Scope note:* AC2 goes beyond the original requirement, which named only events and the prayer
week. A couple's calendar that does not show their anniversary is not their calendar — and once
FR-DATE existed there was nothing to compose it from but a decision. Occurrence is asked per
displayed day (`occursOn`), not per date ("when is this next"): "next" is relative to today, and
a calendar you page backwards and forwards through needs an answer that does not move when you
do. The twenty-ninth of February falls on the twenty-eighth in the years without one, the same
rule the reminder uses, so the day the calendar shows and the day the notification arrives cannot
disagree.

The prayer week appears on the Sunday its own week starts on, and no other. It used to land on
every Sunday of every week paged to, so a week in 2029 promised "a new prayer week begins" — and
because that made every week non-empty, the screen's own "Nothing planned this week yet" could
never appear. A week that has not begun is not a plan. Dates kept without the yearly flag stay off it: their owner said they were part of the
story, not part of the week.

#### FR-CAL-002 Open entry detail

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Demo |

The system shall open an entry's detail screen when it is tapped from the calendar.

**Acceptance criteria**

- **FR-CAL-002.AC1** Given an event or the prayer week entry on the calendar, when a partner taps
  it, then the client shall navigate to that entry's own detail screen (FR-EVT-002/FR-PRAY-008).

*Note:* an event goes to its own screen. The prayer week and a kept date have no per-entry
screen to go to, so they open the week and the milestones list — the nearest thing that exists,
rather than a tap that does nothing.

### 3.7 GOAL — Shared goals

Goals both partners work toward together, with a shared running total and a log of who
contributed what. Deliberately not a gamified productivity product (01 §6).

**Goals:** G-03 (Shared life planning/growth)
**Related:** DEC-19

**Business rules**

- **BR-GOAL-01** — A goal has one shared running total; individual progress entries carry
  `user_id` only so the log can show who logged what, not to split the total per partner.
- **BR-GOAL-02** — A goal's unit is `naira` or `count`, with an optional free-text `unit_label`
  (for example "chapters").

*Data model note:* the v2.0 spec described a stored `current_value` column, incremented by a
service. The built contract (`Goal` in `types.ts`) has no such field: the running total is the
sum of `progress[].amount`, computed on read rather than stored and incremented. See §4(b).

#### FR-GOAL-001 Create goal

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let either partner create a shared goal with a title, optional description
("why"), a target value and unit, and start/end dates.

**Acceptance criteria**

- **FR-GOAL-001.AC1** Given a title, target and unit, when either partner creates a goal, then
  the API shall store it against the couple with progress starting empty and `done` false.

#### FR-GOAL-002 View goals

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall list a couple's goals to both partners identically, each with its running total.

**Acceptance criteria**

- **FR-GOAL-002.AC1** Given a couple with goals, when either partner requests the list, then the
  API shall return the same goals, each with its target, unit and the sum of its progress
  entries.

#### FR-GOAL-003 View goal detail

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall show a single goal's full detail, including its progress log.

**Acceptance criteria**

- **FR-GOAL-003.AC1** Given a goal id belonging to the caller's couple, when it is requested,
  then the API shall return the goal with every progress entry (amount, `user_id`, date).

#### FR-GOAL-004 Log progress

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let either partner add a progress entry to a shared goal.

**Acceptance criteria**

- **FR-GOAL-004.AC1** Given a numeric amount, when either partner logs progress, then the API
  shall append a progress entry carrying their `user_id` and the current date, and the goal's
  running total shall increase by that amount (BR-GOAL-01).

#### FR-GOAL-005 Edit or complete a goal

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Implemented | Test |

The system shall let either partner edit a goal's fields or mark it done.

**Acceptance criteria**

- **FR-GOAL-005.AC1** Given a goal belonging to the caller's couple, when either partner edits
  its fields or marks it done, then the API shall save the change and it shall be visible to both
  partners.

*Contract note:* closed. `PATCH /v1/goals/:id` exists, and the goal screen offers "Mark as done"
and "Still going" — always, rather than appearing once the total is high enough, because reaching
the number is not the only way a goal ends. Archiving is still not built, and is not missed:
a finished goal sorts below the live ones.

---

### 3.8 CHAL — Couple challenges

Short, guided, low-pressure multi-day experiences (for example a 7-day connection challenge),
optional and never framed with penalty language.

**Goals:** G-03 (Shared life planning/growth)
**Related:** DEC-19

**Business rules**

- **BR-CHAL-01** — A challenge runs for a fixed number of days, each with a prompt, and each
  partner marks their own days: `challenge_progress` is keyed on (day, person), as the v2.0
  design had it (DEC-30, resolving Q-23). `ChallengeDay` in `types.ts` carries `done`/`skipped`
  for the caller and `partner_done`/`partner_skipped` alongside, exactly as a prayer week carries
  `my_completed` and `partner_completed`. Both can see both; neither can change the other's.

#### FR-CHAL-001 View current challenge

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall show a couple's current multi-day challenge with each day's prompt and status.

**Acceptance criteria**

- **FR-CHAL-001.AC1** Given a couple with an active challenge, when either partner requests it,
  then the API shall return its title, start date and every day's prompt, `done` and `skipped`
  state.

#### FR-CHAL-002 Mark a day done or skipped

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let a partner mark a challenge day done or skipped.

**Acceptance criteria**

- **FR-CHAL-002.AC1** Given a challenge day, when a partner marks it done, then the API shall
  record it **for that partner only**, and their partner's state for that day shall be unchanged
  (DEC-30). Both can see both, as with prayer completion.
- **FR-CHAL-002.AC2** Given a day marked done, when a partner marks it skipped instead, then the
  API shall update the day's status accordingly, without penalty language in the UI (01 §6).

#### FR-CHAL-003 Start a challenge

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Implemented | Test |

The system shall let either partner start a new multi-day challenge from a template.

**Acceptance criteria**

- **FR-CHAL-003.AC1** Given a couple with no active challenge, when either partner starts one
  from a template, then the API shall create it with day 1 through N and respond 201.

*Contract note:* closed. `GET /v1/challenges/templates` lists a curated catalogue and
`POST /v1/challenges` starts one; `DELETE /v1/challenges/current` leaves it, which is what frees
a couple to start another. One at a time, enforced by a unique index rather than a check, so two
taps race to one challenge. The templates live in Go rather than a table: they are copy, and a
table would mean a migration to fix a typo in a sentence somebody reads on day four.

---

### 3.9 JRNL — Shared journal

Private, couple-shared journal entries: gratitude, reflection, memory, appreciation or a plan,
each tagged and dated.

**Goals:** G-04 (Connection and memory)
**Related:** DEC-19

*Data model note:* the built `JournalEntry` type (`id`, `author_id`, `date`, `tag`, `text`) has no
`photo`, `related_event` or `updated_at` field, and its tag values (`Gratitude`, `Reflection`,
`Memory`, `Appreciation`, `Plans`) differ in casing and set from the v2.0 `journal_type` enum
(which also included `prayer_reflection`). See §4(b).

#### FR-JRNL-001 Add journal entry

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let either partner add a private, couple-shared journal entry with a tag and
body text.

**Acceptance criteria**

- **FR-JRNL-001.AC1** Given a tag from the defined set and non-empty text, when either partner
  submits an entry, then the API shall store it with their id as author and today's date, and it
  shall be visible to both partners.
- **FR-JRNL-001.AC2** Given a tag outside the defined set, or a different spelling of one in it,
  when an entry is submitted, then the API shall refuse it.

*Two things the request does not get to decide:* the author is whoever the session says it is, and
the date is the couple's local day (DEC-27), worked out where the couple row already is rather
than from UTC in Go. Taking the UTC date would file anything written between midnight and one in
the morning in Lagos under yesterday — the hour somebody is most likely to be writing in a
journal. The tag is matched exactly, not case-insensitively: it comes from a picker, so another
spelling means a client sending something this server has never offered, and two spellings of
"Gratitude" is a list the screen cannot group.

#### FR-JRNL-002 View journal entries

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall list a couple's journal entries to both partners identically, in date order.

**Acceptance criteria**

- **FR-JRNL-002.AC1** Given a couple with entries, when either partner requests the journal, then
  the API shall return the same entries to both, newest first.

---

### 3.10 APPR — Appreciation

One partner writes a short, personal note of appreciation to the other. Personal, not social:
there is no feed, no likes, no reactions.

**Goals:** G-04 (Connection and memory)
**Related:** DEC-16, DEC-19

**Business rules**

- **BR-APPR-01** — An appreciation has no stored recipient field; because a couple has exactly two
  members, the recipient is always implicitly "the other partner" (the same two-person
  simplification as DEC-16).
- **BR-APPR-02** — The sender can undo (delete) an appreciation shortly after sending it
  (DEC-21). The client offers "Undo" on the send-confirmation toast for **5 seconds**, only while
  online, and never queues it offline. The API accepts the delete only from the sender and only
  within **30 seconds** of sending (the extra time absorbs a slow connection); after that it
  answers 409 `undo_window_closed`, and for the partner's note 403 `forbidden`.

#### FR-APPR-001 Send appreciation

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let a partner send a short appreciation note to their partner.

**Acceptance criteria**

- **FR-APPR-001.AC1** Given non-empty text, when a partner sends an appreciation, then the API
  shall store it with their id as sender and today's date, and it shall become visible to both
  partners.

#### FR-APPR-002 Notify only the recipient

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall notify only the receiving partner when an appreciation is sent, not the sender.

**Acceptance criteria**

- **FR-APPR-002.AC1** Given an appreciation is sent, when it is stored, then the system shall
  queue exactly one push notification, addressed to the recipient only.

*Built:* the worker sends it, not the endpoint, and that is the point. A note is announced only
once it can no longer be taken back (BR-APPR-02) — one withdrawn ten seconds after sending should
never have reached a lock screen, and the send itself cannot know what happens next, so only
something that looks back can promise it. Journal entries ride the same path with no such wait,
since there is nothing to take back. Both name who wrote it and nothing of what they wrote
(FR-NOTF-005.AC1), and neither ever goes to the person who wrote it.

#### FR-APPR-003 Sender undo

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let the sender withdraw an appreciation shortly after sending it, per BR-APPR-02.

**Acceptance criteria**

- **FR-APPR-003.AC1** Given an appreciation was just sent, when the sender taps Undo on the
  confirmation toast within 5 seconds, then the client shall delete it and it shall disappear
  from both partners' view.
- **FR-APPR-003.AC2** Given more than 5 seconds have passed, when the sender wants to remove it,
  then the client shall no longer offer Undo.
- **FR-APPR-003.AC3** Given the device is offline, when the confirmation toast is showing, then
  the client shall not offer Undo, and no undo shall ever be queued for later.
- **FR-APPR-003.AC4** Given a delete arrives more than 30 seconds after sending, or from the
  partner who did not send it, when the API processes it, then it shall refuse with 409
  `undo_window_closed` or 403 `forbidden` respectively, and the note shall remain.

*Verified 2026-09-24 against the Go endpoint:* the sender takes a note back inside the window and
it disappears for both of them; the partner is refused 403; a delete after the window is refused
409 `undo_window_closed` and the note stays; and a repeated undo of one already gone answers 204,
because a retry should find the world as it wanted it. The undo is never queued offline (AC3), so
it is the one write here that is online-only.

### 3.11 MEM — Memories

A private, couple-shared timeline of moments — read as a quiet archive, not a feed.

**Goals:** G-04 (Connection and memory)
**Related:** DEC-19; Q-06

#### FR-MEM-001 Add memory

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let either partner add a memory with a title, date, optional location and
optional note.

**Acceptance criteria**

- **FR-MEM-001.AC1** Given a title and date, when either partner adds a memory, then the API
  shall store it against the couple with `has_photo` false and return it with a generated id.
- **FR-MEM-001.AC2** Given a request that sets `has_photo` true, when it is stored, then the API
  shall ignore that field and store `has_photo` false.

*Note on `has_photo`:* the client posts the whole `Memory` shape it holds, `has_photo` included,
and the decoder refuses fields it has not been told about — so the field is accepted and then
thrown away. Whether a photo exists is the server's to say, and it becomes true when one is
actually stored (FR-MEM-003), never because a request claimed it. A memory that says it has a
picture it does not have is a broken screen. It is not a column either: `has_photo` is computed
from whether `photo_id` is set, so the two cannot drift apart.

#### FR-MEM-002 View memories

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall list a couple's memories to both partners identically, as a quiet archive rather
than a feed.

**Acceptance criteria**

- **FR-MEM-002.AC1** Given a couple with memories, when either partner requests the list, then
  the API shall return the same memories to both, newest first.

*Not a feed:* a memory has no author, no reactions and no comments. Either partner may keep one
and it belongs to them both (DEC-16). Every feature that would make this a feed is one the two of
them would then have to manage, and the point is an archive to read back.

#### FR-MEM-003 Attach a photo

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Implemented | Test |

The system shall let either partner attach one photo to a memory.

**Acceptance criteria**

- **FR-MEM-003.AC1** Given a memory and an image file under the configured size cap, when a
  partner uploads it, then the system shall store it in private object storage, strip EXIF and
  GPS data, and set `has_photo` true (Q-06 recommendation).

*How it is stored:* Cloudinary, and the browser uploads **straight there** rather than through
the API. A file relayed through our container would cost the one thing the free tier is meanest
with — a 10 MB photo occupying a request slot for its whole upload — for no benefit, since the
bytes end up in the same place either way. So the API only mints a signature
(`POST /v1/memories/{id}/photo/ticket`), the browser posts the file to Cloudinary with it, and the
memory is told about the result afterwards (`PUT /v1/memories/{id}/photo`). `has_photo` becomes
true only at that last step, which keeps FR-MEM-001's rule intact: a memory claims a photo when
one has actually landed, not when a client said so.

The server picks the asset's name — `amorae/{couple}/memories/{memory}` — so the client never
chooses where a file goes, and one memory can hold exactly one photo. EXIF and GPS are dropped by
re-encoding: delivery is `f_auto,q_auto`, which re-compresses and carries no metadata across.

*Deviation from Q-06 — signed URLs that do not expire.* Q-06 asked for **short-lived** signed
URLs. Cloudinary's free plan has no expiring-token feature, so a delivery URL is signed and
unguessable but valid indefinitely. What this costs: someone who obtains a URL keeps access to
that one photo, and deleting the memory does not invalidate a URL already copied. What it does
not cost: the asset is `type: authenticated`, so no photo can be reached by guessing a URL, and
the listing endpoint only signs URLs for the couple the photos belong to. For two people sharing
an album this is the right trade against the alternatives, which are paying for a plan tier or
proxying every image byte through the API. Revisit if this is ever used by strangers.


---

### 3.12 DATE — Important dates and milestones

Birthdays, anniversaries, the relationship start date, and other milestones, modelled as one
entity distinguished by type (DEC-17), with an optional reminder.

**Goals:** G-04 (Connection and memory)
**Related:** DEC-17, DEC-19; Q-16

**Business rules**

- **BR-DATE-01** — Milestones, birthdays and anniversaries are one entity, distinguished by type
  rather than separate tables (DEC-17).

*Data model note:* DEC-17 and the v2.0 spec describe a discrete `type` enum (`birthday`,
`anniversary`, `relationship_start`, `engagement`, `wedding`, `milestone`, `custom`); the built
`Milestone` type (`types.ts`) has no `type` field, only a free-text `sub` (subtitle) and a
`reminder` boolean. The distinguishing "type" is not currently a structured field anywhere in the
contract. See §4(b).

#### FR-DATE-001 Add an important date

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let either partner add an important date with a title, date, optional subtitle
and optional reminder flag.

**Acceptance criteria**

- **FR-DATE-001.AC1** Given a title and date, when either partner adds it, then the API shall
  store it against the couple and return it with a generated id.

#### FR-DATE-002 View important dates

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall list a couple's important dates to both partners identically.

**Acceptance criteria**

- **FR-DATE-002.AC1** Given a couple with important dates, when either partner requests the list,
  then the API shall return the same dates to both, in an order the client can render
  chronologically.

#### FR-DATE-003 Reminder delivery

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Implemented | Test |

The system shall notify both partners ahead of an important date flagged for a reminder.

**Acceptance criteria**

- **FR-DATE-003.AC1** Given an important date with `reminder` true, when its date approaches,
  then the worker shall enqueue a push notification to both partners in time to be useful.

*Built:* two notifications, not one. A week's notice is the one you can act on — book the table,
take the day off — and the morning itself is the one that matters, because nobody wants to hear
about their anniversary only in time to plan it. Each is claimed separately, so one being sent
never swallows the other. Both go out at 8am in the couple's own timezone and stay due for the
rest of that day: a prayer reminder missed by an hour can go out tomorrow, an anniversary
cannot. The twenty-ninth of February falls back to the twenty-eighth in the three years out of
four that have no twenty-ninth, on the client and the server alike, so the day the screen counts
down to and the day the notification arrives are never a day apart. A date with its reminder off
is kept and never announced.

---

### 3.13 NOTF — Notifications

Per-user push notification preferences and delivery, carrying reminders and alerts from every
other module without exposing their content.

**Goals:** G-05 (Trust/privacy/reliability), and indirectly G-02/G-03/G-04 (it carries their
reminders)
**Related:** DEC-19; Q-07, Q-16

#### FR-NOTF-001 Per-user notification preferences

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let each user set their own notification preferences, independent of their
partner's.

**Acceptance criteria**

- **FR-NOTF-001.AC1** Given a signed-in user, when they change any of `new_week`,
  `prayer_reminder`, `event_reminders`, `important_dates`, `appreciation`, `journal`, `goals` or
  `challenges`, then the API shall save it against that user only and shall not affect their
  partner's preferences.

#### FR-NOTF-002 Reminder time in the user's own timezone

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let each user set an `HH:MM` reminder time, interpreted in their own timezone,
not the couple's.

**Acceptance criteria**

- **FR-NOTF-002.AC1** Given a user sets `reminder_time` to a value matching `HH:MM`, when it is
  saved, then the API shall store it, and any personal reminder shall fire at that clock time in
  the user's own timezone (`users.timezone`) — distinct from the couple's timezone used by the
  prayer scheduler (Q-07).
- **FR-NOTF-002.AC2** Given a value that does not match `HH:MM`, when it is submitted, then the
  API shall reject it.

#### FR-NOTF-003 Subscribe to push

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall let a user subscribe their browser to push notifications.

**Acceptance criteria**

- **FR-NOTF-003.AC1** Given the user has granted browser notification permission, when the client
  subscribes, then it shall POST the browser's `PushSubscription` to the API, which shall store
  it against that user and respond 204.

#### FR-NOTF-004 Remove dead subscriptions

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The system shall remove a push subscription once the push service reports it as gone.

**Acceptance criteria**

- **FR-NOTF-004.AC1** Given a push send receives a 404 or 410 from the push service, when that
  response is received, then the worker shall delete the corresponding subscription so no further
  sends are attempted against it.

#### FR-NOTF-005 No private content on the lock screen

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started | Inspection |

The system shall use generic notification copy that does not reveal prayer, journal or
appreciation content on a lock screen.

**Acceptance criteria**

- **FR-NOTF-005.AC1** Given any notification category, when it is composed, then its title and
  body shall describe the type of activity (for example "Adeola sent you an appreciation note")
  without quoting or previewing the private content itself.
- **FR-NOTF-005.AC2** Given an event reminder or an important-date reminder, when it is composed,
  then it may name the event or date and say when it is, and shall carry nothing else about it —
  not an event's location, notes or checklist, and not a date's subtitle.

*Scope note:* AC2 is a deliberate exception, and the line it draws is between private writing and
a shared plan. A prayer, a journal entry and an appreciation note are things one person wrote;
quoting them on a lock screen exposes the person who wrote them. An event is a calendar entry the
two of them made together, and a reminder that will not say what it is for is one you have to
unlock your phone to understand — which defeats the reminder. Everything else the event holds
stays inside the app.

#### FR-NOTF-006 Quiet by default

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Implemented | Demo |

The system shall not request push permission or send a notification until the app has explained
the value of that category to the user.

**Acceptance criteria**

- **FR-NOTF-006.AC1** Given a user has not yet interacted with a feature that offers push (for
  example prayer reminders), when they first reach the relevant screen, then the client shall
  explain what the notification is for before, or as part of, requesting permission — never on
  first launch, unprompted.

*Status note:* a UI/copy rule with no dedicated backend, verified by demo. Both paths that can
ask now explain first: the onboarding screen previews what would arrive before offering to turn
it on, and the settings screen — where the permission has never been asked for — says what the
switches cannot do until the browser allows them, with the request behind that explanation
rather than on a switch. Permission is only ever asked once, so asking it on a stray tap would
spend it.

#### FR-NOTF-007 Push delivery

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Test |

The worker shall send a push notification for each enabled category, respecting each user's own
preferences and reminder time.

Built so far. Three prayer categories respecting `new_week` and `prayer_reminder`: the setter is
told it is their week while it is still empty; the other partner is told when it is published,
which is the moment the shared thing becomes shared; and each partner is reminded once a day, in
their own timezone, while a published week still has prayers left for them.

And event reminders, respecting `event_reminders`. Both partners are nudged before something they
planned, at the lead the event names — "30 minutes before", "the morning of", "the day before".
The phrase is read into a moment in the couple's own timezone (`EventReminderAt`), not the
person's, because an event happens at a place. An event with no start time falls back to the
morning of its day rather than being dropped. A reminder more than an hour late is not sent: the
worker may have been down, and a nudge about something that began ninety minutes ago is noise.
The send is keyed on the event *and the moment*, so moving an event you have already been
reminded about earns a second reminder for the new time, and re-running the worker never earns
two for the same one. This is why the worker ticks every five minutes rather than hourly — "ten
minutes before" on an hourly tick can arrive after the thing it was warning about. The reminder
names the event and says when it is, and nothing else about it (FR-NOTF-005.AC2).

And important dates (FR-DATE-003), respecting `important_dates`.

And the two things one partner writes for the other, respecting `appreciation` and `journal`:
a note of appreciation, once its undo window has closed, and a journal entry straight away.
Neither goes to the person who wrote it, and neither carries a word of what was written.

And the last two, both off unless somebody asks for them (FR-NOTF-006):

- **A goal one of them put something towards.** The goal is named and the
  amount never is. A goal is a shared plan and may be named the way an event
  may (FR-NOTF-005.AC2), but what somebody just moved into their savings is
  not something to put on a lock screen in a coffee shop. Only while the goal
  is still going, and never to whoever logged it.
- **A challenge day nobody has marked.** Once in the morning, in the couple's
  zone, keyed on their own date, and silent as soon as they mark it. It goes
  out in the morning rather than at their prayer reminder time, because those
  are the two recurring nudges in the app and firing both at seven in the
  evening turns one of them into noise. It says which day of how many and
  never what was missed: a challenge is not a streak, and under DEC-30 a
  skipped day is a day rather than a failure.

Every category FR-NOTF-007.AC1 names is now sent.

**Acceptance criteria**

- **FR-NOTF-007.AC1** Given a triggering event for an enabled category (new prayer week, prayer
  reminder, event reminder, appreciation, journal activity, goal update, challenge reminder,
  important-date reminder), when it occurs, then the worker shall send exactly one push per
  subscribed device for that user, and none for a category the user has turned off.

### 3.14 PWA — Install, offline and updates

Amorae is a Progressive Web App first, iPhone-first: installable, tolerant of weak or absent
networks, and clear about what is and is not safe to do offline.

**Goals:** G-05 (Trust/privacy/reliability); enables every other module under weak networks
**Related:** DEC-04, DEC-05, DEC-14; Q-01

**Business rules**

- **BR-PWA-01** — The service worker caches only content-hashed static assets and brand assets; it
  never caches pages, API responses or personal data (DEC-04).
- **BR-PWA-02** — An offline write is saved only if its mutation is registered as resumable. It is
  stored under the signed-in user's id in `localStorage`, for at most 7 days, and the query cache
  itself is never persisted (DEC-05).
- **BR-PWA-03** — Every write declares, in its own definition, what happens if it is sent twice:
  either `idempotent: "<why>"` or `onlineOnly: true`. The type offers no third option, so a write
  cannot be queued without someone having said why replaying it is safe (DEC-28). The resumable
  set is therefore exactly the idempotent writes: prayer completion toggle, save draft prayer
  points, publish a prayer week (DEC-21), save a weekly reflection, complete/reopen an event,
  toggle an event checklist item, mark/skip a challenge day, and save notification preferences.
  Everything that creates a row without a key of its own — save an event, create a goal, add a
  journal entry, send an appreciation, add a memory, add a milestone — and logging goal progress,
  which adds an amount rather than setting one, are `onlineOnly` until their endpoints take an
  idempotency key (FR-PWA-009). Undoing an appreciation is `onlineOnly` for a different reason:
  its meaning depends on when it lands (DEC-21). Account and pairing actions (register, login,
  create/join a couple, leave a couple, delete account) have no `writes.ts` and are never queued.

#### FR-PWA-001 Installable

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Demo |

The system shall be installable as a standalone app from a web manifest with a full icon set.

**Acceptance criteria**

- **FR-PWA-001.AC1** Given a supporting browser, when the manifest is fetched, then it shall
  declare name "Amorae", standalone display, portrait orientation, background and theme colour
  `#F7F4EF` (DEC-14), and 192/512 and maskable icons.
- **FR-PWA-001.AC2** Given the app is installed, when it is launched from the home screen, then
  it shall open standalone (no browser chrome) at its start URL.

#### FR-PWA-002 iOS guided install

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Demo |

The client shall guide an iOS user through Add to Home Screen manually, since iOS offers no
`beforeinstallprompt` event.

**Acceptance criteria**

- **FR-PWA-002.AC1** Given an iOS visitor without the app installed, when they reach the install
  step, then the client shall show step-by-step instructions rather than an automatic prompt, and
  shall not repeat them aggressively on every visit.

#### FR-PWA-003 Offline page on cold navigation

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Demo |

The system shall show a static offline page when a cold (uncached) navigation is attempted with
no network (DEC-04).

**Acceptance criteria**

- **FR-PWA-003.AC1** Given the device is offline and the app has not already loaded the requested
  page in this session, when the user navigates to it, then the service worker shall serve the
  static `/offline` page rather than a browser error.
- **FR-PWA-003.AC2** Given the offline page is shown, when connectivity returns, then a retry
  control shall reload the originally requested page.

#### FR-PWA-004 Subtle offline indicator

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Demo |

The client shall show a quiet, non-blocking indicator when the device is offline, never a
full-screen error.

**Acceptance criteria**

- **FR-PWA-004.AC1** Given the device goes offline while content is already loaded, when the
  state changes, then the client shall show a status banner ("Offline · Showing your saved
  content") without hiding the content underneath.
- **FR-PWA-004.AC2** Given the device comes back online after queued changes synced, when they
  finish, then the client shall show a confirmation banner naming how many changes synced.

#### FR-PWA-005 Offline writes

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Demo |

The client shall let a signed-in user continue making the writes listed in BR-PWA-03 while
offline, queuing them for later delivery.

**Acceptance criteria**

- **FR-PWA-005.AC1** Given the device is offline, when the user performs a write in BR-PWA-03's
  set, then the client shall pause the mutation rather than fail it, and shall reflect the change
  optimistically where the screen already shows one.
- **FR-PWA-005.AC2** Given a write outside BR-PWA-03's set is attempted offline (for example
  registering, or creating a couple), when it is attempted, then the client shall surface that it
  requires a connection rather than queue it silently.

#### FR-PWA-006 Replay on reconnect

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Demo |

The client shall automatically send every queued write once the device reconnects.

**Acceptance criteria**

- **FR-PWA-006.AC1** Given one or more paused writes, when the browser reports it is back online,
  then the client shall resume and send them, in the order each write's scope defines (for
  example prayer completions send one at a time, in order).

#### FR-PWA-007 Clear queued writes on logout, session expiry and deletion

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Demo |

The client shall delete every locally queued write for a user when they sign out, their session
expires, or their account is deleted.

**Acceptance criteria**

- **FR-PWA-007.AC1** Given any of those three events, when it occurs, then the client shall clear
  the in-memory cache and remove the saved offline-changes entry for that user from local
  storage, so nothing queued can be sent under a different session.

#### FR-PWA-008 Warn on logout with unsynced changes

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Implemented | Demo |

The client shall warn a user before logging them out if they have unsynced offline changes.

**Acceptance criteria**

- **FR-PWA-008.AC1** Given one or more paused, unsynced writes, when the user asks to log out,
  then the client shall name how many changes have not synced and state that logging out now will
  lose them, offering to stay logged in instead.

#### FR-PWA-009 Client idempotency keys for queued writes

| Priority | Release | Status | Verification |
|---|---|---|---|
| Must | MVP | Not started — not yet reachable | Test |

Before a queued, non-idempotent write is sent to the API, the client shall attach
a client-generated idempotency key so a retried send cannot be applied twice.

**Acceptance criteria**

- **FR-PWA-009.AC1** Given a queued write that is not naturally idempotent (for example creating
  an event), when it is first attempted, then the client shall generate and attach a stable key
  that survives being saved and restored from local storage.
- **FR-PWA-009.AC2** Given the same write is retried after a partial failure, when it is sent
  again, then the API shall recognise the repeated key and shall not create a second record.

*Status note:* no longer the only thing standing between a dropped response and a duplicate row.
Under DEC-28 a write may not be queued unless it says why a replay is safe, so the non-idempotent
writes are `onlineOnly` rather than queued-and-hoped. This requirement is what lets them queue
again: when an endpoint accepts a key, its write moves from `onlineOnly` to `idempotent`.

None of Step 1's writes needs it. Prayer completion is keyed on (point, person), saving points
replaces the whole week, publishing is a state transition, and a reflection is one row per person
per week — all idempotent in the schema rather than by a key.

#### FR-PWA-010 Conflict rule: last write wins

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Not started | Analysis |

When two writes to the same resource conflict, the system shall resolve them by server
timestamp, the later write winning.

**Acceptance criteria**

- **FR-PWA-010.AC1** Given both partners edit the same resource while one was offline, when the
  offline write is finally sent, then the API shall apply whichever write reaches it with the
  later server timestamp and discard the earlier one's conflicting fields.

*Note:* completions and checklist toggles are naturally idempotent and so never conflict under
this rule; it matters only for fields like an event's title or a journal entry's text.

#### FR-PWA-011 Online-only actions

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Implemented | Demo |

The client shall require an active connection for actions whose meaning depends on when they
land, rather than queue them offline (DEC-21).

**Acceptance criteria**

- **FR-PWA-011.AC1** Given the device is offline, when a user would delete a resource (today:
  undoing an appreciation), then the client shall not offer the action, and if it is triggered
  anyway it shall fail at once rather than be queued.
- **FR-PWA-011.AC2** Given the device is offline, when the setter publishes a prayer week, then
  the client may queue it: publishing is setter-only and idempotent, so a late send is harmless
  (DEC-21).

*Status note:* a write marked `onlineOnly` in its definition (`apps/web/src/lib/query/mutations.ts`)
never pauses and is never registered as resumable. Account and couple actions have no offline
write definition and are never queued.

#### FR-PWA-012 Cold offline reading

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Not started — Blocked by Q-01 | Test |

The system shall let a user read their couple's current prayer week and upcoming events after a
cold start with no network.

**Acceptance criteria**

- **FR-PWA-012.AC1** Given the app was not already open when the device went offline, when the
  user opens it with no network, then the client shall show the last-synced prayer week and
  events rather than only the static offline page.

*Note:* DEC-04 deliberately does not cache personal data today. This requirement applies only if
Q-01 chooses option (b), an encrypted cache; Q-01's recommendation for MVP is option (a), in
which case this requirement becomes **Won't** for MVP.

#### FR-PWA-013 App update prompt

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | MVP | Partial | Demo |

The client shall detect a new deployed version and bring the user onto it without losing unsaved
work.

**Acceptance criteria**

- **FR-PWA-013.AC1** Given a new service worker is available, when the browser next checks
  (`updateViaCache: "none"` forces revalidation), then the client shall fetch it in the
  background.

*Status note:* today the new worker takes over silently on a later full reload; there is no
in-app "update available, tap to refresh" prompt. Status **Partial** reflects the background
check existing without a user-facing prompt.

---

### 3.15 AI — Optional AI assistance

Optional writing help behind the prayer feature: draft suggestions, rewording, reflection
summaries and verified scripture suggestions. Every feature here is Post-MVP, opt-in, and the
product is fully functional with AI disabled (DEC-15, 01 §6 "Human before AI").

**Goals:** G-02 (Faith rhythm), G-05 (Trust/privacy/reliability)
**Related:** DEC-15; Q-18

**Business rules**

- **BR-AI-01** — AI must never claim divine authority, promise prayer outcomes, manipulate
  emotionally, invent Bible references, auto-publish generated content, or replace user judgement;
  every AI output requires the user's explicit review before it is saved or published (DEC-15).
- **BR-AI-02** — Daily/weekly caps: 5 prayer-point generations per user per day, 10 rewrite
  requests per user per day, 1 weekly-reflection generation per user per week; no background
  generation — only on explicit request.

#### FR-AI-001 Prayer point assistant

| Priority | Release | Status | Verification |
|---|---|---|---|
| Could | Post-MVP | Not started | Test |

The system shall let a user request AI-suggested draft prayer points from a described situation.

**Acceptance criteria**

- **FR-AI-001.AC1** Given a user describes a situation, when they request suggestions, then the
  AI service shall return draft points the user can edit or discard, and none shall be saved
  until the user explicitly keeps them (BR-AI-01).
- **FR-AI-001.AC2** Given the user has already made 5 generation requests today, when they
  request a 6th, then the system shall decline with a clear message (BR-AI-02).

#### FR-AI-002 Improve my prayer

| Priority | Release | Status | Verification |
|---|---|---|---|
| Could | Post-MVP | Not started | Test |

The system shall let a user ask AI to refine the wording of a rough prayer point they wrote.

**Acceptance criteria**

- **FR-AI-002.AC1** Given a user's draft text, when they request a rewrite, then the AI service
  shall return a suggested rewording without altering the user's saved point until they accept
  it.
- **FR-AI-002.AC2** Given the user has already made 10 rewrite requests today, when they request
  an 11th, then the system shall decline (BR-AI-02).

#### FR-AI-003 Weekly reflection assistant

| Priority | Release | Status | Verification |
|---|---|---|---|
| Could | Post-MVP | Not started | Test |

The system shall let a user ask AI to help summarise their own reflection on a prayer week.

**Acceptance criteria**

- **FR-AI-003.AC1** Given a user's own notes, when they request a summary, then the AI service
  shall return a draft reflection the user must review and explicitly save (BR-AI-01).
- **FR-AI-003.AC2** Given the user has already used this once this week, when they request it
  again, then the system shall decline until the following week (BR-AI-02).

#### FR-AI-004 Scripture suggestions with verified references

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | Post-MVP | Not started | Test |

The system shall let a user request a scripture suggestion related to a prayer topic, showing
only references the system has verified as real.

**Acceptance criteria**

- **FR-AI-004.AC1** Given a topic, when a suggestion is requested, then the AI service shall check
  any Bible reference it proposes against a verified source before it is shown, and shall omit
  any reference it cannot verify rather than display an invented one (BR-AI-01).

#### FR-AI-005 Opt-in with data-use notice

| Priority | Release | Status | Verification |
|---|---|---|---|
| Should | Post-MVP | Not started | Test |

The system shall require a user to explicitly opt in to AI features, after being told that their
prayer content leaves Amorae to reach the AI provider (Q-18).

**Acceptance criteria**

- **FR-AI-005.AC1** Given a user has not opted in, when they open any AI feature, then the client
  shall show the data-use notice and require an explicit choice before the feature is enabled,
  per user.
- **FR-AI-005.AC2** Given AI is opted out or the feature flag is off, when the user uses the rest
  of the app, then every non-AI feature shall work exactly as it does with AI unavailable
  (DEC-15).

## 4. Data requirements

### 4(a) Data classification

| Class | Description | Examples |
|---|---|---|
| **Public** | Safe for anyone, including logged-out visitors, to see. | Marketing/welcome copy, the Terms and Privacy page text itself, brand assets, app icons. |
| **Internal** | Operational data about the system, not about a person. | Request logs (request id, client IP, route, latency), Prometheus metrics, rate-limit counters. |
| **Personal** | Identifies or is about one user or one couple, but is not itself sensitive in the GDPR/NDPA sense. | Account fields (email, display_name, timezone, avatar_url), couple membership and role labels, invite codes, events, goals, challenges, notification preferences, push subscription endpoints. |
| **Sensitive personal** | Reveals something the law treats with extra care — religious belief, or intimate relationship detail. | Prayer points, reflections and scripture engagement (reveals religious belief); journal entries and appreciations (reveal relationship and emotional detail); `relationship_start_date` and couple profile (reveal relationship status); memory photos, once built (may reveal identity or location via EXIF before it is stripped, Q-06). |

Sensitive personal data drives stricter handling requirements in
[03-non-functional-requirements.md](./03-non-functional-requirements.md) (encryption at rest,
no content in logs, no content on a lock screen — FR-NOTF-005) rather than anything unique to
this document; it is classified here because the entities that carry it are defined here.

### 4(b) Logical data model

Every table is scoped to one `couple_id` (directly, or through a parent that is) except `users`,
`sessions` and `push_subscriptions`, which are scoped to one `user_id`. Couple scoping is enforced
server-side from the caller's membership, never trusted from the client, and a cross-couple id
returns 404 (BR-PAIR-04 / DEC-19).

**Built** — the physical schema in `apps/api/migrations/*.sql` is the source of truth; this
summarises it rather than repeating the SQL.

| Entity | Purpose | Key attributes | Ownership | Key constraints |
|---|---|---|---|---|
| `users` | One person's account. | `id`, `email`, `display_name`, `password_hash`, `avatar_url?`, `timezone` (default `UTC`), `last_login_at?` | Self | `email` unique |
| `sessions` | An active login. | `token_hash` (SHA-256 of the opaque token, PK), `user_id`, `expires_at` | Self, not user-visible | FK `user_id` cascades on delete; indexed on `user_id` and `expires_at` |
| `couples` | The shared account two partners belong to. | `id`, `name?`, `timezone` (default `UTC`), `relationship_start_date?`, `created_by` | Couple | none beyond PK; membership caps size (see below) |
| `couple_members` | Who belongs to which couple, and their per-person state. | `couple_id`, `user_id`, `role` (default `partner`), `joined_at`, `updated_at`, `onboarding_install`, `onboarding_notifications` | Couple + self (onboarding flags are per person) | `unique(couple_id, user_id)`; `unique(user_id)` (BR-PAIR-01, one couple per user); member order = `joined_at` ascending (BR-PRAY-01) |
| `couple_invitations` | An open or resolved invite. | `id`, `couple_id`, `code` (unique), `status` (`pending`\|`accepted`\|`revoked`\|`expired`), `created_by`, `expires_at`, `accepted_at?` | Couple | `code` unique |

*Difference from the v2.0 draft:* the built `couple_invitations` table has no `accepted_by`
column (only `accepted_at`); the v2.0 draft schema included one. Recorded here rather than
silently carried forward.

**Proposed** — everything else. Names are aligned with `apps/web/src/lib/api/types.ts` (the
contract the screens were built against, which Go will implement or change); differences from the v2.0 draft schema
are called out because they change what a future migration needs to build.

| Entity | Purpose | Key attributes | Ownership | Notes / differences from v2.0 |
|---|---|---|---|---|
| `PrayerWeek` | One couple's week of prayer. | `id`, `week_start`, `week_end`, `setter_id`, `status` (`draft`\|`published`\|`waiting`), `points[]`, `my_completed[]`, `partner_completed[]`, `reflection?` | Couple | `unique(couple_id, week_start)` (BR-PRAY-03). `week_end` is new; `setter_id` was `setter_user_id`; `status` gained `waiting` (computed for the non-setter, not a stored draft/published-only enum); completions are exposed as two id arrays per caller rather than a normalised join table. |
| `PrayerPoint` | One point within a week. | `id`, `title`, `text`, `scripture?`, `verse?`, `position` | Couple (via week) | Max 10 per week (BR-PRAY-04). `title` and `verse` are new versus v2.0; `scripture` is new (v2.0 had no reference field at all). |
| `Event` | A shared calendar entry. | `id`, `title`, `date`, `start_time?`, `end_time?`, `location?`, `reminder?`, `notes?`, `checklist[]`, `done` | Couple | v2.0 used `starts_at`/`ends_at` timestamps and a separate `event_reminders` table; the built type uses a plain `date` plus time-of-day strings and folds the reminder into one optional field. `notes` replaces `recap_note` with different, pre-event semantics. `status` (enum) became a single `done` boolean. |
| `Goal` | A shared goal. | `id`, `title`, `why?`, `target`, `unit` (`naira`\|`count`), `unit_label?`, `progress[]`, `start_date`, `end_date`, `done` | Couple | No stored `current_value` (BR-GOAL-01); v2.0's separate `goal_items` checklist does not exist in the built type. `target_unit` (free text) became a closed `unit` enum plus `unit_label`. |
| `GoalProgress` | One logged contribution. | `id`, `user_id`, `amount`, `date` | Couple (via goal) | `user_id` is for the log only (BR-GOAL-01), not a per-partner split. |
| `Challenge` + `ChallengeDay` | A multi-day guided challenge. | `id`, `title`, `started_on`, `days[{n, text, done, skipped}]` | Couple | v2.0 modelled three tables (`challenges`, `challenge_days`, `challenge_progress` with per-user completion); the built type has one shared `done`/`skipped` per day (BR-CHAL-01). |
| `JournalEntry` | A shared journal entry. | `id`, `author_id`, `date`, `tag`, `text` | Couple | `tag` is `Gratitude`\|`Reflection`\|`Memory`\|`Appreciation`\|`Plans` — different casing and set from v2.0's `journal_type` (which also had `prayer_reflection`). No `photo_url`, `related_event_id` or `updated_at`. |
| `Appreciation` | A note from one partner to the other. | `id`, `from_id`, `date`, `text` | Couple | No recipient field (BR-APPR-01); no `read_at` (v2.0 had one for a "seen" state). |
| `Memory` | One entry in the memories archive. | `id`, `title`, `date`, `location?`, `note?`, `has_photo` | Couple | v2.0 had `photo_url`, `related_event_id`, `created_by`, `updated_at`; none of these are in the built type. Photo storage itself is Q-06. |
| `Milestone` (`milestones`) | A birthday, anniversary or milestone. | `id`, `title`, `date`, `sub?`, `reminder?` | Couple | BR-DATE-01; the table is `milestones`, named for the route, the type and the module rather than v2.0's `important_dates`. No structured `type` field exists in the built type, unlike DEC-17's description and v2.0's `important_date_type` enum; `sub` is free text, and there is no `is_recurring` flag. |
| `PushSubscription` | One browser's push endpoint. | `id`, `user_id`, `endpoint` (unique), `p256dh`, `auth`, `user_agent?`, `created_at`, `last_used_at?` | Self | Matches the v2.0 draft; not yet built. |
| `NotificationPrefs` | One user's notification settings. | `new_week`, `prayer_reminder`, `reminder_time`, `event_reminders`, `important_dates`, `appreciation`, `journal`, `goals`, `challenges` | Self | v2.0 modelled this as one `jsonb` map; the built type is a flat object with the same semantics, one row per user. |

### 4(c) Retention and deletion

| Entity / event | Retention rule |
|---|---|
| Sessions | Purged 7 days after `expires_at` (Q-11). |
| A deleted account | Hard-deleted within 30 days of the delete request; backups holding it roll off within 35 days (Q-11). |
| A dissolved couple (leave couple, Q-09 option (a)) | Both former partners get 30 days of read-only access and export to the couple's shared content; after that window, couple-owned content (prayer weeks/points/completions, events, goals, challenges, journal, appreciations, memories, important dates) is deleted. Each partner's own account and personal profile data are unaffected and follow the account's own lifecycle. |
| Application logs | Kept 30 days; never contain request or resource content (Q-11). |
| Push subscription rows | Removed as soon as the push service reports them gone (404/410) via FR-NOTF-004, not on a fixed schedule. |
| Memory photos (once built, Q-06) | Follow the owning memory's own deletion; EXIF and GPS data are stripped on upload and never retained anywhere. |

## 5. External interface requirements

### 5.1 User interface

The full interface contract — design tokens, typography, components and their states, the screen
inventory, interaction and content rules — is [04-design-specification.md](./04-design-specification.md).
This document does not restate it.

### 5.2 HTTP API

The Go API is the source of truth for the wire contract; [`../API.md`](../API.md) documents it in
full, including the planned endpoints nothing serves yet. This document does not
reproduce the endpoint list or payload examples — only the conventions that shape every
requirement above:

- Every response is wrapped in a `{ "data": ... }` envelope on success, or
  `{ "error": { "code", "message", "fields"?, "request_id" } }` on failure. `code` is stable and
  meant for programs to branch on; `message` and `fields` are safe to show a user; internal
  details (stack traces, database errors, internal field names) never reach the client.
- IDs are UUIDv7 strings; timestamps are RFC 3339, UTC, everywhere.
- Request bodies are capped at 1 MiB, and unknown JSON fields are rejected outright.
- Every response carries an `X-Request-ID` header, for correlating logs across the BFF and the
  API; a client may send its own to keep the same id end to end.
- Standard status codes: 400 validation, 401 unauthenticated/invalid credentials, 403 forbidden,
  404 not found *or not yours*, 409 conflict, 429 rate limited (with `Retry-After`), 500 internal
  (always the generic `internal_error` message, looked up by `request_id`).
- A request for another couple's resource returns 404, never 403 (DEC-19), so a client cannot use
  the status code to confirm a resource exists.
- Authenticated endpoints take `Authorization: Bearer <token>`; the browser never holds or sends
  this token directly (DEC-02) — the Next.js BFF attaches it server-side from the httpOnly
  session cookie.
- Rate limits: 20 requests per client IP per minute across `/v1/auth/*`, and 10 login attempts
  per email per 15 minutes, counted whether or not the account exists (DEC-11).

### 5.3 Web Push

VAPID key pair; the browser subscribes through the Push API and posts the subscription to
`POST /v1/notifications/subscribe` (FR-NOTF-003). Sending pushes and managing keys is the
background worker's job (Q-16) and is Not started (FR-NOTF-007).

### 5.4 Email

No provider is chosen yet (Q-04). Every transactional email — password reset (FR-AUTH-008),
verification (FR-AUTH-009), and the email-change notices (FR-ACCT-003) — goes through a single
`Mailer` interface so the provider can be swapped without touching feature code.

### 5.5 Object storage

**Cloudinary**, for memory photos (FR-MEM-003). Assets are `type: authenticated` — unreachable
without a signature — and the browser uploads directly to Cloudinary using a signature the API
mints, so no image byte passes through our own container. Delivery is a signed URL with
`f_auto,q_auto`, which re-encodes and so drops EXIF and GPS. A 10 MB cap and an images-only
filter are enforced client-side before upload and by Cloudinary on receipt.

Chosen over an S3-compatible bucket because it does transformation, format negotiation and CDN
delivery in the same free tier, none of which a bare bucket gives. The seam is
`internal/platform/photos`, one type with two methods, so a move to S3 or R2 is a rewrite of that
file and nothing else. One Q-06 deviation, recorded under FR-MEM-003: signed URLs do not expire
on the free plan.

### 5.6 AI provider

Behind a Go interface (`AIService`) so the provider is swappable and the product works fully with
it disabled (DEC-15). Requires a provider under zero-retention, no-training terms, opt-in per
user with a data-use notice (Q-18). Post-MVP; not yet built (§3.15).

---

## 6. Traceability matrix

Every functional requirement defined in §3, once. "TBD" in the API column means no endpoint is
documented yet, even as a proposed one; "—" means the requirement has no distinct API surface of
its own (a client-only behaviour, or one composed from other requirements' endpoints).

| FR ID | Goal | API endpoint(s) | Status | Verification evidence |
|---|---|---|---|---|
| FR-AUTH-001 | G-01 | `POST /v1/auth/register` | Implemented | Go integration tests (auth) |
| FR-AUTH-002 | G-01 | `POST /v1/auth/login` | Implemented | Go integration tests (auth) |
| FR-AUTH-003 | G-05 | `POST /v1/auth/logout` | Implemented | Go integration tests (auth) |
| FR-AUTH-004 | G-05 | (session check on every authenticated endpoint) | Implemented | Go integration tests (auth) |
| FR-AUTH-005 | G-05 | TBD — Blocked by Q-03 | Not started | None yet |
| FR-AUTH-006 | G-05 | `POST /v1/auth/register` | Implemented | Go integration tests (auth) |
| FR-AUTH-007 | G-05 | `POST /v1/auth/login`, `PUT /v1/users/me/email` | Implemented | Go integration tests (auth) |
| FR-AUTH-008 | G-05 | `POST /v1/auth/forgot`, `POST /v1/auth/reset` | Implemented | Go unit + Postgres integration tests |
| FR-AUTH-009 | G-05 | TBD — Blocked by Q-05 | Not started | None yet |
| FR-AUTH-010 | G-05 | TBD — Blocked by Q-10 | Not started | None yet |
| FR-AUTH-011 | G-05 | — (registration screen copy) | Partial | Manual demo 2026-09-22 |
| FR-AUTH-012 | G-05 | `POST /v1/auth/register` (`faith_consent`), `user_consents` | Partial | Consent captured and recorded; nothing reads it (FR-PAIR-009) |
| FR-ACCT-001 | G-05 | `GET`, `PATCH /v1/users/me` | Implemented | Go integration tests (users) |
| FR-ACCT-002 | G-05 | `PUT /v1/users/me/email` | Implemented | Go integration tests (users) |
| FR-ACCT-003 | G-05 | TBD — Blocked by Q-05 | Not started | None yet |
| FR-ACCT-004 | G-05 | TBD | Not started | None yet |
| FR-ACCT-005 | G-05 | `DELETE /v1/users/me` | UI only | Screen built; no endpoint to test against |
| FR-ACCT-006 | G-05 | TBD | Not started | None yet |
| FR-ACCT-007 | G-05 | — (client only) | Implemented | Manual demo 2026-09-22 |
| FR-ACCT-008 | G-05 | `GET /v1/sessions`, `DELETE /v1/sessions/others` | Implemented | Go unit + Postgres integration tests; browser check 2026-09-23 |
| FR-PAIR-001 | G-01 | `POST /v1/couples` | Partial | Browser check 2026-09-22 against the Go API; Go unit tests (couples, in development) |
| FR-PAIR-002 | G-01 | `GET /v1/couples/me` | Partial | Manual demo 2026-09-22 |
| FR-PAIR-003 | G-01 | `POST /v1/couples/join` | Partial | Browser check 2026-09-22 against the Go API; Go unit tests (couples, in development) |
| FR-PAIR-004 | G-01 | TBD — Blocked by Q-08 | Not started | None yet |
| FR-PAIR-005 | G-01 | `PATCH /v1/couples/me` | Partial | Go handler in the in-progress couples module; no tests yet |
| FR-PAIR-006 | G-01 | `PATCH /v1/couples/role` | Partial | Go handler in the in-progress couples module; no tests yet |
| FR-PAIR-007 | G-01 | `PATCH /v1/couples/me/onboarding` | Partial | Go handler and migration in progress |
| FR-PAIR-008 | G-05 | `DELETE /v1/couples/me`, `GET /v1/couples/archived` | Implemented | Go unit + Postgres integration tests; browser check 2026-09-23 |
| FR-PAIR-009 | G-02, G-05 | Derived from `user_consents` — no endpoint of its own | Not started | None yet |
| FR-PRAY-001 | G-02 | `GET /v1/prayers/current` | Implemented | Go unit + Postgres integration tests; API and browser check 2026-09-23 |
| FR-PRAY-002 | G-02 | `PUT /v1/prayers/current/points` | Implemented | Go unit + Postgres integration tests; API and browser check 2026-09-23 |
| FR-PRAY-003 | G-02 | `POST /v1/prayers/current/publish` | Implemented | Go unit + Postgres integration tests; API and browser check 2026-09-23 |
| FR-PRAY-004 | G-02 | `GET /v1/prayers/current` | Implemented | Go unit + Postgres integration tests; API and browser check 2026-09-23 |
| FR-PRAY-005 | G-02 | `POST`, `DELETE /v1/prayers/points/:id/complete` | Implemented | Go unit + Postgres integration tests; API and browser check 2026-09-23 |
| FR-PRAY-006 | G-02 | `PATCH /v1/prayers/weeks/:id/reflection` | Implemented | Go unit + Postgres integration tests; API and browser check 2026-09-23 |
| FR-PRAY-007 | G-02 | `GET /v1/prayers/history` | Implemented | Go unit + Postgres integration tests; API and browser check 2026-09-23 |
| FR-PRAY-008 | G-02 | `GET /v1/prayers/weeks/:id` | UI only | Screen built; no endpoint to test against |
| FR-PRAY-009 | G-02 | TBD — Blocked by Q-16 | Not started | None yet |
| FR-PRAY-010 | G-02 | TBD — Blocked by Q-16 | Not started | None yet |
| FR-PRAY-011 | G-02 | `PUT`/`DELETE /v1/prayers/points/:id/answered` | Implemented | Go unit tests; API round-trip and browser check 2026-09-25 |
| FR-PRAY-012 | G-02 | `GET /v1/prayers/answered` | Implemented | Go unit tests; API round-trip and browser check 2026-09-25 |
| FR-EVT-001 | G-03 | `POST /v1/events` | Implemented | Go unit tests; 27-check API pass and browser check 2026-09-24 |
| FR-EVT-002 | G-03 | `GET /v1/events` | Implemented | Go unit tests; 27-check API pass and browser check 2026-09-24 |
| FR-EVT-003 | G-03 | `PATCH /v1/events/:id` | Implemented | Go unit tests; 27-check API pass and browser check 2026-09-24 |
| FR-EVT-004 | G-03 | `POST`, `DELETE /v1/events/:id/complete` | Implemented | Go unit tests; 27-check API pass and browser check 2026-09-24 |
| FR-EVT-005 | G-03 | `PATCH /v1/events/:id/checklist/:item` | Implemented | Go unit tests; 27-check API pass and browser check 2026-09-24 |
| FR-EVT-006 | G-03 | `DELETE /v1/events/:id` | Implemented | Go unit tests; 27-check API pass and browser check 2026-09-24 |
| FR-CAL-001 | G-03 | `GET /v1/events`, `GET /v1/milestones`, `GET /v1/prayers/current` | Implemented | Browser demo against live data 2026-09-24 |
| FR-CAL-002 | G-03 | — (client navigation) | Implemented | Browser demo against live data 2026-09-24 |
| FR-GOAL-001 | G-03 | `POST /v1/goals` | Implemented | Go unit tests; 26-check API pass and browser check 2026-09-24 |
| FR-GOAL-002 | G-03 | `GET /v1/goals` | Implemented | Go unit tests; 26-check API pass and browser check 2026-09-24 |
| FR-GOAL-003 | G-03 | `GET /v1/goals/:id` | Implemented | Go unit tests; 26-check API pass and browser check 2026-09-24 |
| FR-GOAL-004 | G-03 | `POST /v1/goals/:id/progress` | Implemented | Go unit tests; 26-check API pass and browser check 2026-09-24 |
| FR-GOAL-005 | G-03 | `PATCH /v1/goals/:id` | Implemented | Go unit tests; 26-check API pass and browser check 2026-09-24 |
| FR-CHAL-001 | G-03 | `GET /v1/challenges/current` | Implemented | Go unit tests; 23-check API pass and browser check 2026-09-24 |
| FR-CHAL-002 | G-03 | `PATCH /v1/challenges/current/days/:n` | Implemented | Go unit tests; 23-check API pass and browser check 2026-09-24 |
| FR-CHAL-003 | G-03 | `GET /v1/challenges/templates`, `POST /v1/challenges` | Implemented | Go unit tests; 23-check API pass and browser check 2026-09-24 |
| FR-JRNL-001 | G-04 | `POST /v1/journal` | Implemented | Go unit tests; 26-check API pass and browser check 2026-09-24 |
| FR-JRNL-002 | G-04 | `GET /v1/journal` | Implemented | Go unit tests; 26-check API pass and browser check 2026-09-24 |
| FR-APPR-001 | G-04 | `POST /v1/appreciations` | Implemented | Go unit tests; 26-check API pass and browser check 2026-09-24 |
| FR-APPR-002 | G-04 | Worker (`ForWritten`) | Implemented | Go unit tests; worker run against the database 2026-09-24 |
| FR-APPR-003 | G-04 | `DELETE /v1/appreciations/:id` | Implemented | Go unit tests; API pass incl. the closed window, and the toast's Undo driven in the browser 2026-09-24 |
| FR-MEM-001 | G-04 | `POST /v1/memories` | Implemented | Go unit tests; 21-check API pass and browser check 2026-09-24 |
| FR-MEM-002 | G-04 | `GET /v1/memories` | Implemented | Go unit tests; 21-check API pass and browser check 2026-09-24 |
| FR-MEM-003 | G-04 | `POST /v1/memories/{id}/photo/ticket`, `PUT`/`DELETE /v1/memories/{id}/photo` | Implemented | Go unit tests incl. a signature pinned to Cloudinary's published vector |
| FR-DATE-001 | G-04 | `POST /v1/milestones` | Implemented | Go unit tests; 20-check API pass and browser check 2026-09-24 |
| FR-DATE-002 | G-04 | `GET /v1/milestones` | Implemented | Go unit tests; 20-check API pass and browser check 2026-09-24 |
| FR-DATE-003 | G-04 | Worker (`ForImportantDates`) | Implemented | Go unit tests; worker run against the database 2026-09-24 |
| FR-NOTF-001 | G-03 | `GET`, `PATCH /v1/notifications/preferences` | Implemented | Go unit tests; 19-check API pass and browser check 2026-09-23 |
| FR-NOTF-002 | G-03 | `PATCH /v1/notifications/preferences` | Implemented | Go unit tests; 19-check API pass and browser check 2026-09-23 |
| FR-NOTF-003 | G-03 | `POST /v1/notifications/subscribe` | Implemented | Go unit tests; 19-check API pass and browser check 2026-09-23 |
| FR-NOTF-004 | G-03 | `cmd/worker` — deletes on 404/410 from the push service | Implemented | Go unit tests with a fake push service; worker run against Postgres 2026-09-24 |
| FR-NOTF-005 | G-05 | — (notification copy) | Not started | None yet |
| FR-NOTF-006 | G-03 | Onboarding and settings screens | Implemented | Browser check 2026-09-24: both paths explain before asking |
| FR-NOTF-007 | G-03 | `cmd/worker` — hourly tick, `notification_sends` for exactly-once | Implemented | Go unit tests with a fake push service; worker run against Postgres 2026-09-24 |
| FR-PWA-001 | G-05 | — (web manifest) | Implemented | Manual demo 2026-09-22 |
| FR-PWA-002 | G-05 | — (client) | Implemented | Manual demo 2026-09-22 |
| FR-PWA-003 | G-05 | — (service worker) | Implemented | Manual demo 2026-09-22 |
| FR-PWA-004 | G-05 | — (client) | Implemented | Manual demo 2026-09-22 |
| FR-PWA-005 | G-05 | — (client, over BR-PWA-03's endpoints) | Implemented | Manual demo 2026-09-22 |
| FR-PWA-006 | G-05 | — (client) | Implemented | Manual demo 2026-09-22 |
| FR-PWA-007 | G-05 | — (client) | Implemented | Manual demo 2026-09-22 |
| FR-PWA-008 | G-05 | — (client) | Implemented | Manual demo 2026-09-22 |
| FR-PWA-009 | G-05 | TBD | Not started | None yet |
| FR-PWA-010 | G-05 | TBD | Not started | None yet |
| FR-PWA-011 | G-05 | — (client) | Implemented | Browser check 2026-09-22 (undo hidden and not queued offline) |
| FR-PWA-012 | G-05 | TBD — Blocked by Q-01 | Not started | None yet |
| FR-PWA-013 | G-05 | — (client) | Partial | Manual demo 2026-09-22 |
| FR-AI-001 | G-02 | TBD | Not started | None yet |
| FR-AI-002 | G-02 | TBD | Not started | None yet |
| FR-AI-003 | G-02 | TBD | Not started | None yet |
| FR-AI-004 | G-02 | TBD | Not started | None yet |
| FR-AI-005 | G-05 | TBD | Not started | None yet |

## Appendix A: Where v2.0 content moved

The v2.0 "Engineering Specification" covered architecture, security, testing, deployment and
scalability alongside functional requirements. This rewrite keeps only functional requirements,
the data model and traceability; everything else has a new, dedicated home:

| v2.0 section | Moved to |
|---|---|
| §1 Architecture overview, §5 Backend architecture | [`../ARCHITECTURE.md`](../ARCHITECTURE.md) and the ADRs ([0001](../adr/0001-modular-monolith.md), [0002](../adr/0002-bff-and-opaque-sessions.md), [0003](../adr/0003-stdlib-first.md), [0004](../adr/0004-pwa-service-worker.md)) |
| §7 Security | [03-non-functional-requirements.md](./03-non-functional-requirements.md) (NFR-SEC) |
| §10 Non-functional requirements (performance, offline, data) | [03-non-functional-requirements.md](./03-non-functional-requirements.md) |
| §11 Testing | [03-non-functional-requirements.md](./03-non-functional-requirements.md) (NFR-MAINT) and Q-15 |
| §12 Environment and deployment | [03-non-functional-requirements.md](./03-non-functional-requirements.md) (NFR-OPS) and [`../ARCHITECTURE.md`](../ARCHITECTURE.md) |
| §13 Scalability and extensibility | [03-non-functional-requirements.md](./03-non-functional-requirements.md) (NFR-SCALE) and [`../ARCHITECTURE.md`](../ARCHITECTURE.md) |
| §14 Open questions | [05-decisions-and-open-questions.md](./05-decisions-and-open-questions.md) §2 |
| §15 Decisions log | [05-decisions-and-open-questions.md](./05-decisions-and-open-questions.md) §1 — specifically DEC-16 (dropped `event_participants`), DEC-17 (`important_dates` unified by type), DEC-18 (`setter_user_id` frozen at creation), DEC-19 (404, not 403, cross-couple) |

Functional content — the requirements themselves, the data model's entity list, the scheduler
rules and the AI rules and caps — was kept and is now in §3, §4 and the business rules above,
reconciled against the code as built.

---

## Change log

| Version | Date | Change |
|---|---|---|
| 3.0 | 2026-09-22 | Later the same day: the mock was removed (DEC-22), so statuses read "UI only", and the appreciation undo became online-only (DEC-21). Full rewrite as an ISO/IEC/IEEE 29148-style Software Requirements Specification (`AMR-REQ-02`). Every requirement now carries a permanent ID, Priority, Release, Implementation status and Verification method, with Given/When/Then acceptance criteria and a traceability matrix. Reconciled against the code as built on 2026-09-22 (couples module Partial/in development; prayers, events, goals, challenges, journal, appreciations, memories, milestones, notification preferences and account deletion are UI only; scheduler, push delivery, email, object storage and AI are Not started). Recorded, rather than silently resolved, the gaps this reconciliation found between the v2.0 spec, [`../API.md`](../API.md), `types.ts` and the actual mock/Go behaviour — see the business-rule and contract notes throughout §3 and the differences column in §4(b). Non-functional, architecture, testing, deployment and scalability content moved out to their own documents (Appendix A); decisions and open questions now live solely in [05](./05-decisions-and-open-questions.md), cited by ID rather than restated. |
| 2.0 | 2026-09-21 | Split out of the combined spec. Resolved vague feature lists into Must / Should / Later with acceptance criteria. Added full data model (columns, types, enums, constraints), API conventions and example payloads, a deterministic scheduler algorithm, concrete security controls, offline sync rules, non-functional targets, and a scalability section. |

