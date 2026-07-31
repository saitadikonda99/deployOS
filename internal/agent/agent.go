// Package agent implements the DeployOS node agent: the process that runs
// on each managed machine, registers itself with the control plane,
// maintains a persistent authenticated connection to it, reports its own
// health, and will eventually execute instructions from the control
// plane. The agent talks only to the DeployOS API - never directly to
// Supabase.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/saitadikonda99/deployOS/internal/commandbus"
	"github.com/saitadikonda99/deployOS/internal/connection"
	"github.com/saitadikonda99/deployOS/internal/containers/docker"
	"github.com/saitadikonda99/deployOS/internal/logging"
	"github.com/saitadikonda99/deployOS/internal/monitoring"
	"github.com/saitadikonda99/deployOS/internal/runtime"
	"github.com/saitadikonda99/deployOS/pkg/api"
	"github.com/saitadikonda99/deployOS/pkg/protocol"
	"github.com/saitadikonda99/deployOS/pkg/types"
)

// Version is set at build time (see cmd/agent) and reported on /version,
// to the control plane during registration, and during authentication.
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
	// APIBaseURL is the DeployOS control plane's base URL, used for
	// device registration.
	APIBaseURL string
	// GRPCServerAddr is the control plane's gRPC address, used for the
	// persistent connection (see docs/connection.md).
	GRPCServerAddr string
	// UserAccessToken authenticates device registration requests. It is
	// the operator's Supabase user access token, obtained out-of-band
	// until a proper login flow exists.
	UserAccessToken string
	// DockerSocket is the path to the Docker daemon's unix socket, used
	// by the LIST_CONTAINERS/INSPECT_CONTAINER commands (see
	// docs/runtime.md).
	DockerSocket string
}

// Agent is a running DeployOS node agent.
type Agent struct {
	cfg        Config
	logger     *slog.Logger
	registry   *monitoring.Registry
	registrar  *registrar
	conn       *connection.Client
	dispatcher *commandbus.Dispatcher
}

// New constructs an Agent. It does not start any network listeners,
// register the device, or connect to the control plane; call Run to do
// that.
func New(cfg Config, logger *slog.Logger) *Agent {
	return &Agent{
		cfg:        cfg,
		logger:     logger,
		registry:   monitoring.NewRegistry(),
		registrar:  newRegistrar(cfg.APIBaseURL),
		conn:       connection.NewClient(cfg.GRPCServerAddr, logger),
		dispatcher: newDispatcher(logger, docker.NewRuntime(cfg.DockerSocket)),
	}
}

// Run loads (or creates) the agent's identity, registers it with the
// control plane, then runs the persistent gRPC connection and the HTTP
// health server concurrently until ctx is canceled, at which point both
// shut down gracefully.
//
// A registration failure is logged but doesn't stop the agent: the
// connection client independently retries with backoff using whatever
// device token is currently on disk, so a transient registration failure
// (network, credentials) doesn't need to be fatal - the connection will
// start succeeding as soon as registration does, without a restart.
func (a *Agent) Run(ctx context.Context) error {
	deviceID, err := loadOrCreateDeviceID(a.cfg.DataDir)
	if err != nil {
		return fmt.Errorf("loading device identity: %w", err)
	}

	info, err := collectSystemInfo()
	if err != nil {
		return fmt.Errorf("collecting system info: %w", err)
	}

	a.registerWithControlPlane(ctx, deviceID, info)
	a.conn.OnCommand(commandbus.WireHandler(a.dispatcher))

	a.registry.Register("control_plane_connection", func(context.Context) error {
		if !a.conn.Connected() {
			return errors.New("not connected to control plane")
		}
		return nil
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.registry.Handler())
	mux.HandleFunc("GET /version", a.handleVersion)
	httpHandler := logging.Middleware(a.logger)(mux)
	httpServer := runtime.NewHTTPServer(a.cfg.HTTPAddr, httpHandler, a.logger)

	deviceInfo := connection.DeviceInfo{
		ID:              deviceID,
		Hostname:        info.Hostname,
		OperatingSystem: info.OperatingSystem,
		Architecture:    info.Architecture,
		DeployOSVersion: Version,
	}

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { return a.conn.Run(gctx, deviceInfo, a.loadToken) })
	g.Go(func() error { return httpServer.Run(gctx) })

	return g.Wait()
}

// loadToken is the connection.TokenSource the agent's gRPC client uses:
// it re-reads the persisted device token on every (re)connect attempt,
// so it always authenticates with whatever registration most recently
// wrote.
func (a *Agent) loadToken() (string, error) {
	token, ok, err := loadDeviceToken(a.cfg.DataDir)
	if err != nil {
		return "", fmt.Errorf("loading device token: %w", err)
	}
	if !ok {
		return "", errors.New("no device token persisted yet; waiting for registration")
	}
	return token, nil
}

func (a *Agent) registerWithControlPlane(ctx context.Context, deviceID types.AgentID, info systemInfo) {
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
