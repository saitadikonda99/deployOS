package protocol

import (
	"time"

	"github.com/saitadikonda99/deployOS/pkg/types"
)

// DeviceRegisterRequest is sent by an agent to the control plane's
// POST /api/v1/devices/register endpoint. The agent generates DeviceID
// itself and persists it locally, so the same ID is reused across
// restarts and re-registrations are idempotent upserts.
type DeviceRegisterRequest struct {
	DeviceID        types.AgentID `json:"device_id"`
	Hostname        string        `json:"hostname"`
	OperatingSystem string        `json:"operating_system"`
	Architecture    string        `json:"architecture"`
	CPUCores        int           `json:"cpu_cores"`
	MemoryBytes     uint64        `json:"memory_bytes"`
	DeployOSVersion string        `json:"deployos_version"`
}

// DeviceRegisterResponse is the control plane's response to a successful
// device registration. The agent persists Token locally and presents it
// on future authenticated requests.
type DeviceRegisterResponse struct {
	DeviceID  types.AgentID `json:"device_id"`
	Token     string        `json:"token"`
	ExpiresAt time.Time     `json:"expires_at"`
}
