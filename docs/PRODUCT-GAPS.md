# Product gaps and recommendations

Written 2026-09-25, against `1ebb145`. Every claim here was checked against the
code, not against the roadmap.

## 0. The roadmap is out of date, which is itself the first problem

`docs/ROADMAP.md` still says the half that makes this Amorae has not been
built. It lists Prayers, Events, Calendar, Goals, Challenges, Journal,
Appreciations, Memories, Milestones and notification preferences as "screens
with no API behind them yet", and the worker, push delivery and photo storage
as "not started at all".

All of it ships. Push has been delivered to a Chrome device and an iPhone at
the same instant; photos upload to Cloudinary, deliver over signed
authenticated URLs and are deleted at the origin when removed; the worker runs
on a five-minute tick.

Four gaps it lists are also closed: idempotency keys (`cd5d6f0`, verified with
an offline create in airplane mode), per-partner challenge marks (DEC-30),
photo storage (Q-06), and "no web tests at all" (there are 34).

That matters more than it looks. The roadmap is the document you would open to
decide what to do next, and it would point you at work that is finished. Fix it
first or stop trusting it.

## 1. Gaps that are real today

Ranked by what I would fix first.

### 1. The faith consent gates nothing — the only promise the product breaks

[FR-AUTH-012, FR-PAIR-009, DEC-29]

Sign-up asks "Show me faith content (prayers and scripture). Optional." and
records the answer with its policy version. Nothing then reads it.
`FaithConsent` appears in the register request, in validation, and in
`recordConsents` — and nowhere else in either app. Untick it and you still get
prayer weeks, scripture, and "Two hearts, one faith".

This is asking someone a question, filing the answer, and ignoring it. It is
also the foundation of a secular mode: a couple's space uses faith vocabulary
only while both partners hold consent. One gate, two outcomes — a promise kept
and a market widened.

### 2. The prayer reflection is one field where the requirements want two

[FR-PRAY-006]

`types.ts` has a single `reflection?: string` on the week. The requirement is
per-partner. As it stands both partners write into one field and the second
overwrites the first — silently, with no indication anything was lost.

This is the same class of bug DEC-30 settled for challenge marks: one shared
flag where two people need one each. It was right there and it is still here.
Cheap to change now; expensive once a year of reflections exists.

### 3. Fifteen notification kinds, and a cap holding them back

Mine, added today. The ratio is better than it was — roughly five delight to
seven obligation, where it used to be three to seven — but the total is
fifteen, and the daily cap is now the only thing between the two of you and a
stream.

I would not add a sixteenth without removing one. The measure is not whether
the cap holds; it is whether you both still look at them.

### 4. No email verification, which also leaks whether an address exists

[Q-05]

There is no verification flow and no `email_verified` anywhere in the schema.
Registration answers `409 email_taken`, so anyone can test an address against
the service. One fix closes both.

### 5. One API instance, enforced by nothing

[DEC-11, Q-13]

The rate limiter is in memory. A second Render instance would silently double
every limit rather than failing loudly. Fine at one instance, which is where
you are; a trap the day anyone scales it, because nothing says so at runtime.

### 6. No Content-Security-Policy on pages

[NFR-SEC-015] — only the service worker has one. Verified absent.

### 7. CI checks nothing about formatting, vulnerabilities or dependencies

[NFR-SEC-019] `ci.yml` has no `govulncheck`, no `npm audit`, no
`prettier --check`, and there is no Dependabot config. Formatting drift proved
real today: twelve files had accumulated, which is what a check that nobody
runs looks like.

### 8. Nothing reports errors

[NFR-OBS-006, Q-14] When something breaks, the way you find out is your
girlfriend telling you. That was true of every production fault today — the
`photo_id` column, the prepared-statement collision, the `[]uuid.UUID` encode.
Each surfaced as a screenshot in chat.

### 9. Web tests are thin rather than absent

34 tests across three files, against 21 passing Go packages. The untested parts
are the logic-heavy ones: the offline write queue, optimistic updates and
rollback, the replay rules. Those are where a subtle bug would hide longest.

## 2. What is not a gap

Worth writing down so nobody "fixes" a decision:

- **No cold-start offline reading** [DEC-04, Q-01] — deliberate.
- **Challenges are not streaks** — deliberate, and the reason the second-mark
  notification is the only one that treats a day as shared.
- **Amounts withheld from lock screens** [FR-NOTF-005.AC2] — deliberate.
- **Straight to `main`, no branches** — deliberate, though a
  `worktree-amorae-idem` branch now exists from a second session's worktree.

## 3. Recommendation

**Do the faith consent gate next.** It is small, it is the only shipped promise
currently broken, and it is the one item on this list that is a product
direction rather than a repair — a secular mode falls out of the same switch.

Then the reflection field, because it is a data-loss bug and it gets more
expensive every week.

Then email verification, which is two problems for one piece of work.

Everything below that is hygiene, and hygiene is worth doing in a batch when
the product work pauses — not instead of it.
