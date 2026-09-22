# Amorae: Product Requirements Document

| | |
|---|---|
| **Document ID** | AMR-REQ-01 |
| **Version** | 3.0 |
| **Status** | Draft for review |
| **Owner** | Product |
| **Last updated** | 2026-09-22 |

---

## 1. Purpose and audience

This document is the product-level case for Amorae: why we are building it, for whom, what
success looks like, what ships and when, and what could stop it from working. It is written for
anyone who needs the "why" before the "how" — Product, Engineering, Design, Legal, and any
founder or sponsor reviewing the direction.

It does not restate requirement detail that lives elsewhere. For that, see:

- [`02-functional-requirements.md`](./02-functional-requirements.md) — what the system shall do,
  feature by feature, with acceptance criteria and a traceability matrix.
- [`03-non-functional-requirements.md`](./03-non-functional-requirements.md) — how well: security,
  privacy, accessibility, performance, availability, and operations.
- [`04-design-specification.md`](./04-design-specification.md) — tokens, components, screen
  inventory, and content rules.
- [`05-decisions-and-open-questions.md`](./05-decisions-and-open-questions.md) — the single
  register of decisions (`DEC-NN`) and open questions (`Q-NN`) cited throughout this document.

This version also folds in a structural change: the old "Decisions log" section (section 8 of
v2.0) has been removed. Those three decisions now live in 05 as **DEC-12**, **DEC-13**, and
**DEC-15**, alongside every other product and architecture decision, so there is one place to
look instead of several.

---

## 2. Problem statement

Couples who want to intentionally build their relationship — spiritually, practically, and
emotionally — do not have one private place to do it. Today that work is scattered across tools
that were not built for a two-person relationship:

- Group chat apps (WhatsApp, iMessage) carry the conversation but not the structure: no shared
  goals, no prayer rhythm, no memory archive, no couple-scoped privacy from other threads.
- Generic productivity and calendar tools organise tasks and events but have no concept of "this
  belongs to us as a couple" and nothing for faith, appreciation, or memory.
- Social platforms are public by design, which is the opposite of what a couple wants for their
  prayer life, their shared goals, or their private memories.
- Church and community apps assume a group, not a pair, and centre the institution rather than
  the two people.

The result: couples either do this work nowhere (it quietly does not happen), or they stitch it
together across four or five tools, none of which is private, couple-scoped, or built for the
rhythm of a relationship.

**This is a hypothesis, not a finding.** Amorae has not run user research, surveys, or market
sizing to confirm how common or how painful this gap is, or which piece of it (faith rhythm,
planning, memory-keeping) matters most to which couples. The private beta (section 8) exists
specifically to test this hypothesis before committing further investment past MVP. Any number
that looks like a market statistic elsewhere in this document is an internal target, not external
research — the metrics in section 5 are what we will use to find out if the hypothesis holds.

---

## 3. Vision and positioning

Amorae is a private digital space for two people to intentionally build their relationship
together.

Prayer and faith matter, but Amorae is not a prayer app. It supports the whole shared life of a
couple: planning, goals, appreciation, journaling, memories, milestones, important dates, and
faith.

The core idea, in one line:

> Two people, intentionally building a life together.

**Primary positioning statement:**

> Amorae is a private space for couples to connect, grow, plan, share experiences, practice their
> faith, and build memories together.

Prayer is a first-class feature, but it does not define the product's identity (DEC-13).
Everything — logo, navigation, content, architecture — stays relevant whether the couple is
praying, planning a date, working toward a goal, or saving a memory.

**Final framing the product should earn:**

> Your private space for the life you are building together.

The product should feel private, calm, intimate, useful, and human. It should never feel like:

- a public social network,
- a generic productivity tool,
- a dating app, or
- a church management system.

---

## 4. Users

### 4.1 Primary personas

Compact personas to keep every requirement grounded in a real job, not a feature wish list.

**Tobi — the setter.**
Paired with his partner for eight months, works full-time, checks his phone in short bursts
during the day.
*When it is my week to set our prayer points, I want to add what is actually on our hearts in
under two minutes, so I can lead our prayer time without it feeling like homework.*

**Ngozi — the planner.**
Organises most of the couple's shared logistics: events, goals, the odd surprise date. Wants
visibility into the relationship's "state" without having to ask.
*When we are both busy during the week, I want one place to see our upcoming plans, goals, and
whether this week's prayer is published, so I can feel connected to our life together even on
days we barely talk.*

**Chidi — the newly paired user.**
Just installed the app after his partner sent an invite. Has never used Amorae before and is
deciding, in the first two minutes, whether this is worth keeping.
*When my partner invites me to Amorae, I want pairing to be fast and obviously about "us," so I
can start our shared space together without friction or confusion about what this app even is.*

### 4.2 User classes

| Class | Definition |
|-------|------------|
| **Unpaired user** | A registered user with no couple yet. Can manage their own account and hold or redeem one invite, but has no access to any couple-scoped feature (Faith, Plan, Grow, Connect, Remember). |
| **Partner** | A member of a paired couple (exactly two members per couple, DEC-12). Full read/write access to that couple's shared resources, and to their own account. |
| **This week's setter** | Not a separate account type — a partner in the state of holding the current prayer week's setter role (FR-PRAY and its business rules in 02). Alternates weekly and is computed once per week (DEC-18). Only the setter can create or edit that week's prayer points before publish. |

### 4.3 Stakeholders

| Stakeholder | Interest |
|-------------|----------|
| Product | Owns this document, the roadmap, and the success metrics. |
| Engineering | Owns feasibility, the API/BFF contract, and the decisions register (05) jointly with Product. |
| Design | Owns the design specification (04), interaction and content rules, and accessibility of the UI. |
| Legal & Compliance | Reviews Terms/Privacy, NDPA and GDPR exposure (faith content, cross-border hosting), owns Q-17. |
| Founder(s) / sponsor | Sets business direction, funds development, makes the go/no-go call on public launch. |

### 4.4 Who this is NOT for

- Prayer groups, congregations, or anyone wanting group or public prayer — Amorae is built for
  exactly two people (DEC-12); it is not a church or community app.
- Singles looking for a dating or matchmaking product.
- Anyone wanting a public or semi-public presence for their relationship — there is no public
  surface, by design (principle 1).
- Polyamorous or multi-partner households needing to coordinate more than two people in one
  shared space — out of scope for MVP's two-member model.
- Anyone needing serious financial co-management (joint budgeting, bill splitting, investment
  tracking) — see the out-of-scope table in section 7.

### 4.5 Launch market assumption

The mock data throughout the product uses `Africa/Lagos` as a default timezone and naira-
denominated example goals. **Assumption, not a confirmed fact — Product to confirm:** the launch
market is Nigeria and the Nigerian diaspora, in English. This shapes currency examples, default
timezone behaviour, and initial marketing, but nothing in the product architecture hard-codes it
(see `couples.timezone`, Q-07). If this assumption is wrong, the impact is mostly on content and
go-to-market, not on the data model.

---

## 5. Goals and success metrics

### 5.1 Goals

| ID | Goal |
|----|------|
| **G-01** | **Activation.** A new user pairs with their partner quickly. |
| **G-02** | **Faith rhythm.** Couples sustain the weekly prayer loop (setter publishes, both partners complete). |
| **G-03** | **Shared life.** Couples plan and grow together (events, goals, challenges). |
| **G-04** | **Connection and memory.** Couples express appreciation and preserve memories. |
| **G-05** | **Trust.** Private, secure and reliable; zero cross-couple data exposure; data rights honoured. |

### 5.2 Success metrics

Principle 3 ("Intentional, not addictive") rules out time-in-app, daily-active-user, or streak
metrics as KPIs — they reward endless engagement, not meaningful use. Every KPI below measures a
weekly meaningful action instead. Every measurement source is a first-party aggregate (Q-14): no
third-party analytics SDK is used anywhere in the product.

**Every initial target below is a hypothesis to be calibrated during private beta, not a
committed number.** They exist so the beta has something concrete to compare against, and they
will be revised once real usage data exists.

| ID | Goal | Metric | Definition / formula | Initial target | Type | Measurement source |
|----|------|--------|-----------------------|-----------------|------|---------------------|
| M-01 | G-01 | Pairing completion within 7 days | % of new sign-ups who complete couple pairing within 7 days of registration | Hypothesis — ≥ 50%, to calibrate | KPI | First-party aggregate SQL over account and couple-membership timestamps (Q-14) |
| M-02 | G-01 | Median time to pair | Median hours from sign-up to successful pairing, among users who do pair | Hypothesis, to calibrate | KPI | First-party aggregate SQL |
| M-03 | G-02 | Weekly publish rate | % of paired couples whose current prayer week is published by Monday 23:59 in the couple's timezone, per rolling week | Hypothesis — ≥ 70%, to calibrate | KPI | First-party aggregate SQL over prayer-week publish times |
| M-04 | G-02 | Weekly completion rate | % of published prayer weeks where both partners mark at least one point as prayed before the week ends | Hypothesis, to calibrate | KPI | First-party aggregate SQL |
| M-05 | G-03 | Weekly meaningful actions per couple | Mean count, per paired couple per week, of: event created or completed, goal progress logged, or challenge day marked done | Hypothesis, to calibrate | KPI | First-party aggregate SQL across EVT/GOAL/CHAL tables |
| M-06 | G-03 | 4-week couple retention | % of couples active (per M-05, or a prayer completion) in week 1 after pairing who are still active in week 4 | Hypothesis, to calibrate | KPI | First-party aggregate SQL |
| M-07 | G-04 | Appreciation notes per couple per week | Median count of appreciation notes sent per paired couple per week | Hypothesis, to calibrate | KPI | First-party aggregate SQL over the appreciation table |
| M-08 | G-04 | Memory capture rate | % of paired couples who save at least one memory within 30 days of pairing | Hypothesis, to calibrate | KPI | First-party aggregate SQL over the memories table |
| M-09 | G-05 | Cross-couple data exposure incidents | Count of confirmed incidents where one couple's data was readable by another couple or a non-member | 0, always | Guardrail | Security review / incident log (Inspection), cross-checked against DEC-19 |
| M-10 | G-05 | 5xx error rate | % of API responses returning a 5xx status, rolling 7 days | Hypothesis — < 0.5%, to calibrate | Guardrail | Prometheus request counters by status code |
| M-11 | G-05 | Notification opt-out rate | % of subscribed users who disable at least one push category within 30 days of subscribing | Hypothesis, to calibrate | Guardrail | First-party aggregate SQL over notification preferences |
| M-12 | G-05 | Account deletion rate | % of accounts that request deletion within 90 days of sign-up | Hypothesis, to calibrate | Guardrail | First-party aggregate SQL over account deletion requests |

No metric in this table counts session length, opens per day, or consecutive-day streaks.

---

## 6. Product principles

These are non-negotiable. Every requirement in 02, 03, and 04 is checked against them.

1. **Private by default.** Couple data belongs to the couple. No public surface, ever.
2. **A two-person experience.** The app always understands it is a shared space between two
   partners.
3. **Intentional, not addictive.** Encourage meaningful moments, not endless scrolling or streak
   pressure. (This is why section 5's metrics avoid DAU and streaks.)
4. **Warm, not childish.** No excessive hearts, pink gradients, cartoon romance, or cliché
   imagery.
5. **Useful before decorative.** Every visual element should improve understanding, orientation,
   or tone.
6. **Human before AI.** AI is optional and never the defining interaction. The product must work
   fully with AI disabled (DEC-15).
7. **Faith without pressure.** Prayer invites reflection. It never uses guilt, manipulation, or
   religious pressure.
8. **Designed for real life.** It must work with limited time, weak network, and one free hand.

---

## 7. Scope

### 7.1 MVP in scope

Listed by functional module (defined in full in 02). This is module-level scope only — numbered
`FR-<MODULE>-NNN` requirements are defined in 02, not here.

| Module | Pillar | Capability | Goal(s) | Phase |
|--------|--------|------------|---------|-------|
| `AUTH` | Foundation | Sign-up, sign-in, sign-out, password reset, session management | G-01, G-05 | 1 |
| `ACCT` | Foundation | Profile (display name, timezone), email and password change, account deletion, data export | G-01, G-05 | 1 |
| `PAIR` | Foundation | Create couple, invite by code/link, join, leave couple, couple profile | G-01 | 1 |
| `PRAY` | Faith | Weekly prayer points with alternating setter, independent completion, prayer history | G-02 | 2 |
| `EVT` | Plan | Shared events: create, edit, complete, recap | G-03 | 3 |
| `CAL` | Plan | Read-focused chronological view of events and the active prayer week | G-03 | 3 |
| `GOAL` | Grow | Shared goals with progress tracking | G-03 | 3 |
| `CHAL` | Grow | Short, guided, low-pressure multi-day challenges | G-03 | 3 |
| `JRNL` | Connect | Shared private journal entries | G-04 | 4 |
| `APPR` | Connect | Appreciation notes between partners | G-04 | 4 |
| `MEM` | Remember | Private timeline of memories, with photos | G-04 | 4 |
| `DATE` | Remember | Milestones and important dates, with reminders | G-04 | 4 |
| `NOTF` | Foundation | Web push notifications with per-category preferences | G-02, G-03, G-04 | 2–4 (per category as its feature ships) |
| `PWA` | Foundation | Installable app, offline reading while the session is open, update handling | G-01, G-05 | 1 |
| `AI` | — | Optional AI assistance for prayer and journal drafting, off by default | G-02, G-04 | 5 (Post-MVP) |

Four items are newly in scope compared with the v2.0 draft, all recorded in 05 §3: **account
deletion**, **data export**, **email verification**, and **leave couple** (Q-05, Q-09). None of
these existed as a requirement in v2.0. All four are needed to meet G-05 (trust, data rights);
their priorities are set in 02.

### 7.2 Out of scope (Won't, for now)

| Item | Reason |
|------|--------|
| Public social feed | Breaks principle 1 (private by default); not a couple-scoped experience. |
| Group prayer | Amorae is a two-person space (principle 2, DEC-12); group faith is a different product. |
| Public profiles | No public surface anywhere in the product, by design. |
| Community features | Out of positioning — Amorae serves a couple, not a community. |
| Voice or video calls | Large scope (media infrastructure, NAT traversal); existing tools already solve this well. |
| A full chat system | Large scope; general messaging apps already serve this need better than a purpose-built couple app could. |
| Payments | No monetisation model defined yet; adds compliance burden that serves no MVP goal. |
| Heavy gamification | Contradicts principle 3 (intentional, not addictive). |
| An AI chatbot | AI in Amorae is scoped to assistive, optional, human-reviewed drafting (DEC-15), not an open-ended chat interface. |
| Complex financial management | Different problem space; not a goal this product serves. |
| Heavy analytics surfaced to users | Adds surface area with no goal behind it; internal metrics stay first-party only (Q-14). |
| Native mobile apps | Constraint C-01: web-only PWA. Revisit only if PWA limitations meaningfully block a goal. |

The first version should be narrow and deeply polished rather than broad and shallow.

---

## 8. Release plan

Five build phases plus two gates. Exit criteria are concrete: every phase requires its Must
requirements at status **Implemented** per the definition of done in README §4 (real Go endpoint
and web client, tested, contract matches `docs/API.md`), the open questions due at that gate
resolved, and the relevant NFR categories checked.

| Phase / gate | Contents | Entry criteria | Exit criteria |
|--------------|----------|-----------------|----------------|
| **Phase 1 — Foundation** | `AUTH`, `ACCT`, `PAIR`, `PWA` shell, home, profile and settings | Project start | All Phase 1 Must requirements Implemented (README §4). Q-03, Q-04, Q-05, Q-08, Q-09, Q-14, Q-15, Q-20, Q-21 resolved. NFR-SEC and NFR-OBS baseline checks pass. |
| **Phase 2 — Faith** | `PRAY` (weekly prayer, setter rotation, completion, history), `NOTF` for prayer categories | Phase 1 exit met. Q-07 and Q-16 resolved (they shape the scheduler). | All Phase 2 Must requirements Implemented. Q-01 and Q-22 resolved. NFR-PERF and NFR-OFFL checked for the prayer flow. |
| **Phase 3 — Together** | `EVT`, `CAL`, `GOAL`, `CHAL`, `NOTF` for their categories | Phase 2 exit met. Q-23 resolved (challenge completion model). | All Phase 3 Must requirements Implemented. NFR-A11Y checked across the new screens. |
| **Phase 4 — Memories (MVP complete)** | `JRNL`, `APPR`, `MEM`, `DATE`, `NOTF` for their categories | Phase 3 exit met. Q-06 resolved (photo storage). | All Phase 4 Must requirements Implemented. NFR-PRIV checked, including photo handling. MVP is feature-complete. |
| **Private beta gate** | No new features; real couples using the MVP | Phase 4 exit met | A cohort of real couples has used the product for at least one full prayer-week cycle. Section 5 metrics have a first baseline reading. No open SEV1/SEV2 defect (severity levels in 03, NFR-OPS). NFR-SEC and NFR-PRIV spot-checked against real (not mock) data. Q-19 resolved. |
| **Public launch gate** | Go-live to the general public | Private beta feedback incorporated | Q-02, Q-10, Q-11, Q-12, and Q-17 resolved. A penetration test completed with findings triaged. A WCAG 2.2 AA audit completed with findings triaged. Backups configured with a tested restore. |
| **Phase 5 — Optional intelligence (Post-MVP)** | `AI`: optional prayer and journal assistance, behind a feature flag, off by default (DEC-15) | Public launch gate passed; Q-18 resolved | All Phase 5 Must requirements Implemented. AI remains fully optional per DEC-15 — verified by testing the product with the flag off. |

### 8.1 Current status (2026-09-22)

- **Implemented in Go:** authentication, profile, and email change.
- **In development in Go:** couple pairing (`PAIR`).
- **Mock only:** phases 2 through 4 exist as working UI against the in-browser mock; no Go
  endpoints exist yet for prayer, events, calendar, goals, challenges, journal, appreciation,
  memories, milestones, or notifications.
- **Not started:** Phase 5 (AI).

So Phase 1 is partly real, and Phases 2–4 have working screens but are not yet Implemented by
the definition of done in README §4 — see RISK-05.

---

## 9. Assumptions, constraints, and dependencies

### 9.1 Assumptions (A-NN)

| ID | Assumption |
|----|------------|
| A-01 | Launch market is Nigeria and the Nigerian diaspora, English-first (section 4.5) — to be confirmed by Product, not yet validated. |
| A-02 | A couple is exactly two people in one active pairing at a time (DEC-12); revisiting this is a later-stage decision, not an MVP one. |
| A-03 | Users have a smartphone capable of running a modern mobile browser that supports PWA install (Android Chrome, or iOS Safari 16.4+ for push). |
| A-04 | Users frequently have limited time, a weak or intermittent connection, and one free hand (principle 8) — this shapes both design and the offline behaviour in DEC-04/DEC-05. |
| A-05 | English is sufficient for MVP; no localisation work is planned before public launch. |

### 9.2 Constraints (C-NN)

| ID | Constraint |
|----|------------|
| C-01 | Web-only PWA. No native iOS or Android app in this plan. |
| C-02 | iOS web push only works for an installed PWA, and only on iOS 16.4 and later. |
| C-03 | Exactly one API instance runs until Q-13 is resolved, because MVP rate limiting is in-memory per process (DEC-11). |
| C-04 | AI infrastructure cost target is near $0 until Phase 5 proves demand (ties to DEC-15's feature flag). |
| C-05 | Postgres is the only stateful dependency; no Redis or queue in MVP (DEC-20). |
| C-06 | One couple per user in MVP (DEC-12) — the schema supports more later without a rewrite, but the product does not expose it. |

### 9.3 Dependencies (DEP-NN)

| ID | Dependency | Blocks |
|----|------------|--------|
| DEP-01 | Transactional email provider selection (Q-04) | Password reset, email verification, email-change notice |
| DEP-02 | Web Push services (browser-vendor push endpoints, VAPID keys) | `NOTF` |
| DEP-03 | Hosting provider and region (Q-12) | Sub-processor disclosure, cross-border transfer basis, public launch |
| DEP-04 | Object storage provider (Q-06) | `MEM` photo upload and storage |
| DEP-05 | AI provider selection, zero-retention terms (Q-18) | Phase 5 |

---

## 10. Risks

| ID | Description | Likelihood | Impact | Mitigation | Owner |
|----|-------------|------------|--------|------------|-------|
| RISK-01 | Two-sided activation risk: one partner signs up but the invited partner never joins, so the couple never activates. | M | H | Keep the invite flow low-friction (short code and link, DEC-08); nudge the inviter to re-share; track M-01/M-02 closely in beta. | Product |
| RISK-02 | iOS PWA limitations: push only works for an installed app, and Safari may evict site storage after roughly 7 days without use. | H | M | Prompt install with a clear value proposition before asking for push; design offline behaviour around DEC-04 rather than fighting it; monitor push delivery success. | Engineering |
| RISK-03 | Faith content (prayer points, prayer history) is sensitive personal data — religious belief under NDPA 2023 and GDPR Article 9 — creating consent and legal-review obligations. | M | H | Explicit consent language at sign-up; data minimisation in prayer content; legal review before public launch (Q-17). | Product + Legal |
| RISK-04 | Relationship breakup or coercive control: shared account structure could let one partner surveil the other, or leave one partner locked out or exposed after a breakup. | M | H | Leave-couple flow (Q-09) gives a clean, mutual exit; no covert or one-sided monitoring capability is built; both partners always have equal visibility into shared content; safety controls while paired are settled by Q-19. | Product |
| RISK-05 | Mock-only features (Phases 2–4) hide API contract problems that will surface only once the Go implementation catches up. | H | M | Contract tests against `docs/API.md`; phase exit criteria require **Implemented** status, not just a working mock (section 8). | Engineering |
| RISK-06 | Notification fatigue contradicts principle 3 ("intentional, not addictive") and drives opt-outs or uninstalls. | M | M | Per-category opt-out is a Must requirement for `NOTF`; cadence caps per category; monitor M-11 as a guardrail. | Product |
| RISK-07 | Low retention once the novelty of a new app wears off. | M | H | Weekly rhythm (prayer, planning) is the retention mechanism by design; track M-06 and iterate based on private beta feedback before public launch. | Product |
| RISK-08 | Spec drift: this document and the code disagree, and the disagreement goes unnoticed (already true in places — see 05 §3). | H | M | README §4's definition of done and the traceability matrix in 02 are updated in the same pull request as any implementation change; 05 §3 stays the single place drift is recorded, not silently resolved. | Engineering + Product |

---

## 11. Glossary

| Term | Meaning |
|------|---------|
| **Couple** | The shared account container. Owns all shared resources. |
| **Member / Partner** | One of the two users in a couple. |
| **Setter** | The partner responsible for setting prayer points in a given week. Alternates weekly, computed once at week creation (DEC-18). |
| **Prayer week** | A weekly container of prayer points, starting Sunday in the couple's timezone. |
| **Prayer point** | A single item to pray about within a prayer week. |
| **Completion** | An individual partner marking a prayer point as prayed. Independent per partner. |
| **Together hub** | The shared-life section that groups events, calendar, goals, challenges, journal, and memories. |
| **Couple-scoped** | Any resource that belongs to a couple and is visible only to its two members. |
| **BFF** | Backend-for-frontend. The Next.js layer the browser talks to; it holds the session and forwards requests to the Go API (DEC-02). |
| **PWA** | Progressive Web App — an installable, offline-capable web app; Amorae's only client platform (C-01). |
| **Mock mode** | A build-time flag (`NEXT_PUBLIC_API_MOCK`) under which the web app answers from an in-browser mock instead of the real Go API. Refused in production builds unless explicitly allowed. |
| **Paused (offline) change** | A write made while offline, held as a paused TanStack Query mutation in `localStorage`, and replayed on reconnect (DEC-05). |
| **Invite code** | A 6-character code (3 letters, 3 digits, shown as `ABC-123`) used to join a couple (DEC-08). |
| **SLO** | Service-level objective — a measurable operational target (for example, an error-rate or latency budget), defined in full in 03. |

Business rules behind these terms — limits, validation, and edge cases — live in 02, not here.

---

## 12. References

- [`02-functional-requirements.md`](./02-functional-requirements.md) — functional requirements
  and acceptance criteria.
- [`03-non-functional-requirements.md`](./03-non-functional-requirements.md) — non-functional
  requirements.
- [`04-design-specification.md`](./04-design-specification.md) — design specification.
- [`05-decisions-and-open-questions.md`](./05-decisions-and-open-questions.md) — decisions and
  open questions register.
- [`../ARCHITECTURE.md`](../ARCHITECTURE.md) — system architecture: module layout, request
  lifecycle, rate limiting, observability, PWA, and scaling path.
- [`../API.md`](../API.md) — the HTTP contract: conventions, envelopes, error codes, and
  endpoints (built and proposed).
- [`../adr/`](../adr/) — architecture decision records.
- [`../BRAND.md`](../BRAND.md) — palette, logo usage, and brand rules.

---

## Change log

| Version | Date | Change |
|---------|------|--------|
| 3.0 | 2026-09-22 | Rewritten as an industry-standard PRD. Added a problem statement (marked as an unvalidated hypothesis), compact personas with jobs-to-be-done, user classes, a stakeholders table, and a "who this is not for" list. Replaced the narrative MVP scope and roadmap with a module-level scope table, an out-of-scope table with reasons, and a release plan with concrete entry/exit criteria per phase plus private-beta and public-launch gates. Added goals G-01–G-05 with a metrics table (M-01–M-12) built around weekly meaningful actions instead of time-in-app or streaks, per principle 3. Added assumptions, constraints, dependencies, and a risk register (RISK-01–RISK-08). Removed the old "Decisions log" section; its content now lives in 05 as DEC-12, DEC-13, and DEC-15. |
| 2.0 | 2026-09-21 | Product foundation split out of the combined spec. |
| 1.0 | (prior) | Original combined product requirements and UX design spec. |
