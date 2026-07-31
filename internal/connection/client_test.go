package connection

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"

	deployosv1 "github.com/saitadikonda99/deployOS/gen/go/deployos/v1"
	"github.com/saitadikonda99/deployOS/pkg/types"
)

// startTCPServer starts a real gRPC server for a Manager/verifier pair
// on an OS-assigned loopback port and returns its address. Client is
// designed to dial a real address (unlike Server, which is also
// exercised over bufconn in server_test.go), so its tests use real TCP.
func startTCPServer(t *testing.T, manager *Manager, verifier TokenVerifier) string {
	t.Helper()
	return startTCPServerWithServer(t, NewServer(manager, verifier, testLogger()))
}

// startTCPServerWithServer is like startTCPServer, but for tests that
// need to configure server (e.g. registering OnMessage) before it starts
// serving.
func startTCPServerWithServer(t *testing.T, server *Server) string {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listening: %v", err)
	}

	grpcSrv := grpc.NewServer()
	deployosv1.RegisterConnectionServiceServer(grpcSrv, server)

	go func() {
		_ = grpcSrv.Serve(lis)
	}()
	t.Cleanup(grpcSrv.Stop)

	return lis.Addr().String()
}

func testClient(addr string) *Client {
	c := NewClient(addr, testLogger())
	c.initialBackoff = 10 * time.Millisecond
	c.maxBackoff = 50 * time.Millisecond
	return c
}

func TestClientConnectsAndAuthenticates(t *testing.T) {
	manager := NewManager()
	verifier := newFakeVerifier()
	verifier.allow("good-token", types.AgentID("device-1"), "user-1")

	addr := startTCPServer(t, manager, verifier)
	client := testClient(addr)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- client.Run(ctx, DeviceInfo{ID: "device-1"}, func() (string, error) { return "good-token", nil })
	}()

	if !waitFor(client.Connected) {
		t.Fatal("client never reported Connected()")
	}
	if !waitFor(func() bool { return manager.IsConnected("device-1") }) {
		t.Fatal("device-1 never appeared in Manager")
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v, want nil on graceful shutdown", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not return after context cancellation")
	}

	if client.Connected() {
		t.Error("client still reports Connected() after shutdown")
	}
}

func TestClientRetriesAndRecoversAfterInvalidToken(t *testing.T) {
	manager := NewManager()
	verifier := newFakeVerifier()
	// Deliberately not allowing "will-be-valid-token" yet.

	addr := startTCPServer(t, manager, verifier)
	client := testClient(addr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tokenValid := make(chan struct{})
	tokenSource := func() (string, error) {
		select {
		case <-tokenValid:
			return "will-be-valid-token", nil
		default:
			return "not-yet-valid-token", nil
		}
	}

	go func() { _ = client.Run(ctx, DeviceInfo{ID: "device-1"}, tokenSource) }()

	// Give it a couple of failed attempts to prove it doesn't give up.
	time.Sleep(60 * time.Millisecond)
	if client.Connected() {
		t.Fatal("client reports Connected() with a token the server hasn't allowed yet")
	}

	verifier.allow("will-be-valid-token", types.AgentID("device-1"), "user-1")
	close(tokenValid)

	if !waitFor(client.Connected) {
		t.Fatal("client never recovered after the token became valid")
	}
}

func TestClientAutomaticallyReconnectsAfterServerRestart(t *testing.T) {
	manager1 := NewManager()
	verifier := newFakeVerifier()
	verifier.allow("good-token", types.AgentID("device-1"), "user-1")

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listening: %v", err)
	}
	addr := lis.Addr().String()

	grpcSrv1 := grpc.NewServer()
	deployosv1.RegisterConnectionServiceServer(grpcSrv1, NewServer(manager1, verifier, testLogger()))
	go func() { _ = grpcSrv1.Serve(lis) }()

	client := testClient(addr)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = client.Run(ctx, DeviceInfo{ID: "device-1"}, func() (string, error) { return "good-token", nil })
	}()

	if !waitFor(client.Connected) {
		t.Fatal("client never connected to the first server")
	}

	// Simulate the control plane going away.
	grpcSrv1.Stop()

	if !waitForNot(client.Connected) {
		t.Fatal("client still reports Connected() after the server stopped")
	}

	// Bring a new server back up on the same address, as if the control
	// plane restarted.
	lis2, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("re-listening on %s: %v", addr, err)
	}
	manager2 := NewManager()
	grpcSrv2 := grpc.NewServer()
	deployosv1.RegisterConnectionServiceServer(grpcSrv2, NewServer(manager2, verifier, testLogger()))
	go func() { _ = grpcSrv2.Serve(lis2) }()
	t.Cleanup(grpcSrv2.Stop)

	if !waitFor(client.Connected) {
		t.Fatal("client never reconnected to the restarted server")
	}
	if !waitFor(func() bool { return manager2.IsConnected("device-1") }) {
		t.Fatal("device-1 never appeared in the restarted server's Manager")
	}
}

// waitForNot polls cond until it reports false, or times out.
func waitForNot(cond func() bool) bool {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !cond() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return !cond()
}
