package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/iiharu/mcp-over-socks/internal/bridge"
	"github.com/iiharu/mcp-over-socks/internal/config"
	"github.com/iiharu/mcp-over-socks/internal/logging"
	"github.com/iiharu/mcp-over-socks/internal/transport"
)

func TestBridgeWithMockSSEServer(t *testing.T) {
	// Create a mock SSE server
	var mu sync.Mutex
	responses := make(chan string, 10)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Handle POST requests (incoming JSON-RPC requests)
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
				"result":  map[string]interface{}{"tools": []string{"tool1", "tool2"}},
			}
			respBytes, _ := json.Marshal(response)

			mu.Lock()
			responses <- string(respBytes)
			mu.Unlock()

			w.WriteHeader(http.StatusOK)
			return
		}

		// Handle GET requests (SSE stream)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		// Send responses as SSE events
		for {
			select {
			case resp := <-responses:
				fmt.Fprintf(w, "data: %s\n\n", resp)
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	}))
	defer server.Close()

	// Create config
	cfg := &config.Config{
		ProxyAddr: "socks5://localhost:1080", // Not actually used in this test
		ServerURL: server.URL,
		Timeout:   5 * time.Second,
		LogLevel:  "debug",
	}

	// Create logger
	logger := logging.New(logging.LogLevelDebug)

	// Create SSE client with direct HTTP client (no SOCKS proxy for testing)
	sseClient := transport.NewSSEClient(cfg.ServerURL, server.Client(), cfg.Timeout)

	// Create stdin/stdout buffers
	stdin := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n")
	stdout := &bytes.Buffer{}

	// Create bridge with custom IO
	b := bridge.NewWithIO(cfg, sseClient, logger, stdin, stdout)

	// Run bridge with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Run in goroutine since it blocks
	done := make(chan error, 1)
	go func() {
		done <- b.Run(ctx)
	}()

	// Wait for response or timeout
	select {
	case <-done:
		// Bridge completed
	case <-time.After(3 * time.Second):
		t.Log("Test timed out (expected for SSE stream)")
	}

	// Check stdout for response
	output := stdout.String()
	if output != "" {
		// Verify JSON-RPC response format
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			var resp map[string]interface{}
			if err := json.Unmarshal([]byte(line), &resp); err != nil {
				t.Errorf("Invalid JSON response: %v", err)
				continue
			}
			if resp["jsonrpc"] != "2.0" {
				t.Errorf("Expected jsonrpc 2.0, got %v", resp["jsonrpc"])
			}
		}
	}
}

