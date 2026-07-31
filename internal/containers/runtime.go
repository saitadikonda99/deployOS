// Package containers defines the Runtime abstraction: the interface
// DeployOS uses to observe containers on a managed machine, independent
// of whatever container engine is actually installed there.
//
// Nothing outside this package's docker subpackage knows Docker exists.
// internal/agent registers commands against the Runtime interface only,
// so adding a Podman, containerd, or Kubernetes provider later means
// writing a new package that implements Runtime - internal/commandbus,
// internal/agent's dispatcher wiring, and the dashboard never change.
//
// This phase only implements observability (ListContainers,
// InspectContainer) - no lifecycle operations (start, stop, create,
// delete) exist yet.
package containers

import (
	"context"
	"errors"
	"time"
)

// ErrContainerNotFound is returned by InspectContainer when no container
// with the given ID exists.
var ErrContainerNotFound = errors.New("container not found")

// Container is a summary of a single container, as returned by
// ListContainers.
type Container struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Image   string    `json:"image"`
	Status  string    `json:"status"`
	State   string    `json:"state"`
	Created time.Time `json:"created"`
}

// Mount describes a single filesystem mount into a container.
type Mount struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Mode        string `json:"mode"`
}

// Port describes a single published port mapping.
type Port struct {
	ContainerPort int    `json:"container_port"`
	HostPort      int    `json:"host_port"`
	Protocol      string `json:"protocol"`
}

// ContainerDetails is the full detail of a single container, as returned
// by InspectContainer.
type ContainerDetails struct {
	Container
	Command      []string `json:"command"`
	Env          []string `json:"env"`
	Mounts       []Mount  `json:"mounts"`
	Ports        []Port   `json:"ports"`
	Networks     []string `json:"networks"`
	RestartCount int      `json:"restart_count"`
}

// Runtime is the set of container operations DeployOS needs from a
// container engine. Implementations wrap a specific engine (initially
// Docker, see the docker subpackage); every other layer of DeployOS
// depends on this interface, never on a concrete engine.
type Runtime interface {
	// ListContainers returns every container the runtime is aware of,
	// running or not.
	ListContainers(ctx context.Context) ([]Container, error)
	// InspectContainer returns full detail for a single container. It
	// returns ErrContainerNotFound if id doesn't exist.
	InspectContainer(ctx context.Context, id string) (ContainerDetails, error)
}
