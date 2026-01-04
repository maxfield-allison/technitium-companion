// technitium-companion syncs DNS A records to Technitium DNS based on Docker/Swarm Traefik labels.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/maxfield-allison/technitium-companion/internal/config"
	"github.com/maxfield-allison/technitium-companion/internal/docker"
	"github.com/maxfield-allison/technitium-companion/internal/health"
	"github.com/maxfield-allison/technitium-companion/internal/metrics"
	"github.com/maxfield-allison/technitium-companion/internal/reconciler"
	"github.com/maxfield-allison/technitium-companion/internal/technitium"
	"github.com/maxfield-allison/technitium-companion/internal/traefik"
	"github.com/maxfield-allison/technitium-companion/internal/watcher"
)

// Version and BuildDate are set via ldflags during build.
// Example: -ldflags="-X main.Version=v1.0.0 -X main.BuildDate=2026-01-03"
var (
	Version   = "dev"
	BuildDate = "unknown"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Set up structured logging
	logLevel := parseLogLevel(cfg.LogLevel)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	logger.Info("technitium-companion starting",
		slog.String("version", Version),
		slog.String("build_date", BuildDate),
		slog.String("log_level", cfg.LogLevel),
		slog.Bool("dry_run", cfg.DryRun),
	)

	// Initialize Prometheus metrics
	metrics.SetBuildInfo(Version, runtime.Version())
	metrics.SetUp()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Docker client
	dockerClient, err := docker.NewClient(ctx, cfg.DockerHost, docker.WithLogger(logger))
	if err != nil {
		return fmt.Errorf("creating docker client: %w", err)
	}
	defer dockerClient.Close()

	logger.Info("docker client connected",
		slog.String("mode", string(dockerClient.Mode())),
		slog.String("host", cfg.DockerHost),
	)

	// Initialize Technitium client
	techClient := technitium.NewClient(
		cfg.TechnitiumURL,
		cfg.TechnitiumToken,
		technitium.WithLogger(logger),
	)

	logger.Info("technitium client configured",
		slog.String("url", cfg.TechnitiumURL),
		slog.String("zone", cfg.TechnitiumZone),
		slog.String("target_ip", cfg.TargetIP),
	)

	// Initialize Traefik parser
	parser := traefik.NewParser(traefik.WithLogger(logger))

	// Initialize reconciler
	rec := reconciler.New(cfg, dockerClient, parser, techClient, reconciler.WithLogger(logger))

	// Initialize health server
	healthServer := health.New(cfg.HealthPort, health.WithLogger(logger), health.WithVersion(Version))

	// Register health checkers
	healthServer.RegisterChecker("docker", func(ctx context.Context) error {
		return dockerClient.Ping(ctx)
	})
	healthServer.RegisterChecker("technitium", func(ctx context.Context) error {
		// Simple check - try to get records for a non-existent hostname
		// This verifies API connectivity without modifying anything
		_, err := techClient.GetRecords(ctx, cfg.TechnitiumZone, "_health-check.invalid")
		// Ignore "record not found" errors - we just want to verify API is reachable
		// The API returns success with empty records if the hostname doesn't exist
		return err
	})

	// Start health server
	healthErrCh := healthServer.Start()

	// Run startup reconciliation if enabled
	if cfg.ReconcileOnStartup {
		logger.Info("running startup reconciliation")
		result, err := rec.Reconcile(ctx)
		if err != nil {
			logger.Error("startup reconciliation failed",
				slog.String("error", err.Error()),
			)
			// Continue anyway - we'll watch for events and reconcile later
		} else {
			logger.Info("startup reconciliation complete",
				slog.Int("workloads_scanned", result.WorkloadsScanned),
				slog.Int("records_created", result.RecordsCreated),
				slog.Int("records_existed", result.RecordsExisted),
				slog.Int("errors", len(result.Errors)),
			)
		}
	}

	// Mark as ready after startup reconciliation
	healthServer.SetReady(true)

	// Initialize and start event watcher
	eventWatcher := watcher.New(
		cfg,
		dockerClient.RawClient(),
		dockerClient.Mode(),
		parser,
		rec,
		watcher.WithLogger(logger),
	)

	// Channel to receive watcher errors
	watcherErrCh := make(chan error, 1)
	go func() {
		if err := eventWatcher.Watch(ctx); err != nil && err != context.Canceled {
			watcherErrCh <- err
		}
		close(watcherErrCh)
	}()

	logger.Info("technitium-companion running",
		slog.Int("health_port", cfg.HealthPort),
	)

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal or fatal error
	select {
	case sig := <-sigCh:
		logger.Info("received shutdown signal", slog.String("signal", sig.String()))
	case err := <-healthErrCh:
		if err != nil {
			logger.Error("health server error", slog.String("error", err.Error()))
		}
	case err := <-watcherErrCh:
		if err != nil {
			logger.Error("event watcher error", slog.String("error", err.Error()))
		}
	}

	// Graceful shutdown
	logger.Info("shutting down")
	cancel()

	// Give the watcher time to stop
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("health server shutdown error", slog.String("error", err.Error()))
	}

	logger.Info("technitium-companion stopped")
	return nil
}

// parseLogLevel converts a string log level to slog.Level.
func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
