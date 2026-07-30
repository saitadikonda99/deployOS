// Package devices implements device registration: agents register
// themselves with the control plane, which persists them in Supabase and
// issues a signed device token. It follows a repository/service/handler
// split - Repository stores devices, Service owns registration and
// validation rules, Handler adapts that to HTTP.
package devices

import (
	"time"

	"github.com/saitadikonda99/deployOS/pkg/types"
)

// StatusRegistered is the status assigned to a device on registration.
// DeployOS does not yet track liveness (that's the future heartbeat
// feature), so this is the only status a device can currently have.
const StatusRegistered = "registered"

// Device is the domain representation of a registered machine, owned by
// exactly one user.
type Device struct {
	ID              types.AgentID
	UserID          string
	Hostname        string
	OperatingSystem string
	Architecture    string
	CPUCores        int
	MemoryBytes     uint64
	DeployOSVersion string
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
