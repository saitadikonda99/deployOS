# Architecture

DeployOS standardizes its entire backend on Go. The only TypeScript left in
the repository is the operator-facing dashboard, in
[`apps/dashboard`](../apps/dashboard); everything that manages a fleet or
runs on a managed machine is Go, under [`cmd/`](../cmd),
[`internal/`](../internal), and [`pkg/`](../pkg).

## Components

| Component                   | Language   | Responsibility                                                                                 |
| --------------------------- | ---------- | ---------------------------------------------------------------------------------------------- |
| `cmd/server`                | Go         | Control plane: source of truth for cluster state; exposes the API the dashboard and agents use |
| `cmd/agent`                 | Go         | Runs on each managed machine; reports health and will execute control-plane orders             |
| `cmd/cli`                   | Go         | `deployos` CLI - operator entry point (version, doctor, agent management)                      |
| `apps/dashboard`            | TypeScript | Web UI for operators to manage deployments, secrets, and monitoring                            |
| `pkg/protocol`, `pkg/types` | Go         | Wire types and value types shared between the control plane and agents                         |
| `pkg/api`                   | Go         | HTTP request/response contracts shared by every Go server                                      |

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

## Why this split

The control plane, agent, and CLI are all infrastructure processes with the
same requirements: predictable resource usage, fast startup, and safe
unattended operation. Standardizing them on Go keeps one toolchain, one set
of idioms, and one dependency-management story across all three, rather than
splitting infrastructure code across two languages.

The dashboard is a product surface that iterates on UI, not infrastructure
behavior, so it stays on the TypeScript/Next.js toolchain that's better
suited to that.

`pkg/protocol` is the contract between the control plane and agents: it
defines message shapes only, not transport. This keeps the transport (HTTP
today, potentially gRPC later) an implementation detail of whichever side is
calling it. `internal/agent` is deliberately structured so a gRPC listener
can be added there later without changing how `cmd/agent` wires things up.

## Status

This document describes the intended shape of the system as features land.
At this stage the repository contains only the scaffolding described above;
no deployment, HTTPS, secrets, database, monitoring, backup, or clustering
logic has been implemented yet.
