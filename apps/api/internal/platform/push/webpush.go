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
// VAPID signature, both via library rather than hand-rolled.
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
		// 24h: a stale weekly notification is worse than a dropped one.
		ttl:    int((24 * time.Hour).Seconds()),
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// vapidSubscriber adapts our RFC 8292 "sub" value (a bare mailto:/https:
// URL) to what webpush-go expects: the library itself prepends "mailto:" to
// anything not an https URL, so passing it our already-prefixed value would
// double it into "mailto:mailto:...", a claim Apple's push service rejects
// with 403 (Google doesn't validate it, so this only shows up there).
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
		// Subscription is dead (unsubscribed/reinstalled/cleared); never worth retrying.
		return ErrGone
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return nil
	default:
		return fmt.Errorf("push service answered %s", resp.Status)
	}
}
