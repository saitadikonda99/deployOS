# The DeployOS protocol

DeployOS defines its agent <-> control-plane protocol in
[Protocol Buffers](https://protobuf.dev/), under [`proto/`](../proto). This
document explains why, how it's versioned, and how to work with it - see
[Status](#status) below for what's actually implemented against it today.

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
        ├── connection.proto  ConnectionService (persistent stream + auth + command routing)
        ├── command.proto     Command, CommandResult (see command-bus.md)
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

As each feature lands, it's expected to extend `ConnectionEnvelope`'s
`oneof` rather than add a standalone RPC. This isn't just a style
preference: a standalone unary RPC has to be initiated by whichever side
holds the gRPC _client_, and in DeployOS that's always the agent (it
dials the control plane; the control plane never dials an agent, since
agents are commonly behind NAT). `command.proto` originally sketched a
`CommandService.Deliver` unary RPC exactly this way - it was removed
once the Command Bus (see [command-bus.md](./command-bus.md)) actually
needed implementing and that design turned out to be unusable. Command
routing now rides `ConnectionEnvelope`'s existing stream instead
(`command_request`/`command_response` oneof cases), which is what let it
be implemented without a protocol-breaking change.

- **Heartbeats** ride the same stream: a new `oneof` case on
  `ConnectionEnvelope`, not a new RPC.
- **Agent operations** (restart, upgrade, configuration push): as
  written, `agent.proto`'s `AgentService.GetAgentStatus` has the same
  control-plane-initiates-a-unary-call shape that turned out not to work
  for commands. Expect it to be revised into `ConnectionEnvelope` oneof
  cases too, the same way `CommandService` was, rather than implemented
  as-is.

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

`ConnectionService` is implemented on both sides (see
[connection.md](./connection.md)): `cmd/server` runs a real gRPC server,
`cmd/agent` runs a real gRPC client, and the persistent connection
between them - authentication, reconnection - works end to end. Command
routing over that connection is also implemented (see
[command-bus.md](./command-bus.md)), with three commands
(`PING`/`GET_VERSION`/`GET_INFO`) proving it end to end.

Not implemented: heartbeats, Docker integration, deployments, metrics,
log streaming, terminal access, file transfer, and `AgentService` (see
[Future protocol evolution](#future-protocol-evolution) above for why
its current shape needs revisiting before it is).
