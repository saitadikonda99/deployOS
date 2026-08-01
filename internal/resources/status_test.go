package resources

import (
	"errors"
	"testing"
	"time"
)

func newTestResource(t *testing.T, status Status) Resource {
	t.Helper()
	res, err := NewResource("user-1", "primary-db", TypeDatabase)
	if err != nil {
		t.Fatalf("NewResource() error = %v", err)
	}
	res.Status = status
	return res
}

func TestResourceStatusTransitionsAllowed(t *testing.T) {
	tests := []struct {
		from Status
		to   Status
	}{
		{StatusPending, StatusProvisioning},
		{StatusProvisioning, StatusAvailable},
		{StatusProvisioning, StatusFailed},
		{StatusAvailable, StatusDeleting},
		{StatusAvailable, StatusFailed},
		{StatusFailed, StatusProvisioning},
		{StatusFailed, StatusDeleting},
		{StatusDeleting, StatusDeleted},
		{StatusDeleting, StatusFailed},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			res := newTestResource(t, tt.from)

			if !res.CanTransitionTo(tt.to) {
				t.Errorf("CanTransitionTo(%s) from %s = false, want true", tt.to, tt.from)
			}

			if err := res.TransitionTo(tt.to); err != nil {
				t.Errorf("TransitionTo(%s) from %s error = %v, want nil", tt.to, tt.from, err)
			}
			if res.Status != tt.to {
				t.Errorf("Status after TransitionTo(%s) = %s, want %s", tt.to, res.Status, tt.to)
			}
		})
	}
}

func TestResourceStatusTransitionsRejected(t *testing.T) {
	tests := []struct {
		from Status
		to   Status
	}{
		{StatusPending, StatusAvailable},
		{StatusPending, StatusFailed},
		{StatusProvisioning, StatusPending},
		{StatusProvisioning, StatusDeleting},
		{StatusAvailable, StatusPending},
		{StatusAvailable, StatusProvisioning},
		{StatusFailed, StatusAvailable},
		{StatusFailed, StatusPending},
		{StatusDeleting, StatusPending},
		{StatusDeleting, StatusAvailable},
		{StatusDeleted, StatusPending},
		{StatusDeleted, StatusProvisioning},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			res := newTestResource(t, tt.from)

			if res.CanTransitionTo(tt.to) {
				t.Errorf("CanTransitionTo(%s) from %s = true, want false", tt.to, tt.from)
			}

			err := res.TransitionTo(tt.to)
			if !errors.Is(err, ErrInvalidTransition) {
				t.Errorf("TransitionTo(%s) from %s error = %v, want %v", tt.to, tt.from, err, ErrInvalidTransition)
			}
			if res.Status != tt.from {
				t.Errorf("Status after rejected TransitionTo(%s) = %s, want unchanged %s", tt.to, res.Status, tt.from)
			}
		})
	}
}

func TestResourceDeletedIsTerminal(t *testing.T) {
	res := newTestResource(t, StatusDeleted)

	for _, target := range []Status{
		StatusPending, StatusProvisioning, StatusAvailable, StatusFailed, StatusDeleting,
	} {
		if res.CanTransitionTo(target) {
			t.Errorf("CanTransitionTo(%s) from Deleted = true, want false", target)
		}
	}
}

func TestResourceTransitionToUpdatesUpdatedAt(t *testing.T) {
	res := newTestResource(t, StatusPending)
	res.UpdatedAt = time.Now().UTC().Add(-time.Hour)
	before := res.UpdatedAt

	if err := res.TransitionTo(StatusProvisioning); err != nil {
		t.Fatalf("TransitionTo() error = %v", err)
	}

	if !res.UpdatedAt.After(before) {
		t.Errorf("UpdatedAt = %v, want after %v", res.UpdatedAt, before)
	}
}

func TestResourceTransitionToLeavesUpdatedAtUnchangedOnRejection(t *testing.T) {
	res := newTestResource(t, StatusPending)
	before := res.UpdatedAt

	if err := res.TransitionTo(StatusAvailable); err == nil {
		t.Fatal("TransitionTo() error = nil, want ErrInvalidTransition")
	}

	if !res.UpdatedAt.Equal(before) {
		t.Errorf("UpdatedAt = %v, want unchanged %v", res.UpdatedAt, before)
	}
}

func TestResourceStatusValid(t *testing.T) {
	for _, s := range []Status{
		StatusPending, StatusProvisioning, StatusAvailable,
		StatusFailed, StatusDeleting, StatusDeleted,
	} {
		if !s.valid() {
			t.Errorf("Status(%q).valid() = false, want true", s)
		}
	}

	if Status("not-a-status").valid() {
		t.Error("Status(\"not-a-status\").valid() = true, want false")
	}
}
