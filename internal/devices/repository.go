package devices

import (
	"context"
	"errors"
)

// ErrOwnedByAnotherUser is returned when a device ID is already
// registered under a different user's account.
var ErrOwnedByAnotherUser = errors.New("device is registered to another user")

// Repository persists devices. Implementations must treat Upsert as
// idempotent for the same (id, user_id) pair, and must return
// ErrOwnedByAnotherUser rather than silently reassigning ownership.
type Repository interface {
	// Upsert creates a device or updates it if a device with the same
	// ID already exists and belongs to the same user.
	Upsert(ctx context.Context, device Device) (Device, error)
	// ListByUser returns every device owned by userID, most recently
	// created first.
	ListByUser(ctx context.Context, userID string) ([]Device, error)
}
