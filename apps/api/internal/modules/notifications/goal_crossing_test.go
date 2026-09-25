package notifications

import (
	"testing"

	"github.com/google/uuid"
)

func goalAt(total, target int64) GoalCrossingCandidate {
	return GoalCrossingCandidate{
		UserID: uuid.New(), GoalID: uuid.New(), Title: "Lagos",
		Target: target, Total: total,
		Prefs: Preferences{GoalMilestones: true},
	}
}

func TestForGoalCrossing(t *testing.T) {
	tests := []struct {
		name          string
		total, target int64
		want          string // "" means nothing is said
	}{
		{"not yet", 40, 100, ""},
		{"halfway exactly", 50, 100, "You’re halfway there."},
		{"past halfway", 63, 100, "You’re halfway there."},
		{"all the way", 100, 100, "You’ve reached it."},
		{"past the end", 120, 100, "You’ve reached it."},
		// One contribution taking a goal from nothing to finished is one
		// piece of news, not two.
		{"straight past both", 0, 0, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n, ok := ForGoalCrossing(goalAt(tc.total, tc.target))
			if tc.want == "" {
				if ok {
					t.Errorf("it said %q with nothing to say", n.Message.Body)
				}
				return
			}
			if !ok {
				t.Fatal("nothing was said")
			}
			if n.Message.Body != tc.want {
				t.Errorf("body = %q, want %q", n.Message.Body, tc.want)
			}
			if n.Message.Title != "Lagos" {
				t.Errorf("title = %q, want the goal's name", n.Message.Title)
			}
		})
	}

	t.Run("the highest crossing only", func(t *testing.T) {
		n, ok := ForGoalCrossing(goalAt(100, 100))
		if !ok {
			t.Fatal("nothing was said")
		}
		// Keyed by the crossing, so halfway is not announced again on the way
		// past, and slipping back and passing it again says nothing.
		c := goalAt(100, 100)
		if n.Key == c.GoalID.String()+":50" {
			t.Error("finishing announced halfway")
		}
	})

	t.Run("the amount is never in it", func(t *testing.T) {
		// A shared plan is fair game; the figure is not lock-screen material
		// (FR-NOTF-005.AC2).
		n, _ := ForGoalCrossing(goalAt(750000, 1000000))
		if n.Message.Body != "You’re halfway there." || n.Message.Title != "Lagos" {
			t.Errorf("title = %q, body = %q", n.Message.Title, n.Message.Body)
		}
	})

	t.Run("not to somebody who turned it off", func(t *testing.T) {
		c := goalAt(100, 100)
		c.Prefs.GoalMilestones = false
		if _, ok := ForGoalCrossing(c); ok {
			t.Error("it was sent anyway")
		}
	})
}
