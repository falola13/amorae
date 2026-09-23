package couples

import (
	"errors"
	"testing"
	"time"
)

func TestPurgeDueAt_IsThirtyDaysAfterTheEnd(t *testing.T) {
	ended := time.Date(2026, 9, 23, 14, 30, 0, 0, time.UTC)
	want := time.Date(2026, 10, 23, 14, 30, 0, 0, time.UTC)
	if got := PurgeDueAt(ended); !got.Equal(want) {
		t.Errorf("PurgeDueAt() = %s, want %s", got, want)
	}
}

func TestCheckWritable(t *testing.T) {
	t.Run("a live couple accepts writes", func(t *testing.T) {
		if err := CheckWritable(nil); err != nil {
			t.Errorf("CheckWritable(nil) = %v, want nil", err)
		}
	})

	t.Run("an ended couple refuses them", func(t *testing.T) {
		ended := time.Now().UTC()
		if err := CheckWritable(&ended); !errors.Is(err, ErrDissolved) {
			t.Errorf("CheckWritable() = %v, want ErrDissolved", err)
		}
	})

	t.Run("a couple past its window still refuses, rather than reopening", func(t *testing.T) {
		// The sweeper may be an hour behind. Until the row is gone, the
		// couple must not quietly start accepting writes again.
		long := time.Now().UTC().Add(-2 * RetentionWindow)
		if err := CheckWritable(&long); !errors.Is(err, ErrDissolved) {
			t.Errorf("CheckWritable() = %v, want ErrDissolved", err)
		}
	})
}

func TestDuePurge_IncludesTheBoundary(t *testing.T) {
	ended := time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)
	due := PurgeDueAt(ended)

	tests := []struct {
		name string
		now  time.Time
		want int
	}{
		{"a moment before the window closes", due.Add(-time.Second), 0},
		{"the moment it closes", due, 1},
		{"after", due.Add(time.Hour), 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := DuePurge([]time.Time{ended}, tc.now); len(got) != tc.want {
				t.Errorf("DuePurge() returned %d, want %d", len(got), tc.want)
			}
		})
	}
}
