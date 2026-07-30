package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateDeviceIDPersistsAndReuses(t *testing.T) {
	dir := t.TempDir()

	first, err := loadOrCreateDeviceID(dir)
	if err != nil {
		t.Fatalf("first loadOrCreateDeviceID() error = %v", err)
	}
	if first == "" {
		t.Fatal("loadOrCreateDeviceID() returned an empty ID")
	}

	second, err := loadOrCreateDeviceID(dir)
	if err != nil {
		t.Fatalf("second loadOrCreateDeviceID() error = %v", err)
	}

	if first != second {
		t.Fatalf("device ID changed across calls: %q -> %q", first, second)
	}
}

func TestLoadOrCreateDeviceIDCreatesDataDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "agent-data")

	if _, err := loadOrCreateDeviceID(dir); err != nil {
		t.Fatalf("loadOrCreateDeviceID() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, deviceIDFileName)); err != nil {
		t.Fatalf("device id file not created: %v", err)
	}
}

func TestPersistDeviceTokenRoundTrips(t *testing.T) {
	dir := t.TempDir()

	if err := persistDeviceToken(dir, "secret-token"); err != nil {
		t.Fatalf("persistDeviceToken() error = %v", err)
	}

	got, ok, err := readTrimmedFile(dir, deviceTokenFileName)
	if err != nil {
		t.Fatalf("readTrimmedFile() error = %v", err)
	}
	if !ok {
		t.Fatal("readTrimmedFile() found no token file")
	}
	if got != "secret-token" {
		t.Errorf("token = %q, want %q", got, "secret-token")
	}
}
