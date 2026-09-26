package goals

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeRepo struct{ goal Goal }

func (r *fakeRepo) List(context.Context, uuid.UUID) ([]Goal, error) { return nil, nil }
func (r *fakeRepo) ByID(context.Context, uuid.UUID, uuid.UUID) (Goal, error) {
	return r.goal, nil
}
func (r *fakeRepo) Create(_ context.Context, g Goal, _ time.Time) (uuid.UUID, error) {
	g.ID = uuid.New()
	r.goal = g
	return g.ID, nil
}
func (r *fakeRepo) Update(_ context.Context, g Goal, _ time.Time) error {
	r.goal = g
	return nil
}
func (r *fakeRepo) AddProgress(_ context.Context, _, _, _ uuid.UUID, _ int64, _ time.Time) error {
	return nil
}

type fakeCouples struct{ id uuid.UUID }

func (c fakeCouples) CoupleFor(context.Context, uuid.UUID) (uuid.UUID, error) { return c.id, nil }

// fakePoker counts pokes so a test can check one happened, without either
// side knowing anything about how notifications work.
type fakePoker struct{ pokes int }

func (p *fakePoker) Poke() { p.pokes++ }

func TestLogProgress_PokesTheWorker(t *testing.T) {
	couple := uuid.New()
	repo := &fakeRepo{goal: Goal{ID: uuid.New(), CoupleID: couple, Target: 1000}}
	poker := &fakePoker{}
	svc := NewService(repo, fakeCouples{id: couple}, time.Now, poker)

	if _, err := svc.LogProgress(context.Background(), uuid.New(), repo.goal.ID, 100); err != nil {
		t.Fatalf("LogProgress: %v", err)
	}
	if poker.pokes != 1 {
		t.Errorf("pokes = %d, want 1 — the partner shouldn't wait for the next cron tick", poker.pokes)
	}
}
