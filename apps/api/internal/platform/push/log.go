package push

import (
	"context"
	"log/slog"
)

// Log is the development Sender: writes what would have been sent, never
// leaves the machine. Logs title and path, never body, which is more likely
// to carry something personal.
type Log struct {
	log *slog.Logger
}

func NewLog(log *slog.Logger) *Log {
	return &Log{log: log}
}

func (s *Log) Send(_ context.Context, device Device, message Message) error {
	s.log.Info("push not sent (log sender)",
		"endpoint", device.Endpoint, "title", message.Title, "path", message.Path)
	return nil
}
