package agent

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"

	"github.com/saitadikonda99/deployOS/internal/commandbus"
	"github.com/saitadikonda99/deployOS/internal/containers"
)

// pingExecutor answers PING with a static "pong" - just enough to prove
// the Command Bus round-trips a command to this agent and back.
func pingExecutor() commandbus.Executor {
	return commandbus.ExecutorFunc(func(_ context.Context, _ commandbus.Request) commandbus.Response {
		return commandbus.Response{Success: true, Message: "pong"}
	})
}

// getVersionExecutor answers GET_VERSION with the running agent's build
// version.
func getVersionExecutor() commandbus.Executor {
	return commandbus.ExecutorFunc(func(_ context.Context, _ commandbus.Request) commandbus.Response {
		return commandbus.Response{
			Success: true,
			Message: Version,
			Details: map[string]string{"version": Version},
		}
	})
}

// getInfoExecutor answers GET_INFO with the same system info the agent
// reports at registration.
func getInfoExecutor() commandbus.Executor {
	return commandbus.ExecutorFunc(func(_ context.Context, _ commandbus.Request) commandbus.Response {
		info, err := collectSystemInfo()
		if err != nil {
			return commandbus.Response{Success: false, Message: err.Error()}
		}
		return commandbus.Response{
			Success: true,
			Details: map[string]string{
				"hostname":         info.Hostname,
				"operating_system": info.OperatingSystem,
				"architecture":     info.Architecture,
				"cpu_cores":        strconv.Itoa(info.CPUCores),
				"memory_bytes":     strconv.FormatUint(info.MemoryBytes, 10),
				"deployos_version": Version,
			},
		}
	})
}

// listContainersExecutor answers LIST_CONTAINERS with every container
// runtime is aware of, JSON-encoded into a single Details entry - the
// Command Bus itself stays opaque to what a container is.
func listContainersExecutor(runtime containers.Runtime) commandbus.Executor {
	return commandbus.ExecutorFunc(func(ctx context.Context, _ commandbus.Request) commandbus.Response {
		list, err := runtime.ListContainers(ctx)
		if err != nil {
			return commandbus.Response{Success: false, Message: err.Error()}
		}

		payload, err := json.Marshal(list)
		if err != nil {
			return commandbus.Response{Success: false, Message: err.Error()}
		}
		return commandbus.Response{Success: true, Details: map[string]string{"containers": string(payload)}}
	})
}

// inspectContainerExecutor answers INSPECT_CONTAINER with the full
// detail of the container named by the request's "id" argument.
func inspectContainerExecutor(runtime containers.Runtime) commandbus.Executor {
	return commandbus.ExecutorFunc(func(ctx context.Context, req commandbus.Request) commandbus.Response {
		id := req.Arguments["id"]
		if id == "" {
			return commandbus.Response{Success: false, Message: "missing required argument: id"}
		}

		details, err := runtime.InspectContainer(ctx, id)
		if err != nil {
			return commandbus.Response{Success: false, Message: err.Error()}
		}

		payload, err := json.Marshal(details)
		if err != nil {
			return commandbus.Response{Success: false, Message: err.Error()}
		}
		return commandbus.Response{Success: true, Details: map[string]string{"container": string(payload)}}
	})
}

// newDispatcher builds a Dispatcher with every command this agent
// supports registered. Adding a new command means adding one more
// Register call here - nothing else.
func newDispatcher(logger *slog.Logger, runtime containers.Runtime) *commandbus.Dispatcher {
	d := commandbus.NewDispatcher(logger)
	d.Register(commandbus.KindPing, pingExecutor())
	d.Register(commandbus.KindGetVersion, getVersionExecutor())
	d.Register(commandbus.KindGetInfo, getInfoExecutor())
	d.Register(commandbus.KindListContainers, listContainersExecutor(runtime))
	d.Register(commandbus.KindInspectContainer, inspectContainerExecutor(runtime))
	return d
}
