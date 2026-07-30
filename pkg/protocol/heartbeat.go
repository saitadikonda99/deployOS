// Package protocol defines the wire message shapes exchanged between the
// DeployOS control plane and node agents. It defines message shapes only;
// transport (HTTP today, potentially gRPC later) is decided by whichever
// side is calling it.
package protocol

import (
	"time"

	"github.com/saitadikonda99/deployOS/pkg/types"
)

// Heartbeat is a periodic liveness signal sent from an agent to the
// control plane.
type Heartbeat struct {
	AgentID      types.AgentID `json:"agent_id"`
	AgentVersion string        `json:"agent_version"`
	// Timestamp is the Unix time, in seconds, at which the heartbeat was
	// produced.
	Timestamp int64 `json:"timestamp"`
}

// NewHeartbeat builds a Heartbeat stamped with the current time.
func NewHeartbeat(agentID types.AgentID, agentVersion string) Heartbeat {
	return Heartbeat{
		AgentID:      agentID,
		AgentVersion: agentVersion,
		Timestamp:    time.Now().Unix(),
	}
}
