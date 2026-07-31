// Package config loads DeployOS configuration from, in ascending order of
// precedence: built-in defaults, a YAML config file, a .env file, and
// environment variables prefixed with DEPLOYOS_.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
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

	Agent       AgentConfig       `mapstructure:"agent"`
	Server      ServerConfig      `mapstructure:"server"`
	Supabase    SupabaseConfig    `mapstructure:"supabase"`
	DeviceToken DeviceTokenConfig `mapstructure:"device_token"`
}

// AgentConfig configures the DeployOS node agent (cmd/agent).
type AgentConfig struct {
	// HTTPAddr is the address the agent's health/status HTTP server
	// listens on, e.g. ":8081".
	HTTPAddr string `mapstructure:"http_addr"`
	// DataDir is where the agent persists its device ID and device
	// token across restarts. Defaults to "<home>/.deployos".
	DataDir string `mapstructure:"data_dir"`
	// APIBaseURL is the DeployOS control plane's base URL. The agent
	// only ever talks to this API - never to Supabase directly.
	APIBaseURL string `mapstructure:"api_base_url"`
	// GRPCServerAddr is the control plane's gRPC address, e.g.
	// "localhost:9090", for the persistent connection (see
	// docs/connection.md).
	GRPCServerAddr string `mapstructure:"grpc_server_addr"`
	// UserAccessToken is the operator's Supabase user access token,
	// used to authenticate device registration requests. Obtained
	// out-of-band until a proper login flow exists.
	UserAccessToken string `mapstructure:"user_access_token"`
	// DockerSocket is the path to the Docker daemon's unix socket, used
	// to observe containers via LIST_CONTAINERS/INSPECT_CONTAINER (see
	// docs/runtime.md). Defaults to "/var/run/docker.sock".
	DockerSocket string `mapstructure:"docker_socket"`
}

// ServerConfig configures the DeployOS control plane (cmd/server).
type ServerConfig struct {
	// HTTPAddr is the address the control plane's HTTP server listens
	// on, e.g. ":8080".
	HTTPAddr string `mapstructure:"http_addr"`
	// GRPCAddr is the address the control plane's gRPC server listens
	// on, e.g. ":9090", for the persistent agent connection (see
	// docs/connection.md).
	GRPCAddr string `mapstructure:"grpc_addr"`
}

// SupabaseConfig configures the control plane's connection to Supabase.
// The control plane is the only DeployOS component that talks to
// Supabase; agents never do.
type SupabaseConfig struct {
	// URL is the Supabase project URL, e.g. "https://xyz.supabase.co".
	URL string `mapstructure:"url"`
	// AnonKey is the Supabase project's anon/public API key, used to
	// authenticate user access tokens against Supabase Auth.
	AnonKey string `mapstructure:"anon_key"`
	// DatabaseURL is the Postgres connection string for the Supabase
	// project's database.
	DatabaseURL string `mapstructure:"database_url"`
}

// DeviceTokenConfig configures the tokens the control plane issues to
// agents on successful registration.
type DeviceTokenConfig struct {
	// Secret signs issued device tokens (HMAC). Required in production.
	Secret string `mapstructure:"secret"`
	// TTL is how long an issued device token remains valid.
	TTL time.Duration `mapstructure:"ttl"`
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
	decodeHook := viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
	))
	if err := v.Unmarshal(&cfg, decodeHook); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	if cfg.Agent.DataDir == "" {
		cfg.Agent.DataDir = defaultAgentDataDir()
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("environment", "development")
	v.SetDefault("log_level", "info")

	v.SetDefault("agent.http_addr", ":8081")
	v.SetDefault("agent.api_base_url", "http://localhost:8080")
	v.SetDefault("agent.grpc_server_addr", "localhost:9090")
	// Empty-string defaults below aren't meaningful values - they exist
	// so viper knows these keys exist and reads their environment
	// variable overrides. Without a default (or config file entry),
	// viper.Unmarshal silently ignores env vars for otherwise-unknown
	// keys even with AutomaticEnv enabled.
	v.SetDefault("agent.data_dir", "")
	v.SetDefault("agent.user_access_token", "")
	v.SetDefault("agent.docker_socket", "/var/run/docker.sock")

	v.SetDefault("server.http_addr", ":8080")
	v.SetDefault("server.grpc_addr", ":9090")

	v.SetDefault("supabase.url", "")
	v.SetDefault("supabase.anon_key", "")
	v.SetDefault("supabase.database_url", "")

	v.SetDefault("device_token.secret", "")
	v.SetDefault("device_token.ttl", "8760h")
}

func defaultAgentDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".deployos"
	}
	return filepath.Join(home, ".deployos")
}
