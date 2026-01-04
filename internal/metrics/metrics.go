// Package metrics provides Prometheus metrics for technitium-companion.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "technitium_companion"
)

var (
	// DNSRecordsCreatedTotal counts the total number of DNS records created.
	DNSRecordsCreatedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "dns_records_created_total",
			Help:      "Total number of DNS A records created",
		},
		[]string{"zone"},
	)

	// DNSRecordsDeletedTotal counts the total number of DNS records deleted.
	DNSRecordsDeletedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "dns_records_deleted_total",
			Help:      "Total number of DNS A records deleted",
		},
		[]string{"zone"},
	)

	// DNSRecordsExistedTotal counts records that already existed (no action needed).
	DNSRecordsExistedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "dns_records_existed_total",
			Help:      "Total number of DNS A records that already existed",
		},
		[]string{"zone"},
	)

	// APIRequestsTotal counts API requests by endpoint and status.
	APIRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "api_requests_total",
			Help:      "Total number of Technitium API requests",
		},
		[]string{"endpoint", "status"},
	)

	// APIRequestDuration tracks API request latency.
	APIRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "api_request_duration_seconds",
			Help:      "Duration of Technitium API requests in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)

	// DockerEventsTotal counts Docker events by type.
	DockerEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "docker_events_total",
			Help:      "Total number of Docker events processed",
		},
		[]string{"type", "action"},
	)

	// ReconciliationsTotal counts reconciliation runs by result.
	ReconciliationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "reconciliations_total",
			Help:      "Total number of reconciliation runs",
		},
		[]string{"status"},
	)

	// ReconciliationDuration tracks reconciliation duration.
	ReconciliationDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "reconciliation_duration_seconds",
			Help:      "Duration of reconciliation runs in seconds",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
		},
	)

	// WorkloadsScanned tracks the number of workloads scanned in the last reconciliation.
	WorkloadsScanned = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "workloads_scanned",
			Help:      "Number of Docker workloads scanned in the last reconciliation",
		},
	)

	// HostnamesFound tracks the number of hostnames found in the last reconciliation.
	HostnamesFound = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "hostnames_found",
			Help:      "Number of Traefik hostnames found in the last reconciliation",
		},
	)

	// LastReconciliationTimestamp tracks when the last successful reconciliation occurred.
	LastReconciliationTimestamp = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "last_reconciliation_timestamp_seconds",
			Help:      "Unix timestamp of the last successful reconciliation",
		},
	)

	// BuildInfo exposes build information as a metric.
	BuildInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "build_info",
			Help:      "Build information for technitium-companion",
		},
		[]string{"version", "go_version"},
	)

	// Up indicates if the service is up and running.
	Up = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Whether technitium-companion is up and running (1 = up, 0 = down)",
		},
	)
)

// SetBuildInfo sets the build information metric.
func SetBuildInfo(version, goVersion string) {
	BuildInfo.WithLabelValues(version, goVersion).Set(1)
}

// SetUp marks the service as up.
func SetUp() {
	Up.Set(1)
}

// RecordAPIRequest records metrics for an API request.
func RecordAPIRequest(endpoint, status string, durationSeconds float64) {
	APIRequestsTotal.WithLabelValues(endpoint, status).Inc()
	APIRequestDuration.WithLabelValues(endpoint).Observe(durationSeconds)
}

// RecordDNSRecordCreated increments the created counter for a zone.
func RecordDNSRecordCreated(zone string) {
	DNSRecordsCreatedTotal.WithLabelValues(zone).Inc()
}

// RecordDNSRecordDeleted increments the deleted counter for a zone.
func RecordDNSRecordDeleted(zone string) {
	DNSRecordsDeletedTotal.WithLabelValues(zone).Inc()
}

// RecordDNSRecordExisted increments the existed counter for a zone.
func RecordDNSRecordExisted(zone string) {
	DNSRecordsExistedTotal.WithLabelValues(zone).Inc()
}

// RecordDockerEvent increments the Docker events counter.
func RecordDockerEvent(eventType, action string) {
	DockerEventsTotal.WithLabelValues(eventType, action).Inc()
}

// RecordReconciliation records metrics for a reconciliation run.
func RecordReconciliation(status string, durationSeconds float64, workloads, hostnames int) {
	ReconciliationsTotal.WithLabelValues(status).Inc()
	ReconciliationDuration.Observe(durationSeconds)
	WorkloadsScanned.Set(float64(workloads))
	HostnamesFound.Set(float64(hostnames))
	if status == "success" {
		LastReconciliationTimestamp.SetToCurrentTime()
	}
}
