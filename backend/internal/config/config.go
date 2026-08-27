// Package config reads the process configuration from the environment.
//
// Everything it produces is validated once, at startup, and never changes afterwards. A
// process that cannot be configured correctly refuses to start rather than serving with
// half its settings guessed.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// DefaultPort is where the server listens when PORT is not set. It is exported so that
// the health check probes the same port the server would have chosen.
const DefaultPort = "8080"

// Config is the validated configuration of the server.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	Port string
	// LogLevel is the minimum level written to the log.
	LogLevel slog.Level
	// CacheSize is how many calculations are kept in memory.
	CacheSize int
	// ShutdownTimeout bounds how long a graceful shutdown may take.
	ShutdownTimeout time.Duration
	// AllowedOrigins enables CORS for those browser origins. Empty means none, which is
	// correct in production where one origin serves both the app and the API.
	AllowedOrigins []string
}

// Load reads and validates the configuration.
//
// Every error names the environment variable at fault, because the person reading it is
// usually looking at a container that will not start.
func Load() (Config, error) {
	cfg := Config{
		Port:           lookup("PORT", DefaultPort),
		AllowedOrigins: splitList(lookup("ALLOWED_ORIGINS", "")),
	}

	if cfg.Port == "" {
		return Config{}, fmt.Errorf("PORT must not be empty")
	}

	cacheSize, err := strconv.Atoi(lookup("CACHE_SIZE", "1024"))
	if err != nil {
		return Config{}, fmt.Errorf("CACHE_SIZE must be a whole number: %w", err)
	}
	if cacheSize <= 0 {
		return Config{}, fmt.Errorf("CACHE_SIZE must be positive, got %d", cacheSize)
	}
	cfg.CacheSize = cacheSize

	shutdownTimeout, err := time.ParseDuration(lookup("SHUTDOWN_TIMEOUT", "10s"))
	if err != nil {
		return Config{}, fmt.Errorf("SHUTDOWN_TIMEOUT must be a duration such as 10s: %w", err)
	}
	if shutdownTimeout <= 0 {
		return Config{}, fmt.Errorf("SHUTDOWN_TIMEOUT must be positive, got %s", shutdownTimeout)
	}
	cfg.ShutdownTimeout = shutdownTimeout

	level, err := parseLevel(lookup("LOG_LEVEL", "info"))
	if err != nil {
		return Config{}, fmt.Errorf("LOG_LEVEL is not a known level: %w", err)
	}
	cfg.LogLevel = level

	return cfg, nil
}

func lookup(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// splitList reads a comma-separated list, ignoring surrounding spaces and empty entries.
func splitList(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseLevel(value string) (slog.Level, error) {
	var level slog.Level
	// slog parses the names it prints, so "debug", "INFO" and "warn+4" all work.
	if err := level.UnmarshalText([]byte(value)); err != nil {
		return 0, err
	}
	return level, nil
}
