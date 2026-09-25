package challenges

import (
	"testing"

	"github.com/google/uuid"
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
