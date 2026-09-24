package app

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/couples"
	"github.com/falola13/amorae/apps/api/internal/modules/prayers"
	"github.com/falola13/amorae/apps/api/internal/modules/user"
)

// prayersCouples adapts the couples service to the one question prayers asks
// of it. It lives here, in the composition root, because that is the only
// place allowed to know both modules exist: prayers does not import couples,
// couples does not import prayers, and neither has to change when the other
// does.
type prayersCouples struct {
	couples *couples.Service
}

func (a prayersCouples) ForPrayers(ctx context.Context, userID uuid.UUID) (prayers.CoupleContext, error) {
	mine, err := a.couples.GetMine(ctx, userID)
	if err != nil {
		return prayers.CoupleContext{}, err
	}

	// The couple's zone, not either partner's: it decides when the week turns
	// over, so both must agree on it (DEC-27). A zone the database somehow
	// holds but this machine cannot load falls back to UTC rather than taking
	// the feature down.
	timezone := mine.Couple.Timezone
	if timezone == "" {
		timezone = user.DefaultTimezone
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		location = time.UTC
	}

	members := make([]prayers.Member, 0, len(mine.Members))
	for _, m := range mine.Members {
		members = append(members, prayers.Member{UserID: m.ID, JoinedAt: m.JoinedAt})
	}

	return prayers.CoupleContext{
		CoupleID: mine.Couple.ID,
		Location: location,
		Members:  members,
	}, nil
}

// CoupleFor is the smaller question the Together modules ask: which couple,
// and nothing else. Same adapter, because the answer comes from the same
// place and neither module should learn what a couples.Mine is.
func (a prayersCouples) CoupleFor(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	mine, err := a.couples.GetMine(ctx, userID)
	if err != nil {
		return uuid.UUID{}, err
	}
	return mine.Couple.ID, nil
}

// PartnerOf is the other member of the caller's couple, or the zero id while
// they are still waiting for one.
func (a prayersCouples) PartnerOf(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	mine, err := a.couples.GetMine(ctx, userID)
	if err != nil {
		return uuid.UUID{}, err
	}
	for _, m := range mine.Members {
		if m.ID != userID {
			return m.ID, nil
		}
	}
	return uuid.UUID{}, nil
}
