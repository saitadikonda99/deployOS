# Architecture

DeployOS standardizes its entire backend on Go. The only TypeScript left in
the repository is the operator-facing dashboard, in
[`apps/dashboard`](../apps/dashboard); everything that manages a fleet or
runs on a managed machine is Go, under [`cmd/`](../cmd),
[`internal/`](../internal), and [`pkg/`](../pkg).

## Components

| Component                    | Language   | Responsibility                                                                                                                   |
| ---------------------------- | ---------- | -------------------------------------------------------------------------------------------------------------------------------- |
| `cmd/server`                 | Go         | Control plane: source of truth for cluster state; exposes the API the dashboard and agents use                                   |
| `cmd/agent`                  | Go         | Runs on each managed machine; registers itself, reports health, and executes commands from the control plane                     |
| `cmd/cli`                    | Go         | `deployos` CLI - operator entry point (version, doctor, agent management)                                                        |
| `apps/dashboard`             | TypeScript | Web UI for operators to manage deployments, secrets, monitoring, and send device commands                                        |
| `internal/devices`           | Go         | Device registration: repository/service/handler for `POST /api/v1/devices/register` and `GET /api/v1/devices`                    |
| `internal/auth`              | Go         | Authenticates operators against Supabase Auth on the control plane's behalf                                                      |
| `internal/connection`        | Go         | Persistent authenticated gRPC connection (client + server + in-memory Connection Manager) - see [connection.md](./connection.md) |
| `internal/commandbus`        | Go         | Command Bus: request/response command routing over the persistent connection - see [command-bus.md](./command-bus.md)            |
| `internal/containers`        | Go         | Runtime abstraction: engine-agnostic container observability interface - see [runtime.md](./runtime.md)                          |
| `internal/containers/docker` | Go         | The first Runtime provider, talking to the Docker Engine API over its unix socket                                                |
| `pkg/protocol`, `pkg/types`  | Go         | Wire types and value types shared between the control plane and agents (HTTP)                                                    |
| `pkg/api`                    | Go         | HTTP request/response contracts shared by every Go server                                                                        |
| `gen/go/deployos/v1`         | Go         | Generated Protocol Buffers/gRPC types (see [protocol.md](./protocol.md))                                                         |

The control plane is the only component with Supabase credentials; see
[device-registration.md](./device-registration.md) for how the agent,
control plane, and Supabase interact for that feature specifically.

## Package layout

- **`cmd/`** holds only entry points (`main` packages) - argument parsing,
  wiring, and calling into `internal/`. No business logic lives here.
- **`internal/`** holds the implementation, split by concern
  (`config`, `logging`, `runtime`, `monitoring`, `agent`, `connection`,
  `commandbus`, `containers` (+ its `docker` provider subpackage),
  `devices`, `auth`, `scheduler`, `discovery`, `secrets`, ...). It is
  unimportable outside this module, which is exactly the point: these
  are DeployOS's own internals, not a public API.
- **`pkg/`** holds the small amount of Go that's meant to be shared beyond
  this module's own binaries - wire types and API contracts - so it stays
  intentionally minimal.
- **`proto/`** holds the Protocol Buffers source for the agent <->
  control-plane gRPC protocol; **`gen/`** holds the Go code generated
  from it, which `internal/connection` implements against - see
  [protocol.md](./protocol.md) and [connection.md](./connection.md).

## Why this split

The control plane, agent, and CLI are all infrastructure processes with the
same requirements: predictable resource usage, fast startup, and safe
unattended operation. Standardizing them on Go keeps one toolchain, one set
of idioms, and one dependency-management story across all three, rather than
splitting infrastructure code across two languages.

The dashboard is a product surface that iterates on UI, not infrastructure
behavior, so it stays on the TypeScript/Next.js toolchain that's better
suited to that.

`pkg/protocol` is the HTTP-era contract between the control plane and
agents: it defines message shapes only, not transport. The Protocol
Buffers definitions in `proto/` (see [protocol.md](./protocol.md)) are the
next evolution of that same idea, for the persistent, bidirectional gRPC
connection `internal/connection` implements (see
[connection.md](./connection.md)). `internal/commandbus` (see
[command-bus.md](./command-bus.md)) is the first thing built on top of
that connection - generic command routing, with no knowledge of what a
command actually does. Future features (heartbeats, metrics, log
streaming) are expected to build on the connection the same way, rather
than fork it.

`internal/containers` (see [runtime.md](./runtime.md)) is split the
same way `pkg/protocol`/`proto` is: an engine-agnostic `Runtime`
interface that higher layers (the two container commands, the
dashboard) depend on, with concrete engines living in their own
subpackage (`internal/containers/docker` today). Adding Podman or
containerd later means a new subpackage, not a change to anything that
already depends on `Runtime`.

## Status

This document describes the intended shape of the system as features land.
Device registration ([device-registration.md](./device-registration.md)),
the persistent authenticated gRPC connection
([connection.md](./connection.md)), the Command Bus
([command-bus.md](./command-bus.md)), and the Runtime abstraction with
its Docker provider ([runtime.md](./runtime.md), observability only -
`LIST_CONTAINERS`/`INSPECT_CONTAINER`) are implemented. Heartbeats,
container lifecycle operations (start, stop, create, delete), image/
network/volume management, deployments, HTTPS, secrets, databases,
monitoring, backups, and clustering are not yet implemented.
