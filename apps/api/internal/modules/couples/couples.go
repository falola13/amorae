package couples

import (
	"errors"
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
	ErrInviteInvalid = inviteError("invite_invalid", "This invite is not valid.")
	ErrInviteExpired = inviteError("invite_expired", "This invite has expired.")
	ErrInviteUsed    = inviteError("invite_used", "This invite has already been used.")
	ErrInviteRevoked = inviteError("invite_revoked", "This invite is no longer valid.")
	ErrCoupleFull    = apperr.Conflict("couple_full", "This couple already has two members.")
	ErrAlreadyPaired = apperr.Conflict("already_paired", "You are already in a couple.")
	ErrNotFound      = apperr.NotFound("couple_not_found", "You are not in a couple yet.")

	// errCodeTaken signals a drawn invite code collision; the service draws
	// another. Never reaches a client.
	errCodeTaken = errors.New("invite code taken")
)

// inviteError repeats the message under fields.code so the join form shows
// it beneath the code input like any other field error.
func inviteError(code, message string) *apperr.Error {
	err := apperr.Invalid(code, message)
	err.Fields = map[string]string{"code": message}
	return err
}

// ValidateRole grants no permissions from the label itself, so only length is bounded.
func ValidateRole(role string) (string, error) {
	role = strings.TrimSpace(role)
	if n := utf8.RuneCountInString(role); n < 1 || n > maxRoleRunes {
		return "", apperr.Validation(map[string]string{
			"role": fmt.Sprintf("Must be between 1 and %d characters.", maxRoleRunes),
		})
	}
	return role, nil
}

// Onboarding has no field for the couple step: membership itself is that step's completed state.
type Onboarding struct {
	Install       bool
	Notifications bool
}

type Member struct {
	ID   uuid.UUID
	Role string
	// Fixes the order of anything that alternates between the two members
	// (e.g. the prayer setter rotation) — not inferable from row arrival order.
	JoinedAt   time.Time
	Onboarding Onboarding
}

// Mine is the signed-in couple view.
type Mine struct {
	Couple     COUPLES
	Members    []Member
	InviteCode string
}

// Member separates "me" from "my partner" in the join-ordered slice.
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
	// IANA zone name; decides when a prayer week turns over. The couple's zone, not either phone's.
	Timezone              string    `json:"timezone"`
	RelationshipStartDate time.Time `json:"relationship_start_date"`
	CreatedBy             uuid.UUID `json:"created_by"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
	// Nil while live. Once set, the couple is read-only until the retention
	// window runs out and it is deleted. See dissolution.go.
	DissolvedAt *time.Time `json:"dissolved_at,omitempty"`
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
