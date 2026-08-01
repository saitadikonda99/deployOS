package applications

import "context"

// Repository persists Applications. No implementation exists yet (no
// Postgres, no in-memory store) - this interface only establishes the
// contract a future persistence layer will satisfy, the same role
// devices.Repository plays for internal/devices.
type Repository interface {
	// Create persists a new Application.
	Create(ctx context.Context, app Application) (Application, error)
	// Get returns the Application identified by id.
	Get(ctx context.Context, id ID) (Application, error)
	// ListByUser returns every Application owned by userID.
	ListByUser(ctx context.Context, userID string) ([]Application, error)
	// Update persists changes to an existing Application (e.g. a
	// Status transition).
	Update(ctx context.Context, app Application) (Application, error)
	// Delete removes an Application.
	Delete(ctx context.Context, id ID) error
}

// Engine represents the future Application Engine: the component
// responsible for actually driving an Application through its
// lifecycle - cloning a repository, building an image, handing it to a
// Runtime Engine provider (see internal/containers) to run, stopping
// it, and so on. No implementation exists yet; this interface only
// establishes the contract the rest of DeployOS (HTTP handlers, the
// Command Bus) will eventually depend on, the same way
// internal/containers.Runtime let LIST_CONTAINERS/INSPECT_CONTAINER be
// written against an interface before any second provider existed.
//
// Engine depends on Application, never the reverse, and nothing in
// this package calls it - keeping the dependency one-directional is
// what lets the Application model exist and be tested before an Engine
// implementation does.
type Engine interface {
	// Deploy takes app from its current Status through Building and
	// Deploying to Running, or to Failed if it cannot.
	Deploy(ctx context.Context, app Application) error
	// Stop takes a Running Application to Stopped.
	Stop(ctx context.Context, app Application) error
	// Restart stops a running Application and deploys it again.
	Restart(ctx context.Context, app Application) error
}
