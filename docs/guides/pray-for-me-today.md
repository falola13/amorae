# Build "Pray for me today", step by step

This walks you through every file. Each step says:
- **where** the file is;
- **what** to type, as real code written against this repo;
- **why**, in a line or two;
- **check:** how to know the step worked before moving on.

Type the code yourself rather than pasting it blind. Reading it as you type is how you'll be able
to change it later. Whenever you hit a **Checkpoint**, send me the diff (`git diff`) and the
output of the check.

**The feature:**
- Each morning either of you can write one line: what you'd like prayed for today.
- The other sees it on Home and taps "I prayed for this".
- You get a notification, and your Home shows "Ada prayed for you · 7:42".

**The decisions this guide uses:**
- A request locks once it's been prayed for.
- Notifications never show the request text.
- A request works even on a day with no published prayer week.

---

## Step 0: Run the app on your machine (once)

You need Docker Desktop open. Run these from the repo root.

```bash
docker compose start db
```

```bash
cd apps/api && go run ./cmd/migrate up
```

Then, in two separate terminals:

```bash
npm run dev:api
```

```bash
npm run dev:web
```

Open http://localhost:3000. You need two users who are a couple:
1. Open a normal window and register Femi.
2. Open a private/incognito window and register Ada.
3. In Femi's window create the couple and copy the invite code. In Ada's window, join with it.

The test accounts I made earlier also still exist in that database.

**A test database for the Go tests (once).** The integration tests skip themselves unless they're
given a database. A **skip is not a pass**.

```bash
docker compose exec db createdb -U amorae amorae_test
```

In PowerShell, set this in each new terminal before running Go tests:

```powershell
$env:AMORAE_TEST_DATABASE_URL = "postgres://amorae:amorae@localhost:5434/amorae_test?sslmode=disable"
```

**Check:** `go test ./internal/modules/prayers/...` from `apps/api` prints `ok`, and
`go test -v ./internal/modules/prayers/... | Select-String SKIP` prints nothing.

---

## Step 1: The table

Create **`apps/api/migrations/00038_prayer_requests.sql`**:

```sql
-- +goose Up
-- One line a partner asks the other to pray for, for one day. The day is the
-- couple's own calendar day, decided in Go (never CURRENT_DATE, which is UTC).
-- Once the other partner has prayed for it, it is kept as it was.

CREATE TABLE prayer_requests (
    id         UUID PRIMARY KEY,
    couple_id  UUID NOT NULL REFERENCES couples (id) ON DELETE CASCADE,
    author_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    asked_on   DATE NOT NULL,
    text       TEXT NOT NULL,
    prayed_by  UUID REFERENCES users (id) ON DELETE SET NULL,
    prayed_at  TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (couple_id, author_id, asked_on),
    -- Both or neither: "prayed" without who, or who without when, is nonsense.
    CHECK ((prayed_by IS NULL) = (prayed_at IS NULL))
);

-- Whether the other partner wants to hear about it. On by default.
ALTER TABLE notification_preferences
    ADD COLUMN prayer_requests BOOLEAN NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE notification_preferences DROP COLUMN prayer_requests;
DROP TABLE prayer_requests;
```

**Why:**
- `UNIQUE (couple_id, author_id, asked_on)` is what makes "one request per person per day" true
  even if the app sends twice.
- `prayed_by` is `SET NULL`, not `CASCADE`: if the one who prayed deletes their account, the
  request shouldn't vanish from the other person's history.

**Check:** run `go run ./cmd/migrate up` from `apps/api`. It should apply 00038 with no error.
Then run `go run ./cmd/migrate down` followed by `up` again, to prove the Down section works.

---

## Step 2: The domain file

Create **`apps/api/internal/modules/prayers/requests.go`**:

```go
package prayers

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

// maxRequestRunes keeps a request to one line on a phone.
const maxRequestRunes = 200

// Request is one partner asking the other to pray for something today.
type Request struct {
	ID        uuid.UUID
	CoupleID  uuid.UUID
	AuthorID  uuid.UUID
	AskedOn   time.Time // a calendar day, midnight UTC, like prayed_on
	Text      string
	PrayedBy  *uuid.UUID
	PrayedAt  *time.Time
	CreatedAt time.Time
}

var (
	ErrNoRequest = apperr.NotFound("no_request_today",
		"They haven’t asked for anything today.")
	ErrAlreadyPrayedFor = apperr.Conflict("already_prayed_for",
		"It’s already been prayed for, so it can’t be changed now.")
)

// ValidateRequestText trims the line and bounds it. Empty is an error here,
// unlike a reflection: an empty request asks for nothing.
func ValidateRequestText(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", apperr.Validation(map[string]string{
			"text": "Write what you’d like prayer for.",
		})
	}
	if utf8.RuneCountInString(text) > maxRequestRunes {
		return "", apperr.Validation(map[string]string{
			"text": fmt.Sprintf("Keep it under %d characters.", maxRequestRunes),
		})
	}
	return text, nil
}
```

**Why:** errors are built with `apperr`, which is how the API turns them into friendly JSON. The
user never sees a database error. Compare `ValidateReflection` in `prayers.go` (around line 228).

**Check:** from `apps/api`, run `go build ./...`. No output means it compiled.

---

## Step 3: The repository (SQL)

Open **`apps/api/internal/modules/prayers/repository_postgres.go`** and add these three methods at
the bottom. They copy the shape of `SetCompletion` in the same file, and of `Keep` in
`anniversary/repository_postgres.go:68-83`.

```go
// RequestsOn is both partners' requests for one day, at most one each.
func (r *PostgresRepository) RequestsOn(ctx context.Context, coupleID uuid.UUID, day time.Time) ([]Request, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT id, couple_id, author_id, asked_on, text, prayed_by, prayed_at, created_at
		FROM prayer_requests
		WHERE couple_id = $1 AND asked_on = $2
	`, coupleID, day)
	if err != nil {
		return nil, fmt.Errorf("reading prayer requests: %w", err)
	}
	defer rows.Close()

	var out []Request
	for rows.Next() {
		var q Request
		if err := rows.Scan(&q.ID, &q.CoupleID, &q.AuthorID, &q.AskedOn, &q.Text,
			&q.PrayedBy, &q.PrayedAt, &q.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning a prayer request: %w", err)
		}
		out = append(out, q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading prayer requests: %w", err)
	}
	return out, nil
}

// KeepRequest writes today's request, or rewrites it — but never once it has
// been prayed for. The WHERE on the update is what enforces that: a locked
// row is left alone, so no row is affected and we say why.
func (r *PostgresRepository) KeepRequest(
	ctx context.Context, coupleID, authorID uuid.UUID, day time.Time, text string, at time.Time,
) error {
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("making a request id: %w", err)
	}
	tag, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO prayer_requests (id, couple_id, author_id, asked_on, text, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		ON CONFLICT (couple_id, author_id, asked_on)
		DO UPDATE SET text = EXCLUDED.text, updated_at = EXCLUDED.updated_at
		WHERE prayer_requests.prayed_at IS NULL
	`, id, coupleID, authorID, day, text, at)
	if err != nil {
		return fmt.Errorf("keeping a prayer request: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAlreadyPrayedFor
	}
	return nil
}

// DeleteRequest withdraws today's request, unless it has been prayed for.
// Deleting one that isn't there is fine: the second tap of "withdraw" after a
// dropped response lands the same way as the first.
func (r *PostgresRepository) DeleteRequest(ctx context.Context, coupleID, authorID uuid.UUID, day time.Time) error {
	if _, err := r.db.Q(ctx).Exec(ctx, `
		DELETE FROM prayer_requests
		WHERE couple_id = $1 AND author_id = $2 AND asked_on = $3 AND prayed_at IS NULL
	`, coupleID, authorID, day); err != nil {
		return fmt.Errorf("withdrawing a prayer request: %w", err)
	}
	return nil
}

// MarkRequestPrayed sets or clears "prayed for" on the request `authorID`
// made today. Returns ErrNoRequest when there is none to mark.
func (r *PostgresRepository) MarkRequestPrayed(
	ctx context.Context, coupleID, authorID uuid.UUID, day time.Time, prayedBy uuid.UUID, done bool, at time.Time,
) error {
	var by *uuid.UUID
	var when *time.Time
	if done {
		by, when = &prayedBy, &at
	}
	tag, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE prayer_requests
		SET prayed_by = $4, prayed_at = COALESCE(prayed_at, $5)
		WHERE couple_id = $1 AND author_id = $2 AND asked_on = $3
	`, coupleID, authorID, day, by, when)
	if err != nil {
		return fmt.Errorf("marking a request prayed for: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNoRequest
	}
	return nil
}
```

**One subtlety in `MarkRequestPrayed`.**
- `COALESCE(prayed_at, $5)` keeps the **first** time someone prayed, so tapping twice doesn't move
  the time.
- When undoing, `$5` is NULL, so `COALESCE(prayed_at, NULL)` would keep the old time and break the
  CHECK. **Fix this yourself:** when `done` is false, the SET must clear both columns. One way is
  two different UPDATE statements, one for done and one for undo, as `SetCompletion` does with
  INSERT and DELETE.
- This is the one deliberate gap in this guide. Tell me how you solved it at Checkpoint 1.

Now add the four methods to the **`Repository` interface** at the top of `service.go`:

```go
	RequestsOn(ctx context.Context, coupleID uuid.UUID, day time.Time) ([]Request, error)
	KeepRequest(ctx context.Context, coupleID, authorID uuid.UUID, day time.Time, text string, at time.Time) error
	DeleteRequest(ctx context.Context, coupleID, authorID uuid.UUID, day time.Time) error
	MarkRequestPrayed(ctx context.Context, coupleID, authorID uuid.UUID, day time.Time, prayedBy uuid.UUID, done bool, at time.Time) error
```

**Check:** `go build ./...` compiles.

---

## Step 4: The service

Add to **`apps/api/internal/modules/prayers/service.go`**. The pattern is `SetCompletion`, which
gets the couple, works out "today", does the work, then pokes.

```go
// TodayRequests is what each of you asked for today, from the caller's side:
// mine is the caller's own, theirs is the partner's. Either may be nil.
func (s *Service) TodayRequests(ctx context.Context, userID uuid.UUID) (mine, theirs *Request, err error) {
	cc, err := s.coupleContext(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	partner := cc.Partner(userID)
	if partner == (uuid.UUID{}) {
		return nil, nil, ErrWaitingForPartner
	}
	all, err := s.repo.RequestsOn(ctx, cc.CoupleID, cc.Today)
	if err != nil {
		return nil, nil, err
	}
	for i := range all {
		switch all[i].AuthorID {
		case userID:
			mine = &all[i]
		case partner:
			theirs = &all[i]
		}
	}
	return mine, theirs, nil
}

// AskForPrayer writes or rewrites the caller's request for today.
func (s *Service) AskForPrayer(ctx context.Context, userID uuid.UUID, text string) error {
	clean, err := ValidateRequestText(text)
	if err != nil {
		return err
	}
	cc, err := s.coupleContext(ctx, userID)
	if err != nil {
		return err
	}
	if cc.Partner(userID) == (uuid.UUID{}) {
		return ErrWaitingForPartner
	}
	if err := s.repo.KeepRequest(ctx, cc.CoupleID, userID, cc.Today, clean, s.now()); err != nil {
		return err
	}
	// Only the first write is announced (the worker reads created_at), so a
	// poke on an edit costs one pass and sends nothing.
	s.poker.Poke()
	return nil
}

// WithdrawRequest takes back the caller's request for today.
func (s *Service) WithdrawRequest(ctx context.Context, userID uuid.UUID) error {
	cc, err := s.coupleContext(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.DeleteRequest(ctx, cc.CoupleID, userID, cc.Today)
}

// PrayedForPartner marks — or unmarks — the partner's request for today.
// There is no request id in the call, so marking your own can't happen.
func (s *Service) PrayedForPartner(ctx context.Context, userID uuid.UUID, done bool) error {
	cc, err := s.coupleContext(ctx, userID)
	if err != nil {
		return err
	}
	partner := cc.Partner(userID)
	if partner == (uuid.UUID{}) {
		return ErrWaitingForPartner
	}
	if err := s.repo.MarkRequestPrayed(ctx, cc.CoupleID, partner, cc.Today, userID, done, s.now()); err != nil {
		return err
	}
	if done {
		s.poker.Poke()
	}
	return nil
}
```

**Check:** `go build ./...` compiles.

---

## Step 5: The routes

In **`apps/api/internal/modules/prayers/handler.go`**:

**5a.** Add these four lines to the `service` interface at the top:

```go
	TodayRequests(ctx context.Context, userID uuid.UUID) (mine, theirs *Request, err error)
	AskForPrayer(ctx context.Context, userID uuid.UUID, text string) error
	WithdrawRequest(ctx context.Context, userID uuid.UUID) error
	PrayedForPartner(ctx context.Context, userID uuid.UUID, done bool) error
```

**5b.** Add these routes in `RegisterRoutes`:

```go
	r.HandleAuthed("GET /prayers/requests/today", http.HandlerFunc(h.todayRequests))
	r.HandleAuthed("PUT /prayers/requests/today", http.HandlerFunc(h.askForPrayer))
	r.HandleAuthed("DELETE /prayers/requests/today", http.HandlerFunc(h.withdrawRequest))
	r.HandleAuthed("POST /prayers/requests/today/prayed", http.HandlerFunc(h.prayedForPartner))
	r.HandleAuthed("DELETE /prayers/requests/today/prayed", http.HandlerFunc(h.unprayedForPartner))
```

**5c.** Add the handlers and the JSON shape at the bottom of the file. Every write answers with
the fresh `{mine, theirs}`, just as every prayer write answers with the whole week.

```go
// requestDTO is one request as the app sees it. prayed_at is when the other
// person prayed for it; nil means not yet.
type requestDTO struct {
	Text     string     `json:"text"`
	AskedOn  string     `json:"asked_on"`
	PrayedAt *time.Time `json:"prayed_at"`
}

type todayRequestsDTO struct {
	Mine   *requestDTO `json:"mine"`
	Theirs *requestDTO `json:"theirs"`
}

func toRequestDTO(q *Request) *requestDTO {
	if q == nil {
		return nil
	}
	return &requestDTO{Text: q.Text, AskedOn: q.AskedOn.Format("2006-01-02"), PrayedAt: q.PrayedAt}
}

// respondRequests answers any request call with today's state, or the error.
func (h *Handler) respondRequests(w http.ResponseWriter, r *http.Request, userID uuid.UUID, err error) {
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	mine, theirs, err := h.svc.TodayRequests(r.Context(), userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusOK, todayRequestsDTO{Mine: toRequestDTO(mine), Theirs: toRequestDTO(theirs)})
}

func (h *Handler) todayRequests(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	h.respondRequests(w, r, userID, nil)
}

type askRequest struct {
	Text string `json:"text"`
}

func (h *Handler) askForPrayer(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	var req askRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	h.respondRequests(w, r, userID, h.svc.AskForPrayer(r.Context(), userID, req.Text))
}

func (h *Handler) withdrawRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	h.respondRequests(w, r, userID, h.svc.WithdrawRequest(r.Context(), userID))
}

func (h *Handler) prayedForPartner(w http.ResponseWriter, r *http.Request) {
	h.setPrayedForPartner(w, r, true)
}

func (h *Handler) unprayedForPartner(w http.ResponseWriter, r *http.Request) {
	h.setPrayedForPartner(w, r, false)
}

func (h *Handler) setPrayedForPartner(w http.ResponseWriter, r *http.Request, done bool) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	h.respondRequests(w, r, userID, h.svc.PrayedForPartner(r.Context(), userID, done))
}
```

`handler.go` will need `"time"` added to its imports. Your editor, or `go build`, will tell you.

**Check:**
- `go build ./...` compiles.
- Restart `npm run dev:api` so the new routes load.
- Use the app in the browser. The routes are only reachable through it, because it holds your
  session cookie.
- Quicker: open DevTools Console on http://localhost:3000 while logged in as Femi, and run:

```js
await fetch("/api/v1/prayers/requests/today", {method:"PUT", headers:{"Content-Type":"application/json"}, body: JSON.stringify({text:"My interview at 2"})}).then(r=>r.json())
```

You should get `{data: {mine: {text: "My interview at 2", ...}, theirs: null}}`. As Ada,
`GET /api/v1/prayers/requests/today` should show it under `theirs`.

## Checkpoint 1

Send me:
- `git diff`;
- your fix for the undo gap in Step 3;
- the two console outputs.

---

## Step 6: Tests

Add these to **`apps/api/internal/modules/prayers/repository_postgres_test.go`**. They use the
existing `pair`, `fakeCouples` and `countingPoker` helpers at the top of that file, and copy the
setup of `TestService_SetCompletion_OnlyTodaysPointsInTheCurrentWeek`.

```go
func TestService_PrayerRequests(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	coupleID, ada, ben := pair(t, db)
	repo := prayers.NewPostgresRepository(db)
	cc := prayers.CoupleContext{
		CoupleID: coupleID,
		Location: time.UTC,
		Members: []prayers.Member{
			{UserID: ada, JoinedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
			{UserID: ben, JoinedAt: time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)},
		},
	}
	now := time.Date(2026, 9, 22, 8, 0, 0, 0, time.UTC)
	poker := &countingPoker{}
	svc := prayers.NewService(repo, fakeCouples{cc: cc}, func() time.Time { return now }, poker)

	t.Run("asking twice the same day keeps one request, with the new words", func(t *testing.T) {
		if err := svc.AskForPrayer(ctx, ada, "The interview"); err != nil {
			t.Fatalf("first ask: %v", err)
		}
		if err := svc.AskForPrayer(ctx, ada, "The interview at two"); err != nil {
			t.Fatalf("second ask: %v", err)
		}
		mine, _, err := svc.TodayRequests(ctx, ada)
		if err != nil || mine == nil || mine.Text != "The interview at two" {
			t.Fatalf("mine = %+v, err %v", mine, err)
		}
	})

	t.Run("the partner sees it as theirs", func(t *testing.T) {
		_, theirs, err := svc.TodayRequests(ctx, ben)
		if err != nil || theirs == nil || theirs.Text != "The interview at two" {
			t.Fatalf("theirs = %+v, err %v", theirs, err)
		}
	})

	t.Run("once prayed for, it can't be changed or withdrawn", func(t *testing.T) {
		if err := svc.PrayedForPartner(ctx, ben, true); err != nil {
			t.Fatalf("PrayedForPartner: %v", err)
		}
		if err := svc.AskForPrayer(ctx, ada, "Something else"); !errors.Is(err, prayers.ErrAlreadyPrayedFor) {
			t.Errorf("edit after prayer = %v, want ErrAlreadyPrayedFor", err)
		}
		mine, _, _ := svc.TodayRequests(ctx, ada)
		if mine == nil || mine.PrayedAt == nil {
			t.Fatalf("mine = %+v, want it marked prayed", mine)
		}
	})

	t.Run("undo clears it", func(t *testing.T) {
		if err := svc.PrayedForPartner(ctx, ben, false); err != nil {
			t.Fatalf("undo: %v", err)
		}
		mine, _, _ := svc.TodayRequests(ctx, ada)
		if mine == nil || mine.PrayedAt != nil || mine.PrayedBy != nil {
			t.Errorf("after undo mine = %+v, want unmarked", mine)
		}
	})

	t.Run("praying when they haven't asked is ErrNoRequest", func(t *testing.T) {
		if err := svc.PrayedForPartner(ctx, ada, true); !errors.Is(err, prayers.ErrNoRequest) {
			t.Errorf("= %v, want ErrNoRequest (Ben never asked)", err)
		}
	})

	t.Run("the next day starts empty", func(t *testing.T) {
		tomorrow := prayers.NewService(repo, fakeCouples{cc: cc},
			func() time.Time { return now.Add(24 * time.Hour) }, poker)
		mine, theirs, err := tomorrow.TodayRequests(ctx, ada)
		if err != nil || mine != nil || theirs != nil {
			t.Errorf("tomorrow: mine %+v theirs %+v err %v", mine, theirs, err)
		}
	})
}

func TestValidateRequestText(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		ok   bool
	}{
		{"empty", "", false},
		{"only spaces", "   ", false},
		{"one line", "My interview at two", true},
		{"exactly 200", strings.Repeat("a", 200), true},
		{"201", strings.Repeat("a", 201), false},
		{"emoji count as one", strings.Repeat("🙏", 200), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := prayers.ValidateRequestText(tc.in)
			if (err == nil) != tc.ok {
				t.Errorf("err = %v, want ok=%v", err, tc.ok)
			}
		})
	}
}
```

Add `"strings"` to that file's imports.

The "undo clears it" subtest **will fail until you've fixed the gap in Step 3**. That's
deliberate: it proves the test is checking the right thing.

**Check:** with `AMORAE_TEST_DATABASE_URL` set (Step 0), run this from `apps/api`:

```bash
go test -v -run "PrayerRequests|ValidateRequestText" ./internal/modules/prayers/
```

Every subtest says `PASS`, and **nothing** says `SKIP`.

---

## Step 7: Notifications

All of this is in **`apps/api/internal/modules/notifications/`**. Do it in order.

**7a. The two kinds** (`notifications.go`, near the other `Kind…` constants, around line 370):

```go
	// Your partner asked you to pray for something today.
	KindPrayerRequest = "prayer_request"
	// Your partner prayed for what you asked.
	KindPrayedForYou = "prayed_for_you"
```

Add both to the `AllKinds` slice a few lines below.

**7b. The kinds test** (`kinds_test.go`):
- Change `19` to `21` in `TestAllKindsIsComplete`.
- Add `KindPrayerRequest: true,` to the `perishable` map. A request is only worth anything on its
  day.
- Leave `KindPrayedForYou` out of both maps, which means "false" for both.

**7c. Perishable** (`budget.go`): add the request kind to the `case` in `Perishable`:

```go
	case KindChallenge, KindPrayerRequest:
```

**7d. The preference switch.** Copy how `PartnerChallenges` / `partner_challenges` is plumbed.
These are the exact places:

| File | Line (about) | Add |
|---|---|---|
| `notifications.go` | 48, the `Preferences` struct | `PrayerRequests bool` |
| `notifications.go` | 84, `Defaults()` | `PrayerRequests: true,` |
| `notifications.go` | 124, the patch struct | `PrayerRequests *bool` |
| `notifications.go` | 166, `Apply` | `setBool(&p.PrayerRequests, patch.PrayerRequests)` |
| `repository_postgres.go` | 33 and 42, the SELECT and Scan | `prayer_requests,` and `&p.PrayerRequests,` |
| `repository_postgres.go` | 63, 86 and 95, the upsert | the column, `prayer_requests = EXCLUDED.prayer_requests,`, and the value. **Count the `$n` placeholders carefully; this is where mistakes happen.** |
| `handler.go` | 62, 88 and 115 | the JSON field `prayer_requests` in the DTO, the mapping, and the patch |

**7e. The query** (`worker_repository.go`, inside `RecentlyWritten`, the big `WITH written AS (...)`).
Add two branches after the challenge branch, before the closing `)`:

```sql
			UNION ALL
			-- A partner asked for prayer today, the first time it is kept.
			-- created_at doesn't move on an edit, so an edit can't announce twice.
			SELECT pr.id, pr.couple_id, pr.author_id, pr.created_at, $9::text AS kind,
			       NULL::text AS subject, pr.asked_on, false
			FROM prayer_requests pr
			WHERE pr.created_at >= $1
			UNION ALL
			-- A partner prayed for what was asked. The one who prayed is the
			-- "author" here, so the outer query sends it to the one who asked.
			SELECT pr.id, pr.couple_id, pr.prayed_by, pr.prayed_at, $10::text AS kind,
			       NULL::text AS subject, pr.asked_on, false
			FROM prayer_requests pr
			WHERE pr.prayed_at >= $1 AND pr.prayed_by IS NOT NULL
```

Then, in the same query:
- In the outer `SELECT` list, after `COALESCE(p.partner_challenges, true),`, add
  `COALESCE(p.prayer_requests, true),`.
- After `KindLetter` in the argument list at the end of the query, add
  `KindPrayerRequest, KindPrayedForYou`. They become `$9` and `$10`.
- In `rows.Scan(...)`, after `&c.Prefs.PartnerChallenges,`, add `&c.Prefs.PrayerRequests,`. The
  order must match the SELECT exactly.
- In the loop under `if c.Kind == KindPrayerAnswered {...}`, add a short undo window, so that
  writing then quickly deleting sends nothing:

```go
		if c.Kind == KindPrayerRequest {
			c.Settles = requestUndoWindow
		}
```

  Next to `answeredUndoWindow`, define:

```go
const requestUndoWindow = time.Minute
```

**7f. The messages** (`worker.go`, inside `ForWritten`'s `switch`, before `default:`). Neither
message contains the request text, because a phone lock screen is not private.

```go
	case KindPrayerRequest:
		wanted = c.Prefs.PrayerRequests
		message = push.Message{
			Title: "🙏 " + c.AuthorName + " asked you to pray for them",
			Body:  "Open Amorae to see what for.",
			Path:  "/",
			Tag:   KindPrayerRequest,
		}
	case KindPrayedForYou:
		wanted = c.Prefs.PrayerRequests
		message = push.Message{
			Title: c.AuthorName + " prayed for you",
			Body:  "For what you asked this morning.",
			Path:  "/",
			Tag:   KindPrayedForYou,
		}
```

**7g. A test for the messages.** Add this to `worker_test.go`, next to `TestForWritten_Letter`
(around line 1224), and copy its shape:

```go
func TestForWritten_PrayerRequests(t *testing.T) {
	at := time.Date(2026, 11, 20, 7, 0, 0, 0, time.UTC)
	asked := WrittenCandidate{
		UserID: uuid.New(), AuthorID: uuid.New(), AuthorName: "Ada",
		ItemID: uuid.New(), Kind: KindPrayerRequest, WrittenAt: at,
		Settles: requestUndoWindow, Prefs: Preferences{PrayerRequests: true},
	}

	t.Run("not inside the undo window", func(t *testing.T) {
		if _, ok := ForWritten(asked, at.Add(10*time.Second)); ok {
			t.Error("sent before the undo window passed")
		}
	})
	t.Run("after it, without the words", func(t *testing.T) {
		n, ok := ForWritten(asked, at.Add(2*time.Minute))
		if !ok || n.Message.Title != "🙏 Ada asked you to pray for them" {
			t.Fatalf("ok %v, message %+v", ok, n.Message)
		}
	})
	t.Run("switched off, nothing", func(t *testing.T) {
		off := asked
		off.Prefs.PrayerRequests = false
		if _, ok := ForWritten(off, at.Add(2*time.Minute)); ok {
			t.Error("sent although the switch is off")
		}
	})
	t.Run("prayed for you", func(t *testing.T) {
		prayed := asked
		prayed.Kind, prayed.Settles = KindPrayedForYou, 0
		n, ok := ForWritten(prayed, at.Add(time.Minute))
		if !ok || n.Message.Title != "Ada prayed for you" {
			t.Fatalf("ok %v, message %+v", ok, n.Message)
		}
	})
}
```

**7h. The web mirror of the new kinds and the switch:**
- `apps/web/src/lib/api/types.ts`:
  - add `"prayer_request" | "prayed_for_you"` to the `NotificationKind` union (around line 355);
  - add `prayer_requests: boolean;` next to `partner_challenges` (line 336). Add it to the
    notification-preferences patch type too, if there's a separate one.
- `apps/web/src/features/notifications/derive.ts`: add both kinds to `ICON_BY_KIND`, using an icon
  name that already exists in that map (look at the other entries).
- `apps/web/src/app/(app)/settings/notifications/page.tsx`: next to the `partner_challenges` row
  (line 154), add:

```tsx
{row(p, "prayer_requests", `When ${partner} asks for prayer, or prays for you`)}
```

**Check:** from `apps/api`, these must all be clean or `ok`:

```bash
go vet ./...
```

```bash
go test ./internal/modules/notifications/... ./internal/modules/prayers/...
```

Then do it for real:
1. Femi writes a request in the console (Step 5).
2. Wait a minute.
3. Ada's bell, the in-app inbox, shows "🙏 Femi asked you to pray for them".

Push notifications need a subscribed device, so the inbox is the easy local check.

## Checkpoint 2

Send me the diff for Step 7 and both test outputs.

---

## Step 8: The web app

Everything here is in **`apps/web/src/`**.

**8a. The types** (`lib/api/types.ts`, near `PrayerWeek` at line 102):

```ts
export interface PrayerRequest {
  text: string;
  asked_on: string;
  /** When the other one prayed for it; null means not yet. */
  prayed_at: string | null;
}

export interface TodayRequests {
  mine: PrayerRequest | null;
  theirs: PrayerRequest | null;
}
```

**8b. The API calls** (`features/prayers/api.ts`, inside `prayersApi`):

```ts
  todayRequests: () =>
    http.get<TodayRequests>("/prayers/requests/today").then((r) => r.data),
  // PUT: asking again today is an edit, not a second request.
  ask: (text: string) =>
    http.put<TodayRequests>("/prayers/requests/today", { text }).then((r) => r.data),
  withdraw: () => http.delete<TodayRequests>("/prayers/requests/today").then((r) => r.data),
  prayedFor: (done: boolean) =>
    (done
      ? http.post<TodayRequests>("/prayers/requests/today/prayed")
      : http.delete<TodayRequests>("/prayers/requests/today/prayed")
    ).then((r) => r.data),
```

Add `TodayRequests` to the type import at the top.

**8c. The query key** (`lib/query/keys.ts`, after `answered`):

```ts
  requestsToday: ["prayers", "requests", "today"] as const,
```

**8d. The writes** (`features/prayers/writes.ts`, inside `prayerWrites`). All three are idempotent,
so they queue safely while offline:

```ts
  ask: defineWrite({
    mutationKey: ["prayers", "ask"],
    mutationFn: (text: string) => prayersApi.ask(text),
    invalidates: [keys.requestsToday],
    idempotent: "one request per person per day, so a replay rewrites the same line.",
  }),
  withdraw: defineWrite<void, TodayRequests>({
    mutationKey: ["prayers", "withdraw"],
    mutationFn: () => prayersApi.withdraw(),
    invalidates: [keys.requestsToday],
    idempotent: "deleting what is already gone changes nothing.",
  }),
  prayedFor: defineWrite({
    mutationKey: ["prayers", "prayedFor"],
    mutationFn: (done: boolean) => prayersApi.prayedFor(done),
    invalidates: [keys.requestsToday],
    // Same scope idea as completion: "prayed, then undo" can never land reversed.
    scope: "prayers.prayedFor",
    idempotent: "sets one flag on one row, so a replay lands the same state.",
  }),
```

Add `TodayRequests` to the type import at the top of `writes.ts`. These writes are already
registered for offline replay, because `components/providers.tsx` spreads every `prayerWrites`
entry.

**8e. The hooks** (`features/prayers/hooks.ts`). Copy `useWeek` and `useWriteLetter`:

```ts
export function useTodayRequests() {
  const couple = useCouple();
  return useQuery({
    queryKey: keys.requestsToday,
    queryFn: prayersApi.todayRequests,
    enabled: Boolean(couple.data?.partner),
  });
}
export const useAsk = () => useWrite(prayerWrites.ask);
export const useWithdraw = () => useWrite(prayerWrites.withdraw);
export const usePrayedFor = () => useWrite(prayerWrites.prayedFor);
```

Open `features/anniversary/hooks.ts` and look at how `useWriteLetter` is used on its page. That
shows you what `useWrite` returns, such as `mutate` and `isPending`. Use it the same way.

**8f. The Home card.**
- Create `features/home/prayer-request-card.tsx`.
- Render it in `features/home/home-screen.tsx` just above `{weekPrompt}` (around line 172), and
  only when not `alone`.
- It has three states. Build them from the shared components (`Section`, `Button` and the field
  components) so it matches the new design. The mockup is in `docs/design/mockups/index.html`,
  under "Pray for me today".

| State | Shows |
|---|---|
| Neither of you has asked | A section titled *Pray for me today*, a one-line input "What should {partner} pray for?", and a Send button |
| You asked | Your line in the serif, then "Waiting for {partner}", or "{partner} prayed for you · 7:42". Show the time from `prayed_at` with the same time formatter Home already uses. While it isn't prayed for yet, a quiet "Change" or "Withdraw" |
| They asked | Their line in the serif (the one raised panel on Home), and an "I prayed for this" button. Once tapped, "Prayed · Undo" |

Both "you asked" and "they asked" can be true at once. Show theirs first, because someone is
waiting on you.

**8g. Prayer mode.** In `features/prayers/components/prayer-session.tsx`, when `theirs` exists and
isn't prayed for yet, show it as the first page before the weekly points, with the same "I prayed
for this" button. Do this last; Home is enough to use the feature.

**Check:** from `apps/web`, all four commands must be clean:

```bash
npm run lint
```

```bash
npm run typecheck
```

```bash
npx vitest run
```

```bash
npm run build
```

Then use it with both windows open:
1. Femi asks.
2. Ada sees it on Home and taps "I prayed for this".
3. Femi's Home shows "Ada prayed for you · {time}".

Do this at phone width, in both light and dark.

## Checkpoint 3

Send me screenshots of the card in all three states, at phone width, in light and dark.

---

## Step 9: Ship it

Commit and push. Render runs migration 00038 on start, and Vercel redeploys the web app.

Then confirm it on your two real phones: ask, pray, and check the notification arrives.

---

## When something goes wrong

| What you see | Most likely cause |
|---|---|
| `relation "prayer_requests" does not exist` | The migration didn't run. Run `go run ./cmd/migrate up` |
| `TestAllKindsIsComplete` fails | You didn't bump 19 to 21, or forgot `AllKinds` |
| `TestEveryKindHasBeenDecidedAbout` fails | `perishable` in the test and `Perishable` in `budget.go` disagree |
| Scan error "number of field descriptions must equal number of destinations" | The SELECT and `rows.Scan` lists in `RecentlyWritten` or the prefs query are out of step |
| The notification never arrives | The undo window hasn't passed (1 minute), the switch is off in Settings, or quiet hours are on. The worker also runs on a tick; `Poke()` should make it prompt |
| Every test says `SKIP` | `AMORAE_TEST_DATABASE_URL` isn't set in **this** terminal |
| 409 `waiting_for_partner` | That user isn't paired yet |
