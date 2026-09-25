package couples

import (
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

// Ending a couple ends it for both partners (FR-PAIR-008, Q-09 option (a)),
// followed by a retention window during which it's read-only before the
// shared content is deleted.

// RetentionWindow is how long a couple stays readable after it ends.
const RetentionWindow = 30 * 24 * time.Hour

var (
	// ErrDissolved refuses a write to an ended couple. Stands ready for
	// modules that scope by couple id directly, though nothing in this
	// module can currently reach it (writes resolve the live membership).
	ErrDissolved = apperr.Conflict(
		"couple_dissolved",
		"This space has ended. You can still read and download what you shared, but not add to it.",
	)
)

func PurgeDueAt(dissolvedAt time.Time) time.Time {
	return dissolvedAt.Add(RetentionWindow)
}

// CheckWritable: nil means live; any end date closes it, even past its
// retention window if the sweeper hasn't reached the row yet.
func CheckWritable(dissolvedAt *time.Time) error {
	if dissolvedAt != nil {
		return ErrDissolved
	}
	return nil
}

func DuePurge(dissolvedAt []time.Time, now time.Time) []time.Time {
	var due []time.Time
	for _, at := range dissolvedAt {
		if !now.Before(PurgeDueAt(at)) {
			due = append(due, at)
		}
	}
	return due
}
