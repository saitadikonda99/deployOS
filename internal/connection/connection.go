// Package connection implements the persistent, authenticated gRPC
// connection between a DeployOS agent and the control plane, defined by
// deployos.v1.ConnectionService (see proto/deployos/v1/connection.proto
// and docs/connection.md). It owns connection lifecycle and message
// routing only: dialing, authenticating, reconnecting, tracking which
// devices are currently connected, and delivering envelope payloads to
// and from a specific device's stream. It has no knowledge of Docker,
// deployments, or monitoring, and no business logic of its own - the
// Command Bus (see docs/command-bus.md) is the first feature built on
// top of it, and future features (heartbeats, metrics, log streaming)
// are expected to do the same rather than fork this package.
package connection

import (
	"context"
	"errors"

	deployosv1 "github.com/saitadikonda99/deployOS/gen/go/deployos/v1"
	"github.com/saitadikonda99/deployOS/pkg/types"
)

// ProtocolVersion identifies the version of the DeployOS protocol this
// build speaks. Sent by the agent in every AuthenticateRequest.
const ProtocolVersion = "v1"

// TokenVerifier validates a device token and reports who it belongs to.
// internal/devices.JWTTokenIssuer implements this; Server depends only
// on this interface, not on internal/devices, so this package stays
// decoupled from how tokens are issued or stored.
type TokenVerifier interface {
	Verify(ctx context.Context, token string) (deviceID types.AgentID, userID string, err error)
}

// ErrDeviceNotConnected is returned by Server.Send when the target
// device has no active connection.
var ErrDeviceNotConnected = errors.New("device is not connected")

// IncomingHandler processes an envelope payload received from an
// authenticated device that isn't part of the authentication handshake
// (e.g. a command result). Server invokes it for every such payload;
// implementations that don't recognize a given payload kind should
// ignore it, so new kinds can be added without touching existing
// handlers.
type IncomingHandler func(deviceID types.AgentID, envelope *deployosv1.ConnectionEnvelope)
