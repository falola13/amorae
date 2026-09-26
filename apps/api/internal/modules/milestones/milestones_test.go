package milestones

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidate(t *testing.T) {
	day := time.Date(2023, 9, 30, 0, 0, 0, 0, time.UTC)

	t.Run("it keeps what was written, trimmed", func(t *testing.T) {
		in, err := Validate(Input{Title: "  Our engagement  ", Date: day, Sub: " Three years ", Reminder: true})
		if err != nil {
			t.Fatalf("a good date was refused: %v", err)
		}
		if in.Title != "Our engagement" || in.Sub != "Three years" {
			t.Errorf("title = %q, sub = %q", in.Title, in.Sub)
		}
	})

	t.Run("a date needs a name", func(t *testing.T) {
		if _, err := Validate(Input{Title: "   ", Date: day}); err == nil {
			t.Error("a nameless date was kept")
		}
	})

	t.Run("and a day", func(t *testing.T) {
		if _, err := Validate(Input{Title: "Our engagement"}); err == nil {
			t.Error("a date with no day was kept")
		}
	})

	t.Run("both are bounded", func(t *testing.T) {
		long := strings.Repeat("a", maxTitleRunes+1)
		if _, err := Validate(Input{Title: long, Date: day}); err == nil {
			t.Error("an oversized title was kept")
		}
		if _, err := Validate(Input{Title: "Fine", Date: day, Sub: long}); err == nil {
			t.Error("an oversized note was kept")
		}
	})

	t.Run("length is counted in characters, not bytes", func(t *testing.T) {
		// Eighty é is eighty characters and a hundred and sixty bytes.
		if _, err := Validate(Input{Title: strings.Repeat("é", maxTitleRunes), Date: day}); err != nil {
			t.Errorf("a title of exactly the limit was refused: %v", err)
		}
	})
}

func TestDerivedID_IsDeterministicAndDistinct(t *testing.T) {
	coupleID := uuid.New()
	userA, userB := uuid.New(), uuid.New()

	if AnniversaryID(coupleID) != AnniversaryID(coupleID) {
		t.Error("AnniversaryID was not deterministic")
	}
	if BirthdayID(coupleID, userA) != BirthdayID(coupleID, userA) {
		t.Error("BirthdayID was not deterministic")
	}
	if AnniversaryID(coupleID) == BirthdayID(coupleID, userA) {
		t.Error("an anniversary and a birthday collided")
	}
	if BirthdayID(coupleID, userA) == BirthdayID(coupleID, userB) {
		t.Error("two different members' birthdays collided")
	}
	if AnniversaryID(coupleID) == AnniversaryID(uuid.New()) {
		t.Error("two different couples' anniversaries collided")
	}
}
