package couples

import (
	"context"
	"log/slog"
	"time"
)

// The retention window is measured in days, so checking hourly is as precise
// as it needs to be, and a restart costs at most one missed tick.
const purgeInterval = time.Hour

type purgeRepository interface {
	PurgeDissolvedBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

// Purger deletes the shared content of couples whose retention window has
// closed (FR-PAIR-008.AC1). It is the half of "leave" that nobody triggers:
// the promise that a shared history does not outlive the relationship by
// more than the window.
//
// It runs in the API process because there is one API instance today
// (Q-13) and no worker yet (Q-16). When cmd/worker arrives this type moves
// there unchanged — it takes a repository and a clock and nothing else.
type Purger struct {
	repo  purgeRepository
	now   func() time.Time
	every time.Duration
	log   *slog.Logger
}

func NewPurger(repo purgeRepository, now func() time.Time, log *slog.Logger) *Purger {
	return &Purger{repo: repo, now: now, every: purgeInterval, log: log}
}

// Run sweeps once at start — so a process that was down past a window's end
// catches up immediately — and then on every tick until ctx is canceled.
//
// A failed sweep is logged, not returned: the couple content still there is
// a reason to try again in an hour, never a reason to take the API down.
func (p *Purger) Run(ctx context.Context) error {
	ticker := time.NewTicker(p.every)
	defer ticker.Stop()

	for {
		p.sweep(ctx)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (p *Purger) sweep(ctx context.Context) {
	cutoff := p.now().Add(-RetentionWindow)
	purged, err := p.repo.PurgeDissolvedBefore(ctx, cutoff)
	if err != nil {
		if ctx.Err() != nil {
			return // shutting down
		}
		p.log.Error("purging ended couples", "error", err)
		return
	}
	if purged > 0 {
		p.log.Info("purged ended couples", "count", purged, "dissolved_before", cutoff)
	}
}
