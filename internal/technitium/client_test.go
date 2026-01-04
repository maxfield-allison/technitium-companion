package technitium

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockZoneInfo creates a zone info object matching the Technitium API response format.
func mockZoneInfo(name string) map[string]interface{} {
	return map[string]interface{}{
		"name":     name,
		"type":     "Primary",
		"disabled": false,
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:5380", "test-token")

	if client.baseURL != "http://localhost:5380" {
		t.Errorf("expected baseURL http://localhost:5380, got %s", client.baseURL)
	}
	if client.token != "test-token" {
		t.Errorf("expected token test-token, got %s", client.token)
	}
	if client.httpClient == nil {
		t.Error("expected httpClient to be initialized")
	}
	if client.logger == nil {
		t.Error("expected logger to be initialized")
	}
}

func TestAddARecord_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request parameters
		if r.URL.Path != "/api/zones/records/add" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		query := r.URL.Query()
		if query.Get("token") != "test-token" {
			t.Errorf("unexpected token: %s", query.Get("token"))
		}
		if query.Get("zone") != "example.com" {
			t.Errorf("unexpected zone: %s", query.Get("zone"))
		}
		if query.Get("domain") != "test.example.com" {
			t.Errorf("unexpected domain: %s", query.Get("domain"))
		}
		if query.Get("type") != "A" {
			t.Errorf("unexpected type: %s", query.Get("type"))
		}
		if query.Get("ipAddress") != "10.0.0.1" {
			t.Errorf("unexpected ipAddress: %s", query.Get("ipAddress"))
		}
		if query.Get("ttl") != "300" {
			t.Errorf("unexpected ttl: %s", query.Get("ttl"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"response": map[string]interface{}{
				"zone": mockZoneInfo("example.com"),
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.AddARecord(context.Background(), "example.com", "test.example.com", "10.0.0.1", 300)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAddARecord_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "error",
			"errorMessage": "Zone does not exist",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.AddARecord(context.Background(), "nonexistent.com", "test.nonexistent.com", "10.0.0.1", 300)

	if err == nil {
		t.Error("expected error, got nil")
	}
	if err.Error() != "adding A record for test.nonexistent.com: API error: Zone does not exist" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestDeleteARecord_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/zones/records/delete" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		query := r.URL.Query()
		if query.Get("zone") != "example.com" {
			t.Errorf("unexpected zone: %s", query.Get("zone"))
		}
		if query.Get("domain") != "test.example.com" {
			t.Errorf("unexpected domain: %s", query.Get("domain"))
		}
		if query.Get("type") != "A" {
			t.Errorf("unexpected type: %s", query.Get("type"))
		}
		if query.Get("ipAddress") != "10.0.0.1" {
			t.Errorf("unexpected ipAddress: %s", query.Get("ipAddress"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.DeleteARecord(context.Background(), "example.com", "test.example.com", "10.0.0.1")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetRecords_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/zones/records/get" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"response": map[string]interface{}{
				"zone": mockZoneInfo("example.com"),
				"name": "test.example.com",
				"records": []map[string]interface{}{
					{
						"name":     "test.example.com",
						"type":     "A",
						"ttl":      300,
						"disabled": false,
						"rData": map[string]interface{}{
							"ipAddress": "10.0.0.1",
						},
					},
					{
						"name":     "test.example.com",
						"type":     "A",
						"ttl":      300,
						"disabled": false,
						"rData": map[string]interface{}{
							"ipAddress": "10.0.0.2",
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	records, err := client.GetRecords(context.Background(), "example.com", "test.example.com")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}

	if records[0].Type != "A" {
		t.Errorf("expected type A, got %s", records[0].Type)
	}
	if records[0].RData.IPAddress != "10.0.0.1" {
		t.Errorf("expected IP 10.0.0.1, got %s", records[0].RData.IPAddress)
	}
}

func TestGetRecords_NoRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"response": map[string]interface{}{
				"zone": mockZoneInfo("example.com"),
				"name":    "nonexistent.example.com",
				"records": []map[string]interface{}{},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	records, err := client.GetRecords(context.Background(), "example.com", "nonexistent.example.com")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestHasARecord_Exists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"response": map[string]interface{}{
				"zone": mockZoneInfo("example.com"),
				"name": "test.example.com",
				"records": []map[string]interface{}{
					{
						"name": "test.example.com",
						"type": "A",
						"ttl":  300,
						"rData": map[string]interface{}{
							"ipAddress": "10.0.0.1",
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	exists, err := client.HasARecord(context.Background(), "example.com", "test.example.com", "10.0.0.1")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected record to exist")
	}
}

func TestHasARecord_NotExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"response": map[string]interface{}{
				"zone": mockZoneInfo("example.com"),
				"name":    "test.example.com",
				"records": []map[string]interface{}{},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	exists, err := client.HasARecord(context.Background(), "example.com", "test.example.com", "10.0.0.1")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected record to not exist")
	}
}

func TestHasARecord_DifferentIP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"response": map[string]interface{}{
				"zone": mockZoneInfo("example.com"),
				"name": "test.example.com",
				"records": []map[string]interface{}{
					{
						"name": "test.example.com",
						"type": "A",
						"ttl":  300,
						"rData": map[string]interface{}{
							"ipAddress": "10.0.0.2",
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	// Check for a different IP than what exists
	exists, err := client.HasARecord(context.Background(), "example.com", "test.example.com", "10.0.0.1")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected record with specific IP to not exist")
	}
}

func TestEnsureARecord_AlreadyExists(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// Should only call GetRecords, not AddRecord
		if r.URL.Path == "/api/zones/records/add" {
			t.Error("should not call add when record exists")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"response": map[string]interface{}{
				"zone": mockZoneInfo("example.com"),
				"name": "test.example.com",
				"records": []map[string]interface{}{
					{
						"name": "test.example.com",
						"type": "A",
						"ttl":  300,
						"rData": map[string]interface{}{
							"ipAddress": "10.0.0.1",
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	created, err := client.EnsureARecord(context.Background(), "example.com", "test.example.com", "10.0.0.1", 300)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if created {
		t.Error("expected created to be false when record already exists")
	}
	if callCount != 1 {
		t.Errorf("expected 1 API call, got %d", callCount)
	}
}

func TestEnsureARecord_Creates(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/api/zones/records/get" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "ok",
				"response": map[string]interface{}{
					"zone": mockZoneInfo("example.com"),
					"name":    "test.example.com",
					"records": []map[string]interface{}{},
				},
			})
		} else if r.URL.Path == "/api/zones/records/add" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "ok",
			})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	created, err := client.EnsureARecord(context.Background(), "example.com", "test.example.com", "10.0.0.1", 300)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected created to be true when record was added")
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls (get + add), got %d", callCount)
	}
}

func TestHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.AddARecord(context.Background(), "example.com", "test.example.com", "10.0.0.1", 300)

	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}

func TestInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.AddARecord(context.Background(), "example.com", "test.example.com", "10.0.0.1", 300)

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
