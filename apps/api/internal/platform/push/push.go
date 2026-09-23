// Package push sends a web push notification to one browser.
//
// Like platform/mailer, this is a seam: a development sender that writes to
// the log, and a real one that talks to the browser's push service. The
// worker takes the interface, so nothing above here knows what VAPID is.
package push

import (
	"context"
	"errors"
)

// Message is what a person sees. Deliberately small: a title, a line, and
// where tapping it goes.
//
// No private content belongs in any of these fields. They land on a lock
// screen, which is the one place in Amorae that is not private (FR-NOTF-005).
type Message struct {
	Title string
	Body  string
	// A path within the app, not a full URL — the service worker resolves it.
	Path string
	// Groups notifications so a second one replaces the first rather than
	// stacking. One per category is enough.
	Tag string
}

// Device is one browser's subscription.
type Device struct {
	Endpoint string
	P256dh   string
	Auth     string
}

// ErrGone means the push service says this subscription no longer exists, so
// the caller should forget it (FR-NOTF-004). It is the one error worth
// telling apart: every other failure is worth retrying, and this one never is.
var ErrGone = errors.New("push subscription is gone")

type Sender interface {
	Send(ctx context.Context, device Device, message Message) error
}
