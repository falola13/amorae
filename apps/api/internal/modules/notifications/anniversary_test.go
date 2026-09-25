package notifications

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func memoryOn(date string, title string) MemoryAnniversaryCandidate {
	d, err := time.Parse(time.DateOnly, date)
	if err != nil {
		panic(err)
	}
	return MemoryAnniversaryCandidate{
		UserID: uuid.New(), MemoryID: uuid.New(), Title: title,
		Timezone: "UTC", Date: d, Prefs: Preferences{Memories: true},
	}
}

func TestForMemoriesOnThisDay(t *testing.T) {
	morning := at("09:00") // 2026-09-25

	t.Run("a year ago, named", func(t *testing.T) {
		n, ok := ForMemoriesOnThisDay(
			[]MemoryAnniversaryCandidate{memoryOn("2025-09-25", "The night it rained")}, morning)
		if !ok {
			t.Fatal("nothing was offered back")
		}
		if n.Message.Title != "A year ago today" {
			t.Errorf("title = %q", n.Message.Title)
		}
		if n.Message.Body != "The night it rained" {
			t.Errorf("body = %q", n.Message.Body)
		}
		// Keyed by the day, so however many fall on it, it arrives once.
		if n.Key != "2026-09-25" {
			t.Errorf("key = %q", n.Key)
		}
	})

	t.Run("more than one is one notification", func(t *testing.T) {
		one := memoryOn("2024-09-25", "The long walk")
		two := memoryOn("2025-09-25", "The night it rained")
		two.UserID = one.UserID
		n, ok := ForMemoriesOnThisDay([]MemoryAnniversaryCandidate{two, one}, morning)
		if !ok {
			t.Fatal("nothing was offered back")
		}
		// The oldest is the one worth naming, whatever order they arrive in.
		if n.Message.Title != "2 years ago today" {
			t.Errorf("title = %q", n.Message.Title)
		}
		if n.Message.Body != "The long walk, and 1 more." {
			t.Errorf("body = %q", n.Message.Body)
		}
	})

	t.Run("a different day is not this day", func(t *testing.T) {
		if _, ok := ForMemoriesOnThisDay(
			[]MemoryAnniversaryCandidate{memoryOn("2025-09-24", "Yesterday, last year")}, morning); ok {
			t.Error("a memory from the 24th was offered on the 25th")
		}
	})

	t.Run("not in the small hours", func(t *testing.T) {
		if _, ok := ForMemoriesOnThisDay(
			[]MemoryAnniversaryCandidate{memoryOn("2025-09-25", "The night it rained")},
			at("03:00")); ok {
			t.Error("it arrived before anybody was awake")
		}
	})

	t.Run("not to somebody who turned it off", func(t *testing.T) {
		c := memoryOn("2025-09-25", "The night it rained")
		c.Prefs.Memories = false
		if _, ok := ForMemoriesOnThisDay([]MemoryAnniversaryCandidate{c}, morning); ok {
			t.Error("it was sent anyway")
		}
	})

	t.Run("nothing kept on this day says nothing", func(t *testing.T) {
		if _, ok := ForMemoriesOnThisDay(nil, morning); ok {
			t.Error("it spoke with nothing to say")
		}
	})
}

func TestYearsAgo(t *testing.T) {
	// A memory from earlier the same year would read as "0 years ago", which
	// nobody says; the query only returns earlier years, and this is the
	// belt to that pair of braces.
	for years, want := range map[int]string{
		0: "A year ago today",
		1: "A year ago today",
		2: "2 years ago today",
		7: "7 years ago today",
	} {
		if got := yearsAgo(years); got != want {
			t.Errorf("yearsAgo(%d) = %q, want %q", years, got, want)
		}
	}
}
