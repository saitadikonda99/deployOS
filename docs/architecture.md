# Architecture

DeployOS standardizes its entire backend on Go. The only TypeScript left in
the repository is the operator-facing dashboard, in
[`apps/dashboard`](../apps/dashboard); everything that manages a fleet or
runs on a managed machine is Go, under [`cmd/`](../cmd),
[`internal/`](../internal), and [`pkg/`](../pkg).

## Components

| Component                   | Language   | Responsibility                                                                                                |
| --------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------- |
| `cmd/server`                | Go         | Control plane: source of truth for cluster state; exposes the API the dashboard and agents use                |
| `cmd/agent`                 | Go         | Runs on each managed machine; registers itself, reports health, and will execute control-plane orders         |
| `cmd/cli`                   | Go         | `deployos` CLI - operator entry point (version, doctor, agent management)                                     |
| `apps/dashboard`            | TypeScript | Web UI for operators to manage deployments, secrets, and monitoring                                           |
| `internal/devices`          | Go         | Device registration: repository/service/handler for `POST /api/v1/devices/register` and `GET /api/v1/devices` |
| `internal/auth`             | Go         | Authenticates operators against Supabase Auth on the control plane's behalf                                   |
| `pkg/protocol`, `pkg/types` | Go         | Wire types and value types shared between the control plane and agents (HTTP)                                 |
| `pkg/api`                   | Go         | HTTP request/response contracts shared by every Go server                                                     |
| `gen/go/deployos/v1`        | Go         | Generated Protocol Buffers/gRPC types (see [protocol.md](./protocol.md)) - design only, not yet implemented   |

The control plane is the only component with Supabase credentials; see
[device-registration.md](./device-registration.md) for how the agent,
control plane, and Supabase interact for that feature specifically.

## Package layout

- **`cmd/`** holds only entry points (`main` packages) - argument parsing,
  wiring, and calling into `internal/`. No business logic lives here.
- **`internal/`** holds the implementation, split by concern
  (`config`, `logging`, `runtime`, `monitoring`, `agent`, `docker`,
  `scheduler`, `auth`, `discovery`, `secrets`, ...). It is unimportable
  outside this module, which is exactly the point: these are DeployOS's own
  internals, not a public API.
- **`pkg/`** holds the small amount of Go that's meant to be shared beyond
  this module's own binaries - wire types and API contracts - so it stays
  intentionally minimal.
- **`proto/`** holds the Protocol Buffers source for the future
  agent <-> control-plane gRPC protocol; **`gen/`** holds the Go code
  generated from it. Neither is implemented against yet - see
  [protocol.md](./protocol.md).

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
next evolution of that same idea, for the persistent, bidirectional
gRPC connection future features (heartbeats, command delivery) need.
`internal/agent` is deliberately structured so a gRPC listener can be
added there later without changing how `cmd/agent` wires things up.

## Status

This document describes the intended shape of the system as features land.
Device registration (see [device-registration.md](./device-registration.md))
is the first real feature implemented. The gRPC protocol (see
[protocol.md](./protocol.md)) is designed and its Go code generated, but
nothing implements it yet - no gRPC server, no gRPC client, no persistent
connection. Deployment, HTTPS, secrets, databases, monitoring, backups,
and clustering are not yet implemented.
