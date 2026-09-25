package push

import "testing"

func TestVapidSubscriber(t *testing.T) {
	tests := map[string]string{
		"mailto:someone@example.com": "someone@example.com",
		"someone@example.com":        "someone@example.com",
		"https://amorae.example/me":  "https://amorae.example/me",
		"":                           "",
	}
	for in, want := range tests {
		if got := vapidSubscriber(in); got != want {
			t.Errorf("vapidSubscriber(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestVapidSubscriber_NeverDoublesThePrefix(t *testing.T) {
	// Whatever we return, the library prefixing it must give exactly one "mailto:".
	got := vapidSubscriber("mailto:someone@example.com")
	rebuilt := "mailto:" + got
	if rebuilt != "mailto:someone@example.com" {
		t.Errorf("the library would send sub=%q", rebuilt)
	}
}
