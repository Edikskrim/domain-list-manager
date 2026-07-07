package config

import (
	"os"
	"strconv"
)

// Config holds all application configuration values.
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Auth       AuthConfig
	Fetcher    FetcherConfig
	Builder    BuilderConfig
	HTTPClient HTTPClientConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host string
	Port int
}

// DatabaseConfig holds database settings.
type DatabaseConfig struct {
	Path string
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	Username string
	Password string
}

// FetcherConfig holds fetcher settings.
type FetcherConfig struct {
	Timeout    int
	MaxRetries int
	MaxBodySize int64
	MaxRedirects int
}

// BuilderConfig holds builder settings.
type BuilderConfig struct {
	SnapshotCount int
	OutputPath string
}

// HTTPClientConfig holds HTTP client settings.
type HTTPClientConfig struct {
	UserAgent string
}

// DefaultConfig returns configuration with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Path: "data/domains.db",
		},
		Auth: AuthConfig{
			Username: "admin",
			Password: "admin",
		},
		Fetcher: FetcherConfig{
			Timeout:    30,
			MaxRetries: 3,
			MaxBodySize: 50 * 1024 * 1024,
			MaxRedirects: 10,
		},
		Builder: BuilderConfig{
			SnapshotCount: 10,
			OutputPath:    "output/domains.lst",
		},
		HTTPClient: HTTPClientConfig{
			UserAgent: "DomainListManager/1.0",
		},
	}
}

// LoadConfig creates a config from defaults, then overrides with environment variables.
// Supported environment variables:
//   - SERVER_HOST, SERVER_PORT, DB_PATH, AUTH_USERNAME, AUTH_PASSWORD,
//     FETCHER_TIMEOUT, FETCHER_MAX_RETRIES, FETCHER_MAX_BODY_SIZE, FETCHER_MAX_REDIRECTS,
//     BUILDER_SNAPSHOT_COUNT, BUILDER_OUTPUT_PATH
func LoadConfig() Config {
	cfg := DefaultConfig()

	if v := os.Getenv("SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.Database.Path = v
	}
	if v := os.Getenv("AUTH_USERNAME"); v != "" {
		cfg.Auth.Username = v
	}
	if v := os.Getenv("AUTH_PASSWORD"); v != "" {
		cfg.Auth.Password = v
	}
	if v := os.Getenv("FETCHER_TIMEOUT"); v != "" {
		if t, err := strconv.Atoi(v); err == nil {
			cfg.Fetcher.Timeout = t
		}
	}
	if v := os.Getenv("FETCHER_MAX_RETRIES"); v != "" {
		if r, err := strconv.Atoi(v); err == nil {
			cfg.Fetcher.MaxRetries = r
		}
	}
	if v := os.Getenv("FETCHER_MAX_BODY_SIZE"); v != "" {
		if s, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.Fetcher.MaxBodySize = s
		}
	}
	if v := os.Getenv("FETCHER_MAX_REDIRECTS"); v != "" {
		if r, err := strconv.Atoi(v); err == nil {
			cfg.Fetcher.MaxRedirects = r
		}
	}
	if v := os.Getenv("BUILDER_SNAPSHOT_COUNT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Builder.SnapshotCount = n
		}
	}
	if v := os.Getenv("BUILDER_OUTPUT_PATH"); v != "" {
		cfg.Builder.OutputPath = v
	}

	return cfg
}
