package mailer

import (
	"context"
	"log/slog"
)

// Log is the development Mailer. It writes the message and never hits the
// network, so local and CI work with no API key.
type Log struct {
	log *slog.Logger
}

func NewLog(log *slog.Logger) *Log {
	return &Log{log: log}
}

func (m *Log) Send(_ context.Context, to, subject, body string) error {
	m.log.Info("mail not sent (log mailer)", "to", to, "subject", subject, "body", body)
	return nil
}
