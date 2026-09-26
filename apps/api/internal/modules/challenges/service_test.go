package challenges

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeRepo struct {
	current Challenge
	marks   int
	clears  int
}

func (r *fakeRepo) Current(context.Context, uuid.UUID) (Challenge, error) { return r.current, nil }
func (r *fakeRepo) Start(_ context.Context, coupleID uuid.UUID, t Template, on time.Time) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (r *fakeRepo) SetMark(context.Context, uuid.UUID, uuid.UUID, int, Mark, time.Time) error {
	r.marks++
	return nil
}
func (r *fakeRepo) ClearMark(context.Context, uuid.UUID, uuid.UUID, int) error {
	r.clears++
	return nil
}
func (r *fakeRepo) Leave(context.Context, uuid.UUID) error { return nil }

type fakeCouples struct{ id uuid.UUID }

func (c fakeCouples) CoupleFor(context.Context, uuid.UUID) (uuid.UUID, error) { return c.id, nil }

// fakePoker counts pokes so a test can check one happened, without either
// side knowing anything about how notifications work.
type fakePoker struct{ pokes int }

func (p *fakePoker) Poke() { p.pokes++ }

func TestMark_PokesOnlyWhenMarking(t *testing.T) {
	couple := uuid.New()
	repo := &fakeRepo{current: Challenge{ID: uuid.New(), CoupleID: couple, Days: []Day{{ID: uuid.New(), N: 1}}}}
	poker := &fakePoker{}
	svc := NewService(repo, fakeCouples{id: couple}, time.Now, poker)

	done := true
	t.Run("marking a day pokes", func(t *testing.T) {
		if _, err := svc.Mark(context.Background(), uuid.New(), 1, &done, nil); err != nil {
			t.Fatalf("Mark: %v", err)
		}
		if poker.pokes != 1 {
			t.Errorf("pokes = %d, want 1 — the both-marked notification shouldn't wait for the next cron tick", poker.pokes)
		}
		if repo.marks != 1 {
			t.Errorf("SetMark calls = %d, want 1", repo.marks)
		}
	})

	t.Run("clearing a mark pokes nobody", func(t *testing.T) {
		poker.pokes = 0
		notDone := false
		if _, err := svc.Mark(context.Background(), uuid.New(), 1, &notDone, nil); err != nil {
			t.Fatalf("Mark (clear): %v", err)
		}
		if poker.pokes != 0 {
			t.Errorf("pokes = %d, want 0 — nobody is told about a day being unmarked", poker.pokes)
		}
		if repo.clears != 1 {
			t.Errorf("ClearMark calls = %d, want 1", repo.clears)
		}
	})
}
