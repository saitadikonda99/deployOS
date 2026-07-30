// Package runtime provides the process-lifecycle plumbing shared by every
// DeployOS server binary: running an HTTP server until its context is
// canceled, then shutting it down gracefully.
package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// DefaultShutdownTimeout bounds how long Run waits for in-flight requests
// to finish once shutdown begins.
const DefaultShutdownTimeout = 10 * time.Second

// HTTPServer runs an http.Server and shuts it down gracefully when its
// context is canceled.
type HTTPServer struct {
	srv             *http.Server
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

// NewHTTPServer builds an HTTPServer listening on addr and serving handler.
func NewHTTPServer(addr string, handler http.Handler, logger *slog.Logger) *HTTPServer {
	return &HTTPServer{
		srv: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
		},
		logger:          logger,
		shutdownTimeout: DefaultShutdownTimeout,
	}
}

// Run starts serving and blocks until ctx is canceled, at which point it
// shuts the server down gracefully and returns. A non-nil error indicates
// the server failed to start or failed to shut down cleanly.
func (s *HTTPServer) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("http server starting", slog.String("addr", s.srv.Addr))
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server: %w", err)
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	s.logger.Info("http server shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	if err := s.srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http server shutdown: %w", err)
	}

	return <-errCh
}
