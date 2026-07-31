package commandbus

import (
	"context"

	deployosv1 "github.com/saitadikonda99/deployOS/gen/go/deployos/v1"
)

// WireHandler adapts dispatcher to internal/connection.CommandHandler's
// signature, so an agent only has to wire
// `client.OnCommand(commandbus.WireHandler(dispatcher))` - all proto
// conversion lives here, not in agent-specific code.
func WireHandler(dispatcher *Dispatcher) func(ctx context.Context, cmd *deployosv1.Command) *deployosv1.CommandResult {
	return func(ctx context.Context, cmd *deployosv1.Command) *deployosv1.CommandResult {
		return responseToProto(dispatcher.Dispatch(ctx, requestFromProto(cmd)))
	}
}
