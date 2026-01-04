// Package traefik provides utilities for parsing Traefik labels.
package traefik

import (
	"log/slog"
	"regexp"
	"strings"
)

// hostRegex matches Host(`hostname`) patterns in Traefik router rules.
// Captures the hostname inside the backticks.
var hostRegex = regexp.MustCompile("Host\\(`([^`]+)`\\)")

// routerRuleSuffix is the label suffix for Traefik router rules.
const routerRuleSuffix = ".rule"

// Parser extracts hostnames from Traefik labels.
type Parser struct {
	logger *slog.Logger
}

// ParserOption is a functional option for configuring the Parser.
type ParserOption func(*Parser)

// WithLogger sets a custom logger.
func WithLogger(logger *slog.Logger) ParserOption {
	return func(p *Parser) {
		p.logger = logger
	}
}

// NewParser creates a new Traefik label parser.
func NewParser(opts ...ParserOption) *Parser {
	p := &Parser{
		logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// ExtractHosts extracts all hostnames from Traefik labels.
// It looks for traefik.http.routers.*.rule labels and extracts Host() values.
// Returns a deduplicated slice of hostnames.
func (p *Parser) ExtractHosts(labels map[string]string) []string {
	seen := make(map[string]struct{})
	var hosts []string

	for key, value := range labels {
		// Only process traefik router rule labels
		if !isRouterRuleLabel(key) {
			continue
		}

		p.logger.Debug("parsing traefik rule",
			slog.String("label", key),
			slog.String("rule", value),
		)

		// Extract all Host() patterns from the rule
		matches := hostRegex.FindAllStringSubmatch(value, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			hostname := strings.TrimSpace(match[1])
			if hostname == "" {
				continue
			}

			// Deduplicate
			if _, exists := seen[hostname]; !exists {
				seen[hostname] = struct{}{}
				hosts = append(hosts, hostname)
				p.logger.Debug("extracted hostname",
					slog.String("hostname", hostname),
				)
			}
		}
	}

	p.logger.Debug("extracted hosts from labels",
		slog.Int("count", len(hosts)),
	)

	return hosts
}

// isRouterRuleLabel checks if a label key is a Traefik HTTP router rule.
// Matches patterns like: traefik.http.routers.myrouter.rule
func isRouterRuleLabel(key string) bool {
	// Must start with traefik.http.routers. and end with .rule
	if !strings.HasPrefix(key, "traefik.http.routers.") {
		return false
	}
	if !strings.HasSuffix(key, routerRuleSuffix) {
		return false
	}

	// Ensure there's a router name between routers. and .rule
	// traefik.http.routers.NAME.rule
	parts := strings.Split(key, ".")
	// Expected: [traefik, http, routers, NAME, rule]
	if len(parts) < 5 {
		return false
	}

	return true
}

// ExtractHostsFromRule extracts all hostnames from a single Traefik rule string.
// Useful for parsing rules directly without the full label map.
func ExtractHostsFromRule(rule string) []string {
	seen := make(map[string]struct{})
	var hosts []string

	matches := hostRegex.FindAllStringSubmatch(rule, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		hostname := strings.TrimSpace(match[1])
		if hostname == "" {
			continue
		}

		if _, exists := seen[hostname]; !exists {
			seen[hostname] = struct{}{}
			hosts = append(hosts, hostname)
		}
	}

	return hosts
}
