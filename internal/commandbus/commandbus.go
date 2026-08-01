// Package commandbus implements the DeployOS Command Bus: generic
// request/response command routing between the control plane and an
// agent, over the persistent connection (internal/connection). See
// docs/command-bus.md.
//
// This package has two halves, used by different processes:
//   - Service (control plane): creates commands, sends them to a
//     specific device, correlates the response by command ID, and
//     enforces a timeout via the caller's context.
//   - Dispatcher (agent): routes a received command to whichever
//     Handler is registered for its kind, gracefully handling unknown
//     commands and recovering from a handler panic so a single bad
//     command can never crash the agent.
//
// Neither half knows anything about Docker, deployments, heartbeats,
// metrics, or log streaming - only command kinds a handler is
// explicitly registered for exist; adding one is "write a Handler,
// register it," nothing else.
package commandbus

import (
	deployosv1 "github.com/saitadikonda99/deployOS/gen/go/deployos/v1"
)

// Kind values for the commands implemented so far. A command's kind is
// a free-form string on the wire (see proto/deployos/v1/command.proto),
// not a proto enum, so adding a new one is never a breaking protocol
// change.
const (
	KindPing             = "PING"
	KindGetVersion       = "GET_VERSION"
	KindGetInfo          = "GET_INFO"
	KindListContainers   = "LIST_CONTAINERS"
	KindInspectContainer = "INSPECT_CONTAINER"
)

// Request is a command's kind and arguments, independent of how it
// arrived (the persistent gRPC connection today).
type Request struct {
	ID        string
	Kind      string
	Arguments map[string]string
}

// Response is a command's structured result.
type Response struct {
	CommandID string
	Success   bool
	// Message is a human-readable outcome description, set on both
	// success and failure.
	Message string
	Details map[string]string
}

func requestFromProto(cmd *deployosv1.Command) Request {
	return Request{
		ID:        cmd.GetId(),
		Kind:      cmd.GetKind(),
		Arguments: metadataToMap(cmd.GetArguments()),
	}
}

func responseToProto(resp Response) *deployosv1.CommandResult {
	return &deployosv1.CommandResult{
		CommandId: resp.CommandID,
		Success:   resp.Success,
		Message:   resp.Message,
		Details:   metadataFromMap(resp.Details),
	}
}

func metadataToMap(m *deployosv1.Metadata) map[string]string {
	if m == nil {
		return nil
	}
	return m.GetEntries()
}

func metadataFromMap(m map[string]string) *deployosv1.Metadata {
	if len(m) == 0 {
		return nil
	}
	return &deployosv1.Metadata{Entries: m}
}
