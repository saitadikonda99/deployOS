// Package connection implements the persistent, authenticated gRPC
// connection between a DeployOS agent and the control plane, defined by
// deployos.v1.ConnectionService (see proto/deployos/v1/connection.proto
// and docs/connection.md). It owns connection lifecycle only: dialing,
// authenticating, reconnecting, and tracking which devices are
// currently connected. It has no knowledge of Docker, deployments, or
// monitoring, and no business logic rides on it yet - heartbeats,
// command delivery, and everything else future features need are
// expected to build on top of this package rather than fork it.
package connection

import (
	"context"

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
