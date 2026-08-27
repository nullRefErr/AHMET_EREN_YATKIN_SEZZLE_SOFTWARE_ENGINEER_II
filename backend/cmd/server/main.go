// Command server runs the calculator HTTP API.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"calculator/internal/config"
	"calculator/internal/repository/memory"
	"calculator/internal/service"
	"calculator/internal/transport/rest"
)

// version is stamped at build time with -ldflags "-X main.version=$TAG".
var version = "dev"

// Timeouts on the server itself, not on a handler. ReadHeaderTimeout is the one that
// matters most: without it a client can hold a connection open by sending headers one byte
// at a time, which is all a Slowloris attack is.
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second
)

func main() {
	// Docker runs the binary with this flag as the container health check; there is no
	// curl inside a distroless image to do it instead.
	healthcheck := flag.Bool("healthcheck", false, "probe the running server and exit")
	flag.Parse()

	if *healthcheck {
		if err := checkHealth(context.Background(), "http://127.0.0.1:"+port()+"/healthz"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if err := run(); err != nil {
		slog.Error("server stopped with an error", "error", err)
		os.Exit(1)
	}
}

// port is where the server listens, read the same way whether the process is serving or
// probing so the two can never disagree.
func port() string {
	if value := os.Getenv("PORT"); value != "" {
		return value
	}
	return config.DefaultPort
}

// run holds the whole lifecycle so that every failure can return an error instead of
// calling os.Exit from somewhere deep, which would skip the deferred cleanup.
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("read configuration: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)
	gin.SetMode(gin.ReleaseMode)

	cache, err := memory.NewCache(cfg.CacheSize)
	if err != nil {
		return fmt.Errorf("create calculation cache: %w", err)
	}

	// The only place the application is wired together.
	router := rest.NewRouter(service.NewCalculator(cache, logger), rest.Config{
		Logger:         logger,
		AllowedOrigins: cfg.AllowedOrigins,
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	// Docker sends SIGTERM and waits. Ignoring it means every deploy cuts off whatever
	// requests were in flight.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	listenFailed := make(chan error, 1)
	go func() {
		defer close(listenFailed)

		logger.Info("server starting", "address", server.Addr, "version", version)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			listenFailed <- err
		}
	}()

	select {
	case err := <-listenFailed:
		return fmt.Errorf("listen on %s: %w", server.Addr, err)
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shut down gracefully within %s: %w", cfg.ShutdownTimeout, err)
	}
	logger.Info("server stopped")
	return nil
}
