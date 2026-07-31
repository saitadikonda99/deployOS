// Integration tests for the devices HTTP surface: real Service, real
// Handler, and real HTTP round-trips over httptest, but a fake
// Repository/Authenticator standing in for Supabase. This exercises the
// full handler -> service -> repository wiring and HTTP semantics
// without needing a live database or Supabase project.
package devices

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/saitadikonda99/deployOS/internal/auth"
	"github.com/saitadikonda99/deployOS/pkg/api"
	"github.com/saitadikonda99/deployOS/pkg/protocol"
	"github.com/saitadikonda99/deployOS/pkg/types"
)

type fakeAuthenticator struct {
	users map[string]auth.User
}

func (f *fakeAuthenticator) Authenticate(_ context.Context, token string) (auth.User, error) {
	user, ok := f.users[token]
	if !ok {
		return auth.User{}, auth.ErrInvalidToken
	}
	return user, nil
}

// fakeConnectionStatusProvider reports every device as disconnected,
// which is all these HTTP-layer tests need - connection-status behavior
// itself is covered in internal/connection.
type fakeConnectionStatusProvider struct{}

func (fakeConnectionStatusProvider) IsConnected(types.AgentID) bool { return false }

func newTestServer() (*httptest.Server, *fakeAuthenticator) {
	authr := &fakeAuthenticator{users: map[string]auth.User{
		"user-1-token": {ID: "user-1", Email: "user1@example.com"},
		"user-2-token": {ID: "user-2", Email: "user2@example.com"},
	}}

	svc := NewService(newFakeRepository(), &fakeTokenIssuer{token: "signed-token"}, testLogger())
	handler := NewHandler(svc, authr, fakeConnectionStatusProvider{}, testLogger())

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/devices/register", handler.Register)
	mux.HandleFunc("GET /api/v1/devices", handler.List)

	return httptest.NewServer(mux), authr
}

func registerRequestBody(t *testing.T, deviceID string) *bytes.Reader {
	t.Helper()
	body, err := json.Marshal(protocol.DeviceRegisterRequest{
		DeviceID:        types.AgentID(deviceID),
		Hostname:        "dev-box",
		OperatingSystem: "linux",
		Architecture:    "amd64",
		CPUCores:        8,
		MemoryBytes:     17179869184,
		DeployOSVersion: "0.1.0",
	})
	if err != nil {
		t.Fatalf("marshaling request: %v", err)
	}
	return bytes.NewReader(body)
}

func TestRegisterHandlerRequiresAuthentication(t *testing.T) {
	srv, _ := newTestServer()
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/devices/register", "application/json",
		registerRequestBody(t, "11111111-1111-1111-1111-111111111111"))
	if err != nil {
		t.Fatalf("POST error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestRegisterHandlerSucceeds(t *testing.T) {
	srv, _ := newTestServer()
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/devices/register",
		registerRequestBody(t, "11111111-1111-1111-1111-111111111111"))
	req.Header.Set("Authorization", "Bearer user-1-token")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var body protocol.DeviceRegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if body.Token != "signed-token" {
		t.Errorf("Token = %q, want %q", body.Token, "signed-token")
	}
	if body.DeviceID.String() != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("DeviceID = %q, want %q", body.DeviceID, "11111111-1111-1111-1111-111111111111")
	}
}

func TestRegisterHandlerRejectsInvalidBody(t *testing.T) {
	srv, _ := newTestServer()
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/devices/register", bytes.NewReader([]byte("not json")))
	req.Header.Set("Authorization", "Bearer user-1-token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestRegisterHandlerConflictOnDifferentOwner(t *testing.T) {
	srv, _ := newTestServer()
	defer srv.Close()

	const deviceID = "22222222-2222-2222-2222-222222222222"

	req1, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/devices/register", registerRequestBody(t, deviceID))
	req1.Header.Set("Authorization", "Bearer user-1-token")
	resp1, err := http.DefaultClient.Do(req1)
	if err != nil {
		t.Fatalf("first POST error = %v", err)
	}
	_ = resp1.Body.Close()
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("first status = %d, want %d", resp1.StatusCode, http.StatusCreated)
	}

	req2, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/devices/register", registerRequestBody(t, deviceID))
	req2.Header.Set("Authorization", "Bearer user-2-token")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("second POST error = %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()

	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("second status = %d, want %d", resp2.StatusCode, http.StatusConflict)
	}
}

func TestListHandlerReturnsOnlyOwnDevices(t *testing.T) {
	srv, _ := newTestServer()
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/devices/register",
		registerRequestBody(t, "33333333-3333-3333-3333-333333333333"))
	req.Header.Set("Authorization", "Bearer user-1-token")
	if _, err := http.DefaultClient.Do(req); err != nil {
		t.Fatalf("seeding device: %v", err)
	}

	listReq, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/devices", nil)
	listReq.Header.Set("Authorization", "Bearer user-2-token")
	resp, err := http.DefaultClient.Do(listReq)
	if err != nil {
		t.Fatalf("GET error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body api.ListDevicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(body.Devices) != 0 {
		t.Fatalf("Devices = %d, want 0 (user-2 owns nothing)", len(body.Devices))
	}
}

// connectedDeviceProvider reports a single device ID as connected and
// everything else as disconnected.
type connectedDeviceProvider struct {
	connected types.AgentID
}

func (p connectedDeviceProvider) IsConnected(deviceID types.AgentID) bool {
	return deviceID == p.connected
}

func TestListHandlerReportsLiveConnectionStatus(t *testing.T) {
	const deviceID = "44444444-4444-4444-4444-444444444444"

	authr := &fakeAuthenticator{users: map[string]auth.User{
		"user-1-token": {ID: "user-1", Email: "user1@example.com"},
	}}
	svc := NewService(newFakeRepository(), &fakeTokenIssuer{token: "signed-token"}, testLogger())
	handler := NewHandler(svc, authr, connectedDeviceProvider{connected: types.AgentID(deviceID)}, testLogger())

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/devices/register", handler.Register)
	mux.HandleFunc("GET /api/v1/devices", handler.List)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	regReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/devices/register", registerRequestBody(t, deviceID))
	regReq.Header.Set("Authorization", "Bearer user-1-token")
	if _, err := http.DefaultClient.Do(regReq); err != nil {
		t.Fatalf("registering device: %v", err)
	}

	listReq, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/devices", nil)
	listReq.Header.Set("Authorization", "Bearer user-1-token")
	resp, err := http.DefaultClient.Do(listReq)
	if err != nil {
		t.Fatalf("GET error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var body api.ListDevicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(body.Devices) != 1 {
		t.Fatalf("Devices = %d, want 1", len(body.Devices))
	}
	if body.Devices[0].Status != connectionStatusOnline {
		t.Errorf("Status = %q, want %q", body.Devices[0].Status, connectionStatusOnline)
	}
}
