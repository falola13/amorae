# Amorae requirements

**Product:** Amorae, a private, mobile-first PWA for two people to build their shared life:
connection, planning, shared goals, faith and memories.
**Set version:** 3.0 · **Status:** Draft for review · **Last updated:** 2026-09-22

This folder is the requirement baseline: *what* Amorae must do and *how well*. It is
structured along the lines of ISO/IEC/IEEE 29148 (stakeholder, system and software
requirements). *How* the system is built lives next door, in the architecture docs, and the
requirements link to them instead of repeating them.

---

## 1. Document map

| ID | Document | Owner | Answers |
|----|----------|-------|---------|
| AMR-REQ-01 | [Product requirements (PRD)](./01-product-requirements.md) | Product | Why we are building this, for whom, what success looks like, scope, release plan, risks. |
| AMR-REQ-02 | [Functional requirements (SRS)](./02-functional-requirements.md) | Engineering | What the system shall do, feature by feature, with acceptance criteria, data requirements, business rules and a traceability matrix. |
| AMR-REQ-03 | [Non-functional requirements](./03-non-functional-requirements.md) | Engineering | How well: performance, availability, security, privacy, accessibility, compatibility, offline, observability, operations. |
| AMR-REQ-04 | [Design specification](./04-design-specification.md) | Design | Tokens, typography, components and their states, the screen inventory, interaction and content rules. |
| AMR-REQ-05 | [Decisions and open questions](./05-decisions-and-open-questions.md) | Product | The single register of decisions (`DEC`) and open questions (`Q`), and how the plan changed from v2.0. |

Related documentation (the "how", referenced rather than duplicated):

| Document | Holds |
|----------|-------|
| [`docs/ARCHITECTURE.md`](../ARCHITECTURE.md) | Dependency rules, module layout, request lifecycle, rate limiting, observability, PWA, scaling path |
| [`docs/API.md`](../API.md) | The HTTP contract: conventions, envelopes, error codes, endpoints (built and proposed) |
| [`docs/adr/`](../adr/) | Architecture decision records |
| [`docs/BRAND.md`](../BRAND.md) | Palette, logo usage, brand rules |

Reading order: 01 first (everyone), then the document for your discipline, and 05 whenever an
ID from it appears.

---

## 2. Identifier scheme

Every requirement has a permanent ID. IDs are never reused or renumbered; a removed requirement
stays in place with status **Withdrawn** and a reason.

| Prefix | Meaning | Defined in |
|--------|---------|------------|
| `G-NN` | Product goal | 01 |
| `M-NN` | Success metric (KPI or guardrail) | 01 |
| `RISK-NN` | Product or delivery risk | 01 |
| `FR-<MODULE>-NNN` | Functional requirement | 02 |
| `FR-<MODULE>-NNN.ACn` | Acceptance criterion of that requirement | 02 |
| `BR-<MODULE>-NN` | Business rule (a rule several requirements rely on) | 02 |
| `NFR-<CATEGORY>-NNN` | Non-functional requirement | 03 |
| `SCR-NN` | Screen in the screen inventory | 04 |
| `DEC-NN` | Decision | 05 (and ADRs) |
| `Q-NN` | Open question | 05 |

**Functional modules:** `AUTH` (sign-up, sign-in, sessions, password reset), `ACCT` (profile,
email change, deletion, export), `PAIR` (couple creation, invites, joining, leaving), `PRAY`
(weekly prayer), `EVT` (shared events), `CAL` (couple calendar), `GOAL` (shared goals), `CHAL`
(challenges), `JRNL` (shared journal), `APPR` (appreciation notes), `MEM` (memories), `DATE`
(milestones and important dates), `NOTF` (notifications and reminders), `PWA` (install,
offline, updates), `AI` (optional assistance).

**Non-functional categories:** `PERF`, `AVAIL`, `SCALE`, `SEC`, `PRIV`, `A11Y`, `COMPAT`,
`OFFL`, `OBS`, `MAINT`, `OPS`.

---

## 3. How requirements are written

**Keywords.** "Shall" is mandatory, "should" is recommended, "may" is optional. Each
requirement states one testable thing in the form *actor + shall + behaviour + condition*
(the EARS patterns: *When* ‹trigger›, *While* ‹state›, *If* ‹unwanted event› *then*, *Where*
‹feature is on›).

**Quality bar** (from ISO/IEC/IEEE 29148): each requirement is necessary, singular,
unambiguous, verifiable, feasible, free of implementation detail unless the detail is itself the
requirement, and traceable to a goal.

**Attributes** carried by every functional and non-functional requirement:

| Attribute | Values |
|-----------|--------|
| Priority (MoSCoW) | **Must** (release fails without it) · **Should** (important, can slip one release) · **Could** (nice to have) · **Won't** (explicitly out of scope for now) |
| Release | **MVP** (phases 1–4) · **Post-MVP** (phase 5 and later) |
| Implementation status (as of the document date) | **Implemented** (real API and web, tested) · **Partial** · **Mock only** (web works against the in-browser mock; Go endpoint not built) · **Not started** |
| Verification | **Test** (automated) · **Demo** (manual scripted check) · **Inspection** (review of code or config) · **Analysis** (measurement or calculation) |

**Acceptance criteria** are written as *Given / When / Then* and numbered `.AC1`, `.AC2`, …
under their requirement, so tests can name exactly what they cover.

**Blocked requirements** show *Not started — Blocked by Q-NN* in the status cell and are not built
until the question is resolved.

---

## 4. Definition of done for a requirement

A requirement moves to **Implemented** only when all of the following hold:

1. The Go endpoint and the web client both exist (no mock in the path).
2. Every acceptance criterion has an automated test, or a documented manual check where
   automation is impractical (for example PWA install on iOS).
3. [`docs/API.md`](../API.md) and `apps/web/src/lib/api/types.ts` describe the same contract.
4. The relevant NFRs (security, accessibility, performance budgets) have been checked for it.
5. The traceability matrix in 02 is updated in the same pull request.

---

## 5. Change control

- Requirement changes go through a pull request reviewed by the owner of the document.
- Bump the document version (major for scope or behaviour changes, minor for clarification) and
  add a change-log line saying what changed and why.
- A change that contradicts a `DEC` needs a new `DEC` that supersedes it; a change to
  architecture also needs an ADR.
- If the code and a requirement disagree, the disagreement is recorded (as a `Q`, or as a status
  of **Partial**), never resolved silently in either direction.

---

## Change log

| Version | Date | Change |
|---------|------|--------|
| 3.0 | 2026-09-22 | Moved the specs into `docs/requirements/` and restructured them to an industry format: PRD, SRS, a new NFR document, design specification, and a single decisions and open-questions register. Introduced permanent IDs, MoSCoW priority, implementation status, Given/When/Then acceptance criteria, traceability and change control. Reconciled the spec with the code as built (see 05 §3). |
| 2.0 | 2026-09-21 | Split the single spec into foundation / engineering / design, added acceptance criteria, a data model, API conventions, design tokens and non-functional targets. |
| 1.0 | (prior) | Original combined product requirements and UX design spec. |
