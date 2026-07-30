package config

import "testing"

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
