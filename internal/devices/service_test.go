package devices

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/saitadikonda99/deployOS/pkg/types"
)

type fakeRepository struct {
	devices map[types.AgentID]Device
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{devices: make(map[types.AgentID]Device)}
}

func (f *fakeRepository) Upsert(_ context.Context, device Device) (Device, error) {
	if existing, ok := f.devices[device.ID]; ok && existing.UserID != device.UserID {
		return Device{}, ErrOwnedByAnotherUser
	}
	if existing, ok := f.devices[device.ID]; ok {
		device.CreatedAt = existing.CreatedAt
	} else {
		device.CreatedAt = time.Now()
	}
	device.UpdatedAt = time.Now()
	f.devices[device.ID] = device
	return device, nil
}

func (f *fakeRepository) ListByUser(_ context.Context, userID string) ([]Device, error) {
	var result []Device
	for _, d := range f.devices {
		if d.UserID == userID {
			result = append(result, d)
		}
	}
	return result, nil
}

type fakeTokenIssuer struct {
	token string
	err   error
}

func (f *fakeTokenIssuer) Issue(_ context.Context, _ Device) (string, time.Time, error) {
	if f.err != nil {
		return "", time.Time{}, f.err
	}
	return f.token, time.Now().Add(time.Hour), nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func validInput() RegisterInput {
	return RegisterInput{
		DeviceID:        types.AgentID("11111111-1111-1111-1111-111111111111"),
		Hostname:        "dev-box",
		OperatingSystem: "linux",
		Architecture:    "amd64",
		CPUCores:        8,
		MemoryBytes:     17179869184,
		DeployOSVersion: "0.1.0",
	}
}

func TestServiceRegisterSuccess(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, &fakeTokenIssuer{token: "signed-token"}, testLogger())

	result, err := svc.Register(context.Background(), "user-1", validInput())
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if result.Token != "signed-token" {
		t.Errorf("Token = %q, want %q", result.Token, "signed-token")
	}
	if result.Device.Status != StatusRegistered {
		t.Errorf("Status = %q, want %q", result.Device.Status, StatusRegistered)
	}
	if result.Device.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", result.Device.UserID, "user-1")
	}
}

func TestServiceRegisterIsIdempotentForSameOwner(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, &fakeTokenIssuer{token: "t"}, testLogger())

	in := validInput()
	first, err := svc.Register(context.Background(), "user-1", in)
	if err != nil {
		t.Fatalf("first Register() error = %v", err)
	}

	in.Hostname = "renamed-box"
	second, err := svc.Register(context.Background(), "user-1", in)
	if err != nil {
		t.Fatalf("second Register() error = %v", err)
	}

	if second.Device.Hostname != "renamed-box" {
		t.Errorf("Hostname = %q, want %q", second.Device.Hostname, "renamed-box")
	}
	if second.Device.CreatedAt != first.Device.CreatedAt {
		t.Errorf("CreatedAt changed on re-registration: %v -> %v", first.Device.CreatedAt, second.Device.CreatedAt)
	}
}

func TestServiceRegisterRejectsDifferentOwner(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, &fakeTokenIssuer{token: "t"}, testLogger())

	in := validInput()
	if _, err := svc.Register(context.Background(), "user-1", in); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}

	_, err := svc.Register(context.Background(), "user-2", in)
	if !errors.Is(err, ErrOwnedByAnotherUser) {
		t.Fatalf("Register() error = %v, want ErrOwnedByAnotherUser", err)
	}
}

func TestServiceRegisterValidatesInput(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*RegisterInput)
	}{
		{"empty device id", func(in *RegisterInput) { in.DeviceID = "" }},
		{"non-uuid device id", func(in *RegisterInput) { in.DeviceID = "not-a-uuid" }},
		{"empty hostname", func(in *RegisterInput) { in.Hostname = "" }},
		{"empty os", func(in *RegisterInput) { in.OperatingSystem = "" }},
		{"empty architecture", func(in *RegisterInput) { in.Architecture = "" }},
		{"zero cpu cores", func(in *RegisterInput) { in.CPUCores = 0 }},
		{"zero memory", func(in *RegisterInput) { in.MemoryBytes = 0 }},
		{"empty version", func(in *RegisterInput) { in.DeployOSVersion = "" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepository()
			svc := NewService(repo, &fakeTokenIssuer{token: "t"}, testLogger())

			in := validInput()
			tt.mutate(&in)

			_, err := svc.Register(context.Background(), "user-1", in)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("Register() error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestServiceRegisterRejectsEmptyUserID(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, &fakeTokenIssuer{token: "t"}, testLogger())

	_, err := svc.Register(context.Background(), "", validInput())
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Register() error = %v, want ErrInvalidInput", err)
	}
}

func TestServiceList(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, &fakeTokenIssuer{token: "t"}, testLogger())

	if _, err := svc.Register(context.Background(), "user-1", validInput()); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	got, err := svc.List(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("List() returned %d devices, want 1", len(got))
	}

	got, err = svc.List(context.Background(), "user-2")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("List() for unrelated user returned %d devices, want 0", len(got))
	}
}

func TestServiceIsOwner(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, &fakeTokenIssuer{token: "t"}, testLogger())

	in := validInput()
	if _, err := svc.Register(context.Background(), "user-1", in); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	owned, err := svc.IsOwner(context.Background(), "user-1", in.DeviceID)
	if err != nil {
		t.Fatalf("IsOwner() error = %v", err)
	}
	if !owned {
		t.Error("IsOwner() = false, want true for the registering user")
	}

	owned, err = svc.IsOwner(context.Background(), "user-2", in.DeviceID)
	if err != nil {
		t.Fatalf("IsOwner() error = %v", err)
	}
	if owned {
		t.Error("IsOwner() = true, want false for an unrelated user")
	}

	owned, err = svc.IsOwner(context.Background(), "user-1", types.AgentID("99999999-9999-9999-9999-999999999999"))
	if err != nil {
		t.Fatalf("IsOwner() error = %v", err)
	}
	if owned {
		t.Error("IsOwner() = true, want false for a device that doesn't exist")
	}
}
