package connection

import (
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	deployosv1 "github.com/saitadikonda99/deployOS/gen/go/deployos/v1"
	"github.com/saitadikonda99/deployOS/pkg/types"
)

// Server implements deployos.v1.ConnectionService: it authenticates each
// incoming stream against a TokenVerifier, tracks authenticated
// connections in a Manager, and removes them when the stream ends for
// any reason.
type Server struct {
	deployosv1.UnimplementedConnectionServiceServer

	manager  *Manager
	verifier TokenVerifier
	logger   *slog.Logger
}

// NewServer builds a Server.
func NewServer(manager *Manager, verifier TokenVerifier, logger *slog.Logger) *Server {
	return &Server{manager: manager, verifier: verifier, logger: logger}
}

// Connect implements the ConnectionService.Connect RPC: the first
// message must be an AuthenticateRequest, after which the connection is
// tracked in Manager until the stream ends.
func (s *Server) Connect(stream deployosv1.ConnectionService_ConnectServer) error {
	deviceID, userID, err := s.authenticate(stream)
	if err != nil {
		return err
	}

	sessionID := uuid.NewString()
	s.manager.Add(State{
		DeviceID:    deviceID,
		UserID:      userID,
		SessionID:   sessionID,
		ConnectedAt: time.Now(),
	})
	defer s.manager.Remove(deviceID)

	if err := stream.Send(&deployosv1.ConnectionEnvelope{
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

	// No message kinds beyond authentication are handled yet; future
	// features (heartbeats, commands) add cases here. For now this loop
	// exists purely to detect when the stream ends, by blocking on
	// Recv() until it returns an error - client disconnect, transport
	// failure, or context cancellation from a server shutdown all
	// surface the same way.
	for {
		if _, err := stream.Recv(); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
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
