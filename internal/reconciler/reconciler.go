// Package reconciler scans Docker workloads and ensures DNS records exist for Traefik-labeled services.
package reconciler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/maxfield-allison/technitium-companion/internal/config"
	"github.com/maxfield-allison/technitium-companion/internal/docker"
	"github.com/maxfield-allison/technitium-companion/internal/metrics"
	"github.com/maxfield-allison/technitium-companion/internal/technitium"
	"github.com/maxfield-allison/technitium-companion/internal/traefik"
)

// ReconcileResult contains the results of a reconciliation run.
type ReconcileResult struct {
	// WorkloadsScanned is the number of Docker workloads (services/containers) scanned.
	WorkloadsScanned int
	// HostnamesFound is the total number of hostnames extracted from Traefik labels.
	HostnamesFound int
	// HostnamesFiltered is the number of hostnames that matched include/exclude filters.
	HostnamesFiltered int
	// RecordsCreated is the number of new DNS A records created.
	RecordsCreated int
	// RecordsExisted is the number of DNS A records that already existed.
	RecordsExisted int
	// Errors contains any errors encountered during reconciliation.
	Errors []error
	// Duration is how long the reconciliation took.
	Duration time.Duration
}

// Reconciler scans Docker workloads and ensures DNS records exist.
type Reconciler struct {
	cfg        *config.Config
	docker     *docker.Client
	parser     *traefik.Parser
	technitium *technitium.Client
	logger     *slog.Logger

	mu sync.Mutex
}

// Option is a functional option for configuring the Reconciler.
type Option func(*Reconciler)

// WithLogger sets a custom logger.
func WithLogger(logger *slog.Logger) Option {
	return func(r *Reconciler) {
		r.logger = logger
	}
}

// New creates a new Reconciler.
func New(
	cfg *config.Config,
	dockerClient *docker.Client,
	parser *traefik.Parser,
	techClient *technitium.Client,
	opts ...Option,
) *Reconciler {
	r := &Reconciler{
		cfg:        cfg,
		docker:     dockerClient,
		parser:     parser,
		technitium: techClient,
		logger:     slog.Default(),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Reconcile scans all Docker workloads and ensures DNS records exist for Traefik-labeled services.
// It returns a result containing statistics about the reconciliation run.
func (r *Reconciler) Reconcile(ctx context.Context) (*ReconcileResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	start := time.Now()
	result := &ReconcileResult{}

	r.logger.Info("starting reconciliation",
		slog.String("mode", string(r.docker.Mode())),
		slog.Bool("dry_run", r.cfg.DryRun),
	)

	// List all workloads (services in Swarm, containers in standalone)
	workloads, err := r.docker.ListWorkloads(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing workloads: %w", err)
	}

	result.WorkloadsScanned = len(workloads)
	r.logger.Debug("scanned workloads",
		slog.Int("count", len(workloads)),
	)

	// Process each workload
	for _, workload := range workloads {
		if err := r.processWorkload(ctx, workload, result); err != nil {
			r.logger.Error("failed to process workload",
				slog.String("name", workload.Name),
				slog.String("type", workload.Type),
				slog.String("error", err.Error()),
			)
			result.Errors = append(result.Errors, fmt.Errorf("workload %s: %w", workload.Name, err))
		}
	}

	result.Duration = time.Since(start)

	// Record reconciliation metrics
	status := "success"
	if len(result.Errors) > 0 {
		status = "error"
	}
	metrics.RecordReconciliation(status, result.Duration.Seconds(), result.WorkloadsScanned, result.HostnamesFound)

	r.logger.Info("reconciliation complete",
		slog.Int("workloads_scanned", result.WorkloadsScanned),
		slog.Int("hostnames_found", result.HostnamesFound),
		slog.Int("hostnames_filtered", result.HostnamesFiltered),
		slog.Int("records_created", result.RecordsCreated),
		slog.Int("records_existed", result.RecordsExisted),
		slog.Int("errors", len(result.Errors)),
		slog.Duration("duration", result.Duration),
	)

	return result, nil
}

// processWorkload extracts hostnames from a workload's Traefik labels and ensures DNS records exist.
func (r *Reconciler) processWorkload(ctx context.Context, workload docker.Workload, result *ReconcileResult) error {
	// Extract hostnames from Traefik labels
	hosts := r.parser.ExtractHosts(workload.Labels)
	if len(hosts) == 0 {
		r.logger.Debug("no traefik hosts found",
			slog.String("workload", workload.Name),
		)
		return nil
	}

	result.HostnamesFound += len(hosts)

	r.logger.Debug("found traefik hosts",
		slog.String("workload", workload.Name),
		slog.Any("hosts", hosts),
	)

	// Process each hostname
	for _, host := range hosts {
		if err := r.ensureRecord(ctx, workload.Name, host, result); err != nil {
			return fmt.Errorf("ensuring record for %s: %w", host, err)
		}
	}

	return nil
}

// ensureRecord ensures a DNS A record exists for a hostname.
func (r *Reconciler) ensureRecord(ctx context.Context, workloadName, hostname string, result *ReconcileResult) error {
	// Apply include/exclude filters
	if !r.cfg.MatchesFilters(hostname) {
		r.logger.Debug("hostname filtered out",
			slog.String("hostname", hostname),
			slog.String("workload", workloadName),
		)
		return nil
	}

	result.HostnamesFiltered++

	// Dry run mode - log what would be created
	if r.cfg.DryRun {
		r.logger.Info("DRY RUN: would ensure A record",
			slog.String("hostname", hostname),
			slog.String("zone", r.cfg.TechnitiumZone),
			slog.String("ip", r.cfg.TargetIP),
			slog.Int("ttl", r.cfg.TTL),
			slog.String("workload", workloadName),
		)
		result.RecordsCreated++ // Count as would-be-created for reporting
		return nil
	}

	// Ensure the A record exists
	created, err := r.technitium.EnsureARecord(
		ctx,
		r.cfg.TechnitiumZone,
		hostname,
		r.cfg.TargetIP,
		r.cfg.TTL,
	)
	if err != nil {
		return fmt.Errorf("creating A record: %w", err)
	}

	if created {
		result.RecordsCreated++
		metrics.RecordDNSRecordCreated(r.cfg.TechnitiumZone)
		r.logger.Info("created A record",
			slog.String("hostname", hostname),
			slog.String("zone", r.cfg.TechnitiumZone),
			slog.String("ip", r.cfg.TargetIP),
			slog.String("workload", workloadName),
		)
	} else {
		result.RecordsExisted++
		metrics.RecordDNSRecordExisted(r.cfg.TechnitiumZone)
		r.logger.Debug("A record already exists",
			slog.String("hostname", hostname),
			slog.String("ip", r.cfg.TargetIP),
		)
	}

	return nil
}

// ReconcileHostnames ensures DNS records exist for a specific set of hostnames.
// This is useful for event-driven reconciliation when a new service is created.
func (r *Reconciler) ReconcileHostnames(ctx context.Context, workloadName string, hostnames []string) (*ReconcileResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	start := time.Now()
	result := &ReconcileResult{
		HostnamesFound: len(hostnames),
	}

	r.logger.Debug("reconciling specific hostnames",
		slog.String("workload", workloadName),
		slog.Any("hostnames", hostnames),
	)

	for _, hostname := range hostnames {
		if err := r.ensureRecord(ctx, workloadName, hostname, result); err != nil {
			r.logger.Error("failed to ensure record",
				slog.String("hostname", hostname),
				slog.String("error", err.Error()),
			)
			result.Errors = append(result.Errors, fmt.Errorf("hostname %s: %w", hostname, err))
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

// DeleteHostnames removes DNS records for a specific set of hostnames.
// This is useful when a service is removed (if orphan cleanup is enabled).
func (r *Reconciler) DeleteHostnames(ctx context.Context, workloadName string, hostnames []string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	deleted := 0

	r.logger.Debug("deleting hostnames",
		slog.String("workload", workloadName),
		slog.Any("hostnames", hostnames),
	)

	for _, hostname := range hostnames {
		// Apply include/exclude filters
		if !r.cfg.MatchesFilters(hostname) {
			continue
		}

		// Dry run mode
		if r.cfg.DryRun {
			r.logger.Info("DRY RUN: would delete A record",
				slog.String("hostname", hostname),
				slog.String("zone", r.cfg.TechnitiumZone),
				slog.String("ip", r.cfg.TargetIP),
				slog.String("workload", workloadName),
			)
			deleted++
			continue
		}

		// Check if record exists before deleting
		exists, err := r.technitium.HasARecord(
			ctx,
			r.cfg.TechnitiumZone,
			hostname,
			r.cfg.TargetIP,
		)
		if err != nil {
			r.logger.Error("failed to check record existence",
				slog.String("hostname", hostname),
				slog.String("error", err.Error()),
			)
			continue
		}

		if !exists {
			r.logger.Debug("A record does not exist, skipping delete",
				slog.String("hostname", hostname),
			)
			continue
		}

		// Delete the record
		if err := r.technitium.DeleteARecord(
			ctx,
			r.cfg.TechnitiumZone,
			hostname,
			r.cfg.TargetIP,
		); err != nil {
			r.logger.Error("failed to delete A record",
				slog.String("hostname", hostname),
				slog.String("error", err.Error()),
			)
			continue
		}

		deleted++
		metrics.RecordDNSRecordDeleted(r.cfg.TechnitiumZone)
		r.logger.Info("deleted A record",
			slog.String("hostname", hostname),
			slog.String("zone", r.cfg.TechnitiumZone),
			slog.String("ip", r.cfg.TargetIP),
			slog.String("workload", workloadName),
		)
	}

	return deleted, nil
}
