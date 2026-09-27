package timeline

import (
	"testing"
	"time"
)

// fakePhotos stands in for the Cloudinary-backed store: URL just says what
// it was asked to resolve, without touching a network.
type fakePhotos struct{ called string }

func (f *fakePhotos) URL(publicID string, version int64) (string, error) {
	f.called = publicID
	return "https://example.com/" + publicID, nil
}

func TestService_PhotoURL(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("a memory with a photo resolves it", func(t *testing.T) {
		photos := &fakePhotos{}
		svc := NewService(nil, nil, photos, func() time.Time { return now })

		got := svc.PhotoURL(Item{Type: TypeMemory, PhotoID: "couples/x/y", Version: now})
		if got != "https://example.com/couples/x/y" {
			t.Errorf("got %q", got)
		}
		if photos.called != "couples/x/y" {
			t.Errorf("resolved %q, want the item's PhotoID", photos.called)
		}
	})

	t.Run("no PhotoID means no photo, without asking the store", func(t *testing.T) {
		photos := &fakePhotos{}
		svc := NewService(nil, nil, photos, func() time.Time { return now })

		if got := svc.PhotoURL(Item{Type: TypeMemory}); got != "" {
			t.Errorf("got %q, want empty", got)
		}
		if photos.called != "" {
			t.Error("the store was asked to resolve a photo that doesn't exist")
		}
	})

	t.Run("no Photos configured means no photo for anything", func(t *testing.T) {
		svc := NewService(nil, nil, nil, func() time.Time { return now })

		if got := svc.PhotoURL(Item{Type: TypeMemory, PhotoID: "couples/x/y", Version: now}); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}
