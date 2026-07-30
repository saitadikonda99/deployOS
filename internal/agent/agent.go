// Package agent implements the DeployOS node agent: the process that runs
// on each managed machine, registers itself with the control plane,
// reports its own health, and will eventually execute instructions from
// the control plane. The agent talks only to the DeployOS API - never
// directly to Supabase.
package agent

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/saitadikonda99/deployOS/internal/logging"
	"github.com/saitadikonda99/deployOS/internal/monitoring"
	"github.com/saitadikonda99/deployOS/internal/runtime"
	"github.com/saitadikonda99/deployOS/pkg/api"
	"github.com/saitadikonda99/deployOS/pkg/protocol"
)

// Version is set at build time (see cmd/agent) and reported on /version
// and to the control plane during registration.
var Version = "dev"

// registrationTimeout bounds how long the agent waits for the control
// plane to respond to a registration request.
const registrationTimeout = 30 * time.Second

// Config configures an Agent.
type Config struct {
	// HTTPAddr is the address the agent's HTTP server listens on.
	HTTPAddr string
	// DataDir is where the agent persists its device ID and device
	// token across restarts.
	DataDir string
	// APIBaseURL is the DeployOS control plane's base URL.
	APIBaseURL string
	// UserAccessToken authenticates device registration requests. It is
	// the operator's Supabase user access token, obtained out-of-band
	// until a proper login flow exists.
	UserAccessToken string
}

// Agent is a running DeployOS node agent.
type Agent struct {
	cfg       Config
	logger    *slog.Logger
	registry  *monitoring.Registry
	registrar *registrar
}

// New constructs an Agent. It does not start any network listeners or
// register the device; call Run to do that.
//
// The HTTP server built by Run is deliberately isolated behind this type
// so that a gRPC listener can be added alongside it later without
// changing how cmd/agent wires the agent up.
func New(cfg Config, logger *slog.Logger) *Agent {
	return &Agent{
		cfg:       cfg,
		logger:    logger,
		registry:  monitoring.NewRegistry(),
		registrar: newRegistrar(cfg.APIBaseURL),
	}
}

// Run registers the agent with the control plane, then starts the
// agent's HTTP server and blocks until ctx is canceled, at which point it
// shuts down gracefully.
//
// A registration failure is logged but does not stop the agent from
// serving its health endpoint: retrying registration is future work, so
// for now an operator who sees the error can restart the agent once the
// underlying problem (network, credentials) is fixed.
func (a *Agent) Run(ctx context.Context) error {
	a.registerDevice(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.registry.Handler())
	mux.HandleFunc("GET /version", a.handleVersion)

	handler := logging.Middleware(a.logger)(mux)

	server := runtime.NewHTTPServer(a.cfg.HTTPAddr, handler, a.logger)
	return server.Run(ctx)
}

func (a *Agent) registerDevice(ctx context.Context) {
	deviceID, err := loadOrCreateDeviceID(a.cfg.DataDir)
	if err != nil {
		a.logger.Error("loading device identity", slog.Any("error", err))
		return
	}

	info, err := collectSystemInfo()
	if err != nil {
		a.logger.Error("collecting system info", slog.Any("error", err))
		return
	}

	regCtx, cancel := context.WithTimeout(ctx, registrationTimeout)
	defer cancel()

	resp, err := a.registrar.register(regCtx, a.cfg.UserAccessToken, protocol.DeviceRegisterRequest{
		DeviceID:        deviceID,
		Hostname:        info.Hostname,
		OperatingSystem: info.OperatingSystem,
		Architecture:    info.Architecture,
		CPUCores:        info.CPUCores,
		MemoryBytes:     info.MemoryBytes,
		DeployOSVersion: Version,
	})
	if err != nil {
		a.logger.Error("registering device", slog.Any("error", err))
		return
	}

	if err := persistDeviceToken(a.cfg.DataDir, resp.Token); err != nil {
		a.logger.Error("persisting device token", slog.Any("error", err))
		return
	}

	a.logger.Info("device registered",
		slog.String("device_id", resp.DeviceID.String()),
		slog.Time("expires_at", resp.ExpiresAt),
	)
}

func (a *Agent) handleVersion(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(api.VersionResponse{
		Name:    "deployos-agent",
		Version: Version,
	})
}
