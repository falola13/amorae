package questions_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/questions"
	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/database"
	"github.com/falola13/amorae/apps/api/internal/platform/database/dbtest"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
	"github.com/falola13/amorae/apps/api/internal/platform/metrics"
)

// couplesOf maps each person to their couple, so one service can serve two
// couples and a test can ask whether one can read the other.
type couplesOf map[uuid.UUID]uuid.UUID

func (c couplesOf) CoupleFor(_ context.Context, userID uuid.UUID) (uuid.UUID, error) {
	return c[userID], nil
}

type countingPoker struct{ pokes int }

func (p *countingPoker) Poke() { p.pokes++ }

func pair(t *testing.T, db *database.DB) (coupleID, a, b uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	a, b = uuid.New(), uuid.New()
	for id, name := range map[uuid.UUID]string{a: "Ada", b: "Bo"} {
		if _, err := db.Q(ctx).Exec(ctx, `
			INSERT INTO users (id, email, display_name, password_hash, created_at, updated_at)
			VALUES ($1, $2, $3, 'hash', $4, $4)
		`, id, "test+"+uuid.NewString()+"@example.com", name, now); err != nil {
			t.Fatalf("insert user: %v", err)
		}
	}
	coupleID = uuid.New()
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO couples (id, name, timezone, created_by, created_at, updated_at)
		VALUES ($1, 'Ada & Bo', 'Africa/Lagos', $2, $3, $3)
	`, coupleID, a, now); err != nil {
		t.Fatalf("insert couple: %v", err)
	}
	for _, id := range []uuid.UUID{a, b} {
		if _, err := db.Q(ctx).Exec(ctx, `
			INSERT INTO couple_members (id, couple_id, user_id, joined_at)
			VALUES ($1, $2, $3, $4)
		`, uuid.New(), coupleID, id, now); err != nil {
			t.Fatalf("insert member: %v", err)
		}
	}
	return coupleID, a, b
}

// api drives the real handler over the real service, so a test sees the raw
// body a client would be sent, not a struct that happens to leave things out.
type api struct {
	t   *testing.T
	mux *http.ServeMux
}

func newAPI(t *testing.T, svc *questions.Service) api {
	t.Helper()
	mux := http.NewServeMux()
	asCaller := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := uuid.Parse(r.Header.Get("X-Test-User"))
			if err != nil {
				http.Error(w, "no user", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(authctx.WithUserID(r.Context(), id)))
		})
	}
	questions.NewHandler(svc).RegisterRoutes(httpx.NewRouter(mux, asCaller, metrics.New()))
	return api{t: t, mux: mux}
}

type answerJSON struct {
	Text string `json:"text"`
}

type entryJSON struct {
	AskedOn        string      `json:"asked_on"`
	Question       string      `json:"question"`
	Mine           *answerJSON `json:"mine"`
	Theirs         *answerJSON `json:"theirs"`
	TheirsAnswered bool        `json:"theirs_answered"`
	Previous       *struct {
		AskedOn string      `json:"asked_on"`
		Mine    *answerJSON `json:"mine"`
		Theirs  *answerJSON `json:"theirs"`
	} `json:"previous"`
}

func (a api) do(user uuid.UUID, method, target, body string) (int, string) {
	a.t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("X-Test-User", user.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func (a api) today(user uuid.UUID) (entryJSON, string) {
	a.t.Helper()
	code, raw := a.do(user, http.MethodGet, "/questions/today", "")
	if code != http.StatusOK {
		a.t.Fatalf("GET today = %d: %s", code, raw)
	}
	var env struct{ Data entryJSON }
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		a.t.Fatalf("decode: %v: %s", err, raw)
	}
	return env.Data, raw
}

func (a api) answer(user uuid.UUID, text string) {
	a.t.Helper()
	body, _ := json.Marshal(map[string]string{"text": text})
	if code, raw := a.do(user, http.MethodPut, "/questions/today", string(body)); code != http.StatusOK {
		a.t.Fatalf("PUT today = %d: %s", code, raw)
	}
}

func (a api) past(user uuid.UUID, query string) (items []entryJSON, next *string, raw string) {
	a.t.Helper()
	code, raw := a.do(user, http.MethodGet, "/questions"+query, "")
	if code != http.StatusOK {
		a.t.Fatalf("GET past = %d: %s", code, raw)
	}
	var env struct {
		Data struct {
			Items []entryJSON `json:"items"`
			Next  *string     `json:"next"`
		}
	}
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		a.t.Fatalf("decode: %v: %s", err, raw)
	}
	return env.Data.Items, env.Data.Next, raw
}

func TestDailyQuestion_Seal(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	coupleID, ada, bo := pair(t, db)
	otherID, carol, _ := pair(t, db)

	clock := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	poker := &countingPoker{}
	svc := questions.NewService(questions.NewPostgresRepository(db),
		couplesOf{ada: coupleID, bo: coupleID, carol: otherID}, func() time.Time { return clock }, poker)
	app := newAPI(t, svc)

	const adaText = "You remembered my mother's birthday without being told."
	const boText = "The way you hum when you cook."

	var question string
	t.Run("before anyone answers", func(t *testing.T) {
		e, _ := app.today(ada)
		if e.Question == "" || e.Mine != nil || e.Theirs != nil || e.TheirsAnswered {
			t.Fatalf("entry = %+v", e)
		}
		question = e.Question
	})

	t.Run("their answer is sealed, in the data and in the raw body", func(t *testing.T) {
		app.answer(ada, adaText)
		if poker.pokes == 0 {
			t.Error("answering did not poke the notifications worker")
		}
		e, raw := app.today(bo)
		if e.Mine != nil || e.Theirs != nil || !e.TheirsAnswered {
			t.Fatalf("Bo before answering = %+v", e)
		}
		if strings.Contains(raw, adaText) || strings.Contains(raw, "mother") {
			t.Fatalf("sealed answer leaked into the body: %s", raw)
		}
		if e.Question != question {
			t.Errorf("Bo was asked %q, Ada %q", e.Question, question)
		}
	})

	t.Run("answering unseals it, for both, with the same question", func(t *testing.T) {
		app.answer(bo, boText)
		eBo, raw := app.today(bo)
		if eBo.Theirs == nil || eBo.Theirs.Text != adaText || eBo.Mine == nil || eBo.Mine.Text != boText {
			t.Fatalf("Bo after answering = %+v (%s)", eBo, raw)
		}
		eAda, _ := app.today(ada)
		if eAda.Theirs == nil || eAda.Theirs.Text != boText || eAda.Question != eBo.Question {
			t.Fatalf("Ada = %+v", eAda)
		}
	})

	t.Run("an answer can be rewritten the same day", func(t *testing.T) {
		app.answer(ada, "Edited.")
		e, _ := app.today(bo)
		if e.Theirs == nil || e.Theirs.Text != "Edited." {
			t.Errorf("Bo sees %+v", e.Theirs)
		}
		app.answer(ada, adaText)
	})

	t.Run("another couple can't read it", func(t *testing.T) {
		e, raw := app.today(carol)
		if e.Mine != nil || e.Theirs != nil || e.TheirsAnswered {
			t.Fatalf("carol = %+v", e)
		}
		if strings.Contains(raw, adaText) || strings.Contains(raw, boText) {
			t.Fatalf("another couple's answer leaked: %s", raw)
		}
		items, _, _ := app.past(carol, "")
		if len(items) != 0 {
			t.Errorf("carol's past = %+v, want none", items)
		}
	})

	t.Run("the next day is a new question and an empty slate", func(t *testing.T) {
		clock = clock.Add(24 * time.Hour)
		e, _ := app.today(ada)
		if e.Question == question || e.Mine != nil || e.Theirs != nil || e.TheirsAnswered {
			t.Fatalf("next day = %+v (yesterday %q)", e, question)
		}
		items, _, _ := app.past(ada, "")
		if len(items) != 1 || items[0].Mine == nil || items[0].Theirs == nil || items[0].Question != question {
			t.Fatalf("past = %+v", items)
		}
	})

	t.Run("a day that has passed can't be answered", func(t *testing.T) {
		// Answering now writes to the new day; yesterday's rows stay as they were.
		app.answer(ada, "Today only.")
		var rows int
		var text string
		if err := db.Q(ctx).QueryRow(ctx, `
			SELECT count(*), min(text) FILTER (WHERE author_id = $2)
			FROM daily_answers WHERE couple_id = $1 AND asked_on = '2026-11-20'
		`, coupleID, ada).Scan(&rows, &text); err != nil {
			t.Fatal(err)
		}
		if rows != 2 || text != adaText {
			t.Errorf("yesterday now has %d rows, Ada's text %q", rows, text)
		}
	})

	t.Run("a day you skipped stays sealed, however long ago", func(t *testing.T) {
		clock = clock.Add(24 * time.Hour) // Ada answered the 21st; Bo did not
		items, _, raw := app.past(bo, "")
		if len(items) != 2 {
			t.Fatalf("past = %+v", items)
		}
		skipped := items[0]
		if skipped.AskedOn != "2026-11-21" || skipped.Mine != nil || skipped.Theirs != nil || !skipped.TheirsAnswered {
			t.Errorf("skipped day = %+v", skipped)
		}
		if strings.Contains(raw, "Today only.") {
			t.Fatalf("sealed answer leaked into past: %s", raw)
		}
		// Bo can no longer answer the 21st, so nothing opens it later.
		app.answer(bo, "Back on the 22nd.")
		items, _, raw = app.past(bo, "")
		if strings.Contains(raw, "Today only.") {
			t.Fatalf("answering a later day opened an earlier one: %s", raw)
		}
	})
}

func TestDailyQuestion_Past_Paging(t *testing.T) {
	db := dbtest.New(t)
	coupleID, ada, bo := pair(t, db)
	clock := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	svc := questions.NewService(questions.NewPostgresRepository(db),
		couplesOf{ada: coupleID, bo: coupleID}, func() time.Time { return clock }, nil)
	app := newAPI(t, svc)

	for i := 0; i < 5; i++ {
		app.answer(ada, "a")
		app.answer(bo, "b")
		clock = clock.Add(24 * time.Hour)
	}

	first, next, _ := app.past(ada, "?limit=2")
	if len(first) != 2 || next == nil || first[0].AskedOn != "2026-03-05" || first[1].AskedOn != "2026-03-04" {
		t.Fatalf("first page = %+v, next %v", first, next)
	}
	second, next, _ := app.past(ada, "?limit=2&before="+*next)
	if len(second) != 2 || next == nil || second[0].AskedOn != "2026-03-03" {
		t.Fatalf("second page = %+v, next %v", second, next)
	}
	last, next, _ := app.past(ada, "?limit=2&before="+*next)
	if len(last) != 1 || next != nil || last[0].AskedOn != "2026-03-01" {
		t.Fatalf("last page = %+v, next %v", last, next)
	}

	t.Run("bad paging is a friendly error", func(t *testing.T) {
		for _, q := range []string{"?before=yesterday", "?limit=lots"} {
			if code, _ := app.do(ada, http.MethodGet, "/questions"+q, ""); code != http.StatusBadRequest {
				t.Errorf("%s = %d, want 400", q, code)
			}
		}
	})
	t.Run("limit is held to 50", func(t *testing.T) {
		if code, _ := app.do(ada, http.MethodGet, "/questions?limit=500", ""); code != http.StatusOK {
			t.Errorf("limit=500 = %d, want it clamped, not refused", code)
		}
	})
}

func TestValidateAnswer(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		ok   bool
	}{
		{"empty", "", false},
		{"only spaces", "  \n ", false},
		{"one word", "Peace", true},
		{"exactly 1000", strings.Repeat("a", 1000), true},
		{"1001", strings.Repeat("a", 1001), false},
		{"1000 emoji count as 1000", strings.Repeat("🙏", 1000), true},
		{"1001 emoji", strings.Repeat("🙏", 1001), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := questions.ValidateAnswer(tc.in)
			if (err == nil) != tc.ok {
				t.Fatalf("err = %v, want ok=%v", err, tc.ok)
			}
			if err != nil {
				var ae *apperr.Error
				if !asAppErr(err, &ae) {
					t.Errorf("error %v is not a friendly apperr", err)
				}
			} else if got != strings.TrimSpace(tc.in) {
				t.Errorf("not trimmed: %q", got)
			}
		})
	}
}

func asAppErr(err error, target **apperr.Error) bool {
	ae, ok := err.(*apperr.Error)
	if ok {
		*target = ae
	}
	return ok
}

func TestPick(t *testing.T) {
	couple := uuid.MustParse("0192f0c0-0000-7000-8000-0000000000aa")
	other := uuid.MustParse("0192f0c0-0000-7000-8000-0000000000bb")
	bank := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	day := func(n int) time.Time { return time.Date(2026, 3, n, 0, 0, 0, 0, time.UTC) }

	t.Run("deterministic, and the same for both partners", func(t *testing.T) {
		if questions.Pick(bank, couple, nil) != questions.Pick(bank, couple, nil) {
			t.Fatal("same inputs, different question")
		}
	})

	t.Run("couples walk the bank in their own order", func(t *testing.T) {
		a, b := map[string]time.Time{}, map[string]time.Time{}
		var orderA, orderB []string
		for range bank {
			qa, qb := questions.Pick(bank, couple, a), questions.Pick(bank, other, b)
			orderA, orderB = append(orderA, qa), append(orderB, qb)
			a[qa], b[qb] = day(1), day(1)
		}
		if strings.Join(orderA, "") == strings.Join(orderB, "") {
			t.Errorf("two couples share an order: %v", orderA)
		}
	})

	t.Run("unused questions come first, each exactly once", func(t *testing.T) {
		used := map[string]time.Time{}
		seen := map[string]bool{}
		for i := range bank {
			q := questions.Pick(bank, couple, used)
			if seen[q] {
				t.Fatalf("%q came round again after %d of %d", q, i, len(bank))
			}
			seen[q] = true
			used[q] = day(i + 1)
		}
	})

	t.Run("a question that was not answered is still unused", func(t *testing.T) {
		first := questions.Pick(bank, couple, nil)
		// Nothing was answered, so nothing is recorded, and tomorrow offers it again.
		if got := questions.Pick(bank, couple, map[string]time.Time{}); got != first {
			t.Errorf("got %q, want %q again", got, first)
		}
	})

	t.Run("once all are used, the longest-ago comes round first", func(t *testing.T) {
		used := map[string]time.Time{}
		for i, q := range bank {
			used[q] = day(i + 1)
		}
		// Make "e" the oldest.
		used["e"] = day(1).AddDate(-1, 0, 0)
		if got := questions.Pick(bank, couple, used); got != "e" {
			t.Errorf("got %q, want e", got)
		}
	})

	t.Run("questions dropped from the bank are ignored", func(t *testing.T) {
		used := map[string]time.Time{"gone": day(1)}
		if got := questions.Pick(bank, couple, used); got == "" || got == "gone" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("an empty bank asks nothing", func(t *testing.T) {
		if got := questions.Pick(nil, couple, nil); got != "" {
			t.Errorf("got %q", got)
		}
	})
}

func TestDailyQuestion_SkippedDayDoesNotUseUpAQuestion(t *testing.T) {
	db := dbtest.New(t)
	coupleID, ada, bo := pair(t, db)
	clock := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	svc := questions.NewService(questions.NewPostgresRepository(db),
		couplesOf{ada: coupleID, bo: coupleID}, func() time.Time { return clock }, nil)
	svc.SetBank([]string{"one?", "two?", "three?", "four?", "five?"})
	app := newAPI(t, svc)

	first, _ := app.today(ada)
	// Nobody answers; the day passes and the same question is offered again.
	clock = clock.Add(24 * time.Hour)
	again, _ := app.today(ada)
	if again.Question != first.Question {
		t.Fatalf("an unanswered day used up %q; today is %q", first.Question, again.Question)
	}
	clock = clock.Add(24 * time.Hour)
	if third, _ := app.today(bo); third.Question != first.Question {
		t.Fatalf("two skipped days used up %q; today is %q", first.Question, third.Question)
	}

	t.Run("both partners see the same one, even after one answers", func(t *testing.T) {
		app.answer(bo, "Mine.")
		a, _ := app.today(ada)
		b, _ := app.today(bo)
		if a.Question != b.Question || a.Question != first.Question {
			t.Errorf("Ada %q, Bo %q, want %q", a.Question, b.Question, first.Question)
		}
	})

	t.Run("an answer from one of you uses it up", func(t *testing.T) {
		clock = clock.Add(24 * time.Hour)
		next, _ := app.today(ada)
		if next.Question == first.Question || next.Previous != nil {
			t.Errorf("next = %+v, want a fresh question with no previous", next)
		}
	})
}

func TestDailyQuestion_Previous(t *testing.T) {
	db := dbtest.New(t)
	coupleID, ada, bo := pair(t, db)
	clock := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	svc := questions.NewService(questions.NewPostgresRepository(db),
		couplesOf{ada: coupleID, bo: coupleID}, func() time.Time { return clock }, nil)
	svc.SetBank([]string{"one?", "two?"})
	app := newAPI(t, svc)
	nextDay := func() { clock = clock.Add(24 * time.Hour) }

	// Day 1 and 2: both answer both questions in the bank.
	q1, _ := app.today(ada)
	app.answer(ada, "Ada first time")
	app.answer(bo, "Bo first time")
	nextDay()
	q2, _ := app.today(ada)
	if q2.Question == q1.Question {
		t.Fatalf("day 2 repeated %q", q1.Question)
	}
	app.answer(ada, "Ada second")
	nextDay()

	t.Run("a repeat shows the last time, once you have answered", func(t *testing.T) {
		// Day 3: the bank is used up; the one asked longest ago is q1.
		e, _ := app.today(bo)
		if e.Question != q1.Question {
			t.Fatalf("day 3 = %q, want %q (longest ago)", e.Question, q1.Question)
		}
		if e.Previous == nil || e.Previous.AskedOn != "2026-05-01" ||
			e.Previous.Mine == nil || e.Previous.Mine.Text != "Bo first time" ||
			e.Previous.Theirs == nil || e.Previous.Theirs.Text != "Ada first time" {
			t.Fatalf("previous = %+v", e.Previous)
		}
	})

	t.Run("a repeat that only one of you answered last time has no previous", func(t *testing.T) {
		app.answer(bo, "Bo third")
		nextDay()
		e, raw := app.today(ada)
		if e.Question != q2.Question {
			t.Fatalf("day 4 = %q, want %q", e.Question, q2.Question)
		}
		if e.Previous != nil || strings.Contains(raw, "Ada second") {
			t.Fatalf("previous = %+v in %s", e.Previous, raw)
		}
	})
}
