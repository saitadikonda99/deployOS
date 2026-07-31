package connection

import (
	"context"
	"errors"
	"sync"
	"testing"

	deployosv1 "github.com/saitadikonda99/deployOS/gen/go/deployos/v1"
	"github.com/saitadikonda99/deployOS/pkg/types"
)

func TestServerSendReturnsErrDeviceNotConnectedWhenUnknown(t *testing.T) {
	ts := newTestServer(t)

	err := ts.server.Send(types.AgentID("never-connected"), &deployosv1.ConnectionEnvelope{})
	if !errors.Is(err, ErrDeviceNotConnected) {
		t.Fatalf("Send() error = %v, want ErrDeviceNotConnected", err)
	}
}

func TestServerSendDeliversToConnectedDevice(t *testing.T) {
	ts := newTestServer(t)
	ts.verifier.allow("good-token", types.AgentID("device-1"), "user-1")

	stream, err := ts.client.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if err := stream.Send(authRequest("good-token", "device-1")); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if _, err := stream.Recv(); err != nil { // AuthenticateResponse
		t.Fatalf("Recv() error = %v", err)
	}

	cmd := &deployosv1.Command{Id: "cmd-1", Kind: "PING"}
	sendErr := make(chan error, 1)
	go func() {
		sendErr <- ts.server.Send(types.AgentID("device-1"), &deployosv1.ConnectionEnvelope{
			Payload: &deployosv1.ConnectionEnvelope_CommandRequest{CommandRequest: cmd},
		})
	}()

	if err := <-sendErr; err != nil {
		t.Fatalf("server.Send() error = %v", err)
	}

	envelope, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv() error = %v", err)
	}
	got := envelope.GetCommandRequest()
	if got == nil || got.GetId() != "cmd-1" || got.GetKind() != "PING" {
		t.Fatalf("received command = %+v, want id=cmd-1 kind=PING", got)
	}
}

func TestServerOnMessageInvokedForNonAuthPayload(t *testing.T) {
	ts := newTestServer(t)
	ts.verifier.allow("good-token", types.AgentID("device-1"), "user-1")

	var mu sync.Mutex
	var gotDeviceID types.AgentID
	var gotResult *deployosv1.CommandResult
	received := make(chan struct{})

	ts.server.OnMessage(func(deviceID types.AgentID, envelope *deployosv1.ConnectionEnvelope) {
		if result := envelope.GetCommandResponse(); result != nil {
			mu.Lock()
			gotDeviceID = deviceID
			gotResult = result
			mu.Unlock()
			close(received)
		}
	})

	stream, err := ts.client.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if err := stream.Send(authRequest("good-token", "device-1")); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if _, err := stream.Recv(); err != nil { // AuthenticateResponse
		t.Fatalf("Recv() error = %v", err)
	}

	if err := stream.Send(&deployosv1.ConnectionEnvelope{
		Payload: &deployosv1.ConnectionEnvelope_CommandResponse{
			CommandResponse: &deployosv1.CommandResult{CommandId: "cmd-1", Success: true},
		},
	}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if !waitForClosed(received) {
		t.Fatal("OnMessage was never invoked")
	}

	mu.Lock()
	defer mu.Unlock()
	if gotDeviceID != "device-1" {
		t.Errorf("deviceID = %q, want %q", gotDeviceID, "device-1")
	}
	if gotResult.GetCommandId() != "cmd-1" || !gotResult.GetSuccess() {
		t.Errorf("result = %+v, want CommandId=cmd-1 Success=true", gotResult)
	}
}

// waitForClosed reports whether ch is closed within a short deadline.
func waitForClosed(ch <-chan struct{}) bool {
	return waitFor(func() bool {
		select {
		case <-ch:
			return true
		default:
			return false
		}
	})
}

func TestClientOnCommandRespondsOverTheWire(t *testing.T) {
	manager := NewManager()
	verifier := newFakeVerifier()
	verifier.allow("good-token", types.AgentID("device-1"), "user-1")

	server := NewServer(manager, verifier, testLogger())

	var mu sync.Mutex
	var gotResult *deployosv1.CommandResult
	received := make(chan struct{})
	server.OnMessage(func(_ types.AgentID, envelope *deployosv1.ConnectionEnvelope) {
		if result := envelope.GetCommandResponse(); result != nil {
			mu.Lock()
			gotResult = result
			mu.Unlock()
			close(received)
		}
	})

	addr := startTCPServerWithServer(t, server)
	client := testClient(addr)
	client.OnCommand(func(_ context.Context, cmd *deployosv1.Command) *deployosv1.CommandResult {
		return &deployosv1.CommandResult{CommandId: cmd.GetId(), Success: true, Message: "pong"}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = client.Run(ctx, DeviceInfo{ID: "device-1"}, func() (string, error) { return "good-token", nil })
	}()

	if !waitFor(client.Connected) {
		t.Fatal("client never connected")
	}

	if err := server.Send(types.AgentID("device-1"), &deployosv1.ConnectionEnvelope{
		Payload: &deployosv1.ConnectionEnvelope_CommandRequest{
			CommandRequest: &deployosv1.Command{Id: "cmd-1", Kind: "PING"},
		},
	}); err != nil {
		t.Fatalf("server.Send() error = %v", err)
	}

	if !waitForClosed(received) {
		t.Fatal("server never received the command result")
	}

	mu.Lock()
	defer mu.Unlock()
	if gotResult.GetCommandId() != "cmd-1" || !gotResult.GetSuccess() || gotResult.GetMessage() != "pong" {
		t.Errorf("result = %+v, want CommandId=cmd-1 Success=true Message=pong", gotResult)
	}
}
