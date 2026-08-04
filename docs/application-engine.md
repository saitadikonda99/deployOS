# The Application Engine

The Application Engine is DeployOS's future orchestration layer: the
thing that will take an `Application` (a declared desired state - what
should run, from where, with what configuration) and actually make it
run, by cloning a repository, building an artifact, and handing it to a
Runtime Engine provider (see [runtime.md](./runtime.md)). This phase
implements neither the cloning, the building, nor the handing-off -
only `internal/applications`, the `Application` domain model, its
lifecycle state machine, and the interfaces (`Repository`, `Engine`)
a future implementation will satisfy. No Git, Docker, image-building,
reverse-proxy, domain, database, or environment-injection behavior
exists yet.

## Why Applications instead of Containers

`internal/containers.Container` (see [runtime.md](./runtime.md)) is
_observed_ state: whatever a Runtime Engine provider (Docker today)
reports actually exists and is running right now. It has no opinion
about where it came from, what it's supposed to look like, or what
should happen if it disappears - it's a read-only snapshot, and its
shape is dictated by what Docker's inspect API happens to return.

An `Application` is _desired_ state, and it needs to exist independent
of any specific runtime for a few concrete reasons:

- **A restart shouldn't lose intent.** If a container crashes,
  something needs to remember what it was supposed to be (which image,
  which env vars, which volumes) in order to bring it back. That
  record has to outlive the container itself, and outlive Docker
  specifically if DeployOS ever supports another engine.
- **Not every Application is a running container yet.** `StatusCreated`
  and `StatusPending` (see below) describe an Application before
  anything has been built or deployed - there's no container to
  observe at all. Modeling desired state as its own resource is what
  lets DeployOS represent "this should exist" separately from "this
  currently exists."
- **One Application, one Runtime Engine choice, many possible
  containers over time.** Redeploys, rebuilds, and rollbacks each
  produce a new container under a stable Application identity. Users
  and the dashboard want to reason about "my API" as one continuous
  thing, not track a rotating cast of container IDs.
- **Decoupling from Docker.** `Application.Runtime` is an opaque label
  (see below) - the model carries it but never branches on it. If
  `Container` were the primary resource instead, "what a deployment is"
  would be defined by whatever fields Docker's API happens to expose,
  which is exactly the coupling [runtime.md](./runtime.md) already
  argues against for the Runtime Engine layer itself.

In short: `Container` answers "what's running right now, per Docker."
`Application` answers "what does the user want to be running,
regardless of engine." The Application Engine's job (once it exists)
is reconciling the second into the first.

## The Application model

`internal/applications.Application` (`application.go`):

```go
type Application struct {
    ID                   ID
    UserID               string
    Name                 string
    Runtime              Runtime
    Repository           *SourceRepository // optional
    EnvironmentVariables map[string]string
    SecretRefs           []SecretRef
    Volumes              []Volume
    Domains              []string
    Status               Status
    CreatedAt            time.Time
    UpdatedAt            time.Time
}
```

- **`Runtime`** identifies which Runtime Engine provider (see
  [runtime.md](./runtime.md)) this Application deploys through. It's a
  plain string, not a closed enum - this package never branches on its
  value, which is what keeps it decoupled from Docker specifically.
  `RuntimeDocker` exists as a known convenience value, not a special
  case in any logic here.
- **`Repository`** is the git repository to build from, and is
  optional (a `nil` pointer, not a zero-value struct, to make "not
  configured yet" unambiguous) - cloning it is explicitly out of scope
  for this phase.
- **`SecretRefs`** record which secret should be injected as which
  environment variable, by name only. Resolving a name to an actual
  value is `internal/secrets`' job (not yet implemented); this package
  only stores the reference.
- **`Volumes`** and **`Domains`** are declared configuration for
  features (volume mounts, custom domains/reverse proxy) that don't
  exist yet either - recorded now so the model doesn't need a breaking
  change once they do.

`NewApplication(userID, name string, runtime Runtime) (Application,
error)` generates an ID, sets `Status` to `StatusCreated`, stamps both
timestamps, and validates the result - the same "construct and
validate in one call" shape `devices.RegisterInput.Validate` uses, so a
caller can never hold a semantically invalid `Application`.
`Application.Validate` checks required fields, that `Name` is a
DNS-label-style slug (it's expected to eventually appear in URLs,
generated domains, and container names), and that every populated
optional collection is individually well-formed. It never performs
I/O - checking that a referenced secret actually exists is a future
`Service`'s job, once one exists.

## Application lifecycle

```
Created ──▶ Pending ──▶ Building ──▶ Deploying ──▶ Running ──▶ Stopped
                │            │            │            │           │
                ▼            ▼            ▼            ▼           │
              Failed ◀───────┴────────────┴────────────┘           │
                │                                                   │
                └───────────────────▶ Pending ◀─────────────────────┘
```

| Status      | Meaning                                                                          |
| ----------- | -------------------------------------------------------------------------------- |
| `Created`   | The Application has been defined. Nothing has happened yet.                      |
| `Pending`   | Queued for the Application Engine to act on.                                     |
| `Building`  | The Engine is building a deployable artifact (e.g. cloning + building an image). |
| `Deploying` | A built artifact is being handed to a Runtime Engine provider.                   |
| `Running`   | Deployed, and the Runtime Engine provider reports it running.                    |
| `Stopped`   | Was running; deliberately stopped.                                               |
| `Failed`    | The Engine could not get the Application into its intended state.                |

`internal/applications/status.go` encodes the arrows above as
`allowedTransitions`, a `map[Status][]Status` - the single source of
truth for which moves are legal. `Application.CanTransitionTo(target)`
checks it without mutating anything;
`Application.TransitionTo(target)` performs the move, updating
`UpdatedAt`, or returns `ErrInvalidTransition` (wrapped with the
attempted `from -> to`) and leaves the Application untouched if the
move isn't allowed. `Stopped` and `Failed` both loop back to `Pending`,
since a stopped Application can be redeployed and a failed one
retried, rather than being dead ends.

This lives on `Application` itself (not a separate state-machine type)
because the transition rules only ever apply to one thing; a generic
FSM abstraction would be a layer of indirection nothing else in this
codebase needs yet.

## Relationship between the Application Engine and the Runtime Engine

These are two different layers with two different jobs, matching the
`Application`/`Container` split above:

- **The Runtime Engine** (`internal/containers`, see
  [runtime.md](./runtime.md)) knows how to talk to one specific
  container engine - today, Docker, over its unix socket. It has no
  concept of an Application, a repository, or a lifecycle; it only
  knows containers as they exist right now.
- **The Application Engine** (`internal/applications.Engine` - an
  interface only, no implementation yet) will know how to take an
  `Application` from `Created` through to `Running`: cloning
  `Repository`, building an artifact, then calling into a
  `containers.Runtime` implementation to actually run it. It depends on
  the Runtime Engine; the Runtime Engine has no dependency back.

That one-directional dependency is deliberate and mirrors how
`internal/commandbus` depends on nothing container-specific even though
`LIST_CONTAINERS`/`INSPECT_CONTAINER` are commands that call into
`internal/containers`: each layer only knows about the one below it.
Concretely, once an `Engine` implementation exists, `Engine.Deploy`
is expected to select a `containers.Runtime` provider based on
`Application.Runtime` and drive `Application.Status` through
`Building` → `Deploying` → `Running` (or `Failed`) as it goes - but
none of that exists yet. `Repository` (`internal/applications`'s
persistence interface) and `Engine` are both interfaces without an
implementation for the same reason `containers.Runtime` was defined a
phase before `internal/containers/docker` gave it a second reason to
exist as an interface: so the `Application` model is usable, testable,
and stable before anything depends on a specific way of fulfilling it.

## Dashboard

The Applications page (`apps/dashboard/src/app/applications/page.tsx`)
lists Name, Status, Runtime, and Created Date. Since no HTTP API or
persistence layer exists yet, it reads from
`apps/dashboard/src/lib/applications.ts`'s placeholder data instead of
a live control-plane call - `fetchApplications` is written as an async
function returning that data so a real API call is a drop-in
replacement later, the same shape `fetchDevices` in `lib/devices.ts`
already has.

## Testing

- `internal/applications/application_test.go`: `NewApplication`
  validation (name slug rules, required `UserID`/`Runtime`),
  `Application.Validate` for every optional collection
  (`EnvironmentVariables`, `SecretRefs`, `Volumes`, `Domains`,
  `Repository`), and that an `Application` with no optional fields set
  still validates.
- `internal/applications/status_test.go`: every transition the table in
  `allowedTransitions` allows, every transition it doesn't (including
  `Failed -> Failed`), that `TransitionTo` updates `UpdatedAt` on
  success and leaves it untouched on rejection, and `Status.valid`.

## Known limitations

- No Git cloning, image building, Docker deployment, reverse proxy,
  domain provisioning, database provisioning, or environment-variable
  injection. Those are all future phases.
- No persistence (`Repository` is an interface only - no Postgres, no
  in-memory store) and no HTTP API, so nothing outside a Go test can
  create an `Application` yet.
- No `Engine` implementation, so `Status` transitions are only ever
  driven manually (e.g. in a test) - nothing actually moves an
  Application from `Created` to `Running`.
