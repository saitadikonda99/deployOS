package api

// SendCommandRequest is the JSON body for
// POST /api/v1/devices/{deviceID}/commands.
type SendCommandRequest struct {
	Kind string `json:"kind"`
}

// SendCommandResponse is a command's structured result, returned once
// the target device has responded (or the request has failed/timed
// out, reported via ErrorResponse instead).
type SendCommandResponse struct {
	CommandID string            `json:"command_id"`
	Success   bool              `json:"success"`
	Message   string            `json:"message"`
	Details   map[string]string `json:"details,omitempty"`
}
