package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
)

// GRPCServer runs a *grpc.Server and shuts it down gracefully when its
// context is canceled, mirroring HTTPServer's lifecycle so both server
// binaries can be run and shut down the same way.
type GRPCServer struct {
	srv             *grpc.Server
	addr            string
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

// NewGRPCServer builds a GRPCServer that will listen on addr and serve
// srv. Register services on srv before passing it in.
func NewGRPCServer(addr string, srv *grpc.Server, logger *slog.Logger) *GRPCServer {
	return &GRPCServer{
		srv:             srv,
		addr:            addr,
		logger:          logger,
		shutdownTimeout: DefaultShutdownTimeout,
	}
}

// Run starts serving and blocks until ctx is canceled, at which point it
// shuts the server down gracefully (waiting for in-flight RPCs and
// streams to finish, up to shutdownTimeout, before forcing a stop) and
// returns. A non-nil error indicates the server failed to start.
func (s *GRPCServer) Run(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("grpc server: listening on %s: %w", s.addr, err)
	}

	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("grpc server starting", slog.String("addr", s.addr))
		errCh <- s.srv.Serve(lis)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	s.logger.Info("grpc server shutting down")

	stopped := make(chan struct{})
	go func() {
		s.srv.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(s.shutdownTimeout):
		s.srv.Stop()
	}

	return <-errCh
}
