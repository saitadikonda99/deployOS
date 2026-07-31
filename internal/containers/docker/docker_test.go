package docker

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/saitadikonda99/deployOS/internal/containers"
)

func TestListContainersSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/containers/json" {
			t.Errorf("path = %q, want /containers/json", r.URL.Path)
		}
		if r.URL.Query().Get("all") != "true" {
			t.Errorf("all query param = %q, want true", r.URL.Query().Get("all"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"Id":"c1","Names":["/web"],"Image":"nginx:latest","Created":1700000000,"State":"running","Status":"Up 3 hours"},
			{"Id":"c2","Names":["/db"],"Image":"postgres:16","Created":1700000100,"State":"exited","Status":"Exited (0) 2 hours ago"}
		]`))
	}))
	defer srv.Close()

	runtime := newRuntimeWithClient(srv.URL, srv.Client())

	got, err := runtime.ListContainers(context.Background())
	if err != nil {
		t.Fatalf("ListContainers() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListContainers() returned %d containers, want 2", len(got))
	}

	want := containers.Container{
		ID:      "c1",
		Name:    "web",
		Image:   "nginx:latest",
		Status:  "Up 3 hours",
		State:   "running",
		Created: time.Unix(1700000000, 0).UTC(),
	}
	if got[0] != want {
		t.Errorf("ListContainers()[0] = %+v, want %+v", got[0], want)
	}
}

func TestListContainersEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	runtime := newRuntimeWithClient(srv.URL, srv.Client())

	got, err := runtime.ListContainers(context.Background())
	if err != nil {
		t.Fatalf("ListContainers() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("ListContainers() = %+v, want empty", got)
	}
}

func TestListContainersDaemonError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"something broke"}`))
	}))
	defer srv.Close()

	runtime := newRuntimeWithClient(srv.URL, srv.Client())

	_, err := runtime.ListContainers(context.Background())
	if err == nil {
		t.Fatal("ListContainers() error = nil, want an error")
	}
}

func TestInspectContainerSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/containers/c1/json" {
			t.Errorf("path = %q, want /containers/c1/json", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"Id": "c1",
			"Name": "/web",
			"Created": "2024-01-01T00:00:00Z",
			"State": {"Status": "running"},
			"Config": {"Image": "nginx:latest", "Cmd": ["nginx", "-g", "daemon off;"], "Env": ["FOO=bar"]},
			"RestartCount": 2,
			"NetworkSettings": {
				"Ports": {"80/tcp": [{"HostPort": "8080"}]},
				"Networks": {"bridge": {}}
			},
			"Mounts": [{"Source": "/host/data", "Destination": "/data", "Mode": "rw"}]
		}`))
	}))
	defer srv.Close()

	runtime := newRuntimeWithClient(srv.URL, srv.Client())

	got, err := runtime.InspectContainer(context.Background(), "c1")
	if err != nil {
		t.Fatalf("InspectContainer() error = %v", err)
	}

	if got.ID != "c1" || got.Name != "web" || got.Image != "nginx:latest" || got.State != "running" {
		t.Errorf("InspectContainer() container fields = %+v", got.Container)
	}
	if got.RestartCount != 2 {
		t.Errorf("RestartCount = %d, want 2", got.RestartCount)
	}
	if len(got.Command) != 3 || got.Command[0] != "nginx" {
		t.Errorf("Command = %+v", got.Command)
	}
	if len(got.Env) != 1 || got.Env[0] != "FOO=bar" {
		t.Errorf("Env = %+v", got.Env)
	}
	if len(got.Mounts) != 1 || got.Mounts[0] != (containers.Mount{Source: "/host/data", Destination: "/data", Mode: "rw"}) {
		t.Errorf("Mounts = %+v", got.Mounts)
	}
	if len(got.Ports) != 1 || got.Ports[0] != (containers.Port{ContainerPort: 80, HostPort: 8080, Protocol: "tcp"}) {
		t.Errorf("Ports = %+v", got.Ports)
	}
	if len(got.Networks) != 1 || got.Networks[0] != "bridge" {
		t.Errorf("Networks = %+v", got.Networks)
	}
}

func TestInspectContainerNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"No such container: nope"}`))
	}))
	defer srv.Close()

	runtime := newRuntimeWithClient(srv.URL, srv.Client())

	_, err := runtime.InspectContainer(context.Background(), "nope")
	if !errors.Is(err, containers.ErrContainerNotFound) {
		t.Errorf("InspectContainer() error = %v, want ErrContainerNotFound", err)
	}
}
