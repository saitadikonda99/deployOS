// Package docker defines the container lifecycle operations DeployOS will
// perform on behalf of deployed applications. It contains an interface
// only - no implementation exists yet, and nothing in this package talks
// to a container runtime.
package docker

import "context"

// ContainerID identifies a container within whatever runtime backs Manager.
type ContainerID string

// ContainerStatus is the lifecycle state of a container.
type ContainerStatus string

const (
	// ContainerStatusRunning means the container is currently running.
	ContainerStatusRunning ContainerStatus = "running"
	// ContainerStatusStopped means the container exists but is not running.
	ContainerStatusStopped ContainerStatus = "stopped"
	// ContainerStatusUnknown means the status could not be determined.
	ContainerStatusUnknown ContainerStatus = "unknown"
)

// Container describes a single container managed by DeployOS.
type Container struct {
	ID     ContainerID
	Name   string
	Image  string
	Status ContainerStatus
}

// Manager represents the container operations DeployOS needs in order to
// deploy and manage applications. Implementations will wrap a specific
// container runtime (initially Docker); callers should depend on this
// interface rather than a concrete runtime.
type Manager interface {
	// ListContainers returns every container Manager is aware of.
	ListContainers(ctx context.Context) ([]Container, error)
	// StartContainer starts a previously created container.
	StartContainer(ctx context.Context, id ContainerID) error
	// StopContainer stops a running container.
	StopContainer(ctx context.Context, id ContainerID) error
	// RemoveContainer removes a stopped container.
	RemoveContainer(ctx context.Context, id ContainerID) error
}
