# DeployOS Documentation

This directory holds documentation that goes deeper than the top-level
[README](../README.md).

- [`architecture.md`](./architecture.md) - system components and how they
  communicate.
- [`device-registration.md`](./device-registration.md) - how an agent
  registers itself with the control plane.
- [`protocol.md`](./protocol.md) - the Protocol Buffers/gRPC protocol
  design for future agent <-> control-plane communication.
- [`connection.md`](./connection.md) - the persistent authenticated gRPC
  connection between agent and control plane.
- [`command-bus.md`](./command-bus.md) - request/response command
  routing between the control plane and an agent.
- [`runtime.md`](./runtime.md) - the Runtime abstraction and its first
  provider (Docker), for observing containers on a managed machine.
- [`application-engine.md`](./application-engine.md) - the
  `Application` domain model, its lifecycle, and how it relates to the
  Runtime abstraction.

As DeployOS grows, add one document per subsystem here (deployments,
networking/HTTPS, secrets, databases, monitoring, backups, clustering) rather
than growing this file indefinitely.
