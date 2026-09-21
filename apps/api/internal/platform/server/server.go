// Package server wraps *http.Server with the timeouts and graceful-shutdown
// behavior every deployment of this API needs, so main.go doesn't have to
// get either right by hand.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// These timeouts protect the server from slow or stalled clients holding a
// connection open; they're fixed rather than configurable because there's
// no legitimate reason a deployment of this API would need different ones.
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
)

type Server struct {
	httpServer      *http.Server
	shutdownTimeout time.Duration
	log             *slog.Logger
}

func New(addr string, handler http.Handler, shutdownTimeout time.Duration, log *slog.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
		shutdownTimeout: shutdownTimeout,
		log:             log,
	}
}

// Run blocks until ctx is canceled — main.go cancels it on SIGINT/SIGTERM
// via signal.NotifyContext — then gives in-flight requests up to
// shutdownTimeout to finish before returning. The listener goroutine's
// result is always read back (even on the shutdown path), so Run never
// returns while that goroutine is still running.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		s.log.Info("http server listening", "addr", s.httpServer.Addr)
		err := s.httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listening: %w", err)
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	s.log.Info("shutting down http server", "timeout", s.shutdownTimeout)
	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutting down: %w", err)
	}

	return <-errCh
}
