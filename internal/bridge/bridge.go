// Package bridge provides the MCP bridge between stdio and SSE/HTTP transport.
package bridge

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/iiharu/mcp-over-socks/internal/config"
	"github.com/iiharu/mcp-over-socks/internal/logging"
	"github.com/iiharu/mcp-over-socks/internal/transport"
)

// Bridge connects stdio to a remote MCP server.
type Bridge struct {
	config    *config.Config
	transport transport.Transport
	sseClient *transport.SSEClient // For SSE-specific event handling
	logger    *logging.Logger

	stdin  io.Reader
	stdout io.Writer
}

// New creates a new Bridge with SSE client (for backward compatibility).
func New(cfg *config.Config, sseClient *transport.SSEClient, logger *logging.Logger) *Bridge {
	return &Bridge{
		config:    cfg,
		transport: sseClient,
		sseClient: sseClient,
		logger:    logger,
		stdin:     os.Stdin,
		stdout:    os.Stdout,
	}
}

// NewWithTransport creates a new Bridge with a generic transport.
func NewWithTransport(cfg *config.Config, t transport.Transport, logger *logging.Logger) *Bridge {
	b := &Bridge{
		config:    cfg,
		transport: t,
		logger:    logger,
		stdin:     os.Stdin,
		stdout:    os.Stdout,
	}
	// Check if it's an SSE client for event handling
	if sseClient, ok := t.(*transport.SSEClient); ok {
		b.sseClient = sseClient
	}
	return b
}

// NewWithIO creates a new Bridge with custom IO (for testing).
func NewWithIO(cfg *config.Config, sseClient *transport.SSEClient, logger *logging.Logger, stdin io.Reader, stdout io.Writer) *Bridge {
	return &Bridge{
		config:    cfg,
		transport: sseClient,
		sseClient: sseClient,
		logger:    logger,
		stdin:     stdin,
		stdout:    stdout,
	}
}

// Run starts the bridge and blocks until the context is cancelled or an error occurs.
func (b *Bridge) Run(ctx context.Context) error {
	// Log connection attempt
	b.logger.Info("Connecting to MCP server: %s", b.transport.ServerURL())
	b.logger.Debug("Using proxy: %s", b.config.ProxyAddr)

	// Connect to server
	if err := b.transport.Connect(ctx); err != nil {
		b.logger.Error("Connection failed: %v", err)
		// Wrap with user-friendly error
		return WrapError(ErrServerConnection, err.Error())
	}
	defer func() {
		b.logger.Info("Disconnecting from MCP server")
		b.transport.Close()
		b.logger.Debug("Connection closed")
	}()

	b.logger.Info("Connected to MCP server successfully")

	// Create channels for coordinating goroutines
	errCh := make(chan error, 2)
	var wg sync.WaitGroup

	// Start stdin reader goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := b.readStdin(ctx); err != nil {
			select {
			case errCh <- fmt.Errorf("stdin reader error: %w", err):
			default:
			}
		}
	}()

	// Start event handler goroutine (for SSE transport)
	if b.sseClient != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := b.handleSSEEvents(ctx); err != nil {
				select {
				case errCh <- fmt.Errorf("event handler error: %w", err):
				default:
				}
			}
		}()
	}

	// Wait for context cancellation or error
	if b.sseClient != nil {
		select {
		case <-ctx.Done():
			b.logger.Info("Shutting down bridge")
			return nil
		case err := <-errCh:
			return err
		case err := <-b.sseClient.Errors():
			return fmt.Errorf("connection error: %w", err)
		}
	} else {
		// For Streamable HTTP, just wait for stdin reader or context
		select {
		case <-ctx.Done():
			b.logger.Info("Shutting down bridge")
			return nil
		case err := <-errCh:
			return err
		}
	}
}

// readStdin reads JSON-RPC requests from stdin and forwards them to the server.
func (b *Bridge) readStdin(ctx context.Context) error {
	scanner := bufio.NewScanner(b.stdin)
	// Increase buffer size for large JSON messages
	const maxScannerSize = 10 * 1024 * 1024 // 10MB
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxScannerSize)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Validate JSON
		if !json.Valid(line) {
			b.logger.Error("Invalid JSON received from stdin")
			continue
		}

		b.logger.Debug("Sending request to server: %s", string(line))

		if err := b.transport.Send(ctx, line); err != nil {
			b.logger.Error("Failed to send request: %v", err)
			// Send error response back to stdout
			b.sendErrorResponse(line, err)
		}

		// For Streamable HTTP, handle response inline
		if streamable, ok := b.transport.(*transport.StreamableHTTPClient); ok {
			select {
			case event := <-streamable.Events():
				if event.Data != "" {
					b.logger.Debug("Received response from server: %s", event.Data)
					if _, err := fmt.Fprintln(b.stdout, event.Data); err != nil {
						return fmt.Errorf("failed to write to stdout: %w", err)
					}
				}
			case <-ctx.Done():
				return nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stdin scanner error: %w", err)
	}

	return nil
}

// handleSSEEvents receives SSE events and writes them to stdout.
func (b *Bridge) handleSSEEvents(ctx context.Context) error {
	if b.sseClient == nil {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-b.sseClient.Events():
			if event.Data == "" {
				continue
			}

			b.logger.Debug("Received event from server: %s", event.Data)

			// Write the event data to stdout
			if _, err := fmt.Fprintln(b.stdout, event.Data); err != nil {
				return fmt.Errorf("failed to write to stdout: %w", err)
			}
		}
	}
}

// sendErrorResponse sends a JSON-RPC error response to stdout.
func (b *Bridge) sendErrorResponse(request []byte, err error) {
	// Try to extract the request ID
	var req struct {
		ID interface{} `json:"id"`
	}
	json.Unmarshal(request, &req)

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      req.ID,
		"error": map[string]interface{}{
			"code":    -32000,
			"message": err.Error(),
		},
	}

	data, _ := json.Marshal(response)
	fmt.Fprintln(b.stdout, string(data))
}
