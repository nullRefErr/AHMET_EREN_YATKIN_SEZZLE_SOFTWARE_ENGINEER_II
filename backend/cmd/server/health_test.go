package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckHealth(t *testing.T) {
	t.Parallel()

	t.Run("passes when the server reports it is alive", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(server.Close)

		require.NoError(t, checkHealth(t.Context(), server.URL))
	})

	t.Run("fails when the server reports it is not ready", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		t.Cleanup(server.Close)

		require.Error(t, checkHealth(t.Context(), server.URL))
	})

	t.Run("fails when nothing is listening", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		url := server.URL
		server.Close() // the address is now certain to be closed

		require.Error(t, checkHealth(t.Context(), url))
	})
}
