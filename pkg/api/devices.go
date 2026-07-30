package api

import "time"

// Device is the representation of a registered device returned by
// GET /api/v1/devices, e.g. to the dashboard.
type Device struct {
	ID              string    `json:"id"`
	Hostname        string    `json:"hostname"`
	OperatingSystem string    `json:"operating_system"`
	Architecture    string    `json:"architecture"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

// ListDevicesResponse is served from GET /api/v1/devices.
type ListDevicesResponse struct {
	Devices []Device `json:"devices"`
}
