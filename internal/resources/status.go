package resources

import (
	"errors"
	"fmt"
	"time"
)

// Status is a Resource's position in its lifecycle. See
// docs/resource-engine.md for the full lifecycle diagram and what each
// state means in practice.
type Status string

const (
	// StatusPending is a Resource's initial state: it has been
	// declared but nothing has been provisioned yet.
	StatusPending Status = "pending"
	// StatusProvisioning means a future Provisioner is creating the
	// underlying infrastructure.
	StatusProvisioning Status = "provisioning"
	// StatusAvailable means the Resource is provisioned and ready for
	// an Application to use.
	StatusAvailable Status = "available"
	// StatusFailed means provisioning, deprovisioning, or the
	// Resource itself could not reach its intended state.
	StatusFailed Status = "failed"
	// StatusDeleting means a future Provisioner is tearing the
	// underlying infrastructure down.
	StatusDeleting Status = "deleting"
	// StatusDeleted is a Resource's terminal state: the underlying
	// infrastructure is gone.
	StatusDeleted Status = "deleted"
)

// ErrInvalidTransition is returned when a Status transition isn't
// allowed by the state machine below.
var ErrInvalidTransition = errors.New("invalid resource status transition")

// allowedTransitions is the Resource state machine: for each Status,
// the set of Statuses it may move to directly. TransitionTo is the
// only place this is consulted - see docs/resource-engine.md for the
// diagram this table encodes.
var allowedTransitions = map[Status][]Status{
	StatusPending:      {StatusProvisioning},
	StatusProvisioning: {StatusAvailable, StatusFailed},
	StatusAvailable:    {StatusDeleting, StatusFailed},
	StatusFailed:       {StatusProvisioning, StatusDeleting},
	StatusDeleting:     {StatusDeleted, StatusFailed},
	StatusDeleted:      {},
}

// valid reports whether s is one of the states defined above.
func (s Status) valid() bool {
	_, ok := allowedTransitions[s]
	return ok
}

// CanTransitionTo reports whether the Resource may move from its
// current Status directly to target.
func (r Resource) CanTransitionTo(target Status) bool {
	for _, s := range allowedTransitions[r.Status] {
		if s == target {
			return true
		}
	}
	return false
}

// TransitionTo moves the Resource to target if the state machine
// allows it, updating UpdatedAt, and leaves the Resource unchanged and
// returns ErrInvalidTransition (wrapped with the attempted transition)
// otherwise.
func (r *Resource) TransitionTo(target Status) error {
	if !r.CanTransitionTo(target) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, r.Status, target)
	}
	r.Status = target
	r.UpdatedAt = time.Now().UTC()
	return nil
}
