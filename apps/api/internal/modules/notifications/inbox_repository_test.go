package notifications_test

import (
	"context"
	"testing"
	"time"

	"github.com/falola13/amorae/apps/api/internal/modules/notifications"
	"github.com/falola13/amorae/apps/api/internal/platform/database/dbtest"
)

// inboxRetention mirrors notifications.inboxRetention, which is unexported;
// kept in sync by hand since these tests live outside the package.
const inboxRetention = 30

func TestPostgresRepository_RecordInbox_ListsNewestFirst(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo := notifications.NewPostgresRepository(db)
	_, a, _ := pair(t, db)

	base := time.Now().UTC().Truncate(time.Microsecond)
	if err := repo.RecordInbox(ctx, a, "journal", "First", "one", "/together/journal", base); err != nil {
		t.Fatalf("RecordInbox: %v", err)
	}
	if err := repo.RecordInbox(ctx, a, "appreciation", "Second", "two", "/together/appreciation", base.Add(time.Second)); err != nil {
		t.Fatalf("RecordInbox: %v", err)
	}

	items, err := repo.Inbox(ctx, a)
	if err != nil {
		t.Fatalf("Inbox: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].Title != "Second" || items[1].Title != "First" {
		t.Errorf("order = %q, %q, want newest first", items[0].Title, items[1].Title)
	}
	for _, it := range items {
		if it.Read {
			t.Errorf("item %q came back read before anyone opened the inbox", it.Title)
		}
	}
}

func TestPostgresRepository_RecordInbox_PrunesToNewestThirty(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo := notifications.NewPostgresRepository(db)
	_, a, _ := pair(t, db)

	base := time.Now().UTC().Truncate(time.Microsecond)
	total := inboxRetention + 5
	for i := 0; i < total; i++ {
		at := base.Add(time.Duration(i) * time.Second)
		if err := repo.RecordInbox(ctx, a, "journal", "Entry", "", "/together/journal", at); err != nil {
			t.Fatalf("RecordInbox #%d: %v", i, err)
		}
	}

	items, err := repo.Inbox(ctx, a)
	if err != nil {
		t.Fatalf("Inbox: %v", err)
	}
	if len(items) != inboxRetention {
		t.Fatalf("got %d items, want the newest %d only", len(items), inboxRetention)
	}
	// The oldest surviving row should be the sixth one written (index 5),
	// since the five oldest of the total were pruned away.
	oldestKept := base.Add(5 * time.Second)
	if !items[len(items)-1].CreatedAt.Equal(oldestKept) {
		t.Errorf("oldest kept row = %v, want %v", items[len(items)-1].CreatedAt, oldestKept)
	}
}

func TestPostgresRepository_MarkInboxRead(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo := notifications.NewPostgresRepository(db)
	_, a, b := pair(t, db)

	now := time.Now().UTC().Truncate(time.Microsecond)
	if err := repo.RecordInbox(ctx, a, "journal", "For Ada", "", "/together/journal", now); err != nil {
		t.Fatalf("RecordInbox: %v", err)
	}
	if err := repo.RecordInbox(ctx, b, "journal", "For Bo", "", "/together/journal", now); err != nil {
		t.Fatalf("RecordInbox: %v", err)
	}

	if err := repo.MarkInboxRead(ctx, a, now.Add(time.Minute)); err != nil {
		t.Fatalf("MarkInboxRead: %v", err)
	}

	adasItems, err := repo.Inbox(ctx, a)
	if err != nil {
		t.Fatalf("Inbox(a): %v", err)
	}
	if len(adasItems) != 1 || !adasItems[0].Read {
		t.Errorf("Ada's row = %+v, want it marked read", adasItems)
	}

	bosItems, err := repo.Inbox(ctx, b)
	if err != nil {
		t.Fatalf("Inbox(b): %v", err)
	}
	if len(bosItems) != 1 || bosItems[0].Read {
		t.Error("marking Ada's inbox read also marked Bo's — reads are not scoped to the caller")
	}
}
