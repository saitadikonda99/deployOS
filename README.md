# DeployOS

**An open-source personal cloud operating system.**

DeployOS turns any computer — a home server, a spare laptop, a rack of
mini PCs, a VPS — into a secure, production-ready cloud server with a single
installation.

[![CI](https://github.com/saitadikonda99/deployOS/actions/workflows/ci.yml/badge.svg)](https://github.com/saitadikonda99/deployOS/actions/workflows/ci.yml)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](./LICENSE)

## Vision

Running production infrastructure today means stitching together a VPS, a
reverse proxy, a container runtime, a secrets manager, a database, a
monitoring stack, and a backup strategy — and then keeping all of it patched,
secure, and observable, by hand.

DeployOS collapses that stack into a single, opinionated system you install
once. Point it at a Git repository and it deploys the app, provisions HTTPS,
and keeps it running. It is infrastructure software for people who want the
capabilities of a cloud platform without renting one.

## Goals

- **One install, production-ready.** A single install script turns a bare
  machine into a secure host, with sane defaults for updates, firewalling,
  and access.
- **Deploy from Git.** Push to a repository; DeployOS builds, ships, and
  runs it.
- **Automatic HTTPS everywhere.** Certificates are issued and renewed
  without operator intervention.
- **First-class secrets and databases.** Application configuration and
  stateful services are managed by the platform, not hand-rolled per app.
- **Observability by default.** Monitoring and backups are part of the
  platform, not an integration left to the user.
- **AI-assisted operations.** Routine operational work (diagnosing a failed
  deploy, proposing a scaling change, summarizing an incident) is assisted
  by the platform itself.
- **One cloud, many machines.** Multiple physical or virtual machines can
  join a single DeployOS fleet and be operated as one logical cloud.

## High-level architecture

DeployOS standardizes its entire backend on Go. The dashboard is the one
TypeScript surface left; everything that manages a fleet or runs on a
managed machine is Go.

```
                         ┌───────────────────────────┐
                         │        Dashboard          │
                         │   apps/dashboard (TS)      │
                         └─────────────┬─────────────┘
                                       │
                                       ▼
                         ┌───────────────────────────┐
                         │      Control Plane        │
                         │      cmd/server (Go)      │
                         │  fleet & deployment state │
                         └─────────────┬─────────────┘
                                       │ pkg/protocol
                       ┌───────────────┼───────────────┐
                       ▼               ▼               ▼
                 ┌───────────┐   ┌───────────┐   ┌───────────┐
                 │   Agent   │   │   Agent   │   │   Agent   │
                 │  cmd/     │   │  cmd/     │   │  cmd/     │
                 │  agent    │   │  agent    │   │  agent    │
                 │   (Go)    │   │   (Go)    │   │   (Go)    │
                 └───────────┘   └───────────┘   └───────────┘
                  node A            node B            node C
```

The **control plane** holds the source of truth for cluster state and is
what the **dashboard** and every **agent** talk to. Each **agent** is a
small Go binary that runs on every managed machine and reports health.
Registration and the dashboard's reads go over HTTP, with types defined
once in `pkg/protocol` so the wire contract can't drift. Alongside that,
every agent holds a persistent, authenticated gRPC connection to the
control plane (`internal/connection`, protocol in `proto/`/`gen/`), which
the Command Bus (`internal/commandbus`) uses to route request/response
commands from the control plane to an agent and back - the transport
future features (heartbeats, metrics, log streaming) will ride on the
same connection. One of those commands observes containers on the
agent's machine through the Runtime abstraction
(`internal/containers`), an interface implemented today by
`internal/containers/docker` and, in principle, by any other container
engine later without touching anything above it. See
[`docs/connection.md`](./docs/connection.md),
[`docs/command-bus.md`](./docs/command-bus.md),
[`docs/runtime.md`](./docs/runtime.md), and
[`docs/protocol.md`](./docs/protocol.md). Operators also have a
`deployos` CLI (`cmd/cli`) for local diagnostics and, eventually, fleet
management. See [`docs/architecture.md`](./docs/architecture.md) for
more detail.

## Repository layout

```
deployos/
├── apps/
│   └── dashboard/       Web dashboard (Next.js/TypeScript)
├── cmd/
│   ├── agent/           Node agent binary entry point (Go)
│   ├── server/          Control plane binary entry point (Go)
│   └── cli/             deployos CLI entry point (Go, Cobra)
├── internal/
│   ├── agent/           Agent process implementation (identity, registration, health)
│   ├── auth/            Authenticates operators against Supabase Auth
│   ├── config/          Configuration loading (env, .env, YAML)
│   ├── commandbus/      Command Bus: request/response command routing over the connection
│   ├── connection/      Persistent authenticated gRPC connection (client, server, Connection Manager)
│   ├── containers/      Runtime abstraction: engine-agnostic container observability interface
│   │   └── docker/      The first Runtime provider (Docker Engine API over its unix socket)
│   ├── devices/         Device registration (repository/service/handler)
│   ├── logging/         Structured JSON logging
│   ├── monitoring/      Health-check registry
│   ├── runtime/         Shared graceful-shutdown HTTP/gRPC servers
│   ├── scheduler/       Future job-scheduling interface
│   ├── discovery/       Future agent/control-plane discovery
│   └── secrets/         Future secrets storage
├── pkg/
│   ├── api/             HTTP request/response contracts
│   ├── protocol/        Wire types shared between control plane and agents
│   └── types/           Foundational value types (agent IDs, versions)
├── proto/
│   └── deployos/v1/     Protocol Buffers source (agent <-> control-plane gRPC protocol)
├── gen/
│   └── go/deployos/v1/  Generated Go code from proto/ (committed, never hand-edited)
├── supabase/
│   └── migrations/      SQL migrations (users, devices, projects)
├── docs/                Architecture and subsystem documentation
├── scripts/             Repo automation (bootstrap, etc.)
├── docker/              Container build/compose files, added per-service
└── .github/             CI workflows, issue/PR templates, Dependabot
```

## Development philosophy

- **Infrastructure-grade, not weekend-project-grade.** Every subsystem is
  designed to be run unattended on real machines: crashes are handled,
  upgrades are safe, and there is no step that only works on the author's
  machine.
- **No placeholders in `cmd/` and `internal/`.** Code that lands on `main`
  compiles, lints, and does what it claims. The CLI's placeholder commands
  are the deliberate exception, called out explicitly rather than left to
  guesswork.
- **A typed contract between fleet and node.** The control plane and the
  agents that run on managed machines never share ambient assumptions about
  message shapes — everything crosses that boundary through
  `pkg/protocol`.
- **Thin `cmd/`, real `internal/`.** Binaries under `cmd/` parse arguments
  and wire dependencies together; the behavior they wire up lives in
  `internal/`, one package per concern.
- **Automate the boring parts.** Formatting, linting, import order, and
  commit hygiene are enforced by tooling (Biome, Prettier, gofmt,
  golangci-lint, Husky, lint-staged, Changesets), not code review comments.

## Development

Requirements: [Node.js](https://nodejs.org) >= 20, [pnpm](https://pnpm.io)
(via [Corepack](https://nodejs.org/api/corepack.html)), and
[Go](https://go.dev/doc/install) (latest stable).

```bash
./scripts/bootstrap.sh   # verifies your toolchain and installs JS deps

pnpm build                # turbo-orchestrated build across all TS apps
pnpm lint                 # Biome across the whole workspace
pnpm format               # Biome + Prettier

go build ./...            # build every Go binary
go vet ./...               # Go's static checks
golangci-lint run ./...    # Go linting
go test ./...              # Go unit tests

buf lint                   # protobuf style checks (only needed if you touch proto/)
buf generate                # regenerate gen/ from proto/ - see docs/protocol.md
```

Copy [`.env.example`](./.env.example) to `.env` and/or
[`config.example.yaml`](./config.example.yaml) to `config.yaml` to
configure the agent and control plane locally; see
[CONTRIBUTING.md](./CONTRIBUTING.md) for details.

## Roadmap

DeployOS is being built in the open, in phases:

1. **Foundation** _(done)_ — monorepo scaffolding, tooling, and CI.
2. **Device registration** _(done - see
   [docs/device-registration.md](./docs/device-registration.md))_ —
   agents register themselves with the control plane and receive a
   signed device token.
3. **Persistent connection** _(done - see
   [docs/connection.md](./docs/connection.md))_ — agents hold a
   persistent, authenticated gRPC connection to the control plane, with
   automatic reconnection.
4. **Command Bus** _(done - see
   [docs/command-bus.md](./docs/command-bus.md))_ — generic
   request/response command routing between the control plane and an
   agent, over the persistent connection.
5. **Runtime abstraction** _(this repository, today - see
   [docs/runtime.md](./docs/runtime.md))_ — an engine-agnostic `Runtime`
   interface for container observability, with Docker as its first
   provider; `LIST_CONTAINERS`/`INSPECT_CONTAINER` are its only commands
   so far.
6. **Deploy from Git** — build and run an application from a repository on
   a single node.
7. **Automatic HTTPS** — certificate issuance and renewal with zero
   configuration.
8. **Container lifecycle management** — start, stop, create, and delete
   as managed platform primitives on top of the Runtime abstraction, not
   a manual `docker` invocation.
9. **Secrets** — first-class secret storage and injection for deployed
   applications.
10. **Databases** — managed database provisioning and lifecycle.
11. **Monitoring** — metrics, logs, and alerting out of the box.
12. **Backups** — automated, verifiable backup and restore.
13. **AI-powered operations** — assisted diagnosis, remediation suggestions,
    and operational summaries.
14. **Multi-device fleets** — multiple machines operated as a single
    logical cloud.

Each phase ships as working software behind the same standards described
above — no phase is considered "started" until it lints, builds, and does
what it says.

## Contributing

Contributions are welcome. Please read
[CONTRIBUTING.md](./CONTRIBUTING.md) and our
[Code of Conduct](./CODE_OF_CONDUCT.md) before opening a pull request.

## Security

See [SECURITY.md](./SECURITY.md) for how to report a vulnerability.

## License

DeployOS is licensed under the [Apache License 2.0](./LICENSE).
