// Integration test against a real Postgres database. Skipped unless
// DEPLOYOS_TEST_DATABASE_URL points at a database with the devices/users
// migrations applied (see supabase/migrations) - there is no live
// Supabase project available in most environments running this suite.
package devices

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saitadikonda99/deployOS/pkg/types"
)

func TestPostgresRepositoryUpsertAndList(t *testing.T) {
	dsn := os.Getenv("DEPLOYOS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("DEPLOYOS_TEST_DATABASE_URL not set; skipping Postgres integration test")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connecting to test database: %v", err)
	}
	defer pool.Close()

	// The devices.user_id foreign key references users(id), so the test
	// user must exist first.
	const userID = "00000000-0000-0000-0000-000000000001"
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email) VALUES ($1, 'integration-test@example.com')
		 ON CONFLICT (id) DO NOTHING`, userID); err != nil {
		t.Fatalf("seeding test user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM devices WHERE user_id = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	repo := NewPostgresRepository(pool)

	device := Device{
		ID:              types.AgentID("11111111-1111-1111-1111-111111111111"),
		UserID:          userID,
		Hostname:        "integration-box",
		OperatingSystem: "linux",
		Architecture:    "amd64",
		CPUCores:        4,
		MemoryBytes:     8589934592,
		DeployOSVersion: "0.1.0",
		Status:          StatusRegistered,
	}

	stored, err := repo.Upsert(ctx, device)
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if stored.Hostname != device.Hostname {
		t.Errorf("Hostname = %q, want %q", stored.Hostname, device.Hostname)
	}

	device.Hostname = "renamed-box"
	updated, err := repo.Upsert(ctx, device)
	if err != nil {
		t.Fatalf("second Upsert() error = %v", err)
	}
	if updated.Hostname != "renamed-box" {
		t.Errorf("Hostname after update = %q, want %q", updated.Hostname, "renamed-box")
	}
	if updated.CreatedAt != stored.CreatedAt {
		t.Errorf("CreatedAt changed on update: %v -> %v", stored.CreatedAt, updated.CreatedAt)
	}

	otherOwner := device
	otherOwner.UserID = "00000000-0000-0000-0000-000000000002"
	if _, err := repo.Upsert(ctx, otherOwner); err == nil {
		t.Fatal("expected Upsert() by a different owner to fail")
	}

	list, err := repo.ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListByUser() returned %d devices, want 1", len(list))
	}
}
