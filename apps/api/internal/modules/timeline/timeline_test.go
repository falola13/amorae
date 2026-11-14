package timeline

import (
	"testing"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

func TestValidateFilter(t *testing.T) {
	cases := map[string]Filter{
		"":          FilterAll,
		"all":       FilterAll,
		"prayer":    FilterPrayer,
		"memories":  FilterMemories,
		"journal":   FilterJournal,
		"notes":     FilterNotes,
		"questions": FilterQuestions,
		"moments":   FilterMoments, // old clients still send it
		"plans":     FilterPlans,
		"nonsense":  FilterAll,
		"Prayer":    FilterAll, // exact spelling only, same rule as journal.Validate
	}
	for raw, want := range cases {
		if got := ValidateFilter(raw); got != want {
			t.Errorf("ValidateFilter(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestTypesFor(t *testing.T) {
	all := TypesFor(FilterAll)
	if len(all) != 9 {
		t.Fatalf("all = %d types, want 9", len(all))
	}

	prayer := TypesFor(FilterPrayer)
	if len(prayer) != 2 || prayer[0] != TypePrayerWeek || prayer[1] != TypePrayerAnswered {
		t.Errorf("prayer = %v", prayer)
	}

	for f, want := range map[Filter]Type{
		FilterMemories: TypeMemory, FilterJournal: TypeJournal,
		FilterNotes: TypeAppreciation, FilterQuestions: TypeQuestion,
	} {
		if got := TypesFor(f); len(got) != 1 || got[0] != want {
			t.Errorf("%s = %v, want [%s]", f, got, want)
		}
	}

	moments := TypesFor(FilterMoments)
	if len(moments) != 5 {
		t.Errorf("moments = %v, want 5 types", moments)
	}

	plans := TypesFor(FilterPlans)
	if len(plans) != 3 || plans[0] != TypeEvent || plans[1] != TypeGoal || plans[2] != TypeChallenge {
		t.Errorf("plans = %v", plans)
	}
}

func TestValidateLimit(t *testing.T) {
	t.Run("absent defaults", func(t *testing.T) {
		n, err := ValidateLimit("")
		if err != nil || n != DefaultLimit {
			t.Errorf("n = %d, err = %v, want %d, nil", n, err, DefaultLimit)
		}
	})

	t.Run("clamped to the max", func(t *testing.T) {
		n, err := ValidateLimit("500")
		if err != nil || n != MaxLimit {
			t.Errorf("n = %d, err = %v, want %d, nil", n, err, MaxLimit)
		}
	})

	t.Run("clamped up from zero or negative", func(t *testing.T) {
		n, err := ValidateLimit("0")
		if err != nil || n != 1 {
			t.Errorf("n = %d, err = %v, want 1, nil", n, err)
		}
		n, err = ValidateLimit("-5")
		if err != nil || n != 1 {
			t.Errorf("n = %d, err = %v, want 1, nil", n, err)
		}
	})

	t.Run("not a number is a validation error, not a silent default", func(t *testing.T) {
		_, err := ValidateLimit("many")
		if _, ok := apperr.As(err); !ok {
			t.Fatalf("err = %v, want a validation error", err)
		}
	})
}

func TestParseCursor(t *testing.T) {
	t.Run("absent means no cursor", func(t *testing.T) {
		got, err := ParseCursor("")
		if err != nil || got != nil {
			t.Errorf("got = %v, err = %v, want nil, nil", got, err)
		}
	})

	t.Run("an RFC3339 instant round-trips", func(t *testing.T) {
		want := time.Date(2026, 3, 1, 12, 30, 0, 0, time.UTC)
		got, err := ParseCursor(want.Format(time.RFC3339))
		if err != nil {
			t.Fatalf("ParseCursor: %v", err)
		}
		if got == nil || !got.Equal(want) {
			t.Errorf("got = %v, want %v", got, want)
		}
	})

	t.Run("garbage is refused, not silently ignored", func(t *testing.T) {
		_, err := ParseCursor("not-a-timestamp")
		if err != ErrBadCursor {
			t.Fatalf("err = %v, want ErrBadCursor", err)
		}
	})
}

func TestParseMonth(t *testing.T) {
	for raw, want := range map[string]string{"": "", "2026-10": "2026-10", "1999-01": "1999-01"} {
		got, err := ParseMonth(raw)
		if err != nil || got != want {
			t.Errorf("ParseMonth(%q) = %q, %v", raw, got, err)
		}
	}
	for _, raw := range []string{"2026-13", "2026-00", "2026-1", "2026-10-01", "oct", "26-10", " 2026-10"} {
		if _, err := ParseMonth(raw); err != ErrBadMonth {
			t.Errorf("ParseMonth(%q) err = %v, want ErrBadMonth", raw, err)
		}
	}
}
