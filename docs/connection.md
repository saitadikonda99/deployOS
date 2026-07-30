# The persistent agent connection

`internal/connection` implements `deployos.v1.ConnectionService` (see
[protocol.md](./protocol.md)): the persistent, authenticated,
bidirectional gRPC connection between an agent and the control plane.
This is transport and session management only - no heartbeats, command
execution, Docker integration, or deployments ride on it yet. Those are
expected to build on top of this package, not fork it.

## Connection lifecycle

```
Agent                                          Control Plane
  │                                                  │
  │  dial (insecure gRPC, cmd/agent's                │
  │  GRPCServerAddr)                                 │
  │ ───────────────────────────────────────────────▶ │
  │                                                  │
  │  ConnectionEnvelope{AuthenticateRequest}         │
  │ ───────────────────────────────────────────────▶ │  verify device token
  │                                                  │  (internal/devices.
  │  ConnectionEnvelope{AuthenticateResponse}        │   JWTTokenIssuer.Verify)
  │ ◀─────────────────────────────────────────────── │
  │                                                  │  Manager.Add(...)
  │  (stream stays open; nothing else defined yet)   │
  │ ◀────────────────────────────────────────────── ▶ │
  │                                                  │
  │  connection lost (either side)                   │
  │ - - - - - - - - - - - - - - - - - - - - - - - -  │  Manager.Remove(...)
  │                                                  │
  │  reconnect with backoff                          │
  │ ───────────────────────────────────────────────▶ │
```

1. `internal/agent.Agent.Run` loads the agent's persisted device ID
   (`internal/agent/identity.go`) and starts
   `internal/connection.Client.Run` alongside the HTTP health server,
   coordinated with an `errgroup` so both shut down together.
2. `Client` dials the control plane's gRPC address
   (`agent.grpc_server_addr`) with insecure transport credentials (see
   [Known limitations](#known-limitations)) and opens the
   `ConnectionService.Connect` stream.
3. The first message the client sends is always an
   `AuthenticateRequest`, carrying the device's token
   (`Connection.device_token`) and identity (`Device`).
4. `internal/connection.Server.Connect` reads that first message,
   verifies the token, and either:
   - registers the device in `Manager` and replies with
     `AuthenticateResponse{authenticated: true, session_id: ...}`, or
   - replies with `AuthenticateResponse{authenticated: false, error: "..."}`
     and ends the stream.
5. Once authenticated, both sides simply block on `Recv()` - there is no
   other message kind defined yet. The stream ending, for any reason
   (client disconnect, network failure, or server shutdown), is what
   `Recv()` returning an error signals.
6. `Server.Connect`'s `defer manager.Remove(deviceID)` runs the instant
   the stream ends, so the Connection Manager never reports a dead
   connection as alive.

## Authentication flow

The token an agent presents is the one issued at device registration
(see [device-registration.md](./device-registration.md)): an HS256 JWT
signed with `device_token.secret`, containing the device ID (`sub`) and
owning user ID (`user_id`).

`internal/devices.JWTTokenIssuer.Verify` (which also implements
`internal/connection.TokenVerifier`, so `internal/connection` never
imports `internal/devices`) checks:

1. **Signature** - signed with the same secret the control plane holds.
2. **Expiry** - `golang-jwt/v5` rejects an expired token automatically.
3. **Device ID match** - `Server.authenticate` additionally checks that
   the `Device.id` in the request matches the token's subject, rejecting
   a token being presented for a different device than it was issued
   for.

Any failure collapses to the same generic rejection
(`AuthenticateResponse{authenticated: false}` /
`codes.Unauthenticated`) - the agent doesn't need to distinguish
"expired" from "wrong device" from "bad signature" to know it should
back off and, if the problem is a stale token, wait for a fresh
registration to replace it.

## Reconnection strategy

`internal/connection.Client.Run` reconnects with exponential backoff,
starting at `DefaultInitialBackoff` (1s) and doubling on each failed
attempt up to `DefaultMaxBackoff` (30s), with roughly +/-20% jitter so
many agents reconnecting after a control-plane restart don't all retry
in lockstep. The backoff resets to its initial value after any attempt
that authenticates successfully - a connection that stays up for a while
before dropping isn't penalized with a long wait next time.

The client re-reads the device token from disk (via the `TokenSource`
function passed to `Run`) on every single connect attempt, rather than
capturing it once at startup. This means:

- A token that hasn't been written yet (registration still in progress
  or failing) simply keeps the client retrying - once registration
  succeeds, the very next attempt uses the new token.
- A token replaced by a fresh registration takes effect on the next
  reconnect, without an agent restart.

Every failure is retried; nothing about a rejected connection is
treated as fatal. `Run` only returns (with a nil error) when its context
is canceled - graceful shutdown, not a connection outcome.

## Connection Manager responsibilities

`internal/connection.Manager` is a thread-safe, in-memory registry -
nothing more:

- `Add`/`Remove`/`IsConnected`/`Get`/`List`/`Count`, all safe for
  concurrent use (see `manager_test.go`'s `-race`-checked concurrency
  test).
- It holds no persistent state and never touches Postgres/Supabase - a
  device that has never connected simply isn't in it, and a
  disconnected one is removed outright, not marked "offline" and kept
  around.
- It's deliberately generic (`State{DeviceID, UserID, SessionID, ConnectedAt}`)
  so it can be reused, unchanged, by whatever needs to
  know "is this device currently reachable" - which today is just
  `GET /api/v1/devices` (via `internal/devices.ConnectionStatusProvider`,
  implemented by `*Manager`), and in the future will include command
  delivery, log streaming, and metrics collection picking a live stream
  to use.

## Exposing connection status

`GET /api/v1/devices` (see [device-registration.md](./device-registration.md))
now reports each device's `status` as `"connected"` or `"disconnected"`,
computed at request time from `Manager.IsConnected`, rather than a
stored value - a device row in Postgres has no "online" column. The
dashboard's Devices page renders this directly.

## Known limitations

- **No TLS.** The gRPC connection uses insecure transport credentials on
  both sides. This is a foundation-stage gap, not a design decision -
  production deployments need TLS between agent and control plane
  before this ships for real. Adding it is scoped to a future change so
  it can be done properly (certificate provisioning, rotation), not
  bolted on ad hoc here.
- **No device-existence check.** `Server` verifies a token
  cryptographically but doesn't query Postgres to confirm the device
  still exists there (e.g. after being deleted). Adding that is future
  work if it turns out to matter in practice.
- **Server shutdown isn't instant for connected agents.**
  `grpc.Server.GracefulStop()` waits for existing streams to end
  naturally, which a persistent connection never does on its own;
  `internal/runtime.GRPCServer` falls back to a hard `Stop()` after
  `DefaultShutdownTimeout` (10s, shared with `HTTPServer`). Connected
  agents detect this as a lost connection and reconnect via backoff once
  the control plane comes back - this is deliberate (bounded
  control-plane restart time, self-healing agents), not a bug.
