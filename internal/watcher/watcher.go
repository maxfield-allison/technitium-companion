// Package watcher subscribes to Docker events and triggers reconciliation.
package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

	"github.com/maxfield-allison/technitium-companion/internal/config"
	"github.com/maxfield-allison/technitium-companion/internal/docker"
	"github.com/maxfield-allison/technitium-companion/internal/metrics"
	"github.com/maxfield-allison/technitium-companion/internal/reconciler"
	"github.com/maxfield-allison/technitium-companion/internal/traefik"
)

// EventHandler is called when a relevant Docker event is received.
type EventHandler func(ctx context.Context, event events.Message)

// Watcher subscribes to Docker events and triggers DNS reconciliation.
type Watcher struct {
	cfg        *config.Config
	docker     *client.Client
	dockerMode docker.Mode
	parser     *traefik.Parser
	reconciler *reconciler.Reconciler
	logger     *slog.Logger

	// Debounce settings to avoid reconciling too frequently
	debounceInterval time.Duration
}

// Option is a functional option for configuring the Watcher.
type Option func(*Watcher)

// WithLogger sets a custom logger.
func WithLogger(logger *slog.Logger) Option {
	return func(w *Watcher) {
		w.logger = logger
	}
}

// WithDebounceInterval sets the debounce interval for event processing.
// Events occurring within this interval will trigger a single reconciliation.
func WithDebounceInterval(d time.Duration) Option {
	return func(w *Watcher) {
		w.debounceInterval = d
	}
}

// New creates a new Watcher.
func New(
	cfg *config.Config,
	dockerClient *client.Client,
	dockerMode docker.Mode,
	parser *traefik.Parser,
	rec *reconciler.Reconciler,
	opts ...Option,
) *Watcher {
	w := &Watcher{
		cfg:              cfg,
		docker:           dockerClient,
		dockerMode:       dockerMode,
		parser:           parser,
		reconciler:       rec,
		logger:           slog.Default(),
		debounceInterval: 5 * time.Second, // Default debounce
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

// Watch starts watching for Docker events and triggers reconciliation.
// This method blocks until the context is cancelled.
func (w *Watcher) Watch(ctx context.Context) error {
	w.logger.Info("starting event watcher",
		slog.String("mode", string(w.dockerMode)),
		slog.Duration("debounce", w.debounceInterval),
	)

	// Build event filters based on Docker mode
	filterArgs := w.buildEventFilters()

	// Subscribe to Docker events
	eventsCh, errCh := w.docker.Events(ctx, events.ListOptions{
		Filters: filterArgs,
	})

	// Debounce channel for batching events
	var debounceTimer *time.Timer
	pendingReconcile := false

	for {
		select {
		case <-ctx.Done():
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			w.logger.Info("event watcher stopped")
			return ctx.Err()

		case err := <-errCh:
			if err != nil {
				w.logger.Error("event stream error",
					slog.String("error", err.Error()),
				)
				return fmt.Errorf("event stream error: %w", err)
			}

		case event := <-eventsCh:
			w.handleEvent(ctx, event)

			// Debounce: schedule a full reconciliation
			if !pendingReconcile {
				pendingReconcile = true
				debounceTimer = time.AfterFunc(w.debounceInterval, func() {
					w.logger.Debug("debounce timer fired, triggering full reconciliation")
					result, err := w.reconciler.Reconcile(ctx)
					if err != nil {
						w.logger.Error("reconciliation failed",
							slog.String("error", err.Error()),
						)
					} else {
						w.logger.Info("reconciliation triggered by events",
							slog.Int("records_created", result.RecordsCreated),
							slog.Int("records_existed", result.RecordsExisted),
						)
					}
					pendingReconcile = false
				})
			}
		}
	}
}

// buildEventFilters creates Docker event filters based on the operating mode.
func (w *Watcher) buildEventFilters() filters.Args {
	f := filters.NewArgs()

	if w.dockerMode == docker.ModeSwarm {
		// Watch Swarm service events
		f.Add("type", string(events.ServiceEventType))
		f.Add("event", "create")
		f.Add("event", "update")
		f.Add("event", "remove")
	} else {
		// Watch container events in standalone mode
		f.Add("type", string(events.ContainerEventType))
		f.Add("event", "start")
		f.Add("event", "die")
		f.Add("event", "destroy")
	}

	return f
}

// handleEvent processes a single Docker event.
func (w *Watcher) handleEvent(ctx context.Context, event events.Message) {
	// Record the event metric
	metrics.RecordDockerEvent(string(event.Type), string(event.Action))

	w.logger.Debug("received event",
		slog.String("type", string(event.Type)),
		slog.String("action", string(event.Action)),
		slog.String("actor_id", event.Actor.ID),
		slog.Any("attributes", event.Actor.Attributes),
	)

	switch event.Type {
	case events.ServiceEventType:
		w.handleServiceEvent(ctx, event)
	case events.ContainerEventType:
		w.handleContainerEvent(ctx, event)
	}
}

// handleServiceEvent processes Swarm service events.
func (w *Watcher) handleServiceEvent(ctx context.Context, event events.Message) {
	serviceName := event.Actor.Attributes["name"]
	if serviceName == "" {
		serviceName = event.Actor.ID[:12]
	}

	switch event.Action {
	case "create", "update":
		w.logger.Info("service event received",
			slog.String("action", string(event.Action)),
			slog.String("service", serviceName),
		)
		// Full reconciliation will be triggered by debounce timer

	case "remove":
		w.logger.Info("service removed",
			slog.String("service", serviceName),
		)
		// Note: We don't auto-delete DNS records for removed services
		// because orphan cleanup is disabled by design.
		// DNS records are intentionally left until manually cleaned up.
		w.logger.Debug("orphan cleanup disabled - DNS records not removed",
			slog.String("service", serviceName),
		)
	}
}

// handleContainerEvent processes standalone container events.
func (w *Watcher) handleContainerEvent(ctx context.Context, event events.Message) {
	containerName := event.Actor.Attributes["name"]
	if containerName == "" {
		containerName = event.Actor.ID[:12]
	}

	switch event.Action {
	case "start":
		w.logger.Info("container started",
			slog.String("container", containerName),
		)
		// Full reconciliation will be triggered by debounce timer

	case "die", "destroy":
		w.logger.Info("container stopped/destroyed",
			slog.String("container", containerName),
		)
		// Note: We don't auto-delete DNS records for stopped containers
		// because orphan cleanup is disabled by design.
		w.logger.Debug("orphan cleanup disabled - DNS records not removed",
			slog.String("container", containerName),
		)
	}
}

// WatchWithHandler starts watching for Docker events and calls a custom handler.
// This is useful for testing or custom event processing.
func (w *Watcher) WatchWithHandler(ctx context.Context, handler EventHandler) error {
	w.logger.Info("starting event watcher with custom handler",
		slog.String("mode", string(w.dockerMode)),
	)

	filterArgs := w.buildEventFilters()

	eventsCh, errCh := w.docker.Events(ctx, events.ListOptions{
		Filters: filterArgs,
	})

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("event watcher stopped")
			return ctx.Err()

		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("event stream error: %w", err)
			}

		case event := <-eventsCh:
			handler(ctx, event)
		}
	}
}
