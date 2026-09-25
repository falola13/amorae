package couples

import (
	"context"
	"log/slog"
	"time"
)

const purgeInterval = time.Hour

type purgeRepository interface {
	PurgeDissolvedBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

// Purger deletes the shared content of couples whose retention window has
// closed (FR-PAIR-008.AC1). Runs in the API process since there's one
// instance today and no worker yet (Q-13, Q-16).
type Purger struct {
	repo  purgeRepository
	now   func() time.Time
	every time.Duration
	log   *slog.Logger
}

func NewPurger(repo purgeRepository, now func() time.Time, log *slog.Logger) *Purger {
	return &Purger{repo: repo, now: now, every: purgeInterval, log: log}
}

// Run sweeps once at start (catches up after downtime), then on every tick
// until ctx is canceled. A failed sweep is logged, not returned — never a
// reason to take the API down.
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
