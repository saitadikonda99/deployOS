// Package resources defines the Resource model: DeployOS's
// representation of a first-class infrastructure object (a database, a
// cache, a volume, a secret, a domain) that an Application (see
// internal/applications) can reference by ID. This package defines the
// domain model, its type registry (registry.go), its lifecycle state
// machine (status.go), and the interfaces a future persistence layer
// and provisioning engine will implement (interfaces.go) - it contains
// no PostgreSQL, Redis, Docker, or DNS/TLS behavior itself. See
// docs/resource-engine.md for the full picture.
package resources

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// ID uniquely identifies a single Resource.
type ID string

// ErrEmptyID is returned when an ID is validated and found empty.
var ErrEmptyID = errors.New("resource id must not be empty")

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

// nameRE is the accepted shape for a Resource Name: a DNS-label-style
// slug, the same shape internal/applications requires of an
// Application Name, since a Resource's Name is expected to eventually
// appear in generated identifiers too (e.g. a database's hostname).
var nameRE = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

var (
	// ErrEmptyUserID is returned when a Resource's UserID is empty.
	ErrEmptyUserID = errors.New("user id must not be empty")
	// ErrEmptyName is returned when a Resource's Name is empty.
	ErrEmptyName = errors.New("name must not be empty")
	// ErrInvalidName is returned when a Resource's Name isn't a valid
	// resource-name slug.
	ErrInvalidName = errors.New("name must be lowercase alphanumeric characters and hyphens, and must not start or end with a hyphen")
	// ErrUnsupportedType is returned when a Resource's Type isn't
	// registered in the Registry used to validate it.
	ErrUnsupportedType = errors.New("unsupported resource type")
	// ErrInvalidStatus is returned when a Resource's Status isn't one
	// of the states defined in status.go.
	ErrInvalidStatus = errors.New("invalid resource status")
	// ErrEmptyConfigKey is returned when a Config key is empty.
	ErrEmptyConfigKey = errors.New("config key must not be empty")
)

// Resource is the domain representation of a first-class infrastructure
// object DeployOS manages on behalf of one or more Applications. It
// deliberately has no idea what an Application is - see "Why resources
// are independent from applications" in docs/resource-engine.md - and
// no idea how it would actually be provisioned on a Runtime Engine
// provider (see internal/containers): Config is opaque, type-specific
// configuration a future Provisioner interprets, not this package.
type Resource struct {
	ID     ID
	UserID string
	Name   string
	Type   Type
	// Config is opaque, type-specific configuration (e.g. an engine
	// version for a DATABASE, a size for a VOLUME). Kept as a generic
	// map, the same way commandbus.Request/Response carry opaque
	// key/value data, so this package never needs a struct-per-Type -
	// interpreting Config is a future Provisioner's job.
	Config    map[string]string
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewResource builds a new Resource owned by userID, in
// StatusPending, ready to be persisted. It validates resourceType
// against the package's default Registry (the five built-in types);
// use Resource.ValidateWithRegistry directly for a custom Registry.
// Callers set Config on the returned value before persisting it.
func NewResource(userID, name string, resourceType Type) (Resource, error) {
	now := time.Now().UTC()
	res := Resource{
		ID:        ID(uuid.NewString()),
		UserID:    userID,
		Name:      name,
		Type:      resourceType,
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := res.Validate(); err != nil {
		return Resource{}, err
	}
	return res, nil
}

// Validate reports whether the Resource is well-formed, checking Type
// against the package's default Registry (the five built-in types).
func (r Resource) Validate() error {
	return r.ValidateWithRegistry(defaultRegistry)
}

// ValidateWithRegistry reports whether the Resource is well-formed:
// required fields are set, Name is a valid resource-name slug, Type is
// supported by registry, Status is one of status.go's defined states,
// and Config has no empty keys. It never performs I/O (e.g. actually
// checking a database is reachable) - that's a future Provisioner's
// job, once one exists.
func (r Resource) ValidateWithRegistry(registry *Registry) error {
	if err := r.ID.Validate(); err != nil {
		return err
	}
	if r.UserID == "" {
		return ErrEmptyUserID
	}
	if r.Name == "" {
		return ErrEmptyName
	}
	if !nameRE.MatchString(r.Name) {
		return ErrInvalidName
	}
	if !registry.Supports(r.Type) {
		return fmt.Errorf("%w: %q", ErrUnsupportedType, r.Type)
	}
	if !r.Status.valid() {
		return fmt.Errorf("%w: %q", ErrInvalidStatus, r.Status)
	}
	for k := range r.Config {
		if k == "" {
			return ErrEmptyConfigKey
		}
	}
	return nil
}
