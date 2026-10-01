package challenges_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/challenges"
	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/database/dbtest"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
	"github.com/falola13/amorae/apps/api/internal/platform/metrics"
)

var sept10 = time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)

func sp(s string) *string { return &s }

func plan(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "Day text."
	}
	return out
}

func mine(in challenges.StartInput) challenges.StartInput {
	in.Kind = "mine"
	return in
}

func fieldOf(t *testing.T, err error, field string) string {
	t.Helper()
	ae, ok := apperr.As(err)
	if !ok {
		t.Fatalf("err = %v, want an apperr", err)
	}
	return ae.Fields[field]
}

func messageOf(t *testing.T, err error) string {
	t.Helper()
	ae, ok := apperr.As(err)
	if !ok {
		t.Fatalf("err = %v, want an apperr", err)
	}
	return ae.Message
}

func mustStart(t *testing.T, svc *challenges.Service, who uuid.UUID, in challenges.StartInput) challenges.Viewer {
	t.Helper()
	v, err := svc.Start(context.Background(), who, in)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	return v
}

func TestEdit_Title(t *testing.T) {
	ctx := context.Background()
	svc, _, _, a, b := newService(t, "UTC", sept10)
	v := mustStart(t, svc, a, custom())

	// Either partner may rename a shared one; it is trimmed.
	got, err := svc.Edit(ctx, b, v.ID, challenges.EditInput{Title: sp("  Our week  ")})
	if err != nil || got.Title != "Our week" {
		t.Fatalf("Edit = %q, %v", got.Title, err)
	}

	for _, bad := range []string{"   ", "", strings.Repeat("x", 81)} {
		_, err := svc.Edit(ctx, a, v.ID, challenges.EditInput{Title: sp(bad)})
		if code(t, err) != "validation_failed" || fieldOf(t, err, "title") == "" {
			t.Errorf("title %q = %v, want a title field error", bad, err)
		}
	}
	if got, _ := svc.Get(ctx, a, v.ID); got.Title != "Our week" {
		t.Errorf("a refused edit changed the title to %q", got.Title)
	}
	// 80 is allowed.
	if _, err := svc.Edit(ctx, a, v.ID, challenges.EditInput{Title: sp(strings.Repeat("x", 80))}); err != nil {
		t.Errorf("an 80-character title: %v", err)
	}
}

func TestEdit_StartedOn_OnlyUntilItBegins(t *testing.T) {
	ctx := context.Background()
	svc, _, _, a, _ := newService(t, "UTC", sept10)
	const rangeMsg = "Pick a day from today to two months ahead."

	v := mustStart(t, svc, a, challenges.StartInput{Template: "ten-conversations", StartedOn: "2026-09-15"})
	if v.StartsIn() != 5 || v.TodayN() != 0 {
		t.Fatalf("scheduled: starts_in = %d, today_n = %d; want 5, 0", v.StartsIn(), v.TodayN())
	}

	// Moved later, then to today (which begins it).
	got, err := svc.Edit(ctx, a, v.ID, challenges.EditInput{StartedOn: sp("2026-09-20")})
	if err != nil || got.StartsIn() != 10 || got.StartedOn.Format(time.DateOnly) != "2026-09-20" {
		t.Fatalf("move later = %v, %v", got.StartedOn, err)
	}
	// The range: today to sixty days ahead, inclusive.
	for _, bad := range []string{"2026-09-09", "2026-11-10", "next week", ""} {
		_, err := svc.Edit(ctx, a, v.ID, challenges.EditInput{StartedOn: sp(bad)})
		if code(t, err) != "validation_failed" || fieldOf(t, err, "started_on") != rangeMsg {
			t.Errorf("started_on %q = %v, want the range error", bad, err)
		}
	}
	if got, err := svc.Edit(ctx, a, v.ID, challenges.EditInput{StartedOn: sp("2026-11-09")}); err != nil || got.StartsIn() != 60 {
		t.Errorf("sixty days ahead = %d, %v; want it allowed", got.StartsIn(), err)
	}
	got, err = svc.Edit(ctx, a, v.ID, challenges.EditInput{StartedOn: sp("2026-09-10")})
	if err != nil || got.StartsIn() != 0 || got.TodayN() != 1 {
		t.Fatalf("move to today = starts_in %d, today_n %d, %v", got.StartsIn(), got.TodayN(), err)
	}

	// Begun: the start cannot move, though the same day sent back is fine.
	_, err = svc.Edit(ctx, a, v.ID, challenges.EditInput{StartedOn: sp("2026-09-12")})
	if code(t, err) != "challenge_begun" || messageOf(t, err) != "It’s already begun — the start can’t move now." {
		t.Errorf("moving a begun challenge = %v, want challenge_begun", err)
	}
	if _, err := svc.Edit(ctx, a, v.ID, challenges.EditInput{StartedOn: sp("2026-09-10"), Title: sp("Renamed")}); err != nil {
		t.Errorf("sending the current start back with a title = %v", err)
	}
}

func TestEdit_StartedOn_UsesTheCouplesCalendar(t *testing.T) {
	ctx := context.Background()
	// Noon UTC on the 10th is just past midnight on the 11th in Auckland.
	svc, _, _, a, _ := newService(t, "Pacific/Auckland", sept10)
	if _, err := svc.Start(ctx, a, challenges.StartInput{Template: "ten-conversations", StartedOn: "2026-09-10"}); code(t, err) != "validation_failed" {
		t.Errorf("yesterday there = %v, want a range error", err)
	}
	v := mustStart(t, svc, a, challenges.StartInput{Template: "ten-conversations", StartedOn: "2026-09-11"})
	if v.TodayN() != 1 || v.StartsIn() != 0 {
		t.Errorf("starting on their today: today_n %d, starts_in %d", v.TodayN(), v.StartsIn())
	}
}

func TestEdit_Kind(t *testing.T) {
	ctx := context.Background()
	svc, _, _, a, b := newService(t, "UTC", sept10)
	v := mustStart(t, svc, a, custom())

	// Only the creator.
	_, err := svc.Edit(ctx, b, v.ID, challenges.EditInput{Kind: sp("mine")})
	if code(t, err) != "validation_failed" || fieldOf(t, err, "kind") == "" {
		t.Errorf("the partner changing the kind = %v", err)
	}
	for _, bad := range []string{"", "ours"} {
		if _, err := svc.Edit(ctx, a, v.ID, challenges.EditInput{Kind: sp(bad)}); code(t, err) != "validation_failed" {
			t.Errorf("kind %q = %v, want a field error", bad, err)
		}
	}

	// To just me and back while nobody else has joined in.
	got, err := svc.Edit(ctx, a, v.ID, challenges.EditInput{Kind: sp("mine")})
	if err != nil || got.Kind != challenges.KindMine {
		t.Fatalf("to mine = %v, %v", got.Kind, err)
	}
	if got, err = svc.Edit(ctx, a, v.ID, challenges.EditInput{Kind: sp("together")}); err != nil || got.Kind != challenges.KindTogether {
		t.Fatalf("back to together = %v, %v", got.Kind, err)
	}

	// The creator's own marks do not lock it; the partner's mark, or even a
	// note, does — until they take it back.
	if _, err := svc.Mark(ctx, a, v.ID, 1, &yes, nil, nil); err != nil {
		t.Fatalf("creator mark: %v", err)
	}
	if _, err := svc.Mark(ctx, b, v.ID, 1, nil, nil, sp("Here.")); err != nil {
		t.Fatalf("partner note: %v", err)
	}
	_, err = svc.Edit(ctx, a, v.ID, challenges.EditInput{Kind: sp("mine")})
	if code(t, err) != "challenge_kind_locked" || messageOf(t, err) != "Your partner has already joined in." {
		t.Errorf("after the partner wrote = %v, want challenge_kind_locked", err)
	}
	if _, err := svc.Mark(ctx, b, v.ID, 1, nil, nil, sp("")); err != nil {
		t.Fatalf("partner clears the note: %v", err)
	}
	if _, err := svc.Edit(ctx, a, v.ID, challenges.EditInput{Kind: sp("mine")}); err != nil {
		t.Errorf("once the partner has nothing on it: %v", err)
	}
}

// Making one shared again takes room on the partner's side too.
func TestEdit_Kind_BackToTogetherNeedsRoomForThePartner(t *testing.T) {
	ctx := context.Background()
	svc, clk, _, a, b := newService(t, "UTC", sept10)
	own := mustStart(t, svc, a, mine(custom()))
	for i := 0; i < challenges.MaxActive; i++ {
		clk.advance(time.Minute)
		mustStart(t, svc, b, mine(custom()))
	}
	_, err := svc.Edit(ctx, a, own.ID, challenges.EditInput{Kind: sp("together")})
	if code(t, err) != "too_many_challenges" || messageOf(t, err) != "Bo already has three going." {
		t.Errorf("sharing with a full partner = %v", err)
	}
}

func TestEdit_WhoMayAndWhenNot(t *testing.T) {
	ctx := context.Background()
	svc, _, _, a, b := newService(t, "UTC", sept10)

	own := mustStart(t, svc, a, mine(custom()))
	for name, err := range map[string]error{
		"edit": func() error { _, e := svc.Edit(ctx, b, own.ID, challenges.EditInput{Title: sp("Mine now")}); return e }(),
		"plan": func() error { _, e := svc.ReplacePlan(ctx, b, own.ID, plan(4)); return e }(),
	} {
		if code(t, err) != "challenge_not_found" {
			t.Errorf("the partner's %s of a just-me challenge = %v, want challenge_not_found", name, err)
		}
	}

	// Finished and ended are over.
	if err := svc.Leave(ctx, a, own.ID); err != nil {
		t.Fatalf("Leave: %v", err)
	}
	if _, err := svc.Edit(ctx, a, own.ID, challenges.EditInput{Title: sp("Too late")}); code(t, err) != "challenge_over" {
		t.Errorf("editing an ended one = %v, want challenge_over", err)
	}
	if _, err := svc.ReplacePlan(ctx, a, own.ID, plan(4)); code(t, err) != "challenge_over" {
		t.Errorf("re-planning an ended one = %v, want challenge_over", err)
	}
	if _, err := svc.Edit(ctx, a, uuid.New(), challenges.EditInput{Title: sp("x")}); code(t, err) != "challenge_not_found" {
		t.Errorf("editing a stranger's = %v, want challenge_not_found", err)
	}
}

func TestPlan_ReplaceExtendAndKeepMarks(t *testing.T) {
	ctx := context.Background()
	svc, _, _, a, b := newService(t, "UTC", sept10)
	v := mustStart(t, svc, a, custom())
	if _, err := svc.Mark(ctx, a, v.ID, 1, &yes, nil, sp("Did it.")); err != nil {
		t.Fatalf("Mark: %v", err)
	}
	if _, err := svc.Mark(ctx, b, v.ID, 1, nil, &yes, nil); err != nil {
		t.Fatalf("Mark: %v", err)
	}

	got, err := svc.ReplacePlan(ctx, b, v.ID, []string{"  Uno. ", "Dos.", "Tres.", "Cuatro."})
	if err != nil {
		t.Fatalf("ReplacePlan: %v", err)
	}
	if len(got.Days) != 4 || got.Days[0].Prompt != "Uno." || got.Days[3].Prompt != "Cuatro." || got.Days[3].N != 4 {
		t.Fatalf("days = %+v", got.Days)
	}
	// Day 1 keeps both marks and the note; the new day is empty.
	if got.Days[0].Marks[a] != challenges.MarkDone || got.Days[0].Marks[b] != challenges.MarkSkipped || got.Days[0].Notes[a] != "Did it." {
		t.Errorf("day 1 lost what was on it: %+v", got.Days[0])
	}
	if len(got.Days[3].Marks) != 0 || got.Status != challenges.StatusActive {
		t.Errorf("the new day = %+v, status %s", got.Days[3], got.Status)
	}
}

func TestPlan_ShortenOnlyWhereNothingWasWritten(t *testing.T) {
	ctx := context.Background()
	svc, clk, _, a, b := newService(t, "UTC", sept10)
	v := mustStart(t, svc, a, custom())
	if _, err := svc.ReplacePlan(ctx, a, v.ID, plan(5)); err != nil {
		t.Fatalf("extend: %v", err)
	}
	clk.advance(4 * 24 * time.Hour) // every day has opened

	// A note by the partner on day 5 keeps it; so does a mark on day 4.
	if _, err := svc.Mark(ctx, b, v.ID, 5, nil, nil, sp("Not yet.")); err != nil {
		t.Fatalf("note: %v", err)
	}
	_, err := svc.ReplacePlan(ctx, a, v.ID, plan(3))
	if code(t, err) != "day_has_marks" || messageOf(t, err) != "Day 5 has been marked, so it can’t be removed." {
		t.Fatalf("shortening past a note = %v", err)
	}
	if _, err := svc.Mark(ctx, a, v.ID, 4, &yes, nil, nil); err != nil {
		t.Fatalf("mark: %v", err)
	}
	_, err = svc.ReplacePlan(ctx, a, v.ID, plan(3))
	if code(t, err) != "day_has_marks" || messageOf(t, err) != "Day 4 has been marked, so it can’t be removed." {
		t.Fatalf("shortening past a mark = %v, want it to name day 4", err)
	}
	// Shortening only to 4 keeps the marked day and drops the one with the note? No: 5 has the note.
	if _, err := svc.ReplacePlan(ctx, a, v.ID, plan(4)); code(t, err) != "day_has_marks" {
		t.Errorf("dropping day 5 = %v, want day_has_marks", err)
	}
	if got, _ := svc.Get(ctx, a, v.ID); len(got.Days) != 5 {
		t.Fatalf("a refused change left %d days", len(got.Days))
	}

	// Take both back and it can go.
	if _, err := svc.Mark(ctx, b, v.ID, 5, nil, nil, sp("")); err != nil {
		t.Fatalf("clear note: %v", err)
	}
	got, err := svc.ReplacePlan(ctx, a, v.ID, plan(4))
	if err != nil || len(got.Days) != 4 || got.Days[3].Marks[a] != challenges.MarkDone {
		t.Errorf("shorten to 4 = %d days, %v; want day 4 and its mark kept", len(got.Days), err)
	}
	if _, err := svc.Mark(ctx, a, v.ID, 4, &no, nil, nil); err != nil {
		t.Fatalf("clear mark: %v", err)
	}
	if got, err := svc.ReplacePlan(ctx, a, v.ID, plan(3)); err != nil || len(got.Days) != 3 {
		t.Errorf("shorten to 3 = %d days, %v", len(got.Days), err)
	}
}

func TestPlan_ShorteningCanFinishIt(t *testing.T) {
	ctx := context.Background()
	svc, clk, _, a, b := newService(t, "UTC", sept10)
	v := mustStart(t, svc, a, custom())
	if _, err := svc.ReplacePlan(ctx, a, v.ID, plan(5)); err != nil {
		t.Fatalf("extend: %v", err)
	}
	clk.advance(4 * 24 * time.Hour)
	for n := 1; n <= 3; n++ {
		for _, who := range []uuid.UUID{a, b} {
			if _, err := svc.Mark(ctx, who, v.ID, n, &yes, nil, nil); err != nil {
				t.Fatalf("mark %d: %v", n, err)
			}
		}
	}
	if got, _ := svc.Get(ctx, a, v.ID); got.Status != challenges.StatusActive {
		t.Fatalf("status = %s, want active with two days to go", got.Status)
	}
	// Adding days to an unfinished one does not finish it; removing the last
	// two leaves every day answered by both.
	got, err := svc.ReplacePlan(ctx, a, v.ID, plan(3))
	if err != nil || got.Status != challenges.StatusFinished || got.EndedAt == nil {
		t.Fatalf("after shortening: %s, ended %v, %v; want finished", got.Status, got.EndedAt, err)
	}
	if got.CanEdit() {
		t.Error("a finished challenge can still be edited")
	}
}

func TestPlan_Validation(t *testing.T) {
	ctx := context.Background()
	svc, _, _, a, _ := newService(t, "UTC", sept10)
	v := mustStart(t, svc, a, custom())

	for name, prompts := range map[string][]string{
		"two days":          plan(2),
		"a hundred and one": plan(101),
		"an empty day":      {"One.", "  ", "Three."},
		"a day over 200":    {"One.", strings.Repeat("x", 201), "Three."},
		"nothing":           nil,
	} {
		_, err := svc.ReplacePlan(ctx, a, v.ID, prompts)
		if code(t, err) != "validation_failed" || fieldOf(t, err, "prompts") == "" {
			t.Errorf("%s = %v, want a prompts field error", name, err)
		}
	}
	if _, err := svc.ReplacePlan(ctx, a, v.ID, plan(challenges.MaxCustomDays)); err != nil {
		t.Errorf("a hundred days: %v", err)
	}
}

func TestMine_PartnerReadsButNeverWrites(t *testing.T) {
	ctx := context.Background()
	svc, clk, _, a, b := newService(t, "UTC", sept10)
	v := mustStart(t, svc, a, mine(custom()))
	if v.Kind != challenges.KindMine || !v.CanEdit() {
		t.Fatalf("kind = %s, can_edit = %v", v.Kind, v.CanEdit())
	}

	// Reading: by id, in the active list.
	seen, err := svc.Get(ctx, b, v.ID)
	if err != nil || seen.Kind != challenges.KindMine || seen.CanEdit() {
		t.Fatalf("partner's Get = %v, can_edit %v, %v; want readable and not editable", seen.Kind, seen.CanEdit(), err)
	}
	if active, err := svc.Active(ctx, b); err != nil || len(active) != 1 || active[0].CanEdit() {
		t.Errorf("partner's Active = %d, %v", len(active), err)
	}

	// Writing: all of it is somebody else's.
	if _, err := svc.Mark(ctx, b, v.ID, 1, &yes, nil, nil); code(t, err) != "challenge_not_found" {
		t.Errorf("partner marking = %v, want challenge_not_found", err)
	}
	if _, err := svc.Mark(ctx, b, v.ID, 1, nil, nil, sp("Hi.")); code(t, err) != "challenge_not_found" {
		t.Errorf("partner noting = %v, want challenge_not_found", err)
	}
	if err := svc.Leave(ctx, b, v.ID); code(t, err) != "challenge_not_found" {
		t.Errorf("partner ending = %v, want challenge_not_found", err)
	}
	if _, err := svc.Reflect(ctx, b, v.ID, "x"); code(t, err) != "challenge_not_found" {
		t.Errorf("partner reflecting on a going one = %v, want challenge_not_found", err)
	}
	if got, _ := svc.Get(ctx, a, v.ID); len(got.Days[0].Marks) != 0 || got.Status != challenges.StatusActive {
		t.Fatalf("the partner changed it: %+v", got)
	}

	// It finishes on its creator's answers alone.
	clk.advance(2 * 24 * time.Hour)
	for n := 1; n <= 3; n++ {
		if _, err := svc.Mark(ctx, a, v.ID, n, &yes, nil, nil); err != nil {
			t.Fatalf("mark %d: %v", n, err)
		}
	}
	done, _ := svc.Get(ctx, b, v.ID)
	if done.Status != challenges.StatusFinished {
		t.Fatalf("status = %s, want finished without the partner", done.Status)
	}

	// Kept to read, and only its creator reflects on it.
	if _, err := svc.Reflect(ctx, b, v.ID, "Proud of you."); code(t, err) != "challenge_not_found" {
		t.Errorf("partner reflecting on a finished one = %v, want challenge_not_found", err)
	}
	if r, err := svc.Reflect(ctx, a, v.ID, "I did it."); err != nil || r.Reflections[a] != "I did it." {
		t.Errorf("creator reflecting = %v, %v", r.Reflections, err)
	}
	past, err := svc.Past(ctx, b)
	if err != nil || len(past) != 1 || past[0].Kind != challenges.KindMine || past[0].CreatedBy != a {
		t.Errorf("partner's Past = %+v, %v", past, err)
	}
}

func TestMine_SharedStillNeedsBothPartners(t *testing.T) {
	ctx := context.Background()
	svc, clk, _, a, b := newService(t, "UTC", sept10)
	v := mustStart(t, svc, a, custom())
	clk.advance(2 * 24 * time.Hour)
	for n := 1; n <= 3; n++ {
		if _, err := svc.Mark(ctx, a, v.ID, n, &yes, nil, nil); err != nil {
			t.Fatalf("mark %d: %v", n, err)
		}
	}
	if got, _ := svc.Get(ctx, b, v.ID); got.Status != challenges.StatusActive {
		t.Errorf("a shared challenge finished on one partner's answers: %s", got.Status)
	}
}

func TestCap_IsPerPerson(t *testing.T) {
	ctx := context.Background()
	svc, clk, _, a, b := newService(t, "UTC", sept10)
	start := func(who uuid.UUID, in challenges.StartInput) (challenges.Viewer, error) {
		clk.advance(time.Minute)
		return svc.Start(ctx, who, in)
	}

	// Ada's own three do not count against Bo.
	for i := 0; i < challenges.MaxActive; i++ {
		if _, err := start(a, mine(custom())); err != nil {
			t.Fatalf("Ada's own %d: %v", i+1, err)
		}
	}
	if _, err := start(a, mine(custom())); code(t, err) != "too_many_challenges" {
		t.Errorf("Ada's fourth = %v, want too_many_challenges", err)
	}
	if _, err := start(a, custom()); code(t, err) != "too_many_challenges" {
		t.Errorf("a shared one for Ada at three = %v, want too_many_challenges", err)
	}
	for i := 0; i < 2; i++ {
		if _, err := start(b, mine(custom())); err != nil {
			t.Fatalf("Bo's own %d, with Ada at three of hers: %v", i+1, err)
		}
	}
	// Bo has room, but a shared one needs it on Ada's side too, and she is full.
	if _, err := start(b, custom()); code(t, err) != "too_many_challenges" || messageOf(t, err) != "Ada already has three going." {
		t.Errorf("Bo sharing with a full Ada = %v, want her named", err)
	}
	// Bo's third of his own is fine; his fourth is not.
	if _, err := start(b, mine(custom())); err != nil {
		t.Errorf("Bo's third own: %v", err)
	}
	if _, err := start(b, mine(custom())); code(t, err) != "too_many_challenges" || messageOf(t, err) != "Three at once is plenty — finish or end one first." {
		t.Errorf("Bo's fourth = %v", err)
	}
}

func TestCap_SharedCountsForBoth(t *testing.T) {
	ctx := context.Background()
	svc, clk, _, a, b := newService(t, "UTC", sept10)
	for i := 0; i < 2; i++ {
		clk.advance(time.Minute)
		mustStart(t, svc, a, custom())
	}
	// Bo has those two shared ones too; his own makes his third.
	clk.advance(time.Minute)
	mustStart(t, svc, b, mine(custom()))
	if _, err := svc.Start(ctx, a, custom()); code(t, err) != "too_many_challenges" {
		t.Errorf("a shared one with Bo full = %v, want too_many_challenges", err)
	}
	// His own does not count against her.
	clk.advance(time.Minute)
	if _, err := svc.Start(ctx, a, mine(custom())); err != nil {
		t.Errorf("Ada's own third: %v", err)
	}
}

func TestDuplicate_ALibraryChallengeOncePerPerson(t *testing.T) {
	ctx := context.Background()
	const key = "ten-conversations"
	cases := []struct {
		name    string
		first   func(a, b uuid.UUID) (uuid.UUID, challenges.StartInput)
		second  func(a, b uuid.UUID) (uuid.UUID, challenges.StartInput)
		refused bool
	}{
		{"shared twice", func(a, b uuid.UUID) (uuid.UUID, challenges.StartInput) { return a, tpl(key) },
			func(a, b uuid.UUID) (uuid.UUID, challenges.StartInput) { return b, tpl(key) }, true},
		{"own twice by one person", func(a, b uuid.UUID) (uuid.UUID, challenges.StartInput) { return a, mine(tpl(key)) },
			func(a, b uuid.UUID) (uuid.UUID, challenges.StartInput) { return a, mine(tpl(key)) }, true},
		{"shared, then their own", func(a, b uuid.UUID) (uuid.UUID, challenges.StartInput) { return a, tpl(key) },
			func(a, b uuid.UUID) (uuid.UUID, challenges.StartInput) { return b, mine(tpl(key)) }, true},
		{"own, then shared", func(a, b uuid.UUID) (uuid.UUID, challenges.StartInput) { return a, mine(tpl(key)) },
			func(a, b uuid.UUID) (uuid.UUID, challenges.StartInput) { return b, tpl(key) }, true},
		{"each their own", func(a, b uuid.UUID) (uuid.UUID, challenges.StartInput) { return a, mine(tpl(key)) },
			func(a, b uuid.UUID) (uuid.UUID, challenges.StartInput) { return b, mine(tpl(key)) }, false},
		{"one they wrote, twice", func(a, b uuid.UUID) (uuid.UUID, challenges.StartInput) { return a, mine(custom()) },
			func(a, b uuid.UUID) (uuid.UUID, challenges.StartInput) { return a, mine(custom()) }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, clk, _, a, b := newService(t, "UTC", sept10)
			who, in := tc.first(a, b)
			mustStart(t, svc, who, in)
			clk.advance(time.Minute)
			who, in = tc.second(a, b)
			_, err := svc.Start(ctx, who, in)
			if tc.refused && code(t, err) != "challenge_already_running" {
				t.Errorf("second = %v, want challenge_already_running", err)
			}
			if !tc.refused && err != nil {
				t.Errorf("second = %v, want it allowed", err)
			}
		})
	}
}

func TestStart_ScheduledNeverOpensAndStillCounts(t *testing.T) {
	ctx := context.Background()
	svc, clk, _, a, b := newService(t, "UTC", sept10)
	v := mustStart(t, svc, a, challenges.StartInput{Template: "ten-conversations", StartedOn: "2026-09-13"})
	if v.TodayN() != 0 || v.StartsIn() != 3 {
		t.Fatalf("today_n %d, starts_in %d", v.TodayN(), v.StartsIn())
	}
	if v.Opened(1) {
		t.Error("day 1 is open before the start")
	}
	for _, who := range []uuid.UUID{a, b} {
		if _, err := svc.Mark(ctx, who, v.ID, 1, &yes, nil, nil); code(t, err) != "day_not_open" {
			t.Errorf("marking before it begins = %v, want day_not_open", err)
		}
	}

	// It counts toward the cap, for both of them.
	clk.advance(time.Minute)
	mustStart(t, svc, a, challenges.StartInput{Template: "seven-days-of-noticing", StartedOn: "2026-10-01"})
	clk.advance(time.Minute)
	mustStart(t, svc, a, challenges.StartInput{Template: "seven-days-of-gratitude", StartedOn: "2026-10-02"})
	if _, err := svc.Start(ctx, a, custom()); code(t, err) != "too_many_challenges" {
		t.Errorf("a fourth with three scheduled = %v, want too_many_challenges", err)
	}

	// Out of range at the start, too.
	for _, bad := range []string{"2026-09-09", "2026-11-10", "soon"} {
		if _, err := svc.Start(ctx, b, challenges.StartInput{Template: "advent", StartedOn: bad, Kind: "mine"}); fieldOf(t, err, "started_on") == "" {
			t.Errorf("started_on %q at the start = %v, want a field error", bad, err)
		}
	}
	// Once the day comes it opens.
	clk.advance(3 * 24 * time.Hour)
	if got, _ := svc.Get(ctx, a, v.ID); got.TodayN() != 1 || got.StartsIn() != 0 || !got.Opened(1) {
		t.Errorf("on the day: today_n %d, starts_in %d", got.TodayN(), got.StartsIn())
	}
	if _, err := svc.Mark(ctx, a, v.ID, 1, &yes, nil, nil); err != nil {
		t.Errorf("marking on the day: %v", err)
	}
}

func TestStart_Kind(t *testing.T) {
	ctx := context.Background()
	svc, _, _, a, _ := newService(t, "UTC", sept10)
	if v := mustStart(t, svc, a, custom()); v.Kind != challenges.KindTogether {
		t.Errorf("default kind = %s", v.Kind)
	}
	if _, err := svc.Start(ctx, a, challenges.StartInput{Template: "ten-conversations", Kind: "ours"}); fieldOf(t, err, "kind") == "" {
		t.Errorf("an unknown kind = %v, want a kind field error", err)
	}
}

func TestAgain_CopiesTheCurrentPlan(t *testing.T) {
	ctx := context.Background()
	svc, clk, _, a, b := newService(t, "UTC", sept10)

	// A library one: refused while it is still running, fine once it is not.
	lib := mustStart(t, svc, a, tpl("ten-conversations"))
	if _, err := svc.Start(ctx, a, challenges.StartInput{Again: lib.ID.String()}); code(t, err) != "challenge_already_running" {
		t.Errorf("again while it runs = %v, want challenge_already_running", err)
	}
	if err := svc.Leave(ctx, a, lib.ID); err != nil {
		t.Fatalf("Leave: %v", err)
	}
	clk.advance(time.Minute)
	again, err := svc.Start(ctx, b, challenges.StartInput{Again: lib.ID.String(), StartedOn: "2026-09-14"})
	if err != nil {
		t.Fatalf("Start again: %v", err)
	}
	if again.ID == lib.ID || again.Template != "ten-conversations" || again.Title != lib.Title || len(again.Days) != len(lib.Days) ||
		again.Kind != challenges.KindTogether || again.CreatedBy != b || again.StartsIn() != 4 || again.Status != challenges.StatusActive {
		t.Errorf("again = %+v", again.Challenge)
	}
	if len(again.Days[0].Marks) != 0 {
		t.Error("marks came along")
	}

	// One they wrote, with its plan as it is now, not as it began.
	own := mustStart(t, svc, a, custom())
	if _, err := svc.ReplacePlan(ctx, a, own.ID, []string{"Eins.", "Zwei.", "Drei.", "Vier."}); err != nil {
		t.Fatalf("ReplacePlan: %v", err)
	}
	if _, err := svc.Edit(ctx, a, own.ID, challenges.EditInput{Title: sp("Renamed")}); err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if err := svc.Leave(ctx, a, own.ID); err != nil {
		t.Fatalf("Leave: %v", err)
	}
	clk.advance(time.Minute)
	copied, err := svc.Start(ctx, a, challenges.StartInput{Again: own.ID.String()})
	if err != nil || copied.Title != "Renamed" || copied.Template != challenges.CustomKey || len(copied.Days) != 4 || copied.Days[3].Prompt != "Vier." {
		t.Fatalf("again of an edited one = %+v, %v", copied.Challenge, err)
	}
}

func TestAgain_JustMeStaysTheirsOnlyForWhoMadeIt(t *testing.T) {
	ctx := context.Background()
	svc, clk, _, a, b := newService(t, "UTC", sept10)
	orig := mustStart(t, svc, a, mine(custom()))
	if err := svc.Leave(ctx, a, orig.ID); err != nil {
		t.Fatalf("Leave: %v", err)
	}

	clk.advance(time.Minute)
	byCreator, err := svc.Start(ctx, a, challenges.StartInput{Again: orig.ID.String()})
	if err != nil || byCreator.Kind != challenges.KindMine {
		t.Fatalf("again by its creator = %v, %v; want it still just theirs", byCreator.Kind, err)
	}
	// The partner can read the original, and doing it again makes a shared one.
	clk.advance(time.Minute)
	byOther, err := svc.Start(ctx, b, challenges.StartInput{Again: orig.ID.String()})
	if err != nil || byOther.Kind != challenges.KindTogether || byOther.CreatedBy != b {
		t.Fatalf("again by the partner = %v, %v; want a shared one", byOther.Kind, err)
	}
}

func TestAgain_Refusals(t *testing.T) {
	ctx := context.Background()
	svc, _, _, a, _ := newService(t, "UTC", sept10)
	v := mustStart(t, svc, a, custom())

	_, err := svc.Start(ctx, a, challenges.StartInput{Again: v.ID.String(), Template: "ten-conversations"})
	if code(t, err) != "validation_failed" || fieldOf(t, err, "again") == "" {
		t.Errorf("again with a template = %v, want a field error on again", err)
	}
	_, err = svc.Start(ctx, a, challenges.StartInput{Again: v.ID.String(), Custom: &challenges.Custom{Title: "x", Prompts: plan(3)}})
	if code(t, err) != "validation_failed" || fieldOf(t, err, "again") == "" {
		t.Errorf("again with a custom = %v, want a field error on again", err)
	}
	for _, id := range []string{uuid.NewString(), "nonsense"} {
		if _, err := svc.Start(ctx, a, challenges.StartInput{Again: id}); code(t, err) != "challenge_not_found" {
			t.Errorf("again of %q = %v, want challenge_not_found", id, err)
		}
	}
	// A custom one can run alongside its own earlier self.
	if _, err := svc.Start(ctx, a, challenges.StartInput{Again: v.ID.String()}); err != nil {
		t.Errorf("again of a custom one still going: %v", err)
	}
}

type startPoker struct{ n int }

func (p *startPoker) Poke() { p.n++ }

// Starting is what tells the other partner, so it asks for a pass at once.
func TestStart_PokesTheWorker(t *testing.T) {
	db := dbtest.New(t)
	coupleID, a, _ := pair(t, db, "UTC")
	poker := &startPoker{}
	svc := challenges.NewService(challenges.NewPostgresRepository(db), fixedCouple{coupleID}, (&clock{t: sept10}).now, poker)
	mustStart(t, svc, a, custom())
	mustStart(t, svc, a, mine(custom()))
	if poker.n != 2 {
		t.Errorf("pokes = %d, want one per start", poker.n)
	}
	if _, err := svc.Start(context.Background(), a, challenges.StartInput{Template: "nope"}); err == nil || poker.n != 2 {
		t.Errorf("a refused start poked (%d) or succeeded (%v)", poker.n, err)
	}
}

// The new fields and routes over HTTP.
func TestEditing_Routes(t *testing.T) {
	db := dbtest.New(t)
	coupleID, a, b := pair(t, db, "UTC")
	clk := &clock{t: sept10}
	svc := challenges.NewService(challenges.NewPostgresRepository(db), fixedCouple{coupleID}, clk.now, noPoke{})

	mux := http.NewServeMux()
	// One router, so the caller is switched rather than a second one registered.
	caller := a
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(authctx.WithUserID(r.Context(), caller)))
		})
	}
	challenges.NewHandler(svc, fakePartners{a, b}).RegisterRoutes(httpx.NewRouter(mux, auth, metrics.New()))

	call := func(who uuid.UUID, method, path, body string) (int, map[string]any) {
		t.Helper()
		caller = who
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
	data := func(out map[string]any) map[string]any { return out["data"].(map[string]any) }
	errCode := func(out map[string]any) any { return out["error"].(map[string]any)["code"] }

	status, out := call(a, "POST", "/challenges", `{"custom":{"title":"Mine","prompts":["a","b","c"]},"kind":"mine","started_on":"2026-09-12"}`)
	if status != 201 {
		t.Fatalf("POST = %d %v", status, out)
	}
	d := data(out)
	id := d["id"].(string)
	if d["kind"] != "mine" || d["starts_in"] != float64(2) || d["today_n"] != float64(0) || d["can_edit"] != true || d["started_on"] != "2026-09-12" {
		t.Errorf("created = %v", d)
	}
	if d["days"].([]any)[0].(map[string]any)["open"] != false {
		t.Error("day 1 is open before the start")
	}

	// The partner reads it, but it is not theirs to edit or mark.
	_, out = call(b, "GET", "/challenges/"+id, "")
	if d := data(out); d["kind"] != "mine" || d["can_edit"] != false {
		t.Errorf("the partner's view = %v", d)
	}
	if status, out := call(b, "PATCH", "/challenges/"+id, `{"title":"Hers now"}`); status != 404 || errCode(out) != "challenge_not_found" {
		t.Errorf("partner PATCH = %d %v", status, out)
	}
	if status, out := call(b, "PUT", "/challenges/"+id+"/plan", `{"prompts":["a","b","c","d"]}`); status != 404 || errCode(out) != "challenge_not_found" {
		t.Errorf("partner PUT plan = %d %v", status, out)
	}
	if status, _ := call(b, "PATCH", "/challenges/"+id+"/days/1", `{"done":true}`); status != 404 {
		t.Errorf("partner marking = %d, want 404", status)
	}

	// The creator edits.
	status, out = call(a, "PATCH", "/challenges/"+id, `{"title":" Mine, renamed ","started_on":"2026-09-11","kind":"together"}`)
	if d := data(out); status != 200 || d["title"] != "Mine, renamed" || d["started_on"] != "2026-09-11" || d["kind"] != "together" || d["starts_in"] != float64(1) {
		t.Errorf("PATCH = %d %v", status, out)
	}
	if status, out := call(a, "PATCH", "/challenges/"+id, `{"started_on":"2027-01-01"}`); status != 400 || out["error"].(map[string]any)["fields"].(map[string]any)["started_on"] != "Pick a day from today to two months ahead." {
		t.Errorf("PATCH out of range = %d %v", status, out)
	}
	status, out = call(a, "PUT", "/challenges/"+id+"/plan", `{"prompts":["a","b","c","d"]}`)
	if d := data(out); status != 200 || len(d["days"].([]any)) != 4 {
		t.Errorf("PUT plan = %d %v", status, out)
	}
	if status, out := call(a, "PUT", "/challenges/"+id+"/plan", `{"prompts":["a"]}`); status != 400 || errCode(out) != "validation_failed" {
		t.Errorf("PUT short plan = %d %v", status, out)
	}

	// Once begun the start is fixed.
	clk.advance(24 * time.Hour)
	if status, out := call(a, "PATCH", "/challenges/"+id, `{"started_on":"2026-09-20"}`); status != 409 || errCode(out) != "challenge_begun" {
		t.Errorf("PATCH a begun challenge = %d %v", status, out)
	}
	// ... and the partner having joined in locks the kind (it is shared now).
	call(b, "PATCH", "/challenges/"+id+"/days/1", `{"done":true}`)
	if status, out := call(a, "PATCH", "/challenges/"+id, `{"kind":"mine"}`); status != 409 || errCode(out) != "challenge_kind_locked" {
		t.Errorf("PATCH kind after the partner joined = %d %v", status, out)
	}
	if status, out := call(a, "PUT", "/challenges/"+id+"/plan", `{"prompts":["a","b"]}`); status != 400 {
		t.Errorf("PUT 2 days = %d %v", status, out)
	}
	// Day 4 has nothing on it, so going back to three days is fine.
	if status, out := call(a, "PUT", "/challenges/"+id+"/plan", `{"prompts":["a","b","c"]}`); status != 200 {
		t.Errorf("PUT 3 days = %d %v", status, out)
	}

	// Do it again, from the past list; Past carries kind and created_by.
	if status, _ := call(a, "DELETE", "/challenges/"+id, ""); status != 204 {
		t.Fatalf("DELETE = %d", status)
	}
	_, out = call(b, "GET", "/challenges/past", "")
	past := out["data"].([]any)
	if len(past) != 1 || past[0].(map[string]any)["kind"] != "together" || past[0].(map[string]any)["created_by"] != a.String() {
		t.Errorf("past = %v", past)
	}
	status, out = call(b, "POST", "/challenges", `{"again":"`+id+`","kind":"mine"}`)
	if d := data(out); status != 201 || d["kind"] != "mine" || d["created_by"] != b.String() || d["title"] != "Mine, renamed" || d["can_edit"] != true {
		t.Errorf("again = %d %v", status, out)
	}
	if status, out := call(b, "POST", "/challenges", `{"again":"`+id+`","template":"advent"}`); status != 400 || out["error"].(map[string]any)["fields"].(map[string]any)["again"] == nil {
		t.Errorf("again with a template = %d %v", status, out)
	}
}
