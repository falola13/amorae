// Package timeline owns the couple's shared story: one read-only feed of
// what happened, assembled from seven other modules' own tables rather than
// stored anywhere itself. There is nothing to write here — a memory, a
// prayer answered, a finished goal each already belong to the module that
// keeps them; this package only ever reads.
package timeline

import (
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

// Type is which kind of thing an Item is, as the client's grouping and
// per-kind styling key off it.
type Type string

const (
	TypePrayerWeek     Type = "prayer_week"
	TypePrayerAnswered Type = "prayer_answered"
	TypeMemory         Type = "memory"
	TypeEvent          Type = "event"
	TypeGoal           Type = "goal"
	TypeJournal        Type = "journal"
	TypeAppreciation   Type = "appreciation"
)

// Filter is a client-facing grouping of Types — coarser than Type, and the
// only thing a caller ever chooses between.
type Filter string

const (
	FilterAll     Filter = "all"
	FilterPrayer  Filter = "prayer"
	FilterMoments Filter = "moments"
	FilterPlans   Filter = "plans"
)

// TypesFor is which concrete Types a Filter includes. all is every Type this
// package knows about — a new source added here only has to be added to one
// of these lists to show up in the right places.
func TypesFor(f Filter) []Type {
	switch f {
	case FilterPrayer:
		return []Type{TypePrayerWeek, TypePrayerAnswered}
	case FilterMoments:
		return []Type{TypeMemory, TypeJournal, TypeAppreciation}
	case FilterPlans:
		return []Type{TypeEvent, TypeGoal}
	default:
		return []Type{
			TypePrayerWeek, TypePrayerAnswered, TypeMemory, TypeEvent,
			TypeGoal, TypeJournal, TypeAppreciation,
		}
	}
}

// ValidateFilter defaults an empty, unknown, or stale filter value to "all"
// rather than erroring — a client that sends nothing, or sends a filter this
// version no longer knows, should see everything, not a 400.
func ValidateFilter(raw string) Filter {
	switch Filter(raw) {
	case FilterPrayer, FilterMoments, FilterPlans:
		return Filter(raw)
	default:
		return FilterAll
	}
}

const (
	DefaultLimit = 30
	MaxLimit     = 50
)

// ValidateLimit clamps to [1, MaxLimit]; an absent value is DefaultLimit, and
// a malformed one is a validation error rather than silently defaulted, so a
// client with a typo in its own query string finds out.
func ValidateLimit(raw string) (int, error) {
	if raw == "" {
		return DefaultLimit, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, apperr.Validation(map[string]string{"limit": "That's not a number."})
	}
	if n < 1 {
		n = 1
	}
	if n > MaxLimit {
		n = MaxLimit
	}
	return n, nil
}

// ErrBadCursor is ParseCursor's answer for a "before" that isn't a valid
// instant — a client's own bug, not this package's.
var ErrBadCursor = apperr.Validation(map[string]string{"before": "That's not a valid timestamp."})

// ParseCursor reads the "before" query parameter; an absent one means "no
// cursor", i.e. the most recent page.
func ParseCursor(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, ErrBadCursor
	}
	return &t, nil
}

// Item is one entry in the couple's timeline, already resolved from
// whichever source produced it — the timeline never knows about prayer
// weeks or memories as such once an Item exists, only these fields.
type Item struct {
	Type Type
	ID   uuid.UUID
	// At is the instant this item sorts and pages by.
	At time.Time
	// Date is At's couple-local calendar date, date-only — what the client
	// groups items by month with, computed once here so it never has to
	// reason about the couple's timezone itself.
	Date time.Time
	// Title, Sub and Path are already exactly what the client shows and
	// links to; nothing further is derived from them.
	Title string
	Sub   string
	Path  string
	// PhotoID is a memory's stored Cloudinary public id; empty for every
	// other Type, and for a memory with no photo. Never sent to a client as
	// such — Service.PhotoURL resolves it to a URL, or leaves it out.
	PhotoID string
	// Version is what PhotoURL's cache-busting reads off, mirroring
	// memories.Memory.UpdatedAt; the zero value for every Type but memory.
	Version time.Time
	// ActorID is who did this, when that's part of the story (answering a
	// prayer, writing a journal entry); the zero uuid when there is no
	// single actor (a memory, a finished goal, a whole prayer week).
	ActorID uuid.UUID
}
