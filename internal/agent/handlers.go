package agent

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/saitadikonda99/deployOS/internal/commandbus"
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

// newDispatcher builds a Dispatcher with every command this agent
// supports registered. Adding a new command means adding one more
// Register call here - nothing else.
func newDispatcher(logger *slog.Logger) *commandbus.Dispatcher {
	d := commandbus.NewDispatcher(logger)
	d.Register(commandbus.KindPing, pingExecutor())
	d.Register(commandbus.KindGetVersion, getVersionExecutor())
	d.Register(commandbus.KindGetInfo, getInfoExecutor())
	return d
}
