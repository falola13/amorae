package couples

import (
	"fmt"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/google/uuid"
)

var (
	ErrInviteInvalid = apperr.Invalid("invite_invalid", "This invite is not valid.")
	ErrInviteExpired = apperr.Invalid("invite_expired", "This invite has expired.")
	ErrInviteUsed    = apperr.Invalid("invite_used", "This invite has already been used.")
	ErrInviteRevoked = apperr.Invalid("invite_revoked", "This invite is no longer valid.")
	ErrCoupleFull    = apperr.Conflict("couple_full", "This couple already has two members.")
	ErrAlreadyPaired = apperr.Conflict("already_paired", "You are already in a couple.")
)

type COUPLES struct {
	ID                    uuid.UUID  `json:"id"`
	Name                  string     `json:"name"`
	Timezone              *time.Time `json:"timezone"`
	RelationshipStartDate time.Time  `json:"relationship_start_date"`
	CreatedBy             uuid.UUID  `json:"created_by"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func New(name string, created_by uuid.UUID, relationship_start_date *time.Time, now time.Time) (COUPLES, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return COUPLES{}, apperr.Internal(fmt.Errorf("generating couple id: %w", err))
	}
	c := COUPLES{
		ID:        id,
		Name:      name,
		CreatedBy: created_by,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if relationship_start_date != nil {
		c.RelationshipStartDate = *relationship_start_date
	}
	return c, nil
}
