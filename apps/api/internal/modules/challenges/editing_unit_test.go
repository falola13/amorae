package challenges

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

func TestChallenge_ScheduledHasNoDayYet(t *testing.T) {
	couple := uuid.New()
	c := running(couple, onDay(1))
	c.StartedOn = onDay(4)
	if c.TodayN() != 0 || c.Begun() || c.StartsIn() != 3 || c.Opened(1) {
		t.Errorf("today_n %d, begun %v, starts_in %d, day 1 open %v; want 0, false, 3, false",
			c.TodayN(), c.Begun(), c.StartsIn(), c.Opened(1))
	}
	c.Today = onDay(4)
	if c.TodayN() != 1 || !c.Begun() || c.StartsIn() != 0 || !c.Opened(1) {
		t.Errorf("on the day: today_n %d, begun %v, starts_in %d", c.TodayN(), c.Begun(), c.StartsIn())
	}
}

func TestChallenge_MayTouch(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	shared := Challenge{Kind: KindTogether, CreatedBy: a}
	own := Challenge{Kind: KindMine, CreatedBy: a}
	orphaned := Challenge{Kind: KindMine}
	legacy := Challenge{CreatedBy: a} // from before kinds: shared

	for name, tc := range map[string]struct {
		c    Challenge
		who  uuid.UUID
		want bool
	}{
		"shared, creator":  {shared, a, true},
		"shared, partner":  {shared, b, true},
		"mine, creator":    {own, a, true},
		"mine, partner":    {own, b, false},
		"mine, no creator": {orphaned, b, false},
		"no kind at all":   {legacy, b, true},
	} {
		if got := tc.c.MayTouch(tc.who); got != tc.want {
			t.Errorf("%s: MayTouch = %v, want %v", name, got, tc.want)
		}
	}
}

func TestCheckEdit_Rules(t *testing.T) {
	couple, a, b := uuid.New(), uuid.New(), uuid.New()
	title := func(s string) *string { return &s }
	day := func(n int) *time.Time { d := onDay(n); return &d }
	kind := func(k Kind) *Kind { return &k }

	begun := running(couple, onDay(2))
	begun.CreatedBy, begun.Kind, begun.Title = a, KindTogether, "Old"
	scheduled := running(couple, onDay(1))
	scheduled.CreatedBy, scheduled.Kind, scheduled.StartedOn = a, KindTogether, onDay(5)

	t.Run("a begun one keeps its start; the same day sent back is not a change", func(t *testing.T) {
		if _, err := begun.CheckEdit(a, Edit{StartedOn: day(3)}); codeOf(t, err) != "challenge_begun" {
			t.Errorf("= %v, want challenge_begun", err)
		}
		e, err := begun.CheckEdit(a, Edit{StartedOn: day(1), Title: title("New")})
		if err != nil || e.StartedOn != nil || e.Title == nil {
			t.Errorf("= %+v, %v; want the title only", e, err)
		}
	})

	t.Run("a scheduled one moves within today to sixty days", func(t *testing.T) {
		for n, wantErr := range map[int]bool{1: false, 7: false, 61: false, 62: true} {
			_, err := scheduled.CheckEdit(a, Edit{StartedOn: day(n)})
			if (err != nil) != wantErr {
				t.Errorf("day %d: err = %v, want error %v", n, err, wantErr)
			}
		}
		// Earlier than today is out, though today itself is fine.
		past := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
		if _, err := scheduled.CheckEdit(a, Edit{StartedOn: &past}); codeOf(t, err) != "validation_failed" {
			t.Errorf("a day in the past = %v", err)
		}
	})

	t.Run("kind: only the creator, and not once the partner has joined in", func(t *testing.T) {
		if _, err := begun.CheckEdit(b, Edit{Kind: kind(KindMine)}); codeOf(t, err) != "validation_failed" {
			t.Errorf("by the partner = %v", err)
		}
		if _, err := begun.CheckEdit(a, Edit{Kind: kind(KindMine)}); err != nil {
			t.Errorf("by the creator with nobody else on it = %v", err)
		}
		joined := begun
		joined.Days = append([]Day(nil), begun.Days...)
		joined.Days[0].Notes = map[uuid.UUID]string{b: "Hello."}
		if _, err := joined.CheckEdit(a, Edit{Kind: kind(KindMine)}); codeOf(t, err) != "challenge_kind_locked" {
			t.Errorf("after the partner's note = %v", err)
		}
		// Their own marks never lock it, and asking for what it already is is no change.
		own := begun
		own.Days = append([]Day(nil), begun.Days...)
		own.Days[0].Marks = map[uuid.UUID]Mark{a: MarkDone}
		if e, err := own.CheckEdit(a, Edit{Kind: kind(KindMine)}); err != nil || e.Kind == nil {
			t.Errorf("after the creator's own mark = %+v, %v", e, err)
		}
		if e, err := begun.CheckEdit(a, Edit{Kind: kind(KindTogether)}); err != nil || e.Kind != nil {
			t.Errorf("the same kind = %+v, %v; want it dropped", e, err)
		}
	})

	t.Run("not theirs, then over", func(t *testing.T) {
		own := begun
		own.Kind = KindMine
		if _, err := own.CheckEdit(b, Edit{Title: title("x")}); codeOf(t, err) != "challenge_not_found" {
			t.Errorf("the partner of a just-me one = %v", err)
		}
		over := begun
		over.Status = StatusEnded
		if _, err := over.CheckEdit(a, Edit{Title: title("x")}); codeOf(t, err) != "challenge_over" {
			t.Errorf("an ended one = %v", err)
		}
	})
}

func TestCheckPlan_ShorteningStopsAtTheFirstWrittenDay(t *testing.T) {
	couple, a := uuid.New(), uuid.New()
	c := running(couple, onDay(7))
	c.CreatedBy, c.Kind = a, KindTogether
	c.Days[4].Notes = map[uuid.UUID]string{a: "x"} // day 5
	c.Days[6].Marks = map[uuid.UUID]Mark{a: MarkDone}

	if err := c.CheckPlan(a, make([]string, 7)); err != nil {
		t.Errorf("same length = %v", err)
	}
	if err := c.CheckPlan(a, make([]string, 12)); err != nil {
		t.Errorf("extending = %v", err)
	}
	if err := c.CheckPlan(a, make([]string, 5)); codeOf(t, err) != "day_has_marks" {
		t.Errorf("dropping day 7 = %v", err)
	}
	err := c.CheckPlan(a, make([]string, 3))
	if codeOf(t, err) != "day_has_marks" || messageOf(err) != "Day 5 has been marked, so it can’t be removed." {
		t.Errorf("dropping 4-7 = %v, want the first written day named", err)
	}
	if err := c.CheckPlan(a, make([]string, 7)); err != nil {
		t.Errorf("= %v", err)
	}
}

func messageOf(err error) string {
	if ae, ok := apperr.As(err); ok {
		return ae.Message
	}
	return ""
}

func TestService_StartPassesWhatItWasGiven(t *testing.T) {
	ctx := context.Background()
	couple, user := uuid.New(), uuid.New()
	current := running(couple, onDay(3))
	current.CreatedBy, current.Kind, current.Title, current.Template = user, KindMine, "Mine", "ten-conversations"
	repo := &fakeRepo{current: current}
	poker := &fakePoker{}
	svc := NewService(repo, fakeCouples{id: couple}, func() time.Time { return time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC) }, poker)

	// Kind and a start day go through; a bad one never reaches the repository.
	if _, err := svc.Start(ctx, user, StartInput{Template: "ten-conversations", Kind: "mine", StartedOn: "2026-09-10"}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	spec := repo.specs[0]
	if spec.Kind != KindMine || spec.StartedOn == nil || !spec.StartedOn.Equal(onDay(10)) || spec.Template.Key != "ten-conversations" {
		t.Errorf("spec = %+v", spec)
	}
	for _, in := range []StartInput{
		{Template: "ten-conversations", Kind: "ours"},
		{Template: "ten-conversations", StartedOn: "2026-09-02"},
		{Template: "ten-conversations", StartedOn: "2026-12-01"},
		{Template: "ten-conversations", StartedOn: "later"},
		{Again: current.ID.String(), Template: "ten-conversations"},
	} {
		if _, err := svc.Start(ctx, user, in); codeOf(t, err) != "validation_failed" {
			t.Errorf("%+v = %v, want validation_failed", in, err)
		}
	}
	if len(repo.specs) != 1 || poker.pokes != 1 {
		t.Errorf("repository saw %d starts, %d pokes; want 1 and 1", len(repo.specs), poker.pokes)
	}

	// Again copies the plan, title and template; its kind follows who asks.
	if _, err := svc.Start(ctx, user, StartInput{Again: current.ID.String()}); err != nil {
		t.Fatalf("Start again: %v", err)
	}
	again := repo.specs[1]
	if again.Template.Key != "ten-conversations" || again.Template.Title != "Mine" || len(again.Template.Prompts) != 7 || again.Kind != KindMine {
		t.Errorf("again by its creator = %+v", again)
	}
	if _, err := svc.Start(ctx, uuid.New(), StartInput{Again: current.ID.String()}); err != nil {
		t.Fatalf("Start again by the partner: %v", err)
	}
	if repo.specs[2].Kind != KindTogether {
		t.Errorf("again by the partner = %s, want together", repo.specs[2].Kind)
	}
}

func TestService_WritesToAJustMeChallengeAreItsCreatorsOnly(t *testing.T) {
	ctx := context.Background()
	couple, creator, other := uuid.New(), uuid.New(), uuid.New()
	current := running(couple, onDay(2))
	current.CreatedBy, current.Kind = creator, KindMine
	repo := &fakeRepo{current: current}
	svc := NewService(repo, fakeCouples{id: couple}, time.Now, &fakePoker{})

	yes := true
	if _, err := svc.Mark(ctx, other, current.ID, 1, &yes, nil, nil); codeOf(t, err) != "challenge_not_found" {
		t.Errorf("Mark = %v", err)
	}
	if _, err := svc.Reflect(ctx, other, current.ID, "x"); codeOf(t, err) != "challenge_not_found" {
		t.Errorf("Reflect = %v", err)
	}
	if repo.records != 0 || len(repo.reflects) != 0 {
		t.Errorf("the repository was written to: %d records, %d reflections", repo.records, len(repo.reflects))
	}
	if _, err := svc.Mark(ctx, creator, current.ID, 1, &yes, nil, nil); err != nil {
		t.Errorf("the creator's Mark = %v", err)
	}
}
