package connection

import (
	"errors"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	deployosv1 "github.com/saitadikonda99/deployOS/gen/go/deployos/v1"
	"github.com/saitadikonda99/deployOS/pkg/types"
)

// streamHandle serializes writes to a single stream: gRPC streams are
// not safe for concurrent Send calls, but both Connect's own goroutine
// and a Send from the Command Bus (or any future feature) can want to
// write to the same device's stream at once.
type streamHandle struct {
	mu     sync.Mutex
	stream deployosv1.ConnectionService_ConnectServer
}

func (h *streamHandle) send(envelope *deployosv1.ConnectionEnvelope) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.stream.Send(envelope)
}

// Server implements deployos.v1.ConnectionService: it authenticates each
// incoming stream against a TokenVerifier, tracks authenticated
// connections in a Manager, routes non-auth payloads to an optional
// IncomingHandler, and removes the connection when its stream ends for
// any reason.
type Server struct {
	deployosv1.UnimplementedConnectionServiceServer

	manager  *Manager
	verifier TokenVerifier
	logger   *slog.Logger

	onMessage IncomingHandler

	mu      sync.RWMutex
	streams map[types.AgentID]*streamHandle
}

// NewServer builds a Server.
func NewServer(manager *Manager, verifier TokenVerifier, logger *slog.Logger) *Server {
	return &Server{
		manager:  manager,
		verifier: verifier,
		logger:   logger,
		streams:  make(map[types.AgentID]*streamHandle),
	}
}

// OnMessage registers the callback invoked for every envelope payload
// received from an authenticated device that isn't part of the
// authentication handshake. Register it before Server starts serving
// connections - it isn't safe to change concurrently with Connect.
func (s *Server) OnMessage(handler IncomingHandler) {
	s.onMessage = handler
}

// Send delivers envelope to deviceID's active connection, if any.
func (s *Server) Send(deviceID types.AgentID, envelope *deployosv1.ConnectionEnvelope) error {
	s.mu.RLock()
	handle, ok := s.streams[deviceID]
	s.mu.RUnlock()

	if !ok {
		return ErrDeviceNotConnected
	}

	return handle.send(envelope)
}

// Connect implements the ConnectionService.Connect RPC: the first
// message must be an AuthenticateRequest, after which the connection is
// tracked in Manager and made reachable via Send until the stream ends.
func (s *Server) Connect(stream deployosv1.ConnectionService_ConnectServer) error {
	deviceID, userID, err := s.authenticate(stream)
	if err != nil {
		return err
	}

	handle := &streamHandle{stream: stream}
	s.mu.Lock()
	s.streams[deviceID] = handle
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		// Only remove our own entry: a rapid reconnect may already have
		// registered a newer handle for the same device by the time
		// this stream ends.
		if s.streams[deviceID] == handle {
			delete(s.streams, deviceID)
		}
		s.mu.Unlock()
	}()

	sessionID := uuid.NewString()
	s.manager.Add(State{
		DeviceID:    deviceID,
		UserID:      userID,
		SessionID:   sessionID,
		ConnectedAt: time.Now(),
	})
	defer s.manager.Remove(deviceID)

	if err := handle.send(&deployosv1.ConnectionEnvelope{
		Payload: &deployosv1.ConnectionEnvelope_AuthenticateResponse{
			AuthenticateResponse: &deployosv1.AuthenticateResponse{
				Authenticated: true,
				SessionId:     sessionID,
			},
		},
	}); err != nil {
		return err
	}

	s.logger.Info("agent connected",
		slog.String("device_id", deviceID.String()),
		slog.String("session_id", sessionID),
	)
	defer s.logger.Info("agent disconnected",
		slog.String("device_id", deviceID.String()),
		slog.String("session_id", sessionID),
	)

	// Beyond authentication, this loop just receives and routes -
	// forwarding every payload to onMessage (if registered) and relying
	// on Recv() returning an error to detect the stream ending (client
	// disconnect, transport failure, or server shutdown all surface the
	// same way).
	for {
		envelope, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		if s.onMessage != nil {
			s.onMessage(deviceID, envelope)
		}
	}
}

// authenticate reads the first message of stream, which must be an
// AuthenticateRequest, and validates its device token. It sends a
// failure AuthenticateResponse before returning an error, so the agent
// knows *why* it was rejected rather than just seeing the stream close.
func (s *Server) authenticate(stream deployosv1.ConnectionService_ConnectServer) (types.AgentID, string, error) {
	envelope, err := stream.Recv()
	if err != nil {
		return "", "", err
	}

	req := envelope.GetAuthenticateRequest()
	if req == nil {
		return "", "", status.Error(codes.InvalidArgument, "first message must be an authenticate request")
	}

	deviceID, userID, err := s.verifier.Verify(stream.Context(), req.GetConnection().GetDeviceToken())
	if err != nil {
		s.sendAuthFailure(stream, "invalid or expired device token")
		return "", "", status.Error(codes.Unauthenticated, "invalid or expired device token")
	}

	presentedID := types.AgentID(req.GetDevice().GetId())
	if presentedID == "" || presentedID != deviceID {
		s.sendAuthFailure(stream, "device token does not match presented device id")
		return "", "", status.Error(codes.Unauthenticated, "device token does not match presented device id")
	}

	return deviceID, userID, nil
}

func (s *Server) sendAuthFailure(stream deployosv1.ConnectionService_ConnectServer, reason string) {
	_ = stream.Send(&deployosv1.ConnectionEnvelope{
		Payload: &deployosv1.ConnectionEnvelope_AuthenticateResponse{
			AuthenticateResponse: &deployosv1.AuthenticateResponse{
				Authenticated: false,
				Error:         reason,
			},
		},
	})
}
