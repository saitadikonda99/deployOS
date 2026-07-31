// Command deployos-server runs the DeployOS control plane: it loads
// configuration, starts structured logging, and serves the device
// registration API, the persistent agent gRPC connection, and a health
// endpoint until it receives a shutdown signal.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	deployosv1 "github.com/saitadikonda99/deployOS/gen/go/deployos/v1"
	"github.com/saitadikonda99/deployOS/internal/auth"
	"github.com/saitadikonda99/deployOS/internal/config"
	"github.com/saitadikonda99/deployOS/internal/connection"
	"github.com/saitadikonda99/deployOS/internal/devices"
	"github.com/saitadikonda99/deployOS/internal/logging"
	"github.com/saitadikonda99/deployOS/internal/monitoring"
	"github.com/saitadikonda99/deployOS/internal/runtime"
	"github.com/saitadikonda99/deployOS/pkg/api"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(config.Options{})
	if err != nil {
		return err
	}

	logger := logging.New(cfg.LogLevel)

	if err := validateConfig(cfg); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.Supabase.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	registry := monitoring.NewRegistry()
	registry.Register("database", func(ctx context.Context) error {
		return pool.Ping(ctx)
	})

	authenticator := auth.NewSupabaseAuthenticator(cfg.Supabase.URL, cfg.Supabase.AnonKey)
	deviceRepo := devices.NewPostgresRepository(pool)
	tokenIssuer := devices.NewJWTTokenIssuer(cfg.DeviceToken.Secret, cfg.DeviceToken.TTL)
	deviceService := devices.NewService(deviceRepo, tokenIssuer, logger)

	connManager := connection.NewManager()
	deviceHandler := devices.NewHandler(deviceService, authenticator, connManager, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", registry.Handler())
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.VersionResponse{
			Name:    "deployos-server",
			Version: version,
		})
	})
	mux.HandleFunc("POST /api/v1/devices/register", deviceHandler.Register)
	mux.HandleFunc("GET /api/v1/devices", deviceHandler.List)

	httpHandler := logging.Middleware(logger)(mux)
	httpServer := runtime.NewHTTPServer(cfg.Server.HTTPAddr, httpHandler, logger)

	grpcServer := grpc.NewServer()
	connServer := connection.NewServer(connManager, tokenIssuer, logger)
	deployosv1.RegisterConnectionServiceServer(grpcServer, connServer)
	grpcRuntime := runtime.NewGRPCServer(cfg.Server.GRPCAddr, grpcServer, logger)

	logger.Info("starting deployos-server", slog.String("version", version))

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { return httpServer.Run(gctx) })
	g.Go(func() error { return grpcRuntime.Run(gctx) })

	if err := g.Wait(); err != nil {
		return err
	}

	logger.Info("deployos-server stopped")
	return nil
}

func validateConfig(cfg *config.Config) error {
	switch {
	case cfg.Supabase.DatabaseURL == "":
		return fmt.Errorf("DEPLOYOS_SUPABASE_DATABASE_URL is required")
	case cfg.Supabase.URL == "":
		return fmt.Errorf("DEPLOYOS_SUPABASE_URL is required")
	case cfg.Supabase.AnonKey == "":
		return fmt.Errorf("DEPLOYOS_SUPABASE_ANON_KEY is required")
	case cfg.DeviceToken.Secret == "":
		return fmt.Errorf("DEPLOYOS_DEVICE_TOKEN_SECRET is required")
	}
	return nil
}
