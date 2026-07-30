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

DeployOS is split along one boundary: components that manage the fleet, and
components that run on each machine being managed.

```
                         ┌───────────────────────────┐
                         │        Dashboard          │
                         │   apps/dashboard (TS)      │
                         └─────────────┬─────────────┘
                                       │
                                       ▼
                         ┌───────────────────────────┐
                         │      Control Plane        │
                         │ apps/control-plane (TS)   │
                         │  fleet & deployment state │
                         └─────────────┬─────────────┘
                                       │ deployos-protocol
                       ┌───────────────┼───────────────┐
                       ▼               ▼               ▼
                 ┌───────────┐   ┌───────────┐   ┌───────────┐
                 │   Agent   │   │   Agent   │   │   Agent   │
                 │ crates/   │   │ crates/   │   │ crates/   │
                 │  agent    │   │  agent    │   │  agent    │
                 │  (Rust)   │   │  (Rust)   │   │  (Rust)   │
                 └───────────┘   └───────────┘   └───────────┘
                  node A            node B            node C
```

The **control plane** and **dashboard** are TypeScript applications that
iterate quickly and hold the source of truth for cluster state. The
**agent** is a small, dependency-light Rust binary that runs on every
managed machine, reports health, and executes instructions from the control
plane. The two sides communicate over types defined once, in
`deployos-protocol`, so the wire contract can't drift between them. See
[`docs/architecture.md`](./docs/architecture.md) for more detail.

## Repository layout

```
deployos/
├── apps/
│   ├── dashboard/       Web dashboard (Next.js/TypeScript)
│   └── control-plane/   Fleet & deployment orchestration API (Node/TypeScript)
├── crates/
│   ├── agent/           Node agent binary (Rust)
│   ├── common/          Shared Rust utilities and error types
│   └── protocol/        Wire types shared between control plane and agents
├── packages/            Shared TypeScript packages (as they're extracted)
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
- **No placeholders.** Code that lands on `main` compiles, lints, and does
  what it claims. Speculative stubs and TODO-driven scaffolding are not
  merged; a feature ships when it's real.
- **A typed contract between fleet and node.** The control plane and the
  agents that run on managed machines never share ambient assumptions about
  message shapes — everything crosses that boundary through
  `deployos-protocol`.
- **Small, composable pieces.** Prefer several well-scoped crates/packages
  over one large one. `packages/` and `crates/` are expected to grow as
  functionality is extracted, not stay flat forever.
- **Automate the boring parts.** Formatting, linting, import order, and
  commit hygiene are enforced by tooling (Biome, Prettier, Husky,
  lint-staged, Changesets), not code review comments.

## Development

Requirements: [Node.js](https://nodejs.org) >= 20, [pnpm](https://pnpm.io)
(via [Corepack](https://nodejs.org/api/corepack.html)), and
[Rust](https://www.rust-lang.org/tools/install) (stable).

```bash
./scripts/bootstrap.sh   # verifies your toolchain and installs JS deps

pnpm build                # turbo-orchestrated build across all TS apps/packages
pnpm lint                 # Biome across the whole workspace
pnpm format               # Biome + Prettier
cargo build --workspace   # build the Rust crates
cargo test --workspace    # test the Rust crates
```

## Roadmap

DeployOS is being built in the open, in phases:

1. **Foundation** _(this repository, today)_ — monorepo scaffolding,
   tooling, and CI.
2. **Deploy from Git** — build and run an application from a repository on
   a single node.
3. **Automatic HTTPS** — certificate issuance and renewal with zero
   configuration.
4. **Docker management** — container lifecycle as a managed platform
   primitive, not a manual `docker` invocation.
5. **Secrets** — first-class secret storage and injection for deployed
   applications.
6. **Databases** — managed database provisioning and lifecycle.
7. **Monitoring** — metrics, logs, and alerting out of the box.
8. **Backups** — automated, verifiable backup and restore.
9. **AI-powered operations** — assisted diagnosis, remediation suggestions,
   and operational summaries.
10. **Multi-device fleets** — multiple machines operated as a single
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
