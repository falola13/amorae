// Package user owns the User entity and its validation rules; net/http and
// pgx concerns live in handler.go and repository_postgres.go.
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

// Sentinel errors checked by identity (errors.Is/apperr.As) so the HTTP
// status is decided once, here.
var (
	ErrNotFound   = apperr.NotFound("user_not_found", "User not found.")
	ErrEmailTaken = apperr.Conflict("email_taken", "An account with this email already exists.")
)

const maxDisplayNameRunes = 50

// PasswordHash carries json:"-" as defense in depth in case a future change
// marshals User directly instead of via DTO.
type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	DisplayName  string     `json:"display_name"`
	PasswordHash string     `json:"-"`
	Timezone     string     `json:"timezone"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	// BirthMonth and BirthDay travel together: both set or both nil (see
	// the users_birthday_month_day_together constraint). BirthYear is
	// optional even then — plenty of people would rather not say.
	BirthMonth *int `json:"birth_month"`
	BirthDay   *int `json:"birth_day"`
	BirthYear  *int `json:"birth_year"`
}

// Birthday is a validated month/day, with an optional year.
type Birthday struct {
	Month int
	Day   int
	Year  *int
}

// minBirthYear bounds a birth year the same way the far end of a lifetime
// does; ValidateBirthday's future check handles the near end.
const minBirthYear = 1900

// ValidateBirthday checks month and day form a real calendar date, and that
// a given year is plausible and not in the future. Year 2000 (a leap year)
// stands in when none is given, so a Feb 29 birthday is never refused for
// lacking one — the placeholder is a stand-in for validation only, never
// stored as if it meant something.
func ValidateBirthday(month, day int, year *int, now time.Time) (Birthday, error) {
	calendarYear := 2000
	if year != nil {
		calendarYear = *year
	}
	date := time.Date(calendarYear, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if int(date.Month()) != month || date.Day() != day {
		return Birthday{}, apperr.Validation(map[string]string{"birthday": "That isn’t a date."})
	}
	if year != nil {
		if *year < minBirthYear {
			return Birthday{}, apperr.Validation(map[string]string{"birthday": "That isn’t a date."})
		}
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		if date.After(today) {
			return Birthday{}, apperr.Validation(map[string]string{"birthday": "That’s in the future."})
		}
	}
	return Birthday{Month: month, Day: day, Year: year}, nil
}

// DefaultTimezone matches the users.timezone column default.
const DefaultTimezone = "UTC"

// New validates email and displayName; passwordHash is taken as-is (hashing is auth's job).
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

// ValidateTimezone accepts an IANA zone name ("Africa/Lagos"); the embedded
// tzdata means this works even without zone data on the host (e.g. distroless).
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

// validEmail also rejects "Name <a@b.com>" forms that mail.ParseAddress
// otherwise accepts, by requiring the parsed address equal the input exactly.
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

// ValidateDisplayName is exposed so UpdateProfile can re-run just this rule without going through New.
func ValidateDisplayName(displayName string) (string, error) {
	displayName = strings.TrimSpace(displayName)
	if n := utf8.RuneCountInString(displayName); n < 1 || n > maxDisplayNameRunes {
		return "", apperr.Validation(map[string]string{
			"display_name": fmt.Sprintf("Must be between 1 and %d characters.", maxDisplayNameRunes),
		})
	}
	return displayName, nil
}
