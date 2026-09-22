// Package user owns the User entity, its validation rules, and the use
// cases (via Service) that operate on it. Nothing in this package imports
// net/http or pgx — those belong to handler.go and repository_postgres.go
// respectively, keeping this file readable as "what is a valid user" alone.
package user

import (
	"fmt"
	"net/mail"
	"strings"
	"time"
	_ "time/tzdata" // embeds the IANA zone database for ValidateTimezone
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

// Sentinel errors every layer above the repository checks for by identity
// (errors.Is / apperr.As), so a 409 vs 404 is decided once, here, rather
// than re-derived from a raw driver error at each call site.
var (
	ErrNotFound   = apperr.NotFound("user_not_found", "User not found.")
	ErrEmailTaken = apperr.Conflict("email_taken", "An account with this email already exists.")
)

const maxDisplayNameRunes = 50

// User is never encoded directly into an HTTP response — handler.go maps it
// to a DTO instead — but PasswordHash still carries json:"-" as a second
// line of defense: if a future change ever marshals a User by mistake, the
// hash can't leak through it.
type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	DisplayName  string     `json:"display_name"`
	PasswordHash string     `json:"-"`
	Timezone     string     `json:"timezone"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at"`
}

// DefaultTimezone matches the users.timezone column default.
const DefaultTimezone = "UTC"

// New validates email and displayName and constructs a User ready to
// persist. passwordHash is taken as-is — hashing the plaintext password is
// auth's job, not user's, since user has no opinion on hashing algorithms.
func New(email, displayName, passwordHash string, now time.Time) (User, error) {
	email = NormalizeEmail(email)
	displayName = strings.TrimSpace(displayName)

	fields := map[string]string{}
	if !validEmail(email) {
		fields["email"] = "Enter a valid email address."
	}
	if n := utf8.RuneCountInString(displayName); n < 1 || n > maxDisplayNameRunes {
		fields["display_name"] = fmt.Sprintf("Must be between 1 and %d characters.", maxDisplayNameRunes)
	}
	if len(fields) > 0 {
		return User{}, apperr.Validation(fields)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return User{}, apperr.Internal(fmt.Errorf("generating user id: %w", err))
	}

	return User{
		ID:           id,
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: passwordHash,
		Timezone:     DefaultTimezone,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// ValidateEmail normalizes and checks an email the same way New does, for
// use cases that change only the email (auth.Service.ChangeEmail).
func ValidateEmail(email string) (string, error) {
	email = NormalizeEmail(email)
	if !validEmail(email) {
		return "", apperr.Validation(map[string]string{"email": "Enter a valid email address."})
	}
	return email, nil
}

// ValidateTimezone accepts an IANA zone name ("Africa/Lagos"). The zone
// database is embedded (the time/tzdata import), so validation doesn't depend
// on the host having one: the distroless runtime image, for example.
func ValidateTimezone(tz string) (string, error) {
	tz = strings.TrimSpace(tz)
	if _, err := time.LoadLocation(tz); err != nil || tz == "" || tz == "Local" {
		return "", apperr.Validation(map[string]string{"timezone": "Choose a timezone from the list."})
	}
	return tz, nil
}

// NormalizeEmail is exported so auth can normalize an email the same way
// before looking it up, without duplicating the rule.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// validEmail requires both a length under the common 254-byte mail limit
// and that mail.ParseAddress reads the string back as exactly the bare
// address given — that second check is what rejects "Name <a@b.com>" and
// similar forms ParseAddress otherwise accepts, since a display name has
// no meaning in this field.
func validEmail(email string) bool {
	if email == "" || len(email) > 254 {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	return addr.Address == email
}

// ValidateDisplayName is exposed so UpdateProfile can re-run just this rule
// without going through New (which also requires an email and a password
// hash that a profile update doesn't have).
func ValidateDisplayName(displayName string) (string, error) {
	displayName = strings.TrimSpace(displayName)
	if n := utf8.RuneCountInString(displayName); n < 1 || n > maxDisplayNameRunes {
		return "", apperr.Validation(map[string]string{
			"display_name": fmt.Sprintf("Must be between 1 and %d characters.", maxDisplayNameRunes),
		})
	}
	return displayName, nil
}
