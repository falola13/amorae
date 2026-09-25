package main

import (
	"strings"
	"testing"
)

func TestSchemaNotice(t *testing.T) {
	for _, tc := range []struct {
		name           string
		migrateOnStart bool
		production     bool
		wantWarn       bool
		wantMentions   string
	}{
		{
			name:           "managed here, so say so quietly",
			migrateOnStart: true,
			production:     true,
			wantWarn:       false,
			wantMentions:   "before this process serves",
		},
		{
			name:         "production with the flag off is the one to shout about",
			production:   true,
			wantWarn:     true,
			wantMentions: "MIGRATE_ON_START",
		},
		{
			name:         "a laptop running migrate by hand is not a problem",
			wantWarn:     false,
			wantMentions: "cmd/migrate",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			warn, msg := schemaNotice(tc.migrateOnStart, tc.production)
			if warn != tc.wantWarn {
				t.Errorf("warn = %v, want %v — for %q", warn, tc.wantWarn, msg)
			}
			if !strings.Contains(msg, tc.wantMentions) {
				t.Errorf("msg = %q, should mention %q", msg, tc.wantMentions)
			}
		})
	}
}

// The warning must name the variable, not just say something's wrong.
func TestSchemaNotice_WarningNamesTheVariable(t *testing.T) {
	warn, msg := schemaNotice(false, true)
	if !warn {
		t.Fatal("production without MIGRATE_ON_START did not warn")
	}
	for _, want := range []string{"MIGRATE_ON_START", `"true"`} {
		if !strings.Contains(msg, want) {
			t.Errorf("warning %q does not contain %q", msg, want)
		}
	}
}
