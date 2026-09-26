package appreciation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeRepo struct{ created Appreciation }

func (r *fakeRepo) List(context.Context, uuid.UUID) ([]Appreciation, error) { return nil, nil }
func (r *fakeRepo) ByID(context.Context, uuid.UUID, uuid.UUID) (Appreciation, error) {
	return r.created, nil
}
func (r *fakeRepo) Create(_ context.Context, a Appreciation, at time.Time) (Appreciation, error) {
	a.ID, a.SentAt = uuid.New(), at
	r.created = a
	return a, nil
}
func (r *fakeRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }

type fakeCouples struct{ id uuid.UUID }

func (c fakeCouples) CoupleFor(context.Context, uuid.UUID) (uuid.UUID, error) { return c.id, nil }

// fakePoker counts pokes so a test can check one happened, without either
// side knowing anything about how notifications work.
type fakePoker struct{ pokes int }

func (p *fakePoker) Poke() { p.pokes++ }

func TestSend_PokesTheWorker(t *testing.T) {
	couple := uuid.New()
	poker := &fakePoker{}
	svc := NewService(&fakeRepo{}, fakeCouples{id: couple}, time.Now, poker)

	if _, err := svc.Send(context.Background(), uuid.New(), "You made today easier."); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if poker.pokes != 1 {
		t.Errorf("pokes = %d, want 1 — the partner shouldn't wait for the next cron tick", poker.pokes)
	}
}
