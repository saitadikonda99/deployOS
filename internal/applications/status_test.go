package applications

import (
	"errors"
	"testing"
	"time"
)

func newTestApplication(t *testing.T, status Status) Application {
	t.Helper()
	app, err := NewApplication("user-1", "my-app", RuntimeDocker)
	if err != nil {
		t.Fatalf("NewApplication() error = %v", err)
	}
	app.Status = status
	return app
}

func TestStatusTransitionsAllowed(t *testing.T) {
	tests := []struct {
		from Status
		to   Status
	}{
		{StatusCreated, StatusPending},
		{StatusPending, StatusBuilding},
		{StatusPending, StatusFailed},
		{StatusBuilding, StatusDeploying},
		{StatusBuilding, StatusFailed},
		{StatusDeploying, StatusRunning},
		{StatusDeploying, StatusFailed},
		{StatusRunning, StatusStopped},
		{StatusRunning, StatusFailed},
		{StatusStopped, StatusPending},
		{StatusStopped, StatusFailed},
		{StatusFailed, StatusPending},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			app := newTestApplication(t, tt.from)

			if !app.CanTransitionTo(tt.to) {
				t.Errorf("CanTransitionTo(%s) from %s = false, want true", tt.to, tt.from)
			}

			if err := app.TransitionTo(tt.to); err != nil {
				t.Errorf("TransitionTo(%s) from %s error = %v, want nil", tt.to, tt.from, err)
			}
			if app.Status != tt.to {
				t.Errorf("Status after TransitionTo(%s) = %s, want %s", tt.to, app.Status, tt.to)
			}
		})
	}
}

func TestStatusTransitionsRejected(t *testing.T) {
	tests := []struct {
		from Status
		to   Status
	}{
		{StatusCreated, StatusRunning},
		{StatusCreated, StatusBuilding},
		{StatusPending, StatusRunning},
		{StatusPending, StatusDeploying},
		{StatusBuilding, StatusRunning},
		{StatusBuilding, StatusPending},
		{StatusDeploying, StatusBuilding},
		{StatusRunning, StatusBuilding},
		{StatusRunning, StatusDeploying},
		{StatusRunning, StatusPending},
		{StatusStopped, StatusRunning},
		{StatusStopped, StatusBuilding},
		{StatusFailed, StatusRunning},
		{StatusFailed, StatusBuilding},
		{StatusFailed, StatusFailed},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			app := newTestApplication(t, tt.from)

			if app.CanTransitionTo(tt.to) {
				t.Errorf("CanTransitionTo(%s) from %s = true, want false", tt.to, tt.from)
			}

			err := app.TransitionTo(tt.to)
			if !errors.Is(err, ErrInvalidTransition) {
				t.Errorf("TransitionTo(%s) from %s error = %v, want %v", tt.to, tt.from, err, ErrInvalidTransition)
			}
			if app.Status != tt.from {
				t.Errorf("Status after rejected TransitionTo(%s) = %s, want unchanged %s", tt.to, app.Status, tt.from)
			}
		})
	}
}

func TestTransitionToUpdatesUpdatedAt(t *testing.T) {
	app := newTestApplication(t, StatusCreated)
	app.UpdatedAt = time.Now().UTC().Add(-time.Hour)
	before := app.UpdatedAt

	if err := app.TransitionTo(StatusPending); err != nil {
		t.Fatalf("TransitionTo() error = %v", err)
	}

	if !app.UpdatedAt.After(before) {
		t.Errorf("UpdatedAt = %v, want after %v", app.UpdatedAt, before)
	}
}

func TestTransitionToLeavesUpdatedAtUnchangedOnRejection(t *testing.T) {
	app := newTestApplication(t, StatusCreated)
	before := app.UpdatedAt

	if err := app.TransitionTo(StatusRunning); err == nil {
		t.Fatal("TransitionTo() error = nil, want ErrInvalidTransition")
	}

	if !app.UpdatedAt.Equal(before) {
		t.Errorf("UpdatedAt = %v, want unchanged %v", app.UpdatedAt, before)
	}
}

func TestStatusValid(t *testing.T) {
	for _, s := range []Status{
		StatusCreated, StatusPending, StatusBuilding, StatusDeploying,
		StatusRunning, StatusStopped, StatusFailed,
	} {
		if !s.valid() {
			t.Errorf("Status(%q).valid() = false, want true", s)
		}
	}

	if Status("not-a-status").valid() {
		t.Error("Status(\"not-a-status\").valid() = true, want false")
	}
}
