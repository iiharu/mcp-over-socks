package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/iiharu/mcp-over-socks/internal/transport"
)

func TestSSEClientConnectionErrors(t *testing.T) {
	t.Run("connection refused", func(t *testing.T) {
		// Use a port that's likely not in use
		client := transport.NewSSEClient(
			"http://localhost:59999/sse",
			&http.Client{Timeout: 1 * time.Second},
			1*time.Second,
		)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := client.Connect(ctx)
		if err == nil {
			t.Fatal("expected connection error, got nil")
		}

		errStr := err.Error()
		if !strings.Contains(errStr, "refused") && !strings.Contains(errStr, "Failed to connect") {
			t.Errorf("expected error about connection failure, got: %s", errStr)
		}
	})

	t.Run("wrong content type", func(t *testing.T) {
		// Create a server that returns wrong content type
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"error": "not an SSE endpoint"}`))
		}))
		defer server.Close()

		client := transport.NewSSEClient(
			server.URL,
			server.Client(),
			5*time.Second,
		)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := client.Connect(ctx)
		if err == nil {
			t.Fatal("expected content type error, got nil")
		}

		errStr := err.Error()
		if !strings.Contains(errStr, "content type") && !strings.Contains(errStr, "text/event-stream") {
			t.Errorf("expected error about content type, got: %s", errStr)
		}
	})

	t.Run("non-200 status code", func(t *testing.T) {
		// Create a server that returns 404
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer server.Close()

		client := transport.NewSSEClient(
			server.URL,
			server.Client(),
			5*time.Second,
		)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := client.Connect(ctx)
		if err == nil {
			t.Fatal("expected status code error, got nil")
		}

		errStr := err.Error()
		if !strings.Contains(errStr, "404") && !strings.Contains(errStr, "status") {
			t.Errorf("expected error about status code, got: %s", errStr)
		}
	})
}

func TestSOCKSDialerErrors(t *testing.T) {
	t.Run("empty proxy address", func(t *testing.T) {
		_, err := transport.NewSOCKSDialer("", nil, false)
		if err == nil {
			t.Fatal("expected error for empty proxy address")
		}
	})

	t.Run("socks5h remote DNS", func(t *testing.T) {
		dialer, err := transport.NewSOCKSDialer("localhost:1080", nil, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !dialer.IsRemoteDNS() {
			t.Error("expected IsRemoteDNS() to return true")
		}
	})

	t.Run("socks5 local DNS", func(t *testing.T) {
		dialer, err := transport.NewSOCKSDialer("localhost:1080", nil, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dialer.IsRemoteDNS() {
			t.Error("expected IsRemoteDNS() to return false")
		}
	})
}
