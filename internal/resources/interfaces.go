package resources

import "context"

// Repository persists Resources. No implementation exists yet (no
// Postgres, no in-memory store) - this interface only establishes the
// contract a future persistence layer will satisfy, the same role
// devices.Repository and applications.Repository play for their
// packages.
type Repository interface {
	// Create persists a new Resource.
	Create(ctx context.Context, res Resource) (Resource, error)
	// Get returns the Resource identified by id.
	Get(ctx context.Context, id ID) (Resource, error)
	// ListByUser returns every Resource owned by userID.
	ListByUser(ctx context.Context, userID string) ([]Resource, error)
	// Update persists changes to an existing Resource (e.g. a Status
	// transition).
	Update(ctx context.Context, res Resource) (Resource, error)
	// Delete removes a Resource.
	Delete(ctx context.Context, id ID) error
}

// Provisioner represents the future behavior of actually creating and
// tearing down a Resource's underlying infrastructure - e.g. running a
// Postgres container via a Runtime Engine provider (see
// internal/containers) for a DATABASE, or writing a certificate for a
// DOMAIN. No implementation exists yet, and different Types will very
// likely need different Provisioner implementations dispatched by
// Type, the way internal/commandbus.Dispatcher dispatches by command
// kind - see "Future provisioning architecture" in
// docs/resource-engine.md.
//
// Provisioner depends on Resource, never the reverse, and nothing in
// this package calls it - keeping the dependency one-directional is
// what lets the Resource model exist and be tested before a
// Provisioner implementation does, the same reasoning
// internal/applications.Engine follows for Application.
type Provisioner interface {
	// Provision creates the underlying infrastructure for res and
	// should leave it in StatusAvailable, or StatusFailed if it
	// cannot.
	Provision(ctx context.Context, res Resource) error
	// Deprovision tears down the underlying infrastructure for res.
	Deprovision(ctx context.Context, res Resource) error
}
