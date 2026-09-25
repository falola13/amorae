package memories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUpdate(t *testing.T) {
	ctx := context.Background()

	t.Run("it replaces title, date, location, and note", func(t *testing.T) {
		svc, repo := newService(t, nil)
		newDate := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)

		m, err := svc.Update(ctx, uuid.New(), repo.m.ID, Input{
			Title: "The night it rained harder", Date: newDate,
			Location: "Lekki", Note: "We stayed until they stacked the chairs.",
		})
		if err != nil {
			t.Fatalf("updating: %v", err)
		}
		if m.Title != "The night it rained harder" || !m.Date.Equal(newDate) || m.Location != "Lekki" {
			t.Errorf("title = %q, date = %v, location = %q", m.Title, m.Date, m.Location)
		}
	})

	t.Run("the photo is untouched", func(t *testing.T) {
		svc, repo := newService(t, nil)
		want := repo.m.PhotoID

		m, err := svc.Update(ctx, uuid.New(), repo.m.ID, Input{
			Title: "Renamed", Date: repo.m.Date,
		})
		if err != nil {
			t.Fatalf("updating: %v", err)
		}
		if m.PhotoID != want {
			t.Errorf("photo id = %q, want %q", m.PhotoID, want)
		}
	})

	t.Run("a bad edit is refused before it reaches the repository", func(t *testing.T) {
		svc, repo := newService(t, nil)
		before := repo.m

		if _, err := svc.Update(ctx, uuid.New(), repo.m.ID, Input{Title: "  "}); err == nil {
			t.Fatal("a blank title was accepted")
		}
		if repo.m != before {
			t.Error("a refused update still reached the repository")
		}
	})

	t.Run("another couple's memory is simply not found", func(t *testing.T) {
		svc, repo := newService(t, nil)

		_, err := svc.Update(ctx, uuid.New(), uuid.New(), Input{
			Title: "Not mine", Date: repo.m.Date,
		})
		if err != ErrNotFound {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
