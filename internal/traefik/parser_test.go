package traefik

import (
	"reflect"
	"sort"
	"testing"
)

func TestNewParser(t *testing.T) {
	parser := NewParser()

	if parser == nil {
		t.Fatal("expected parser to be initialized")
	}
	if parser.logger == nil {
		t.Error("expected logger to be initialized")
	}
}

func TestExtractHosts_SingleHost(t *testing.T) {
	parser := NewParser()

	labels := map[string]string{
		"traefik.http.routers.myapp.rule": "Host(`myapp.example.com`)",
	}

	hosts := parser.ExtractHosts(labels)

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0] != "myapp.example.com" {
		t.Errorf("expected myapp.example.com, got %s", hosts[0])
	}
}

func TestExtractHosts_MultipleHostsOR(t *testing.T) {
	parser := NewParser()

	labels := map[string]string{
		"traefik.http.routers.myapp.rule": "Host(`app.example.com`) || Host(`www.example.com`)",
	}

	hosts := parser.ExtractHosts(labels)
	sort.Strings(hosts)

	expected := []string{"app.example.com", "www.example.com"}
	sort.Strings(expected)

	if !reflect.DeepEqual(hosts, expected) {
		t.Errorf("expected %v, got %v", expected, hosts)
	}
}

func TestExtractHosts_HostWithPathPrefix(t *testing.T) {
	parser := NewParser()

	labels := map[string]string{
		"traefik.http.routers.myapp.rule": "Host(`myapp.example.com`) && PathPrefix(`/api`)",
	}

	hosts := parser.ExtractHosts(labels)

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0] != "myapp.example.com" {
		t.Errorf("expected myapp.example.com, got %s", hosts[0])
	}
}

func TestExtractHosts_MultipleRouters(t *testing.T) {
	parser := NewParser()

	labels := map[string]string{
		"traefik.http.routers.app.rule":       "Host(`app.example.com`)",
		"traefik.http.routers.api.rule":       "Host(`api.example.com`)",
		"traefik.http.routers.dashboard.rule": "Host(`dash.example.com`)",
	}

	hosts := parser.ExtractHosts(labels)
	sort.Strings(hosts)

	expected := []string{"api.example.com", "app.example.com", "dash.example.com"}

	if !reflect.DeepEqual(hosts, expected) {
		t.Errorf("expected %v, got %v", expected, hosts)
	}
}

func TestExtractHosts_DuplicatesRemoved(t *testing.T) {
	parser := NewParser()

	labels := map[string]string{
		"traefik.http.routers.http.rule":  "Host(`app.example.com`)",
		"traefik.http.routers.https.rule": "Host(`app.example.com`)",
	}

	hosts := parser.ExtractHosts(labels)

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host (deduplicated), got %d", len(hosts))
	}
	if hosts[0] != "app.example.com" {
		t.Errorf("expected app.example.com, got %s", hosts[0])
	}
}

func TestExtractHosts_NoTraefikLabels(t *testing.T) {
	parser := NewParser()

	labels := map[string]string{
		"com.docker.stack.namespace": "mystack",
		"maintainer":                 "admin@example.com",
	}

	hosts := parser.ExtractHosts(labels)

	if len(hosts) != 0 {
		t.Errorf("expected 0 hosts, got %d", len(hosts))
	}
}

func TestExtractHosts_EmptyLabels(t *testing.T) {
	parser := NewParser()

	hosts := parser.ExtractHosts(nil)

	if len(hosts) != 0 {
		t.Errorf("expected 0 hosts from nil labels, got %d", len(hosts))
	}
}

func TestExtractHosts_NonRuleLabels(t *testing.T) {
	parser := NewParser()

	labels := map[string]string{
		"traefik.http.routers.myapp.entrypoints": "websecure",
		"traefik.http.routers.myapp.tls":         "true",
		"traefik.http.services.myapp.loadbalancer.server.port": "8080",
	}

	hosts := parser.ExtractHosts(labels)

	if len(hosts) != 0 {
		t.Errorf("expected 0 hosts from non-rule labels, got %d", len(hosts))
	}
}

func TestExtractHosts_MixedLabels(t *testing.T) {
	parser := NewParser()

	labels := map[string]string{
		"traefik.enable":                         "true",
		"traefik.http.routers.myapp.rule":        "Host(`app.example.com`)",
		"traefik.http.routers.myapp.entrypoints": "websecure",
		"traefik.http.routers.myapp.tls":         "true",
		"com.docker.stack.namespace":             "mystack",
	}

	hosts := parser.ExtractHosts(labels)

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0] != "app.example.com" {
		t.Errorf("expected app.example.com, got %s", hosts[0])
	}
}

func TestExtractHosts_ComplexRule(t *testing.T) {
	parser := NewParser()

	// Complex rule with multiple conditions
	labels := map[string]string{
		"traefik.http.routers.myapp.rule": "(Host(`app.example.com`) || Host(`app2.example.com`)) && PathPrefix(`/api`)",
	}

	hosts := parser.ExtractHosts(labels)
	sort.Strings(hosts)

	expected := []string{"app.example.com", "app2.example.com"}
	sort.Strings(expected)

	if !reflect.DeepEqual(hosts, expected) {
		t.Errorf("expected %v, got %v", expected, hosts)
	}
}

func TestExtractHosts_EmptyHostname(t *testing.T) {
	parser := NewParser()

	labels := map[string]string{
		"traefik.http.routers.myapp.rule": "Host(``)",
	}

	hosts := parser.ExtractHosts(labels)

	if len(hosts) != 0 {
		t.Errorf("expected 0 hosts from empty hostname, got %d", len(hosts))
	}
}

func TestExtractHosts_WhitespaceHostname(t *testing.T) {
	parser := NewParser()

	labels := map[string]string{
		"traefik.http.routers.myapp.rule": "Host(`  `)",
	}

	hosts := parser.ExtractHosts(labels)

	if len(hosts) != 0 {
		t.Errorf("expected 0 hosts from whitespace hostname, got %d", len(hosts))
	}
}

func TestIsRouterRuleLabel(t *testing.T) {
	tests := []struct {
		label    string
		expected bool
	}{
		{"traefik.http.routers.myapp.rule", true},
		{"traefik.http.routers.my-app.rule", true},
		{"traefik.http.routers.my_app_123.rule", true},
		{"traefik.http.routers.a.rule", true},
		{"traefik.http.routers..rule", true}, // Edge case: empty router name (still matches pattern)
		{"traefik.http.routers.myapp.tls", false},
		{"traefik.http.routers.myapp.entrypoints", false},
		{"traefik.http.services.myapp.loadbalancer.server.port", false},
		{"traefik.enable", false},
		{"com.docker.stack.namespace", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			result := isRouterRuleLabel(tt.label)
			if result != tt.expected {
				t.Errorf("isRouterRuleLabel(%q) = %v, want %v", tt.label, result, tt.expected)
			}
		})
	}
}

func TestExtractHostsFromRule(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		expected []string
	}{
		{
			name:     "single host",
			rule:     "Host(`example.com`)",
			expected: []string{"example.com"},
		},
		{
			name:     "multiple hosts with OR",
			rule:     "Host(`a.example.com`) || Host(`b.example.com`)",
			expected: []string{"a.example.com", "b.example.com"},
		},
		{
			name:     "host with path prefix",
			rule:     "Host(`example.com`) && PathPrefix(`/api`)",
			expected: []string{"example.com"},
		},
		{
			name:     "no host",
			rule:     "PathPrefix(`/api`)",
			expected: nil,
		},
		{
			name:     "empty rule",
			rule:     "",
			expected: nil,
		},
		{
			name:     "three hosts",
			rule:     "Host(`a.com`) || Host(`b.com`) || Host(`c.com`)",
			expected: []string{"a.com", "b.com", "c.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractHostsFromRule(tt.rule)
			sort.Strings(result)
			sort.Strings(tt.expected)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ExtractHostsFromRule(%q) = %v, want %v", tt.rule, result, tt.expected)
			}
		})
	}
}

func TestExtractHosts_TCPRoutersIgnored(t *testing.T) {
	parser := NewParser()

	labels := map[string]string{
		"traefik.tcp.routers.mytcp.rule": "HostSNI(`tcp.example.com`)",
		"traefik.http.routers.myhttp.rule": "Host(`http.example.com`)",
	}

	hosts := parser.ExtractHosts(labels)

	// Should only get the HTTP host
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0] != "http.example.com" {
		t.Errorf("expected http.example.com, got %s", hosts[0])
	}
}
