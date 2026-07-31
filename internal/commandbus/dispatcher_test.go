package commandbus

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestDispatcherDispatchesToRegisteredExecutor(t *testing.T) {
	d := NewDispatcher(testLogger())
	d.Register(KindPing, ExecutorFunc(func(_ context.Context, _ Request) Response {
		return Response{Success: true, Message: "pong"}
	}))

	resp := d.Dispatch(context.Background(), Request{ID: "cmd-1", Kind: KindPing})

	if !resp.Success || resp.Message != "pong" {
		t.Errorf("Dispatch() = %+v, want Success=true Message=pong", resp)
	}
	if resp.CommandID != "cmd-1" {
		t.Errorf("CommandID = %q, want %q", resp.CommandID, "cmd-1")
	}
}

func TestDispatcherHandlesUnknownCommand(t *testing.T) {
	d := NewDispatcher(testLogger())

	resp := d.Dispatch(context.Background(), Request{ID: "cmd-1", Kind: "NOT_REGISTERED"})

	if resp.Success {
		t.Error("Dispatch() Success = true, want false for an unknown kind")
	}
	if resp.CommandID != "cmd-1" {
		t.Errorf("CommandID = %q, want %q", resp.CommandID, "cmd-1")
	}
	if resp.Message == "" {
		t.Error("Message is empty, want an explanation of the unknown kind")
	}
}

func TestDispatcherRecoversFromExecutorPanic(t *testing.T) {
	d := NewDispatcher(testLogger())
	d.Register(KindPing, ExecutorFunc(func(_ context.Context, _ Request) Response {
		// Simulates an executor that panics on a malformed/unexpected
		// payload - Dispatch must never let this escape.
		var arguments map[string]string
		_ = arguments["missing"][0] // deliberately panics: index into nil map value
		return Response{Success: true}
	}))

	resp := d.Dispatch(context.Background(), Request{ID: "cmd-1", Kind: KindPing})

	if resp.Success {
		t.Error("Dispatch() Success = true, want false after a panicking executor")
	}
	if resp.CommandID != "cmd-1" {
		t.Errorf("CommandID = %q, want %q", resp.CommandID, "cmd-1")
	}
}

func TestDispatcherRegisterReplacesExistingExecutor(t *testing.T) {
	d := NewDispatcher(testLogger())
	d.Register(KindPing, ExecutorFunc(func(_ context.Context, _ Request) Response {
		return Response{Success: true, Message: "first"}
	}))
	d.Register(KindPing, ExecutorFunc(func(_ context.Context, _ Request) Response {
		return Response{Success: true, Message: "second"}
	}))

	resp := d.Dispatch(context.Background(), Request{Kind: KindPing})

	if resp.Message != "second" {
		t.Errorf("Message = %q, want %q", resp.Message, "second")
	}
}

// TestDispatcherConcurrentDispatch exercises Dispatch from many
// goroutines at once, for both a registered and an unregistered kind;
// run with -race to catch data races in Register/Dispatch.
func TestDispatcherConcurrentDispatch(_ *testing.T) {
	d := NewDispatcher(testLogger())
	d.Register(KindPing, ExecutorFunc(func(_ context.Context, _ Request) Response {
		return Response{Success: true}
	}))

	const goroutines = 50
	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			kind := KindPing
			if i%2 == 0 {
				kind = "UNKNOWN"
			}
			d.Dispatch(context.Background(), Request{Kind: kind})
		}(i)
	}
	wg.Wait()
}
