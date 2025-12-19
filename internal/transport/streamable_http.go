// Package transport provides transport implementations for the MCP over SOCKS bridge.
package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// StreamableHTTPClient handles Streamable HTTP communication with an MCP server.
// Unlike SSE, Streamable HTTP uses regular HTTP request/response for each message.
type StreamableHTTPClient struct {
	serverURL  string
	httpClient *http.Client
	timeout    time.Duration

	mu       sync.Mutex
	closed   bool
	eventsCh chan StreamableEvent
	errCh    chan error
}

// StreamableEvent represents a response from a Streamable HTTP server.
type StreamableEvent struct {
	Data string
}

// NewStreamableHTTPClient creates a new Streamable HTTP client.
func NewStreamableHTTPClient(serverURL string, httpClient *http.Client, timeout time.Duration) *StreamableHTTPClient {
	return &StreamableHTTPClient{
		serverURL:  serverURL,
		httpClient: httpClient,
		timeout:    timeout,
		eventsCh:   make(chan StreamableEvent, 100),
		errCh:      make(chan error, 1),
	}
}

// Connect tests the connection to the Streamable HTTP server.
// Unlike SSE, Streamable HTTP doesn't maintain a persistent connection.
func (c *StreamableHTTPClient) Connect(ctx context.Context) error {
	// For Streamable HTTP, we just verify the server is reachable
	// by making a simple request (OPTIONS or HEAD)
	req, err := http.NewRequestWithContext(ctx, http.MethodOptions, c.serverURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &SSEError{
			Message: fmt.Sprintf("Failed to connect to Streamable HTTP server at %s", c.serverURL),
			Err:     err,
		}
	}
	defer resp.Body.Close()

	// Accept various success statuses (200, 204, 405 for servers that don't support OPTIONS)
	if resp.StatusCode >= 400 && resp.StatusCode != 405 {
		return &SSEError{
			Message: fmt.Sprintf("Server returned status %d", resp.StatusCode),
		}
	}

	return nil
}

// Send sends a JSON-RPC request and receives the response.
func (c *StreamableHTTPClient) Send(ctx context.Context, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.serverURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create POST request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Validate JSON
	if !json.Valid(body) {
		return fmt.Errorf("invalid JSON response from server")
	}

	// Send response to events channel
	select {
	case c.eventsCh <- StreamableEvent{Data: string(body)}:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// Events returns a channel for receiving responses.
func (c *StreamableHTTPClient) Events() <-chan StreamableEvent {
	return c.eventsCh
}

// Errors returns a channel for receiving errors.
func (c *StreamableHTTPClient) Errors() <-chan error {
	return c.errCh
}

// Close closes the client.
func (c *StreamableHTTPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	close(c.eventsCh)
	return nil
}

// ServerURL returns the server URL.
func (c *StreamableHTTPClient) ServerURL() string {
	return c.serverURL
}
