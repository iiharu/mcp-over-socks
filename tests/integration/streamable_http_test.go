package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/iiharu/mcp-over-socks/internal/transport"
)

func TestStreamableHTTPClient(t *testing.T) {
	// Create a mock Streamable HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodPost {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read body", http.StatusInternalServerError)
				return
			}

			var req map[string]interface{}
			if err := json.Unmarshal(body, &req); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			// Create a response
			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result":  map[string]interface{}{"status": "ok"},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	client := transport.NewStreamableHTTPClient(
		server.URL,
		server.Client(),
		5*time.Second,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test Connect
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Test Send
	request := []byte(`{"jsonrpc":"2.0","id":1,"method":"test"}`)
	err = client.Send(ctx, request)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Check response
	select {
	case event := <-client.Events():
		var resp map[string]interface{}
		if err := json.Unmarshal([]byte(event.Data), &resp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		if resp["jsonrpc"] != "2.0" {
			t.Errorf("Expected jsonrpc 2.0, got %v", resp["jsonrpc"])
		}
		if resp["id"] != float64(1) {
			t.Errorf("Expected id 1, got %v", resp["id"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for response")
	}

	// Test Close
	err = client.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestStreamableHTTPClientErrors(t *testing.T) {
	t.Run("invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("not valid json"))
		}))
		defer server.Close()

		client := transport.NewStreamableHTTPClient(
			server.URL,
			server.Client(),
			5*time.Second,
		)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		client.Connect(ctx)

		request := []byte(`{"jsonrpc":"2.0","id":1,"method":"test"}`)
		err := client.Send(ctx, request)
		if err == nil {
			t.Fatal("Expected error for invalid JSON response")
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}))
		defer server.Close()

		client := transport.NewStreamableHTTPClient(
			server.URL,
			server.Client(),
			5*time.Second,
		)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		client.Connect(ctx)

		request := []byte(`{"jsonrpc":"2.0","id":1,"method":"test"}`)
		err := client.Send(ctx, request)
		if err == nil {
			t.Fatal("Expected error for server error response")
		}
	})
}
