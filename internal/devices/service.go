package devices

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/saitadikonda99/deployOS/pkg/types"
)

// ErrInvalidInput is returned when a RegisterInput fails validation.
var ErrInvalidInput = errors.New("invalid device registration input")

// RegisterInput is the validated set of fields needed to register a
// device, independent of how the request arrived (HTTP today).
type RegisterInput struct {
	DeviceID        types.AgentID
	Hostname        string
	OperatingSystem string
	Architecture    string
	CPUCores        int
	MemoryBytes     uint64
	DeployOSVersion string
}

// Validate reports whether in has everything Service.Register requires.
func (in RegisterInput) Validate() error {
	if err := in.DeviceID.Validate(); err != nil {
		return fmt.Errorf("device_id: %w", err)
	}
	if _, err := uuid.Parse(in.DeviceID.String()); err != nil {
		return fmt.Errorf("device_id: must be a UUID: %w", err)
	}
	if in.Hostname == "" {
		return errors.New("hostname must not be empty")
	}
	if in.OperatingSystem == "" {
		return errors.New("operating_system must not be empty")
	}
	if in.Architecture == "" {
		return errors.New("architecture must not be empty")
	}
	if in.CPUCores <= 0 {
		return errors.New("cpu_cores must be positive")
	}
	if in.MemoryBytes == 0 {
		return errors.New("memory_bytes must be positive")
	}
	if in.DeployOSVersion == "" {
		return errors.New("deployos_version must not be empty")
	}
	return nil
}

// RegisterResult is the outcome of a successful registration.
type RegisterResult struct {
	Device    Device
	Token     string
	ExpiresAt time.Time
}

// Service owns device registration: validating input, persisting the
// device, and issuing its token. It has no knowledge of HTTP or of how
// devices are stored - those are Handler's and Repository's concerns.
type Service struct {
	repo   Repository
	tokens TokenIssuer
	logger *slog.Logger
}

// NewService builds a Service.
func NewService(repo Repository, tokens TokenIssuer, logger *slog.Logger) *Service {
	return &Service{repo: repo, tokens: tokens, logger: logger}
}

// Register validates in, upserts the device under userID, and issues it
// a token. Registering the same device ID again (e.g. on agent restart)
// is idempotent, provided userID hasn't changed; a different owner gets
// ErrOwnedByAnotherUser.
func (s *Service) Register(ctx context.Context, userID string, in RegisterInput) (RegisterResult, error) {
	if userID == "" {
		return RegisterResult{}, fmt.Errorf("%w: user id must not be empty", ErrInvalidInput)
	}
	if err := in.Validate(); err != nil {
		return RegisterResult{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	device := Device{
		ID:              in.DeviceID,
		UserID:          userID,
		Hostname:        in.Hostname,
		OperatingSystem: in.OperatingSystem,
		Architecture:    in.Architecture,
		CPUCores:        in.CPUCores,
		MemoryBytes:     in.MemoryBytes,
		DeployOSVersion: in.DeployOSVersion,
		Status:          StatusRegistered,
	}

	stored, err := s.repo.Upsert(ctx, device)
	if err != nil {
		if errors.Is(err, ErrOwnedByAnotherUser) {
			return RegisterResult{}, err
		}
		return RegisterResult{}, fmt.Errorf("storing device: %w", err)
	}

	token, expiresAt, err := s.tokens.Issue(ctx, stored)
	if err != nil {
		return RegisterResult{}, fmt.Errorf("issuing device token: %w", err)
	}

	s.logger.Info("device registered",
		slog.String("device_id", stored.ID.String()),
		slog.String("user_id", userID),
	)

	return RegisterResult{Device: stored, Token: token, ExpiresAt: expiresAt}, nil
}

// List returns every device owned by userID.
func (s *Service) List(ctx context.Context, userID string) ([]Device, error) {
	if userID == "" {
		return nil, fmt.Errorf("%w: user id must not be empty", ErrInvalidInput)
	}
	devices, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing devices: %w", err)
	}
	return devices, nil
}

// IsOwner reports whether deviceID belongs to userID. It satisfies
// internal/commandbus.OwnershipChecker, which the Command Bus's HTTP
// handler uses to make sure an operator can only send commands to their
// own devices.
func (s *Service) IsOwner(ctx context.Context, userID string, deviceID types.AgentID) (bool, error) {
	devices, err := s.List(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, d := range devices {
		if d.ID == deviceID {
			return true, nil
		}
	}
	return false, nil
}
