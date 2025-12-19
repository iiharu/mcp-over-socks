// Package transport provides transport implementations for the MCP over SOCKS bridge.
package transport

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// TransportType represents the type of MCP transport.
type TransportType string

const (
	// TransportTypeSSE represents Server-Sent Events transport.
	TransportTypeSSE TransportType = "sse"
	// TransportTypeStreamableHTTP represents Streamable HTTP transport.
	TransportTypeStreamableHTTP TransportType = "streamable"
	// TransportTypeAuto represents automatic detection.
	TransportTypeAuto TransportType = "auto"
)

// ParseTransportType parses a string into a TransportType.
func ParseTransportType(s string) TransportType {
	switch strings.ToLower(s) {
	case "sse":
		return TransportTypeSSE
	case "streamable", "http":
		return TransportTypeStreamableHTTP
	default:
		return TransportTypeAuto
	}
}

// DetectTransportType attempts to detect the transport type of a server.
func DetectTransportType(ctx context.Context, serverURL string, httpClient *http.Client) (TransportType, error) {
	// Create a request with Accept header for SSE
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL, nil)
	if err != nil {
		return TransportTypeAuto, err
	}

	req.Header.Set("Accept", "text/event-stream, application/json")

	// Set a short timeout for detection
	client := &http.Client{
		Transport: httpClient.Transport,
		Timeout:   5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		// If we can't connect, return auto (let the actual connection attempt fail)
		return TransportTypeAuto, err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")

	// Check for SSE content type
	if strings.HasPrefix(contentType, "text/event-stream") {
		return TransportTypeSSE, nil
	}

	// Check for JSON content type (likely Streamable HTTP)
	if strings.HasPrefix(contentType, "application/json") {
		return TransportTypeStreamableHTTP, nil
	}

	// Default to SSE as it's more common for MCP
	return TransportTypeSSE, nil
}

// Transport is an interface for MCP transports.
type Transport interface {
	// Connect establishes a connection to the server.
	Connect(ctx context.Context) error
	// Send sends data to the server.
	Send(ctx context.Context, data []byte) error
	// Close closes the connection.
	Close() error
	// ServerURL returns the server URL.
	ServerURL() string
}

// CreateTransport creates a transport based on the transport type.
func CreateTransport(
	transportType TransportType,
	serverURL string,
	httpClient *http.Client,
	timeout time.Duration,
) Transport {
	switch transportType {
	case TransportTypeStreamableHTTP:
		return NewStreamableHTTPClient(serverURL, httpClient, timeout)
	case TransportTypeSSE:
		fallthrough
	default:
		return NewSSEClient(serverURL, httpClient, timeout)
	}
}

