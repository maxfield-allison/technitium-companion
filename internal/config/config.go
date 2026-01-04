// Package config provides configuration loading from environment variables.
package config

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Config holds the application configuration.
type Config struct {
	// Technitium DNS settings
	TechnitiumURL   string
	TechnitiumToken string
	TechnitiumZone  string

	// Target IP for DNS records
	TargetIP string

	// DNS record settings
	TTL int

	// Filtering
	IncludePattern *regexp.Regexp
	ExcludePattern *regexp.Regexp

	// Docker settings
	DockerHost string
	DockerMode string // "auto", "swarm", or "standalone"

	// Behavior
	ReconcileOnStartup bool
	DryRun             bool

	// Health server
	HealthPort int

	// Logging
	LogLevel string
}

// Defaults
const (
	DefaultTTL                = 300
	DefaultIncludePattern     = ".*"
	DefaultDockerHost         = "unix:///var/run/docker.sock"
	DefaultDockerMode         = "auto"
	DefaultReconcileOnStartup = true
	DefaultDryRun             = false
	DefaultHealthPort         = 8080
	DefaultLogLevel           = "info"
)

// Load reads configuration from environment variables.
// Supports _FILE suffix for Docker secrets (reads the file contents).
func Load() (*Config, error) {
	cfg := &Config{}
	var errs []string

	// Required: Technitium URL
	cfg.TechnitiumURL = getEnvOrFile("TECHNITIUM_URL")
	if cfg.TechnitiumURL == "" {
		errs = append(errs, "TECHNITIUM_URL is required")
	}

	// Required: Technitium Token (supports _FILE for secrets)
	cfg.TechnitiumToken = getEnvOrFile("TECHNITIUM_TOKEN")
	if cfg.TechnitiumToken == "" {
		errs = append(errs, "TECHNITIUM_TOKEN or TECHNITIUM_TOKEN_FILE is required")
	}

	// Required: Zone
	cfg.TechnitiumZone = getEnvOrFile("TECHNITIUM_ZONE")
	if cfg.TechnitiumZone == "" {
		errs = append(errs, "TECHNITIUM_ZONE is required")
	}

	// Required: Target IP
	cfg.TargetIP = getEnvOrFile("TARGET_IP")
	if cfg.TargetIP == "" {
		errs = append(errs, "TARGET_IP is required")
	} else if net.ParseIP(cfg.TargetIP) == nil {
		errs = append(errs, fmt.Sprintf("TARGET_IP is not a valid IP address: %s", cfg.TargetIP))
	}

	// Optional: TTL
	ttlStr := os.Getenv("TTL")
	if ttlStr != "" {
		ttl, err := strconv.Atoi(ttlStr)
		if err != nil {
			errs = append(errs, fmt.Sprintf("TTL must be a valid integer: %v", err))
		} else if ttl < 1 {
			errs = append(errs, "TTL must be at least 1")
		} else {
			cfg.TTL = ttl
		}
	} else {
		cfg.TTL = DefaultTTL
	}

	// Optional: Include pattern
	includeStr := os.Getenv("INCLUDE_PATTERN")
	if includeStr == "" {
		includeStr = DefaultIncludePattern
	}
	includeRe, err := regexp.Compile(includeStr)
	if err != nil {
		errs = append(errs, fmt.Sprintf("INCLUDE_PATTERN is not a valid regex: %v", err))
	} else {
		cfg.IncludePattern = includeRe
	}

	// Optional: Exclude pattern
	excludeStr := os.Getenv("EXCLUDE_PATTERN")
	if excludeStr != "" {
		excludeRe, err := regexp.Compile(excludeStr)
		if err != nil {
			errs = append(errs, fmt.Sprintf("EXCLUDE_PATTERN is not a valid regex: %v", err))
		} else {
			cfg.ExcludePattern = excludeRe
		}
	}

	// Optional: Docker host
	cfg.DockerHost = os.Getenv("DOCKER_HOST")
	if cfg.DockerHost == "" {
		cfg.DockerHost = DefaultDockerHost
	}

	// Optional: Docker mode
	cfg.DockerMode = strings.ToLower(os.Getenv("DOCKER_MODE"))
	if cfg.DockerMode == "" {
		cfg.DockerMode = DefaultDockerMode
	}
	if cfg.DockerMode != "auto" && cfg.DockerMode != "swarm" && cfg.DockerMode != "standalone" {
		errs = append(errs, "DOCKER_MODE must be 'auto', 'swarm', or 'standalone'")
	}

	// Optional: Reconcile on startup
	reconcileStr := os.Getenv("RECONCILE_ON_STARTUP")
	if reconcileStr == "" {
		cfg.ReconcileOnStartup = DefaultReconcileOnStartup
	} else {
		cfg.ReconcileOnStartup = parseBool(reconcileStr, DefaultReconcileOnStartup)
	}

	// Optional: Dry run
	dryRunStr := os.Getenv("DRY_RUN")
	if dryRunStr == "" {
		cfg.DryRun = DefaultDryRun
	} else {
		cfg.DryRun = parseBool(dryRunStr, DefaultDryRun)
	}

	// Optional: Health port
	healthPortStr := os.Getenv("HEALTH_PORT")
	if healthPortStr != "" {
		port, err := strconv.Atoi(healthPortStr)
		if err != nil {
			errs = append(errs, fmt.Sprintf("HEALTH_PORT must be a valid integer: %v", err))
		} else if port < 1 || port > 65535 {
			errs = append(errs, "HEALTH_PORT must be between 1 and 65535")
		} else {
			cfg.HealthPort = port
		}
	} else {
		cfg.HealthPort = DefaultHealthPort
	}

	// Optional: Log level
	cfg.LogLevel = strings.ToLower(os.Getenv("LOG_LEVEL"))
	if cfg.LogLevel == "" {
		cfg.LogLevel = DefaultLogLevel
	}
	if cfg.LogLevel != "debug" && cfg.LogLevel != "info" && cfg.LogLevel != "warn" && cfg.LogLevel != "error" {
		errs = append(errs, "LOG_LEVEL must be 'debug', 'info', 'warn', or 'error'")
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("configuration errors:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return cfg, nil
}

// getEnvOrFile returns the value of an environment variable,
// or if VAR_FILE is set, reads the contents from that file.
// Supports Docker secrets pattern.
func getEnvOrFile(key string) string {
	// First check if the direct value is set
	if val := os.Getenv(key); val != "" {
		return val
	}

	// Check for _FILE suffix (Docker secrets)
	fileKey := key + "_FILE"
	if filePath := os.Getenv(fileKey); filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(data))
	}

	return ""
}

// parseBool parses a boolean string, returning defaultValue on parse failure.
func parseBool(s string, defaultValue bool) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultValue
	}
}

// MatchesFilters checks if a hostname matches the include pattern
// and does not match the exclude pattern.
func (c *Config) MatchesFilters(hostname string) bool {
	// Must match include pattern
	if c.IncludePattern != nil && !c.IncludePattern.MatchString(hostname) {
		return false
	}

	// Must not match exclude pattern (if set)
	if c.ExcludePattern != nil && c.ExcludePattern.MatchString(hostname) {
		return false
	}

	return true
}

// Validate performs additional validation that requires all fields to be loaded.
func (c *Config) Validate() error {
	// Ensure the Technitium URL doesn't have trailing slashes
	c.TechnitiumURL = strings.TrimRight(c.TechnitiumURL, "/")

	return nil
}
