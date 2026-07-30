// Package types holds foundational value types shared across DeployOS
// components. It intentionally contains no behavior beyond simple
// validation - it exists so that every package (agent, control plane,
// CLI) refers to the same definition of an agent ID, a version string,
// or a node role, instead of each redefining its own.
package types

import (
	"errors"
	"fmt"
)

// AgentID uniquely identifies a single DeployOS node agent within a fleet.
type AgentID string

// ErrEmptyAgentID is returned when an AgentID is validated and found empty.
var ErrEmptyAgentID = errors.New("agent id must not be empty")

// Validate reports whether the AgentID is well-formed.
func (id AgentID) Validate() error {
	if id == "" {
		return ErrEmptyAgentID
	}
	return nil
}

func (id AgentID) String() string {
	return string(id)
}

// NodeRole describes the part a machine plays within a DeployOS fleet.
type NodeRole string

const (
	// NodeRoleAgent is a machine that runs workloads and reports to a
	// control plane.
	NodeRoleAgent NodeRole = "agent"
	// NodeRoleControlPlane is the machine (or machines) holding fleet
	// state and coordinating agents.
	NodeRoleControlPlane NodeRole = "control-plane"
)

// Version is a semantic-version-shaped build identifier reported by
// DeployOS binaries.
type Version struct {
	// Number is the semantic version, e.g. "0.1.0".
	Number string
	// Commit is the git commit the binary was built from, if known.
	Commit string
}

func (v Version) String() string {
	if v.Commit == "" {
		return v.Number
	}
	return fmt.Sprintf("%s (%s)", v.Number, v.Commit)
}
