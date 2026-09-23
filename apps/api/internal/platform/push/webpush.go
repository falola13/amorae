package push

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
		subject:    subject,
		// A notification about this week is worthless next week. If a phone
		// has been off for a day, let the push service drop it rather than
		// deliver something stale.
		ttl:    int((24 * time.Hour).Seconds()),
		client: &http.Client{Timeout: 10 * time.Second},
	}
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
