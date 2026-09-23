package couples

import (
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

// Ending a couple (FR-PAIR-008, Q-09 option (a)).
//
// Leaving does not remove the leaver and leave the other partner holding
// everything: it ends the couple for both. Neither partner can keep or lock
// the other out of a history they both wrote. What follows is a window in
// which the couple is frozen — readable and exportable, closed to writes —
// after which the shared content is deleted.
//
// The rules live here as plain functions of their arguments, so the window
// can be tested without a database or a clock.

// RetentionWindow is how long a couple stays readable after it ends. Long
// enough to take a copy of what you wrote, short enough that a shared
// history doesn't outlive the relationship by much.
const RetentionWindow = 30 * 24 * time.Hour

var (
	// ErrDissolved refuses a write to a couple that has ended. Nothing in the
	// couples module can reach it — a write resolves the caller's *live*
	// membership, and ending a couple ends both — so it stands for the
	// modules that will scope by couple id directly.
	ErrDissolved = apperr.Conflict(
		"couple_dissolved",
		"This space has ended. You can still read and download what you shared, but not add to it.",
	)
)

// PurgeDueAt is the moment a couple that ended at dissolvedAt stops being
// readable and its shared content is deleted.
func PurgeDueAt(dissolvedAt time.Time) time.Time {
	return dissolvedAt.Add(RetentionWindow)
}

// CheckWritable reports whether a couple in this state still accepts writes.
// nil means it is live; any end date at all closes it, including one whose
// window has since run out but whose row the sweeper hasn't reached yet.
func CheckWritable(dissolvedAt *time.Time) error {
	if dissolvedAt != nil {
		return ErrDissolved
	}
	return nil
}

// DuePurge lists which of these couples have outlived their window by now.
// It takes the whole set rather than one date so the decision — and its
// boundary — is testable in one place.
func DuePurge(dissolvedAt []time.Time, now time.Time) []time.Time {
	var due []time.Time
	for _, at := range dissolvedAt {
		if !now.Before(PurgeDueAt(at)) {
			due = append(due, at)
		}
	}
	return due
}
