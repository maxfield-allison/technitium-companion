package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_RequiredFields(t *testing.T) {
	// Clear all env vars
	clearEnv()

	_, err := Load()
	if err == nil {
		t.Error("expected error when required fields are missing")
	}
}

func TestLoad_AllRequired(t *testing.T) {
	clearEnv()
	os.Setenv("TECHNITIUM_URL", "http://dns.example.com:5380")
	os.Setenv("TECHNITIUM_TOKEN", "secret-token")
	os.Setenv("TECHNITIUM_ZONE", "example.com")
	os.Setenv("TARGET_IP", "10.0.0.1")
	defer clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.TechnitiumURL != "http://dns.example.com:5380" {
		t.Errorf("expected URL http://dns.example.com:5380, got %s", cfg.TechnitiumURL)
	}
	if cfg.TechnitiumToken != "secret-token" {
		t.Errorf("expected token secret-token, got %s", cfg.TechnitiumToken)
	}
	if cfg.TechnitiumZone != "example.com" {
		t.Errorf("expected zone example.com, got %s", cfg.TechnitiumZone)
	}
	if cfg.TargetIP != "10.0.0.1" {
		t.Errorf("expected IP 10.0.0.1, got %s", cfg.TargetIP)
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	defer clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.TTL != DefaultTTL {
		t.Errorf("expected default TTL %d, got %d", DefaultTTL, cfg.TTL)
	}
	if cfg.DockerHost != DefaultDockerHost {
		t.Errorf("expected default DockerHost %s, got %s", DefaultDockerHost, cfg.DockerHost)
	}
	if cfg.DockerMode != DefaultDockerMode {
		t.Errorf("expected default DockerMode %s, got %s", DefaultDockerMode, cfg.DockerMode)
	}
	if cfg.ReconcileOnStartup != DefaultReconcileOnStartup {
		t.Errorf("expected default ReconcileOnStartup %v, got %v", DefaultReconcileOnStartup, cfg.ReconcileOnStartup)
	}
	if cfg.DryRun != DefaultDryRun {
		t.Errorf("expected default DryRun %v, got %v", DefaultDryRun, cfg.DryRun)
	}
	if cfg.HealthPort != DefaultHealthPort {
		t.Errorf("expected default HealthPort %d, got %d", DefaultHealthPort, cfg.HealthPort)
	}
	if cfg.LogLevel != DefaultLogLevel {
		t.Errorf("expected default LogLevel %s, got %s", DefaultLogLevel, cfg.LogLevel)
	}
}

func TestLoad_CustomTTL(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("TTL", "600")
	defer clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.TTL != 600 {
		t.Errorf("expected TTL 600, got %d", cfg.TTL)
	}
}

func TestLoad_InvalidTTL(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("TTL", "not-a-number")
	defer clearEnv()

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid TTL")
	}
}

func TestLoad_ZeroTTL(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("TTL", "0")
	defer clearEnv()

	_, err := Load()
	if err == nil {
		t.Error("expected error for zero TTL")
	}
}

func TestLoad_TokenFromFile(t *testing.T) {
	clearEnv()

	// Create a temp file with the token
	tmpDir := t.TempDir()
	tokenFile := filepath.Join(tmpDir, "token")
	if err := os.WriteFile(tokenFile, []byte("file-secret-token\n"), 0600); err != nil {
		t.Fatalf("failed to write token file: %v", err)
	}

	os.Setenv("TECHNITIUM_URL", "http://dns.example.com:5380")
	os.Setenv("TECHNITIUM_TOKEN_FILE", tokenFile)
	os.Setenv("TECHNITIUM_ZONE", "example.com")
	os.Setenv("TARGET_IP", "10.0.0.1")
	defer clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be trimmed of whitespace/newline
	if cfg.TechnitiumToken != "file-secret-token" {
		t.Errorf("expected token 'file-secret-token', got '%s'", cfg.TechnitiumToken)
	}
}

func TestLoad_IncludePattern(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("INCLUDE_PATTERN", `^.*\.local\.example\.com$`)
	defer clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.IncludePattern.MatchString("app.local.example.com") {
		t.Error("expected pattern to match app.local.example.com")
	}
	if cfg.IncludePattern.MatchString("app.example.com") {
		t.Error("expected pattern not to match app.example.com")
	}
}

func TestLoad_ExcludePattern(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("EXCLUDE_PATTERN", `^test\.`)
	defer clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.ExcludePattern.MatchString("test.example.com") {
		t.Error("expected pattern to match test.example.com")
	}
}

func TestLoad_InvalidIncludePattern(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("INCLUDE_PATTERN", `[invalid`)
	defer clearEnv()

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestLoad_InvalidDockerMode(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("DOCKER_MODE", "invalid")
	defer clearEnv()

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid docker mode")
	}
}

func TestLoad_ValidDockerModes(t *testing.T) {
	modes := []string{"auto", "swarm", "standalone", "AUTO", "SWARM", "STANDALONE"}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			clearEnv()
			setRequiredEnv()
			os.Setenv("DOCKER_MODE", mode)
			defer clearEnv()

			_, err := Load()
			if err != nil {
				t.Fatalf("unexpected error for mode %s: %v", mode, err)
			}
			// Loader accepts any case and stores as lowercase
		})
	}
}

func TestLoad_BooleanParsing(t *testing.T) {
	trueValues := []string{"true", "TRUE", "1", "yes", "YES", "on", "ON"}
	falseValues := []string{"false", "FALSE", "0", "no", "NO", "off", "OFF"}

	for _, val := range trueValues {
		t.Run("true_"+val, func(t *testing.T) {
			clearEnv()
			setRequiredEnv()
			os.Setenv("DRY_RUN", val)
			defer clearEnv()

			cfg, err := Load()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !cfg.DryRun {
				t.Errorf("expected DryRun to be true for value %s", val)
			}
		})
	}

	for _, val := range falseValues {
		t.Run("false_"+val, func(t *testing.T) {
			clearEnv()
			setRequiredEnv()
			os.Setenv("RECONCILE_ON_STARTUP", val)
			defer clearEnv()

			cfg, err := Load()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.ReconcileOnStartup {
				t.Errorf("expected ReconcileOnStartup to be false for value %s", val)
			}
		})
	}
}

func TestLoad_InvalidLogLevel(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("LOG_LEVEL", "verbose")
	defer clearEnv()

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid log level")
	}
}

func TestLoad_ValidLogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "DEBUG", "INFO"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			clearEnv()
			setRequiredEnv()
			os.Setenv("LOG_LEVEL", level)
			defer clearEnv()

			_, err := Load()
			if err != nil {
				t.Fatalf("unexpected error for level %s: %v", level, err)
			}
		})
	}
}

func TestLoad_InvalidHealthPort(t *testing.T) {
	tests := []struct {
		name string
		port string
	}{
		{"not a number", "abc"},
		{"too low", "0"},
		{"too high", "70000"},
		{"negative", "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv()
			setRequiredEnv()
			os.Setenv("HEALTH_PORT", tt.port)
			defer clearEnv()

			_, err := Load()
			if err == nil {
				t.Errorf("expected error for health port %s", tt.port)
			}
		})
	}
}

func TestMatchesFilters(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("INCLUDE_PATTERN", `\.local\.example\.com$`)
	os.Setenv("EXCLUDE_PATTERN", `^test\.`)
	defer clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		hostname string
		expected bool
	}{
		{"app.local.example.com", true},
		{"api.local.example.com", true},
		{"test.local.example.com", false},  // excluded by EXCLUDE_PATTERN
		{"app.example.com", false},         // doesn't match INCLUDE_PATTERN
		{"test.example.com", false},        // doesn't match INCLUDE_PATTERN
	}

	for _, tt := range tests {
		t.Run(tt.hostname, func(t *testing.T) {
			result := cfg.MatchesFilters(tt.hostname)
			if result != tt.expected {
				t.Errorf("MatchesFilters(%q) = %v, want %v", tt.hostname, result, tt.expected)
			}
		})
	}
}

func TestMatchesFilters_NoExclude(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("INCLUDE_PATTERN", `\.example\.com$`)
	defer clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		hostname string
		expected bool
	}{
		{"app.example.com", true},
		{"test.example.com", true},
		{"example.org", false},
	}

	for _, tt := range tests {
		t.Run(tt.hostname, func(t *testing.T) {
			result := cfg.MatchesFilters(tt.hostname)
			if result != tt.expected {
				t.Errorf("MatchesFilters(%q) = %v, want %v", tt.hostname, result, tt.expected)
			}
		})
	}
}

func TestLoad_InvalidTargetIP(t *testing.T) {
	clearEnv()
	os.Setenv("TECHNITIUM_URL", "http://dns.example.com:5380")
	os.Setenv("TECHNITIUM_TOKEN", "token")
	os.Setenv("TECHNITIUM_ZONE", "example.com")
	os.Setenv("TARGET_IP", "not-an-ip")
	defer clearEnv()

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid IP address")
	}
	if !strings.Contains(err.Error(), "not a valid IP address") {
		t.Errorf("expected 'not a valid IP address' error, got: %v", err)
	}
}

func TestLoad_ValidIPv6(t *testing.T) {
	clearEnv()
	os.Setenv("TECHNITIUM_URL", "http://dns.example.com:5380")
	os.Setenv("TECHNITIUM_TOKEN", "token")
	os.Setenv("TECHNITIUM_ZONE", "example.com")
	os.Setenv("TARGET_IP", "2001:db8::1")
	defer clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error for valid IPv6: %v", err)
	}
	if cfg.TargetIP != "2001:db8::1" {
		t.Errorf("expected IPv6 2001:db8::1, got %s", cfg.TargetIP)
	}
}

func TestValidate_TrimsTrailingSlash(t *testing.T) {
	clearEnv()
	os.Setenv("TECHNITIUM_URL", "http://dns.example.com:5380/")
	os.Setenv("TECHNITIUM_TOKEN", "token")
	os.Setenv("TECHNITIUM_ZONE", "example.com")
	os.Setenv("TARGET_IP", "10.0.0.1")
	defer clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	if cfg.TechnitiumURL != "http://dns.example.com:5380" {
		t.Errorf("expected trailing slash to be trimmed, got %s", cfg.TechnitiumURL)
	}
}

// Helper functions

func clearEnv() {
	envVars := []string{
		"TECHNITIUM_URL", "TECHNITIUM_URL_FILE",
		"TECHNITIUM_TOKEN", "TECHNITIUM_TOKEN_FILE",
		"TECHNITIUM_ZONE", "TECHNITIUM_ZONE_FILE",
		"TARGET_IP", "TARGET_IP_FILE",
		"TTL", "INCLUDE_PATTERN", "EXCLUDE_PATTERN",
		"DOCKER_HOST", "DOCKER_MODE",
		"RECONCILE_ON_STARTUP", "DRY_RUN",
		"HEALTH_PORT", "LOG_LEVEL",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
}

func setRequiredEnv() {
	os.Setenv("TECHNITIUM_URL", "http://dns.example.com:5380")
	os.Setenv("TECHNITIUM_TOKEN", "secret-token")
	os.Setenv("TECHNITIUM_ZONE", "example.com")
	os.Setenv("TARGET_IP", "10.0.0.1")
}
