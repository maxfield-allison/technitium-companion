// Package health provides HTTP health check endpoints.
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Checker is a function that checks if a dependency is healthy.
type Checker func(ctx context.Context) error

// Status represents the health status of a component.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// ComponentHealth represents the health of a single component.
type ComponentHealth struct {
	Status  Status  `json:"status"`
	Message string  `json:"message,omitempty"`
	Latency string  `json:"latency,omitempty"`
}

// HealthResponse is the response from health endpoints.
type HealthResponse struct {
	Status     Status                     `json:"status"`
	Version    string                     `json:"version,omitempty"`
	Uptime     string                     `json:"uptime,omitempty"`
	Components map[string]ComponentHealth `json:"components,omitempty"`
}

// Server provides HTTP health check endpoints.
type Server struct {
	port      int
	version   string
	startTime time.Time
	logger    *slog.Logger
	server    *http.Server

	mu       sync.RWMutex
	checkers map[string]Checker
	ready    bool
}

// Option is a functional option for configuring the Server.
type Option func(*Server)

// WithLogger sets a custom logger.
func WithLogger(logger *slog.Logger) Option {
	return func(s *Server) {
		s.logger = logger
	}
}

// WithVersion sets the application version for health responses.
func WithVersion(version string) Option {
	return func(s *Server) {
		s.version = version
	}
}

// New creates a new health Server.
func New(port int, opts ...Option) *Server {
	s := &Server{
		port:      port,
		startTime: time.Now(),
		logger:    slog.Default(),
		checkers:  make(map[string]Checker),
		ready:     false,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// RegisterChecker adds a health checker for a named component.
func (s *Server) RegisterChecker(name string, checker Checker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkers[name] = checker
}

// SetReady marks the server as ready to receive traffic.
func (s *Server) SetReady(ready bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ready = ready
	s.logger.Info("readiness state changed",
		slog.Bool("ready", ready),
	)
}

// Start starts the health server in a goroutine.
// It returns a channel that will receive an error if the server fails.
func (s *Server) Start() <-chan error {
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/healthz", s.handleHealth) // Kubernetes alias
	mux.HandleFunc("/ready", s.handleReady)
	mux.HandleFunc("/readyz", s.handleReady) // Kubernetes alias
	mux.Handle("/metrics", promhttp.Handler()) // Prometheus metrics

	s.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}

	go func() {
		s.logger.Info("health server starting",
			slog.Int("port", s.port),
		)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("health server error: %w", err)
		}
		close(errCh)
	}()

	return errCh
}

// Shutdown gracefully shuts down the health server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	s.logger.Info("health server shutting down")
	return s.server.Shutdown(ctx)
}

// handleHealth responds to liveness probe requests.
// Returns 200 if the application is alive (can process requests).
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	checkers := make(map[string]Checker, len(s.checkers))
	for k, v := range s.checkers {
		checkers[k] = v
	}
	s.mu.RUnlock()

	resp := HealthResponse{
		Status:     StatusHealthy,
		Version:    s.version,
		Uptime:     time.Since(s.startTime).Round(time.Second).String(),
		Components: make(map[string]ComponentHealth),
	}

	// Check all registered components
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	for name, checker := range checkers {
		start := time.Now()
		err := checker(ctx)
		latency := time.Since(start)

		if err != nil {
			resp.Status = StatusDegraded
			resp.Components[name] = ComponentHealth{
				Status:  StatusUnhealthy,
				Message: err.Error(),
				Latency: latency.String(),
			}
		} else {
			resp.Components[name] = ComponentHealth{
				Status:  StatusHealthy,
				Latency: latency.String(),
			}
		}
	}

	statusCode := http.StatusOK
	if resp.Status == StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	s.writeJSON(w, statusCode, resp)
}

// handleReady responds to readiness probe requests.
// Returns 200 if the application is ready to receive traffic.
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	ready := s.ready
	checkers := make(map[string]Checker, len(s.checkers))
	for k, v := range s.checkers {
		checkers[k] = v
	}
	s.mu.RUnlock()

	if !ready {
		resp := HealthResponse{
			Status:  StatusUnhealthy,
			Version: s.version,
		}
		s.writeJSON(w, http.StatusServiceUnavailable, resp)
		return
	}

	resp := HealthResponse{
		Status:     StatusHealthy,
		Version:    s.version,
		Components: make(map[string]ComponentHealth),
	}

	// Check all registered components
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	allHealthy := true
	for name, checker := range checkers {
		start := time.Now()
		err := checker(ctx)
		latency := time.Since(start)

		if err != nil {
			allHealthy = false
			resp.Components[name] = ComponentHealth{
				Status:  StatusUnhealthy,
				Message: err.Error(),
				Latency: latency.String(),
			}
		} else {
			resp.Components[name] = ComponentHealth{
				Status:  StatusHealthy,
				Latency: latency.String(),
			}
		}
	}

	if !allHealthy {
		resp.Status = StatusUnhealthy
		s.writeJSON(w, http.StatusServiceUnavailable, resp)
		return
	}

	s.writeJSON(w, http.StatusOK, resp)
}

// writeJSON writes a JSON response.
func (s *Server) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("failed to write health response",
			slog.String("error", err.Error()),
		)
	}
}
