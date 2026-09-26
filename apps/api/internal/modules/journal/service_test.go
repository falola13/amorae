package journal

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

// fakeRepo stands in for Postgres: Update and Delete only touch the entry
// when couple, author, and id all match, mirroring the SQL's WHERE clause.
type fakeRepo struct {
	e Entry
}

func (r *fakeRepo) List(context.Context, uuid.UUID) ([]Entry, error) { return []Entry{r.e}, nil }

func (r *fakeRepo) Create(_ context.Context, e Entry, _ time.Time) (Entry, error) {
	e.ID = r.e.ID
	e.Date = r.e.Date
	r.e = e
	return e, nil
}

func (r *fakeRepo) Update(_ context.Context, e Entry) (Entry, error) {
	if e.CoupleID != r.e.CoupleID || e.AuthorID != r.e.AuthorID || e.ID != r.e.ID {
		return Entry{}, ErrNotFound
	}
	e.Date = r.e.Date
	r.e = e
	return e, nil
}

func (r *fakeRepo) Delete(_ context.Context, coupleID, authorID, id uuid.UUID) error {
	if coupleID != r.e.CoupleID || authorID != r.e.AuthorID || id != r.e.ID {
		return ErrNotFound
	}
	r.e = Entry{}
	return nil
}

type fakeCouples struct{ id uuid.UUID }

func (c fakeCouples) CoupleFor(context.Context, uuid.UUID) (uuid.UUID, error) { return c.id, nil }

// fakePoker counts pokes so a test can check one happened, without either
// side knowing anything about how notifications work.
type fakePoker struct{ pokes int }

func (p *fakePoker) Poke() { p.pokes++ }

func newService(t *testing.T) (*Service, *fakeRepo, uuid.UUID, uuid.UUID) {
	t.Helper()
	couple, author := uuid.New(), uuid.New()
	repo := &fakeRepo{e: Entry{
		ID: uuid.New(), CoupleID: couple, AuthorID: author,
		Date: time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC),
		Tag:  TagGratitude, Text: "For the quiet morning.",
	}}
	return NewService(repo, fakeCouples{id: couple}, time.Now, &fakePoker{}), repo, couple, author
}

func TestAdd_PokesTheWorker(t *testing.T) {
	couple, author := uuid.New(), uuid.New()
	repo := &fakeRepo{e: Entry{ID: uuid.New(), CoupleID: couple, AuthorID: author}}
	poker := &fakePoker{}
	svc := NewService(repo, fakeCouples{id: couple}, time.Now, poker)

	if _, err := svc.Add(context.Background(), author, string(TagGratitude), "Something good."); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if poker.pokes != 1 {
		t.Errorf("pokes = %d, want 1 — the partner shouldn't wait for the next cron tick", poker.pokes)
	}
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()

	t.Run("the author can change tag and text", func(t *testing.T) {
		svc, repo, _, author := newService(t)
		want := repo.e.Date

		e, err := svc.Update(ctx, author, repo.e.ID, "Plans", "Sunday brunch.")
		if err != nil {
			t.Fatalf("updating: %v", err)
		}
		if e.Tag != TagPlans || e.Text != "Sunday brunch." {
			t.Errorf("tag = %q, text = %q", e.Tag, e.Text)
		}
		if !e.Date.Equal(want) {
			t.Errorf("date changed: got %v, want %v", e.Date, want)
		}
	})

	t.Run("someone else's entry is simply not found", func(t *testing.T) {
		svc, repo, _, _ := newService(t)
		stranger := uuid.New()

		_, err := svc.Update(ctx, stranger, repo.e.ID, "Plans", "Sunday brunch.")
		if err != ErrNotFound {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("a bad tag or blank text is refused before it reaches the repository", func(t *testing.T) {
		svc, repo, _, author := newService(t)
		before := repo.e

		_, err := svc.Update(ctx, author, repo.e.ID, "Nonsense", "")
		if _, ok := apperr.As(err); !ok {
			t.Fatalf("err = %v, want a validation error", err)
		}
		if repo.e != before {
			t.Error("a refused update still reached the repository")
		}
	})
}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	t.Run("the author can remove their own entry", func(t *testing.T) {
		svc, repo, _, author := newService(t)

		if err := svc.Delete(ctx, author, repo.e.ID); err != nil {
			t.Fatalf("deleting: %v", err)
		}
		if repo.e.ID != uuid.Nil {
			t.Error("the entry is still there")
		}
	})

	t.Run("someone else's entry is simply not found", func(t *testing.T) {
		svc, repo, _, _ := newService(t)
		stranger := uuid.New()
		before := repo.e

		err := svc.Delete(ctx, stranger, repo.e.ID)
		if err != ErrNotFound {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
		if repo.e != before {
			t.Error("the entry was removed by someone who didn't write it")
		}
	})
}
