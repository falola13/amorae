package challenges

import (
	"testing"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

func TestTemplates_AreUsable(t *testing.T) {
	all := Templates()
	if len(all) == 0 {
		t.Fatal("there are no challenges to start")
	}

	seen := map[string]bool{}
	for _, tpl := range all {
		if seen[tpl.Key] {
			t.Errorf("two templates share the key %q", tpl.Key)
		}
		seen[tpl.Key] = true

		if tpl.Title == "" || tpl.Blurb == "" {
			t.Errorf("%s has no title or blurb", tpl.Key)
		}
		if len(tpl.Prompts) == 0 {
			t.Errorf("%s has no days", tpl.Key)
		}
		for i, prompt := range tpl.Prompts {
			if prompt == "" {
				t.Errorf("%s day %d has no prompt", tpl.Key, i+1)
			}
		}
	}
}

func TestTemplateByKey(t *testing.T) {
	first := Templates()[0]
	got, err := TemplateByKey(first.Key)
	if err != nil {
		t.Fatalf("TemplateByKey(%q): %v", first.Key, err)
	}
	if got.Title != first.Title {
		t.Errorf("got %q, want %q", got.Title, first.Title)
	}

	if _, err := TemplateByKey("something-we-never-wrote"); err == nil {
		t.Error("an unknown key was accepted")
	}
}

func TestValidateMark(t *testing.T) {
	yes, no := true, false

	tests := []struct {
		name       string
		done       *bool
		skipped    *bool
		wantMark   Mark
		wantMarked bool
		wantErr    bool
	}{
		{"done", &yes, nil, MarkDone, true, false},
		{"skipped", nil, &yes, MarkSkipped, true, false},
		// Explicit false means un-marking, not a third kind of request.
		{"taking it back", &no, nil, "", false, false},
		{"taking a skip back", nil, &no, "", false, false},
		{"saying nothing at all", nil, nil, "", false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mark, marked, err := ValidateMark(tc.done, tc.skipped)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil {
				return
			}
			if mark != tc.wantMark || marked != tc.wantMarked {
				t.Errorf("mark = %q, marked = %v", mark, marked)
			}
		})
	}

	t.Run("done wins if a client sends both", func(t *testing.T) {
		// Not an error: both true is read kindly as "done".
		mark, marked, err := ValidateMark(&yes, &yes)
		if err != nil || !marked || mark != MarkDone {
			t.Errorf("mark = %q, marked = %v, err = %v", mark, marked, err)
		}
	})
}

func TestDay_MarkFor_SeparatesThePartners(t *testing.T) {
	ada, ben := uuid.New(), uuid.New()
	day := Day{Marks: map[uuid.UUID]Mark{ada: MarkDone}}

	if mark, ok := day.MarkFor(ada); !ok || mark != MarkDone {
		t.Errorf("Ada: mark = %q, ok = %v", mark, ok)
	}

	// DEC-30: one partner's mark says nothing about the other (not yet != skipped).
	if mark, ok := day.MarkFor(ben); ok {
		t.Errorf("Ben was marked %q by somebody else", mark)
	}
}

func TestTemplates_TheLibrary(t *testing.T) {
	days := map[string]int{
		"seven-days-of-noticing":           7,
		"seven-days-of-praying-together":   7,
		"fourteen-days-of-small-things":    14,
		"seven-days-of-gratitude":          7,
		"seven-days-of-serving-each-other": 7,
		"fourteen-days-in-the-psalms":      14,
		"ten-conversations":                10,
		"advent":                           24,
		"lent":                             40,
	}
	for key, want := range days {
		tpl, err := TemplateByKey(key)
		if err != nil {
			t.Errorf("%s: %v", key, err)
			continue
		}
		if len(tpl.Prompts) != want {
			t.Errorf("%s has %d days, want %d", key, len(tpl.Prompts), want)
		}
	}
	if len(Templates()) != len(days) {
		t.Errorf("%d templates, want %d", len(Templates()), len(days))
	}

	for _, tpl := range Templates() {
		switch tpl.Category {
		case CategoryConnection, CategoryFaith, CategoryService, CategorySeason:
		default:
			t.Errorf("%s has category %q", tpl.Key, tpl.Category)
		}
		switch tpl.Season {
		case "", SeasonAdvent, SeasonLent:
		default:
			t.Errorf("%s has season %q", tpl.Key, tpl.Season)
		}
		if (tpl.Season != "") != (tpl.Category == CategorySeason) {
			t.Errorf("%s: a season and the season category go together", tpl.Key)
		}
	}
	if a, _ := TemplateByKey("advent"); a.Season != SeasonAdvent {
		t.Errorf("advent season = %q", a.Season)
	}
	if l, _ := TemplateByKey("lent"); l.Season != SeasonLent {
		t.Errorf("lent season = %q", l.Season)
	}
}

func TestValidateCustom(t *testing.T) {
	repeat := func(s string, n int) string {
		out := ""
		for i := 0; i < n; i++ {
			out += s
		}
		return out
	}
	many := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = "Something kind."
		}
		return out
	}
	tests := []struct {
		name      string
		in        Custom
		wantField string
	}{
		{"fine", Custom{Title: "Our month", Prompts: many(3)}, ""},
		{"forty days", Custom{Title: "Our month", Prompts: many(40)}, ""},
		{"a name of exactly eighty", Custom{Title: repeat("é", 80), Prompts: many(3)}, ""},
		{"no name", Custom{Title: "   ", Prompts: many(3)}, "custom.title"},
		{"a name too long", Custom{Title: repeat("é", 81), Prompts: many(3)}, "custom.title"},
		{"too few days", Custom{Title: "x", Prompts: many(2)}, "custom.prompts"},
		{"too many days", Custom{Title: "x", Prompts: many(MaxCustomDays + 1)}, "custom.prompts"},
		{"a hundred, the most", Custom{Title: "x", Prompts: many(MaxCustomDays)}, ""},
		{"an empty day", Custom{Title: "x", Prompts: []string{"a", "  ", "c"}}, "custom.prompts"},
		{"a day too long", Custom{Title: "x", Prompts: []string{"a", repeat("é", 201), "c"}}, "custom.prompts"},
		{"a day of exactly two hundred", Custom{Title: "x", Prompts: []string{"a", repeat("é", 200), "c"}}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tpl, err := ValidateCustom(tc.in)
			if tc.wantField == "" {
				if err != nil || tpl.Key != CustomKey {
					t.Fatalf("tpl = %+v, err = %v", tpl, err)
				}
				return
			}
			ae, ok := apperr.As(err)
			if !ok || ae.Fields[tc.wantField] == "" {
				t.Fatalf("err = %v, want a field error on %s", err, tc.wantField)
			}
		})
	}
}

func TestValidateEntry(t *testing.T) {
	yes, no := true, false
	note := func(s string) *string { return &s }

	t.Run("a note on its own leaves the mark alone", func(t *testing.T) {
		e, err := ValidateEntry(nil, nil, note("  Lovely.  "))
		if err != nil || e.SetMark || e.ClearMark || e.Note == nil || *e.Note != "Lovely." {
			t.Fatalf("e = %+v, err = %v", e, err)
		}
	})
	t.Run("a mark with a note", func(t *testing.T) {
		e, err := ValidateEntry(&yes, nil, note("Good."))
		if err != nil || !e.SetMark || e.Mark != MarkDone || e.Note == nil {
			t.Fatalf("e = %+v, err = %v", e, err)
		}
	})
	t.Run("taking a mark back", func(t *testing.T) {
		e, err := ValidateEntry(&no, nil, nil)
		if err != nil || !e.ClearMark || e.SetMark {
			t.Fatalf("e = %+v, err = %v", e, err)
		}
	})
	t.Run("saying nothing is still an error", func(t *testing.T) {
		if _, err := ValidateEntry(nil, nil, nil); err == nil {
			t.Fatal("an empty request was accepted")
		}
	})
	t.Run("two hundred and eighty characters, and one more", func(t *testing.T) {
		ok := ""
		for i := 0; i < MaxNoteRunes; i++ {
			ok += "é"
		}
		if _, err := ValidateEntry(nil, nil, &ok); err != nil {
			t.Errorf("exactly %d runes: %v", MaxNoteRunes, err)
		}
		tooLong := ok + "é"
		_, err := ValidateEntry(nil, nil, &tooLong)
		if ae, _ := apperr.As(err); ae == nil || ae.Fields["note"] != "Keep it under 280 characters." {
			t.Errorf("err = %v, want the note field error", err)
		}
	})
	t.Run("an empty note clears it", func(t *testing.T) {
		e, err := ValidateEntry(nil, nil, note(""))
		if err != nil || e.Note == nil || *e.Note != "" {
			t.Fatalf("e = %+v, err = %v", e, err)
		}
	})
}
