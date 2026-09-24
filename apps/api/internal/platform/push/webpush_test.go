package push

import "testing"

func TestVapidSubscriber(t *testing.T) {
	// webpush-go prepends "mailto:" to anything that is not an https URL, so
	// the value RFC 8292 asks for is the one value it corrupts. Apple checks
	// the claim and answers 403; Google does not look, which is why this
	// survived until a real iPhone was pointed at it.
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
	// What the library will build from our value, spelled out: whatever we
	// return, prefixing it must give exactly one "mailto:".
	got := vapidSubscriber("mailto:someone@example.com")
	rebuilt := "mailto:" + got
	if rebuilt != "mailto:someone@example.com" {
		t.Errorf("the library would send sub=%q", rebuilt)
	}
}
