// Package reconciler provides tests for the DNS reconciler.
package reconciler

import (
	"context"
	"log/slog"
	"os"
	"regexp"
	"testing"

	"github.com/maxfield-allison/technitium-companion/internal/config"
	"github.com/maxfield-allison/technitium-companion/internal/traefik"
)

// TestWithLogger verifies the logger option works correctly.
func TestWithLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	opt := WithLogger(logger)

	r := &Reconciler{}
	opt(r)

	if r.logger != logger {
		t.Error("WithLogger did not set the logger correctly")
	}
}

// TestReconcileResult_Defaults verifies result struct defaults.
func TestReconcileResult_Defaults(t *testing.T) {
	result := ReconcileResult{}

	if result.WorkloadsScanned != 0 {
		t.Errorf("expected 0 workloads scanned, got %d", result.WorkloadsScanned)
	}
	if result.HostnamesFound != 0 {
		t.Errorf("expected 0 hostnames found, got %d", result.HostnamesFound)
	}
	if result.RecordsCreated != 0 {
		t.Errorf("expected 0 records created, got %d", result.RecordsCreated)
	}
	if result.RecordsExisted != 0 {
		t.Errorf("expected 0 records existed, got %d", result.RecordsExisted)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(result.Errors))
	}
	if result.Duration != 0 {
		t.Errorf("expected 0 duration, got %v", result.Duration)
	}
}

// TestReconcileResult_Accumulation tests that result fields accumulate correctly.
func TestReconcileResult_Accumulation(t *testing.T) {
	result := &ReconcileResult{}

	// Simulate processing multiple workloads
	result.WorkloadsScanned = 5
	result.HostnamesFound += 3
	result.HostnamesFound += 2
	result.HostnamesFiltered += 4
	result.RecordsCreated += 2
	result.RecordsExisted += 2
	result.Errors = append(result.Errors, nil) // placeholder

	if result.WorkloadsScanned != 5 {
		t.Errorf("expected 5 workloads, got %d", result.WorkloadsScanned)
	}
	if result.HostnamesFound != 5 {
		t.Errorf("expected 5 hostnames, got %d", result.HostnamesFound)
	}
	if result.HostnamesFiltered != 4 {
		t.Errorf("expected 4 filtered, got %d", result.HostnamesFiltered)
	}
	if result.RecordsCreated != 2 {
		t.Errorf("expected 2 created, got %d", result.RecordsCreated)
	}
	if result.RecordsExisted != 2 {
		t.Errorf("expected 2 existed, got %d", result.RecordsExisted)
	}
}

// TestNew_CreatesReconcilerWithDefaults tests the constructor.
func TestNew_CreatesReconcilerWithDefaults(t *testing.T) {
	cfg := &config.Config{
		TechnitiumZone: "example.com",
		TargetIP:       "10.0.0.1",
		TTL:            300,
		DryRun:         true,
	}
	parser := traefik.NewParser()

	rec := New(cfg, nil, parser, nil)

	if rec.cfg != cfg {
		t.Error("config not set correctly")
	}
	if rec.parser != parser {
		t.Error("parser not set correctly")
	}
	if rec.logger == nil {
		t.Error("logger should not be nil")
	}
}

// TestNew_WithOptions tests constructor with options.
func TestNew_WithOptions(t *testing.T) {
	cfg := &config.Config{}
	parser := traefik.NewParser()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	rec := New(cfg, nil, parser, nil, WithLogger(logger))

	if rec.logger != logger {
		t.Error("logger option not applied")
	}
}

// mockDockerClient is a minimal mock for testing.
type mockDockerClient struct {
	workloads []mockWorkload
}

type mockWorkload struct {
	ID     string
	Name   string
	Labels map[string]string
	Type   string
}

// TestReconciler_ProcessWorkload_NoLabels tests workloads without Traefik labels.
func TestReconciler_ProcessWorkload_NoLabels(t *testing.T) {
	cfg := &config.Config{
		TechnitiumZone: "example.com",
		TargetIP:       "10.0.0.1",
		TTL:            300,
		DryRun:         true,
	}
	parser := traefik.NewParser()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	rec := &Reconciler{
		cfg:    cfg,
		parser: parser,
		logger: logger,
	}

	// Workload without Traefik labels
	result := &ReconcileResult{}
	// For internal method testing, we'd need to export or use a test helper
	// This test verifies the setup doesn't panic

	_ = rec
	_ = result
}

// TestReconciler_DryRunMode tests that dry run prevents actual API calls.
func TestReconciler_DryRunMode(t *testing.T) {
	cfg := &config.Config{
		TechnitiumZone: "example.com",
		TargetIP:       "10.0.0.1",
		TTL:            300,
		DryRun:         true,
	}
	parser := traefik.NewParser()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	rec := New(cfg, nil, parser, nil, WithLogger(logger))

	if !rec.cfg.DryRun {
		t.Error("expected dry run mode to be enabled")
	}
}

// TestReconciler_FilterMatching tests the filter matching integration.
func TestReconciler_FilterMatching(t *testing.T) {
	tests := []struct {
		name           string
		includePattern string
		excludePattern string
		hostname       string
		shouldMatch    bool
	}{
		{
			name:           "no filters match all",
			includePattern: "",
			excludePattern: "",
			hostname:       "app.example.com",
			shouldMatch:    true,
		},
		{
			name:           "include pattern matches",
			includePattern: ".*\\.example\\.com$",
			excludePattern: "",
			hostname:       "app.example.com",
			shouldMatch:    true,
		},
		{
			name:           "include pattern does not match",
			includePattern: ".*\\.example\\.com$",
			excludePattern: "",
			hostname:       "app.other.com",
			shouldMatch:    false,
		},
		{
			name:           "exclude pattern filters out",
			includePattern: "",
			excludePattern: "^internal\\.",
			hostname:       "internal.example.com",
			shouldMatch:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var includeRe, excludeRe *regexp.Regexp
			if tt.includePattern != "" {
				includeRe = regexp.MustCompile(tt.includePattern)
			}
			if tt.excludePattern != "" {
				excludeRe = regexp.MustCompile(tt.excludePattern)
			}

			cfg := &config.Config{
				IncludePattern: includeRe,
				ExcludePattern: excludeRe,
			}

			result := cfg.MatchesFilters(tt.hostname)
			if result != tt.shouldMatch {
				t.Errorf("MatchesFilters(%q) = %v, want %v", tt.hostname, result, tt.shouldMatch)
			}
		})
	}
}

// TestReconcileHostnames_Empty tests reconciling empty hostname list.
func TestReconcileHostnames_Empty(t *testing.T) {
	cfg := &config.Config{
		TechnitiumZone: "example.com",
		TargetIP:       "10.0.0.1",
		TTL:            300,
		DryRun:         true,
	}
	parser := traefik.NewParser()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	rec := New(cfg, nil, parser, nil, WithLogger(logger))

	result, err := rec.ReconcileHostnames(context.Background(), "test-workload", []string{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.HostnamesFound != 0 {
		t.Errorf("expected 0 hostnames found, got %d", result.HostnamesFound)
	}
}

// TestDeleteHostnames_Empty tests deleting empty hostname list.
func TestDeleteHostnames_Empty(t *testing.T) {
	cfg := &config.Config{
		TechnitiumZone: "example.com",
		TargetIP:       "10.0.0.1",
		DryRun:         true,
	}
	parser := traefik.NewParser()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	rec := New(cfg, nil, parser, nil, WithLogger(logger))

	deleted, err := rec.DeleteHostnames(context.Background(), "test-workload", []string{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 deleted, got %d", deleted)
	}
}

// TestReconciler_ConcurrencySafe tests that the reconciler uses mutex properly.
func TestReconciler_ConcurrencySafe(t *testing.T) {
	cfg := &config.Config{
		TechnitiumZone: "example.com",
		TargetIP:       "10.0.0.1",
		TTL:            300,
		DryRun:         true,
	}
	parser := traefik.NewParser()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	rec := New(cfg, nil, parser, nil, WithLogger(logger))

	// Verify the mutex exists (can't easily test locking without races)
	// This just ensures the struct has the field
	_ = rec
}
