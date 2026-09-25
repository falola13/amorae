package goals

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

func ptr[T any](v T) *T { return &v }

func fieldsOf(t *testing.T, err error) map[string]string {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	appErr, ok := err.(*apperr.Error)
	if !ok {
		t.Fatalf("err = %T, want *apperr.Error", err)
	}
	return appErr.Fields
}

func TestTotal_IsTheSumOfWhatWasLogged(t *testing.T) {
	g := Goal{Progress: []Progress{{Amount: 50000}, {Amount: 25000}, {Amount: 10000}}}
	if got := g.Total(); got != 85000 {
		t.Errorf("Total() = %d, want 85000", got)
	}

	t.Run("a correction brings it back down", func(t *testing.T) {
		g.Progress = append(g.Progress, Progress{Amount: -10000})
		if got := g.Total(); got != 75000 {
			t.Errorf("Total() = %d, want 75000", got)
		}
	})

	t.Run("nothing logged is nothing", func(t *testing.T) {
		if got := (Goal{}).Total(); got != 0 {
			t.Errorf("Total() = %d, want 0", got)
		}
	})

	t.Run("the total is the couple's, not split between them", func(t *testing.T) {
		ada, ben := uuid.New(), uuid.New()
		shared := Goal{Progress: []Progress{
			{UserID: ada, Amount: 40000},
			{UserID: ben, Amount: 60000},
		}}
		if got := shared.Total(); got != 100000 {
			t.Errorf("Total() = %d, want 100000 — one total, not two", got)
		}
	})
}

func TestValidate_Creating(t *testing.T) {
	full := Input{
		Title:     ptr("A deposit"),
		Target:    ptr(int64(2500000)),
		Unit:      ptr("naira"),
		StartDate: ptr("2026-10-01"),
		EndDate:   ptr("2027-09-30"),
	}

	t.Run("a complete goal", func(t *testing.T) {
		g, err := (Goal{}).Validate(full, true)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if g.Title != "A deposit" || g.Target != 2500000 || g.Unit != UnitNaira {
			t.Errorf("goal = %+v", g)
		}
		if g.Done {
			t.Error("a new goal started done")
		}
	})

	t.Run("everything missing is reported at once", func(t *testing.T) {
		fields := fieldsOf(t, func() error { _, err := (Goal{}).Validate(Input{}, true); return err }())
		for _, want := range []string{"title", "target", "unit", "start_date", "end_date"} {
			if fields[want] == "" {
				t.Errorf("%s was not reported: %v", want, fields)
			}
		}
	})

	t.Run("a target of zero or less", func(t *testing.T) {
		for _, bad := range []int64{0, -1} {
			in := full
			in.Target = ptr(bad)
			if _, err := (Goal{}).Validate(in, true); err == nil {
				t.Errorf("Validate accepted target %d", bad)
			}
		}
	})

	t.Run("a unit that is neither", func(t *testing.T) {
		in := full
		in.Unit = ptr("dollars")
		if _, err := (Goal{}).Validate(in, true); err == nil {
			t.Error("Validate accepted an unknown unit")
		}
	})

	t.Run("ending before it starts", func(t *testing.T) {
		in := full
		in.EndDate = ptr("2026-09-30")
		fields := fieldsOf(t, func() error { _, err := (Goal{}).Validate(in, true); return err }())
		if fields["end_date"] == "" {
			t.Errorf("fields = %v, want end_date", fields)
		}
	})
}

func TestValidate_Editing(t *testing.T) {
	existing := Goal{
		Title:     "A deposit",
		Why:       "Somewhere of our own",
		Target:    2500000,
		Unit:      UnitNaira,
		StartDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2027, 9, 30, 0, 0, 0, 0, time.UTC),
	}

	t.Run("a field nobody mentioned is left alone", func(t *testing.T) {
		g, err := existing.Validate(Input{Target: ptr(int64(3000000))}, false)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if g.Target != 3000000 {
			t.Errorf("target = %d", g.Target)
		}
		if g.Title != existing.Title || g.Why != existing.Why || g.Unit != existing.Unit {
			t.Error("a field that was not mentioned changed")
		}
		if !g.StartDate.Equal(existing.StartDate) || !g.EndDate.Equal(existing.EndDate) {
			t.Error("the dates changed without being mentioned")
		}
	})

	t.Run("marking it done changes nothing else", func(t *testing.T) {
		g, err := existing.Validate(Input{Done: ptr(true)}, false)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if !g.Done || g.Title != existing.Title || g.Target != existing.Target {
			t.Errorf("goal = %+v", g)
		}
	})

	t.Run("an edit does not have to carry the required fields", func(t *testing.T) {
		if _, err := existing.Validate(Input{Why: ptr("For the two of us")}, false); err != nil {
			t.Errorf("Validate: %v", err)
		}
	})
}

func TestValidateAmount(t *testing.T) {
	if err := ValidateAmount(0); err == nil {
		t.Error("zero was accepted, but it records nothing")
	}
	if err := ValidateAmount(5000); err != nil {
		t.Errorf("ValidateAmount(5000) = %v", err)
	}
	if err := ValidateAmount(-5000); err != nil {
		t.Errorf("a correction was refused: %v", err)
	}
}
