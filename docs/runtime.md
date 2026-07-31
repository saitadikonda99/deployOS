# The Runtime abstraction

The Runtime abstraction is how DeployOS observes containers on a managed
machine without coupling itself to any specific container engine. It's
implemented in `internal/containers` (the interface) and
`internal/containers/docker` (the first provider), and exposed to the
control plane and dashboard through two new Command Bus commands:
`LIST_CONTAINERS` and `INSPECT_CONTAINER`. This phase only implements
observability - no lifecycle operations (start, stop, create, delete,
images, networks, volumes) exist yet.

## Why an interface, not a Docker call

Docker is what most DeployOS hosts run today, but it won't be the only
option forever - Podman, containerd, and Kubernetes are all plausible
future backends. If `internal/agent`'s command handlers called Docker's
API directly, adding a second engine would mean touching every layer
that currently assumes Docker. Instead, everything above
`internal/containers` - the two commands, the dashboard - depends only
on the `Runtime` interface:

```go
type Runtime interface {
    ListContainers(ctx context.Context) ([]Container, error)
    InspectContainer(ctx context.Context, id string) (ContainerDetails, error)
}
```

Adding a Podman or containerd provider means writing a new package that
implements `Runtime` (the way `internal/containers/docker` does) and
changing one line in `internal/agent`'s wiring
(`docker.NewRuntime(...)` -> `podman.NewRuntime(...)`, say). Nothing in
`internal/commandbus`, the dashboard, or the two container commands'
handlers would need to change.

## The Docker provider

`internal/containers/docker.Runtime` implements `containers.Runtime` by
calling the Docker Engine API directly over its unix socket
(`/var/run/docker.sock` by default - see `docker_socket` in
`config.example.yaml`/`.env.example`). The Docker Engine API is plain
HTTP+JSON, so this uses `net/http` with a custom dialer rather than
pulling in Docker's own client SDK - one dependency avoided, per this
repo's preference for fewer of them.

- `ListContainers` calls `GET /containers/json?all=true` and maps each
  entry to a `containers.Container` summary (ID, name, image, status,
  state, created time).
- `InspectContainer` calls `GET /containers/{id}/json` and maps the
  richer response to `containers.ContainerDetails` (adds command, env,
  mounts, published ports, networks, restart count). A 404 from the
  daemon becomes `containers.ErrContainerNotFound`.

## Command Bus integration

`LIST_CONTAINERS` and `INSPECT_CONTAINER` are ordinary Command Bus
commands (see [command-bus.md](./command-bus.md)) - the bus itself has
no idea containers exist. `internal/agent/handlers.go` registers two
more executors alongside `PING`/`GET_VERSION`/`GET_INFO`:

- `listContainersExecutor(runtime)` calls `runtime.ListContainers`,
  JSON-encodes the result, and returns it as a single
  `Response.Details["containers"]` entry.
- `inspectContainerExecutor(runtime)` reads the target container's ID
  from `Request.Arguments["id"]`, calls `runtime.InspectContainer`, and
  returns the result the same way, under `Details["container"]`.

Encoding the payload as a JSON string inside `Details` (rather than
extending `commandbus.Response` with a structured field) is what keeps
the Command Bus itself generic: it still only ever carries
`map[string]string`, exactly as it did for `PING`/`GET_VERSION`/
`GET_INFO`, and every future command is free to make the same choice
without commandbus needing to know what any of them mean.

Sending `INSPECT_CONTAINER` requires an argument, which meant extending
the wire path that previously had none:
`api.SendCommandRequest.Arguments` (new, optional field) flows through
`commandbus.Handler.Send` into `commandbus.Request.Arguments`, which
already existed - only the HTTP layer was missing a way to populate it.

Adding a new command remains exactly what [command-bus.md](./command-bus.md)
describes: write an `Executor`, register it in `internal/agent/handlers.go`.
`LIST_CONTAINERS`/`INSPECT_CONTAINER` didn't need anything beyond that
except the one HTTP field above, since they're the first two commands
that take an argument.

## Dashboard

The devices page (`apps/dashboard/src/app/devices/page.tsx`) has a
"Containers" column, rendering `ContainerList` per device
(`container-list.tsx`, a client component). "View Containers" sends
`LIST_CONTAINERS` and renders the result as a table; each row's
"Inspect" button sends `INSPECT_CONTAINER` with that container's ID and
renders the detail response below the table. Both go through the same
`sendCommand` server action already used by the PING/GET_VERSION/
GET_INFO actions menu (`lib/commands.ts`), now extended to accept an
optional arguments map.

## Testing

- `internal/containers/docker/docker_test.go`: `ListContainers` and
  `InspectContainer` against a fake Docker daemon (`httptest.Server`
  standing in for the unix socket) - success, empty result, daemon
  error, and not-found.
- `internal/agent/handlers_test.go`: `listContainersExecutor`/
  `inspectContainerExecutor` against a fake `containers.Runtime` -
  success, runtime error, and a missing `id` argument.
- `internal/commandbus/integration_test.go`
  (`TestIntegrationContainerCommandsRoundTrip`): both commands over a
  real connection, proving `Request.Arguments` (the inspect target's
  ID) actually survives the wire, not just an in-process call.
- `internal/commandbus/handler_test.go`
  (`TestHandlerSendPassesArgumentsThrough`): the HTTP layer forwards
  `SendCommandRequest.Arguments` into the command actually sent to the
  device.

## Known limitations

- Observability only: no start, stop, create, delete, image, network,
  or volume operations exist. Those are future phases, likely each
  adding their own command(s) against the same `Runtime` interface (and
  extending it if a lifecycle operation needs to be represented).
- Only Docker is implemented. Podman/containerd/Kubernetes providers
  are possible without touching anything above `internal/containers`,
  but none exist yet.
- `internal/containers/docker` assumes a local unix socket; a remote
  Docker host (TCP, TLS) isn't wired up, though `Runtime`'s HTTP client
  could support one without an interface change.
