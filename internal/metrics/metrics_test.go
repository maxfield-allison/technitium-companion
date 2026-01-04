package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestSetBuildInfo(t *testing.T) {
	// Reset the metric for testing
	BuildInfo.Reset()

	SetBuildInfo("1.0.0", "go1.24")

	// Check that the metric was set
	count := testutil.CollectAndCount(BuildInfo)
	if count != 1 {
		t.Errorf("expected 1 metric, got %d", count)
	}
}

func TestSetUp(t *testing.T) {
	// Reset
	Up.Set(0)

	SetUp()

	// Verify
	value := testutil.ToFloat64(Up)
	if value != 1 {
		t.Errorf("expected Up=1, got %f", value)
	}
}

func TestRecordAPIRequest(t *testing.T) {
	// Reset
	APIRequestsTotal.Reset()
	APIRequestDuration.Reset()

	RecordAPIRequest("/api/zones/records/add", "success", 0.5)
	RecordAPIRequest("/api/zones/records/add", "error", 0.1)
	RecordAPIRequest("/api/zones/records/get", "success", 0.2)

	// Check total requests
	expected := `
		# HELP technitium_companion_api_requests_total Total number of Technitium API requests
		# TYPE technitium_companion_api_requests_total counter
		technitium_companion_api_requests_total{endpoint="/api/zones/records/add",status="error"} 1
		technitium_companion_api_requests_total{endpoint="/api/zones/records/add",status="success"} 1
		technitium_companion_api_requests_total{endpoint="/api/zones/records/get",status="success"} 1
	`
	if err := testutil.CollectAndCompare(APIRequestsTotal, strings.NewReader(expected)); err != nil {
		t.Errorf("unexpected metric: %v", err)
	}

	// Check that duration was recorded
	count := testutil.CollectAndCount(APIRequestDuration)
	if count != 2 { // 2 unique endpoints
		t.Errorf("expected 2 histogram metrics, got %d", count)
	}
}

func TestRecordDNSRecordCreated(t *testing.T) {
	DNSRecordsCreatedTotal.Reset()

	RecordDNSRecordCreated("local.example.com")
	RecordDNSRecordCreated("local.example.com")
	RecordDNSRecordCreated("other.example.com")

	// Verify counts
	localCount := testutil.ToFloat64(DNSRecordsCreatedTotal.WithLabelValues("local.example.com"))
	if localCount != 2 {
		t.Errorf("expected 2 records created for local.example.com, got %f", localCount)
	}

	otherCount := testutil.ToFloat64(DNSRecordsCreatedTotal.WithLabelValues("other.example.com"))
	if otherCount != 1 {
		t.Errorf("expected 1 record created for other.example.com, got %f", otherCount)
	}
}

func TestRecordDNSRecordDeleted(t *testing.T) {
	DNSRecordsDeletedTotal.Reset()

	RecordDNSRecordDeleted("local.example.com")

	count := testutil.ToFloat64(DNSRecordsDeletedTotal.WithLabelValues("local.example.com"))
	if count != 1 {
		t.Errorf("expected 1 record deleted, got %f", count)
	}
}

func TestRecordDNSRecordExisted(t *testing.T) {
	DNSRecordsExistedTotal.Reset()

	RecordDNSRecordExisted("local.example.com")
	RecordDNSRecordExisted("local.example.com")

	count := testutil.ToFloat64(DNSRecordsExistedTotal.WithLabelValues("local.example.com"))
	if count != 2 {
		t.Errorf("expected 2 records existed, got %f", count)
	}
}

func TestRecordDockerEvent(t *testing.T) {
	DockerEventsTotal.Reset()

	RecordDockerEvent("service", "create")
	RecordDockerEvent("service", "update")
	RecordDockerEvent("container", "start")

	// Verify
	serviceCreate := testutil.ToFloat64(DockerEventsTotal.WithLabelValues("service", "create"))
	if serviceCreate != 1 {
		t.Errorf("expected 1 service create event, got %f", serviceCreate)
	}

	containerStart := testutil.ToFloat64(DockerEventsTotal.WithLabelValues("container", "start"))
	if containerStart != 1 {
		t.Errorf("expected 1 container start event, got %f", containerStart)
	}
}

func TestRecordReconciliation(t *testing.T) {
	// Reset all related metrics
	ReconciliationsTotal.Reset()
	WorkloadsScanned.Set(0)
	HostnamesFound.Set(0)
	LastReconciliationTimestamp.Set(0)

	RecordReconciliation("success", 1.5, 100, 50)

	// Check counter
	successCount := testutil.ToFloat64(ReconciliationsTotal.WithLabelValues("success"))
	if successCount != 1 {
		t.Errorf("expected 1 successful reconciliation, got %f", successCount)
	}

	// Check gauges
	workloads := testutil.ToFloat64(WorkloadsScanned)
	if workloads != 100 {
		t.Errorf("expected 100 workloads scanned, got %f", workloads)
	}

	hostnames := testutil.ToFloat64(HostnamesFound)
	if hostnames != 50 {
		t.Errorf("expected 50 hostnames found, got %f", hostnames)
	}

	// Check timestamp was set (should be > 0 for success)
	timestamp := testutil.ToFloat64(LastReconciliationTimestamp)
	if timestamp == 0 {
		t.Error("expected last reconciliation timestamp to be set")
	}

	// Test error case - timestamp should not be updated
	currentTimestamp := timestamp
	RecordReconciliation("error", 0.5, 10, 5)

	errorCount := testutil.ToFloat64(ReconciliationsTotal.WithLabelValues("error"))
	if errorCount != 1 {
		t.Errorf("expected 1 error reconciliation, got %f", errorCount)
	}

	// Gauges should be updated even on error
	workloads = testutil.ToFloat64(WorkloadsScanned)
	if workloads != 10 {
		t.Errorf("expected 10 workloads scanned after error, got %f", workloads)
	}

	// But timestamp should remain the same (from previous success)
	// Note: This is a design choice - we only update timestamp on success
	newTimestamp := testutil.ToFloat64(LastReconciliationTimestamp)
	if newTimestamp != currentTimestamp {
		t.Errorf("expected timestamp to remain %f, got %f", currentTimestamp, newTimestamp)
	}
}

func TestMetricsAreRegistered(t *testing.T) {
	// Verify that all metrics are registered with the default registry
	// by attempting to gather them
	metrics, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	// Check that we have at least some of our metrics
	expectedMetrics := map[string]bool{
		"technitium_companion_up":                      false,
		"technitium_companion_workloads_scanned":       false,
		"technitium_companion_hostnames_found":         false,
		"technitium_companion_reconciliation_duration_seconds": false,
	}

	for _, mf := range metrics {
		if _, ok := expectedMetrics[mf.GetName()]; ok {
			expectedMetrics[mf.GetName()] = true
		}
	}

	for name, found := range expectedMetrics {
		if !found {
			t.Errorf("expected metric %s to be registered", name)
		}
	}
}
