# DeployOS Documentation

This directory holds documentation that goes deeper than the top-level
[README](../README.md).

- [`architecture.md`](./architecture.md) - system components and how they
  communicate.
- [`device-registration.md`](./device-registration.md) - how an agent
  registers itself with the control plane.
- [`protocol.md`](./protocol.md) - the Protocol Buffers/gRPC protocol
  design for future agent <-> control-plane communication.

As DeployOS grows, add one document per subsystem here (deployments,
networking/HTTPS, secrets, databases, monitoring, backups, clustering) rather
than growing this file indefinitely.
