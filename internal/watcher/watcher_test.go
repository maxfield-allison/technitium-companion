// Package watcher provides tests for the Docker event watcher.
package watcher

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/docker/docker/api/types/events"

	"github.com/maxfield-allison/technitium-companion/internal/config"
	"github.com/maxfield-allison/technitium-companion/internal/docker"
)

// TestWithLogger verifies the logger option works correctly.
func TestWithLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	opt := WithLogger(logger)

	w := &Watcher{}
	opt(w)

	if w.logger != logger {
		t.Error("WithLogger did not set the logger correctly")
	}
}

// TestWithDebounceInterval verifies the debounce interval option.
func TestWithDebounceInterval(t *testing.T) {
	interval := 10 * time.Second
	opt := WithDebounceInterval(interval)

	w := &Watcher{}
	opt(w)

	if w.debounceInterval != interval {
		t.Errorf("expected debounce interval %v, got %v", interval, w.debounceInterval)
	}
}

// TestNew verifies the watcher is created with correct defaults.
func TestNew(t *testing.T) {
	cfg := &config.Config{
		DryRun: true,
	}

	w := New(cfg, nil, docker.ModeSwarm, nil, nil)

	if w.cfg != cfg {
		t.Error("config not set correctly")
	}
	if w.dockerMode != docker.ModeSwarm {
		t.Errorf("expected mode swarm, got %s", w.dockerMode)
	}
	if w.debounceInterval != 5*time.Second {
		t.Errorf("expected default debounce 5s, got %v", w.debounceInterval)
	}
}

// TestNew_WithOptions verifies options are applied correctly.
func TestNew_WithOptions(t *testing.T) {
	cfg := &config.Config{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	debounce := 15 * time.Second

	w := New(
		cfg, nil, docker.ModeStandalone, nil, nil,
		WithLogger(logger),
		WithDebounceInterval(debounce),
	)

	if w.logger != logger {
		t.Error("logger option not applied")
	}
	if w.debounceInterval != debounce {
		t.Errorf("debounce option not applied: expected %v, got %v", debounce, w.debounceInterval)
	}
}

// TestBuildEventFilters_Swarm verifies correct filters for Swarm mode.
func TestBuildEventFilters_Swarm(t *testing.T) {
	w := &Watcher{
		dockerMode: docker.ModeSwarm,
	}

	filters := w.buildEventFilters()

	// Check type filter
	typeFilter := filters.Get("type")
	if len(typeFilter) != 1 || typeFilter[0] != "service" {
		t.Errorf("expected type filter [service], got %v", typeFilter)
	}

	// Check event filters
	eventFilter := filters.Get("event")
	expectedEvents := []string{"create", "update", "remove"}
	if len(eventFilter) != len(expectedEvents) {
		t.Errorf("expected %d event filters, got %d", len(expectedEvents), len(eventFilter))
	}

	for _, expected := range expectedEvents {
		found := false
		for _, actual := range eventFilter {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected event filter %q not found in %v", expected, eventFilter)
		}
	}
}

// TestBuildEventFilters_Standalone verifies correct filters for standalone mode.
func TestBuildEventFilters_Standalone(t *testing.T) {
	w := &Watcher{
		dockerMode: docker.ModeStandalone,
	}

	filters := w.buildEventFilters()

	// Check type filter
	typeFilter := filters.Get("type")
	if len(typeFilter) != 1 || typeFilter[0] != "container" {
		t.Errorf("expected type filter [container], got %v", typeFilter)
	}

	// Check event filters
	eventFilter := filters.Get("event")
	expectedEvents := []string{"start", "die", "destroy"}
	if len(eventFilter) != len(expectedEvents) {
		t.Errorf("expected %d event filters, got %d", len(expectedEvents), len(eventFilter))
	}

	for _, expected := range expectedEvents {
		found := false
		for _, actual := range eventFilter {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected event filter %q not found in %v", expected, eventFilter)
		}
	}
}

// TestHandleEvent_ServiceCreate tests handling of service create events.
func TestHandleEvent_ServiceCreate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	w := &Watcher{
		dockerMode: docker.ModeSwarm,
		logger:     logger,
	}

	event := events.Message{
		Type:   events.ServiceEventType,
		Action: "create",
		Actor: events.Actor{
			ID: "service-123",
			Attributes: map[string]string{
				"name": "my-service",
			},
		},
	}

	// This should not panic
	w.handleEvent(context.Background(), event)
}

// TestHandleEvent_ServiceUpdate tests handling of service update events.
func TestHandleEvent_ServiceUpdate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	w := &Watcher{
		dockerMode: docker.ModeSwarm,
		logger:     logger,
	}

	event := events.Message{
		Type:   events.ServiceEventType,
		Action: "update",
		Actor: events.Actor{
			ID: "service-123",
			Attributes: map[string]string{
				"name": "my-service",
			},
		},
	}

	// This should not panic
	w.handleEvent(context.Background(), event)
}

// TestHandleEvent_ServiceRemove tests handling of service remove events.
func TestHandleEvent_ServiceRemove(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	w := &Watcher{
		dockerMode: docker.ModeSwarm,
		logger:     logger,
	}

	event := events.Message{
		Type:   events.ServiceEventType,
		Action: "remove",
		Actor: events.Actor{
			ID: "service-123",
			Attributes: map[string]string{
				"name": "my-service",
			},
		},
	}

	// This should not panic
	w.handleEvent(context.Background(), event)
}

// TestHandleEvent_ContainerStart tests handling of container start events.
func TestHandleEvent_ContainerStart(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	w := &Watcher{
		dockerMode: docker.ModeStandalone,
		logger:     logger,
	}

	event := events.Message{
		Type:   events.ContainerEventType,
		Action: "start",
		Actor: events.Actor{
			ID: "container-456",
			Attributes: map[string]string{
				"name": "my-container",
			},
		},
	}

	// This should not panic
	w.handleEvent(context.Background(), event)
}

// TestHandleEvent_ContainerDie tests handling of container die events.
func TestHandleEvent_ContainerDie(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	w := &Watcher{
		dockerMode: docker.ModeStandalone,
		logger:     logger,
	}

	event := events.Message{
		Type:   events.ContainerEventType,
		Action: "die",
		Actor: events.Actor{
			ID: "container-456",
			Attributes: map[string]string{
				"name": "my-container",
			},
		},
	}

	// This should not panic
	w.handleEvent(context.Background(), event)
}

// TestHandleEvent_MissingName tests handling when name attribute is missing.
func TestHandleEvent_MissingName(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	w := &Watcher{
		dockerMode: docker.ModeSwarm,
		logger:     logger,
	}

	// Event with no name attribute - should use truncated ID
	event := events.Message{
		Type:   events.ServiceEventType,
		Action: "create",
		Actor: events.Actor{
			ID:         "abcdef123456789",
			Attributes: map[string]string{},
		},
	}

	// This should not panic and should use truncated ID
	w.handleEvent(context.Background(), event)
}

// TestEventHandler type is exported for testing.
func TestEventHandlerType(t *testing.T) {
	var handler EventHandler = func(ctx context.Context, event events.Message) {
		// Test handler
	}

	if handler == nil {
		t.Error("EventHandler should not be nil")
	}
}
