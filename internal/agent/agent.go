// Package agent implements the DeployOS node agent: the process that runs
// on each managed machine, reports its own health, and will eventually
// execute instructions from the control plane. This package holds no
// deployment logic yet - only the process foundation (config, logging,
// health reporting, graceful shutdown) that future features build on.
package agent

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/saitadikonda99/deployOS/internal/logging"
	"github.com/saitadikonda99/deployOS/internal/monitoring"
	"github.com/saitadikonda99/deployOS/internal/runtime"
	"github.com/saitadikonda99/deployOS/pkg/api"
	"github.com/saitadikonda99/deployOS/pkg/types"
)

// Version is set at build time (see cmd/agent) and reported on /version.
var Version = "dev"

// Config configures an Agent.
type Config struct {
	// ID identifies this agent within a fleet. May be empty if the agent
	// has not yet been registered with a control plane.
	ID types.AgentID
	// HTTPAddr is the address the agent's HTTP server listens on.
	HTTPAddr string
}

// Agent is a running DeployOS node agent.
type Agent struct {
	cfg      Config
	logger   *slog.Logger
	registry *monitoring.Registry
}

// New constructs an Agent. It does not start any network listeners; call
// Run to do that.
//
// The HTTP server built by Run is deliberately isolated behind this type
// so that a gRPC listener can be added alongside it later without
// changing how cmd/agent wires the agent up.
func New(cfg Config, logger *slog.Logger) *Agent {
	return &Agent{
		cfg:      cfg,
		logger:   logger,
		registry: monitoring.NewRegistry(),
	}
}

// Run starts the agent's HTTP server and blocks until ctx is canceled, at
// which point it shuts down gracefully.
func (a *Agent) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.registry.Handler())
	mux.HandleFunc("GET /version", a.handleVersion)

	handler := logging.Middleware(a.logger)(mux)

	server := runtime.NewHTTPServer(a.cfg.HTTPAddr, handler, a.logger)
	return server.Run(ctx)
}

func (a *Agent) handleVersion(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(api.VersionResponse{
		Name:    "deployos-agent",
		Version: Version,
	})
}
