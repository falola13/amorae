package events

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

// fakeRepo is an in-memory Repository, scoped by couple the same way the
// Postgres one is: an id outside the couple simply is not there.
type fakeRepo struct {
	events map[uuid.UUID]Event
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{events: map[uuid.UUID]Event{}}
}

func (f *fakeRepo) List(_ context.Context, coupleID uuid.UUID) ([]Event, error) {
	var out []Event
	for _, e := range f.events {
		if e.CoupleID == coupleID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeRepo) ByID(_ context.Context, coupleID, eventID uuid.UUID) (Event, error) {
	e, ok := f.events[eventID]
	if !ok || e.CoupleID != coupleID {
		return Event{}, ErrNotFound
	}
	return e, nil
}

func (f *fakeRepo) Create(_ context.Context, e Event, _ time.Time) (uuid.UUID, error) {
	id := uuid.New()
	e.ID = id
	f.events[id] = e
	return id, nil
}

func (f *fakeRepo) Update(_ context.Context, e Event, _ bool, _ time.Time) error {
	current, ok := f.events[e.ID]
	if !ok || current.CoupleID != e.CoupleID {
		return ErrNotFound
	}
	e.CreatedBy = current.CreatedBy
	f.events[e.ID] = e
	return nil
}

func (f *fakeRepo) SetOutcome(_ context.Context, coupleID, eventID uuid.UUID, done, didntHappen bool, _ time.Time) error {
	e, ok := f.events[eventID]
	if !ok || e.CoupleID != coupleID {
		return ErrNotFound
	}
	e.Done = done
	e.DidntHappen = didntHappen
	f.events[eventID] = e
	return nil
}

func (f *fakeRepo) SetChecklistItem(_ context.Context, coupleID, eventID, itemID uuid.UUID, done bool, _ time.Time) error {
	e, ok := f.events[eventID]
	if !ok || e.CoupleID != coupleID {
		return ErrNotFound
	}
	for i, item := range e.Checklist {
		if item.ID == itemID {
			e.Checklist[i].Done = done
			f.events[eventID] = e
			return nil
		}
	}
	return ErrNotFound
}

func (f *fakeRepo) Delete(_ context.Context, coupleID, eventID uuid.UUID) error {
	e, ok := f.events[eventID]
	if !ok || e.CoupleID != coupleID {
		return ErrNotFound
	}
	delete(f.events, eventID)
	return nil
}

// fakeCouples pairs everybody with the same couple, since ownership — not
// couple scoping — is what this file is testing.
type fakeCouples struct {
	coupleID uuid.UUID
}

func (f fakeCouples) CoupleFor(context.Context, uuid.UUID) (uuid.UUID, error) {
	return f.coupleID, nil
}

// fakePoker counts pokes so a test can check one happened, without either
// side knowing anything about how notifications work.
type fakePoker struct{ pokes int }

func (p *fakePoker) Poke() { p.pokes++ }

func serviceFor(repo Repository) (*Service, uuid.UUID) {
	coupleID := uuid.New()
	now := func() time.Time { return time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC) }
	return NewService(repo, fakeCouples{coupleID: coupleID}, now, &fakePoker{}), coupleID
}

func TestService_Create_PokesTheWorker(t *testing.T) {
	poker := &fakePoker{}
	coupleID := uuid.New()
	now := func() time.Time { return time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC) }
	svc := NewService(newFakeRepo(), fakeCouples{coupleID: coupleID}, now, poker)

	if _, err := svc.Create(context.Background(), uuid.New(), Input{
		Title: ptr("Dinner"), Date: ptr("2026-10-10"),
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if poker.pokes != 1 {
		t.Errorf("pokes = %d, want 1 — a together event shouldn't wait for the next cron tick", poker.pokes)
	}
}

func TestService_Create(t *testing.T) {
	repo := newFakeRepo()
	svc, _ := serviceFor(repo)
	creator := uuid.New()

	t.Run("records who made it, and defaults to together", func(t *testing.T) {
		e, err := svc.Create(context.Background(), creator, Input{
			Title: ptr("Dinner"), Date: ptr("2026-10-10"),
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if e.CreatedBy == nil || *e.CreatedBy != creator {
			t.Errorf("created_by = %v, want %s", e.CreatedBy, creator)
		}
		if e.Kind != KindTogether {
			t.Errorf("kind = %q, want %q", e.Kind, KindTogether)
		}
	})

	t.Run("mine is honoured on the way in", func(t *testing.T) {
		e, err := svc.Create(context.Background(), creator, Input{
			Title: ptr("Dentist"), Date: ptr("2026-10-11"), Kind: ptr("mine"),
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if e.Kind != KindMine {
			t.Errorf("kind = %q, want %q", e.Kind, KindMine)
		}
	})
}

// mineEvent seeds a "mine" event belonging to creator, in the service's couple.
func mineEvent(t *testing.T, repo *fakeRepo, coupleID, creator uuid.UUID) Event {
	t.Helper()
	id := uuid.New()
	e := Event{
		ID: id, CoupleID: coupleID, CreatedBy: &creator, Kind: KindMine,
		Title: "Dentist", Date: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC),
		Checklist: []ChecklistItem{{ID: uuid.New(), Text: "Bring the card"}},
	}
	repo.events[id] = e
	return e
}

func togetherEvent(t *testing.T, repo *fakeRepo, coupleID, creator uuid.UUID) Event {
	t.Helper()
	id := uuid.New()
	e := Event{
		ID: id, CoupleID: coupleID, CreatedBy: &creator, Kind: KindTogether,
		Title: "Dinner", Date: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC),
		Checklist: []ChecklistItem{{ID: uuid.New(), Text: "Book a table"}},
	}
	repo.events[id] = e
	return e
}

func TestService_MineEventIsClosedToTheOtherPartner(t *testing.T) {
	repo := newFakeRepo()
	svc, coupleID := serviceFor(repo)
	creator, partner := uuid.New(), uuid.New()

	newMine := func() Event { return mineEvent(t, repo, coupleID, creator) }

	t.Run("the partner cannot update it", func(t *testing.T) {
		e := newMine()
		_, err := svc.Update(context.Background(), partner, e.ID, Input{Title: ptr("Snooping")})
		if err != ErrNotFound {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("the partner cannot delete it", func(t *testing.T) {
		e := newMine()
		if err := svc.Delete(context.Background(), partner, e.ID); err != ErrNotFound {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("the partner cannot complete it", func(t *testing.T) {
		e := newMine()
		_, err := svc.SetDone(context.Background(), partner, e.ID, true)
		if err != ErrNotFound {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("the partner cannot tick its checklist", func(t *testing.T) {
		e := newMine()
		_, err := svc.SetChecklistItem(context.Background(), partner, e.ID, e.Checklist[0].ID, true)
		if err != ErrNotFound {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("but the creator can do every one of those", func(t *testing.T) {
		e := newMine()
		if _, err := svc.Update(context.Background(), creator, e.ID, Input{Title: ptr("Dentist, moved")}); err != nil {
			t.Errorf("Update: %v", err)
		}
		if _, err := svc.SetDone(context.Background(), creator, e.ID, true); err != nil {
			t.Errorf("SetDone: %v", err)
		}
		if _, err := svc.SetChecklistItem(context.Background(), creator, e.ID, e.Checklist[0].ID, true); err != nil {
			t.Errorf("SetChecklistItem: %v", err)
		}
		if err := svc.Delete(context.Background(), creator, e.ID); err != nil {
			t.Errorf("Delete: %v", err)
		}
	})

	t.Run("both partners can still read it", func(t *testing.T) {
		e := newMine()
		if _, err := svc.Get(context.Background(), partner, e.ID); err != nil {
			t.Errorf("a partner could not even see the event: %v", err)
		}
	})
}

func TestService_TogetherEventIsOpenToEitherPartner(t *testing.T) {
	repo := newFakeRepo()
	svc, coupleID := serviceFor(repo)
	creator, partner := uuid.New(), uuid.New()

	e := togetherEvent(t, repo, coupleID, creator)
	if _, err := svc.Update(context.Background(), partner, e.ID, Input{Title: ptr("Dinner, later")}); err != nil {
		t.Errorf("the partner could not edit a together event: %v", err)
	}
	if _, err := svc.SetDone(context.Background(), partner, e.ID, true); err != nil {
		t.Errorf("the partner could not complete a together event: %v", err)
	}
}

func TestService_OnlyTheCreatorChangesKind(t *testing.T) {
	repo := newFakeRepo()
	svc, coupleID := serviceFor(repo)
	creator, partner := uuid.New(), uuid.New()

	t.Run("the partner is refused with a field error, not silently ignored", func(t *testing.T) {
		e := togetherEvent(t, repo, coupleID, creator)
		_, err := svc.Update(context.Background(), partner, e.ID, Input{Kind: ptr("mine")})
		appErr, ok := err.(*apperr.Error)
		if !ok {
			t.Fatalf("err = %T (%v), want *apperr.Error", err, err)
		}
		if appErr.Fields["kind"] == "" {
			t.Errorf("fields = %v, want kind", appErr.Fields)
		}
	})

	t.Run("the creator may", func(t *testing.T) {
		e := togetherEvent(t, repo, coupleID, creator)
		updated, err := svc.Update(context.Background(), creator, e.ID, Input{Kind: ptr("mine")})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if updated.Kind != KindMine {
			t.Errorf("kind = %q, want %q", updated.Kind, KindMine)
		}
	})

	t.Run("with no creator on record, either partner may", func(t *testing.T) {
		id := uuid.New()
		repo.events[id] = Event{
			ID: id, CoupleID: coupleID, CreatedBy: nil, Kind: KindTogether,
			Title: "From before ownership", Date: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC),
		}
		if _, err := svc.Update(context.Background(), partner, id, Input{Kind: ptr("mine")}); err != nil {
			t.Errorf("a pre-existing event refused a kind change: %v", err)
		}
	})
}

func TestService_SetOutcome(t *testing.T) {
	repo := newFakeRepo()
	svc, coupleID := serviceFor(repo)
	creator, partner := uuid.New(), uuid.New()
	ctx := context.Background()

	t.Run("happened marks it done", func(t *testing.T) {
		e := togetherEvent(t, repo, coupleID, creator)
		got, err := svc.SetOutcome(ctx, creator, e.ID, OutcomeHappened)
		if err != nil {
			t.Fatalf("SetOutcome: %v", err)
		}
		if !got.Done || got.DidntHappen {
			t.Errorf("done = %v, didnt_happen = %v", got.Done, got.DidntHappen)
		}
	})

	t.Run("didnt_happen replaces done rather than joining it", func(t *testing.T) {
		e := togetherEvent(t, repo, coupleID, creator)
		if _, err := svc.SetOutcome(ctx, creator, e.ID, OutcomeHappened); err != nil {
			t.Fatalf("SetOutcome: %v", err)
		}
		got, err := svc.SetOutcome(ctx, partner, e.ID, OutcomeDidntHappen)
		if err != nil {
			t.Fatalf("SetOutcome: %v", err)
		}
		if got.Done || !got.DidntHappen {
			t.Errorf("done = %v, didnt_happen = %v", got.Done, got.DidntHappen)
		}
		back, err := svc.SetOutcome(ctx, partner, e.ID, OutcomeHappened)
		if err != nil {
			t.Fatalf("SetOutcome: %v", err)
		}
		if !back.Done || back.DidntHappen {
			t.Errorf("done = %v, didnt_happen = %v", back.Done, back.DidntHappen)
		}
	})

	t.Run("none clears both", func(t *testing.T) {
		e := togetherEvent(t, repo, coupleID, creator)
		if _, err := svc.SetOutcome(ctx, creator, e.ID, OutcomeDidntHappen); err != nil {
			t.Fatalf("SetOutcome: %v", err)
		}
		got, err := svc.SetOutcome(ctx, creator, e.ID, OutcomeNone)
		if err != nil {
			t.Fatalf("SetOutcome: %v", err)
		}
		if got.Done || got.DidntHappen {
			t.Errorf("done = %v, didnt_happen = %v, want neither", got.Done, got.DidntHappen)
		}
	})

	t.Run("complete and uncomplete still work, and uncomplete clears didnt_happen too", func(t *testing.T) {
		e := togetherEvent(t, repo, coupleID, creator)
		got, err := svc.SetDone(ctx, creator, e.ID, true)
		if err != nil || !got.Done {
			t.Fatalf("SetDone(true) = %+v, %v", got, err)
		}
		if _, err := svc.SetOutcome(ctx, creator, e.ID, OutcomeDidntHappen); err != nil {
			t.Fatalf("SetOutcome: %v", err)
		}
		got, err = svc.SetDone(ctx, creator, e.ID, false)
		if err != nil || got.Done || got.DidntHappen {
			t.Errorf("SetDone(false) = %+v, %v", got, err)
		}
	})

	t.Run("anything else is refused", func(t *testing.T) {
		e := togetherEvent(t, repo, coupleID, creator)
		fields := fieldsOf(t, func() error { _, err := svc.SetOutcome(ctx, creator, e.ID, "maybe"); return err }())
		if fields["outcome"] == "" {
			t.Errorf("fields = %v, want outcome", fields)
		}
	})

	t.Run("a mine event's outcome is its creator's alone", func(t *testing.T) {
		e := togetherEvent(t, repo, coupleID, creator)
		e.Kind = KindMine
		repo.events[e.ID] = e
		for _, outcome := range []string{OutcomeHappened, OutcomeDidntHappen, OutcomeNone} {
			if _, err := svc.SetOutcome(ctx, partner, e.ID, outcome); err != ErrNotFound {
				t.Errorf("%s by the partner: err = %v, want ErrNotFound", outcome, err)
			}
		}
		if _, err := svc.SetOutcome(ctx, creator, e.ID, OutcomeDidntHappen); err != nil {
			t.Errorf("the creator was refused: %v", err)
		}
	})
}
