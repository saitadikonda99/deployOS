package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(Options{})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Environment != "development" {
		t.Errorf("Environment = %q, want %q", cfg.Environment, "development")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.Agent.HTTPAddr != ":8081" {
		t.Errorf("Agent.HTTPAddr = %q, want %q", cfg.Agent.HTTPAddr, ":8081")
	}
	if cfg.Server.HTTPAddr != ":8080" {
		t.Errorf("Server.HTTPAddr = %q, want %q", cfg.Server.HTTPAddr, ":8080")
	}
	if cfg.Agent.APIBaseURL != "http://localhost:8080" {
		t.Errorf("Agent.APIBaseURL = %q, want %q", cfg.Agent.APIBaseURL, "http://localhost:8080")
	}
	if cfg.Agent.DataDir == "" {
		t.Error("Agent.DataDir = \"\", want a non-empty default")
	}
	if cfg.DeviceToken.TTL != 8760*time.Hour {
		t.Errorf("DeviceToken.TTL = %v, want %v", cfg.DeviceToken.TTL, 8760*time.Hour)
	}
}

func TestLoadAgentDataDirOverride(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "agent-data")
	t.Setenv("DEPLOYOS_AGENT_DATA_DIR", dir)

	cfg, err := Load(Options{})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Agent.DataDir != dir {
		t.Errorf("Agent.DataDir = %q, want %q", cfg.Agent.DataDir, dir)
	}
}

func TestLoadDeviceTokenTTLFromEnv(t *testing.T) {
	t.Setenv("DEPLOYOS_DEVICE_TOKEN_TTL", "1h")
	t.Setenv("DEPLOYOS_DEVICE_TOKEN_SECRET", "test-secret")

	cfg, err := Load(Options{})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DeviceToken.TTL != time.Hour {
		t.Errorf("DeviceToken.TTL = %v, want %v", cfg.DeviceToken.TTL, time.Hour)
	}
	if cfg.DeviceToken.Secret != "test-secret" {
		t.Errorf("DeviceToken.Secret = %q, want %q", cfg.DeviceToken.Secret, "test-secret")
	}
}

func TestLoadEnvOverridesDefaults(t *testing.T) {
	t.Setenv("DEPLOYOS_LOG_LEVEL", "debug")
	t.Setenv("DEPLOYOS_AGENT_HTTP_ADDR", ":9091")

	cfg, err := Load(Options{})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.Agent.HTTPAddr != ":9091" {
		t.Errorf("Agent.HTTPAddr = %q, want %q", cfg.Agent.HTTPAddr, ":9091")
	}
}
