package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/saitadikonda99/deployOS/pkg/api"
)

func TestHandlerWithNoChecksReportsOK(t *testing.T) {
	r := NewRegistry()

	rec := httptest.NewRecorder()
	r.Handler()(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body api.HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != api.StatusOK {
		t.Errorf("status = %q, want %q", body.Status, api.StatusOK)
	}
}

func TestHandlerReportsDegradedWhenACheckFails(t *testing.T) {
	r := NewRegistry()
	r.Register("always-fails", func(_ context.Context) error {
		return errors.New("boom")
	})

	rec := httptest.NewRecorder()
	r.Handler()(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var body api.HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != api.StatusDegraded {
		t.Errorf("status = %q, want %q", body.Status, api.StatusDegraded)
	}
}
