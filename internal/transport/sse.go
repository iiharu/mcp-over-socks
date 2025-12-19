// Package transport provides transport implementations for the MCP over SOCKS bridge.
package transport

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SSEClient handles Server-Sent Events communication with an MCP server.
type SSEClient struct {
	serverURL  string
	httpClient *http.Client
	timeout    time.Duration

	mu       sync.Mutex
	conn     io.ReadCloser
	eventsCh chan SSEEvent
	errCh    chan error
	closed   bool
}

// SSEEvent represents a Server-Sent Event.
type SSEEvent struct {
	// Event is the event type (optional).
	Event string
	// Data is the event data.
	Data string
	// ID is the event ID (optional).
	ID string
}

// NewSSEClient creates a new SSE client.
func NewSSEClient(serverURL string, httpClient *http.Client, timeout time.Duration) *SSEClient {
	return &SSEClient{
		serverURL:  serverURL,
		httpClient: httpClient,
		timeout:    timeout,
		eventsCh:   make(chan SSEEvent, 100),
		errCh:      make(chan error, 1),
	}
}

// SSEError represents an SSE-related error with user-friendly message.
type SSEError struct {
	Message string
	Err     error
}

func (e *SSEError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *SSEError) Unwrap() error {
	return e.Err
}

// Connect establishes a connection to the SSE server.
func (c *SSEClient) Connect(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.serverURL, nil)
	if err != nil {
		return &SSEError{
			Message: "Failed to create request",
			Err:     err,
		}
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Check for common error patterns
		errStr := err.Error()
		if strings.Contains(errStr, "connection refused") {
			return &SSEError{
				Message: fmt.Sprintf("Connection refused to %s - is the server running?", c.serverURL),
				Err:     err,
			}
		}
		if strings.Contains(errStr, "no such host") {
			return &SSEError{
				Message: fmt.Sprintf("Cannot resolve host for %s - check the URL", c.serverURL),
				Err:     err,
			}
		}
		if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
			return &SSEError{
				Message: fmt.Sprintf("Connection timeout to %s - check network connectivity", c.serverURL),
				Err:     err,
			}
		}
		return &SSEError{
			Message: fmt.Sprintf("Failed to connect to SSE server at %s", c.serverURL),
			Err:     err,
		}
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return &SSEError{
			Message: fmt.Sprintf("SSE server returned status %d (expected 200)", resp.StatusCode),
		}
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/event-stream") {
		resp.Body.Close()
		return &SSEError{
			Message: fmt.Sprintf("Unexpected content type '%s' - expected 'text/event-stream'. Is this an SSE endpoint?", contentType),
		}
	}

	c.mu.Lock()
	c.conn = resp.Body
	c.mu.Unlock()

	// Start reading events in background
	go c.readEvents(ctx)

	return nil
}

// readEvents reads SSE events from the connection.
func (c *SSEClient) readEvents(ctx context.Context) {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return
	}

	scanner := bufio.NewScanner(conn)
	var event SSEEvent
	var dataLines []string

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()

		// Empty line indicates end of event
		if line == "" {
			if len(dataLines) > 0 {
				event.Data = strings.Join(dataLines, "\n")
				select {
				case c.eventsCh <- event:
				case <-ctx.Done():
					return
				}
				event = SSEEvent{}
				dataLines = nil
			}
			continue
		}

		// Parse field
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimPrefix(data, " ")
			dataLines = append(dataLines, data)
		} else if strings.HasPrefix(line, "event:") {
			event.Event = strings.TrimPrefix(line, "event:")
			event.Event = strings.TrimPrefix(event.Event, " ")
		} else if strings.HasPrefix(line, "id:") {
			event.ID = strings.TrimPrefix(line, "id:")
			event.ID = strings.TrimPrefix(event.ID, " ")
		}
		// Ignore retry: and comments (lines starting with :)
	}

	if err := scanner.Err(); err != nil {
		select {
		case c.errCh <- err:
		default:
		}
	}
}

// Events returns a channel for receiving SSE events.
func (c *SSEClient) Events() <-chan SSEEvent {
	return c.eventsCh
}

// Errors returns a channel for receiving errors.
func (c *SSEClient) Errors() <-chan error {
	return c.errCh
}

// Send sends a JSON-RPC request to the server via HTTP POST.
// For SSE-based MCP, requests are typically sent via a separate HTTP endpoint.
func (c *SSEClient) Send(ctx context.Context, data []byte) error {
	// Determine the POST endpoint (typically the SSE URL without /sse or with /message suffix)
	postURL := c.serverURL
	if strings.HasSuffix(postURL, "/sse") {
		postURL = strings.TrimSuffix(postURL, "/sse")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, postURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create POST request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Close closes the SSE connection.
func (c *SSEClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ServerURL returns the server URL.
func (c *SSEClient) ServerURL() string {
	return c.serverURL
}
