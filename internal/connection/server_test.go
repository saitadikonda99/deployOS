package connection

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	deployosv1 "github.com/saitadikonda99/deployOS/gen/go/deployos/v1"
	"github.com/saitadikonda99/deployOS/pkg/types"
)

const bufSize = 1024 * 1024

// fakeVerifier is a deterministic TokenVerifier for tests: valid maps a
// token to the (deviceID, userID) it authenticates as, everything else
// is rejected.
type fakeVerifier struct {
	valid map[string]struct {
		deviceID types.AgentID
		userID   string
	}
}

func newFakeVerifier() *fakeVerifier {
	return &fakeVerifier{valid: make(map[string]struct {
		deviceID types.AgentID
		userID   string
	})}
}

func (f *fakeVerifier) allow(token string, deviceID types.AgentID, userID string) {
	f.valid[token] = struct {
		deviceID types.AgentID
		userID   string
	}{deviceID, userID}
}

func (f *fakeVerifier) Verify(_ context.Context, token string) (types.AgentID, string, error) {
	entry, ok := f.valid[token]
	if !ok {
		return "", "", errors.New("invalid token")
	}
	return entry.deviceID, entry.userID, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// testServer wires a real Server and Manager behind an in-memory
// (bufconn) gRPC server, returning a client factory and the Manager so
// tests can assert on connection state.
type testServer struct {
	manager  *Manager
	verifier *fakeVerifier
	client   deployosv1.ConnectionServiceClient
	conn     *grpc.ClientConn
	grpcSrv  *grpc.Server
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	manager := NewManager()
	verifier := newFakeVerifier()

	grpcSrv := grpc.NewServer()
	deployosv1.RegisterConnectionServiceServer(grpcSrv, NewServer(manager, verifier, testLogger()))

	go func() {
		_ = grpcSrv.Serve(lis)
	}()
	t.Cleanup(grpcSrv.Stop)

	dialer := func(context.Context, string) (net.Conn, error) { return lis.Dial() }
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dialing bufconn: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return &testServer{
		manager:  manager,
		verifier: verifier,
		client:   deployosv1.NewConnectionServiceClient(conn),
		conn:     conn,
		grpcSrv:  grpcSrv,
	}
}

func authRequest(token, deviceID string) *deployosv1.ConnectionEnvelope {
	return &deployosv1.ConnectionEnvelope{
		Payload: &deployosv1.ConnectionEnvelope_AuthenticateRequest{
			AuthenticateRequest: &deployosv1.AuthenticateRequest{
				Connection: &deployosv1.Connection{DeviceToken: token, ProtocolVersion: ProtocolVersion},
				Device:     &deployosv1.Device{Id: deviceID, Hostname: "test-host"},
			},
		},
	}
}

func TestServerAcceptsValidAuthentication(t *testing.T) {
	ts := newTestServer(t)
	ts.verifier.allow("good-token", types.AgentID("device-1"), "user-1")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := ts.client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if err := stream.Send(authRequest("good-token", "device-1")); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	envelope, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv() error = %v", err)
	}
	resp := envelope.GetAuthenticateResponse()
	if resp == nil || !resp.GetAuthenticated() {
		t.Fatalf("AuthenticateResponse = %+v, want authenticated = true", resp)
	}
	if resp.GetSessionId() == "" {
		t.Error("SessionId is empty")
	}

	if !waitFor(func() bool { return ts.manager.IsConnected(types.AgentID("device-1")) }) {
		t.Fatal("device-1 not tracked as connected in Manager")
	}

	cancel()

	if !waitFor(func() bool { return !ts.manager.IsConnected(types.AgentID("device-1")) }) {
		t.Fatal("device-1 still tracked as connected after client disconnected")
	}
}

func TestServerRejectsInvalidToken(t *testing.T) {
	ts := newTestServer(t)

	stream, err := ts.client.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if err := stream.Send(authRequest("bad-token", "device-1")); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	envelope, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv() error = %v", err)
	}
	resp := envelope.GetAuthenticateResponse()
	if resp == nil || resp.GetAuthenticated() {
		t.Fatalf("AuthenticateResponse = %+v, want authenticated = false", resp)
	}
	if resp.GetError() == "" {
		t.Error("Error is empty, want a rejection reason")
	}

	if ts.manager.Count() != 0 {
		t.Fatalf("Manager.Count() = %d, want 0 for rejected authentication", ts.manager.Count())
	}
}

func TestServerRejectsDeviceIDMismatch(t *testing.T) {
	ts := newTestServer(t)
	ts.verifier.allow("good-token", types.AgentID("device-1"), "user-1")

	stream, err := ts.client.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	// Token is valid for device-1, but the request claims to be device-2.
	if err := stream.Send(authRequest("good-token", "device-2")); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	envelope, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv() error = %v", err)
	}
	resp := envelope.GetAuthenticateResponse()
	if resp == nil || resp.GetAuthenticated() {
		t.Fatalf("AuthenticateResponse = %+v, want authenticated = false", resp)
	}
}

func TestServerRejectsNonAuthenticateFirstMessage(t *testing.T) {
	ts := newTestServer(t)

	stream, err := ts.client.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	// An AuthenticateResponse (server->client kind) sent by a client as
	// its first message is not a valid AuthenticateRequest.
	if err := stream.Send(&deployosv1.ConnectionEnvelope{
		Payload: &deployosv1.ConnectionEnvelope_AuthenticateResponse{
			AuthenticateResponse: &deployosv1.AuthenticateResponse{Authenticated: true},
		},
	}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if _, err := stream.Recv(); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestServerHandlesMultipleConcurrentAgents(t *testing.T) {
	ts := newTestServer(t)

	const numAgents = 10
	for i := range numAgents {
		ts.verifier.allow(tokenFor(i), types.AgentID(deviceIDFor(i)), "user-1")
	}

	var wg sync.WaitGroup
	ctxs := make([]context.CancelFunc, numAgents)

	for i := range numAgents {
		ctx, cancel := context.WithCancel(context.Background())
		ctxs[i] = cancel

		wg.Add(1)
		go func(i int, ctx context.Context) {
			defer wg.Done()

			stream, err := ts.client.Connect(ctx)
			if err != nil {
				t.Errorf("agent %d: Connect() error = %v", i, err)
				return
			}
			if err := stream.Send(authRequest(tokenFor(i), deviceIDFor(i))); err != nil {
				t.Errorf("agent %d: Send() error = %v", i, err)
				return
			}
			envelope, err := stream.Recv()
			if err != nil {
				t.Errorf("agent %d: Recv() error = %v", i, err)
				return
			}
			if !envelope.GetAuthenticateResponse().GetAuthenticated() {
				t.Errorf("agent %d: not authenticated", i)
			}
		}(i, ctx)
	}
	wg.Wait()

	if !waitFor(func() bool { return ts.manager.Count() == numAgents }) {
		t.Fatalf("Manager.Count() = %d, want %d", ts.manager.Count(), numAgents)
	}

	for _, cancel := range ctxs {
		cancel()
	}

	if !waitFor(func() bool { return ts.manager.Count() == 0 }) {
		t.Fatalf("Manager.Count() = %d after all agents disconnected, want 0", ts.manager.Count())
	}
}

func tokenFor(i int) string    { return "token-" + string(rune('a'+i)) }
func deviceIDFor(i int) string { return "device-" + string(rune('a'+i)) }

// waitFor polls cond for up to a second, for asserting on state that
// changes asynchronously (e.g. Manager updates driven by the server
// goroutine noticing a stream ended).
func waitFor(cond func() bool) bool {
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return cond()
}
