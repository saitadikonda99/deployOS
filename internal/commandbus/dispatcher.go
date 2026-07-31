package commandbus

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Executor executes one command kind and produces its Response. An
// Executor should never need to know about gRPC, the connection, or
// anything transport-related - that's Dispatcher's job.
type Executor interface {
	Execute(ctx context.Context, req Request) Response
}

// ExecutorFunc adapts a plain function to Executor.
type ExecutorFunc func(ctx context.Context, req Request) Response

// Execute implements Executor.
func (f ExecutorFunc) Execute(ctx context.Context, req Request) Response {
	return f(ctx, req)
}

// Dispatcher routes an incoming command to its registered executor by
// kind. Adding a new command requires writing an Executor and calling
// Register - nothing else.
type Dispatcher struct {
	mu        sync.RWMutex
	executors map[string]Executor
	logger    *slog.Logger
}

// NewDispatcher builds an empty Dispatcher.
func NewDispatcher(logger *slog.Logger) *Dispatcher {
	return &Dispatcher{executors: make(map[string]Executor), logger: logger}
}

// Register adds an executor for kind. Registering the same kind twice
// replaces the previous executor.
func (d *Dispatcher) Register(kind string, executor Executor) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.executors[kind] = executor
}

// Dispatch routes req to its registered executor and always returns a
// Response - never an error, and never a panic. An unregistered kind
// produces a structured failure response instead of being rejected at a
// lower layer, and an executor panic is recovered and converted to a
// failure response too, so one broken command can never take down the
// agent process handling it.
func (d *Dispatcher) Dispatch(ctx context.Context, req Request) (resp Response) {
	defer func() {
		if r := recover(); r != nil {
			d.logger.Error("command executor panicked",
				slog.String("command_id", req.ID),
				slog.String("kind", req.Kind),
				slog.Any("panic", r),
			)
			resp = Response{CommandID: req.ID, Success: false, Message: "internal error handling command"}
		}
	}()

	d.mu.RLock()
	executor, ok := d.executors[req.Kind]
	d.mu.RUnlock()

	if !ok {
		return Response{CommandID: req.ID, Success: false, Message: fmt.Sprintf("unknown command kind %q", req.Kind)}
	}

	resp = executor.Execute(ctx, req)
	resp.CommandID = req.ID
	return resp
}
