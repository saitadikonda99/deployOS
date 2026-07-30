package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/saitadikonda99/deployOS/pkg/api"
	"github.com/saitadikonda99/deployOS/pkg/protocol"
	"github.com/saitadikonda99/deployOS/pkg/types"
)

func TestRegistrarRegisterSuccess(t *testing.T) {
	var gotAuth, gotMethod, gotPath string
	var gotBody protocol.DeviceRegisterRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(protocol.DeviceRegisterResponse{
			DeviceID:  gotBody.DeviceID,
			Token:     "signed-token",
			ExpiresAt: time.Now().Add(time.Hour).UTC(),
		})
	}))
	defer srv.Close()

	r := newRegistrar(srv.URL)
	req := protocol.DeviceRegisterRequest{
		DeviceID:        types.AgentID("11111111-1111-1111-1111-111111111111"),
		Hostname:        "dev-box",
		OperatingSystem: "linux",
		Architecture:    "amd64",
		CPUCores:        8,
		MemoryBytes:     17179869184,
		DeployOSVersion: "0.1.0",
	}

	resp, err := r.register(context.Background(), "user-token", req)
	if err != nil {
		t.Fatalf("register() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/api/v1/devices/register" {
		t.Errorf("path = %q, want %q", gotPath, "/api/v1/devices/register")
	}
	if gotAuth != "Bearer user-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer user-token")
	}
	if gotBody != req {
		t.Errorf("request body = %+v, want %+v", gotBody, req)
	}
	if resp.Token != "signed-token" {
		t.Errorf("Token = %q, want %q", resp.Token, "signed-token")
	}
}

func TestRegistrarRegisterSurfacesServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(api.ErrorResponse{Error: "device is already registered to another account"})
	}))
	defer srv.Close()

	r := newRegistrar(srv.URL)

	_, err := r.register(context.Background(), "user-token", protocol.DeviceRegisterRequest{
		DeviceID: types.AgentID("11111111-1111-1111-1111-111111111111"),
	})
	if err == nil {
		t.Fatal("expected an error for a non-201 response")
	}
}
