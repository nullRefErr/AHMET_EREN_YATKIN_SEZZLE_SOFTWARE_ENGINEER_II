package config_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"calculator/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	got, err := config.Load()

	require.NoError(t, err)
	require.Equal(t, "8080", got.Port)
	require.Equal(t, 1024, got.CacheSize)
	require.Equal(t, 10*time.Second, got.ShutdownTimeout)
	require.Equal(t, slog.LevelInfo, got.LogLevel)
	require.Empty(t, got.AllowedOrigins)
}

func TestLoadReadsTheEnvironment(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("CACHE_SIZE", "32")
	t.Setenv("SHUTDOWN_TIMEOUT", "3s")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("ALLOWED_ORIGINS", "http://localhost:3000, http://localhost:5173")

	got, err := config.Load()

	require.NoError(t, err)
	require.Equal(t, "9090", got.Port)
	require.Equal(t, 32, got.CacheSize)
	require.Equal(t, 3*time.Second, got.ShutdownTimeout)
	require.Equal(t, slog.LevelDebug, got.LogLevel)
	require.Equal(t, []string{"http://localhost:3000", "http://localhost:5173"}, got.AllowedOrigins)
}

// A misconfigured process must refuse to start. A service that is half up is harder to
// diagnose than one that is plainly down.
func TestLoadRejectsBadValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"cache size is not a number", "CACHE_SIZE", "plenty"},
		{"cache size is zero", "CACHE_SIZE", "0"},
		{"cache size is negative", "CACHE_SIZE", "-1"},
		{"shutdown timeout is not a duration", "SHUTDOWN_TIMEOUT", "soon"},
		{"shutdown timeout is negative", "SHUTDOWN_TIMEOUT", "-5s"},
		{"log level is unknown", "LOG_LEVEL", "chatty"},
		{"port is empty", "PORT", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, tt.value)

			_, err := config.Load()

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.key, "the error should name the variable to fix")
		})
	}
}
