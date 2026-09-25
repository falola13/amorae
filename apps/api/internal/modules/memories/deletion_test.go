package memories

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/photos"
)

// A repository held in memory, so these tests are about what the service
// decides rather than what Postgres does with it.
type fakeRepo struct {
	m       Memory
	deleted bool
	refuse  error
}

func (r *fakeRepo) List(context.Context, uuid.UUID) ([]Memory, error) { return []Memory{r.m}, nil }

func (r *fakeRepo) ByID(_ context.Context, _, _ uuid.UUID) (Memory, error) {
	if r.deleted {
		return Memory{}, ErrNotFound
	}
	return r.m, nil
}

func (r *fakeRepo) Create(context.Context, Memory, time.Time) (uuid.UUID, error) {
	return r.m.ID, nil
}

func (r *fakeRepo) SetPhoto(_ context.Context, _, _ uuid.UUID, photoID string, _ time.Time) error {
	if r.refuse != nil {
		return r.refuse
	}
	r.m.PhotoID = photoID
	return nil
}

func (r *fakeRepo) Delete(_ context.Context, _, _ uuid.UUID) error {
	if r.refuse != nil {
		return r.refuse
	}
	r.deleted = true
	return nil
}

type fakePhotos struct {
	destroyed []string
	refuse    error
}

func (p *fakePhotos) Ticket(string, time.Time) (photos.Ticket, error) { return photos.Ticket{}, nil }
func (p *fakePhotos) URL(string) (string, error)                      { return "https://example.test/p", nil }

func (p *fakePhotos) Destroy(_ context.Context, publicID string) error {
	if p.refuse != nil {
		return p.refuse
	}
	p.destroyed = append(p.destroyed, publicID)
	return nil
}

type fakeCouples struct{ id uuid.UUID }

func (c fakeCouples) CoupleFor(context.Context, uuid.UUID) (uuid.UUID, error) { return c.id, nil }

func newService(t *testing.T, pics Photos) (*Service, *fakeRepo) {
	t.Helper()
	couple, me := uuid.New(), uuid.New()
	repo := &fakeRepo{m: Memory{
		ID: me, CoupleID: couple, Title: "The night it rained",
		Date:    time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC),
		PhotoID: photos.PublicID(couple, me),
	}}
	return NewService(repo, fakeCouples{id: couple}, pics, time.Now), repo
}

func TestRemovePhoto(t *testing.T) {
	ctx := context.Background()

	t.Run("it deletes the file, not just the pointer to it", func(t *testing.T) {
		pics := &fakePhotos{}
		svc, repo := newService(t, pics)
		want := repo.m.PhotoID

		m, err := svc.RemovePhoto(ctx, uuid.New(), repo.m.ID)
		if err != nil {
			t.Fatalf("removing: %v", err)
		}
		if len(pics.destroyed) != 1 || pics.destroyed[0] != want {
			t.Errorf("destroyed %v, want [%s]", pics.destroyed, want)
		}
		if m.HasPhoto() {
			t.Error("the memory still claims a photo")
		}
	})

	t.Run("the moment survives losing its picture", func(t *testing.T) {
		svc, repo := newService(t, &fakePhotos{})
		if _, err := svc.RemovePhoto(ctx, uuid.New(), repo.m.ID); err != nil {
			t.Fatalf("removing: %v", err)
		}
		if repo.deleted {
			t.Error("removing a photo took the memory with it")
		}
	})

	// The reason the delete comes before the database write. Saying the
	// picture is gone while it is still stored is the one outcome worth
	// ruling out; a refusal here has to leave the photo on the screen so
	// that asking again tries again.
	t.Run("a refusal at Cloudinary changes nothing here", func(t *testing.T) {
		boom := errors.New("cloudinary is having a day")
		svc, repo := newService(t, &fakePhotos{refuse: boom})

		if _, err := svc.RemovePhoto(ctx, uuid.New(), repo.m.ID); !errors.Is(err, boom) {
			t.Fatalf("err = %v, want the refusal", err)
		}
		if !repo.m.HasPhoto() {
			t.Error("the pointer was cleared for a photo that still exists")
		}
	})
}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	t.Run("it takes the picture with it", func(t *testing.T) {
		pics := &fakePhotos{}
		svc, repo := newService(t, pics)
		want := repo.m.PhotoID

		if err := svc.Delete(ctx, uuid.New(), repo.m.ID); err != nil {
			t.Fatalf("deleting: %v", err)
		}
		if !repo.deleted {
			t.Error("the memory is still there")
		}
		// Otherwise the row that held the file's name is gone and the file
		// is not: paid for, and reachable by nobody.
		if len(pics.destroyed) != 1 || pics.destroyed[0] != want {
			t.Errorf("destroyed %v, want [%s]", pics.destroyed, want)
		}
	})

	t.Run("a memory with no picture needs no storage at all", func(t *testing.T) {
		svc, repo := newService(t, nil)
		repo.m.PhotoID = ""
		if err := svc.Delete(ctx, uuid.New(), repo.m.ID); err != nil {
			t.Fatalf("deleting: %v", err)
		}
		if !repo.deleted {
			t.Error("the memory is still there")
		}
	})

	t.Run("a refusal at Cloudinary keeps the memory", func(t *testing.T) {
		boom := errors.New("cloudinary is having a day")
		svc, repo := newService(t, &fakePhotos{refuse: boom})

		if err := svc.Delete(ctx, uuid.New(), repo.m.ID); !errors.Is(err, boom) {
			t.Fatalf("err = %v, want the refusal", err)
		}
		if repo.deleted {
			t.Error("the memory went while its photo stayed")
		}
	})
}
