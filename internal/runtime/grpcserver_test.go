package runtime

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// reserveAddr binds a loopback port, immediately frees it, and returns
// its address - GRPCServer.Run() needs to be the one to bind it, so
// tests need a free address to hand it in advance.
func reserveAddr(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserving address: %v", err)
	}
	addr := lis.Addr().String()
	_ = lis.Close()
	return addr
}

func TestGRPCServerServesAndShutsDownGracefully(t *testing.T) {
	addr := reserveAddr(t)
	server := NewGRPCServer(addr, grpc.NewServer(), testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx) }()

	if !waitForListening(addr) {
		t.Fatal("grpc server never started listening")
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v, want nil after graceful shutdown", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not return after context cancellation")
	}
}

func TestGRPCServerReturnsErrorForBadAddress(t *testing.T) {
	server := NewGRPCServer("this-is-not-a-valid-address:::", grpc.NewServer(), testLogger())

	if err := server.Run(context.Background()); err == nil {
		t.Fatal("Run() error = nil, want an error for an invalid listen address")
	}
}

// waitForListening polls addr until a plain TCP connection succeeds,
// confirming the server has bound its listener.
func waitForListening(addr string) bool {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
