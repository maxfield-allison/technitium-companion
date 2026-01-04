// Package technitium provides a client for the Technitium DNS Server HTTP API.
package technitium

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/maxfield-allison/technitium-companion/internal/metrics"
)

// Record represents a DNS record from the Technitium API.
type Record struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	TTL     int    `json:"ttl"`
	RData   RData  `json:"rData"`
	Disabled bool  `json:"disabled"`
}

// RData contains the record-specific data.
type RData struct {
	IPAddress string `json:"ipAddress,omitempty"` // For A records
	Value     string `json:"value,omitempty"`     // Generic value field
}

// Client is a Technitium DNS Server API client.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	logger     *slog.Logger
}

// ClientOption is a functional option for configuring the Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithLogger sets a custom logger.
func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// NewClient creates a new Technitium API client.
func NewClient(baseURL, token string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// apiResponse is the standard Technitium API response wrapper.
type apiResponse struct {
	Status       string          `json:"status"`
	ErrorMessage string          `json:"errorMessage,omitempty"`
	Response     json.RawMessage `json:"response,omitempty"`
}

// zoneInfo contains zone metadata from the API response.
type zoneInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Disabled bool   `json:"disabled"`
}

// recordsResponse is the response from the records/get endpoint.
type recordsResponse struct {
	Zone    zoneInfo `json:"zone"`
	Name    string   `json:"name"`
	Records []Record `json:"records"`
}

// doRequest performs an HTTP request to the Technitium API.
func (c *Client) doRequest(ctx context.Context, endpoint string, params url.Values) (*apiResponse, error) {
	start := time.Now()

	// Add token to params
	if params == nil {
		params = url.Values{}
	}
	params.Set("token", c.token)

	reqURL := fmt.Sprintf("%s%s?%s", c.baseURL, endpoint, params.Encode())

	c.logger.Debug("making API request",
		slog.String("endpoint", endpoint),
		slog.String("url", c.baseURL+endpoint),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		metrics.RecordAPIRequest(endpoint, "error", time.Since(start).Seconds())
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		metrics.RecordAPIRequest(endpoint, "error", time.Since(start).Seconds())
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		metrics.RecordAPIRequest(endpoint, "error", time.Since(start).Seconds())
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		metrics.RecordAPIRequest(endpoint, "error", time.Since(start).Seconds())
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		metrics.RecordAPIRequest(endpoint, "error", time.Since(start).Seconds())
		return nil, fmt.Errorf("parsing response JSON: %w", err)
	}

	if apiResp.Status == "error" {
		metrics.RecordAPIRequest(endpoint, "error", time.Since(start).Seconds())
		return nil, fmt.Errorf("API error: %s", apiResp.ErrorMessage)
	}

	metrics.RecordAPIRequest(endpoint, "success", time.Since(start).Seconds())
	return &apiResp, nil
}

// AddARecord creates an A record in the specified zone.
func (c *Client) AddARecord(ctx context.Context, zone, hostname, ip string, ttl int) error {
	params := url.Values{}
	params.Set("zone", zone)
	params.Set("domain", hostname)
	params.Set("type", "A")
	params.Set("ipAddress", ip)
	params.Set("ttl", strconv.Itoa(ttl))

	_, err := c.doRequest(ctx, "/api/zones/records/add", params)
	if err != nil {
		return fmt.Errorf("adding A record for %s: %w", hostname, err)
	}

	c.logger.Info("added A record",
		slog.String("hostname", hostname),
		slog.String("ip", ip),
		slog.String("zone", zone),
		slog.Int("ttl", ttl),
	)

	return nil
}

// DeleteARecord removes an A record from the specified zone.
func (c *Client) DeleteARecord(ctx context.Context, zone, hostname, ip string) error {
	params := url.Values{}
	params.Set("zone", zone)
	params.Set("domain", hostname)
	params.Set("type", "A")
	params.Set("ipAddress", ip)

	_, err := c.doRequest(ctx, "/api/zones/records/delete", params)
	if err != nil {
		return fmt.Errorf("deleting A record for %s: %w", hostname, err)
	}

	c.logger.Info("deleted A record",
		slog.String("hostname", hostname),
		slog.String("ip", ip),
		slog.String("zone", zone),
	)

	return nil
}

// GetRecords retrieves all records for a given hostname in the specified zone.
func (c *Client) GetRecords(ctx context.Context, zone, hostname string) ([]Record, error) {
	params := url.Values{}
	params.Set("zone", zone)
	params.Set("domain", hostname)

	apiResp, err := c.doRequest(ctx, "/api/zones/records/get", params)
	if err != nil {
		return nil, fmt.Errorf("getting records for %s: %w", hostname, err)
	}

	var recordsResp recordsResponse
	if err := json.Unmarshal(apiResp.Response, &recordsResp); err != nil {
		return nil, fmt.Errorf("parsing records response: %w", err)
	}

	c.logger.Debug("retrieved records",
		slog.String("hostname", hostname),
		slog.String("zone", zone),
		slog.Int("count", len(recordsResp.Records)),
	)

	return recordsResp.Records, nil
}

// HasARecord checks if a specific A record exists.
func (c *Client) HasARecord(ctx context.Context, zone, hostname, ip string) (bool, error) {
	records, err := c.GetRecords(ctx, zone, hostname)
	if err != nil {
		return false, err
	}

	for _, r := range records {
		if r.Type == "A" && r.RData.IPAddress == ip {
			return true, nil
		}
	}

	return false, nil
}

// EnsureARecord creates an A record if it doesn't already exist.
// Returns true if a record was created, false if it already existed.
func (c *Client) EnsureARecord(ctx context.Context, zone, hostname, ip string, ttl int) (bool, error) {
	exists, err := c.HasARecord(ctx, zone, hostname, ip)
	if err != nil {
		return false, err
	}

	if exists {
		c.logger.Debug("A record already exists",
			slog.String("hostname", hostname),
			slog.String("ip", ip),
		)
		return false, nil
	}

	if err := c.AddARecord(ctx, zone, hostname, ip, ttl); err != nil {
		return false, err
	}

	return true, nil
}
