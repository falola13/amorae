package user

import (
	"strings"
	"testing"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

func TestNormalizeEmail(t *testing.T) {
	cases := map[string]string{
		"  A@B.com  ": "a@b.com",
		"Foo@Bar.COM": "foo@bar.com",
		"a@b.com":     "a@b.com",
	}
	for in, want := range cases {
		if got := NormalizeEmail(in); got != want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidEmail(t *testing.T) {
	cases := []struct {
		email string
		valid bool
	}{
		{"a@b.com", true},
		{"first.last+tag@example.co.uk", true},
		{"", false},
		{"not-an-email", false},
		{"missing-at.com", false},
		{"Name <a@b.com>", false}, // display-name form is rejected; this field is a bare address
		{"a@" + strings.Repeat("b", 250) + ".com", false},
	}

	for _, tc := range cases {
		if got := validEmail(tc.email); got != tc.valid {
			t.Errorf("validEmail(%q) = %v, want %v", tc.email, got, tc.valid)
		}
	}
}

func TestNew_ValidInput(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	u, err := New(" A@B.com ", "  Ada Lovelace  ", "hashed", now)
	if err != nil {
		t.Fatalf("New() returned an error for valid input: %v", err)
	}

	if u.Email != "a@b.com" {
		t.Errorf("Email = %q, want normalized a@b.com", u.Email)
	}
	if u.DisplayName != "Ada Lovelace" {
		t.Errorf("DisplayName = %q, want trimmed Ada Lovelace", u.DisplayName)
	}
	if u.ID.String() == "00000000-0000-0000-0000-000000000000" {
		t.Error("New() did not assign an id")
	}
	if !u.CreatedAt.Equal(now) || !u.UpdatedAt.Equal(now) {
		t.Errorf("CreatedAt/UpdatedAt = %v/%v, want both %v", u.CreatedAt, u.UpdatedAt, now)
	}
}

func TestNew_ReturnsAllFieldErrorsAtOnce(t *testing.T) {
	_, err := New("not-an-email", "", "hashed", time.Now())

	appErr, ok := apperr.As(err)
	if !ok {
		t.Fatalf("New() returned a non-apperr error: %v", err)
	}
	if appErr.Code != "validation_failed" {
		t.Errorf("Code = %q, want validation_failed", appErr.Code)
	}
	if _, ok := appErr.Fields["email"]; !ok {
		t.Error("Fields is missing email")
	}
	if _, ok := appErr.Fields["display_name"]; !ok {
		t.Error("Fields is missing display_name")
	}
}

func TestValidateBirthday(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	year := func(y int) *int { return &y }

	t.Run("Feb 29 without a year is fine — 2000 stands in, and it's a leap year", func(t *testing.T) {
		got, err := ValidateBirthday(2, 29, nil, now)
		if err != nil {
			t.Fatalf("Feb 29 with no year was refused: %v", err)
		}
		if got.Month != 2 || got.Day != 29 || got.Year != nil {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("Feb 30 is never a date, with or without a year", func(t *testing.T) {
		if _, err := ValidateBirthday(2, 30, nil, now); err == nil {
			t.Error("Feb 30 with no year was accepted")
		}
		if _, err := ValidateBirthday(2, 30, year(2000), now); err == nil {
			t.Error("Feb 30 with a year was accepted")
		}
	})

	t.Run("a year in the future is refused", func(t *testing.T) {
		_, err := ValidateBirthday(6, 1, year(2027), now)
		f := fieldsOf(t, err)
		if f["birthday"] != "That’s in the future." {
			t.Errorf("fields = %v, want the future message under birthday", f)
		}
	})

	t.Run("this year, but the day hasn't happened yet, is also in the future", func(t *testing.T) {
		if _, err := ValidateBirthday(6, 2, year(2026), now); err == nil {
			t.Error("a birthday later this year was accepted")
		}
		if _, err := ValidateBirthday(6, 1, year(2026), now); err != nil {
			t.Errorf("today's own date this year was refused: %v", err)
		}
	})

	t.Run("a year before 1900 isn't a date", func(t *testing.T) {
		_, err := ValidateBirthday(6, 1, year(1899), now)
		f := fieldsOf(t, err)
		if f["birthday"] != "That isn’t a date." {
			t.Errorf("fields = %v, want the not-a-date message under birthday", f)
		}
	})

	t.Run("a real, past year is kept as given", func(t *testing.T) {
		got, err := ValidateBirthday(9, 30, year(1990), now)
		if err != nil {
			t.Fatalf("a good birthday was refused: %v", err)
		}
		if got.Year == nil || *got.Year != 1990 {
			t.Errorf("Year = %v, want 1990", got.Year)
		}
	})
}

func TestNew_DisplayNameBounds(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name    string
		display string
		wantErr bool
	}{
		{"one rune", "A", false},
		{"fifty runes", strings.Repeat("A", 50), false},
		{"fifty-one runes", strings.Repeat("A", 51), true},
		{"empty after trim", "   ", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New("a@b.com", tc.display, "hashed", now)
			if tc.wantErr && err == nil {
				t.Error("New() returned nil error, want a validation error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("New() returned an error for a valid display name: %v", err)
			}
		})
	}
}
