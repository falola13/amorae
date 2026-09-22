package couples

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/google/uuid"
)

const maxRoleRunes = 32

var (
	ErrInviteInvalid = apperr.Invalid("invite_invalid", "This invite is not valid.")
	ErrInviteExpired = apperr.Invalid("invite_expired", "This invite has expired.")
	ErrInviteUsed    = apperr.Invalid("invite_used", "This invite has already been used.")
	ErrInviteRevoked = apperr.Invalid("invite_revoked", "This invite is no longer valid.")
	ErrCoupleFull    = apperr.Conflict("couple_full", "This couple already has two members.")
	ErrAlreadyPaired = apperr.Conflict("already_paired", "You are already in a couple.")
	ErrNotFound      = apperr.NotFound("couple_not_found", "You are not in a couple yet.")
)

// A role is how a member labels themselves in the couple ("Husband", "Wife",
// whatever they prefer) and grants no permissions, so the wording is left to
// the couple and only the length is bounded.
func ValidateRole(role string) (string, error) {
	role = strings.TrimSpace(role)
	if n := utf8.RuneCountInString(role); n < 1 || n > maxRoleRunes {
		return "", apperr.Validation(map[string]string{
			"role": fmt.Sprintf("Must be between 1 and %d characters.", maxRoleRunes),
		})
	}
	return role, nil
}

// Onboarding is the per-person tail of the signup flow. There is no field for
// the couple step because membership itself is that step's completed state.
type Onboarding struct {
	Install       bool
	Notifications bool
}

type Member struct {
	ID         uuid.UUID
	Role       string
	Onboarding Onboarding
}

// Mine is the signed-in couple view: the couple row, both memberships, and
// the latest invite code.
type Mine struct {
	Couple     COUPLES
	Members    []Member
	InviteCode string
}

// Member finds one membership by user id. Callers use it to separate "me"
// from "my partner", since the slice holds both in join order.
func (m Mine) Member(userID uuid.UUID) (Member, bool) {
	for _, member := range m.Members {
		if member.ID == userID {
			return member, true
		}
	}
	return Member{}, false
}

type COUPLES struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	// An IANA zone name, matching the TEXT column. It decides when a prayer
	// week turns over, so it is the couple's zone, not either phone's.
	Timezone              string    `json:"timezone"`
	RelationshipStartDate time.Time `json:"relationship_start_date"`
	CreatedBy             uuid.UUID `json:"created_by"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func New(name string, created_by uuid.UUID, timezone string, relationship_start_date *time.Time, now time.Time) (COUPLES, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return COUPLES{}, apperr.Internal(fmt.Errorf("generating couple id: %w", err))
	}
	timezone, err = user.ValidateTimezone(timezone)
	if err != nil {
		return COUPLES{}, err
	}
	c := COUPLES{
		ID:        id,
		Name:      name,
		Timezone:  timezone,
		CreatedBy: created_by,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if relationship_start_date != nil {
		c.RelationshipStartDate = *relationship_start_date
	}
	return c, nil
}
