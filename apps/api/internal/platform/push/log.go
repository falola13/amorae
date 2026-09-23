package push

import (
	"context"
	"log/slog"
)

// Log is the development Sender. It writes what would have been sent and
// never leaves the machine, so local work and CI need no VAPID keys.
//
// It logs the title and the path, never the body: the body is the part most
// likely to carry something about a person, and a development log is not a
// place to start putting that.
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
