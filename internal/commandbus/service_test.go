package commandbus

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	deployosv1 "github.com/saitadikonda99/deployOS/gen/go/deployos/v1"
	"github.com/saitadikonda99/deployOS/pkg/types"
)

// fakeSender records every envelope it's asked to send, keyed by the
// command ID Service generated for it, so tests can discover the ID
// without depending on how Service generates it.
type fakeSender struct {
	mu  sync.Mutex
	env map[string]*deployosv1.ConnectionEnvelope
	err error
}

func newFakeSender() *fakeSender {
	return &fakeSender{env: make(map[string]*deployosv1.ConnectionEnvelope)}
}

func (f *fakeSender) Send(_ types.AgentID, envelope *deployosv1.ConnectionEnvelope) error {
	if f.err != nil {
		return f.err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.env[envelope.GetCommandRequest().GetId()] = envelope
	return nil
}

// waitForIDs blocks until at least n commands have been sent, returning
// their command IDs (in map-iteration, i.e. unspecified, order).
func (f *fakeSender) waitForIDs(t *testing.T, n int) []string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		if len(f.env) >= n {
			ids := make([]string, 0, len(f.env))
			for id := range f.env {
				ids = append(ids, id)
			}
			f.mu.Unlock()
			return ids
		}
		f.mu.Unlock()
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d commands to be sent", n)
	return nil
}

func resultEnvelope(commandID string, success bool, message string) *deployosv1.ConnectionEnvelope {
	return &deployosv1.ConnectionEnvelope{
		Payload: &deployosv1.ConnectionEnvelope_CommandResponse{
			CommandResponse: &deployosv1.CommandResult{CommandId: commandID, Success: success, Message: message},
		},
	}
}

func TestServiceSendSuccess(t *testing.T) {
	sender := newFakeSender()
	svc := NewService(sender, testLogger())

	done := make(chan struct{})
	var resp Response
	var sendErr error
	go func() {
		resp, sendErr = svc.Send(context.Background(), types.AgentID("device-1"), Request{Kind: KindPing})
		close(done)
	}()

	ids := sender.waitForIDs(t, 1)
	svc.HandleResult(types.AgentID("device-1"), resultEnvelope(ids[0], true, "pong"))

	<-done
	if sendErr != nil {
		t.Fatalf("Send() error = %v", sendErr)
	}
	if !resp.Success || resp.Message != "pong" {
		t.Errorf("Send() response = %+v, want Success=true Message=pong", resp)
	}
	if resp.CommandID != ids[0] {
		t.Errorf("CommandID = %q, want %q", resp.CommandID, ids[0])
	}
}

func TestServiceSendReturnsErrDeviceNotConnected(t *testing.T) {
	sender := newFakeSender()
	sender.err = errors.New("no active stream")
	svc := NewService(sender, testLogger())

	_, err := svc.Send(context.Background(), types.AgentID("device-1"), Request{Kind: KindPing})
	if !errors.Is(err, ErrDeviceNotConnected) {
		t.Fatalf("Send() error = %v, want ErrDeviceNotConnected", err)
	}
}

func TestServiceSendTimesOut(t *testing.T) {
	sender := newFakeSender()
	svc := NewService(sender, testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	// Deliberately never call HandleResult - the device never responds.
	_, err := svc.Send(ctx, types.AgentID("device-1"), Request{Kind: KindPing})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Send() error = %v, want context.DeadlineExceeded", err)
	}
}

func TestServiceHandleResultForUnknownCommandIsDropped(_ *testing.T) {
	sender := newFakeSender()
	svc := NewService(sender, testLogger())

	// Must not panic, and must not affect anything else.
	svc.HandleResult(types.AgentID("device-1"), resultEnvelope("never-sent", true, "stray"))
}

func TestServiceHandleResultIgnoresNonCommandEnvelopes(_ *testing.T) {
	sender := newFakeSender()
	svc := NewService(sender, testLogger())

	// An auth envelope (or any non-command payload) must be ignored,
	// not misinterpreted as a command result.
	svc.HandleResult(types.AgentID("device-1"), &deployosv1.ConnectionEnvelope{
		Payload: &deployosv1.ConnectionEnvelope_AuthenticateResponse{
			AuthenticateResponse: &deployosv1.AuthenticateResponse{Authenticated: true},
		},
	})
}

func TestServiceResponseCorrelationWithConcurrentCommands(t *testing.T) {
	sender := newFakeSender()
	svc := NewService(sender, testLogger())

	type outcome struct {
		resp Response
		err  error
	}
	results := make(chan outcome, 2)

	go func() {
		resp, err := svc.Send(context.Background(), types.AgentID("device-1"), Request{Kind: KindPing})
		results <- outcome{resp, err}
	}()
	go func() {
		resp, err := svc.Send(context.Background(), types.AgentID("device-1"), Request{Kind: KindGetVersion})
		results <- outcome{resp, err}
	}()

	ids := sender.waitForIDs(t, 2)

	// Deliver results in the reverse of however they happened to be
	// captured, each with a message tied to its own ID - if correlation
	// were broken (e.g. a single shared channel), a Send() call could
	// receive the wrong result.
	svc.HandleResult(types.AgentID("device-1"), resultEnvelope(ids[1], true, "for-"+ids[1]))
	svc.HandleResult(types.AgentID("device-1"), resultEnvelope(ids[0], true, "for-"+ids[0]))

	got := make(map[string]string, 2)
	for range 2 {
		o := <-results
		if o.err != nil {
			t.Fatalf("Send() error = %v", o.err)
		}
		got[o.resp.CommandID] = o.resp.Message
	}

	for _, id := range ids {
		if want := "for-" + id; got[id] != want {
			t.Errorf("result for command %s = %q, want %q", id, got[id], want)
		}
	}
}

func TestServiceHandlesMultipleConcurrentCommands(t *testing.T) {
	sender := newFakeSender()
	svc := NewService(sender, testLogger())

	const n = 10
	var wg sync.WaitGroup
	errs := make([]error, n)
	resps := make([]Response, n)

	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resps[i], errs[i] = svc.Send(context.Background(), types.AgentID("device-1"), Request{Kind: KindPing})
		}(i)
	}

	ids := sender.waitForIDs(t, n)
	for _, id := range ids {
		svc.HandleResult(types.AgentID("device-1"), resultEnvelope(id, true, ""))
	}

	wg.Wait()
	for i := range n {
		if errs[i] != nil {
			t.Fatalf("Send()[%d] error = %v", i, errs[i])
		}
		if !resps[i].Success {
			t.Fatalf("Send()[%d].Success = false, want true", i)
		}
	}
}
