package agent

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/saitadikonda99/deployOS/internal/commandbus"
)

func testLoggerForAgent() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestPingExecutor(t *testing.T) {
	resp := pingExecutor().Execute(context.Background(), commandbus.Request{})

	if !resp.Success || resp.Message != "pong" {
		t.Errorf("Execute() = %+v, want Success=true Message=pong", resp)
	}
}

func TestGetVersionExecutor(t *testing.T) {
	oldVersion := Version
	Version = "1.2.3-test"
	defer func() { Version = oldVersion }()

	resp := getVersionExecutor().Execute(context.Background(), commandbus.Request{})

	if !resp.Success || resp.Message != "1.2.3-test" {
		t.Errorf("Execute() = %+v, want Success=true Message=1.2.3-test", resp)
	}
	if resp.Details["version"] != "1.2.3-test" {
		t.Errorf("Details[version] = %q, want %q", resp.Details["version"], "1.2.3-test")
	}
}

func TestGetInfoExecutor(t *testing.T) {
	resp := getInfoExecutor().Execute(context.Background(), commandbus.Request{})

	if !resp.Success {
		t.Fatalf("Execute() Success = false, want true (Message: %s)", resp.Message)
	}
	for _, key := range []string{"hostname", "operating_system", "architecture", "cpu_cores", "memory_bytes", "deployos_version"} {
		if resp.Details[key] == "" {
			t.Errorf("Details[%q] is empty", key)
		}
	}
}

func TestNewDispatcherRegistersAllThreeCommands(t *testing.T) {
	d := newDispatcher(testLoggerForAgent())

	for _, kind := range []string{commandbus.KindPing, commandbus.KindGetVersion, commandbus.KindGetInfo} {
		resp := d.Dispatch(context.Background(), commandbus.Request{ID: "cmd-1", Kind: kind})
		if !resp.Success {
			t.Errorf("Dispatch(%q) = %+v, want Success=true", kind, resp)
		}
	}

	resp := d.Dispatch(context.Background(), commandbus.Request{ID: "cmd-1", Kind: "NOT_A_COMMAND"})
	if resp.Success {
		t.Errorf("Dispatch(unknown) = %+v, want Success=false", resp)
	}
}
