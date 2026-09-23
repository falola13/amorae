package couples

import (
	"strings"
	"time"

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

// Flat, matching the Couple interface in apps/web/src/lib/api/types.ts and
// the mock adapter. Every couples endpoint answers with this one shape.
type MineDTO struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Me         personDTO     `json:"me"`
	Partner    *partnerDTO   `json:"partner"`
	InviteCode string        `json:"invite_code"`
	StartedOn  *string       `json:"started_on,omitempty"`
	Onboarding onboardingDTO `json:"onboarding"`
}

// The date arrives as "2006-01-02", which encoding/json cannot decode into a
// time.Time (it only accepts RFC 3339). The service parses it.
type UpdateDto struct {
	RelationshipStartDate *string `json:"relationship_start_date"`
	Name                  *string `json:"name"`
}

// Couple is accepted so the client can send the whole object back, but it is
// not stored — membership is what makes that step complete.
type OnboardingDto struct {
	Couple        *bool `json:"couple"`
	Install       *bool `json:"install"`
	Notifications *bool `json:"notifications"`
}

// The partner's role is read from mine rather than passed in, so there is no
// way to hand this function a partner and a role belonging to someone else.
func ToMineDTO(mine Mine, me user.User, partner *user.User) MineDTO {
	timezone := me.Timezone
	if timezone == "" {
		timezone = user.DefaultTimezone
	}

	out := MineDTO{
		ID:   mine.Couple.ID.String(),
		Name: mine.Couple.Name,
		Me: personDTO{
			ID:          me.ID.String(),
			Email:       me.Email,
			DisplayName: me.DisplayName,
			Timezone:    timezone,
			CreatedAt:   me.CreatedAt,
			UpdatedAt:   me.UpdatedAt,
		},
		InviteCode: formatInviteCode(mine.InviteCode),
		// Being in a couple is the couple step; the rest is per person.
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
