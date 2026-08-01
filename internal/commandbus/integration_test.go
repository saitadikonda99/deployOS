// Integration test wiring the real internal/connection.Server/Client
// together with the real commandbus.Service/Dispatcher over a live TCP
// gRPC connection - the same shape cmd/server and cmd/agent wire up in
// production, just without the HTTP layer or Postgres.
package commandbus

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"

	deployosv1 "github.com/saitadikonda99/deployOS/gen/go/deployos/v1"
	"github.com/saitadikonda99/deployOS/internal/connection"
	"github.com/saitadikonda99/deployOS/pkg/types"
)

type fakeVerifier struct {
	deviceID types.AgentID
	userID   string
}

func (f fakeVerifier) Verify(_ context.Context, token string) (types.AgentID, string, error) {
	if token != "good-token" {
		return "", "", errors.New("invalid token")
	}
	return f.deviceID, f.userID, nil
}

// integrationRig wires a real control-plane side (connection.Server +
// commandbus.Service) to a real agent side (connection.Client +
// commandbus.Dispatcher) over a live TCP connection.
type integrationRig struct {
	service *Service
	client  *connection.Client
}

func newIntegrationRig(t *testing.T, deviceID types.AgentID) *integrationRig {
	t.Helper()

	manager := connection.NewManager()
	server := connection.NewServer(manager, fakeVerifier{deviceID: deviceID, userID: "user-1"}, testLogger())
	service := NewService(server, testLogger())
	server.OnMessage(service.HandleResult)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listening: %v", err)
	}
	grpcSrv := grpc.NewServer()
	deployosv1.RegisterConnectionServiceServer(grpcSrv, server)
	go func() { _ = grpcSrv.Serve(lis) }()
	t.Cleanup(grpcSrv.Stop)

	client := connection.NewClient(lis.Addr().String(), testLogger())

	return &integrationRig{service: service, client: client}
}

func (r *integrationRig) start(t *testing.T, deviceID types.AgentID) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() {
		_ = r.client.Run(ctx, connection.DeviceInfo{ID: deviceID}, func() (string, error) { return "good-token", nil })
	}()

	deadline := time.Now().Add(2 * time.Second)
	for !r.client.Connected() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !r.client.Connected() {
		t.Fatal("agent never connected")
	}
}

func TestIntegrationPingCommandRoundTrip(t *testing.T) {
	const deviceID = types.AgentID("device-1")

	rig := newIntegrationRig(t, deviceID)

	dispatcher := NewDispatcher(testLogger())
	dispatcher.Register(KindPing, ExecutorFunc(func(_ context.Context, _ Request) Response {
		return Response{Success: true, Message: "pong"}
	}))
	rig.client.OnCommand(WireHandler(dispatcher))
	rig.start(t, deviceID)

	resp, err := rig.service.Send(context.Background(), deviceID, Request{Kind: KindPing})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !resp.Success || resp.Message != "pong" {
		t.Fatalf("Send() response = %+v, want Success=true Message=pong", resp)
	}
}

func TestIntegrationUnknownCommandRoundTrip(t *testing.T) {
	const deviceID = types.AgentID("device-1")

	rig := newIntegrationRig(t, deviceID)

	// No executors registered at all - the agent's Dispatcher must
	// still respond gracefully, not hang or crash.
	dispatcher := NewDispatcher(testLogger())
	rig.client.OnCommand(WireHandler(dispatcher))
	rig.start(t, deviceID)

	resp, err := rig.service.Send(context.Background(), deviceID, Request{Kind: "NOT_A_REAL_COMMAND"})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if resp.Success {
		t.Fatalf("Send() response = %+v, want Success=false for an unknown command", resp)
	}
}

func TestIntegrationMultipleConcurrentCommandsRoundTrip(t *testing.T) {
	const deviceID = types.AgentID("device-1")

	rig := newIntegrationRig(t, deviceID)

	dispatcher := NewDispatcher(testLogger())
	dispatcher.Register(KindPing, ExecutorFunc(func(_ context.Context, _ Request) Response {
		return Response{Success: true, Message: "pong"}
	}))
	dispatcher.Register(KindGetVersion, ExecutorFunc(func(_ context.Context, _ Request) Response {
		return Response{Success: true, Message: "0.1.0"}
	}))
	rig.client.OnCommand(WireHandler(dispatcher))
	rig.start(t, deviceID)

	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	resps := make([]Response, n)

	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			kind := KindPing
			if i%2 == 0 {
				kind = KindGetVersion
			}
			resps[i], errs[i] = rig.service.Send(context.Background(), deviceID, Request{Kind: kind})
		}(i)
	}
	wg.Wait()

	for i := range n {
		if errs[i] != nil {
			t.Fatalf("Send()[%d] error = %v", i, errs[i])
		}
		if !resps[i].Success {
			t.Fatalf("Send()[%d] = %+v, want Success=true", i, resps[i])
		}
		wantMessage := "pong"
		if i%2 == 0 {
			wantMessage = "0.1.0"
		}
		if resps[i].Message != wantMessage {
			t.Errorf("Send()[%d].Message = %q, want %q", i, resps[i].Message, wantMessage)
		}
	}
}

// TestIntegrationContainerCommandsRoundTrip proves LIST_CONTAINERS and
// INSPECT_CONTAINER round-trip over a real connection like any other
// command - including that Request.Arguments (INSPECT_CONTAINER's "id")
// actually survives the wire. It uses stand-in executors, not
// internal/agent's real ones, since commandbus has no knowledge of
// containers.
func TestIntegrationContainerCommandsRoundTrip(t *testing.T) {
	const deviceID = types.AgentID("device-1")

	rig := newIntegrationRig(t, deviceID)

	dispatcher := NewDispatcher(testLogger())
	dispatcher.Register(KindListContainers, ExecutorFunc(func(_ context.Context, _ Request) Response {
		return Response{Success: true, Details: map[string]string{"containers": `[{"id":"c1"}]`}}
	}))
	dispatcher.Register(KindInspectContainer, ExecutorFunc(func(_ context.Context, req Request) Response {
		if req.Arguments["id"] != "c1" {
			return Response{Success: false, Message: "missing or wrong id argument"}
		}
		return Response{Success: true, Details: map[string]string{"container": `{"id":"c1"}`}}
	}))
	rig.client.OnCommand(WireHandler(dispatcher))
	rig.start(t, deviceID)

	listResp, err := rig.service.Send(context.Background(), deviceID, Request{Kind: KindListContainers})
	if err != nil {
		t.Fatalf("Send(LIST_CONTAINERS) error = %v", err)
	}
	if !listResp.Success || listResp.Details["containers"] == "" {
		t.Fatalf("Send(LIST_CONTAINERS) = %+v, want Success=true with a containers detail", listResp)
	}

	inspectResp, err := rig.service.Send(context.Background(), deviceID, Request{
		Kind:      KindInspectContainer,
		Arguments: map[string]string{"id": "c1"},
	})
	if err != nil {
		t.Fatalf("Send(INSPECT_CONTAINER) error = %v", err)
	}
	if !inspectResp.Success || inspectResp.Details["container"] == "" {
		t.Fatalf("Send(INSPECT_CONTAINER) = %+v, want Success=true with a container detail", inspectResp)
	}
}
