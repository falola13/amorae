# Build guide: "Pray for me today" and the daily question

A working guide for building two features by hand. Every step names the existing code to copy,
so the new code reads like the rest of Amorae. Paths are from the repo root.

Build #1 first. It is smaller, it lives inside the prayers module, and it teaches you the
notification pattern you will reuse in #2.

Stop at each **Checkpoint** and send the diff for review before going on.

---

## Decisions to make before you start

Write your answer next to each. The guide assumes the **bold** default.

| # | Question | Default |
|---|---|---|
| D1 | Can you ask for prayer on a day with no published prayer week? | **Yes.** Requests don't depend on the weekly set |
| D2 | Can you edit your request after your partner has prayed for it? | **No.** It locks once prayed for, so "prayed for your interview" stays true |
| D3 | Does the notification show the request text? | **No.** It's lock-screen-safe: "Femi asked you to pray for something today". Same rule as answered prayers (FR-NOTF-005.AC1) |
| D4 | Daily question: can you answer a past day? | **No, today only.** |
| D5 | Daily question: if you skipped a day, does your partner's answer for it stay sealed? | **Yes, sealed forever.** That's the reason to answer. Alternative: it opens the next day |
| D6 | Where do the questions come from? | **A fixed list in Go code**, picked by date, the same for both partners. No admin screen |

---

# Feature 1: "Pray for me today"

**What it does:**
- Each of you can write one short line a day: what you'd like prayed for.
- Your partner sees it on Home and in prayer mode, and taps "I prayed for this".
- You get told, and Home shows "Femi prayed for you · 7:42".

## Step 1: Migration

Create `apps/api/migrations/00038_prayer_requests.sql`.
- **Copy the shape of:** `00037_anniversary_letters.sql` (one row per person per day, with a unique key).
- **Copy the preference column from:** `00036_challenge_editing.sql`.

The table `prayer_requests` needs these columns:
- `id UUID PK`
- `couple_id` FK → couples, cascade
- `author_id` FK → users, cascade (it's their own words, so it goes when they do)
- `asked_on DATE` (the couple's calendar day)
- `text` (non-empty, at most about 200 characters)
- `prayed_by UUID NULL` FK → users, `ON DELETE SET NULL`
- `prayed_at TIMESTAMPTZ NULL`
- `created_at`, `updated_at`

Also add:
- `UNIQUE (couple_id, author_id, asked_on)`
- a `CHECK` that `prayed_by` and `prayed_at` are both null or both set
- `ALTER TABLE notification_preferences ADD COLUMN prayer_requests BOOLEAN NOT NULL DEFAULT true`
- a `-- +goose Down` section that drops both

**Watch for:** `asked_on` must be the **couple's** date, not UTC. You'll pass it in from Go; don't use `CURRENT_DATE`.

## Step 2: Domain and repository (prayers module)

Put it in `apps/api/internal/modules/prayers/`. A new module isn't needed, because prayers already
knows the couple, the partner and "today".

- **Domain (`prayers.go`, or a new `requests.go`):**
  - a `Request` struct;
  - `ValidateRequestText(s)`: trim it, then reject empty text or more than 200 runes. Copy the letter
    validation in `anniversary/service.go` (`Write`).
- **Repository (`repository_postgres.go`), add four methods:**
  - `RequestsOn(ctx, coupleID, day)` returns both partners' requests for the day.
  - `KeepRequest(ctx, coupleID, authorID, day, text, now)` is an upsert on the unique key, keeping
    `created_at`. Copy `Keep` in `anniversary/repository_postgres.go:68-83`. It must **not**
    overwrite a row where `prayed_at` is set (D2), so add `WHERE prayer_requests.prayed_at IS NULL`
    to the `DO UPDATE`.
  - `DeleteRequest(ctx, coupleID, authorID, day)`
  - `MarkRequestPrayed(ctx, coupleID, requestAuthorID, day, prayedBy, now, done bool)` sets or
    clears `prayed_by`/`prayed_at`.
- Every query filters on `couple_id`, as all the prayers queries do.

## Step 3: Service

In `prayers/service.go`, follow `SetCompletion` (~line 190).

1. `cc := s.coupleContext(ctx, userID)` gives you `cc.Today` (the couple's date) and the partner.
2. If there's no partner, return `ErrWaitingForPartner`.
3. `AskForPrayer(ctx, userID, text)`: validate, keep, then call `s.poker.Poke()` so the
   notification goes out promptly.
4. `PrayedForRequest(ctx, userID, done)` marks **the partner's** request for today. If they haven't
   asked today, return a not-found error. Poke when `done` is true.
5. `TodayRequests(ctx, userID)` returns `{mine, theirs}`.

**Rule to enforce in the service:** you can't mark your own request as prayed. Write a test for it.

## Step 4: Handler and routes

In `prayers/handler.go`, next to the completion routes (~line 42):

```
GET    /prayers/requests/today            -> {mine, theirs}
PUT    /prayers/requests/today            -> body {text}; write or edit mine
DELETE /prayers/requests/today            -> withdraw mine
POST   /prayers/requests/today/prayed     -> I prayed for theirs
DELETE /prayers/requests/today/prayed     -> undo
```

- Use `r.HandleAuthed`, `authctx.UserID`, `httpx.Data` and `httpx.Error`, as the file already does.
- Add the DTO to `dto.go`: `{text, asked_on, prayed_at, prayed_by_me}`.
- Map the errors to friendly messages, never raw database errors:
  - editing after it's been prayed for → 409 "It's already been prayed for, so it can't be changed."
  - empty or too long → 422

**Checkpoint 1:** send me the migration, domain, repository, service and handler before any tests.

## Step 5: API tests

- **Domain:** a table test for `ValidateRequestText` covering empty, whitespace only, 200 and 201
  runes, and emoji.
- **Integration**, using `dbtest.New(t)` and the `pair(t, db)` helper in
  `prayers/repository_postgres_test.go`:
  - [ ] asking twice the same day keeps one row and updates the text
  - [ ] after the partner prays, the edit is refused and the text is unchanged
  - [ ] you can't mark your own request
  - [ ] another couple can't see or mark it
  - [ ] the next day starts empty (pin the clock with the fixed `now` func)
  - [ ] undo clears `prayed_by` and `prayed_at`
- Run them with a real database:
  `AMORAE_TEST_DATABASE_URL=... go test ./internal/modules/prayers/...`
  Without that variable the integration tests **skip**, and a skip is not a pass.

## Step 6: Notifications (two new kinds)

In `apps/api/internal/modules/notifications/`:

1. **Add the kinds** in `notifications.go` (~370):
   - `KindPrayerRequest = "prayer_request"` means your partner asked.
   - `KindPrayedForYou = "prayed_for_you"` means your partner prayed for yours.

   Add both to `AllKinds`.
2. **`kinds_test.go`:**
   - bump `len(AllKinds)` from 19 to 21;
   - add both kinds to the `asked` table (false) and the `perishable` table;
   - make `prayer_request` **perishable (true)**, because a request is worthless after its day;
   - make `prayed_for_you` not perishable (false).
3. **`budget.go`:** add `KindPrayerRequest` to `Perishable`.
4. **Preference:** add `PrayerRequests` to `Preferences`, `Defaults()` (true), `Patch` and `Apply`.
   Add it to the SELECT and upsert in `repository_postgres.go` (~33-95) and to the handler DTO
   (~62-115).
5. **Query** (`worker_repository.go`, `RecentlyWritten`): add two `UNION ALL` branches shaped like
   the letter branch at line 262.
   - **Asked:** `SELECT r.id, r.couple_id, r.author_id, r.created_at, $N::text, NULL, r.asked_on, false FROM prayer_requests r WHERE r.created_at >= $1`
   - **Prayed:** use `r.prayed_by` as the author and `r.prayed_at` as the time:
     `... FROM prayer_requests r WHERE r.prayed_at >= $1 AND r.prayed_by IS NOT NULL`.
     The outer query sends to the member who **isn't** the author, which here is the person who
     asked. That's exactly right.
   - The id must differ between the two branches, because the send key de-duplicates. Use the kind
     in the key, or the key format the other branches use.
   - Give "asked" a short `Settles` (about 1 minute) so a quick edit or withdrawal doesn't send.
     See the undo windows near line 310.
6. **Message** (`worker.go`, `ForWritten`): add two cases, gated by `c.Prefs.PrayerRequests`.
   - Asked: title "🙏 {author} asked you to pray for them", body "Open Amorae to see what for.",
     path `/`
   - Prayed: title "{author} prayed for you", no request text, path `/`
7. **Web mirror:**
   - add the two kinds to `NotificationKind` in `apps/web/src/lib/api/types.ts`;
   - add icons in `features/notifications/derive.ts`;
   - add a toggle row in `app/(app)/settings/notifications/page.tsx`.
8. **Test:** unit-test the two new `ForWritten` cases: the pref off sends nothing, and the body never
   contains the request text. Copy the shape of the existing `ForWritten` tests.

**Checkpoint 2:** send me the notification diff and the test output.

## Step 7: Web

In `apps/web/src/features/prayers/`:

1. **`types.ts`:** add `PrayerRequests { mine: PrayerRequest | null; theirs: PrayerRequest | null }`,
   and a zod schema in `lib/api/schemas.ts` if the other prayer types have one.
2. **`api.ts`:** add `requests.today()`, `ask(text)`, `withdraw()` and `prayed(done)`, using the
   `apiPath` template.
3. **`lib/query/keys.ts`:** add `keys.prayerRequests = ["prayers", "requests", "today"]`.
4. **`hooks.ts`:** `useTodayRequests()`. Copy `useWeek()`, enabled only when there's a partner.
5. **`writes.ts`:** use `defineWrite` for ask, withdraw and prayed.
   - Each write must declare `idempotent` or `keyed`. A PUT of today's text is idempotent
     ("same text, same day, same row"), and so is the prayed toggle.
   - `invalidates: [keys.prayerRequests]`
   - Add them to `resumableWrites` in `components/providers.tsx` so they queue offline.
6. **Home** (`features/home/home-screen.tsx`, near the Weekly prayer row ~238): add a small card.
   - **No request from you yet:** "What should {partner} pray for today?", a one-line input and Send.
   - **You've asked:** your line, then "Waiting" or "{partner} prayed for you · 7:42" (from `prayed_at`,
     formatted in the user's time).
   - **They've asked:** their line, with an "I prayed for this" button and undo.
7. **Prayer mode** (`features/prayers/components/prayer-session.tsx`): show the partner's request
   **first**, before the weekly points, with the same button. That's the moment it matters.

**Checkpoint 3:** send me screenshots of the Home card in all three states, on a phone width.

## Step 8: Finish
- [ ] `go test ./...` with the database variable set, and `go vet ./...`
- [ ] web: `npm run lint`, `npm run typecheck`, `npx vitest run`
- [ ] Try it as both of you: one phone asks, the other prays, and the first gets the notification
- [ ] Commit and push (Render runs the migration on start)

---

# Feature 2: The daily question, sealed until you both answer

**What it does:**
- One question a day, the same for both of you.
- You can't see your partner's answer until you've written yours.
- Every day you both answered becomes part of your story.

The seal works like the anniversary letters, but opens on a different condition:
- **Letters open on a date.** See `Letter.Open(today)` in `anniversary/anniversary.go`.
- **Answers open when you've answered too.**

The hiding must happen **in the API**, as `anniversary/service.go:81-99` (`assemble`) does. The web
must never receive the partner's text early. Hiding it in the UI isn't a seal.

## Step 1: The questions

Create a new module, `apps/api/internal/modules/questions/`. Copy the file layout of
`modules/anniversary` (service.go, handler.go, repository_postgres.go and a domain file).

- **`bank.go`:** `var bank = []string{ ... }` with at least 120 questions. Write them yourselves:
  that's the part that makes it yours.
- **`QuestionFor(coupleID uuid.UUID, day time.Time) (index int, text string)`:**
  - Use days since a fixed epoch, plus a number derived from the couple id, modulo `len(bank)`.
  - It's deterministic, so both partners get the same question.
  - Adding questions later only reshuffles **future** days, as long as each answer stores the text
    it was given (next step).

## Step 2: Migration

Create `00039_daily_answers.sql`, or `00038` if you build this first. Table `daily_answers`:
- `id`
- `couple_id` (cascade)
- `author_id` (cascade)
- `asked_on DATE`
- `question TEXT` (a snapshot of the question, so history survives edits to the bank)
- `text`
- `created_at`, `updated_at`
- `UNIQUE (couple_id, author_id, asked_on)`

Also add a preference column, `daily_question BOOLEAN NOT NULL DEFAULT true`.

## Step 3: Service, where the seal lives

`Today(ctx, userID)` returns:

```
{ asked_on, question,
  mine:  {text, created_at} | null,
  theirs: {text, created_at} | null,     // only when mine != null
  theirs_answered: bool }                // true even while sealed
```

- `Answer(ctx, userID, text)` writes today's answer only (D4). Validate it like the letter (trim,
  non-empty, max about 1000), keep it, and Poke.
- `Past(ctx, userID, before, limit)` lists past days. For each day:
  - show both answers if both of you answered;
  - show yours alone, with `theirs_answered`, if only you did;
  - otherwise the partner's stays sealed (D5).

  Use the same `assemble`-style function for today and the past, so there's one rule in one place.

**Checkpoint 4:** send me the service with its tests before the handler.

## Step 4: Tests that prove the seal
- [ ] before I answer, `theirs` is null and `theirs_answered` is true
- [ ] after I answer, `theirs` has their text
- [ ] the raw JSON for a sealed answer doesn't contain their text. Assert on the response body
  string, not just the struct.
- [ ] both partners get the same question for the same day; the next day gives a different one
- [ ] answering a past day is refused
- [ ] another couple can't read it

## Step 5: Routes, notification, story feed

- **Routes:** `GET /questions/today`, `PUT /questions/today` and `GET /questions?before=&limit=`.
  Wire the module in `apps/api/internal/app/app.go` the way anniversary is wired, and pass the
  worker as its Poker.
- **Notification:** add `KindQuestionAnswered`, using the same steps as Feature 1, Step 6.
  - Title "{author} answered today's question", body "Answer yours to see it.", path `/`
  - Mark it perishable.
  - Better still: in the query, only send it if the recipient hasn't answered yet, by adding
    `NOT EXISTS` on their row.
- **Our story:** add a `UNION ALL` branch in `timeline/repository_postgres.go` (75-219) for days where
  **both** answered, so they show in the feed. Update the timeline tests (`repository_postgres_test.go`).

## Step 6: Web
- `features/questions/` with `api.ts`, `hooks.ts`, `writes.ts` and `derive.ts`, following the
  prayers feature.
- **Home card:**
  - the question;
  - your input, or your answer;
  - the partner's answer behind a small seal while you haven't answered. Reuse the look of
    `SealedEnvelope` in `features/anniversary/sections.tsx:264`, smaller.
  - After you answer, reveal theirs with the flap animation that file already has.
- A past questions list, reached from the card ("Past questions"), using `GET /questions`.

**Checkpoint 5:** send me screenshots of sealed, answered by you only, and both revealed.

---

## Notification budget

These features add up to four more possible pushes a day per person. Check your cap and quiet hours
in Settings. All new kinds go through `Budget.Allows`, so the existing cap still holds. Perishable
kinds are dropped when they can't be sent in time, rather than arriving at midnight.

## Faith wording

The app's copy is faith-worded everywhere today, and nothing switches it to secular wording yet
(see `docs/PRODUCT-GAPS.md`). Keep the new copy consistent with the rest and don't build a
vocabulary layer as part of this.
