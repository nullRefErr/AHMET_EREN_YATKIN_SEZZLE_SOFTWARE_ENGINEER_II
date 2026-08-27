package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
)

const healthCheckTimeout = 3 * time.Second

// checkHealth asks a running server whether it is alive.
//
// The container image is distroless: it has no shell, no package manager and no curl, so
// there is nothing available to write a health check with except the binary itself. Run
// as "server -healthcheck", this is what Docker calls.
func checkHealth(ctx context.Context, url string) error {
	ctx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build health request: %w", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("reach %s: %w", url, err)
	}
	// Closing cannot change the verdict — the status line has already been received — but
	// the failure is still reported rather than dropped.
	defer func() {
		if err := response.Body.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "close health response body:", err)
		}
	}()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("health endpoint answered %s", response.Status)
	}
	return nil
}
