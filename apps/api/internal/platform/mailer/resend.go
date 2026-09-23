package mailer

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v4"
)

type Resend struct {
	client *resend.Client
	from   string
}

func NewResend(apiKey string, defaultFrom string) *Resend {
	return &Resend{
		client: resend.NewClient(apiKey),
		from:   defaultFrom,
	}
}

func (m *Resend) Send(ctx context.Context, to, subject, body string) error {
	_, err := m.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    m.from,
		To:      []string{to},
		Subject: subject,
		Html:    body,
	})
	if err != nil {
		return fmt.Errorf("sending email via resend: %w", err)
	}
	return nil
}
