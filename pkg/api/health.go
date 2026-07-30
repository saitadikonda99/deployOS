// Package api defines the HTTP request/response contracts shared between
// DeployOS servers (agent, control plane) and their callers (CLI,
// dashboard). It contains data shapes only, not handler implementations.
package api

// Status is the health of a DeployOS component.
type Status string

const (
	// StatusOK indicates the component is healthy.
	StatusOK Status = "ok"
	// StatusDegraded indicates the component is running but impaired.
	StatusDegraded Status = "degraded"
)

// HealthResponse is served from every DeployOS component's health endpoint.
type HealthResponse struct {
	Status Status `json:"status"`
}

// VersionResponse identifies the component and build serving a request.
type VersionResponse struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
