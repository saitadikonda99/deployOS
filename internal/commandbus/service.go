package commandbus

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"

	deployosv1 "github.com/saitadikonda99/deployOS/gen/go/deployos/v1"
	"github.com/saitadikonda99/deployOS/pkg/types"
)

// Sender delivers an envelope to a specific device's connection.
// internal/connection.Server implements this; Service depends only on
// this interface, not on internal/connection, so this package stays
// decoupled from how the connection itself is dialed, authenticated, or
// reconnected.
type Sender interface {
	Send(deviceID types.AgentID, envelope *deployosv1.ConnectionEnvelope) error
}

// ErrDeviceNotConnected is returned by Send when the target device has
// no active connection. It wraps whatever the underlying Sender
// reported, so callers can still inspect the original cause.
var ErrDeviceNotConnected = errors.New("device is not connected")

type pendingRequest struct {
	resultCh chan *deployosv1.CommandResult
}

// Service is the control plane's half of the Command Bus: it creates
// commands, sends them to a specific device over its existing
// connection, and waits for the matching result - correlating it by
// command ID, and enforcing a timeout via the caller's context.
type Service struct {
	sender Sender
	logger *slog.Logger

	mu      sync.Mutex
	pending map[string]*pendingRequest
}

// NewService builds a Service backed by sender.
func NewService(sender Sender, logger *slog.Logger) *Service {
	return &Service{sender: sender, logger: logger, pending: make(map[string]*pendingRequest)}
}

// Send delivers req to deviceID and blocks until a matching result
// arrives or ctx is done. Callers that want a timeout should give ctx
// one (e.g. via context.WithTimeout) - Send has no timeout of its own,
// so "handle timeouts" is just "respect the context you're given," the
// same as everywhere else in this codebase.
func (s *Service) Send(ctx context.Context, deviceID types.AgentID, req Request) (Response, error) {
	id := uuid.NewString()
	resultCh := make(chan *deployosv1.CommandResult, 1)

	s.mu.Lock()
	s.pending[id] = &pendingRequest{resultCh: resultCh}
	s.mu.Unlock()

	cleanup := func() {
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
	}

	envelope := &deployosv1.ConnectionEnvelope{
		Payload: &deployosv1.ConnectionEnvelope_CommandRequest{
			CommandRequest: &deployosv1.Command{
				Id:        id,
				Kind:      req.Kind,
				Arguments: metadataFromMap(req.Arguments),
			},
		},
	}

	if err := s.sender.Send(deviceID, envelope); err != nil {
		cleanup()
		return Response{}, fmt.Errorf("%w: %v", ErrDeviceNotConnected, err)
	}

	select {
	case result := <-resultCh:
		return Response{
			CommandID: result.GetCommandId(),
			Success:   result.GetSuccess(),
			Message:   result.GetMessage(),
			Details:   metadataToMap(result.GetDetails()),
		}, nil
	case <-ctx.Done():
		cleanup()
		return Response{}, ctx.Err()
	}
}

// HandleResult delivers a received CommandResult to whichever Send call
// is waiting for it, matched by command ID. It's the
// internal/connection.IncomingHandler Service should be registered as
// (via Server.OnMessage); envelopes carrying anything other than a
// command result are ignored, and a result for an unknown or
// already-completed command ID is logged and dropped rather than
// panicking or blocking.
func (s *Service) HandleResult(deviceID types.AgentID, envelope *deployosv1.ConnectionEnvelope) {
	result := envelope.GetCommandResponse()
	if result == nil {
		return
	}

	s.mu.Lock()
	pending, ok := s.pending[result.GetCommandId()]
	if ok {
		delete(s.pending, result.GetCommandId())
	}
	s.mu.Unlock()

	if !ok {
		s.logger.Warn("received command result for unknown or already-completed command",
			slog.String("device_id", deviceID.String()),
			slog.String("command_id", result.GetCommandId()),
		)
		return
	}

	// Buffered by 1 and only ever written once per command ID (this is
	// the only place that writes to it), so this never blocks.
	pending.resultCh <- result
}
