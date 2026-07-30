package devices

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saitadikonda99/deployOS/pkg/types"
)

// PostgresRepository stores devices in the Supabase Postgres database.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository builds a PostgresRepository backed by pool.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Upsert implements Repository.
//
// The UPDATE half of the upsert is guarded by "WHERE devices.user_id =
// excluded.user_id": if a device with this ID already exists under a
// different user, that guard fails, the conflict is not resolved, and
// RETURNING yields no row. That is the only way this statement can
// return zero rows, so it unambiguously signals ErrOwnedByAnotherUser.
func (r *PostgresRepository) Upsert(ctx context.Context, device Device) (Device, error) {
	const query = `
		INSERT INTO devices (
			id, user_id, hostname, operating_system, architecture,
			cpu_cores, memory_bytes, deployos_version, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			hostname = excluded.hostname,
			operating_system = excluded.operating_system,
			architecture = excluded.architecture,
			cpu_cores = excluded.cpu_cores,
			memory_bytes = excluded.memory_bytes,
			deployos_version = excluded.deployos_version,
			updated_at = now()
		WHERE devices.user_id = excluded.user_id
		RETURNING id, user_id, hostname, operating_system, architecture,
			cpu_cores, memory_bytes, deployos_version, status, created_at, updated_at`

	rows, err := r.pool.Query(ctx, query,
		device.ID, device.UserID, device.Hostname, device.OperatingSystem, device.Architecture,
		device.CPUCores, device.MemoryBytes, device.DeployOSVersion, device.Status,
	)
	if err != nil {
		return Device{}, fmt.Errorf("upserting device: %w", err)
	}
	defer rows.Close()

	stored, err := pgx.CollectExactlyOneRow(rows, scanDevice)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Device{}, ErrOwnedByAnotherUser
		}
		return Device{}, fmt.Errorf("upserting device: %w", err)
	}

	return stored, nil
}

// ListByUser implements Repository.
func (r *PostgresRepository) ListByUser(ctx context.Context, userID string) ([]Device, error) {
	const query = `
		SELECT id, user_id, hostname, operating_system, architecture,
			cpu_cores, memory_bytes, deployos_version, status, created_at, updated_at
		FROM devices
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing devices: %w", err)
	}
	defer rows.Close()

	result, err := pgx.CollectRows(rows, scanDevice)
	if err != nil {
		return nil, fmt.Errorf("listing devices: %w", err)
	}

	return result, nil
}

func scanDevice(row pgx.CollectableRow) (Device, error) {
	var d Device
	var id string
	err := row.Scan(
		&id, &d.UserID, &d.Hostname, &d.OperatingSystem, &d.Architecture,
		&d.CPUCores, &d.MemoryBytes, &d.DeployOSVersion, &d.Status, &d.CreatedAt, &d.UpdatedAt,
	)
	d.ID = types.AgentID(id)
	return d, err
}
