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
small Go binary that runs on every managed machine, reports health, and
will execute instructions from the control plane. The two sides communicate
over types defined once, in `pkg/protocol`, so the wire contract can't
drift between them. Operators also have a `deployos` CLI
(`cmd/cli`) for local diagnostics and, eventually, fleet management. See
[`docs/architecture.md`](./docs/architecture.md) for more detail.

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
│   ├── devices/         Device registration (repository/service/handler)
│   ├── docker/          Future container-lifecycle interface
│   ├── logging/         Structured JSON logging
│   ├── monitoring/      Health-check registry
│   ├── runtime/         Shared graceful-shutdown HTTP server
│   ├── scheduler/       Future job-scheduling interface
│   ├── discovery/       Future agent/control-plane discovery
│   └── secrets/         Future secrets storage
├── pkg/
│   ├── api/             HTTP request/response contracts
│   ├── protocol/        Wire types shared between control plane and agents
│   └── types/           Foundational value types (agent IDs, versions)
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
```

Copy [`.env.example`](./.env.example) to `.env` and/or
[`config.example.yaml`](./config.example.yaml) to `config.yaml` to
configure the agent and control plane locally; see
[CONTRIBUTING.md](./CONTRIBUTING.md) for details.

## Roadmap

DeployOS is being built in the open, in phases:

1. **Foundation** _(done)_ — monorepo scaffolding, tooling, and CI.
2. **Device registration** _(this repository, today - see
   [docs/device-registration.md](./docs/device-registration.md))_ —
   agents register themselves with the control plane and receive a
   signed device token.
3. **Deploy from Git** — build and run an application from a repository on
   a single node.
4. **Automatic HTTPS** — certificate issuance and renewal with zero
   configuration.
5. **Docker management** — container lifecycle as a managed platform
   primitive, not a manual `docker` invocation.
6. **Secrets** — first-class secret storage and injection for deployed
   applications.
7. **Databases** — managed database provisioning and lifecycle.
8. **Monitoring** — metrics, logs, and alerting out of the box.
9. **Backups** — automated, verifiable backup and restore.
10. **AI-powered operations** — assisted diagnosis, remediation suggestions,
    and operational summaries.
11. **Multi-device fleets** — multiple machines operated as a single
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
