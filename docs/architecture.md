# Architecture

DeployOS is split along one boundary: components that manage the fleet
(TypeScript, in [`apps/`](../apps)) and components that run on each machine
being managed (Rust, in [`crates/`](../crates)).

## Components

| Component            | Language   | Responsibility                                                                  |
| -------------------- | ---------- | ------------------------------------------------------------------------------- |
| `apps/control-plane` | TypeScript | Source of truth for cluster state; exposes the API the dashboard and agents use |
| `apps/dashboard`     | TypeScript | Web UI for operators to manage deployments, secrets, and monitoring             |
| `crates/agent`       | Rust       | Runs on each managed machine; reports health and executes control-plane orders  |
| `crates/protocol`    | Rust       | Wire types shared between the control plane and agents                          |
| `crates/common`      | Rust       | Shared utilities and error types across Rust crates                             |

## Why this split

The control plane and dashboard are product surfaces that iterate quickly, so
they use the same TypeScript/pnpm/Turborepo toolchain. The agent runs
unattended on end-user hardware with a much smaller acceptable failure
surface (resource usage, crash safety, startup latency), so it is written in
Rust and kept dependency-light.

`crates/protocol` is the contract between the two halves: it defines message
shapes only, not transport. This keeps the transport (HTTP today, potentially
gRPC/QUIC later) an implementation detail of whichever side is calling it.

## Status

This document describes the intended shape of the system as features land.
At this stage the repository contains only the scaffolding described above;
no deployment, HTTPS, secrets, database, monitoring, backup, or clustering
logic has been implemented yet.
