// Command deployos-server runs the DeployOS control plane: it loads
// configuration, starts structured logging, and serves a health endpoint
// until it receives a shutdown signal.
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/saitadikonda99/deployOS/internal/config"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	registry := monitoring.NewRegistry()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", registry.Handler())
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.VersionResponse{
			Name:    "deployos-server",
			Version: version,
		})
	})

	handler := logging.Middleware(logger)(mux)
	server := runtime.NewHTTPServer(cfg.Server.HTTPAddr, handler, logger)

	logger.Info("starting deployos-server", slog.String("version", version))

	if err := server.Run(ctx); err != nil {
		return err
	}

	logger.Info("deployos-server stopped")
	return nil
}
