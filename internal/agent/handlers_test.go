package agent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/saitadikonda99/deployOS/internal/commandbus"
	"github.com/saitadikonda99/deployOS/internal/containers"
)

func testLoggerForAgent() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeRuntime is a containers.Runtime test double: no real container
// engine involved, just canned results/errors an executor test can
// control.
type fakeRuntime struct {
	containers []containers.Container
	listErr    error

	details    containers.ContainerDetails
	inspectErr error
}

func (f *fakeRuntime) ListContainers(context.Context) ([]containers.Container, error) {
	return f.containers, f.listErr
}

func (f *fakeRuntime) InspectContainer(_ context.Context, _ string) (containers.ContainerDetails, error) {
	return f.details, f.inspectErr
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

func TestListContainersExecutorSuccess(t *testing.T) {
	runtime := &fakeRuntime{containers: []containers.Container{{ID: "c1", Name: "web"}}}

	resp := listContainersExecutor(runtime).Execute(context.Background(), commandbus.Request{})

	if !resp.Success {
		t.Fatalf("Execute() Success = false, want true (Message: %s)", resp.Message)
	}

	var got []containers.Container
	if err := json.Unmarshal([]byte(resp.Details["containers"]), &got); err != nil {
		t.Fatalf("unmarshaling Details[containers]: %v", err)
	}
	if len(got) != 1 || got[0].ID != "c1" {
		t.Errorf("decoded containers = %+v, want one container with ID c1", got)
	}
}

func TestListContainersExecutorRuntimeError(t *testing.T) {
	runtime := &fakeRuntime{listErr: errors.New("docker daemon unreachable")}

	resp := listContainersExecutor(runtime).Execute(context.Background(), commandbus.Request{})

	if resp.Success {
		t.Errorf("Execute() Success = true, want false")
	}
	if resp.Message != "docker daemon unreachable" {
		t.Errorf("Execute() Message = %q, want %q", resp.Message, "docker daemon unreachable")
	}
}

func TestInspectContainerExecutorSuccess(t *testing.T) {
	runtime := &fakeRuntime{details: containers.ContainerDetails{
		Container: containers.Container{ID: "c1", Name: "web"},
	}}

	resp := inspectContainerExecutor(runtime).Execute(context.Background(), commandbus.Request{
		Arguments: map[string]string{"id": "c1"},
	})

	if !resp.Success {
		t.Fatalf("Execute() Success = false, want true (Message: %s)", resp.Message)
	}

	var got containers.ContainerDetails
	if err := json.Unmarshal([]byte(resp.Details["container"]), &got); err != nil {
		t.Fatalf("unmarshaling Details[container]: %v", err)
	}
	if got.ID != "c1" {
		t.Errorf("decoded container ID = %q, want c1", got.ID)
	}
}

func TestInspectContainerExecutorMissingID(t *testing.T) {
	resp := inspectContainerExecutor(&fakeRuntime{}).Execute(context.Background(), commandbus.Request{})

	if resp.Success {
		t.Errorf("Execute() Success = true, want false")
	}
}

func TestInspectContainerExecutorRuntimeError(t *testing.T) {
	runtime := &fakeRuntime{inspectErr: containers.ErrContainerNotFound}

	resp := inspectContainerExecutor(runtime).Execute(context.Background(), commandbus.Request{
		Arguments: map[string]string{"id": "nope"},
	})

	if resp.Success {
		t.Errorf("Execute() Success = true, want false")
	}
}

func TestNewDispatcherRegistersAllCommands(t *testing.T) {
	d := newDispatcher(testLoggerForAgent(), &fakeRuntime{})

	for _, kind := range []string{
		commandbus.KindPing,
		commandbus.KindGetVersion,
		commandbus.KindGetInfo,
		commandbus.KindListContainers,
	} {
		resp := d.Dispatch(context.Background(), commandbus.Request{ID: "cmd-1", Kind: kind})
		if !resp.Success {
			t.Errorf("Dispatch(%q) = %+v, want Success=true", kind, resp)
		}
	}

	resp := d.Dispatch(context.Background(), commandbus.Request{
		ID:        "cmd-1",
		Kind:      commandbus.KindInspectContainer,
		Arguments: map[string]string{"id": "c1"},
	})
	if !resp.Success {
		t.Errorf("Dispatch(%q) = %+v, want Success=true", commandbus.KindInspectContainer, resp)
	}

	resp = d.Dispatch(context.Background(), commandbus.Request{ID: "cmd-1", Kind: "NOT_A_COMMAND"})
	if resp.Success {
		t.Errorf("Dispatch(unknown) = %+v, want Success=false", resp)
	}
}
