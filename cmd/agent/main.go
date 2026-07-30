// Command deployos-agent runs the DeployOS node agent: it loads
// configuration, starts structured logging, and serves a health endpoint
// until it receives a shutdown signal.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/saitadikonda99/deployOS/internal/agent"
	"github.com/saitadikonda99/deployOS/internal/config"
	"github.com/saitadikonda99/deployOS/internal/logging"
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
	agent.Version = version

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	a := agent.New(agent.Config{
		HTTPAddr:        cfg.Agent.HTTPAddr,
		DataDir:         cfg.Agent.DataDir,
		APIBaseURL:      cfg.Agent.APIBaseURL,
		GRPCServerAddr:  cfg.Agent.GRPCServerAddr,
		UserAccessToken: cfg.Agent.UserAccessToken,
	}, logger)

	logger.Info("starting deployos-agent", slog.String("version", version))

	if err := a.Run(ctx); err != nil {
		return err
	}

	logger.Info("deployos-agent stopped")
	return nil
}
