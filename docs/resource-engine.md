# The Resource Engine

The Resource Engine is DeployOS's future infrastructure-provisioning
layer: the thing that will take a `Resource` (a declared desired
state - a database, a cache, a volume, a secret, a domain) and
actually create it, the way the Application Engine (see
[application-engine.md](./application-engine.md)) will take an
`Application` and actually run it. This phase implements neither
PostgreSQL nor Redis provisioning, volumes, secrets storage, domains,
nor deployments - only `internal/resources`: the `Resource` domain
model, its type registry, its lifecycle state machine, the interfaces
(`Repository`, `Provisioner`) a future implementation will satisfy, and
the reference an `Application` uses to depend on one.

## Resources are first-class, independent of Applications

A `Resource` is DeployOS's representation of a piece of infrastructure
an `Application` needs but doesn't itself define - a database, a
cache, a volume, a secret, or a domain. It's "first-class" in the sense
the task set out to establish: a `Resource` is a real object with its
own identity, lifecycle, and validation rules, not a nested field
buried inside `Application`.

### Why resources are independent from applications

`internal/resources` has no dependency on `internal/applications` at
all - the `Resource` struct, its registry, and its state machine are
defined with zero knowledge that an `Application` exists. The
dependency runs the other way: `Application.ResourceRefs` (see
[application-engine.md](./application-engine.md)) is a
`[]resources.ID`, added in this phase to the existing `Application`
struct. Concretely, this one-directional relationship is what makes
several things possible:

- **Sharing.** The same database can be `ResourceRefs`'d by more than
  one `Application` (e.g. a shared Postgres instance behind an API and
  a background worker) without `Resource` needing to track a set of
  owners - it just doesn't know or care who references it.
- **Independent lifecycle.** A `Resource` can be `Pending` or
  `Provisioning` before any `Application` depends on it, and can keep
  existing (`Available`) after the `Application` that first needed it
  is deleted, if another still references it. Coupling the two would
  force them to share a lifecycle they don't actually share.
- **No circular dependency.** If `Resource` held a list of the
  `Application`s using it, `internal/resources` would have to import
  `internal/applications`, which already imports `internal/resources`
  for the ID type - a cycle Go doesn't allow, and a coupling that
  isn't needed anyway (see "Used By" below).
- **Reuse beyond Applications.** Nothing about a `Resource` mentions
  deployment. A future feature that isn't an `Application` at all could
  still reference a `Resource` by ID the same way.

"Used By" (the dashboard's fourth column) is a **derived** view, not a
stored field: it's every `Application` whose `ResourceRefs` contains a
given `Resource`'s ID, computed by whoever's asking (today, hand-
written placeholder data; eventually a `Service` that joins the two).
Nothing about `Resource` itself changes to support it.

## The Resource model

`internal/resources.Resource` (`resource.go`):

```go
type Resource struct {
    ID        ID
    UserID    string
    Name      string
    Type      Type
    Config    map[string]string
    Status    Status
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

- **`Type`** is a plain string (`"DATABASE"`, `"CACHE"`, `"VOLUME"`,
  `"SECRET"`, `"DOMAIN"`), not a closed Go enum - the same choice
  `Application.Runtime` makes, and for the same reason: this package
  never branches on it, so adding a type later doesn't require
  changing any logic here, only registering it (see Registry, below).
- **`Config`** is a generic `map[string]string` for type-specific
  configuration (a database's engine version, a volume's size, ...),
  the same shape `commandbus.Request`/`Response` already use for
  opaque data elsewhere in this codebase. There's deliberately no
  struct-per-`Type` - that would mean this package growing a new field
  set every time a type is added, exactly the coupling a generic model
  is meant to avoid. Interpreting `Config` is a future `Provisioner`'s
  job.

`NewResource(userID, name string, resourceType Type) (Resource,
error)` generates an ID, sets `Status` to `StatusPending`, stamps both
timestamps, and validates the result against the package's default
type registry - the same "construct and validate in one call" shape
`Application`'s constructor uses.

## Resource registry

```go
type Registry struct { /* ... */ }

func NewRegistry() *Registry
func (r *Registry) Register(desc TypeDescriptor)
func (r *Registry) Supports(t Type) bool
func (r *Registry) Get(t Type) (TypeDescriptor, bool)
func (r *Registry) Types() []Type
```

`NewRegistry` returns a `Registry` pre-populated with the five built-in
types. `Resource.Validate` checks `Type` against a package-level
default `Registry` (also the five built-ins); `Resource.ValidateWithRegistry`
takes an explicit one. This is what "generic model, specific built-in
types" means in practice: `Resource.Type` isn't restricted to five
hardcoded string constants by the compiler - it's restricted by
whatever's registered, and a sixth type (`"QUEUE"`, say) becomes valid
the moment something calls `Register`, with no change to `Resource`,
`Validate`, or the state machine. This mirrors
`internal/commandbus.Dispatcher.Register` (a kind maps to a handler)
and anticipates a `Provisioner` needing the same per-`Type` dispatch
(see below).

## Resource validation

`Resource.Validate` (called by `NewResource`, and available directly
for a `Resource` built by hand, e.g. after deserializing one) checks:
`ID` is non-empty, `UserID` is non-empty, `Name` matches the same DNS-
label-style slug `internal/applications` requires (a `Resource`'s name
is expected to eventually appear in generated identifiers, like a
database hostname), `Type` is registered in the given `Registry`,
`Status` is one of the states `status.go` defines, and no `Config` key
is empty. Like `Application.Validate`, it never performs I/O -
confirming a database is actually reachable is a future `Provisioner`'s
job.

## Resource lifecycle

```
Pending ──▶ Provisioning ──▶ Available ──▶ Deleting ──▶ Deleted
                │                 │            ▲
                ▼                 ▼            │
              Failed ─────────────┴────────────┘
                │
                └──────────▶ Provisioning
```

| Status         | Meaning                                                             |
| -------------- | ------------------------------------------------------------------- |
| `Pending`      | The Resource has been declared. Nothing has been provisioned yet.   |
| `Provisioning` | A future Provisioner is creating the underlying infrastructure.     |
| `Available`    | Provisioned and ready for an Application to use.                    |
| `Failed`       | Provisioning, deprovisioning, or the Resource itself broke.         |
| `Deleting`     | A future Provisioner is tearing the underlying infrastructure down. |
| `Deleted`      | Terminal: the underlying infrastructure is gone.                    |

`internal/resources/status.go` encodes the arrows above as
`allowedTransitions`, a `map[Status][]Status` - the same
single-source-of-truth shape `internal/applications/status.go` uses.
`Resource.CanTransitionTo(target)` checks without mutating;
`Resource.TransitionTo(target)` performs the move (updating
`UpdatedAt`) or returns `ErrInvalidTransition`, leaving the `Resource`
untouched, if the move isn't allowed. `Failed` loops back to
`Provisioning` (retry) or forward to `Deleting` (give up and remove
it) rather than being a dead end; `Deleted` has no outgoing
transitions at all - a deleted Resource is actually gone, not
resurrectable in place (a new `Resource` would need a new `ID`).

This lifecycle is intentionally not identical to `Application`'s: an
`Application` builds and deploys (`Building`, `Deploying`), while a
`Resource` is provisioned directly (`Provisioning`) - there's no
build step for a database. Keeping them as two separate state machines,
each shaped for what it actually represents, is simpler than forcing
one shared enum to mean slightly different things in two packages.

## Application ↔ Resource relationship

`Application.ResourceRefs []resources.ID` (see
[application-engine.md](./application-engine.md)) is the entire
relationship: a list of `Resource` IDs an `Application` depends on.
`Application.Validate` checks every entry is a well-formed
`resources.ID` (non-empty) - nothing more, since actually confirming a
referenced `Resource` exists and belongs to the same user is a future
`Service`'s job (the same "no I/O in `Validate`" rule both packages
follow). `internal/applications/application_test.go`'s
`TestApplicationResourceRefsRelationship` builds a real `Resource` via
`resources.NewResource`, attaches its `ID` to an `Application`, and
confirms the `Application` still validates - proving the two models
cooperate through nothing more than an opaque ID, exactly as designed.

## Future provisioning architecture

```go
type Provisioner interface {
    Provision(ctx context.Context, res Resource) error
    Deprovision(ctx context.Context, res Resource) error
}
```

`Provisioner` (in `interfaces.go`, alongside a `Repository` persistence
interface following `devices.Repository`/`applications.Repository`'s
shape) is where actual infrastructure creation will live - not in this
package, and not yet at all. The expected shape, once it's built:

- **One `Provisioner` per `Type`**, dispatched the way
  `commandbus.Dispatcher` dispatches by command kind: a
  `DATABASE`/`CACHE` `Provisioner` would most likely run a container
  via a Runtime Engine provider (`internal/containers` - see
  [runtime.md](./runtime.md)), the same layer the Application Engine
  is expected to use to actually run an `Application`. A `SECRET`
  `Provisioner` would talk to `internal/secrets` (not yet
  implemented). A `DOMAIN` `Provisioner` would issue a certificate and
  configure a reverse proxy (neither implemented yet either).
- **A Resource Engine** analogous to the future Application Engine:
  something that reads a `Resource`'s `Status`, picks the right
  `Provisioner` for its `Type` (very plausibly via a registry the same
  shape as this phase's `Registry`, mapping `Type` to `Provisioner`
  instead of `Type` to `TypeDescriptor`), and drives it through
  `Pending` → `Provisioning` → `Available` (or `Failed`).
- **Still no coupling to Docker specifically.** Even once a `DATABASE`
  `Provisioner` exists and happens to use `internal/containers/docker`
  under the hood, `Resource` itself continues to know nothing about
  it - the same separation `internal/containers` already maintains
  between "what a container is" and "Docker specifically."

None of this exists yet. `Provisioner` is an interface with zero
implementations, the same starting point `internal/containers.Runtime`
had before `internal/containers/docker` gave it a second reason to be
an interface, and `internal/applications.Engine` has today.

## Dashboard

The Resources page (`apps/dashboard/src/app/resources/page.tsx`) lists
Name, Type, Status, Used By, and Created At. Since no HTTP API or
persistence layer exists yet, it reads from
`apps/dashboard/src/lib/resources.ts`'s placeholder data - `used_by`
is hand-written to reference the same placeholder application names
`lib/applications.ts` uses, illustrating the derived relationship
described above without an actual join. `fetchResources` is written as
an async function so a real API call is a drop-in replacement later,
the same shape `fetchDevices`/`fetchApplications` already have.

## Testing

- `internal/resources/resource_test.go`: `NewResource` validation
  (name slug rules, required `UserID`, unsupported `Type`), `Resource.Validate`
  for `Status` and `Config`, and `ValidateWithRegistry` against a
  custom registry.
- `internal/resources/registry_test.go`: the five built-in types are
  registered by default, an unknown type is rejected, `Register` adds
  a new type and can replace an existing one's descriptor.
- `internal/resources/status_test.go`: every transition
  `allowedTransitions` allows, every transition it doesn't (including
  every attempt to leave `Deleted`), that `TransitionTo` updates
  `UpdatedAt` on success and leaves it untouched on rejection, and
  `Status.valid`.
- `internal/applications/application_test.go`:
  `TestApplicationResourceRefsRelationship` (a real `Resource` attached
  to a real `Application` validates), plus that `ResourceRefs` is
  optional and that an empty entry is rejected.

## Known limitations

- No PostgreSQL provisioning, Redis provisioning, volume backing
  storage, secrets storage, domain/DNS/TLS handling, or deployments.
  Those are all future phases.
- No persistence (`Repository` is an interface only) and no HTTP API,
  so nothing outside a Go test can create a `Resource` yet.
- No `Provisioner` implementation, so `Status` transitions are only
  ever driven manually (e.g. in a test) - nothing actually provisions
  anything.
- "Used By" is illustrated with hand-written placeholder data on the
  dashboard, not a real join - there's no `Repository` to query yet.
