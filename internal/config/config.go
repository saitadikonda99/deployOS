// Package config loads DeployOS configuration from, in ascending order of
// precedence: built-in defaults, a YAML config file, a .env file, and
// environment variables prefixed with DEPLOYOS_.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config is the configuration shared by every DeployOS component.
type Config struct {
	// Environment is a free-form deployment environment label, e.g.
	// "development" or "production".
	Environment string `mapstructure:"environment"`
	// LogLevel is one of "debug", "info", "warn", "error".
	LogLevel string `mapstructure:"log_level"`

	Agent  AgentConfig  `mapstructure:"agent"`
	Server ServerConfig `mapstructure:"server"`
}

// AgentConfig configures the DeployOS node agent (cmd/agent).
type AgentConfig struct {
	// ID identifies this agent within a fleet. Empty until the agent has
	// been registered with a control plane.
	ID string `mapstructure:"id"`
	// HTTPAddr is the address the agent's health/status HTTP server
	// listens on, e.g. ":8081".
	HTTPAddr string `mapstructure:"http_addr"`
}

// ServerConfig configures the DeployOS control plane (cmd/server).
type ServerConfig struct {
	// HTTPAddr is the address the control plane's HTTP server listens
	// on, e.g. ":8080".
	HTTPAddr string `mapstructure:"http_addr"`
}

// Options controls how Load locates its configuration file.
type Options struct {
	// FilePath is an explicit path to a YAML config file. If empty, Load
	// looks for "config.yaml" in the current directory and proceeds
	// without error if it isn't found.
	FilePath string
	// EnvFilePath is the .env file to load into the process environment
	// before reading configuration. Defaults to ".env" if empty. A
	// missing .env file is not an error.
	EnvFilePath string
}

// Load builds a Config from defaults, an optional YAML file, an optional
// .env file, and environment variables (highest precedence).
func Load(opts Options) (*Config, error) {
	envFile := opts.EnvFilePath
	if envFile == "" {
		envFile = ".env"
	}
	if err := godotenv.Load(envFile); err != nil && !os.IsNotExist(err) {
		// A missing .env file is expected in most environments; only
		// surface genuine read errors (permissions, malformed file).
		return nil, fmt.Errorf("loading %s: %w", envFile, err)
	}

	v := viper.New()
	setDefaults(v)

	v.SetEnvPrefix("DEPLOYOS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if opts.FilePath != "" {
		v.SetConfigFile(opts.FilePath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("environment", "development")
	v.SetDefault("log_level", "info")
	v.SetDefault("agent.http_addr", ":8081")
	v.SetDefault("server.http_addr", ":8080")
}
