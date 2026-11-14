package notifications_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/notifications"
	"github.com/falola13/amorae/apps/api/internal/platform/database/dbtest"
)

// An ask reaches the one who was asked, with an undo window; a prayer
// reaches the one who asked, credited to the one who prayed.
func TestPostgresRepository_RecentlyWritten_PrayerRequests(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo := notifications.NewPostgresRepository(db)
	coupleID, a, b := pair(t, db)
	now := time.Now().UTC().Truncate(time.Microsecond)

	id := uuid.New()
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO prayer_requests (id, couple_id, author_id, asked_on, text, created_at, updated_at)
		VALUES ($1, $2, $3, ($4 AT TIME ZONE 'UTC')::date, 'The interview', $4, $4)
	`, id, coupleID, a, now); err != nil {
		t.Fatalf("insert request: %v", err)
	}

	got := writtenFor(t, repo, id)
	if len(got) != 1 {
		t.Fatalf("candidates = %d, want just the partner", len(got))
	}
	c := got[0]
	if c.UserID != b || c.AuthorID != a || c.Kind != notifications.KindPrayerRequest ||
		c.Settles != time.Minute || !c.Prefs.PrayerRequests {
		t.Errorf("ask candidate = %+v", c)
	}

	if _, err := db.Q(ctx).Exec(ctx, `
		UPDATE prayer_requests SET prayed_by = $2, prayed_at = $3 WHERE id = $1
	`, id, b, now); err != nil {
		t.Fatalf("mark prayed: %v", err)
	}
	var prayed *notifications.WrittenCandidate
	for _, c := range writtenFor(t, repo, id) {
		if c.Kind == notifications.KindPrayedForYou {
			c := c
			prayed = &c
		}
	}
	if prayed == nil || prayed.UserID != a || prayed.AuthorID != b || prayed.AuthorName != "Bo" || prayed.Settles != 0 {
		t.Errorf("prayed candidate = %+v", prayed)
	}
}

// An answer is announced to the partner only while they have not answered
// that day themselves.
func TestPostgresRepository_RecentlyWritten_QuestionAnswered(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo := notifications.NewPostgresRepository(db)
	coupleID, a, b := pair(t, db)
	now := time.Now().UTC().Truncate(time.Microsecond)

	insert := func(author uuid.UUID) uuid.UUID {
		t.Helper()
		id := uuid.New()
		if _, err := db.Q(ctx).Exec(ctx, `
			INSERT INTO daily_answers (id, couple_id, author_id, asked_on, question, text, created_at, updated_at)
			VALUES ($1, $2, $3, ($4 AT TIME ZONE 'UTC')::date, 'A question?', 'secret', $4, $4)
		`, id, coupleID, author, now); err != nil {
			t.Fatalf("insert answer: %v", err)
		}
		return id
	}

	first := insert(a)
	got := writtenFor(t, repo, first)
	if len(got) != 1 || got[0].UserID != b || got[0].AuthorID != a ||
		got[0].Kind != notifications.KindQuestionAnswered || !got[0].Prefs.DailyQuestion {
		t.Fatalf("first answer candidates = %+v", got)
	}

	second := insert(b)
	if got := writtenFor(t, repo, second); len(got) != 0 {
		t.Errorf("second answer was announced to the one who had already answered: %+v", got)
	}
	if got := writtenFor(t, repo, first); len(got) != 0 {
		t.Errorf("once both have answered, the first is no longer news: %+v", got)
	}
}
