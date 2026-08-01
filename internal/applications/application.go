// Package applications defines the Application resource: DeployOS's
// core representation of a deployable unit of work, independent of
// whatever Runtime Engine (see internal/containers) eventually runs
// it. This package defines the domain model, its lifecycle state
// machine (see status.go), and the interfaces a future Application
// Engine and persistence layer will implement (see interfaces.go) - it
// contains no Git, Docker, image-building, or reverse-proxy behavior
// itself. See docs/application-engine.md for the full picture.
package applications

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"

	"github.com/saitadikonda99/deployOS/internal/resources"
)

// ID uniquely identifies a single Application.
type ID string

// ErrEmptyID is returned when an ID is validated and found empty.
var ErrEmptyID = errors.New("application id must not be empty")

// Validate reports whether id is well-formed.
func (id ID) Validate() error {
	if id == "" {
		return ErrEmptyID
	}
	return nil
}

func (id ID) String() string {
	return string(id)
}

// Runtime identifies which Runtime Engine provider (see
// internal/containers) an Application deploys through. It's an opaque
// label this package carries but never interprets - deciding what
// "docker" means, and whether a given Runtime is actually supported,
// is the future Application Engine's job, not this package's. Keeping
// this a plain string rather than a closed enum is what keeps the
// Application model decoupled from Docker specifically.
type Runtime string

// RuntimeDocker is the only Runtime Engine provider that exists today
// (see internal/containers/docker). It's a convenience value, not a
// special case anywhere in this package's logic.
const RuntimeDocker Runtime = "docker"

// ErrEmptyRuntime is returned when an Application's Runtime is empty.
var ErrEmptyRuntime = errors.New("runtime must not be empty")

// Validate reports whether r is well-formed. It does not check that r
// names a Runtime Engine provider that actually exists.
func (r Runtime) Validate() error {
	if r == "" {
		return ErrEmptyRuntime
	}
	return nil
}

// SourceRepository is the git repository an Application deploys from.
// It's optional: an Application with a nil Repository simply has no
// source configured yet (or may never need one, e.g. a future
// prebuilt-image deployment) - cloning it is entirely out of scope for
// this package, which only records the reference.
type SourceRepository struct {
	URL    string
	Branch string
}

// SecretRef references a secret DeployOS should inject into an
// Application's environment at deploy time, by name only. Resolving
// SecretName to an actual value is internal/secrets' job (not yet
// implemented); this package only records the reference.
type SecretRef struct {
	// EnvVar is the environment variable name the secret is injected
	// as.
	EnvVar string
	// SecretName identifies the secret to resolve.
	SecretName string
}

// Volume is a filesystem path an Application expects mounted into its
// runtime environment.
type Volume struct {
	HostPath      string
	ContainerPath string
	ReadOnly      bool
}

// nameRE is the accepted shape for an Application Name: a DNS-label-
// style slug, since a Name is expected to eventually appear in URLs,
// container names, and generated domains.
var nameRE = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

var (
	// ErrEmptyUserID is returned when an Application's UserID is empty.
	ErrEmptyUserID = errors.New("user id must not be empty")
	// ErrEmptyName is returned when an Application's Name is empty.
	ErrEmptyName = errors.New("name must not be empty")
	// ErrInvalidName is returned when an Application's Name isn't a
	// valid resource-name slug.
	ErrInvalidName = errors.New("name must be lowercase alphanumeric characters and hyphens, and must not start or end with a hyphen")
	// ErrInvalidStatus is returned when an Application's Status isn't
	// one of the states defined in status.go.
	ErrInvalidStatus = errors.New("invalid application status")
	// ErrEmptyEnvVarName is returned when an EnvironmentVariables key
	// is empty.
	ErrEmptyEnvVarName = errors.New("environment variable name must not be empty")
	// ErrInvalidSecretRef is returned when a SecretRef is missing its
	// EnvVar or SecretName.
	ErrInvalidSecretRef = errors.New("secret reference must have both an env var and a secret name")
	// ErrInvalidVolume is returned when a Volume is missing its
	// HostPath or ContainerPath.
	ErrInvalidVolume = errors.New("volume must have both a host path and a container path")
	// ErrInvalidDomain is returned when a Domains entry is empty.
	ErrInvalidDomain = errors.New("domain must not be empty")
	// ErrInvalidResourceRef is returned when a ResourceRefs entry is
	// an invalid resources.ID.
	ErrInvalidResourceRef = errors.New("resource reference must be a valid resource id")
)

// Application is the domain representation of an application DeployOS
// manages. It is a declared desired state - what should run, from
// where, with what configuration - not a running process; internal/
// containers.Container is the corresponding observed state of an
// actual container once something has deployed this Application. See
// docs/application-engine.md for why that distinction matters.
type Application struct {
	ID     ID
	UserID string
	Name   string
	// Runtime is which Runtime Engine provider this Application
	// deploys through.
	Runtime Runtime
	// Repository is the git repository to build from, if any is
	// configured yet.
	Repository           *SourceRepository
	EnvironmentVariables map[string]string
	SecretRefs           []SecretRef
	Volumes              []Volume
	Domains              []string
	// ResourceRefs are the Resources (see internal/resources) this
	// Application depends on - a database, a cache, and so on -
	// referenced by ID only. internal/applications depends on
	// internal/resources for this ID type alone, never for how a
	// Resource is provisioned; internal/resources has no dependency
	// back. See "Why resources are independent from applications" in
	// docs/resource-engine.md.
	ResourceRefs []resources.ID
	Status       Status
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewApplication builds a new Application owned by userID, in
// StatusCreated, ready to be persisted. Callers set any optional
// fields (Repository, EnvironmentVariables, SecretRefs, Volumes,
// Domains) on the returned value before persisting it.
func NewApplication(userID, name string, runtime Runtime) (Application, error) {
	now := time.Now().UTC()
	app := Application{
		ID:        ID(uuid.NewString()),
		UserID:    userID,
		Name:      name,
		Runtime:   runtime,
		Status:    StatusCreated,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := app.Validate(); err != nil {
		return Application{}, err
	}
	return app, nil
}

// Validate reports whether the Application is well-formed: required
// fields are set, Name is a valid resource-name slug, and every
// optional collection's entries are individually well-formed. It never
// performs I/O (e.g. checking that a referenced secret actually
// exists) - that's a future Service's job, once one exists.
func (a Application) Validate() error {
	if err := a.ID.Validate(); err != nil {
		return err
	}
	if a.UserID == "" {
		return ErrEmptyUserID
	}
	if a.Name == "" {
		return ErrEmptyName
	}
	if !nameRE.MatchString(a.Name) {
		return ErrInvalidName
	}
	if err := a.Runtime.Validate(); err != nil {
		return err
	}
	if !a.Status.valid() {
		return fmt.Errorf("%w: %q", ErrInvalidStatus, a.Status)
	}
	for k := range a.EnvironmentVariables {
		if k == "" {
			return ErrEmptyEnvVarName
		}
	}
	for _, ref := range a.SecretRefs {
		if ref.EnvVar == "" || ref.SecretName == "" {
			return ErrInvalidSecretRef
		}
	}
	for _, v := range a.Volumes {
		if v.HostPath == "" || v.ContainerPath == "" {
			return ErrInvalidVolume
		}
	}
	for _, d := range a.Domains {
		if d == "" {
			return ErrInvalidDomain
		}
	}
	for _, id := range a.ResourceRefs {
		if err := id.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidResourceRef, err)
		}
	}
	return nil
}
