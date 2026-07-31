package connection

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	deployosv1 "github.com/saitadikonda99/deployOS/gen/go/deployos/v1"
	"github.com/saitadikonda99/deployOS/pkg/types"
)

// DefaultInitialBackoff and DefaultMaxBackoff bound Client's reconnect
// delay: it starts at DefaultInitialBackoff and doubles (with jitter) up
// to DefaultMaxBackoff after each failed attempt, resetting after any
// attempt that authenticates successfully.
const (
	DefaultInitialBackoff = 1 * time.Second
	DefaultMaxBackoff     = 30 * time.Second
)

// DeviceInfo is the identity and system info an agent presents when
// authenticating a connection.
type DeviceInfo struct {
	ID              types.AgentID
	Hostname        string
	OperatingSystem string
	Architecture    string
	DeployOSVersion string
}

// TokenSource returns the device token to authenticate with. It's a
// function rather than a fixed string so Client always uses whatever
// token is currently valid - e.g. re-reading it from disk on every
// (re)connect attempt, in case it was replaced since the last one.
type TokenSource func() (string, error)

// Client maintains a persistent, authenticated gRPC connection to the
// control plane from the agent's side: dialing, authenticating, and
// reconnecting with exponential backoff whenever the connection is
// lost. It has no knowledge of what rides on the connection once
// authenticated - that's for future features to build on.
type Client struct {
	serverAddr     string
	logger         *slog.Logger
	initialBackoff time.Duration
	maxBackoff     time.Duration

	connected atomic.Bool
}

// NewClient builds a Client that will dial serverAddr.
func NewClient(serverAddr string, logger *slog.Logger) *Client {
	return &Client{
		serverAddr:     serverAddr,
		logger:         logger,
		initialBackoff: DefaultInitialBackoff,
		maxBackoff:     DefaultMaxBackoff,
	}
}

// Connected reports whether the client currently holds an authenticated
// connection to the control plane.
func (c *Client) Connected() bool {
	return c.connected.Load()
}

// Run dials the control plane and authenticates as device, then blocks,
// reconnecting with exponential backoff whenever the connection is lost,
// until ctx is canceled. It always returns nil on a canceled context
// (graceful shutdown); every connection failure is retried rather than
// treated as fatal.
func (c *Client) Run(ctx context.Context, device DeviceInfo, token TokenSource) error {
	backoff := c.initialBackoff

	for ctx.Err() == nil {
		authenticated, err := c.connectOnce(ctx, device, token)
		c.connected.Store(false)

		if err != nil && ctx.Err() == nil {
			c.logger.Error("connection to control plane ended", slog.Any("error", err))
		}

		if ctx.Err() != nil {
			return nil
		}

		if authenticated {
			backoff = c.initialBackoff
		}

		wait := jitter(backoff)
		c.logger.Info("reconnecting to control plane", slog.Duration("in", wait))

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(wait):
		}

		backoff = min(backoff*2, c.maxBackoff)
	}

	return nil
}

// connectOnce dials the control plane, authenticates once, and then
// blocks until the connection ends. authenticated reports whether the
// AuthenticateResponse said authenticated = true, regardless of what
// happened afterward - Run uses it to decide whether to reset backoff.
func (c *Client) connectOnce(ctx context.Context, device DeviceInfo, token TokenSource) (authenticated bool, err error) {
	tok, err := token()
	if err != nil {
		return false, fmt.Errorf("loading device token: %w", err)
	}

	c.logger.Info("connecting to control plane", slog.String("addr", c.serverAddr))

	conn, err := grpc.NewClient(c.serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return false, fmt.Errorf("dialing control plane: %w", err)
	}
	defer func() { _ = conn.Close() }()

	client := deployosv1.NewConnectionServiceClient(conn)
	stream, err := client.Connect(ctx)
	if err != nil {
		return false, fmt.Errorf("opening connection stream: %w", err)
	}

	if err := stream.Send(&deployosv1.ConnectionEnvelope{
		Payload: &deployosv1.ConnectionEnvelope_AuthenticateRequest{
			AuthenticateRequest: &deployosv1.AuthenticateRequest{
				Connection: &deployosv1.Connection{
					DeviceToken:     tok,
					ProtocolVersion: ProtocolVersion,
				},
				Device: &deployosv1.Device{
					Id:              device.ID.String(),
					Hostname:        device.Hostname,
					OperatingSystem: device.OperatingSystem,
					Architecture:    device.Architecture,
					DeployosVersion: device.DeployOSVersion,
				},
			},
		},
	}); err != nil {
		return false, fmt.Errorf("sending authenticate request: %w", err)
	}

	envelope, err := stream.Recv()
	if err != nil {
		return false, fmt.Errorf("receiving authenticate response: %w", err)
	}

	resp := envelope.GetAuthenticateResponse()
	if resp == nil {
		return false, errors.New("expected an authenticate response as the first message")
	}
	if !resp.GetAuthenticated() {
		return false, fmt.Errorf("control plane rejected authentication: %s", resp.GetError())
	}

	c.connected.Store(true)
	c.logger.Info("connected to control plane", slog.String("session_id", resp.GetSessionId()))

	for {
		if _, err := stream.Recv(); err != nil {
			if errors.Is(err, io.EOF) {
				return true, nil
			}
			return true, err
		}
	}
}

// jitter randomizes d by roughly +/-20%, to keep many agents reconnecting
// at once (e.g. after a control plane restart) from all retrying in
// lockstep.
func jitter(d time.Duration) time.Duration {
	factor := 0.8 + rand.Float64()*0.4
	return time.Duration(float64(d) * factor)
}
