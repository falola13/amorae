package challenges_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/challenges"
	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/database/dbtest"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
	"github.com/falola13/amorae/apps/api/internal/platform/metrics"
)

func tpl(key string) challenges.StartInput { return challenges.StartInput{Template: key} }

func TestSeveral_UpToThreeAtOnce(t *testing.T) {
	ctx := context.Background()
	svc, clk, _, a, _ := newService(t, "UTC", time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))

	var first challenges.Viewer
	for i, in := range []challenges.StartInput{tpl("seven-days-of-noticing"), tpl("ten-conversations"), custom()} {
		clk.advance(time.Minute)
		v, err := svc.Start(ctx, a, in)
		if err != nil {
			t.Fatalf("start %d: %v", i+1, err)
		}
		if i == 0 {
			first = v
		}
	}

	clk.advance(time.Minute)
	if _, err := svc.Start(ctx, a, tpl("seven-days-of-gratitude")); code(t, err) != "too_many_challenges" {
		t.Fatalf("a fourth = %v, want too_many_challenges", err)
	}

	// Finishing or leaving one makes room.
	if err := svc.Leave(ctx, a, first.ID); err != nil {
		t.Fatalf("Leave: %v", err)
	}
	if _, err := svc.Start(ctx, a, tpl("seven-days-of-gratitude")); err != nil {
		t.Errorf("a fourth after leaving one: %v", err)
	}
}

func TestSeveral_TheSameOneTwiceIsRefusedButCustomIsNot(t *testing.T) {
	ctx := context.Background()
	svc, clk, _, a, _ := newService(t, "UTC", time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))

	v, err := svc.Start(ctx, a, tpl("ten-conversations"))
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if _, err := svc.Start(ctx, a, tpl("ten-conversations")); code(t, err) != "challenge_already_running" {
		t.Errorf("the same template twice = %v, want challenge_already_running", err)
	}

	// Two of their own, which are two different things.
	clk.advance(time.Minute)
	if _, err := svc.Start(ctx, a, custom()); err != nil {
		t.Fatalf("first custom: %v", err)
	}
	clk.advance(time.Minute)
	if _, err := svc.Start(ctx, a, custom()); err != nil {
		t.Fatalf("second custom: %v", err)
	}

	// Once the first has ended, the same template can run again.
	if err := svc.Leave(ctx, a, v.ID); err != nil {
		t.Fatalf("Leave: %v", err)
	}
	if _, err := svc.Start(ctx, a, tpl("ten-conversations")); err != nil {
		t.Errorf("the same template after it ended: %v", err)
	}
}

// Seven distinct starts at once, only three of which can fit: the cap holds
// however the transactions interleave.
func TestSeveral_ConcurrentStartsCannotExceedTheCap(t *testing.T) {
	ctx := context.Background()
	svc, _, _, a, _ := newService(t, "UTC", time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))

	keys := []string{
		"seven-days-of-noticing", "seven-days-of-praying-together", "fourteen-days-of-small-things",
		"seven-days-of-gratitude", "seven-days-of-serving-each-other", "fourteen-days-in-the-psalms",
		"ten-conversations",
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	started, refused := 0, 0
	for _, k := range keys {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.Start(ctx, a, tpl(k))
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				started++
			case code(t, err) == "too_many_challenges":
				refused++
			default:
				t.Errorf("start %s: %v", k, err)
			}
		}()
	}
	wg.Wait()
	if started != challenges.MaxActive || refused != len(keys)-challenges.MaxActive {
		t.Fatalf("started = %d, refused = %d; want %d and %d", started, refused, challenges.MaxActive, len(keys)-challenges.MaxActive)
	}
	active, err := svc.Active(ctx, a)
	if err != nil || len(active) != challenges.MaxActive {
		t.Errorf("active = %d, err = %v; want %d", len(active), err, challenges.MaxActive)
	}
}

func TestSeveral_ActiveIsOldestStartedFirst(t *testing.T) {
	ctx := context.Background()
	svc, clk, _, a, _ := newService(t, "UTC", time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))

	if got, err := svc.Active(ctx, a); err != nil || len(got) != 0 {
		t.Fatalf("active with none = %v, %v; want empty", got, err)
	}

	var ids []uuid.UUID
	for _, in := range []challenges.StartInput{tpl("ten-conversations"), tpl("seven-days-of-noticing"), custom()} {
		clk.advance(time.Minute)
		v, err := svc.Start(ctx, a, in)
		if err != nil {
			t.Fatalf("Start: %v", err)
		}
		ids = append(ids, v.ID)
	}
	got, err := svc.Active(ctx, a)
	if err != nil || len(got) != 3 {
		t.Fatalf("Active = %d, %v", len(got), err)
	}
	for i := range ids {
		if got[i].ID != ids[i] {
			t.Errorf("position %d = %s, want %s", i, got[i].ID, ids[i])
		}
	}

	// Current, for older clients, is the newest of them; and a finished or
	// left one drops out of the active list.
	if cur, err := svc.Current(ctx, a); err != nil || cur.ID != ids[2] {
		t.Errorf("Current = %v, %v; want the most recently started", cur.ID, err)
	}
	if err := svc.Leave(ctx, a, ids[1]); err != nil {
		t.Fatalf("Leave: %v", err)
	}
	if got, _ := svc.Active(ctx, a); len(got) != 2 || got[0].ID != ids[0] || got[1].ID != ids[2] {
		t.Errorf("Active after leaving the middle one = %+v", got)
	}
}

func TestSeveral_MarkingOneLeavesTheOthersAlone(t *testing.T) {
	ctx := context.Background()
	svc, _, _, a, b := newService(t, "UTC", time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	one, _ := svc.Start(ctx, a, tpl("ten-conversations"))
	two, err := svc.Start(ctx, a, custom())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	v, err := svc.Mark(ctx, a, one.ID, 1, &yes, nil, nil)
	if err != nil || v.ID != one.ID || v.Days[0].Marks[a] != challenges.MarkDone {
		t.Fatalf("Mark one = %+v, %v", v.Challenge, err)
	}
	other, err := svc.Get(ctx, b, two.ID)
	if err != nil || len(other.Days[0].Marks) != 0 {
		t.Errorf("the other challenge = %+v, %v; want untouched", other.Days[0], err)
	}
	if _, err := svc.Mark(ctx, a, uuid.New(), 1, &yes, nil, nil); code(t, err) != "challenge_not_found" {
		t.Errorf("marking a stranger's = %v, want challenge_not_found", err)
	}
	if err := svc.Leave(ctx, a, uuid.New()); code(t, err) != "challenge_not_found" {
		t.Errorf("leaving a stranger's = %v, want challenge_not_found", err)
	}
}

type fakePartners struct{ a, b uuid.UUID }

func (f fakePartners) PartnerOf(_ context.Context, u uuid.UUID) (uuid.UUID, error) {
	if u == f.a {
		return f.b, nil
	}
	return f.a, nil
}

// The routes over HTTP: the id-based ones, and the older /current ones acting
// on the most recently started challenge.
func TestSeveral_Routes(t *testing.T) {
	db := dbtest.New(t)
	coupleID, a, b := pair(t, db, "UTC")
	clk := &clock{t: time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)}
	svc := challenges.NewService(challenges.NewPostgresRepository(db), fixedCouple{coupleID}, clk.now, noPoke{})

	mux := http.NewServeMux()
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(authctx.WithUserID(r.Context(), a)))
		})
	}
	challenges.NewHandler(svc, fakePartners{a, b}).RegisterRoutes(httpx.NewRouter(mux, auth, metrics.New()))

	call := func(method, path, body string) (int, map[string]any) {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		var out map[string]any
		if w.Body.Len() > 0 {
			if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
				t.Fatalf("%s %s: bad json %q", method, path, w.Body.String())
			}
		}
		return w.Code, out
	}
	activeIDs := func() []string {
		t.Helper()
		status, out := call("GET", "/challenges/active", "")
		if status != 200 {
			t.Fatalf("GET /challenges/active = %d", status)
		}
		var ids []string
		for _, c := range out["data"].([]any) {
			ids = append(ids, c.(map[string]any)["id"].(string))
		}
		return ids
	}
	done := func(out map[string]any, day int) bool {
		return out["data"].(map[string]any)["days"].([]any)[day-1].(map[string]any)["done"].(bool)
	}

	if ids := activeIDs(); len(ids) != 0 {
		t.Fatalf("active with none = %v", ids)
	}
	if status, out := call("GET", "/challenges/current", ""); status != 404 || out["error"].(map[string]any)["code"] != "challenge_not_found" {
		t.Fatalf("current with none = %d %v", status, out)
	}

	status, first := call("POST", "/challenges", `{"template":"ten-conversations"}`)
	if status != 201 {
		t.Fatalf("POST = %d %v", status, first)
	}
	clk.advance(time.Minute)
	status, second := call("POST", "/challenges", `{"custom":{"title":"Ours","prompts":["a","b","c"]}}`)
	if status != 201 {
		t.Fatalf("second POST = %d %v", status, second)
	}
	firstID := first["data"].(map[string]any)["id"].(string)
	secondID := second["data"].(map[string]any)["id"].(string)

	if ids := activeIDs(); len(ids) != 2 || ids[0] != firstID || ids[1] != secondID {
		t.Fatalf("active = %v, want [%s %s]", ids, firstID, secondID)
	}
	if _, out := call("GET", "/challenges/current", ""); out["data"].(map[string]any)["id"] != secondID {
		t.Errorf("current = %v, want the most recently started", out["data"].(map[string]any)["id"])
	}

	// The alias marks the newest; the id route marks the one named.
	status, out := call("PATCH", "/challenges/current/days/1", `{"done":true}`)
	if status != 200 || out["data"].(map[string]any)["id"] != secondID || !done(out, 1) {
		t.Fatalf("PATCH current = %d %v", status, out)
	}
	if _, out := call("GET", "/challenges/"+firstID, ""); done(out, 1) {
		t.Error("marking current touched the older challenge")
	}
	status, out = call("PATCH", "/challenges/"+firstID+"/days/1", `{"done":true}`)
	if status != 200 || out["data"].(map[string]any)["id"] != firstID || !done(out, 1) {
		t.Fatalf("PATCH by id = %d %v", status, out)
	}
	if status, _ := call("PATCH", "/challenges/"+uuid.NewString()+"/days/1", `{"done":true}`); status != 404 {
		t.Errorf("PATCH on a stranger's = %d, want 404", status)
	}
	if status, _ := call("PATCH", "/challenges/not-an-id/days/1", `{"done":true}`); status != 404 {
		t.Errorf("PATCH on a malformed id = %d, want 404", status)
	}

	// The cap and the duplicate, over the wire.
	clk.advance(time.Minute)
	call("POST", "/challenges", `{"template":"seven-days-of-noticing"}`)
	if status, out := call("POST", "/challenges", `{"template":"seven-days-of-gratitude"}`); status != 409 || out["error"].(map[string]any)["code"] != "too_many_challenges" {
		t.Errorf("a fourth = %d %v", status, out)
	}

	// The alias ends the newest (noticing); the id route ends the one named.
	if status, _ := call("DELETE", "/challenges/current", ""); status != 204 {
		t.Fatalf("DELETE current = %d", status)
	}
	if ids := activeIDs(); len(ids) != 2 || ids[0] != firstID || ids[1] != secondID {
		t.Fatalf("active after DELETE current = %v", ids)
	}
	if status, _ := call("DELETE", "/challenges/"+firstID, ""); status != 204 {
		t.Fatalf("DELETE by id = %d", status)
	}
	if status, _ := call("DELETE", "/challenges/"+firstID, ""); status != 404 {
		t.Errorf("DELETE of one already ended = %d, want 404", status)
	}
	if ids := activeIDs(); len(ids) != 1 || ids[0] != secondID {
		t.Fatalf("active = %v, want only the second", ids)
	}
	// Ended is kept, not deleted.
	if _, out := call("GET", "/challenges/"+firstID, ""); out["data"].(map[string]any)["status"] != "ended" {
		t.Errorf("a left challenge = %v, want it kept as ended", out["data"])
	}
}
