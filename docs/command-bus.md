# The Command Bus

The Command Bus is generic request/response command routing between the
control plane and an agent, implemented in `internal/commandbus` on top
of the persistent gRPC connection (see [connection.md](./connection.md)).
It is the communication layer every future DeployOS capability - Docker
management, deployments, log streaming, terminal access, file transfer -
is built on. `PING`, `GET_VERSION`, and `GET_INFO` proved the bus works
end to end; `LIST_CONTAINERS` and `INSPECT_CONTAINER` (see
[runtime.md](./runtime.md)) are the first commands built on top of it
for a real capability - observing containers through the new Runtime
abstraction.

## Why it rides the existing connection

`proto/deployos/v1/command.proto` originally sketched a standalone
`CommandService` with a unary `Deliver` RPC. It turned out to be
unusable: a unary RPC has to be initiated by whichever side holds the
gRPC client, and in DeployOS that's always the agent (it dials the
control plane; the control plane never dials an agent, since agents are
commonly behind NAT). `CommandService` and its `Deliver` RPC have been
removed from the proto - `Command` and `CommandResult` remain as
messages, now sent as new `ConnectionEnvelope` oneof cases
(`command_request`, `command_response`) over `ConnectionService`'s
existing bidirectional stream. This is exactly what that oneof was
designed for (see [protocol.md](./protocol.md)'s versioning strategy):
new message kinds join the stream without a new RPC.

## Components

```
Control Plane                                          Agent
┌─────────────────────────┐                    ┌─────────────────────────┐
│ commandbus.Service       │                    │ connection.Client        │
│  - Send(ctx, device, req)│──Command─────────▶ │  - OnCommand callback    │
│  - HandleResult(...)     │◀──CommandResult──── │                         │
└───────────┬─────────────┘                    └───────────┬─────────────┘
            │ Sender                                        │ commandbus.WireHandler
            ▼                                                ▼
  connection.Server                                commandbus.Dispatcher
  (routes by device ID,                             - Register(kind, executor)
   Send/OnMessage)                                   - Dispatch(ctx, req) → resp
                                                                │
                                        ┌───────────┼───────────┬───────────────┬──────────────────┐
                                        ▼           ▼           ▼               ▼                  ▼
                                      PING   GET_VERSION   GET_INFO   LIST_CONTAINERS   INSPECT_CONTAINER
                                    (executors registered in internal/agent/handlers.go;
                                     the last two call internal/containers.Runtime - see runtime.md)
```

- **`commandbus.Service`** (control plane): creates a command, sends it
  to a specific device, and blocks until the matching result arrives or
  the caller's context is done.
- **`connection.Server`** (control plane, extended for this feature):
  routes an outbound envelope to the right device's stream (`Send`) and
  forwards every inbound non-auth envelope to a registered callback
  (`OnMessage`) - `Service.HandleResult` is that callback.
- **`connection.Client`** (agent, extended for this feature): invokes a
  registered `CommandHandler` whenever the server sends a command over
  the stream, and sends the resulting `CommandResult` back - each
  command runs in its own goroutine, so multiple commands execute
  concurrently rather than queuing behind each other.
- **`commandbus.Dispatcher`** (agent): routes a command to whichever
  `Executor` is registered for its `kind`.
- **`commandbus.WireHandler`** adapts a `Dispatcher` to
  `connection.Client`'s `CommandHandler` signature, so all proto
  conversion lives in `internal/commandbus`, not in agent-specific code.

Neither `Service` nor `Dispatcher` knows anything about Docker,
deployments, or any other feature - only the commands
`internal/agent/handlers.go` explicitly registers exist. That's true
even for `LIST_CONTAINERS`/`INSPECT_CONTAINER`: their executors call
`internal/containers.Runtime`, a Docker-agnostic interface (see
[runtime.md](./runtime.md)), so `internal/commandbus` still never
imports anything container- or Docker-specific.

## Handler registration

Adding a new command is exactly two steps:

1. **Write an `Executor`** (a function is enough, via
   `commandbus.ExecutorFunc`):

   ```go
   func myExecutor() commandbus.Executor {
       return commandbus.ExecutorFunc(func(ctx context.Context, req commandbus.Request) commandbus.Response {
           return commandbus.Response{Success: true, Message: "..."}
       })
   }
   ```

2. **Register it** in `internal/agent/handlers.go`'s `newDispatcher`:

   ```go
   d.Register("MY_COMMAND", myExecutor())
   ```

Nothing else changes - not the wire format, not `connection.Client`, not
`commandbus.Service`. An `Executor` is independently testable with a
plain `context.Context` and `commandbus.Request`; it never needs a real
connection to test (see `internal/agent/handlers_test.go`).

## Request/response lifecycle

1. An HTTP caller (the dashboard, via `internal/commandbus.Handler`)
   posts `{"kind": "PING"}` to
   `POST /api/v1/devices/{deviceID}/commands`.
2. The handler authenticates the caller, confirms the device belongs to
   them (`OwnershipChecker`, implemented by `devices.Service.IsOwner`),
   and calls `Service.Send` with a bounded-timeout context.
3. `Send` generates a command ID, registers it as pending, and sends a
   `Command` envelope to the device via `Sender.Send`
   (`connection.Server`). If the device isn't connected, this fails
   immediately with `ErrDeviceNotConnected` - no waiting.
4. The agent's `connection.Client` receives the envelope, and (via
   `commandbus.WireHandler`) calls `Dispatcher.Dispatch` in its own
   goroutine.
5. `Dispatch` runs the registered `Executor` (or produces a structured
   failure response for an unknown `kind`, or recovers from a panic -
   either way, always a `Response`) and the client sends it back as a
   `CommandResult` envelope.
6. `connection.Server`'s read loop forwards that envelope to
   `Service.HandleResult`, which looks up the pending request by command
   ID and delivers the result, unblocking the waiting `Send` call.
7. The HTTP handler returns the result as JSON, or a `503`/`504` if the
   device wasn't connected or didn't respond in time.

## Command correlation

Because a single persistent connection can carry many commands over its
lifetime - and, with concurrent dispatch, several in flight to the same
device at once - `Service` tracks pending requests in a map keyed by the
command ID it generated:

```go
type pendingRequest struct {
    resultCh chan *deployosv1.CommandResult
}
```

`Send` registers the pending request before sending, then blocks on
`resultCh` (or the caller's context). `HandleResult` looks up the result
by `CommandResult.command_id`, delivers it to that specific channel, and
removes the entry - a result for an ID that's unknown or already
resolved is logged and dropped rather than panicking or blocking
anything. This is what lets two concurrent `Send` calls to the same
device each get back their own result, regardless of the order the
device happens to respond in (`TestServiceResponseCorrelationWithConcurrentCommands`
in `internal/commandbus/service_test.go` exercises exactly this by
delivering results in reverse order).

## Timeouts

`Send` has no timeout logic of its own - it blocks on `ctx.Done()` like
everything else in this codebase. The HTTP handler is what supplies a
bounded context (`defaultSendTimeout`, 15s); a `context.DeadlineExceeded`
becomes a `504 Gateway Timeout` response.

## Testing

- `internal/connection`'s `routing_test.go` covers `Send`/`OnMessage`/
  `OnCommand` at the transport level (including "device not connected"
  and a full command round trip over a real stream).
- `internal/commandbus`'s `dispatcher_test.go` covers successful
  dispatch, unknown commands, panic recovery, and concurrent dispatch
  (`-race`).
- `service_test.go` covers successful `Send`, device-not-connected,
  timeout, correlation under concurrency, and dropping a result for an
  unknown/late command ID.
- `handler_test.go` covers the HTTP layer: auth, ownership, invalid
  payloads, device-not-connected, and a full success path.
- `integration_test.go` wires the real `connection.Server`/`Client` and
  `commandbus.Service`/`Dispatcher` together over a live TCP connection -
  the same shape `cmd/server`/`cmd/agent` use in production - for a
  `PING` round trip, an unknown command, multiple concurrent commands,
  and `LIST_CONTAINERS`/`INSPECT_CONTAINER` (proving `Request.Arguments`
  survives the wire).
- Manually verified against the real compiled `deployos-agent` binary:
  `PING`, `GET_VERSION`, and `GET_INFO` all returned correct, real
  results (actual hostname/CPU/memory), and an unrecognized command kind
  came back as a structured failure rather than a crash or a hang.

## Known limitations

- Five commands exist: `PING`, `GET_VERSION`, `GET_INFO`,
  `LIST_CONTAINERS`, and `INSPECT_CONTAINER` (see
  [runtime.md](./runtime.md)). Everything else (deployments, heartbeats,
  metrics, log streaming, terminal access, file transfer, and any
  container lifecycle operation - start, stop, create, delete) is
  explicitly out of scope so far.
- `Send` waits for exactly one result per command. A future command that
  needs to stream partial progress (e.g. a long-running deployment)
  will need its own envelope kind(s) - this bus doesn't preclude that,
  but doesn't implement it either.
