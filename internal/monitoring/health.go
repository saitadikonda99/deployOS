// Package monitoring provides the health-check registry backing every
// DeployOS component's HTTP health endpoint. Metrics and alerting are
// future work; this package only establishes the health-check contract.
package monitoring

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/saitadikonda99/deployOS/pkg/api"
)

// Check reports whether a single dependency or subsystem is healthy.
type Check func(ctx context.Context) error

// Registry holds the set of health checks a component exposes through its
// HTTP health endpoint.
type Registry struct {
	mu     sync.RWMutex
	checks map[string]Check
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{checks: make(map[string]Check)}
}

// Register adds a named health check. Registering a name that already
// exists replaces the previous check.
func (r *Registry) Register(name string, check Check) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checks[name] = check
}

// Handler returns an http.HandlerFunc that runs every registered check and
// reports overall status as JSON. With no checks registered, it reports
// healthy - suitable as a liveness probe before any checks are added.
func (r *Registry) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		status := api.StatusOK

		r.mu.RLock()
		for _, check := range r.checks {
			if err := check(req.Context()); err != nil {
				status = api.StatusDegraded
				break
			}
		}
		r.mu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		if status != api.StatusOK {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: status})
	}
}
