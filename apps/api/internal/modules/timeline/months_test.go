package timeline_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/timeline"
	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/database"
	"github.com/falola13/amorae/apps/api/internal/platform/database/dbtest"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
	"github.com/falola13/amorae/apps/api/internal/platform/metrics"
)

// seedMonths writes notes, a journal entry and a memory across three months at
// fixed dates: Sep has a memory and a note, Oct a note and a journal entry
// (plus a second note), Nov a note.
func seedMonths(t *testing.T, db *database.DB, coupleID, a uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	at := func(s string) time.Time {
		v, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	note := func(when string) {
		ts := at(when)
		if _, err := db.Q(ctx).Exec(ctx, `
			INSERT INTO appreciations (id, couple_id, from_id, date, text, created_at)
			VALUES ($1, $2, $3, $4, 'Thank you.', $5)
		`, uuid.New(), coupleID, a, ts, ts); err != nil {
			t.Fatalf("note: %v", err)
		}
	}
	note("2026-09-10T10:00:00Z")
	note("2026-10-05T10:00:00Z")
	note("2026-10-20T10:00:00Z")
	note("2026-11-02T10:00:00Z")
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO journal_entries (id, couple_id, author_id, date, tag, text, created_at)
		VALUES ($1, $2, $3, '2026-10-06', 'Gratitude', 'For today.', $4)
	`, uuid.New(), coupleID, a, at("2026-10-06T10:00:00Z")); err != nil {
		t.Fatalf("journal: %v", err)
	}
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO memories (id, couple_id, title, date, location, created_at, updated_at)
		VALUES ($1, $2, 'Sept day', '2026-09-15', 'Lagos', $3, $3)
	`, uuid.New(), coupleID, at("2026-09-16T10:00:00Z")); err != nil {
		t.Fatalf("memory: %v", err)
	}
}

func TestTimeline_Months(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo := timeline.NewPostgresRepository(db)

	coupleID, a, _ := pair(t, db)
	seedMonths(t, db, coupleID, a)

	cases := []struct {
		filter timeline.Filter
		want   []timeline.MonthCount
	}{
		{timeline.FilterAll, []timeline.MonthCount{{"2026-11", 1}, {"2026-10", 3}, {"2026-09", 2}}},
		{timeline.FilterNotes, []timeline.MonthCount{{"2026-11", 1}, {"2026-10", 2}, {"2026-09", 1}}},
		{timeline.FilterJournal, []timeline.MonthCount{{"2026-10", 1}}},
		{timeline.FilterMemories, []timeline.MonthCount{{"2026-09", 1}}},
		{timeline.FilterQuestions, []timeline.MonthCount{}},
	}
	for _, tc := range cases {
		t.Run(string(tc.filter), func(t *testing.T) {
			got, err := repo.Months(ctx, coupleID, timeline.TypesFor(tc.filter))
			if err != nil {
				t.Fatalf("Months: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}

	t.Run("another couple sees none of it", func(t *testing.T) {
		other, _, _ := pair(t, db)
		got, err := repo.Months(ctx, other, timeline.TypesFor(timeline.FilterAll))
		if err != nil {
			t.Fatalf("Months: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("got %v, want none", got)
		}
	})
}

func TestTimeline_MonthEndUsesCoupleTimezone(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo := timeline.NewPostgresRepository(db)

	coupleID, _, _ := pair(t, db)
	if _, err := db.Q(ctx).Exec(ctx, `UPDATE couples SET timezone = 'America/New_York' WHERE id = $1`, coupleID); err != nil {
		t.Fatal(err)
	}
	got, err := repo.MonthEnd(ctx, coupleID, "2026-10")
	if err != nil {
		t.Fatalf("MonthEnd: %v", err)
	}
	// Midnight 1 Nov in New York (still on EDT) is 04:00 UTC.
	if want := time.Date(2026, 11, 1, 4, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

type monthsAPI struct {
	t   *testing.T
	mux *http.ServeMux
}

func newMonthsAPI(t *testing.T, db *database.DB, coupleID uuid.UUID) monthsAPI {
	t.Helper()
	svc := timeline.NewService(timeline.NewPostgresRepository(db), fixedCouple{coupleID}, nil, time.Now)
	mux := http.NewServeMux()
	asCaller := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(authctx.WithUserID(r.Context(), uuid.New())))
		})
	}
	timeline.NewHandler(svc).RegisterRoutes(httpx.NewRouter(mux, asCaller, metrics.New()))
	return monthsAPI{t: t, mux: mux}
}

func (a monthsAPI) get(path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestTimelineHTTP_MonthParam(t *testing.T) {
	db := dbtest.New(t)
	coupleID, a, _ := pair(t, db)
	seedMonths(t, db, coupleID, a)
	api := newMonthsAPI(t, db, coupleID)

	type page struct {
		Data struct {
			Items []struct {
				At   string `json:"at"`
				Date string `json:"date"`
			} `json:"items"`
			Next *string `json:"next"`
		} `json:"data"`
	}
	load := func(path string) page {
		rec := api.get(path)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d: %s", path, rec.Code, rec.Body)
		}
		var p page
		if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return p
	}

	// October starts the page at its own end: nothing from November, and the
	// normal cursor then pages onward into September.
	first := load("/timeline?month=2026-10&limit=2")
	if len(first.Data.Items) != 2 || first.Data.Next == nil {
		t.Fatalf("first page = %+v", first.Data)
	}
	for _, it := range first.Data.Items {
		if it.Date > "2026-10-31" {
			t.Errorf("item dated %s is past the end of October", it.Date)
		}
	}
	if first.Data.Items[0].Date != "2026-10-20" {
		t.Errorf("newest item = %s, want 2026-10-20", first.Data.Items[0].Date)
	}

	rest := load("/timeline?month=2026-10&limit=10&before=" + *first.Data.Next)
	total := len(first.Data.Items) + len(rest.Data.Items)
	if total != 5 { // 3 in October + 2 in September, none from November
		t.Errorf("paged %d items, want 5", total)
	}
	if rest.Data.Next != nil {
		t.Errorf("expected the end of the story, got next = %s", *rest.Data.Next)
	}

	// An explicit cursor beats the month.
	cur := load("/timeline?month=2026-10&before=2026-09-12T00:00:00Z")
	if len(cur.Data.Items) != 1 {
		t.Errorf("cursor before 12 Sep gave %d items, want 1", len(cur.Data.Items))
	}

	for _, bad := range []string{"2026-13", "October", "2026-1", "2026-10-01", "26-10"} {
		if rec := api.get("/timeline?month=" + bad); rec.Code != http.StatusBadRequest {
			t.Errorf("month=%s gave %d, want 400", bad, rec.Code)
		}
	}
}

func TestTimelineHTTP_MonthsEndpoint(t *testing.T) {
	db := dbtest.New(t)
	coupleID, a, _ := pair(t, db)
	seedMonths(t, db, coupleID, a)
	api := newMonthsAPI(t, db, coupleID)

	rec := api.get("/timeline/months?filter=notes")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	var body struct {
		Data []struct {
			Month string `json:"month"`
			Count int    `json:"count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data) != 3 || body.Data[0].Month != "2026-11" || body.Data[1].Month != "2026-10" || body.Data[1].Count != 2 {
		t.Errorf("months = %+v", body.Data)
	}
}
