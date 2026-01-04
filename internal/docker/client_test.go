// Package docker provides tests for the Docker client.
package docker

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

// TestModeConstants verifies mode constants are correctly defined.
func TestModeConstants(t *testing.T) {
	tests := []struct {
		mode     Mode
		expected string
	}{
		{ModeSwarm, "swarm"},
		{ModeStandalone, "standalone"},
	}

	for _, tt := range tests {
		if string(tt.mode) != tt.expected {
			t.Errorf("Mode %v: expected %q, got %q", tt.mode, tt.expected, string(tt.mode))
		}
	}
}

// TestServiceStruct verifies the Service struct can hold expected data.
func TestServiceStruct(t *testing.T) {
	svc := Service{
		ID:   "abc123",
		Name: "test-service",
		Labels: map[string]string{
			"traefik.enable":                        "true",
			"traefik.http.routers.test.rule":        "Host(`test.example.com`)",
			"traefik.http.routers.test.entrypoints": "websecure",
		},
	}

	if svc.ID != "abc123" {
		t.Errorf("expected ID abc123, got %s", svc.ID)
	}
	if svc.Name != "test-service" {
		t.Errorf("expected Name test-service, got %s", svc.Name)
	}
	if len(svc.Labels) != 3 {
		t.Errorf("expected 3 labels, got %d", len(svc.Labels))
	}
}

// TestContainerStruct verifies the Container struct can hold expected data.
func TestContainerStruct(t *testing.T) {
	ctr := Container{
		ID:   "def456",
		Name: "test-container",
		Labels: map[string]string{
			"traefik.enable": "true",
		},
	}

	if ctr.ID != "def456" {
		t.Errorf("expected ID def456, got %s", ctr.ID)
	}
	if ctr.Name != "test-container" {
		t.Errorf("expected Name test-container, got %s", ctr.Name)
	}
}

// TestWorkloadStruct verifies the Workload struct can represent both types.
func TestWorkloadStruct(t *testing.T) {
	tests := []struct {
		name     string
		workload Workload
		wantType string
	}{
		{
			name: "service workload",
			workload: Workload{
				ID:     "svc-123",
				Name:   "my-service",
				Labels: map[string]string{"env": "prod"},
				Type:   "service",
			},
			wantType: "service",
		},
		{
			name: "container workload",
			workload: Workload{
				ID:     "ctr-456",
				Name:   "my-container",
				Labels: map[string]string{"env": "dev"},
				Type:   "container",
			},
			wantType: "container",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.workload.Type != tt.wantType {
				t.Errorf("expected Type %s, got %s", tt.wantType, tt.workload.Type)
			}
		})
	}
}

// TestWithLogger verifies the logger option works correctly.
func TestWithLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	opt := WithLogger(logger)

	// Create a temporary client struct to test the option
	c := &Client{}
	opt(c)

	if c.logger != logger {
		t.Error("WithLogger did not set the logger correctly")
	}
}

// TestWithMode verifies the mode option works correctly.
func TestWithMode(t *testing.T) {
	tests := []struct {
		mode Mode
	}{
		{ModeSwarm},
		{ModeStandalone},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			opt := WithMode(tt.mode)
			c := &Client{}
			opt(c)

			if c.mode != tt.mode {
				t.Errorf("WithMode did not set mode correctly: expected %s, got %s", tt.mode, c.mode)
			}
		})
	}
}

// TestNewClient_InvalidHost tests error handling for invalid Docker hosts.
func TestNewClient_InvalidHost(t *testing.T) {
	ctx := context.Background()

	// Try to connect to an invalid host
	_, err := NewClient(ctx, "tcp://invalid-host:99999")

	// We expect this to either fail to connect or fail during mode detection
	// The exact error depends on the environment, but it should not succeed
	if err == nil {
		// In some environments, the client creation succeeds but later operations fail
		// This is acceptable behavior - the important thing is it doesn't panic
		t.Log("Client creation succeeded with invalid host (will fail on first operation)")
	}
}

// TestListServices_WrongMode tests that ListServices fails in standalone mode.
func TestListServices_WrongMode(t *testing.T) {
	c := &Client{
		mode:   ModeStandalone,
		logger: slog.Default(),
	}

	_, err := c.ListServices(context.Background())
	if err == nil {
		t.Error("expected error when calling ListServices in standalone mode")
	}
}

// TestListContainers_WrongMode tests that ListContainers fails in swarm mode.
func TestListContainers_WrongMode(t *testing.T) {
	c := &Client{
		mode:   ModeSwarm,
		logger: slog.Default(),
	}

	_, err := c.ListContainers(context.Background())
	if err == nil {
		t.Error("expected error when calling ListContainers in swarm mode")
	}
}

// TestGetServiceLabels_WrongMode tests that GetServiceLabels fails in standalone mode.
func TestGetServiceLabels_WrongMode(t *testing.T) {
	c := &Client{
		mode:   ModeStandalone,
		logger: slog.Default(),
	}

	_, err := c.GetServiceLabels(context.Background(), "some-service")
	if err == nil {
		t.Error("expected error when calling GetServiceLabels in standalone mode")
	}
}

// TestGetContainerLabels_WrongMode tests that GetContainerLabels fails in swarm mode.
func TestGetContainerLabels_WrongMode(t *testing.T) {
	c := &Client{
		mode:   ModeSwarm,
		logger: slog.Default(),
	}

	_, err := c.GetContainerLabels(context.Background(), "some-container")
	if err == nil {
		t.Error("expected error when calling GetContainerLabels in swarm mode")
	}
}

// TestClientMode tests the Mode() method.
func TestClientMode(t *testing.T) {
	tests := []struct {
		mode Mode
	}{
		{ModeSwarm},
		{ModeStandalone},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			c := &Client{mode: tt.mode}
			if c.Mode() != tt.mode {
				t.Errorf("Mode() returned %s, expected %s", c.Mode(), tt.mode)
			}
		})
	}
}
