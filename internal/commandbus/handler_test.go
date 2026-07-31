package commandbus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/saitadikonda99/deployOS/internal/auth"
	"github.com/saitadikonda99/deployOS/pkg/api"
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

// fakeOwnership reports every device in owned as belonging to its
// mapped user.
type fakeOwnership struct {
	owned map[types.AgentID]string // deviceID -> userID
}

func (f *fakeOwnership) IsOwner(_ context.Context, userID string, deviceID types.AgentID) (bool, error) {
	return f.owned[deviceID] == userID, nil
}

// testHandlerServer bundles an httptest.Server exercising Handler with
// the fakeSender and Service backing it, so tests can both make HTTP
// requests and resolve the commands those requests generate.
type testHandlerServer struct {
	srv     *httptest.Server
	sender  *fakeSender
	service *Service
}

func newTestHandlerServer(sender *fakeSender) *testHandlerServer {
	if sender == nil {
		sender = newFakeSender()
	}

	svc := NewService(sender, testLogger())
	authr := &fakeAuthenticator{users: map[string]auth.User{
		"user-1-token": {ID: "user-1", Email: "user1@example.com"},
	}}
	ownership := &fakeOwnership{owned: map[types.AgentID]string{
		types.AgentID("device-1"): "user-1",
	}}
	handler := NewHandler(svc, authr, ownership, testLogger())

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/devices/{deviceID}/commands", handler.Send)

	return &testHandlerServer{srv: httptest.NewServer(mux), sender: sender, service: svc}
}

func (ts *testHandlerServer) Close() {
	ts.srv.Close()
}

func sendCommandBody(t *testing.T, kind string) *bytes.Reader {
	t.Helper()
	body, err := json.Marshal(api.SendCommandRequest{Kind: kind})
	if err != nil {
		t.Fatalf("marshaling request: %v", err)
	}
	return bytes.NewReader(body)
}

func TestHandlerSendRequiresAuthentication(t *testing.T) {
	ts := newTestHandlerServer(nil)
	defer ts.Close()

	resp, err := http.Post(ts.srv.URL+"/api/v1/devices/device-1/commands", "application/json", sendCommandBody(t, "PING"))
	if err != nil {
		t.Fatalf("POST error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestHandlerSendRejectsDeviceNotOwnedByCaller(t *testing.T) {
	ts := newTestHandlerServer(nil)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.srv.URL+"/api/v1/devices/someone-elses-device/commands", sendCommandBody(t, "PING"))
	req.Header.Set("Authorization", "Bearer user-1-token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandlerSendRejectsInvalidPayload(t *testing.T) {
	ts := newTestHandlerServer(nil)
	defer ts.Close()

	t.Run("malformed JSON", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, ts.srv.URL+"/api/v1/devices/device-1/commands", bytes.NewReader([]byte("not json")))
		req.Header.Set("Authorization", "Bearer user-1-token")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST error = %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})

	t.Run("empty kind", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, ts.srv.URL+"/api/v1/devices/device-1/commands", sendCommandBody(t, ""))
		req.Header.Set("Authorization", "Bearer user-1-token")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST error = %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})
}

func TestHandlerSendReportsDeviceNotConnected(t *testing.T) {
	fs := newFakeSender()
	fs.err = errors.New("no active stream")
	ts := newTestHandlerServer(fs)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.srv.URL+"/api/v1/devices/device-1/commands", sendCommandBody(t, "PING"))
	req.Header.Set("Authorization", "Bearer user-1-token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestHandlerSendSucceeds(t *testing.T) {
	ts := newTestHandlerServer(nil)
	defer ts.Close()

	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		req, _ := http.NewRequest(http.MethodPost, ts.srv.URL+"/api/v1/devices/device-1/commands", sendCommandBody(t, "PING"))
		req.Header.Set("Authorization", "Bearer user-1-token")
		resp, err := http.DefaultClient.Do(req)
		respCh <- resp
		errCh <- err
	}()

	ids := ts.sender.waitForIDs(t, 1)
	ts.service.HandleResult(types.AgentID("device-1"), resultEnvelope(ids[0], true, "pong"))

	resp := <-respCh
	if err := <-errCh; err != nil {
		t.Fatalf("POST error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body api.SendCommandResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if !body.Success || body.Message != "pong" {
		t.Errorf("response = %+v, want Success=true Message=pong", body)
	}
}
