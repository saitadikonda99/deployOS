package types

import "testing"

func TestAgentIDValidate(t *testing.T) {
	if err := AgentID("node-1").Validate(); err != nil {
		t.Fatalf("expected non-empty AgentID to be valid, got %v", err)
	}

	if err := AgentID("").Validate(); err != ErrEmptyAgentID {
		t.Fatalf("expected ErrEmptyAgentID for empty AgentID, got %v", err)
	}
}

func TestVersionString(t *testing.T) {
	if got, want := (Version{Number: "0.1.0"}).String(), "0.1.0"; got != want {
		t.Fatalf("Version.String() = %q, want %q", got, want)
	}

	v := Version{Number: "0.1.0", Commit: "abc123"}
	if got, want := v.String(), "0.1.0 (abc123)"; got != want {
		t.Fatalf("Version.String() = %q, want %q", got, want)
	}
}
