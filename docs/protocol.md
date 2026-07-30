# The DeployOS protocol

DeployOS defines its agent <-> control-plane protocol in
[Protocol Buffers](https://protobuf.dev/), under [`proto/`](../proto). This
document explains why, how it's versioned, and how to work with it. It is
API design only: nothing in this repository implements a gRPC server or
client against this protocol yet - see
[Status](#status) below.

## Why gRPC

Device registration (see [device-registration.md](./device-registration.md))
is a single request/response HTTP call, which is all it needs. Future
features - heartbeats, command delivery, config push - need something HTTP
doesn't give us cleanly: a long-lived, bidirectional stream between an
agent and the control plane, where either side can push a message at any
time without the other having polled for it.

gRPC is built for exactly that (native bidirectional streaming over HTTP/2,
one persistent connection instead of one connection per poll), and it's a
natural fit for a fleet of small, resource-constrained agents: generated
client code, binary encoding instead of JSON, and built-in support for
deadlines and cancellation via `context.Context`, which every DeployOS Go
component already threads through.

The device registration HTTP API isn't going away - it stays the right
tool for one-shot, request/response operations initiated by a human or a
script (registration, and later the dashboard's reads). gRPC is additive,
for the agent/control-plane relationship specifically.

## Why Protocol Buffers

Given gRPC, Protocol Buffers follow almost by default - it's gRPC's native
IDL. Beyond that, three things matter for DeployOS specifically:

- **A schema agents and the control plane both compile against.** Neither
  side can silently send the other a field that doesn't exist; the
  contract is checked at compile time, not discovered at runtime the way
  untyped JSON is.
- **Small wire size.** Agents may run on modest hardware and talk to the
  control plane continuously (once streaming lands) rather than once per
  request - binary encoding matters more here than for the human-facing
  dashboard API.
- **Forward/backward compatibility rules that are well understood.** Adding
  a field, or a new `oneof` case, doesn't break old clients or servers, as
  long as the rules below are followed. That's the whole point of doing
  this design work now, before anything depends on it.

## Layout

```
proto/
└── deployos/
    └── v1/
        ├── common.proto      Device, Connection, Status, Metadata
        ├── connection.proto  ConnectionService (persistent stream + auth)
        ├── command.proto     CommandService (future command delivery)
        └── agent.proto       AgentService (future agent operations)

gen/
└── go/
    └── deployos/
        └── v1/               Generated Go code (committed, never hand-edited)
```

Splitting into `common`/`connection`/`command`/`agent` keeps each service's
messages next to the service that owns them, while `common.proto` holds
only what's genuinely shared. `common.proto` has no imports from the
others - that's intentional, so it can never end up in an import cycle.

## Versioning strategy

The package name `deployos.v1` is the version. When a breaking change is
unavoidable, the new shape goes in a new package - `deployos.v2` - as a
new set of files under `proto/deployos/v2/`. `v1` keeps compiling and
keeps working for clients that haven't migrated; nothing is retroactively
edited. `Connection.protocol_version` (see `common.proto`) lets a
connecting agent and the control plane identify which version they're
speaking, for whenever more than one version is live at once.

Within `v1`, everything is expected to change only in backward-compatible
ways:

- New fields get new numbers; nothing already assigned is reused. Every
  message reserves a small range of field numbers after the last one in
  use, specifically so the next field added doesn't require a design
  decision about numbering.
- New `oneof` cases (see `ConnectionEnvelope` in `connection.proto`) are
  how new message kinds join an existing stream without changing the RPC
  signature.
- New RPCs are added to a service; existing RPCs are not removed or
  repurposed.
- `buf breaking` (see [Workflow](#workflow)) checks this mechanically
  against a prior version of the protocol. There's no baseline to check
  against until this change merges; once `main` has a `v1` to compare
  against, running it against `main` before every change is the way to
  catch an accidental break before it ships.

## Future protocol evolution

None of the services in this package are implemented, by design (see
[Status](#status)). As each feature lands, its RPCs are expected to gain
real handlers without changing the shapes defined today:

- **Heartbeats** ride `ConnectionService.Connect`'s stream once it's
  implemented: a new `oneof` case on `ConnectionEnvelope` (e.g.
  `Heartbeat`), not a new RPC.
- **Command delivery** implements `CommandService.Deliver`. If commands
  need to be pushed asynchronously rather than delivered as a unary call,
  that's a new streaming RPC on `CommandService`, added alongside
  `Deliver`.
- **Agent operations** (restart, upgrade, configuration push) are new RPCs
  on `AgentService`, following the same request/response wrapper pattern
  as `GetAgentStatus`.

## Workflow

Regenerating code requires [buf](https://buf.build/docs/installation) and
the Go protobuf plugins:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Then, from the repository root:

```bash
buf lint                               # protobuf style/API-design checks (buf.yaml)
buf breaking --against '.git#branch=main'  # fails if this change breaks main's protocol
buf generate                           # regenerates gen/go/deployos/v1 from buf.gen.yaml
```

`buf breaking` needs something to compare against - `.git#branch=main`
compares your working copy to `main`. CI doesn't run this yet (there's no
`v1` on `main` until this change merges); run it locally before changing
an existing `.proto` file.

Generated code is **committed**, not gitignored - `gen/` is checked into
the repository like any other Go source, so `go build` never needs `buf`
or `protoc` installed. CI's `proto` job runs `buf lint` and then
`buf generate` followed by `git diff`, to catch a `.proto` change that
was pushed without regenerating `gen/`.

Never hand-edit anything under `gen/` - it starts with
`// Code generated ... DO NOT EDIT.` and will be silently overwritten by
the next `buf generate`.

## Status

This document describes the protocol's shape, not a running system. As of
this writing:

- No gRPC server exists (`cmd/server` still only serves the HTTP API from
  [device-registration.md](./device-registration.md)).
- No gRPC client exists in `cmd/agent`.
- No persistent connection, heartbeat, command execution, or Docker
  integration has been implemented.

`buf lint`, `buf generate`, and `go build ./...` all succeed against what
exists today; that's the extent of what's verified.
