// Package docker provides a client for interacting with Docker in both standalone and Swarm modes.
package docker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// Mode represents the Docker runtime mode.
type Mode string

const (
	// ModeSwarm indicates Docker is running in Swarm mode.
	ModeSwarm Mode = "swarm"
	// ModeStandalone indicates Docker is running in standalone mode.
	ModeStandalone Mode = "standalone"
)

// Service represents a Docker Swarm service with relevant fields for DNS management.
type Service struct {
	ID     string
	Name   string
	Labels map[string]string
}

// Container represents a Docker container with relevant fields for DNS management.
type Container struct {
	ID     string
	Name   string
	Labels map[string]string
}

// Client wraps the Docker client with convenience methods.
type Client struct {
	docker *client.Client
	mode   Mode
	logger *slog.Logger
}

// ClientOption is a functional option for configuring the Client.
type ClientOption func(*Client)

// WithLogger sets a custom logger.
func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithMode forces a specific Docker mode instead of auto-detecting.
func WithMode(mode Mode) ClientOption {
	return func(c *Client) {
		c.mode = mode
	}
}

// NewClient creates a new Docker client.
// If host is empty, uses the DOCKER_HOST environment variable or default socket.
func NewClient(ctx context.Context, host string, opts ...ClientOption) (*Client, error) {
	var dockerOpts []client.Opt

	dockerOpts = append(dockerOpts, client.FromEnv)
	dockerOpts = append(dockerOpts, client.WithAPIVersionNegotiation())

	if host != "" {
		dockerOpts = append(dockerOpts, client.WithHost(host))
	}

	dockerClient, err := client.NewClientWithOpts(dockerOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}

	c := &Client{
		docker: dockerClient,
		logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(c)
	}

	// Auto-detect mode if not explicitly set
	if c.mode == "" {
		detectedMode, err := c.detectMode(ctx)
		if err != nil {
			dockerClient.Close()
			return nil, fmt.Errorf("detecting docker mode: %w", err)
		}
		c.mode = detectedMode
	}

	c.logger.Info("docker client initialized",
		slog.String("mode", string(c.mode)),
	)

	return c, nil
}

// detectMode determines whether Docker is running in Swarm or standalone mode.
func (c *Client) detectMode(ctx context.Context) (Mode, error) {
	info, err := c.docker.Info(ctx)
	if err != nil {
		return "", fmt.Errorf("getting docker info: %w", err)
	}

	c.logger.Debug("docker info retrieved",
		slog.String("swarm_state", string(info.Swarm.LocalNodeState)),
		slog.String("node_id", info.Swarm.NodeID),
	)

	if info.Swarm.LocalNodeState == swarm.LocalNodeStateActive {
		// Verify we're on a manager node
		if !info.Swarm.ControlAvailable {
			return "", fmt.Errorf("swarm mode detected but this node is not a manager - cannot list services")
		}
		return ModeSwarm, nil
	}

	return ModeStandalone, nil
}

// Mode returns the detected Docker mode.
func (c *Client) Mode() Mode {
	return c.mode
}

// RawClient returns the underlying Docker client for advanced operations.
// Use with caution - prefer using the wrapped methods when possible.
func (c *Client) RawClient() *client.Client {
	return c.docker
}

// Close closes the underlying Docker client.
func (c *Client) Close() error {
	return c.docker.Close()
}

// ListServices returns all Swarm services with their labels.
// Only valid in Swarm mode.
func (c *Client) ListServices(ctx context.Context) ([]Service, error) {
	if c.mode != ModeSwarm {
		return nil, fmt.Errorf("ListServices only available in swarm mode")
	}

	services, err := c.docker.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}

	result := make([]Service, 0, len(services))
	for _, svc := range services {
		result = append(result, Service{
			ID:     svc.ID,
			Name:   svc.Spec.Name,
			Labels: svc.Spec.Labels,
		})
	}

	c.logger.Debug("listed swarm services",
		slog.Int("count", len(result)),
	)

	return result, nil
}

// ListContainers returns all running containers with their labels.
// Only valid in standalone mode.
func (c *Client) ListContainers(ctx context.Context) ([]Container, error) {
	if c.mode != ModeStandalone {
		return nil, fmt.Errorf("ListContainers only available in standalone mode")
	}

	containers, err := c.docker.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("status", "running"),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("listing containers: %w", err)
	}

	result := make([]Container, 0, len(containers))
	for _, ctr := range containers {
		name := ""
		if len(ctr.Names) > 0 {
			// Container names start with /
			name = ctr.Names[0]
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
		}

		result = append(result, Container{
			ID:     ctr.ID,
			Name:   name,
			Labels: ctr.Labels,
		})
	}

	c.logger.Debug("listed containers",
		slog.Int("count", len(result)),
	)

	return result, nil
}

// GetServiceLabels returns the labels for a specific Swarm service by ID.
func (c *Client) GetServiceLabels(ctx context.Context, serviceID string) (map[string]string, error) {
	if c.mode != ModeSwarm {
		return nil, fmt.Errorf("GetServiceLabels only available in swarm mode")
	}

	svc, _, err := c.docker.ServiceInspectWithRaw(ctx, serviceID, types.ServiceInspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("inspecting service %s: %w", serviceID, err)
	}

	return svc.Spec.Labels, nil
}

// GetContainerLabels returns the labels for a specific container by ID.
func (c *Client) GetContainerLabels(ctx context.Context, containerID string) (map[string]string, error) {
	if c.mode != ModeStandalone {
		return nil, fmt.Errorf("GetContainerLabels only available in standalone mode")
	}

	ctr, err := c.docker.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("inspecting container %s: %w", containerID, err)
	}

	return ctr.Config.Labels, nil
}

// Workload represents either a Swarm service or a standalone container.
// Used to provide a unified interface for both modes.
type Workload struct {
	ID     string
	Name   string
	Labels map[string]string
	Type   string // "service" or "container"
}

// ListWorkloads returns all workloads (services in Swarm mode, containers in standalone).
// Provides a unified interface regardless of Docker mode.
func (c *Client) ListWorkloads(ctx context.Context) ([]Workload, error) {
	if c.mode == ModeSwarm {
		services, err := c.ListServices(ctx)
		if err != nil {
			return nil, err
		}

		workloads := make([]Workload, 0, len(services))
		for _, svc := range services {
			workloads = append(workloads, Workload{
				ID:     svc.ID,
				Name:   svc.Name,
				Labels: svc.Labels,
				Type:   "service",
			})
		}
		return workloads, nil
	}

	containers, err := c.ListContainers(ctx)
	if err != nil {
		return nil, err
	}

	workloads := make([]Workload, 0, len(containers))
	for _, ctr := range containers {
		workloads = append(workloads, Workload{
			ID:     ctr.ID,
			Name:   ctr.Name,
			Labels: ctr.Labels,
			Type:   "container",
		})
	}
	return workloads, nil
}

// Ping verifies connectivity to the Docker daemon.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.docker.Ping(ctx)
	if err != nil {
		return fmt.Errorf("pinging docker: %w", err)
	}
	return nil
}
