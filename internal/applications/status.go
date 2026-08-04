package applications

import (
	"errors"
	"fmt"
	"time"
)

// Status is an Application's position in its lifecycle. See
// docs/application-engine.md for the full lifecycle diagram and what
// each state means in practice.
type Status string

const (
	// StatusCreated is an Application's initial state: it has been
	// defined but nothing has happened yet.
	StatusCreated Status = "created"
	// StatusPending means the Application is queued for the
	// Application Engine to act on.
	StatusPending Status = "pending"
	// StatusBuilding means the Application Engine is building a
	// deployable artifact (e.g. cloning a repository and building an
	// image).
	StatusBuilding Status = "building"
	// StatusDeploying means a built artifact is being handed to a
	// Runtime Engine provider.
	StatusDeploying Status = "deploying"
	// StatusRunning means the Application is deployed and its Runtime
	// Engine provider reports it running.
	StatusRunning Status = "running"
	// StatusStopped means the Application was running and was
	// deliberately stopped.
	StatusStopped Status = "stopped"
	// StatusFailed means the Application Engine could not get the
	// Application into its intended state.
	StatusFailed Status = "failed"
)

// ErrInvalidTransition is returned when a Status transition isn't
// allowed by the state machine below.
var ErrInvalidTransition = errors.New("invalid application status transition")

// allowedTransitions is the Application state machine: for each
// Status, the set of Statuses it may move to directly. TransitionTo is
// the only place this is consulted - see docs/application-engine.md
// for the diagram this table encodes.
var allowedTransitions = map[Status][]Status{
	StatusCreated:   {StatusPending},
	StatusPending:   {StatusBuilding, StatusFailed},
	StatusBuilding:  {StatusDeploying, StatusFailed},
	StatusDeploying: {StatusRunning, StatusFailed},
	StatusRunning:   {StatusStopped, StatusFailed},
	StatusStopped:   {StatusPending, StatusFailed},
	StatusFailed:    {StatusPending},
}

// valid reports whether s is one of the states defined above.
func (s Status) valid() bool {
	_, ok := allowedTransitions[s]
	return ok
}

// CanTransitionTo reports whether the Application may move from its
// current Status directly to target.
func (a Application) CanTransitionTo(target Status) bool {
	for _, s := range allowedTransitions[a.Status] {
		if s == target {
			return true
		}
	}
	return false
}

// TransitionTo moves the Application to target if the state machine
// allows it, updating UpdatedAt, and leaves the Application unchanged
// and returns ErrInvalidTransition (wrapped with the attempted
// transition) otherwise.
func (a *Application) TransitionTo(target Status) error {
	if !a.CanTransitionTo(target) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, a.Status, target)
	}
	a.Status = target
	a.UpdatedAt = time.Now().UTC()
	return nil
}
