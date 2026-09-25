// Package push sends a web push notification to one browser, behind a
// Sender interface with a log-only dev implementation and a real one.
package push

import (
	"context"
	"errors"
)

// Message is what a person sees: a title, a line, and where tapping it goes.
// No private content belongs here — it lands on a lock screen (FR-NOTF-005).
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

// ErrGone means the push service says this subscription no longer exists;
// the caller should forget it (FR-NOTF-004) rather than retry.
var ErrGone = errors.New("push subscription is gone")

type Sender interface {
	Send(ctx context.Context, device Device, message Message) error
}
