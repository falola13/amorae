package couples

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/user"
)

// Stored as 6 characters; shown as ABC-123.
func formatInviteCode(code string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(code) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if len(s) > 6 {
		s = s[:6]
	}
	if len(s) <= 3 {
		return s
	}
	return s[:3] + "-" + s[3:]
}

type personDTO struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Timezone    string    `json:"timezone"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Role        string    `json:"role"`
}

type partnerDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type onboardingDTO struct {
	Couple        bool `json:"couple"`
	Install       bool `json:"install"`
	Notifications bool `json:"notifications"`
}

// MineDTO matches the Couple interface in apps/web/src/lib/api/types.ts.
type MineDTO struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Me         personDTO   `json:"me"`
	Partner    *partnerDTO `json:"partner"`
	InviteCode string      `json:"invite_code"`
	StartedOn  *string     `json:"started_on,omitempty"`
	// Distinct from me.timezone, which is when this person's reminders fire (DEC-27).
	Timezone   string        `json:"timezone"`
	Onboarding onboardingDTO `json:"onboarding"`
}

// EndedCoupleDTO is not a Couple: no invite code, no onboarding, nothing to act on.
type EndedCoupleDTO struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	People        []partnerDTO `json:"people"`
	StartedOn     *string      `json:"started_on,omitempty"`
	DissolvedAt   time.Time    `json:"dissolved_at"`
	ReadOnlyUntil time.Time    `json:"read_only_until"`
}

// RelationshipStartDate arrives as "2006-01-02"; encoding/json only accepts
// RFC 3339, so the service parses it.
type UpdateDto struct {
	RelationshipStartDate *string `json:"relationship_start_date"`
	Name                  *string `json:"name"`
	// The couple's zone, not either person's (DEC-27).
	Timezone *string `json:"timezone"`
}

// Couple is accepted but not stored — membership makes that step complete.
type OnboardingDto struct {
	Couple        *bool `json:"couple"`
	Install       *bool `json:"install"`
	Notifications *bool `json:"notifications"`
}

// The partner's role is read from mine, not passed in, so it can't be mismatched to the wrong partner.
func ToMineDTO(mine Mine, me user.User, partner *user.User) MineDTO {
	timezone := me.Timezone
	if timezone == "" {
		timezone = user.DefaultTimezone
	}

	out := MineDTO{
		ID:       mine.Couple.ID.String(),
		Name:     mine.Couple.Name,
		Timezone: mine.Couple.Timezone,
		Me: personDTO{
			ID:          me.ID.String(),
			Email:       me.Email,
			DisplayName: me.DisplayName,
			Timezone:    timezone,
			CreatedAt:   me.CreatedAt,
			UpdatedAt:   me.UpdatedAt,
		},
		InviteCode: formatInviteCode(mine.InviteCode),
		Onboarding: onboardingDTO{Couple: true},
	}
	if member, ok := mine.Member(me.ID); ok {
		out.Me.Role = member.Role
		out.Onboarding.Install = member.Onboarding.Install
		out.Onboarding.Notifications = member.Onboarding.Notifications
	}

	if !mine.Couple.RelationshipStartDate.IsZero() {
		day := mine.Couple.RelationshipStartDate.Format("2006-01-02")
		out.StartedOn = &day
	}
	if partner != nil {
		out.Partner = &partnerDTO{ID: partner.ID.String(), DisplayName: partner.DisplayName}
		if member, ok := mine.Member(partner.ID); ok {
			out.Partner.Role = member.Role
		}
	}
	return out
}

// ToEndedDTO leaves out a member whose account has since gone, rather than showing a blank person.
func ToEndedDTO(mine Mine, names map[uuid.UUID]string) EndedCoupleDTO {
	out := EndedCoupleDTO{
		ID:     mine.Couple.ID.String(),
		Name:   mine.Couple.Name,
		People: make([]partnerDTO, 0, len(mine.Members)),
	}
	if mine.Couple.DissolvedAt != nil {
		out.DissolvedAt = *mine.Couple.DissolvedAt
		out.ReadOnlyUntil = PurgeDueAt(*mine.Couple.DissolvedAt)
	}
	if !mine.Couple.RelationshipStartDate.IsZero() {
		day := mine.Couple.RelationshipStartDate.Format("2006-01-02")
		out.StartedOn = &day
	}
	for _, m := range mine.Members {
		name, ok := names[m.ID]
		if !ok {
			continue
		}
		out.People = append(out.People, partnerDTO{ID: m.ID.String(), DisplayName: name, Role: m.Role})
	}
	return out
}
