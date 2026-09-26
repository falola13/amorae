package notifications

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestForBothPrayed(t *testing.T) {
	week := uuid.New()
	base := BothPrayedCandidate{
		UserID: uuid.New(), WeekID: week,
		Date:   time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC),
		Points: 2,
		Prefs:  Preferences{Together: true},
	}

	t.Run("both finishing today is announced, and says today rather than the week", func(t *testing.T) {
		n, ok := ForBothPrayed(base)
		if !ok {
			t.Fatal("nothing was said")
		}
		if n.Message.Body != "You’ve both prayed everything today." {
			t.Errorf("body = %q", n.Message.Body)
		}
		if n.Key != week.String()+":2026-09-23" {
			t.Errorf("key = %q, want the week and the day", n.Key)
		}
	})

	t.Run("a different day is a different key, so tomorrow can fire too", func(t *testing.T) {
		tomorrow := base
		tomorrow.Date = base.Date.AddDate(0, 0, 1)
		n, _ := ForBothPrayed(tomorrow)
		if n.Key == "" || n.Key == week.String()+":2026-09-23" {
			t.Errorf("key = %q, did not move to the new day", n.Key)
		}
	})

	t.Run("together off silences it", func(t *testing.T) {
		off := base
		off.Prefs.Together = false
		if _, ok := ForBothPrayed(off); ok {
			t.Error("it was said despite together being off")
		}
	})
}
