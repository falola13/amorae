package push

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// WebPush is the real sender: RFC 8291 payload encryption and an RFC 8292
// VAPID signature, both from a library rather than by hand. Encryption
// written here would be encryption nobody has reviewed, and its failure mode
// is silent — a notification that never arrives, or arrives unreadable, with
// nothing to say which.
type WebPush struct {
	publicKey  string
	privateKey string
	subject    string
	ttl        int
	client     *http.Client
}

func NewWebPush(publicKey, privateKey, subject string) *WebPush {
	return &WebPush{
		publicKey:  publicKey,
		privateKey: privateKey,
		subject:    vapidSubscriber(subject),
		// A notification about this week is worthless next week. If a phone
		// has been off for a day, let the push service drop it rather than
		// deliver something stale.
		ttl:    int((24 * time.Hour).Seconds()),
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// vapidSubscriber is the contact detail in the form webpush-go wants, which
// is not the form RFC 8292 wants.
//
// The JWT's "sub" claim must be a mailto: or https: URL, which is why the
// configuration insists on one. But the library prepends "mailto:" itself to
// anything that is not an https URL — so handing it the correct value
// produces "mailto:mailto:someone@example.com", and a claim no validator
// should accept.
//
// Google's push service does not look, so Chrome and Android worked
// perfectly. Apple does look, and answered 403 to every send, which is how
// this was found: one platform silently fine, one platform never once
// delivering. Passing the bare address lets the library build the URL it
// intends to.
func vapidSubscriber(subject string) string {
	if strings.HasPrefix(subject, "https:") {
		return subject
	}
	return strings.TrimPrefix(subject, "mailto:")
}

// payload is what the service worker receives.
type payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Path  string `json:"path"`
	Tag   string `json:"tag"`
}

func (s *WebPush) Send(ctx context.Context, device Device, message Message) error {
	body, err := json.Marshal(payload{
		Title: message.Title,
		Body:  message.Body,
		Path:  message.Path,
		Tag:   message.Tag,
	})
	if err != nil {
		return fmt.Errorf("encoding push payload: %w", err)
	}

	resp, err := webpush.SendNotificationWithContext(ctx, body, &webpush.Subscription{
		Endpoint: device.Endpoint,
		Keys: webpush.Keys{
			P256dh: device.P256dh,
			Auth:   device.Auth,
		},
	}, &webpush.Options{
		HTTPClient:      s.client,
		Subscriber:      s.subject,
		VAPIDPublicKey:  s.publicKey,
		VAPIDPrivateKey: s.privateKey,
		TTL:             s.ttl,
		Urgency:         webpush.UrgencyNormal,
	})
	if err != nil {
		return fmt.Errorf("sending push: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		// The browser has unsubscribed, been reinstalled, or cleared its
		// data. Retrying will never work, so say so plainly and let the
		// caller delete it.
		return ErrGone
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return nil
	default:
		return fmt.Errorf("push service answered %s", resp.Status)
	}
}
